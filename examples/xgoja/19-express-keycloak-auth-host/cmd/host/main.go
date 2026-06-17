package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"

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
	root, err := newRootCommand()
	if err != nil {
		log.Fatal(err)
	}
	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}

func newRootCommand() (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "goja-keycloak-auth-host",
		Short: "Run the go-go-goja Keycloak auth host example",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
	}
	if err := logging.AddLoggingSectionToRootCommand(root, "goja-keycloak-auth-host"); err != nil {
		return nil, err
	}
	serveCommand := newServeCommand()
	cobraCommand, err := cli.BuildCobraCommand(serveCommand,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		return nil, err
	}
	root.AddCommand(cobraCommand)
	return root, nil
}

type serveCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*serveCommand)(nil)

func newServeCommand() *serveCommand {
	return &serveCommand{CommandDescription: cmds.NewCommandDescription("serve",
		cmds.WithShort("Serve the Keycloak-backed planned-auth example host"),
		cmds.WithLong(`Serve the Keycloak-backed planned-auth example host.

The host is intentionally still an example binary, but all operator-facing
configuration is exposed as Glazed flags with environment-backed defaults. Use
--public-base-url for the external HTTPS origin behind ingress; --redirect-url is
an advanced explicit callback override.`),
		cmds.WithFlags(
			fields.New("listen", fields.TypeString, fields.WithDefault(envOr("LISTEN_ADDR", ":8080")), fields.WithHelp("Listen address for the in-pod HTTP server")),
			fields.New("script", fields.TypeString, fields.WithDefault(envOr("SCRIPT_PATH", "/app/server.js")), fields.WithHelp("JavaScript route script to load")),
			fields.New("issuer", fields.TypeString, fields.WithDefault(os.Getenv("KEYCLOAK_ISSUER")), fields.WithHelp("OIDC issuer URL for the Keycloak realm")),
			fields.New("client-id", fields.TypeString, fields.WithDefault(envOr("KEYCLOAK_CLIENT_ID", "goja-auth-host-demo")), fields.WithHelp("OIDC client ID")),
			fields.New("client-secret", fields.TypeString, fields.WithDefault(os.Getenv("KEYCLOAK_CLIENT_SECRET")), fields.WithHelp("OIDC client secret for confidential clients")),
			fields.New("public-base-url", fields.TypeString, fields.WithDefault(os.Getenv("PUBLIC_BASE_URL")), fields.WithHelp("External browser-visible base URL, for example https://goja-auth.yolo.scapegoat.dev")),
			fields.New("redirect-url", fields.TypeString, fields.WithDefault(os.Getenv("KEYCLOAK_REDIRECT_URL")), fields.WithHelp("Explicit OIDC callback URL; defaults to <public-base-url>/auth/callback")),
			fields.New("after-login-url", fields.TypeString, fields.WithDefault(envOr("AFTER_LOGIN_URL", "/")), fields.WithHelp("Local path to redirect to after successful login")),
			fields.New("after-logout-url", fields.TypeString, fields.WithDefault(envOr("AFTER_LOGOUT_URL", "/")), fields.WithHelp("Local path to redirect to after logout")),
			fields.New("allow-insecure-http", fields.TypeBool, fields.WithDefault(envBool("ALLOW_INSECURE_HTTP", false)), fields.WithHelp("Use insecure localhost cookie and URL settings; must be false behind HTTPS ingress")),
			fields.New("session-db-dsn", fields.TypeString, fields.WithDefault(os.Getenv("SESSION_DB_DSN")), fields.WithHelp("Postgres DSN for server-side app sessions")),
			fields.New("audit-db-dsn", fields.TypeString, fields.WithDefault(os.Getenv("AUDIT_DB_DSN")), fields.WithHelp("Postgres DSN for audit records")),
			fields.New("app-db-dsn", fields.TypeString, fields.WithDefault(os.Getenv("APPAUTH_DB_DSN")), fields.WithHelp("Postgres DSN for appauth users, memberships, and resources")),
			fields.New("capability-db-dsn", fields.TypeString, fields.WithDefault(os.Getenv("CAPABILITY_DB_DSN")), fields.WithHelp("Postgres DSN for capability tokens")),
		),
	)}
}

func (c *serveCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := serveSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	redirectURL, err := resolveRedirectURL(settings)
	if err != nil {
		return err
	}
	return run(ctx, config{
		Listen:            settings.Listen,
		Script:            settings.Script,
		Issuer:            settings.Issuer,
		ClientID:          settings.ClientID,
		ClientSecret:      settings.ClientSecret,
		RedirectURL:       redirectURL,
		AfterLoginURL:     settings.AfterLoginURL,
		AfterLogoutURL:    settings.AfterLogoutURL,
		AllowInsecureHTTP: settings.AllowInsecureHTTP,
		SessionDBDSN:      settings.SessionDBDSN,
		AuditDBDSN:        settings.AuditDBDSN,
		AppDBDSN:          settings.AppDBDSN,
		CapabilityDBDSN:   settings.CapabilityDBDSN,
	})
}

type serveSettings struct {
	Listen            string `glazed:"listen"`
	Script            string `glazed:"script"`
	Issuer            string `glazed:"issuer"`
	ClientID          string `glazed:"client-id"`
	ClientSecret      string `glazed:"client-secret"`
	PublicBaseURL     string `glazed:"public-base-url"`
	RedirectURL       string `glazed:"redirect-url"`
	AfterLoginURL     string `glazed:"after-login-url"`
	AfterLogoutURL    string `glazed:"after-logout-url"`
	AllowInsecureHTTP bool   `glazed:"allow-insecure-http"`
	SessionDBDSN      string `glazed:"session-db-dsn"`
	AuditDBDSN        string `glazed:"audit-db-dsn"`
	AppDBDSN          string `glazed:"app-db-dsn"`
	CapabilityDBDSN   string `glazed:"capability-db-dsn"`
}

type config struct {
	Listen            string
	Script            string
	Issuer            string
	ClientID          string
	ClientSecret      string
	RedirectURL       string
	AfterLoginURL     string
	AfterLogoutURL    string
	AllowInsecureHTTP bool
	SessionDBDSN      string
	AuditDBDSN        string
	AppDBDSN          string
	CapabilityDBDSN   string
}

func resolveRedirectURL(settings serveSettings) (string, error) {
	if redirectURL := strings.TrimSpace(settings.RedirectURL); redirectURL != "" {
		return redirectURL, requireAllowedURLScheme(redirectURL, settings.AllowInsecureHTTP)
	}
	publicBase := strings.TrimRight(strings.TrimSpace(settings.PublicBaseURL), "/")
	if publicBase == "" {
		return "", errors.New("public-base-url or redirect-url is required")
	}
	if err := requireAllowedURLScheme(publicBase, settings.AllowInsecureHTTP); err != nil {
		return "", err
	}
	return publicBase + "/auth/callback", nil
}

func requireAllowedURLScheme(raw string, allowInsecureHTTP bool) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse URL %q: %w", raw, err)
	}
	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		if allowInsecureHTTP && isLocalhost(parsed.Hostname()) {
			return nil
		}
		return fmt.Errorf("%s must use https unless allow-insecure-http is true for localhost", raw)
	default:
		return fmt.Errorf("%s must use http or https", raw)
	}
}

func isLocalhost(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func envBool(name string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return fallback
	}
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
	sessions, cleanupSessions, err := newSessionManager(cfg.SessionDBDSN, cfg.AllowInsecureHTTP)
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
		RedirectURL:    cfg.RedirectURL,
		AfterLoginURL:  cfg.AfterLoginURL,
		AfterLogoutURL: cfg.AfterLogoutURL,
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
	factory, err := engine.NewRuntimeFactoryBuilder().
		UseModuleMiddleware(engine.MiddlewareOnly("timer")).
		WithModules(express.NewRegistrar(host)).
		Build()
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
	log.Printf("serving Keycloak auth example on %s", cfg.Listen)
	log.Printf("Keycloak issuer: %s", cfg.Issuer)
	log.Printf("OIDC redirect URL: %s", cfg.RedirectURL)
	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return serveWithShutdown(ctx, server)
}

func serveWithShutdown(ctx context.Context, server *http.Server) error {
	serveCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-serveCtx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return <-errCh
}

func newSessionManager(sessionDBDSN string, allowInsecureHTTP bool) (*sessionauth.Manager, func(), error) {
	if sessionDBDSN == "" {
		sessions, err := sessionauth.New(sessionauth.Config{Store: sessionauth.NewMemoryStore(), AllowInsecureHTTP: allowInsecureHTTP})
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
	sessions, err := sessionauth.New(sessionauth.Config{Store: store, AllowInsecureHTTP: allowInsecureHTTP})
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
