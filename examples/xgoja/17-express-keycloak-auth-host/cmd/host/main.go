package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules/express"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/keycloakauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8790", "listen address")
	script := flag.String("script", "examples/xgoja/17-express-keycloak-auth-host/scripts/server.js", "JavaScript route script")
	issuer := flag.String("issuer", envOr("KEYCLOAK_ISSUER", "http://127.0.0.1:18080/realms/goja-demo"), "OIDC issuer URL")
	clientID := flag.String("client-id", envOr("KEYCLOAK_CLIENT_ID", "goja-app"), "OIDC client ID")
	clientSecret := flag.String("client-secret", os.Getenv("KEYCLOAK_CLIENT_SECRET"), "OIDC client secret, if configured")
	flag.Parse()
	if err := run(context.Background(), config{Listen: *listen, Script: *script, Issuer: *issuer, ClientID: *clientID, ClientSecret: *clientSecret}); err != nil {
		log.Fatal(err)
	}
}

type config struct {
	Listen       string
	Script       string
	Issuer       string
	ClientID     string
	ClientSecret string
}

func run(ctx context.Context, cfg config) error {
	appStore := seedAppStore()
	sessions, err := sessionauth.New(sessionauth.Config{Store: sessionauth.NewMemoryStore(), AllowInsecureHTTP: true})
	if err != nil {
		return err
	}
	keycloakHandlers, err := keycloakauth.New(ctx, keycloakauth.Config{
		IssuerURL:      cfg.Issuer,
		ClientID:       cfg.ClientID,
		ClientSecret:   cfg.ClientSecret,
		RedirectURL:    "http://" + cfg.Listen + "/auth/callback",
		AfterLoginURL:  "/",
		AfterLogoutURL: "/",
		SessionManager: sessions,
		UserNormalizer: keycloakauth.UserNormalizerFunc(func(ctx context.Context, claims keycloakauth.OIDCClaims) (keycloakauth.UserSession, error) {
			user, err := appStore.UpsertFromOIDC(ctx, claims.Subject, claims.Email, claims.EmailVerified)
			if err != nil {
				return keycloakauth.UserSession{}, err
			}
			appStore.AddMembership(appauth.Membership{UserID: user.ID, TenantID: "o1", Role: "admin"})
			return keycloakauth.UserSession{UserID: user.ID, Email: user.Email, EmailVerified: user.EmailVerified, TenantIDs: []string{"o1"}, Claims: map[string]any{"keycloakSub": claims.Subject, "preferredUsername": claims.PreferredUsername}}, nil
		}),
	})
	if err != nil {
		return err
	}
	auditSink := audit.LogSink{}
	host := gojahttp.NewHost(gojahttp.HostOptions{
		Dev:             true,
		RejectRawRoutes: true,
		Auth: gojahttp.AuthOptions{
			Authenticator: sessions,
			CSRF:          sessions,
			Resources:     appauth.Resolver{Store: appStore},
			Authorizer:    appauth.Authorizer{Memberships: appStore},
			Audit:         auditSink,
		},
	})
	factory, err := engine.NewRuntimeFactoryBuilder().WithModules(express.NewRegistrar(host)).Build()
	if err != nil {
		return err
	}
	rt, err := factory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))
	if err != nil {
		return err
	}
	defer func() { _ = rt.Close(ctx) }()
	host.SetRuntime(rt.Owner)
	data, err := os.ReadFile(cfg.Script)
	if err != nil {
		return err
	}
	if _, err := rt.Owner.Call(ctx, "load-keycloak-auth-example", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(string(data))
		return nil, runErr
	}); err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("GET /auth/login", keycloakHandlers.LoginHandler())
	mux.Handle("GET /auth/callback", keycloakHandlers.CallbackHandler())
	mux.Handle("POST /auth/logout", keycloakHandlers.LogoutHandler())
	mux.Handle("GET /auth/session", sessionHandler(sessions))
	mux.Handle("/", indexPage(host))
	log.Printf("serving Keycloak auth example on http://%s", cfg.Listen)
	log.Printf("Keycloak issuer: %s", cfg.Issuer)
	return http.ListenAndServe(cfg.Listen, mux)
}

func seedAppStore() *appauth.MemoryStore {
	store := appauth.NewMemoryStore()
	store.AddResource(appauth.Resource{Type: "project", ID: "p1", TenantID: "o1", Claims: map[string]any{"name": "Docker Compose Project"}})
	store.AddResource(appauth.Resource{Type: "org", ID: "o1"})
	return store
}

func sessionHandler(sessions *sessionauth.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := sessions.SessionFromRequest(r.Context(), r)
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		writeJSON(w, map[string]any{"userId": session.UserID, "csrfToken": session.CSRFToken, "tenantIds": session.TenantIDs})
	})
}

func indexPage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<h1>go-go-goja Keycloak auth example</h1>
<ul>
  <li><a href="/auth/login">Login with Keycloak</a></li>
  <li><a href="/healthz">/healthz</a></li>
  <li><a href="/me">/me</a></li>
  <li><a href="/auth/session">/auth/session</a> returns the CSRF token after login</li>
</ul>`))
	})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write json: %v", err)
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
