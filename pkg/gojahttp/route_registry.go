package gojahttp

import (
	"strings"
	"sync"

	"github.com/dop251/goja"
)

type Route struct {
	Method  string
	Pattern string
	Handler goja.Callable
	Plan    *RoutePlan
}

type RouteDescriptor struct {
	Method       string       `json:"method"`
	Pattern      string       `json:"pattern"`
	Planned      bool         `json:"planned"`
	SecurityMode SecurityMode `json:"securityMode,omitempty"`
	Action       string       `json:"action,omitempty"`
	Name         string       `json:"name,omitempty"`
}

type Registry struct {
	mu     sync.RWMutex
	routes []Route
}

func NewRegistry() *Registry { return &Registry{} }

func (r *Registry) Add(method, pattern string, handler goja.Callable) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes = append(r.routes, Route{Method: strings.ToUpper(method), Pattern: cleanPath(pattern), Handler: handler})
}

func (r *Registry) AddPlanned(plan RoutePlan, handler goja.Callable) {
	r.mu.Lock()
	defer r.mu.Unlock()
	plan.Method = strings.ToUpper(plan.Method)
	plan.Pattern = cleanPath(plan.Pattern)
	r.routes = append(r.routes, Route{Method: plan.Method, Pattern: plan.Pattern, Handler: handler, Plan: &plan})
}

func (r *Registry) Routes() []RouteDescriptor {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]RouteDescriptor, 0, len(r.routes))
	for _, route := range r.routes {
		descriptor := RouteDescriptor{Method: route.Method, Pattern: route.Pattern}
		if route.Plan != nil {
			descriptor.Planned = true
			descriptor.SecurityMode = route.Plan.Security.Mode
			descriptor.Action = route.Plan.Action
			descriptor.Name = route.Plan.Name
		}
		out = append(out, descriptor)
	}
	return out
}

func (r *Registry) Match(method, path string) (Route, map[string]string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	method = strings.ToUpper(method)
	path = cleanPath(path)
	for _, route := range r.routes {
		if route.Method != method && route.Method != "ALL" {
			continue
		}
		params, ok := matchPattern(route.Pattern, path)
		if ok {
			return route, params, true
		}
	}
	return Route{}, nil, false
}

func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if len(p) > 1 {
		p = strings.TrimRight(p, "/")
	}
	if p == "" {
		return "/"
	}
	return p
}

func splitPath(p string) []string {
	p = strings.Trim(cleanPath(p), "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

func matchPattern(pattern, path string) (map[string]string, bool) {
	pp := splitPath(pattern)
	sp := splitPath(path)
	params := map[string]string{}
	for i := 0; i < len(pp); i++ {
		if pp[i] == "*" {
			return params, true
		}
		if i >= len(sp) {
			return nil, false
		}
		if strings.HasPrefix(pp[i], ":") {
			name := strings.TrimPrefix(pp[i], ":")
			if name == "" {
				return nil, false
			}
			params[name] = sp[i]
			continue
		}
		if pp[i] != sp[i] {
			return nil, false
		}
	}
	return params, len(pp) == len(sp)
}
