package gojahttp

import (
	"net/http"
	"strings"
)

// MiddlewareOptions configures PlannedMiddleware for Go programs that want the
// planned auth pipeline around a standard net/http route instead of routing
// through Host directly.
type MiddlewareOptions struct {
	Auth      AuthOptions
	Sessions  SessionOptions
	Dev       bool
	ParamFunc func(r *http.Request, name string) string
}

// PlannedMiddleware validates plan and returns an http.Handler that enforces the
// same planned auth pipeline as Host.RegisterPlannedHTTP before calling next.
func PlannedMiddleware(opts MiddlewareOptions, plan RoutePlan, next PlannedHTTPHandler) (http.Handler, error) {
	plan, err := ValidateRoutePlan(plan)
	if err != nil {
		return nil, err
	}
	host := NewHost(HostOptions{Dev: opts.Dev, Sessions: opts.Sessions, Auth: opts.Auth})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if plan.Method != "ALL" && r.Method != plan.Method && (r.Method != http.MethodHead || plan.Method != http.MethodGet) {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		params, ok := middlewareParams(plan.Pattern, r, opts.ParamFunc)
		if !ok {
			http.NotFound(w, r)
			return
		}
		session, err := host.sessions.Session(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req, err := NewRequestDTO(r, params, session)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		loggingWriter, wrappedWriter := newAccessLogResponseWriter(w)
		route := Route{Method: plan.Method, Pattern: plan.Pattern, Kind: RouteKindPlannedHTTP, Plan: &plan, HTTPHandler: next}
		host.servePlannedHTTP(wrappedWriter, r, route, req, loggingWriter)
	}), nil
}

func middlewareParams(pattern string, r *http.Request, paramFunc func(*http.Request, string) string) (map[string]string, bool) {
	if paramFunc == nil {
		params, ok := matchPattern(pattern, r.URL.Path)
		return params, ok
	}
	params := map[string]string{}
	for _, part := range splitPath(pattern) {
		if !strings.HasPrefix(part, ":") {
			continue
		}
		name := strings.TrimPrefix(part, ":")
		if name == "" {
			return nil, false
		}
		value := strings.TrimSpace(paramFunc(r, name))
		if value == "" {
			return nil, false
		}
		params[name] = value
	}
	return params, true
}
