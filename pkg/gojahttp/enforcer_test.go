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
	if sec.Auth.Method != gojahttp.AuthMethodSession || sec.Auth.PrincipalKind != gojahttp.PrincipalKindUser || sec.Auth.PrincipalID != "u1" || !sec.Auth.CSRFRequired {
		t.Fatalf("legacy authenticator auth result = %#v", sec.Auth)
	}
}

func TestEnforcerAppliesRouteAuthRequirements(t *testing.T) {
	for _, tt := range []struct {
		name     string
		spec     gojahttp.SecuritySpec
		auth     gojahttp.AuthResult
		wantCode int
	}{
		{
			name: "agent accepts api token agent",
			spec: gojahttp.Agent(),
			auth: gojahttp.AuthResult{Actor: &gojahttp.Actor{ID: "agt_1", Kind: "agent"}, Method: gojahttp.AuthMethodAPIToken, PrincipalKind: gojahttp.PrincipalKindAgent, PrincipalID: "agt_1", CSRFRequired: false},
		},
		{
			name:     "agent rejects session user",
			spec:     gojahttp.Agent(),
			auth:     gojahttp.AuthResult{Actor: &gojahttp.Actor{ID: "u1", Kind: "user"}, Method: gojahttp.AuthMethodSession, PrincipalKind: gojahttp.PrincipalKindUser, PrincipalID: "u1", CSRFRequired: true},
			wantCode: http.StatusForbidden,
		},
		{
			name: "anyOf accepts session user",
			spec: gojahttp.AnyOf(gojahttp.Agent(), gojahttp.SessionUser()),
			auth: gojahttp.AuthResult{Actor: &gojahttp.Actor{ID: "u1", Kind: "user"}, Method: gojahttp.AuthMethodSession, PrincipalKind: gojahttp.PrincipalKindUser, PrincipalID: "u1", CSRFRequired: true},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			enforcer := gojahttp.NewEnforcer(gojahttp.EnforcerOptions{Auth: gojahttp.AuthOptions{
				Authenticator: resultAuthenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (gojahttp.AuthResult, error) {
					return tt.auth, nil
				}),
				Authorizer: authorizerFunc(func(context.Context, gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
					return gojahttp.AuthorizationDecision{Allowed: true}, nil
				}),
			}})
			req, err := enforcer.Request(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/route", nil), nil)
			if err != nil {
				t.Fatalf("Request: %v", err)
			}
			sec, status, err := enforcer.Enforce(context.Background(), httptest.NewRequest(http.MethodGet, "/route", nil), req, &gojahttp.RoutePlan{Method: http.MethodGet, Pattern: "/route", Security: tt.spec, Action: "route.read"})
			if tt.wantCode == 0 {
				if err != nil || status != 0 {
					t.Fatalf("Enforce status=%d err=%v", status, err)
				}
				return
			}
			if err == nil || status != tt.wantCode {
				t.Fatalf("Enforce status=%d err=%v", status, err)
			}
			if sec.Auth.PrincipalID != tt.auth.PrincipalID {
				t.Fatalf("expected denied context to retain auth result: %#v", sec.Auth)
			}
		})
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
