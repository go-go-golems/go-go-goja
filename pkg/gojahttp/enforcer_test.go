package gojahttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func TestEnforcerBuildsSecureContextWithoutHostRouter(t *testing.T) {
	enforcer := gojahttp.NewEnforcer(gojahttp.EnforcerOptions{
		Auth: gojahttp.AuthOptions{
			Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
				return &gojahttp.Actor{ID: "u1", Kind: "user"}, nil
			}),
			Resources: resolverFunc(func(_ context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
				if req.ID != "p1" {
					t.Fatalf("resource id = %q", req.ID)
				}
				return &gojahttp.ResourceRef{ID: req.ID, Type: req.Spec.Type}, nil
			}),
			Authorizer: authorizerFunc(func(_ context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
				if req.Action != "project.read" || req.Actor.ID != "u1" || req.Resource.ID != "p1" {
					t.Fatalf("authorization request = %#v", req)
				}
				return gojahttp.AuthorizationDecision{Allowed: true}, nil
			}),
		},
	})

	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodGet, "/projects/p1", nil)
	req, err := enforcer.Request(w, httpReq, map[string]string{"projectID": "p1"})
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	sec, status, err := enforcer.Enforce(context.Background(), httpReq, req, &gojahttp.RoutePlan{
		Method:   http.MethodGet,
		Pattern:  "/projects/:projectID",
		Security: gojahttp.User().Required(),
		Resources: []gojahttp.ResourceSpec{
			gojahttp.Resource("project").IDFromParam("projectID").MustExist(),
		},
		Action: "project.read",
	})
	if err != nil || status != 0 {
		t.Fatalf("Enforce status=%d err=%v", status, err)
	}
	if sec.Actor.ID != "u1" || sec.Resource.ID != "p1" || sec.Resources["project"].ID != "p1" {
		t.Fatalf("secure context = %#v", sec)
	}
}

func TestEnforcerReportsUnauthenticatedStatus(t *testing.T) {
	enforcer := gojahttp.NewEnforcer(gojahttp.EnforcerOptions{
		Auth: gojahttp.AuthOptions{
			Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
				return nil, gojahttp.ErrUnauthenticated
			}),
		},
	})
	req, err := enforcer.Request(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/me", nil), nil)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	sec, status, err := enforcer.Enforce(context.Background(), httptest.NewRequest(http.MethodGet, "/me", nil), req, &gojahttp.RoutePlan{Method: http.MethodGet, Pattern: "/me", Security: gojahttp.User().Required(), Action: "user.read"})
	if err == nil || status != http.StatusUnauthorized {
		t.Fatalf("status=%d err=%v", status, err)
	}
	if sec == nil {
		t.Fatalf("expected partial secure context")
	}
}
