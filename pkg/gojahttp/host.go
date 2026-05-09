package gojahttp

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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
	owner    runtimeowner.Runner
	sessions *SessionManager
	static   []StaticMount
}

func NewHost(opts HostOptions) *Host {
	return &Host{registry: NewRegistry(), dev: opts.Dev, renderer: opts.Renderer, sessions: NewSessionManager(opts.Sessions)}
}

func (h *Host) SetRuntime(owner runtimeowner.Runner) { h.owner = owner }
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
	_, err = h.owner.Call(r.Context(), "http-handler", func(ctx context.Context, vm *goja.Runtime) (any, error) {
		result, err := route.Handler(goja.Undefined(), vm.ToValue(req.Map()), res.JSObject(vm))
		if err != nil {
			return nil, err
		}
		if !res.Sent() && !goja.IsUndefined(result) && !goja.IsNull(result) {
			if _, ok := result.Export().(string); ok {
				return nil, res.Send(vm, result)
			}
			return nil, res.HTML(vm, result)
		}
		if !res.Sent() {
			return nil, res.End()
		}
		return nil, nil
	})
	if err != nil && !res.Sent() {
		if h.dev {
			http.Error(w, fmt.Sprintf("JavaScript handler error: %v", err), http.StatusInternalServerError)
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}
}

type headResponseWriter struct {
	http.ResponseWriter
}

func (w headResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}
