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
	Dev      bool
	Renderer Renderer
	Sessions SessionOptions
}

type StaticMount struct {
	Prefix  string
	Handler http.Handler
}

type Host struct {
	registry *Registry
	dev      bool
	renderer Renderer
	owner    runtimeowner.RuntimeOwner
	sessions *SessionManager
	static   []StaticMount
}

func NewHost(opts HostOptions) *Host {
	return &Host{registry: NewRegistry(), dev: opts.Dev, renderer: opts.Renderer, sessions: NewSessionManager(opts.Sessions)}
}

func (h *Host) SetRuntime(owner runtimeowner.RuntimeOwner) { h.owner = owner }
func (h *Host) Register(method, pattern string, handler goja.Callable) {
	h.registry.Add(method, pattern, handler)
}
func (h *Host) RegisterStatic(prefix, dir string) {
	prefix = cleanPath(prefix)
	h.static = append(h.static, StaticMount{Prefix: prefix, Handler: http.StripPrefix(prefix, http.FileServer(http.Dir(dir)))})
}

func (h *Host) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, mount := range h.static {
		if r.URL.Path == mount.Prefix || strings.HasPrefix(r.URL.Path, mount.Prefix+"/") {
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
