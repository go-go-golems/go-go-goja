package gojahttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func TestPlannedMiddlewareUsesStandardMuxPathValues(t *testing.T) {
	handler, err := gojahttp.PlannedMiddleware(gojahttp.MiddlewareOptions{
		Auth: gojahttp.AuthOptions{
			Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
				return &gojahttp.Actor{ID: "u1", Kind: "user"}, nil
			}),
			Resources: resolverFunc(func(_ context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
				if req.ID != "p1" || req.TenantID != "o1" {
					t.Fatalf("resource request id=%q tenant=%q", req.ID, req.TenantID)
				}
				return &gojahttp.ResourceRef{Type: "project", ID: req.ID, TenantID: req.TenantID}, nil
			}),
			Authorizer: authorizerFunc(func(context.Context, gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
				return gojahttp.AuthorizationDecision{Allowed: true}, nil
			}),
		},
		ParamFunc: func(r *http.Request, name string) string { return r.PathValue(name) },
	}, gojahttp.RoutePlan{
		Method:   http.MethodGet,
		Pattern:  "/orgs/:orgID/projects/:projectID",
		Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser},
		Resources: []gojahttp.ResourceSpec{
			gojahttp.Resource("project").IDFromParam("projectID").TenantFromParam("orgID").Spec(),
		},
		Action: "project.read",
	}, func(_ context.Context, sec *gojahttp.SecureContext, w http.ResponseWriter, _ *http.Request) error {
		_, _ = w.Write([]byte(sec.Resource.ID + ":" + sec.Actor.ID))
		return nil
	})
	if err != nil {
		t.Fatalf("PlannedMiddleware: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /orgs/{orgID}/projects/{projectID}", handler)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/orgs/o1/projects/p1", nil))
	if rr.Code != http.StatusOK || strings.TrimSpace(rr.Body.String()) != "p1:u1" {
		t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
	}
}

func TestPlannedMiddlewareRejectsWrongMethod(t *testing.T) {
	handler, err := gojahttp.PlannedMiddleware(gojahttp.MiddlewareOptions{}, gojahttp.RoutePlan{Method: http.MethodGet, Pattern: "/healthz", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic}}, func(context.Context, *gojahttp.SecureContext, http.ResponseWriter, *http.Request) error {
		return nil
	})
	if err != nil {
		t.Fatalf("PlannedMiddleware: %v", err)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/healthz", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
