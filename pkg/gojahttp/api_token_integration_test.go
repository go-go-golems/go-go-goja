package gojahttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth"
)

func mustRouteGrantSet(t *testing.T, grants ...gojahttp.Grant) gojahttp.GrantSet {
	t.Helper()
	set, err := gojahttp.NewGrantSet(grants...)
	if err != nil {
		t.Fatalf("NewGrantSet: %v", err)
	}
	return set
}

func TestAPITokenAuthenticatesPlannedRouteSkipsCSRFAndChecksGrants(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	agents := programauth.AgentService{Store: programauth.NewMemoryAgentStore(), Now: func() time.Time { return now }, NewID: func() (string, error) { return "agt_route", nil }}
	_, err := agents.CreateAgent(ctx, programauth.AgentCreateSpec{Name: "route bot", TenantID: "o1", Policy: mustRouteGrantSet(t, gojahttp.Grant{Action: "project.read", TenantID: "o1"})})
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}
	tokens := programauth.APITokenService{Store: programauth.NewMemoryAPITokenStore(), Agents: agents, Now: func() time.Time { return now }, NewID: func() (string, error) { return "tok_route", nil }}
	issued, err := tokens.IssueAPIToken(ctx, programauth.APITokenIssueSpec{Name: "route", AgentID: "agt_route", Grants: mustRouteGrantSet(t, gojahttp.Grant{Action: "project.read", TenantID: "o1"})})
	if err != nil {
		t.Fatalf("IssueAPIToken: %v", err)
	}
	csrfCalled := false
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
		Authenticator: programauth.CompositeAuthenticator{APITokens: tokens},
		CSRF: csrfFunc(func(context.Context, gojahttp.CSRFRequest) error {
			csrfCalled = true
			return nil
		}),
		Resources: resolverFunc(func(_ context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
			return &gojahttp.ResourceRef{Name: req.Spec.Name, Type: req.Spec.Type, ID: req.ID, TenantID: req.TenantID}, nil
		}),
		Authorizer: authorizerFunc(func(_ context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
			return gojahttp.AuthorizationDecision{Allowed: req.Actor != nil && req.Actor.ID == "agt_route"}, nil
		}),
	}})
	if err := host.RegisterPlannedHTTP(gojahttp.RoutePlan{
		Method:   http.MethodPost,
		Pattern:  "/orgs/:orgID/projects/:projectID",
		Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser},
		CSRF:     gojahttp.CSRFSpec{Required: true},
		Resources: []gojahttp.ResourceSpec{
			gojahttp.Resource("project").IDFromParam("projectID").TenantFromParam("orgID").MustExist(),
		},
		Action: "project.read",
	}, func(_ context.Context, sec *gojahttp.SecureContext, w http.ResponseWriter, _ *http.Request) error {
		_, _ = w.Write([]byte(sec.Auth.CredentialHint + ":" + sec.Actor.ID))
		return nil
	}); err != nil {
		t.Fatalf("RegisterPlannedHTTP read: %v", err)
	}
	if err := host.RegisterPlannedHTTP(gojahttp.RoutePlan{Method: http.MethodPost, Pattern: "/orgs/:orgID/projects/:projectID/delete", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}, Resources: []gojahttp.ResourceSpec{gojahttp.Resource("project").IDFromParam("projectID").TenantFromParam("orgID").MustExist()}, Action: "project.delete"}, func(context.Context, *gojahttp.SecureContext, http.ResponseWriter, *http.Request) error {
		t.Fatal("delete handler should not run")
		return nil
	}); err != nil {
		t.Fatalf("RegisterPlannedHTTP delete: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orgs/o1/projects/p1", nil)
	req.Header.Set("Authorization", "Bearer "+issued.Value)
	host.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "agt_route") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if csrfCalled {
		t.Fatal("csrf should be skipped for API-token auth")
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/orgs/o1/projects/p1/delete", nil)
	req.Header.Set("Authorization", "Bearer "+issued.Value)
	host.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("delete status=%d body=%s", rr.Code, rr.Body.String())
	}
}
