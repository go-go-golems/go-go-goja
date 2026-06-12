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
