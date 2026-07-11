package gojahttp_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
)

type authenticatorFunc func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error)

func (f authenticatorFunc) Authenticate(ctx context.Context, req *http.Request, session *gojahttp.SessionDTO, spec gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
	return f(ctx, req, session, spec)
}

type resolverFunc func(context.Context, gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error)

func (f resolverFunc) ResolveResource(ctx context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
	return f(ctx, req)
}

type authorizerFunc func(context.Context, gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error)

func (f authorizerFunc) Authorize(ctx context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
	return f(ctx, req)
}

type csrfFunc func(context.Context, gojahttp.CSRFRequest) error

func (f csrfFunc) VerifyCSRF(ctx context.Context, req gojahttp.CSRFRequest) error {
	return f(ctx, req)
}

type auditFunc func(context.Context, gojahttp.AuditEvent) error

func (f auditFunc) RecordAudit(ctx context.Context, event gojahttp.AuditEvent) error {
	return f(ctx, event)
}

func plannedTestRuntime(t *testing.T, host *gojahttp.Host, script string) goja.Callable {
	t.Helper()
	factory, err := engine.NewRuntimeFactoryBuilder().Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(engine.WithStartupContext(context.Background()), engine.WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	t.Cleanup(func() { _ = rt.Close(context.Background()) })
	host.SetRuntime(rt.Owner)
	ret, err := rt.Owner.Call(context.Background(), "load-planned-test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(script)
	})
	if err != nil {
		t.Fatalf("load script: %v", err)
	}
	fn, ok := goja.AssertFunction(ret.(goja.Value))
	if !ok {
		t.Fatalf("script did not return function: %T", ret)
	}
	return fn
}

func TestRejectRawRoutesBlocksUnplannedRoute(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, RejectRawRoutes: true})
	handler := plannedTestRuntime(t, host, `(function(_req, res) { res.send("should not run"); })`)
	host.Register("GET", "/raw", handler)

	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/raw", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "raw route GET /raw rejected") {
		t.Fatalf("body=%s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "should not run") {
		t.Fatalf("handler ran: %s", rr.Body.String())
	}
}

func TestRejectRawRoutesStillAllowsPlannedRoute(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, RejectRawRoutes: true})
	handler := plannedTestRuntime(t, host, `(function(_ctx, res) { res.json({ ok: true }); })`)
	if err := host.RegisterPlanned(gojahttp.RoutePlan{Method: "GET", Pattern: "/planned", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic}}, handler); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}

	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/planned", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"ok":true`) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestPlannedPublicRouteDispatchesSecureContext(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	handler := plannedTestRuntime(t, host, `(function(ctx, res) { res.type("text/plain").send("hello " + ctx.params.name); })`)
	if err := host.RegisterPlanned(gojahttp.RoutePlan{Method: "GET", Pattern: "/hello/:name", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic}}, handler); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/hello/goja", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.TrimSpace(rr.Body.String()) != "hello goja" {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestPlannedUserRouteAuthenticatesAndAuthorizes(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
			return &gojahttp.Actor{ID: "u1", Kind: "user"}, nil
		}),
		Authorizer: authorizerFunc(func(_ context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
			if req.Actor == nil || req.Actor.ID != "u1" || req.Action != "user.self.read" {
				return gojahttp.AuthorizationDecision{}, nil
			}
			return gojahttp.AuthorizationDecision{Allowed: true}, nil
		}),
	}})
	handler := plannedTestRuntime(t, host, `(function(ctx, res) { res.json({ actor: ctx.actor.id, action: ctx.action }); })`)
	if err := host.RegisterPlanned(gojahttp.RoutePlan{Method: "GET", Pattern: "/me", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}, Action: "user.self.read"}, handler); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/me", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"actor":"u1"`) || !strings.Contains(rr.Body.String(), `"action":"user.self.read"`) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestPlannedUserRouteInstallsActorInOwnerContext(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
			return &gojahttp.Actor{ID: "context-user", Kind: "user"}, nil
		}),
		Authorizer: authorizerFunc(func(context.Context, gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
			return gojahttp.AuthorizationDecision{Allowed: true}, nil
		}),
	}})
	factory, err := engine.NewRuntimeFactoryBuilder().Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(engine.WithStartupContext(context.Background()), engine.WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	host.SetRuntime(rt.Owner)
	var observedActorID string
	ret, err := rt.Owner.Call(context.Background(), "create actor-context handler", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.ToValue(func(goja.FunctionCall) goja.Value {
			actor, ok := gojahttp.ActorFromContext(runtimebridge.CurrentOwnerContext(vm))
			if ok {
				observedActorID = actor.ID
			}
			return goja.Undefined()
		}), nil
	})
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	handler, ok := goja.AssertFunction(ret.(goja.Value))
	if !ok {
		t.Fatalf("handler type=%T", ret)
	}
	if err := host.RegisterPlanned(gojahttp.RoutePlan{Method: http.MethodGet, Pattern: "/context-actor", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}, Action: "actor.context.read"}, handler); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/context-actor", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if observedActorID != "context-user" {
		t.Fatalf("owner context actor=%q, want context-user", observedActorID)
	}
}

func TestPlannedHandlerCannotMutateHostOwnedAuthValues(t *testing.T) {
	actor := &gojahttp.Actor{ID: "u1", Kind: "user", TenantIDs: []string{"o1"}, Claims: map[string]any{
		"role":   "reader",
		"nested": map[string]any{"tier": "basic"},
		"tags":   []any{"alpha"},
	}}
	resource := &gojahttp.ResourceRef{Name: "project", Type: "project", ID: "p1", TenantID: "o1", Claims: map[string]any{
		"owner":  "u1",
		"nested": map[string]any{"state": "open"},
	}}
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
			return actor, nil
		}),
		Resources: resolverFunc(func(context.Context, gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
			return resource, nil
		}),
		Authorizer: authorizerFunc(func(context.Context, gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
			return gojahttp.AuthorizationDecision{Allowed: true}, nil
		}),
	}})
	handler := plannedTestRuntime(t, host, `(function(ctx, res) {
  ctx.actor.tenantIds[0] = "mutated";
  ctx.actor.claims.role = "admin";
  ctx.actor.claims.nested.tier = "root";
  ctx.actor.claims.tags[0] = "omega";
  ctx.resources.project.claims.owner = "attacker";
  ctx.resources.project.claims.nested.state = "closed";
  ctx.resource("project").claims.owner = "other";
  res.json({ ok: true });
})`)
	if err := host.RegisterPlanned(gojahttp.RoutePlan{
		Method:   "PATCH",
		Pattern:  "/projects/:projectId",
		Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser},
		Resources: []gojahttp.ResourceSpec{{
			Name: "project",
			Type: "project",
			ID:   gojahttp.ValueSource{Kind: gojahttp.ValueSourceParam, Key: "projectId"},
		}},
		Action: "project.update",
	}, handler); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodPatch, "/projects/p1", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if actor.TenantIDs[0] != "o1" || actor.Claims["role"] != "reader" {
		t.Fatalf("actor mutated: %#v", actor)
	}
	if nested := actor.Claims["nested"].(map[string]any); nested["tier"] != "basic" {
		t.Fatalf("actor nested claims mutated: %#v", actor.Claims)
	}
	if tags := actor.Claims["tags"].([]any); tags[0] != "alpha" {
		t.Fatalf("actor claim slice mutated: %#v", actor.Claims)
	}
	if resource.Claims["owner"] != "u1" {
		t.Fatalf("resource claims mutated: %#v", resource.Claims)
	}
	if nested := resource.Claims["nested"].(map[string]any); nested["state"] != "open" {
		t.Fatalf("resource nested claims mutated: %#v", resource.Claims)
	}
}

func TestPlannedUserRouteReturns401WhenUnauthenticated(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
			return nil, gojahttp.ErrUnauthenticated
		}),
		Authorizer: authorizerFunc(func(context.Context, gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
			return gojahttp.AuthorizationDecision{Allowed: true}, nil
		}),
	}})
	handler := plannedTestRuntime(t, host, `(function(_ctx, res) { res.send("should not run"); })`)
	if err := host.RegisterPlanned(gojahttp.RoutePlan{Method: "GET", Pattern: "/me", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}, Action: "user.self.read"}, handler); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/me", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "should not run") {
		t.Fatalf("handler ran: %s", rr.Body.String())
	}
}

func TestPlannedRouteVerifiesCSRFBeforeHandler(t *testing.T) {
	called := false
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		CSRF: csrfFunc(func(_ context.Context, req gojahttp.CSRFRequest) error {
			called = true
			if req.Plan.Pattern != "/submit" || req.Session == nil {
				t.Fatalf("csrf request = %#v", req)
			}
			return nil
		}),
	}})
	handler := plannedTestRuntime(t, host, `(function(_ctx, res) { res.json({ ok: true }); })`)
	if err := host.RegisterPlanned(gojahttp.RoutePlan{Method: "POST", Pattern: "/submit", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic}, CSRF: gojahttp.CSRFSpec{Required: true}}, handler); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/submit", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !called {
		t.Fatal("expected csrf verifier to be called")
	}
}

func TestPlannedRouteCSRFUsesIncomingMethodForAllRoutes(t *testing.T) {
	calls := 0
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		CSRF: csrfFunc(func(_ context.Context, req gojahttp.CSRFRequest) error {
			calls++
			if req.HTTPRequest.Method != http.MethodPost {
				t.Fatalf("csrf checked safe method %s", req.HTTPRequest.Method)
			}
			return nil
		}),
	}})
	handler := plannedTestRuntime(t, host, `(function(ctx, res) { res.type("text/plain").send(ctx.request.method); })`)
	if err := host.RegisterPlanned(gojahttp.RoutePlan{Method: "ALL", Pattern: "/submit", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic}, CSRF: gojahttp.CSRFSpec{Required: true}}, handler); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}

	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/submit", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", rr.Code, rr.Body.String())
	}
	if calls != 0 {
		t.Fatalf("expected GET to skip csrf verifier, got %d calls", calls)
	}

	rr = httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/submit", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("POST status=%d body=%s", rr.Code, rr.Body.String())
	}
	if calls != 1 {
		t.Fatalf("expected POST to call csrf verifier once, got %d calls", calls)
	}
}

func TestPlannedRouteCSRFErrorBlocksHandler(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		CSRF: csrfFunc(func(context.Context, gojahttp.CSRFRequest) error {
			return errors.New("bad token")
		}),
	}})
	handler := plannedTestRuntime(t, host, `(function(_ctx, res) { res.send("should not run"); })`)
	if err := host.RegisterPlanned(gojahttp.RoutePlan{Method: "POST", Pattern: "/submit", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic}, CSRF: gojahttp.CSRFSpec{Required: true}}, handler); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/submit", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "should not run") {
		t.Fatalf("handler ran: %s", rr.Body.String())
	}
}

func TestPlannedRouteAuditsDeniedAndCompleted(t *testing.T) {
	var events []gojahttp.AuditEvent
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		Audit: auditFunc(func(_ context.Context, event gojahttp.AuditEvent) error {
			events = append(events, event)
			return nil
		}),
	}})
	handler := plannedTestRuntime(t, host, `(function(_ctx, res) { res.status(204).end(); })`)
	if err := host.RegisterPlanned(gojahttp.RoutePlan{Method: "POST", Pattern: "/submit", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic}, Audit: gojahttp.AuditSpec{Event: "submit.created"}}, handler); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/submit", nil))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(events) != 2 {
		t.Fatalf("events = %#v", events)
	}
	if events[0].Outcome != "allowed" || events[0].Event != "submit.created" {
		t.Fatalf("first event = %#v", events[0])
	}
	if events[1].Outcome != "completed" || events[1].StatusCode != http.StatusNoContent {
		t.Fatalf("second event = %#v", events[1])
	}
}

func TestPlannedRouteAuditsCSRFDenied(t *testing.T) {
	var events []gojahttp.AuditEvent
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		CSRF: csrfFunc(func(context.Context, gojahttp.CSRFRequest) error { return errors.New("bad token") }),
		Audit: auditFunc(func(_ context.Context, event gojahttp.AuditEvent) error {
			events = append(events, event)
			return nil
		}),
	}})
	handler := plannedTestRuntime(t, host, `(function(_ctx, res) { res.send("should not run"); })`)
	if err := host.RegisterPlanned(gojahttp.RoutePlan{Method: "POST", Pattern: "/submit", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic}, CSRF: gojahttp.CSRFSpec{Required: true}, Audit: gojahttp.AuditSpec{Event: "submit.created"}}, handler); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/submit", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(events) != 1 || events[0].Outcome != "denied" || events[0].StatusCode != http.StatusForbidden || !strings.Contains(events[0].Reason, "bad token") {
		t.Fatalf("events = %#v", events)
	}
}

func TestPlannedResourceRouteResolvesAndAuthorizesResource(t *testing.T) {
	var seenResourceReq gojahttp.ResourceRequest
	var seenAuthReq gojahttp.AuthorizationRequest
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
			return &gojahttp.Actor{ID: "u1", Kind: "user"}, nil
		}),
		Resources: resolverFunc(func(_ context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
			seenResourceReq = req
			return &gojahttp.ResourceRef{Name: req.Spec.Name, Type: req.Spec.Type, ID: req.ID, TenantID: req.TenantID}, nil
		}),
		Authorizer: authorizerFunc(func(_ context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
			seenAuthReq = req
			return gojahttp.AuthorizationDecision{Allowed: req.Resource != nil && req.Resource.ID == "p1"}, nil
		}),
	}})
	handler := plannedTestRuntime(t, host, `(function(ctx, res) { var project = ctx.resource("project"); res.json({ project: project.id, tenant: project.tenantId }); })`)
	tenantSource := gojahttp.ValueSource{Kind: gojahttp.ValueSourceParam, Key: "orgId"}
	if err := host.RegisterPlanned(gojahttp.RoutePlan{
		Method:   "PATCH",
		Pattern:  "/orgs/:orgId/projects/:projectId",
		Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser},
		Action:   "project.update",
		Resources: []gojahttp.ResourceSpec{{
			Name:   "project",
			Type:   "project",
			ID:     gojahttp.ValueSource{Kind: gojahttp.ValueSourceParam, Key: "projectId"},
			Tenant: &tenantSource,
		}},
	}, handler); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodPatch, "/orgs/o1/projects/p1", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if seenResourceReq.ID != "p1" || seenResourceReq.TenantID != "o1" {
		t.Fatalf("resource request = %#v", seenResourceReq)
	}
	if seenAuthReq.Resource == nil || seenAuthReq.Resource.ID != "p1" || seenAuthReq.Action != "project.update" {
		t.Fatalf("authorization request = %#v", seenAuthReq)
	}
	if !strings.Contains(rr.Body.String(), `"project":"p1"`) || !strings.Contains(rr.Body.String(), `"tenant":"o1"`) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestPlannedResourceRouteMapsNotFound(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
			return &gojahttp.Actor{ID: "u1", Kind: "user"}, nil
		}),
		Resources: resolverFunc(func(context.Context, gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
			return nil, gojahttp.ErrNotFound
		}),
		Authorizer: authorizerFunc(func(context.Context, gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
			return gojahttp.AuthorizationDecision{Allowed: true}, nil
		}),
	}})
	handler := plannedTestRuntime(t, host, `(function(_ctx, res) { res.send("should not run"); })`)
	if err := host.RegisterPlanned(gojahttp.RoutePlan{
		Method:    "GET",
		Pattern:   "/projects/:projectId",
		Security:  gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser},
		Action:    "project.read",
		Resources: []gojahttp.ResourceSpec{{Type: "project", ID: gojahttp.ValueSource{Kind: gojahttp.ValueSourceParam, Key: "projectId"}}},
	}, handler); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/projects/missing", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPlannedUserRouteMapsAuthorizerErrors(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
			return &gojahttp.Actor{ID: "u1", Kind: "user"}, nil
		}),
		Authorizer: authorizerFunc(func(context.Context, gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
			return gojahttp.AuthorizationDecision{}, errors.New("policy backend unavailable")
		}),
	}})
	handler := plannedTestRuntime(t, host, `(function(_ctx, res) { res.send("should not run"); })`)
	if err := host.RegisterPlanned(gojahttp.RoutePlan{Method: "GET", Pattern: "/me", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}, Action: "user.self.read"}, handler); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/me", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
