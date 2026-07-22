package oidcauth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
	"golang.org/x/oauth2"
)

func TestLoginCallbackCreatesSession(t *testing.T) {
	ctx := context.Background()
	provider := newFakeProvider(t)
	defer provider.Close()
	store := sessionauth.NewMemoryStore()
	manager, err := sessionauth.New(sessionauth.Config{Store: store, AllowInsecureHTTP: true})
	if err != nil {
		t.Fatalf("session manager: %v", err)
	}
	var gotClaims OIDCClaims
	var handlers *Handlers
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			handlers.LoginHandler().ServeHTTP(w, r)
		case "/auth/register":
			handlers.RegistrationHandler().ServeHTTP(w, r)
		case "/auth/callback":
			handlers.CallbackHandler().ServeHTTP(w, r)
		case "/after":
			_, _ = w.Write([]byte("after login"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer app.Close()
	handlers, err = New(ctx, Config{
		IssuerURL:      provider.URL(),
		ClientID:       "goja-app",
		RedirectURL:    app.URL + "/auth/callback",
		AfterLoginURL:  "/after",
		SessionManager: manager,
		UserNormalizer: UserNormalizerFunc(func(_ context.Context, claims OIDCClaims) (UserSession, error) {
			gotClaims = claims
			return UserSession{UserID: "u1", Email: claims.Email, EmailVerified: claims.EmailVerified, TenantIDs: []string{"o1"}, Claims: map[string]any{"sub": claims.Subject}}, nil
		}),
	})
	if err != nil {
		t.Fatalf("keycloak handlers: %v", err)
	}
	client := clientWithJar(t)
	resp, err := client.Get(app.URL + "/auth/login?return_to=/after")
	if err != nil {
		t.Fatalf("login flow: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK || resp.Request.URL.Path != "/after" {
		t.Fatalf("final status/path = %d %s", resp.StatusCode, resp.Request.URL.Path)
	}
	if gotClaims.Subject != "keycloak-sub-1" || gotClaims.Email != "demo@example.test" || !gotClaims.EmailVerified {
		t.Fatalf("unexpected claims: %#v", gotClaims)
	}
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	for _, cookie := range client.Jar.Cookies(mustURL(t, app.URL)) {
		req.AddCookie(cookie)
	}
	actor, err := manager.Authenticate(ctx, req, nil, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser})
	if err != nil {
		t.Fatalf("authenticate session: %v", err)
	}
	if actor.ID != "u1" || actor.Claims["sub"] != "keycloak-sub-1" {
		t.Fatalf("unexpected actor: %#v", actor)
	}
	registerResponse, err := clientWithJar(t).Get(app.URL + "/auth/register?return_to=/after")
	if err != nil {
		t.Fatalf("registration flow: %v", err)
	}
	_ = registerResponse.Body.Close()
	provider.mu.Lock()
	registrationRequested := provider.registrationRequested
	provider.mu.Unlock()
	if !registrationRequested {
		t.Fatal("registration flow did not send tinyidp_signup=1")
	}
}

func TestInProcessIssuerClientCoversDiscoveryExchangeAndKeysWithoutDial(t *testing.T) {
	ctx := context.Background()
	provider := newInProcessFakeProvider(t, "https://identity.example.test/idp")
	transport, err := NewInProcessIssuerTransport(provider.URL(), provider.Handler())
	if err != nil {
		t.Fatalf("transport: %v", err)
	}
	manager, err := sessionauth.New(sessionauth.Config{Store: sessionauth.NewMemoryStore(), AllowInsecureHTTP: true})
	if err != nil {
		t.Fatalf("session manager: %v", err)
	}
	var handlers *Handlers
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			handlers.LoginHandler().ServeHTTP(w, r)
		case "/auth/callback":
			handlers.CallbackHandler().ServeHTTP(w, r)
		case "/after":
			_, _ = w.Write([]byte("after login"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer app.Close()
	handlers, err = New(ctx, Config{
		IssuerURL:      provider.URL(),
		ClientID:       "goja-app",
		RedirectURL:    app.URL + "/auth/callback",
		AfterLoginURL:  "/after",
		SessionManager: manager,
		UserNormalizer: passthroughNormalizer(),
		HTTPClient:     &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatalf("handlers: %v", err)
	}

	client := clientWithJar(t)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Host == "identity.example.test" {
			return http.ErrUseLastResponse
		}
		return nil
	}

	loginResponse, err := client.Get(app.URL + "/auth/login?return_to=/after")
	if err != nil {
		t.Fatalf("login redirect: %v", err)
	}
	defer func() { _ = loginResponse.Body.Close() }()
	if loginResponse.StatusCode != http.StatusFound {
		t.Fatalf("login status=%d", loginResponse.StatusCode)
	}

	authorizeRequest, err := http.NewRequest(http.MethodGet, loginResponse.Header.Get("Location"), nil)
	if err != nil {
		t.Fatalf("authorize request: %v", err)
	}
	authorizeResponse, err := transport.RoundTrip(authorizeRequest)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	_ = authorizeResponse.Body.Close()
	callback := authorizeResponse.Header.Get("Location")
	if callback == "" {
		t.Fatal("authorize response did not contain callback")
	}
	callbackResponse, err := client.Get(callback)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	defer func() { _ = callbackResponse.Body.Close() }()
	if callbackResponse.StatusCode != http.StatusOK || callbackResponse.Request.URL.Path != "/after" {
		t.Fatalf("callback status/path=%d %s", callbackResponse.StatusCode, callbackResponse.Request.URL.Path)
	}
}

func TestPublicClientUsesTokenEndpointParametersWithoutProbe(t *testing.T) {
	provider := newInProcessFakeProvider(t, "https://identity.example.test/idp")
	transport, err := NewInProcessIssuerTransport(provider.URL(), provider.Handler())
	if err != nil {
		t.Fatal(err)
	}
	manager, err := sessionauth.New(sessionauth.Config{Store: sessionauth.NewMemoryStore(), AllowInsecureHTTP: true})
	if err != nil {
		t.Fatal(err)
	}
	handlers, err := New(context.Background(), Config{
		IssuerURL:      provider.URL(),
		ClientID:       "public-app",
		RedirectURL:    "http://127.0.0.1:8787/auth/callback",
		SessionManager: manager,
		UserNormalizer: passthroughNormalizer(),
		HTTPClient:     &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatal(err)
	}
	if handlers.oauth2Config.Endpoint.AuthStyle != oauth2.AuthStyleInParams {
		t.Fatalf("auth style = %v, want body parameters", handlers.oauth2Config.Endpoint.AuthStyle)
	}
}

func TestInProcessIssuerTransportFailsClosed(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	transport, err := NewInProcessIssuerTransport("https://identity.example.test/idp", handler)
	if err != nil {
		t.Fatalf("transport: %v", err)
	}
	for _, rawURL := range []string{
		"https://other.example.test/idp/keys",
		"http://identity.example.test/idp/keys",
		"https://identity.example.test/application",
		"https://user@identity.example.test/idp/keys",
		"/idp/keys",
	} {
		req, requestErr := http.NewRequest(http.MethodGet, rawURL, nil)
		if requestErr != nil {
			t.Fatalf("request %q: %v", rawURL, requestErr)
		}
		if _, roundTripErr := transport.RoundTrip(req); roundTripErr == nil {
			t.Errorf("RoundTrip(%q) succeeded, want fail-closed error", rawURL)
		}
	}
}

func TestInProcessIssuerTransportProvidesServerRequestMetadata(t *testing.T) {
	t.Parallel()
	var remoteAddr string
	var requestURI string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteAddr = r.RemoteAddr
		requestURI = r.RequestURI
		w.WriteHeader(http.StatusNoContent)
	})
	transport, err := NewInProcessIssuerTransport("https://identity.example.test/idp", handler)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, "https://identity.example.test/idp/token?trace=1", strings.NewReader("grant_type=authorization_code"))
	if err != nil {
		t.Fatal(err)
	}
	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if remoteAddr != "127.0.0.1:0" || requestURI != "/idp/token?trace=1" {
		t.Fatalf("server metadata remote=%q requestURI=%q", remoteAddr, requestURI)
	}
}

func TestNewInProcessIssuerTransportRejectsMalformedIssuer(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	for _, issuer := range []string{"", "/idp", "ftp://identity.example.test/idp", "https://user@identity.example.test/idp", "https://identity.example.test/idp?q=1", "https://identity.example.test/idp#fragment"} {
		if _, err := NewInProcessIssuerTransport(issuer, handler); err == nil {
			t.Errorf("issuer %q accepted, want error", issuer)
		}
	}
	if _, err := NewInProcessIssuerTransport("https://identity.example.test/idp", nil); err == nil {
		t.Error("nil handler accepted, want error")
	}
}

func TestCallbackRejectsBadStateAndNonce(t *testing.T) {
	ctx := context.Background()
	provider := newFakeProvider(t)
	defer provider.Close()
	manager, err := sessionauth.New(sessionauth.Config{Store: sessionauth.NewMemoryStore(), AllowInsecureHTTP: true})
	if err != nil {
		t.Fatalf("session manager: %v", err)
	}
	handlers, err := New(ctx, Config{IssuerURL: provider.URL(), ClientID: "goja-app", RedirectURL: "http://app.test/auth/callback", SessionManager: manager, UserNormalizer: passthroughNormalizer()})
	if err != nil {
		t.Fatalf("handlers: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/callback?state=missing&code=code", nil)
	handlers.CallbackHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad state status=%d body=%s", rec.Code, rec.Body.String())
	}

	provider.wrongNonce = true
	var nonceHandlers *Handlers
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			nonceHandlers.LoginHandler().ServeHTTP(w, r)
		case "/auth/callback":
			nonceHandlers.CallbackHandler().ServeHTTP(w, r)
		default:
			_, _ = w.Write([]byte("unexpected"))
		}
	}))
	defer app.Close()
	nonceHandlers, err = New(ctx, Config{IssuerURL: provider.URL(), ClientID: "goja-app", RedirectURL: app.URL + "/auth/callback", SessionManager: manager, UserNormalizer: passthroughNormalizer()})
	if err != nil {
		t.Fatalf("nonce handlers: %v", err)
	}
	client := clientWithJar(t)
	resp, err := client.Get(app.URL + "/auth/login")
	if err != nil {
		t.Fatalf("nonce flow: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("nonce mismatch status=%d", resp.StatusCode)
	}
}

func TestCallbackProviderErrorRendersSafeRecoveryHTML(t *testing.T) {
	ctx := context.Background()
	provider := newFakeProvider(t)
	defer provider.Close()
	manager, err := sessionauth.New(sessionauth.Config{Store: sessionauth.NewMemoryStore(), AllowInsecureHTTP: true})
	if err != nil {
		t.Fatalf("session manager: %v", err)
	}
	handlers, err := New(ctx, Config{
		IssuerURL:      provider.URL(),
		ClientID:       "goja-app",
		RedirectURL:    "http://app.test/auth/callback",
		SessionManager: manager,
		UserNormalizer: passthroughNormalizer(),
		CallbackErrorPage: CallbackErrorPage{
			StylesheetPath: "/static/styles.css",
			RetryPath:      "/auth/login",
			HomePath:       "/",
		},
	})
	if err != nil {
		t.Fatalf("handlers: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/callback?error=access_denied&error_description=%3Cscript%3Ealert(1)%3C%2Fscript%3E", nil)
	handlers.CallbackHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("provider error status=%d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content type=%q", got)
	}
	if got := rec.Header().Get("Content-Security-Policy"); got != "default-src 'none'; style-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'none'" {
		t.Fatalf("CSP=%q", got)
	}
	body := rec.Body.String()
	for _, want := range []string{"Sign-in was canceled", "/static/styles.css", "Try signing in again", "Return to the application"} {
		if !strings.Contains(body, want) {
			t.Fatalf("recovery page missing %q: %s", want, body)
		}
	}
	for _, forbidden := range []string{"access_denied", "error_description", "<script>", "alert(1)"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("recovery page reflected %q: %s", forbidden, body)
		}
	}
}

func TestCallbackErrorPageRejectsExternalPaths(t *testing.T) {
	for _, page := range []CallbackErrorPage{
		{StylesheetPath: "https://evil.example/style.css"},
		{RetryPath: "//evil.example"},
		{HomePath: "/\\evil.example"},
	} {
		if _, err := normalizeCallbackErrorPage(page); err == nil {
			t.Fatalf("expected rejected callback error page: %#v", page)
		}
	}
}

func TestCallbackRejectsExpiredTokenAndWrongAudience(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		mutate   func(*fakeProvider)
		wantCode int
	}{
		{name: "expired", mutate: func(p *fakeProvider) { p.expOffset = -time.Hour }, wantCode: http.StatusUnauthorized},
		{name: "wrong audience", mutate: func(p *fakeProvider) { p.audience = "other-app" }, wantCode: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := newFakeProvider(t)
			defer provider.Close()
			tc.mutate(provider)
			manager, err := sessionauth.New(sessionauth.Config{Store: sessionauth.NewMemoryStore(), AllowInsecureHTTP: true})
			if err != nil {
				t.Fatalf("session manager: %v", err)
			}
			var handlers *Handlers
			app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/auth/login":
					handlers.LoginHandler().ServeHTTP(w, r)
				case "/auth/callback":
					handlers.CallbackHandler().ServeHTTP(w, r)
				default:
					_, _ = w.Write([]byte("unexpected"))
				}
			}))
			defer app.Close()
			handlers, err = New(ctx, Config{IssuerURL: provider.URL(), ClientID: "goja-app", RedirectURL: app.URL + "/auth/callback", SessionManager: manager, UserNormalizer: passthroughNormalizer()})
			if err != nil {
				t.Fatalf("handlers: %v", err)
			}
			resp, err := clientWithJar(t).Get(app.URL + "/auth/login")
			if err != nil {
				t.Fatalf("flow: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != tc.wantCode {
				t.Fatalf("status=%d want=%d", resp.StatusCode, tc.wantCode)
			}
		})
	}
}

func TestCallbackRejectsNormalizerFailureAndLogoutClearsCookie(t *testing.T) {
	ctx := context.Background()
	provider := newFakeProvider(t)
	defer provider.Close()
	manager, err := sessionauth.New(sessionauth.Config{Store: sessionauth.NewMemoryStore(), AllowInsecureHTTP: true})
	if err != nil {
		t.Fatalf("session manager: %v", err)
	}
	var handlers *Handlers
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			handlers.LoginHandler().ServeHTTP(w, r)
		case "/auth/callback":
			handlers.CallbackHandler().ServeHTTP(w, r)
		case "/auth/logout":
			handlers.LogoutHandler().ServeHTTP(w, r)
		default:
			_, _ = w.Write([]byte("ok"))
		}
	}))
	defer app.Close()
	handlers, err = New(ctx, Config{
		IssuerURL:      provider.URL(),
		ClientID:       "goja-app",
		RedirectURL:    app.URL + "/auth/callback",
		SessionManager: manager,
		UserNormalizer: UserNormalizerFunc(func(context.Context, OIDCClaims) (UserSession, error) { return UserSession{}, fmt.Errorf("no user") }),
	})
	if err != nil {
		t.Fatalf("handlers: %v", err)
	}
	client := clientWithJar(t)
	resp, err := client.Get(app.URL + "/auth/login")
	if err != nil {
		t.Fatalf("normalizer flow: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("normalizer failure status=%d", resp.StatusCode)
	}

	session, err := manager.NewSession(ctx, "u1")
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	client.Jar.SetCookies(mustURL(t, app.URL), []*http.Cookie{{Name: sessionauth.InsecureCookieName, Value: session.ID, Path: "/"}})
	req, err := http.NewRequest(http.MethodPost, app.URL+"/auth/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("logout without csrf status=%d", resp.StatusCode)
	}
	if _, err = manager.Authenticate(ctx, requestWithCookies(client.Jar.Cookies(mustURL(t, app.URL))), nil, gojahttp.SecuritySpec{}); err != nil {
		t.Fatalf("csrf-rejected logout revoked session: %v", err)
	}

	getReq, err := http.NewRequest(http.MethodGet, app.URL+"/auth/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	getResp, err := client.Do(getReq)
	if err != nil {
		t.Fatalf("GET logout: %v", err)
	}
	_ = getResp.Body.Close()
	if getResp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET logout status=%d", getResp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodPost, app.URL+"/auth/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(sessionauth.CSRFHeaderName, session.CSRFToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("csrf-authenticated logout: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("csrf-authenticated logout status=%d", resp.StatusCode)
	}
	_, err = manager.Authenticate(ctx, requestWithCookies(client.Jar.Cookies(mustURL(t, app.URL))), nil, gojahttp.SecuritySpec{})
	if err == nil {
		t.Fatalf("expected session revoked after logout")
	}
}

func TestLogoutRevocationFailureDoesNotReportSuccessOrClearCookie(t *testing.T) {
	ctx := context.Background()
	baseStore := sessionauth.NewMemoryStore()
	manager, err := sessionauth.New(sessionauth.Config{
		Store:             revokeFailStore{Store: baseStore},
		AllowInsecureHTTP: true,
	})
	if err != nil {
		t.Fatalf("session manager: %v", err)
	}
	session, err := manager.NewSession(ctx, "u1")
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://app.example.test/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionauth.InsecureCookieName, Value: session.ID, Path: "/"})
	req.Header.Set(sessionauth.CSRFHeaderName, session.CSRFToken)
	recorder := httptest.NewRecorder()
	(&Handlers{sessionManager: manager}).LogoutHandler().ServeHTTP(recorder, req)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("revocation failure status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("revocation failure cleared browser cookie: %s", recorder.Header().Get("Set-Cookie"))
	}
	if _, err := manager.Authenticate(ctx, req, nil, gojahttp.SecuritySpec{}); err != nil {
		t.Fatalf("revocation failure changed stored session: %v", err)
	}
}

type revokeFailStore struct{ sessionauth.Store }

func (revokeFailStore) Revoke(context.Context, string) error {
	return fmt.Errorf("injected revoke failure")
}

func TestLogoutReportsSessionStoreFailure(t *testing.T) {
	provider := newFakeProvider(t)
	defer provider.Close()
	storeError := errors.New("database unavailable")
	manager, err := sessionauth.New(sessionauth.Config{Store: revokeErrorStore{Store: sessionauth.NewMemoryStore(), err: storeError}, AllowInsecureHTTP: true})
	if err != nil {
		t.Fatalf("session manager: %v", err)
	}
	handlers, err := New(context.Background(), Config{
		IssuerURL:      provider.URL(),
		ClientID:       "goja-app",
		RedirectURL:    "http://app.example.test/auth/callback",
		SessionManager: manager,
		UserNormalizer: passthroughNormalizer(),
	})
	if err != nil {
		t.Fatalf("handlers: %v", err)
	}
	session, err := manager.NewSession(context.Background(), "u1")
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://app.example.test/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionauth.InsecureCookieName, Value: session.ID})
	req.Header.Set(sessionauth.CSRFHeaderName, session.CSRFToken)
	res := httptest.NewRecorder()
	handlers.LogoutHandler().ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("logout status=%d, want %d", res.Code, http.StatusInternalServerError)
	}
}

type revokeErrorStore struct {
	sessionauth.Store
	err error
}

func (s revokeErrorStore) Revoke(context.Context, string) error { return s.err }

func passthroughNormalizer() UserNormalizer {
	return UserNormalizerFunc(func(_ context.Context, claims OIDCClaims) (UserSession, error) {
		return UserSession{UserID: claims.Subject, Email: claims.Email, EmailVerified: claims.EmailVerified}, nil
	})
}

type fakeProvider struct {
	server                *httptest.Server
	issuer                string
	key                   *rsa.PrivateKey
	mu                    sync.Mutex
	codes                 map[string]string
	wrongNonce            bool
	audience              string
	expOffset             time.Duration
	registrationRequested bool
}

func newFakeProvider(t *testing.T) *fakeProvider {
	t.Helper()
	provider := newInProcessFakeProvider(t, "")
	provider.server = httptest.NewServer(provider.Handler())
	provider.issuer = provider.server.URL
	return provider
}

func newInProcessFakeProvider(t *testing.T, issuer string) *fakeProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return &fakeProvider{issuer: issuer, key: key, codes: map[string]string{}, audience: "goja-app", expOffset: time.Hour}
}

func (p *fakeProvider) Handler() http.Handler {
	mux := http.NewServeMux()
	issuer, err := url.Parse(p.URL())
	if err != nil {
		panic(err)
	}
	prefix := strings.TrimSuffix(issuer.Path, "/")
	mux.HandleFunc(prefix+"/.well-known/openid-configuration", p.discovery)
	mux.HandleFunc(prefix+"/keys", p.keys)
	mux.HandleFunc(prefix+"/auth", p.auth)
	mux.HandleFunc(prefix+"/token", p.token)
	return mux
}

func (p *fakeProvider) URL() string { return p.issuer }
func (p *fakeProvider) Close() {
	if p.server != nil {
		p.server.Close()
	}
}

func (p *fakeProvider) discovery(w http.ResponseWriter, _ *http.Request) {
	writeJSON(tResponseWriter{w}, map[string]any{"issuer": p.URL(), "authorization_endpoint": p.URL() + "/auth", "token_endpoint": p.URL() + "/token", "jwks_uri": p.URL() + "/keys"})
}

func (p *fakeProvider) keys(w http.ResponseWriter, _ *http.Request) {
	pub := p.key.PublicKey
	writeJSON(tResponseWriter{w}, map[string]any{"keys": []map[string]any{{"kty": "RSA", "kid": "test-key", "use": "sig", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())}}})
}

func (p *fakeProvider) auth(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	nonce := r.URL.Query().Get("nonce")
	redirectURI := r.URL.Query().Get("redirect_uri")
	code := "code-" + state
	p.mu.Lock()
	p.codes[code] = nonce
	if r.URL.Query().Get("tinyidp_signup") == "1" {
		p.registrationRequested = true
	}
	p.mu.Unlock()
	callback, _ := url.Parse(redirectURI)
	q := callback.Query()
	q.Set("state", state)
	q.Set("code", code)
	callback.RawQuery = q.Encode()
	http.Redirect(w, r, callback.String(), http.StatusFound)
}

func (p *fakeProvider) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	code := r.Form.Get("code")
	p.mu.Lock()
	nonce, ok := p.codes[code]
	delete(p.codes, code)
	wrongNonce := p.wrongNonce
	p.mu.Unlock()
	if !ok {
		http.Error(w, "bad code", http.StatusUnauthorized)
		return
	}
	if wrongNonce {
		nonce = "wrong-" + nonce
	}
	idToken, err := p.signIDToken(nonce)
	if err != nil {
		http.Error(w, "sign token", http.StatusInternalServerError)
		return
	}
	writeJSON(tResponseWriter{w}, map[string]any{"access_token": "access", "id_token": idToken, "token_type": "Bearer", "expires_in": 3600})
}

func (p *fakeProvider) signIDToken(nonce string) (string, error) {
	now := time.Now().Unix()
	payload := map[string]any{"iss": p.URL(), "sub": "keycloak-sub-1", "aud": p.audience, "exp": time.Now().Add(p.expOffset).Unix(), "iat": now, "nonce": nonce, "email": "demo@example.test", "email_verified": true, "preferred_username": "demo", "groups": []string{"dev"}}
	header := map[string]any{"alg": "RS256", "kid": "test-key", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)
	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(payloadJSON)
	digest := sha256.Sum256([]byte(unsigned))
	sig, err := rsa.SignPKCS1v15(rand.Reader, p.key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

type tResponseWriter struct{ http.ResponseWriter }

func writeJSON(w tResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func clientWithJar(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func requestWithCookies(cookies []*http.Cookie) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	for _, cookie := range cookies {
		if strings.TrimSpace(cookie.Value) != "" {
			req.AddCookie(cookie)
		}
	}
	return req
}
