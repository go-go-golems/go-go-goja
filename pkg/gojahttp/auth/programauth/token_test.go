package programauth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth"
)

func TestAPITokenIssueAuthenticateListAndRevoke(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	agents := programauth.AgentService{Store: programauth.NewMemoryAgentStore(), Now: func() time.Time { return now }, NewID: func() (string, error) { return "agt_token", nil }}
	_, err := agents.CreateAgent(ctx, programauth.AgentCreateSpec{Name: "deploy", TenantID: "o1", Policy: mustGrantSet(t, gojahttp.Grant{Action: "project.read", TenantID: "o1"})})
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}
	expiresAt := now.Add(time.Hour)
	service := programauth.APITokenService{
		Store:  programauth.NewMemoryAPITokenStore(),
		Agents: agents,
		Now:    func() time.Time { return now },
		NewID:  func() (string, error) { return "tok_1", nil },
	}
	issued, err := service.IssueAPIToken(ctx, programauth.APITokenIssueSpec{
		Name:      "deploy-key",
		AgentID:   "agt_token",
		CreatedBy: "u1",
		ExpiresAt: &expiresAt,
		Grants:    mustGrantSet(t, gojahttp.Grant{Action: "project.read", TenantID: "o1"}),
	})
	if err != nil {
		t.Fatalf("IssueAPIToken: %v", err)
	}
	if issued.Value == "" || issued.Token.ID != "tok_1" || issued.Token.CredentialHint == "" || len(issued.Token.Scopes) != 1 {
		t.Fatalf("issued = %#v", issued)
	}
	result, err := service.AuthenticateBearer(ctx, issued.Value, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser})
	if err != nil {
		t.Fatalf("AuthenticateBearer: %v", err)
	}
	if result.Method != gojahttp.AuthMethodAPIToken || result.PrincipalKind != gojahttp.PrincipalKindAgent || result.PrincipalID != "agt_token" || result.CredentialID != "tok_1" || result.CSRFRequired {
		t.Fatalf("auth result = %#v", result)
	}
	if !result.Grants.AllowsResource("project.read", "o1", "project", "p1") {
		t.Fatalf("auth grants = %#v", result.Grants)
	}
	listed, err := service.ListAPITokens(ctx, programauth.APITokenQuery{AgentID: "agt_token"})
	if err != nil {
		t.Fatalf("ListAPITokens: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != "tok_1" || listed[0].CredentialHint == "" {
		t.Fatalf("listed = %#v", listed)
	}
	if _, err := service.RevokeAPIToken(ctx, "tok_1"); err != nil {
		t.Fatalf("RevokeAPIToken: %v", err)
	}
	if _, err := service.AuthenticateBearer(ctx, issued.Value, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}); !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("revoked auth err=%v", err)
	}
}

func TestAPITokenAuthenticationRejectsExpiredAndDisabledAgent(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	agents := programauth.AgentService{Store: programauth.NewMemoryAgentStore(), Now: func() time.Time { return now }, NewID: func() (string, error) { return "agt_exp", nil }}
	_, err := agents.CreateAgent(ctx, programauth.AgentCreateSpec{Name: "deploy", Policy: mustGrantSet(t, gojahttp.Grant{Action: "project.read"})})
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}
	expiresAt := now.Add(time.Minute)
	current := now
	service := programauth.APITokenService{Store: programauth.NewMemoryAPITokenStore(), Agents: agents, Now: func() time.Time { return current }, NewID: func() (string, error) { return "tok_exp", nil }}
	issued, err := service.IssueAPIToken(ctx, programauth.APITokenIssueSpec{Name: "short", AgentID: "agt_exp", ExpiresAt: &expiresAt, Grants: mustGrantSet(t, gojahttp.Grant{Action: "project.read"})})
	if err != nil {
		t.Fatalf("IssueAPIToken: %v", err)
	}
	current = expiresAt
	if _, err := service.AuthenticateBearer(ctx, issued.Value, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}); !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("expired err=%v", err)
	}
	current = now
	if _, err := agents.DisableAgent(ctx, "agt_exp"); err != nil {
		t.Fatalf("DisableAgent: %v", err)
	}
	if _, err := service.AuthenticateBearer(ctx, issued.Value, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}); !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("disabled err=%v", err)
	}
}

func TestBearerFromHeaderRejectsMalformedAndAlternateTransports(t *testing.T) {
	if token, ok, err := programauth.BearerFromHeader(httptest.NewRequest(http.MethodGet, "/", nil)); err != nil || ok || token != "" {
		t.Fatalf("missing header token=%q ok=%v err=%v", token, ok, err)
	}
	for _, req := range []*http.Request{
		headerRequest("Basic abc"),
		headerRequest("Bearer"),
		headerRequest("Bearer abc def"),
		headerRequest("Bearer   "),
		headerRequest("Bearer abc"),
	} {
		if req.Header.Get("Authorization") == "Bearer abc" {
			req.Header.Add("Authorization", "Bearer def")
		}
		if _, _, err := programauth.BearerFromHeader(req); !errors.Is(err, gojahttp.ErrUnauthenticated) {
			t.Fatalf("expected unauthenticated for %q err=%v", req.Header.Values("Authorization"), err)
		}
	}
	queryReq := httptest.NewRequest(http.MethodGet, "/?access_token=abc", nil)
	if _, _, err := programauth.BearerFromHeader(queryReq); !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("query token err=%v", err)
	}
	valid := headerRequest("Bearer ggpat_00112233_" + stringsRepeat("a", 64))
	if token, ok, err := programauth.BearerFromHeader(valid); err != nil || !ok || token == "" {
		t.Fatalf("valid token=%q ok=%v err=%v", token, ok, err)
	}
}

func headerRequest(header string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", header)
	return req
}

func stringsRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
