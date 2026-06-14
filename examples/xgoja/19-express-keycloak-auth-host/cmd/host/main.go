package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dop251/goja"
	_ "github.com/lib/pq"

	"github.com/go-go-golems/go-go-goja/modules/express"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	appauthSQLStore "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth/sqlstore"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
	auditSQLStore "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit/sqlstore"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
	capabilitySQLStore "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability/sqlstore"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/keycloakauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
	sessionSQLStore "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth/sqlstore"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8790", "listen address")
	script := flag.String("script", "examples/xgoja/19-express-keycloak-auth-host/scripts/server.js", "JavaScript route script")
	issuer := flag.String("issuer", envOr("KEYCLOAK_ISSUER", "http://127.0.0.1:18080/realms/goja-demo"), "OIDC issuer URL")
	clientID := flag.String("client-id", envOr("KEYCLOAK_CLIENT_ID", "goja-app"), "OIDC client ID")
	clientSecret := flag.String("client-secret", os.Getenv("KEYCLOAK_CLIENT_SECRET"), "OIDC client secret, if configured")
	sessionDBDSN := flag.String("session-db-dsn", os.Getenv("SESSION_DB_DSN"), "Postgres DSN for persistent app sessions; empty uses in-memory sessions")
	auditDBDSN := flag.String("audit-db-dsn", os.Getenv("AUDIT_DB_DSN"), "Postgres DSN for persistent audit records; empty logs audit records")
	appDBDSN := flag.String("app-db-dsn", os.Getenv("APPAUTH_DB_DSN"), "Postgres DSN for persistent appauth users/resources; empty uses in-memory appauth")
	capabilityDBDSN := flag.String("capability-db-dsn", os.Getenv("CAPABILITY_DB_DSN"), "Postgres DSN for persistent capability tokens; empty uses in-memory capabilities")
	flag.Parse()
	if err := run(context.Background(), config{Listen: *listen, Script: *script, Issuer: *issuer, ClientID: *clientID, ClientSecret: *clientSecret, SessionDBDSN: *sessionDBDSN, AuditDBDSN: *auditDBDSN, AppDBDSN: *appDBDSN, CapabilityDBDSN: *capabilityDBDSN}); err != nil {
		log.Fatal(err)
	}
}

type config struct {
	Listen          string
	Script          string
	Issuer          string
	ClientID        string
	ClientSecret    string
	SessionDBDSN    string
	AuditDBDSN      string
	AppDBDSN        string
	CapabilityDBDSN string
}

func run(ctx context.Context, cfg config) error {
	appStores, err := newAppStore(ctx, cfg.AppDBDSN)
	if err != nil {
		return err
	}
	defer appStores.cleanup()
	capabilityService, cleanupCapabilities, err := newCapabilityService(ctx, cfg.CapabilityDBDSN)
	if err != nil {
		return err
	}
	defer cleanupCapabilities()
	sessions, cleanupSessions, err := newSessionManager(cfg.SessionDBDSN)
	if err != nil {
		return err
	}
	defer cleanupSessions()
	auditSink, cleanupAudit, err := newAuditSink(cfg.AuditDBDSN)
	if err != nil {
		return err
	}
	defer cleanupAudit()
	keycloakHandlers, err := keycloakauth.New(ctx, keycloakauth.Config{
		IssuerURL:      cfg.Issuer,
		ClientID:       cfg.ClientID,
		ClientSecret:   cfg.ClientSecret,
		RedirectURL:    "http://" + cfg.Listen + "/auth/callback",
		AfterLoginURL:  "/",
		AfterLogoutURL: "/",
		SessionManager: sessions,
		UserNormalizer: keycloakauth.UserNormalizerFunc(func(ctx context.Context, claims keycloakauth.OIDCClaims) (keycloakauth.UserSession, error) {
			user, err := appStores.store.UpsertFromOIDC(ctx, claims.Subject, claims.Email, claims.EmailVerified)
			if err != nil {
				return keycloakauth.UserSession{}, err
			}
			if err := appStores.addMembership(ctx, appauth.Membership{UserID: user.ID, TenantID: "o1", Role: "admin"}); err != nil {
				return keycloakauth.UserSession{}, err
			}
			return keycloakauth.UserSession{UserID: user.ID, Email: user.Email, EmailVerified: user.EmailVerified, TenantIDs: []string{"o1"}, Claims: map[string]any{"keycloakSub": claims.Subject, "preferredUsername": claims.PreferredUsername}}, nil
		}),
	})
	if err != nil {
		return err
	}
	host := gojahttp.NewHost(gojahttp.HostOptions{
		Dev:             true,
		RejectRawRoutes: true,
		Auth: gojahttp.AuthOptions{
			Authenticator: sessions,
			CSRF:          sessions,
			Resources:     appauth.Resolver{Store: appStores.store},
			Authorizer:    appauth.Authorizer{Memberships: appStores.store},
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
	mux.Handle("POST /orgs/o1/invites", issueInviteHandler(sessions, appStores.store, capabilityService))
	mux.Handle("POST /org-invites/accept", acceptInviteHandler(capabilityService))
	mux.Handle("/", indexPage(host))
	log.Printf("serving Keycloak auth example on http://%s", cfg.Listen)
	log.Printf("Keycloak issuer: %s", cfg.Issuer)
	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return server.ListenAndServe()
}

func newSessionManager(sessionDBDSN string) (*sessionauth.Manager, func(), error) {
	if sessionDBDSN == "" {
		sessions, err := sessionauth.New(sessionauth.Config{Store: sessionauth.NewMemoryStore(), AllowInsecureHTTP: true})
		return sessions, func() {}, err
	}
	db, err := sql.Open("postgres", sessionDBDSN)
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() { _ = db.Close() }
	if err := db.Ping(); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	store, err := sessionSQLStore.New(sessionSQLStore.Config{DB: db, Dialect: sessionSQLStore.DialectPostgres})
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if err := store.ApplySchema(context.Background()); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	sessions, err := sessionauth.New(sessionauth.Config{Store: store, AllowInsecureHTTP: true})
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	log.Printf("using Postgres-backed app sessions")
	return sessions, cleanup, nil
}

func newAuditSink(auditDBDSN string) (gojahttp.AuditSink, func(), error) {
	if auditDBDSN == "" {
		return audit.LogSink{}, func() {}, nil
	}
	db, err := sql.Open("postgres", auditDBDSN)
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() { _ = db.Close() }
	if err := db.Ping(); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	store, err := auditSQLStore.New(auditSQLStore.Config{DB: db, Dialect: auditSQLStore.DialectPostgres})
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if err := store.ApplySchema(context.Background()); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	log.Printf("using Postgres-backed audit records")
	return audit.Sink{Store: store}, cleanup, nil
}

type appStore interface {
	appauth.UserStore
	appauth.MembershipStore
	appauth.ResourceStore
}

type appStoreBundle struct {
	store         appStore
	addMembership func(context.Context, appauth.Membership) error
	cleanup       func()
}

func newAppStore(ctx context.Context, appDBDSN string) (appStoreBundle, error) {
	if appDBDSN == "" {
		store := appauth.NewMemoryStore()
		seedMemoryAppStore(store)
		return appStoreBundle{store: store, addMembership: func(_ context.Context, membership appauth.Membership) error {
			store.AddMembership(membership)
			return nil
		}, cleanup: func() {}}, nil
	}
	db, err := sql.Open("postgres", appDBDSN)
	if err != nil {
		return appStoreBundle{}, err
	}
	cleanup := func() { _ = db.Close() }
	if err := db.Ping(); err != nil {
		cleanup()
		return appStoreBundle{}, err
	}
	store, err := appauthSQLStore.New(appauthSQLStore.Config{DB: db, Dialect: appauthSQLStore.DialectPostgres})
	if err != nil {
		cleanup()
		return appStoreBundle{}, err
	}
	if err := store.ApplySchema(ctx); err != nil {
		cleanup()
		return appStoreBundle{}, err
	}
	if err := seedSQLAppStore(ctx, store); err != nil {
		cleanup()
		return appStoreBundle{}, err
	}
	log.Printf("using Postgres-backed appauth users/resources")
	return appStoreBundle{store: store, addMembership: store.AddMembership, cleanup: cleanup}, nil
}

func seedMemoryAppStore(store *appauth.MemoryStore) {
	store.AddResource(appauth.Resource{Type: "project", ID: "p1", TenantID: "o1", Claims: map[string]any{"name": "Docker Compose Project"}})
	store.AddResource(appauth.Resource{Type: "org", ID: "o1"})
}

func seedSQLAppStore(ctx context.Context, store *appauthSQLStore.Store) error {
	if err := store.AddTenant(ctx, appauth.Tenant{ID: "o1", Slug: "demo", Name: "Demo Organization"}); err != nil {
		return err
	}
	if err := store.AddResource(ctx, appauth.Resource{Type: "project", ID: "p1", TenantID: "o1", Claims: map[string]any{"name": "Docker Compose Project"}}); err != nil {
		return err
	}
	if err := store.AddResource(ctx, appauth.Resource{Type: "org", ID: "o1"}); err != nil {
		return err
	}
	return nil
}

func newCapabilityService(ctx context.Context, capabilityDBDSN string) (capability.Service, func(), error) {
	if capabilityDBDSN == "" {
		return capability.Service{Store: capability.NewMemoryStore()}, func() {}, nil
	}
	db, err := sql.Open("postgres", capabilityDBDSN)
	if err != nil {
		return capability.Service{}, func() {}, err
	}
	cleanup := func() { _ = db.Close() }
	if err := db.Ping(); err != nil {
		cleanup()
		return capability.Service{}, func() {}, err
	}
	store, err := capabilitySQLStore.New(capabilitySQLStore.Config{DB: db, Dialect: capabilitySQLStore.DialectPostgres})
	if err != nil {
		cleanup()
		return capability.Service{}, func() {}, err
	}
	if err := store.ApplySchema(ctx); err != nil {
		cleanup()
		return capability.Service{}, func() {}, err
	}
	log.Printf("using Postgres-backed capability tokens")
	return capability.Service{Store: store}, cleanup, nil
}

func issueInviteHandler(sessions *sessionauth.Manager, memberships appauth.MembershipStore, capabilityService capability.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := sessions.SessionFromRequest(r.Context(), r)
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		actor := &gojahttp.Actor{ID: session.UserID, Kind: "user"}
		if err := sessions.VerifyCSRF(r.Context(), gojahttp.CSRFRequest{HTTPRequest: r, Actor: actor}); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		decision, err := (appauth.Authorizer{Memberships: memberships}).Authorize(r.Context(), gojahttp.AuthorizationRequest{Actor: actor, Action: appauth.ActionOrgInvite, Resource: &gojahttp.ResourceRef{Type: "org", ID: "o1"}})
		if err != nil {
			http.Error(w, "authorization failed", http.StatusInternalServerError)
			return
		}
		if !decision.Allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var req struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Email == "" {
			http.Error(w, "email is required", http.StatusBadRequest)
			return
		}
		if req.Role == "" {
			req.Role = "viewer"
		}
		issued, err := capabilityService.IssueOrgInvite(r.Context(), capability.OrgInviteSpec{OrgID: "o1", Email: req.Email, Role: req.Role, TTL: 15 * time.Minute, CreatedBy: session.UserID})
		if err != nil {
			http.Error(w, "issue invite", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"capabilityId": issued.Capability.ID, "token": issued.Token, "email": req.Email, "role": req.Role})
	})
}

func acceptInviteHandler(capabilityService capability.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		accepted, err := capabilityService.AcceptOrgInvite(r.Context(), req.Token)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, capability.ErrUsed) || errors.Is(err, capability.ErrRevoked) || errors.Is(err, capability.ErrExpired) {
				status = http.StatusConflict
			}
			http.Error(w, err.Error(), status)
			return
		}
		writeJSON(w, accepted)
	})
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
