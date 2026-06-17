package devauth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func TestLoginSessionAuthenticateCSRFAndLogout(t *testing.T) {
	kit := New(Config{})
	cookie, csrf := login(t, kit, DefaultUsername, DefaultPassword, http.StatusOK)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(cookie)
	actor, err := kit.Authenticate(context.Background(), req, nil, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if actor.ID != DefaultUserID || actor.Kind != "user" {
		t.Fatalf("unexpected actor: %#v", actor)
	}

	csrfReq := httptest.NewRequest(http.MethodPatch, "/projects/p1", nil)
	csrfReq.AddCookie(cookie)
	csrfReq.Header.Set("X-CSRF-Token", csrf)
	if err := kit.VerifyCSRF(context.Background(), gojahttp.CSRFRequest{HTTPRequest: csrfReq, Actor: actor}); err != nil {
		t.Fatalf("verify csrf: %v", err)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/dev/logout", nil)
	logoutReq.AddCookie(cookie)
	logoutReq.Header.Set("X-CSRF-Token", csrf)
	logoutRec := httptest.NewRecorder()
	kit.LogoutHandler().ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d body=%s", logoutRec.Code, logoutRec.Body.String())
	}

	_, err = kit.Authenticate(context.Background(), req, nil, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser})
	if err == nil {
		t.Fatalf("expected unauthenticated after logout")
	}
}

func TestBadLoginAndMissingCSRF(t *testing.T) {
	kit := New(Config{})
	login(t, kit, DefaultUsername, "wrong-password", http.StatusUnauthorized)
	cookie, _ := login(t, kit, DefaultUsername, DefaultPassword, http.StatusOK)

	req := httptest.NewRequest(http.MethodPatch, "/projects/p1", nil)
	req.AddCookie(cookie)
	if err := kit.VerifyCSRF(context.Background(), gojahttp.CSRFRequest{HTTPRequest: req}); err == nil {
		t.Fatalf("expected csrf failure")
	}
}

func TestResourceResolutionAndAuthorization(t *testing.T) {
	kit := New(Config{})
	cookie, _ := login(t, kit, DefaultUsername, DefaultPassword, http.StatusOK)
	httpReq := httptest.NewRequest(http.MethodPatch, "/orgs/o1/projects/p1", nil)
	httpReq.AddCookie(cookie)
	actor, err := kit.Authenticate(context.Background(), httpReq, nil, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}

	resource, err := kit.ResolveResource(context.Background(), gojahttp.ResourceRequest{
		HTTPRequest: httpReq,
		Actor:       actor,
		Spec:        gojahttp.ResourceSpec{Name: "project", Type: "project"},
		ID:          DefaultProjectID,
		TenantID:    DefaultTenantID,
	})
	if err != nil {
		t.Fatalf("resolve resource: %v", err)
	}
	decision, err := kit.Authorize(context.Background(), gojahttp.AuthorizationRequest{Actor: actor, Action: "project.update", Resource: resource})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("expected project.update allowed: %#v", decision)
	}

	decision, err = kit.Authorize(context.Background(), gojahttp.AuthorizationRequest{Actor: actor, Action: "unknown.action", Resource: resource})
	if err != nil {
		t.Fatalf("authorize unknown: %v", err)
	}
	if decision.Allowed || decision.Reason == "" {
		t.Fatalf("expected unknown action denial with reason: %#v", decision)
	}

	_, err = kit.ResolveResource(context.Background(), gojahttp.ResourceRequest{Spec: gojahttp.ResourceSpec{Name: "project", Type: "project"}, ID: "missing", TenantID: DefaultTenantID})
	if err == nil {
		t.Fatalf("expected missing project error")
	}
}

func TestSessionHandlerAndAuditCapture(t *testing.T) {
	kit := New(Config{})
	cookie, _ := login(t, kit, DefaultUsername, DefaultPassword, http.StatusOK)

	req := httptest.NewRequest(http.MethodGet, "/auth/dev/session", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	kit.SessionHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("session status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"id":"u1"`)) {
		t.Fatalf("session body missing user id: %s", rec.Body.String())
	}

	if err := kit.RecordAudit(context.Background(), gojahttp.AuditEvent{Event: "demo", Outcome: "completed"}); err != nil {
		t.Fatalf("record audit: %v", err)
	}
	if kit.AuditCount() != 1 || len(kit.AuditEvents()) != 1 {
		t.Fatalf("unexpected audit count")
	}
}

func login(t *testing.T, kit *Kit, username, password string, wantStatus int) (*http.Cookie, string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/auth/dev/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	kit.LoginHandler().ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("login status=%d body=%s want=%d", rec.Code, rec.Body.String(), wantStatus)
	}
	if wantStatus != http.StatusOK {
		return nil, ""
	}
	var response struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if response.CSRFToken == "" {
		t.Fatalf("missing csrf token")
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("missing session cookie")
	}
	return cookies[0], response.CSRFToken
}
