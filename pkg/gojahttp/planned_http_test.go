package gojahttp_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func TestPlannedHTTPPublicRouteDoesNotRequireRuntime(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	if err := host.RegisterPlannedHTTP(gojahttp.RoutePlan{Method: "GET", Pattern: "/hello/:name", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic}}, func(_ context.Context, sec *gojahttp.SecureContext, w http.ResponseWriter, _ *http.Request) error {
		if sec.Params["name"] != "go" {
			t.Fatalf("params = %#v", sec.Params)
		}
		_, _ = w.Write([]byte("hello " + sec.Params["name"]))
		return nil
	}); err != nil {
		t.Fatalf("RegisterPlannedHTTP: %v", err)
	}

	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/hello/go", nil))
	if rr.Code != http.StatusOK || strings.TrimSpace(rr.Body.String()) != "hello go" {
		t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
	}
}

func TestPlannedHTTPAuthenticatedRouteUsesSecureContext(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
			return &gojahttp.Actor{ID: "u1", Kind: "user"}, nil
		}),
		Authorizer: authorizerFunc(func(_ context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
			if req.Actor == nil || req.Actor.ID != "u1" || req.Action != "user.self.read" {
				t.Fatalf("authorization request = %#v", req)
			}
			return gojahttp.AuthorizationDecision{Allowed: true}, nil
		}),
	}})
	if err := host.RegisterPlannedHTTP(gojahttp.RoutePlan{Method: "GET", Pattern: "/me", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}, Action: "user.self.read"}, func(_ context.Context, sec *gojahttp.SecureContext, w http.ResponseWriter, _ *http.Request) error {
		_, _ = w.Write([]byte(sec.Actor.ID))
		return nil
	}); err != nil {
		t.Fatalf("RegisterPlannedHTTP: %v", err)
	}

	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/me", nil))
	if rr.Code != http.StatusOK || strings.TrimSpace(rr.Body.String()) != "u1" {
		t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
	}
}

func TestPlannedHTTPCSRFErrorBlocksHandler(t *testing.T) {
	called := false
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		CSRF: csrfFunc(func(context.Context, gojahttp.CSRFRequest) error { return errors.New("bad csrf") }),
	}})
	if err := host.RegisterPlannedHTTP(gojahttp.RoutePlan{Method: "POST", Pattern: "/submit", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic}, CSRF: gojahttp.CSRFSpec{Required: true}}, func(context.Context, *gojahttp.SecureContext, http.ResponseWriter, *http.Request) error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("RegisterPlannedHTTP: %v", err)
	}

	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/submit", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if called {
		t.Fatal("handler should not run after csrf denial")
	}
}

func TestPlannedHTTPHandlerErrorAuditsFailed(t *testing.T) {
	outcomes := []string{}
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		Audit: auditFunc(func(_ context.Context, event gojahttp.AuditEvent) error {
			outcomes = append(outcomes, event.Outcome)
			return nil
		}),
	}})
	if err := host.RegisterPlannedHTTP(gojahttp.RoutePlan{Method: "GET", Pattern: "/boom", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic}, Audit: gojahttp.AuditSpec{Event: "boom"}}, func(context.Context, *gojahttp.SecureContext, http.ResponseWriter, *http.Request) error {
		return errors.New("boom")
	}); err != nil {
		t.Fatalf("RegisterPlannedHTTP: %v", err)
	}

	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/boom", nil))
	if rr.Code != http.StatusInternalServerError || !strings.Contains(rr.Body.String(), "boom") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Join(outcomes, ",") != "allowed,failed" {
		t.Fatalf("outcomes=%v", outcomes)
	}
}
