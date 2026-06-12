package express

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules/uidsl"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

type expressAuthenticatorFunc func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error)

func (f expressAuthenticatorFunc) Authenticate(ctx context.Context, req *http.Request, session *gojahttp.SessionDTO, spec gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
	return f(ctx, req, session, spec)
}

type expressResolverFunc func(context.Context, gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error)

func (f expressResolverFunc) ResolveResource(ctx context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
	return f(ctx, req)
}

type expressAuthorizerFunc func(context.Context, gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error)

func (f expressAuthorizerFunc) Authorize(ctx context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
	return f(ctx, req)
}

func newExpressAuthRuntime(t *testing.T, host *gojahttp.Host) *engine.Runtime {
	t.Helper()
	factory, err := engine.NewRuntimeFactoryBuilder().WithModules(NewRegistrar(host), uidsl.NewRegistrar()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(engine.WithStartupContext(context.Background()), engine.WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	t.Cleanup(func() { _ = rt.Close(context.Background()) })
	host.SetRuntime(rt.Owner)
	return rt
}

func runExpressAuthScript(t *testing.T, rt *engine.Runtime, script string) {
	t.Helper()
	_, err := rt.Owner.Call(context.Background(), "load-test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(script)
		return nil, err
	})
	if err != nil {
		t.Fatalf("load script: %v", err)
	}
}

func TestExpressPlannedPublicRouteBuilder(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Renderer: uidsl.RenderAny})
	rt := newExpressAuthRuntime(t, host)
	runExpressAuthScript(t, rt, `
		const express = require("express");
		const app = express.app();
		app.get("/planned/:name")
		  .public()
		  .handle((ctx, res) => res.type("text/plain").send("hello " + ctx.params.name));
	`)
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/planned/goja", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.TrimSpace(rr.Body.String()) != "hello goja" {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestExpressGenericRouteBuilderStillWorks(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Renderer: uidsl.RenderAny})
	rt := newExpressAuthRuntime(t, host)
	runExpressAuthScript(t, rt, `
		const express = require("express");
		const app = express.app();
		app.route("GET", "/generic/:name")
		  .public()
		  .handle((ctx, res) => res.json({ hello: ctx.params.name }));
	`)
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/generic/goja", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"hello":"goja"`) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestExpressPlannedAuthRouteBuilder(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		Authenticator: expressAuthenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
			return &gojahttp.Actor{ID: "u1", Kind: "user"}, nil
		}),
		Authorizer: expressAuthorizerFunc(func(_ context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
			return gojahttp.AuthorizationDecision{Allowed: req.Actor != nil && req.Action == "user.self.read"}, nil
		}),
	}})
	rt := newExpressAuthRuntime(t, host)
	runExpressAuthScript(t, rt, `
		const express = require("express");
		const app = express.app();
		app.get("/me")
		  .auth(express.user().required())
		  .allow("user.self.read")
		  .handle((ctx, res) => res.json({ actor: ctx.actor.id, action: ctx.action }));
	`)
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/me", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"actor":"u1"`) || !strings.Contains(rr.Body.String(), `"action":"user.self.read"`) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestExpressPlannedResourceRouteBuilder(t *testing.T) {
	var seenResourceReq gojahttp.ResourceRequest
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		Authenticator: expressAuthenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
			return &gojahttp.Actor{ID: "u1", Kind: "user"}, nil
		}),
		Resources: expressResolverFunc(func(_ context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
			seenResourceReq = req
			return &gojahttp.ResourceRef{Name: req.Spec.Name, Type: req.Spec.Type, ID: req.ID, TenantID: req.TenantID}, nil
		}),
		Authorizer: expressAuthorizerFunc(func(_ context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
			return gojahttp.AuthorizationDecision{Allowed: req.Resource != nil && req.Resource.ID == "p1"}, nil
		}),
	}})
	rt := newExpressAuthRuntime(t, host)
	runExpressAuthScript(t, rt, `
		const express = require("express");
		const app = express.app();
		app.patch("/orgs/:orgId/projects/:projectId")
		  .auth(express.user().required())
		  .resource(express.resource("project").idFromParam("projectId").tenantFromParam("orgId").mustExist())
		  .allow("project.update")
		  .handle((ctx, res) => {
		    const project = ctx.resource("project");
		    res.json({ project: project.id, tenant: project.tenantId });
		  });
	`)
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodPatch, "/orgs/o1/projects/p1", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if seenResourceReq.ID != "p1" || seenResourceReq.TenantID != "o1" {
		t.Fatalf("resource request = %#v", seenResourceReq)
	}
	if !strings.Contains(rr.Body.String(), `"project":"p1"`) || !strings.Contains(rr.Body.String(), `"tenant":"o1"`) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestExpressPlannedBuilderSupportsCSRFAudit(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	rt := newExpressAuthRuntime(t, host)
	runExpressAuthScript(t, rt, `
		const express = require("express");
		const app = express.app();
		app.post("/contact")
		  .public()
		  .csrf()
		  .audit("contact.submitted")
		  .handle((_ctx, res) => res.json({ ok: true }));
	`)
	routes := host.Routes()
	if len(routes) != 1 {
		t.Fatalf("routes = %#v", routes)
	}
	if !routes[0].Planned || !routes[0].CSRFRequired || routes[0].AuditEvent != "contact.submitted" {
		t.Fatalf("route descriptor = %#v", routes[0])
	}
}

func TestExpressPlannedBuilderRejectsPlainAuthObject(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	rt := newExpressAuthRuntime(t, host)
	_, err := rt.Owner.Call(context.Background(), "load-test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const express = require("express");
			const app = express.app();
			app.get("/bad")
			  .auth({ required: true })
			  .allow("bad.read")
			  .handle(() => "bad");
		`)
		return nil, err
	})
	if err == nil {
		t.Fatal("expected plain auth object to be rejected")
	}
	if !strings.Contains(err.Error(), "express.user") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExpressVerbHelperRejectsLegacyHandlerOverload(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	rt := newExpressAuthRuntime(t, host)
	_, err := rt.Owner.Call(context.Background(), "load-test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const express = require("express");
			const app = express.app();
			app.get("/bad", (_req, res) => res.json({ ok: true }));
		`)
		return nil, err
	})
	if err == nil {
		t.Fatal("expected legacy handler overload to be rejected")
	}
	if !strings.Contains(err.Error(), "public().handle") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExpressPlannedBuilderRequiresSecurityBeforeHandle(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	rt := newExpressAuthRuntime(t, host)
	_, err := rt.Owner.Call(context.Background(), "load-test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const express = require("express");
			const app = express.app();
			app.get("/bad").handle(() => "bad");
		`)
		return nil, err
	})
	if err == nil {
		t.Fatal("expected handle to be unavailable before security mode")
	}
	if !strings.Contains(err.Error(), "handle") {
		t.Fatalf("unexpected error: %v", err)
	}
}
