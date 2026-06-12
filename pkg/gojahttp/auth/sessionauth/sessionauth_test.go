package sessionauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func TestAuthenticateAndCSRF(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	manager, err := New(Config{Store: store, AllowInsecureHTTP: true})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	session, err := manager.NewSession(ctx, "u1", WithEmail("demo@example.test", true), WithTenantIDs("o1"))
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	resp := httptest.NewRecorder()
	manager.SetCookie(resp, session.ID)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	for _, cookie := range resp.Result().Cookies() {
		req.AddCookie(cookie)
	}
	actor, err := manager.Authenticate(ctx, req, nil, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if actor.ID != "u1" || actor.Claims["email"] != "demo@example.test" {
		t.Fatalf("unexpected actor: %#v", actor)
	}

	csrfReq := httptest.NewRequest(http.MethodPatch, "/projects/p1", nil)
	for _, cookie := range resp.Result().Cookies() {
		csrfReq.AddCookie(cookie)
	}
	csrfReq.Header.Set(CSRFHeaderName, session.CSRFToken)
	if err := manager.VerifyCSRF(ctx, gojahttp.CSRFRequest{HTTPRequest: csrfReq, Actor: actor}); err != nil {
		t.Fatalf("verify csrf: %v", err)
	}
}

func TestAuthenticateFailures(t *testing.T) {
	ctx := context.Background()
	manager, err := New(Config{Store: NewMemoryStore(), AllowInsecureHTTP: true})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	_, err = manager.Authenticate(ctx, httptest.NewRequest(http.MethodGet, "/me", nil), nil, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser})
	if !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("missing cookie err=%v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: manager.cookieName, Value: "not valid"})
	_, err = manager.Authenticate(ctx, req, nil, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser})
	if !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("invalid cookie err=%v", err)
	}
}

func TestAuthenticateRequiresFreshMFA(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	manager, err := New(Config{Store: NewMemoryStore(), AllowInsecureHTTP: true, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	spec := gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser, MFAFreshWithin: 10 * time.Minute}

	withoutMFA, err := manager.NewSession(ctx, "u1")
	if err != nil {
		t.Fatalf("new session without mfa: %v", err)
	}
	_, err = manager.Authenticate(ctx, requestWithCookie(manager.cookieName, withoutMFA.ID), nil, spec)
	if !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("missing mfa err=%v", err)
	}

	staleMFA, err := manager.NewSession(ctx, "u2", WithMFAAt(now.Add(-11*time.Minute)))
	if err != nil {
		t.Fatalf("new session with stale mfa: %v", err)
	}
	_, err = manager.Authenticate(ctx, requestWithCookie(manager.cookieName, staleMFA.ID), nil, spec)
	if !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("stale mfa err=%v", err)
	}

	freshMFA, err := manager.NewSession(ctx, "u3", WithMFAAt(now.Add(-9*time.Minute)))
	if err != nil {
		t.Fatalf("new session with fresh mfa: %v", err)
	}
	actor, err := manager.Authenticate(ctx, requestWithCookie(manager.cookieName, freshMFA.ID), nil, spec)
	if err != nil {
		t.Fatalf("fresh mfa authenticate: %v", err)
	}
	if actor.ID != "u3" {
		t.Fatalf("unexpected actor: %#v", actor)
	}
}

func TestExpiredRevokedAndRotatedSessions(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	manager, err := New(Config{Store: NewMemoryStore(), AllowInsecureHTTP: true, IdleTimeout: time.Minute, AbsoluteTimeout: time.Hour, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	session, err := manager.NewSession(ctx, "u1")
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	resp := httptest.NewRecorder()
	manager.SetCookie(resp, session.ID)
	req := requestWithCookies(resp.Result().Cookies())

	now = now.Add(2 * time.Minute)
	_, err = manager.Authenticate(ctx, req, nil, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser})
	if !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("expired err=%v", err)
	}

	now = time.Date(2026, 6, 12, 13, 0, 0, 0, time.UTC)
	active, err := manager.NewSession(ctx, "u2")
	if err != nil {
		t.Fatalf("new active session: %v", err)
	}
	if err := manager.store.Revoke(ctx, active.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	req = requestWithCookie(manager.cookieName, active.ID)
	_, err = manager.Authenticate(ctx, req, nil, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser})
	if !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("revoked err=%v", err)
	}

	next := Session{ID: mustToken(t), UserID: "u3", CSRFToken: mustToken(t), CreatedAt: now, LastSeenAt: now, IdleExpiresAt: now.Add(time.Minute), AbsoluteExpiresAt: now.Add(time.Hour)}
	if err := manager.store.Rotate(ctx, active.ID, next); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if _, err := manager.store.Get(ctx, active.ID); err == nil {
		t.Fatalf("old session should be gone after rotate")
	}
	if _, err := manager.store.Get(ctx, next.ID); err != nil {
		t.Fatalf("next session missing after rotate: %v", err)
	}
}

func TestCSRFMismatchAndCookieClearing(t *testing.T) {
	ctx := context.Background()
	manager, err := New(Config{Store: NewMemoryStore(), AllowInsecureHTTP: true})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	session, err := manager.NewSession(ctx, "u1")
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	req := requestWithCookie(manager.cookieName, session.ID)
	req.Header.Set(CSRFHeaderName, "wrong-token")
	if err := manager.VerifyCSRF(ctx, gojahttp.CSRFRequest{HTTPRequest: req, Actor: &gojahttp.Actor{ID: "u1"}}); err == nil {
		t.Fatalf("expected csrf mismatch")
	}

	resp := httptest.NewRecorder()
	manager.ClearCookie(resp)
	cookies := resp.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge != -1 {
		t.Fatalf("expected clearing cookie, got %#v", cookies)
	}
}

func TestCustomActorLoader(t *testing.T) {
	ctx := context.Background()
	manager, err := New(Config{
		Store:             NewMemoryStore(),
		AllowInsecureHTTP: true,
		ActorLoader: ActorLoaderFunc(func(_ context.Context, session *Session) (*gojahttp.Actor, error) {
			return &gojahttp.Actor{ID: "actor-" + session.UserID, Kind: "service"}, nil
		}),
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	session, err := manager.NewSession(ctx, "u1")
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	actor, err := manager.Authenticate(ctx, requestWithCookie(manager.cookieName, session.ID), nil, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if actor.ID != "actor-u1" || actor.Kind != "service" {
		t.Fatalf("unexpected actor: %#v", actor)
	}
}

func requestWithCookies(cookies []*http.Cookie) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	return req
}

func requestWithCookie(name, value string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: name, Value: value})
	return req
}

func mustToken(t *testing.T) string {
	t.Helper()
	token, err := RandomToken()
	if err != nil {
		t.Fatal(err)
	}
	return token
}
