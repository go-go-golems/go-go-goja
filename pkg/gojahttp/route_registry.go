package gojahttp

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/dop251/goja"
)

// PlannedHTTPHandler is a Go-native planned route handler. The handler runs
// only after the route plan has authenticated, checked CSRF when required,
// resolved resources, and authorized the requested action.
type PlannedHTTPHandler func(ctx context.Context, sec *SecureContext, w http.ResponseWriter, r *http.Request) error

// RouteKind identifies which handler backend owns a matched route.
type RouteKind string

const (
	RouteKindRawGoja     RouteKind = "raw-goja"
	RouteKindPlannedGoja RouteKind = "planned-goja"
	RouteKindPlannedHTTP RouteKind = "planned-http"
)

type Route struct {
	Method  string
	Pattern string
	Kind    RouteKind
	Plan    *RoutePlan

	GojaHandler goja.Callable
	HTTPHandler PlannedHTTPHandler
}

type RouteDescriptor struct {
	Method            string       `json:"method"`
	Pattern           string       `json:"pattern"`
	Kind              RouteKind    `json:"kind,omitempty"`
	Planned           bool         `json:"planned"`
	SecurityMode      SecurityMode `json:"securityMode,omitempty"`
	Action            string       `json:"action,omitempty"`
	Name              string       `json:"name,omitempty"`
	CSRFRequired      bool         `json:"csrfRequired,omitempty"`
	AuditEvent        string       `json:"auditEvent,omitempty"`
	RateLimitPolicies string       `json:"rateLimitPolicies,omitempty"`
}

type Registry struct {
	mu     sync.RWMutex
	routes []Route
}

func NewRegistry() *Registry { return &Registry{} }

func (r *Registry) Add(method, pattern string, handler goja.Callable) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes = append(r.routes, Route{Method: strings.ToUpper(method), Pattern: cleanPath(pattern), Kind: RouteKindRawGoja, GojaHandler: handler})
}

func (r *Registry) AddPlanned(plan RoutePlan, handler goja.Callable) {
	r.mu.Lock()
	defer r.mu.Unlock()
	plan.Method = strings.ToUpper(plan.Method)
	plan.Pattern = cleanPath(plan.Pattern)
	r.routes = append(r.routes, Route{Method: plan.Method, Pattern: plan.Pattern, Kind: RouteKindPlannedGoja, Plan: &plan, GojaHandler: handler})
}

func (r *Registry) AddPlannedHTTP(plan RoutePlan, handler PlannedHTTPHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	plan.Method = strings.ToUpper(plan.Method)
	plan.Pattern = cleanPath(plan.Pattern)
	r.routes = append(r.routes, Route{Method: plan.Method, Pattern: plan.Pattern, Kind: RouteKindPlannedHTTP, Plan: &plan, HTTPHandler: handler})
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
		if route.kind() == RouteKindPlannedHTTP {
			descriptor.Kind = RouteKindPlannedHTTP
		}
		if route.Plan != nil {
			descriptor.Planned = true
			descriptor.SecurityMode = route.Plan.Security.Mode
			descriptor.Action = route.Plan.Action
			descriptor.Name = route.Plan.Name
			descriptor.CSRFRequired = route.Plan.CSRF.Required
			descriptor.AuditEvent = route.Plan.Audit.Event
			if len(route.Plan.RateLimits) > 0 {
				policies := make([]string, 0, len(route.Plan.RateLimits))
				for _, limit := range route.Plan.RateLimits {
					policies = append(policies, limit.Policy)
				}
				descriptor.RateLimitPolicies = strings.Join(policies, ",")
			}
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

func (route Route) kind() RouteKind {
	if route.Kind != "" {
		return route.Kind
	}
	if route.Plan != nil {
		return RouteKindPlannedGoja
	}
	return RouteKindRawGoja
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
