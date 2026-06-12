package gojahttp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

type HostOptions struct {
	Dev             bool
	Renderer        Renderer
	Sessions        SessionOptions
	Auth            AuthOptions
	RejectRawRoutes bool
}

type StaticMount struct {
	Prefix          string
	Handler         http.Handler
	ExcludePrefixes []string
}

type Host struct {
	registry        *Registry
	dev             bool
	renderer        Renderer
	owner           runtimeowner.RuntimeOwner
	sessions        *SessionManager
	auth            AuthOptions
	rejectRawRoutes bool
	static          []StaticMount
}

func NewHost(opts HostOptions) *Host {
	return &Host{registry: NewRegistry(), dev: opts.Dev, renderer: opts.Renderer, sessions: NewSessionManager(opts.Sessions), auth: opts.Auth, rejectRawRoutes: opts.RejectRawRoutes}
}

func (h *Host) SetRuntime(owner runtimeowner.RuntimeOwner) { h.owner = owner }
func (h *Host) Register(method, pattern string, handler goja.Callable) {
	h.registry.Add(method, pattern, handler)
}
func (h *Host) RegisterPlanned(plan RoutePlan, handler goja.Callable) error {
	plan, err := ValidateRoutePlan(plan)
	if err != nil {
		return err
	}
	h.registry.AddPlanned(plan, handler)
	return nil
}
func (h *Host) Routes() []RouteDescriptor {
	if h == nil || h.registry == nil {
		return nil
	}
	return h.registry.Routes()
}
func (h *Host) RegisterStatic(prefix, dir string) {
	h.RegisterStaticHandler(prefix, http.FileServer(http.Dir(dir)))
}

func (h *Host) RegisterStaticHandler(prefix string, handler http.Handler) {
	h.RegisterStaticHandlerWithOptions(prefix, handler, nil)
}

func (h *Host) RegisterStaticHandlerWithOptions(prefix string, handler http.Handler, excludePrefixes []string) {
	prefix = cleanPath(prefix)
	excludes := make([]string, 0, len(excludePrefixes))
	for _, exclude := range excludePrefixes {
		exclude = cleanPath(exclude)
		if exclude != "" {
			excludes = append(excludes, exclude)
		}
	}
	h.static = append(h.static, StaticMount{Prefix: prefix, Handler: stripMountPrefix(prefix, handler), ExcludePrefixes: excludes})
}

func stripMountPrefix(prefix string, handler http.Handler) http.Handler {
	if prefix == "/" {
		return handler
	}
	return http.StripPrefix(prefix, handler)
}

func staticMountMatches(prefix, requestPath string) bool {
	prefix = cleanPath(prefix)
	requestPath = cleanPath(requestPath)
	if prefix == "/" {
		return true
	}
	return requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/")
}

func staticMountExcluded(excludePrefixes []string, requestPath string) bool {
	for _, exclude := range excludePrefixes {
		if staticMountMatches(exclude, requestPath) {
			return true
		}
	}
	return false
}

func (h *Host) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, mount := range h.static {
		if staticMountMatches(mount.Prefix, r.URL.Path) {
			if staticMountExcluded(mount.ExcludePrefixes, r.URL.Path) {
				continue
			}
			mount.Handler.ServeHTTP(w, r)
			return
		}
	}
	if h.owner == nil {
		http.Error(w, "runtime not initialized", http.StatusInternalServerError)
		return
	}
	route, params, ok := h.registry.Match(r.Method, r.URL.Path)
	if !ok && r.Method == http.MethodHead {
		route, params, ok = h.registry.Match(http.MethodGet, r.URL.Path)
		if ok {
			w = headResponseWriter{ResponseWriter: w}
		}
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	if route.Plan == nil && h.rejectRawRoutes {
		h.writeRawRouteRejected(w, route)
		return
	}
	session, err := h.sessions.Session(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req, err := NewRequestDTO(r, params, session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if route.Plan != nil {
		h.servePlannedRoute(w, r, route, req)
		return
	}
	res := NewResponse(w, h.renderer)
	ret, err := h.owner.Call(r.Context(), "http-handler", func(ctx context.Context, vm *goja.Runtime) (any, error) {
		result, err := route.Handler(goja.Undefined(), vm.ToValue(req.Map()), res.JSObject(vm))
		if err != nil {
			return nil, err
		}
		if promise, ok := result.Export().(*goja.Promise); ok {
			return promise, nil
		}
		return nil, h.finishHandlerResult(vm, res, result)
	})
	if err == nil {
		if promise, ok := ret.(*goja.Promise); ok {
			err = h.awaitAndFinishPromise(r.Context(), res, promise)
		}
	}
	if err != nil && !res.Sent() {
		if h.dev {
			http.Error(w, fmt.Sprintf("JavaScript handler error: %v", err), http.StatusInternalServerError)
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}
}

func (h *Host) writeRawRouteRejected(w http.ResponseWriter, route Route) {
	message := "raw routes disabled"
	if h.dev {
		message = fmt.Sprintf("raw route %s %s rejected: register a planned route with .public() or auth", route.Method, route.Pattern)
	}
	http.Error(w, message, http.StatusInternalServerError)
}

func (h *Host) finishHandlerResult(vm *goja.Runtime, res *Response, result goja.Value) error {
	if !res.Sent() && !goja.IsUndefined(result) && !goja.IsNull(result) {
		if _, ok := result.Export().(string); ok {
			return res.Send(vm, result)
		}
		return res.HTML(vm, result)
	}
	if !res.Sent() {
		return res.End()
	}
	return nil
}

func (h *Host) awaitAndFinishPromise(ctx context.Context, res *Response, promise *goja.Promise) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		ret, err := h.owner.Call(ctx, "http-handler.promise-state", func(_ context.Context, vm *goja.Runtime) (any, error) {
			snapshot := promiseSnapshot{State: promise.State(), Result: promise.Result()}
			if snapshot.State == goja.PromiseStateFulfilled {
				return snapshot, h.finishHandlerResult(vm, res, snapshot.Result)
			}
			return snapshot, nil
		})
		if err != nil {
			return err
		}
		snapshot := ret.(promiseSnapshot)
		switch snapshot.State {
		case goja.PromiseStatePending:
			time.Sleep(5 * time.Millisecond)
		case goja.PromiseStateRejected:
			return fmt.Errorf("promise rejected: %s", valueString(snapshot.Result))
		case goja.PromiseStateFulfilled:
			return nil
		}
	}
}

type promiseSnapshot struct {
	State  goja.PromiseState
	Result goja.Value
}

func valueString(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return "undefined"
	}
	return value.String()
}

type headResponseWriter struct {
	http.ResponseWriter
}

func (w headResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}
