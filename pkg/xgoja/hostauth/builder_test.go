package hostauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/keycloakauth"
	keycloakauthsql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/keycloakauth/sqlstore"
	programauthsql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth/sqlstore"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

func TestBuildSessionManagerMapsResolvedConfig(t *testing.T) {
	now := time.Date(2026, 6, 14, 13, 0, 0, 0, time.UTC)
	manager, err := BuildSessionManager(ResolvedSessionConfig{
		Cookie: ResolvedCookieConfig{
			AllowInsecureHTTP: true,
			Name:              "test_session",
			SameSite:          http.SameSiteStrictMode,
			Path:              "/app",
		},
		IdleTimeout:     10 * time.Minute,
		AbsoluteTimeout: time.Hour,
	}, sessionauth.NewMemoryStore(), nil, func() time.Time { return now })
	if err != nil {
		t.Fatalf("BuildSessionManager: %v", err)
	}

	session, err := manager.NewSession(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if got := session.IdleExpiresAt.Sub(now); got != 10*time.Minute {
		t.Fatalf("idle timeout = %s", got)
	}
	if got := session.AbsoluteExpiresAt.Sub(now); got != time.Hour {
		t.Fatalf("absolute timeout = %s", got)
	}

	recorder := httptest.NewRecorder()
	manager.SetCookie(recorder, session.ID)
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %#v", cookies)
	}
	cookie := cookies[0]
	if cookie.Name != "test_session" || cookie.Path != "/app" || cookie.SameSite != http.SameSiteStrictMode || cookie.Secure {
		t.Fatalf("cookie = %#v", cookie)
	}
}

func TestBuildAuthOptionsWiresSessionAuditResourcesAndAuthorizer(t *testing.T) {
	stores, err := BuildStores(context.Background(), mustResolveStores(t, Config{}))
	if err != nil {
		t.Fatalf("BuildStores: %v", err)
	}
	manager, err := BuildSessionManager(ResolvedSessionConfig{}, stores.Session, nil, nil)
	if err != nil {
		t.Fatalf("BuildSessionManager: %v", err)
	}
	limiter := gojahttp.NewMemoryRateLimiter()
	options := BuildAuthOptions(manager, stores, nil, limiter, nil, nil)
	if options.Authenticator == nil || options.CSRF == nil || options.Resources == nil || options.Authorizer == nil || options.RateLimiter == nil {
		t.Fatalf("auth options missing fields: %#v", options)
	}
	if options.Audit != nil {
		t.Fatalf("audit should be nil when no sink is provided")
	}
}

func TestServiceFactoryModeNoneBuildsNoAuthOptions(t *testing.T) {
	services, err := NewServiceFactory(BuilderOptions{Config: Config{Mode: ModeNone}}).BuildHostAuthServices(context.Background(), nil)
	if err != nil {
		t.Fatalf("BuildHostAuthServices: %v", err)
	}
	if services.Config.Mode != ModeNone {
		t.Fatalf("mode = %q", services.Config.Mode)
	}
	if services.AuthOptions != (gojahttp.AuthOptions{}) || services.SessionManager != nil || len(services.Closers) != 0 {
		t.Fatalf("services = %#v", services)
	}
}

func TestServiceFactoryDevBuildsUsableAuthServices(t *testing.T) {
	now := time.Date(2026, 6, 14, 14, 0, 0, 0, time.UTC)
	services, err := NewServiceFactory(BuilderOptions{
		Config: Config{
			Mode:    ModeDev,
			Session: SessionConfig{Cookie: CookieConfig{AllowInsecureHTTP: true}},
		},
		Now: func() time.Time { return now },
	}).BuildHostAuthServices(context.Background(), nil)
	if err != nil {
		t.Fatalf("BuildHostAuthServices: %v", err)
	}
	defer func() {
		if err := services.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()
	if services.AuthOptions.Authenticator == nil || services.AuthOptions.CSRF == nil || services.AuthOptions.Audit == nil || services.AuthOptions.Resources == nil || services.AuthOptions.Authorizer == nil || services.AuthOptions.RateLimiter == nil {
		t.Fatalf("auth options missing fields: %#v", services.AuthOptions)
	}
	if services.RateLimiter == nil || services.SessionManager == nil || services.SessionStore == nil || services.AuditStore == nil || services.Capability == nil || services.AgentStore == nil || services.APITokenStore == nil || services.AccessTokenStore == nil || services.RefreshTokenStore == nil || services.DeviceStore == nil {
		t.Fatalf("services missing stores/managers: %#v", services)
	}

	session, err := services.SessionManager.NewSession(context.Background(), "user-1", sessionauth.WithEmail("demo@example.test", true))
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	recorder := httptest.NewRecorder()
	services.SessionManager.SetCookie(recorder, session.ID)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range recorder.Result().Cookies() {
		request.AddCookie(cookie)
	}
	actor, err := services.AuthOptions.Authenticator.Authenticate(context.Background(), request, nil, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser, Required: true})
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if actor.ID != "user-1" || actor.Claims["email"] != "demo@example.test" {
		t.Fatalf("actor = %#v", actor)
	}
}

func TestServiceFactoryOIDCBuildsNativeHandlers(t *testing.T) {
	issuer := newOIDCDiscoveryServer(t)
	services, err := NewServiceFactory(BuilderOptions{Config: Config{
		Mode:    ModeOIDC,
		Session: SessionConfig{Cookie: CookieConfig{AllowInsecureHTTP: true}},
		OIDC: OIDCConfig{
			IssuerURL:      issuer.URL,
			ClientID:       "goja-app",
			PublicBaseURL:  "http://localhost:8787",
			AfterLoginURL:  "/after",
			AfterLogoutURL: "/bye",
		},
	}}).BuildHostAuthServices(context.Background(), nil)
	if err != nil {
		t.Fatalf("BuildHostAuthServices: %v", err)
	}
	defer func() { _ = services.Close(context.Background()) }()
	if services.Config.Mode != ModeOIDC || services.Config.OIDC.RedirectURL != "http://localhost:8787/auth/callback" {
		t.Fatalf("config = %#v", services.Config)
	}
	if services.SessionManager == nil || services.AuthOptions.Authenticator == nil || services.AuthOptions.RateLimiter == nil {
		t.Fatalf("missing session/auth options: %#v", services)
	}
	got := map[string]bool{}
	for _, route := range services.NativeHandlers {
		got[route.Method+" "+route.Path] = route.Handler != nil
	}
	for _, want := range []string{"GET /auth/login", "GET /auth/callback", "POST /auth/logout", "GET /auth/logout", "GET /auth/session"} {
		if !got[want] {
			t.Fatalf("native handlers missing %s: %#v", want, services.NativeHandlers)
		}
	}
	for _, removed := range []string{"GET /auth/audit", "POST /orgs/o1/invites", "POST /org-invites/accept"} {
		if got[removed] {
			t.Fatalf("native demo handler %s should be owned by application code, got %#v", removed, services.NativeHandlers)
		}
	}
}

func TestServiceFactoryOIDCUsesConfiguredSQLTransactionStore(t *testing.T) {
	issuer := newOIDCDiscoveryServer(t)
	applySchema := true
	services, err := NewServiceFactory(BuilderOptions{Config: Config{
		Mode:    ModeOIDC,
		Session: SessionConfig{Cookie: CookieConfig{AllowInsecureHTTP: true}},
		Stores: StoresConfig{Default: StoreConfig{
			Driver:      "sqlite",
			DSN:         "file:hostauth-oidc-transaction?mode=memory&cache=shared",
			ApplySchema: &applySchema,
		}},
		OIDC: OIDCConfig{IssuerURL: issuer.URL, ClientID: "goja-app", PublicBaseURL: "http://localhost:8787"},
	}}).BuildHostAuthServices(context.Background(), nil)
	if err != nil {
		t.Fatalf("BuildHostAuthServices: %v", err)
	}
	defer func() { _ = services.Close(context.Background()) }()
	if _, ok := services.OIDCTransactionStore.(*keycloakauthsql.Store); !ok {
		t.Fatalf("OIDCTransactionStore type = %T", services.OIDCTransactionStore)
	}
}

func TestDefaultOIDCUserNormalizerUpsertsUserWithoutGrantingMemberships(t *testing.T) {
	stores, err := BuildStores(context.Background(), mustResolveStores(t, Config{}))
	if err != nil {
		t.Fatalf("BuildStores: %v", err)
	}
	defer func() { _ = stores.Close(context.Background()) }()
	normalizer := DefaultOIDCUserNormalizer(stores)
	session, err := normalizer.NormalizeOIDCUser(context.Background(), fakeOIDCClaims("kc-sub-1", "demo@example.test"))
	if err != nil {
		t.Fatalf("NormalizeOIDCUser: %v", err)
	}
	if session.UserID == "" || session.Email != "demo@example.test" || !session.EmailVerified {
		t.Fatalf("session = %#v", session)
	}
	if len(session.TenantIDs) != 0 {
		t.Fatalf("generic normalizer must not grant memberships, got %#v", session.TenantIDs)
	}
	if session.Claims["keycloakSub"] != "kc-sub-1" || session.Claims["preferredUsername"] != "demo" {
		t.Fatalf("claims = %#v", session.Claims)
	}
}

func TestServiceFactoryUsesSQLProgramAuthStore(t *testing.T) {
	applySchema := true
	factory := NewServiceFactory(BuilderOptions{Config: Config{
		Mode: ModeDev,
		Stores: StoresConfig{
			Default:     StoreConfig{Driver: "memory"},
			ProgramAuth: StoreConfig{Driver: "sqlite", DSN: "file:hostauth-programauth-store?mode=memory&cache=shared", ApplySchema: &applySchema},
		},
	}})
	services, err := factory.BuildHostAuthServices(context.Background(), nil)
	if err != nil {
		t.Fatalf("BuildHostAuthServices: %v", err)
	}
	defer func() {
		if err := services.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()
	if _, ok := services.AgentStore.(*programauthsql.Store); !ok {
		t.Fatalf("agent store type = %T", services.AgentStore)
	}
	if _, ok := services.APITokenStore.(*programauthsql.Store); !ok {
		t.Fatalf("api token store type = %T", services.APITokenStore)
	}
	if _, ok := services.AccessTokenStore.(*programauthsql.Store); !ok {
		t.Fatalf("access token store type = %T", services.AccessTokenStore)
	}
	if _, ok := services.RefreshTokenStore.(*programauthsql.Store); !ok {
		t.Fatalf("refresh token store type = %T", services.RefreshTokenStore)
	}
	if _, ok := services.DeviceStore.(*programauthsql.Store); !ok {
		t.Fatalf("device store type = %T", services.DeviceStore)
	}
}

func TestServiceFactoryUsesDirectDSNAtBuildTime(t *testing.T) {
	factory := NewServiceFactory(BuilderOptions{Config: Config{
		Mode:   ModeDev,
		Stores: StoresConfig{Default: StoreConfig{Driver: "sqlite"}},
	}})
	_, err := factory.BuildHostAuthServices(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected missing dsn error")
	}

	applySchema := true
	factory = NewServiceFactory(BuilderOptions{Config: Config{
		Mode:   ModeDev,
		Stores: StoresConfig{Default: StoreConfig{Driver: "sqlite", DSN: "file:hostauth-service-factory-dsn?mode=memory&cache=shared", ApplySchema: &applySchema}},
	}})
	services, err := factory.BuildHostAuthServices(context.Background(), nil)
	if err != nil {
		t.Fatalf("BuildHostAuthServices: %v", err)
	}
	if len(services.Closers) != 1 {
		t.Fatalf("closers = %d, want shared sqlite closer", len(services.Closers))
	}
	if err := services.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func newOIDCDiscoveryServer(t *testing.T) *httptest.Server {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":                                server.URL,
				"authorization_endpoint":                server.URL + "/auth",
				"token_endpoint":                        server.URL + "/token",
				"jwks_uri":                              server.URL + "/jwks",
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/jwks":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"keys":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func fakeOIDCClaims(sub string, email string) keycloakauth.OIDCClaims {
	return keycloakauth.OIDCClaims{Subject: sub, Email: email, EmailVerified: true, PreferredUsername: strings.TrimSuffix(email, "@example.test")}
}
