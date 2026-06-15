package gojahttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func TestGoAppPublicHandleJSON(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	app := gojahttp.NewApp(host)
	if err := app.Get("/healthz").Public().Audit("health.checked").HandleJSON(func(_ context.Context, sec *gojahttp.SecureContext) (any, error) {
		return map[string]any{"ok": true, "path": sec.Request.Path}, nil
	}); err != nil {
		t.Fatalf("HandleJSON: %v", err)
	}

	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"ok":true`) {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type=%q", ct)
	}
}

func TestGoAppAuthResourceCSRFAndAllow(t *testing.T) {
	csrfCalled := false
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
			return &gojahttp.Actor{ID: "u1", Kind: "user"}, nil
		}),
		CSRF: csrfFunc(func(_ context.Context, req gojahttp.CSRFRequest) error {
			csrfCalled = true
			if req.Actor == nil || req.Actor.ID != "u1" {
				t.Fatalf("csrf request actor = %#v", req.Actor)
			}
			return nil
		}),
		Resources: resolverFunc(func(_ context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
			if req.ID != "p1" || req.TenantID != "o1" {
				t.Fatalf("resource request id=%q tenant=%q", req.ID, req.TenantID)
			}
			return &gojahttp.ResourceRef{Type: "project", ID: req.ID, TenantID: req.TenantID}, nil
		}),
		Authorizer: authorizerFunc(func(_ context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
			if req.Action != "project.update" || req.Resource == nil || req.Resource.ID != "p1" {
				t.Fatalf("authorization request = %#v", req)
			}
			return gojahttp.AuthorizationDecision{Allowed: true}, nil
		}),
	}})
	app := gojahttp.NewApp(host)
	err := app.Patch("/orgs/:orgID/projects/:projectID").
		Auth(gojahttp.User().Required()).
		Resource(gojahttp.Resource("project").IDFromParam("projectID").TenantFromParam("orgID").MustExist()).
		CSRF().
		Allow("project.update").
		HandleJSON(func(_ context.Context, sec *gojahttp.SecureContext) (any, error) {
			return map[string]any{"resource": sec.Resource.ID, "actor": sec.Actor.ID}, nil
		})
	if err != nil {
		t.Fatalf("HandleJSON: %v", err)
	}

	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodPatch, "/orgs/o1/projects/p1", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"resource":"p1"`) {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !csrfCalled {
		t.Fatal("csrf was not called")
	}
}

func TestGoAppBuilderValidationErrors(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	app := gojahttp.NewApp(host)
	err := app.Get("/me").Auth(gojahttp.User().MFAFresh(10 * time.Minute)).Allow("").Handle(func(context.Context, *gojahttp.SecureContext, http.ResponseWriter, *http.Request) error {
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "requires .allow(action)") {
		t.Fatalf("expected allow validation error, got %v", err)
	}

	err = gojahttp.NewApp(nil).Get("/healthz").Public().Handle(func(context.Context, *gojahttp.SecureContext, http.ResponseWriter, *http.Request) error {
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "requires host") {
		t.Fatalf("expected nil host error, got %v", err)
	}
}
