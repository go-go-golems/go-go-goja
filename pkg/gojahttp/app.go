package gojahttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// App is a Go-native fluent route builder for registering planned HTTP routes
// on a Host. It mirrors the JavaScript Express planned-route builder while
// producing the same RoutePlan contract and using Go handlers.
type App struct {
	host *Host
}

// NewApp returns a Go-native planned route builder backed by host.
func NewApp(host *Host) *App { return &App{host: host} }

func (a *App) Get(pattern string) *RouteNeedsSecurity    { return a.Route(http.MethodGet, pattern) }
func (a *App) Post(pattern string) *RouteNeedsSecurity   { return a.Route(http.MethodPost, pattern) }
func (a *App) Put(pattern string) *RouteNeedsSecurity    { return a.Route(http.MethodPut, pattern) }
func (a *App) Patch(pattern string) *RouteNeedsSecurity  { return a.Route(http.MethodPatch, pattern) }
func (a *App) Delete(pattern string) *RouteNeedsSecurity { return a.Route(http.MethodDelete, pattern) }
func (a *App) All(pattern string) *RouteNeedsSecurity    { return a.Route("ALL", pattern) }

// Route starts a planned route declaration for method and pattern.
func (a *App) Route(method, pattern string) *RouteNeedsSecurity {
	return &RouteNeedsSecurity{builder: &plannedRouteBuilder{host: a.host, plan: RoutePlan{Method: method, Pattern: pattern}}}
}

type plannedRouteBuilder struct {
	host *Host
	plan RoutePlan
}

// RouteNeedsSecurity is the first builder stage. Choose Public or Auth before a
// handler can be registered.
type RouteNeedsSecurity struct{ builder *plannedRouteBuilder }

// RouteNeedsPolicy is the authenticated route stage. Declare resources, CSRF,
// audit, and an action before registering a handler.
type RouteNeedsPolicy struct{ builder *plannedRouteBuilder }

// RouteNeedsHandler is the final route stage. Register the Go handler here.
type RouteNeedsHandler struct{ builder *plannedRouteBuilder }

func (r *RouteNeedsSecurity) Name(name string) *RouteNeedsSecurity {
	r.builder.plan.Name = strings.TrimSpace(name)
	return r
}

func (r *RouteNeedsSecurity) Public() *RouteNeedsHandler {
	r.builder.plan.Security = SecuritySpec{Mode: SecurityModePublic}
	return &RouteNeedsHandler{builder: r.builder}
}

func (r *RouteNeedsSecurity) Auth(spec SecuritySpec) *RouteNeedsPolicy {
	r.builder.plan.Security = spec
	return &RouteNeedsPolicy{builder: r.builder}
}

func (r *RouteNeedsPolicy) Resource(spec ResourceSpec) *RouteNeedsPolicy {
	r.builder.plan.Resources = append(r.builder.plan.Resources, spec)
	return r
}

func (r *RouteNeedsPolicy) CSRF(required ...bool) *RouteNeedsPolicy {
	r.builder.plan.CSRF.Required = optionalBool(true, required...)
	return r
}

func (r *RouteNeedsPolicy) Audit(event string) *RouteNeedsPolicy {
	r.builder.plan.Audit.Event = strings.TrimSpace(event)
	return r
}

func (r *RouteNeedsPolicy) Allow(action string) *RouteNeedsHandler {
	r.builder.plan.Action = strings.TrimSpace(action)
	return &RouteNeedsHandler{builder: r.builder}
}

func (r *RouteNeedsHandler) CSRF(required ...bool) *RouteNeedsHandler {
	r.builder.plan.CSRF.Required = optionalBool(true, required...)
	return r
}

func (r *RouteNeedsHandler) Audit(event string) *RouteNeedsHandler {
	r.builder.plan.Audit.Event = strings.TrimSpace(event)
	return r
}

// Handle validates the accumulated plan and registers handler as a planned Go
// HTTP route on the backing host.
func (r *RouteNeedsHandler) Handle(handler PlannedHTTPHandler) error {
	if r.builder.host == nil {
		return fmt.Errorf("gojahttp app route requires host")
	}
	return r.builder.host.RegisterPlannedHTTP(r.builder.plan, handler)
}

// JSONHandler is a convenience handler for planned routes that return a JSON
// value or an error.
type JSONHandler func(context.Context, *SecureContext) (any, error)

// HandleJSON registers a planned route that encodes the handler result as JSON.
func (r *RouteNeedsHandler) HandleJSON(handler JSONHandler) error {
	return r.Handle(func(ctx context.Context, sec *SecureContext, w http.ResponseWriter, _ *http.Request) error {
		value, err := handler(ctx, sec)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(value)
	})
}

// User starts a Go-native user auth spec builder.
func User() UserAuthBuilder {
	return UserAuthBuilder{spec: SecuritySpec{Mode: SecurityModeUser, Required: true}}
}

// UserAuthBuilder builds SecuritySpec values for Go planned routes.
type UserAuthBuilder struct{ spec SecuritySpec }

func (b UserAuthBuilder) Required() SecuritySpec {
	b.spec.Required = true
	return b.spec
}

func (b UserAuthBuilder) MFAFresh(within time.Duration) SecuritySpec {
	b.spec.Required = true
	b.spec.MFAFreshWithin = within
	return b.spec
}

// Resource starts a Go-native resource spec builder.
func Resource(typ string) ResourceBuilder {
	typ = strings.TrimSpace(typ)
	return ResourceBuilder{spec: ResourceSpec{Name: typ, Type: typ}}
}

// ResourceBuilder builds ResourceSpec values for Go planned routes.
type ResourceBuilder struct{ spec ResourceSpec }

func (b ResourceBuilder) Named(name string) ResourceBuilder {
	b.spec.Name = strings.TrimSpace(name)
	return b
}

func (b ResourceBuilder) IDFromParam(param string) ResourceBuilder {
	b.spec.ID = ValueSource{Kind: ValueSourceParam, Key: strings.TrimSpace(param)}
	return b
}

func (b ResourceBuilder) IDFromQuery(key string) ResourceBuilder {
	b.spec.ID = ValueSource{Kind: ValueSourceQuery, Key: strings.TrimSpace(key)}
	return b
}

func (b ResourceBuilder) IDFromBody(key string) ResourceBuilder {
	b.spec.ID = ValueSource{Kind: ValueSourceBody, Key: strings.TrimSpace(key)}
	return b
}

func (b ResourceBuilder) IDLiteral(value string) ResourceBuilder {
	b.spec.ID = ValueSource{Kind: ValueSourceLiteral, Value: strings.TrimSpace(value)}
	return b
}

func (b ResourceBuilder) TenantFromParam(param string) ResourceBuilder {
	source := ValueSource{Kind: ValueSourceParam, Key: strings.TrimSpace(param)}
	b.spec.Tenant = &source
	return b
}

func (b ResourceBuilder) TenantLiteral(value string) ResourceBuilder {
	source := ValueSource{Kind: ValueSourceLiteral, Value: strings.TrimSpace(value)}
	b.spec.Tenant = &source
	return b
}

func (b ResourceBuilder) MustExist() ResourceSpec {
	b.spec.MustExist = true
	return b.spec
}

func (b ResourceBuilder) Spec() ResourceSpec { return b.spec }

func optionalBool(defaultValue bool, values ...bool) bool {
	if len(values) == 0 {
		return defaultValue
	}
	return values[0]
}
