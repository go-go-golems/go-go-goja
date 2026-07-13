package hostauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/oidcauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

// BuilderOptions configures a generated-host auth service factory.
type BuilderOptions struct {
	Config      Config
	ActorLoader sessionauth.ActorLoader
	Now         func() time.Time
	// OIDCHTTPClient is used for OIDC discovery, token exchange, and JWKS
	// retrieval. Custom hosts can route a same-process issuer through a
	// fail-closed transport without starting the public listener early.
	OIDCHTTPClient *http.Client
}

// Builder is the default hostauth ServiceFactory implementation.
type Builder struct {
	options BuilderOptions
}

var (
	_ ServiceFactory = (*Builder)(nil)

	errServiceFactoryNil = errors.New("hostauth service factory is nil")
)

// NewServiceFactory returns a lazy generated-host auth service factory. The
// factory resolves config and opens stores only when BuildHostAuthServices is
// called, so command providers can discover the factory during command
// construction without touching databases or env-dependent DSNs.
func NewServiceFactory(opts BuilderOptions) *Builder {
	return &Builder{options: opts}
}

// BuildHostAuthServices builds concrete auth services for one command/runtime
// execution. The vals argument is reserved for future Glazed-value overlays;
// this first implementation resolves from BuilderOptions.Config and env refs.
func (b *Builder) AuthConfigDefaults() Config {
	if b == nil {
		return Config{}
	}
	return b.options.Config
}

func (b *Builder) BuildHostAuthServices(ctx context.Context, vals *values.Values) (*Services, error) {
	if b == nil {
		return nil, errNilBuilder()
	}
	cfg, err := ConfigFromValues(vals, b.options.Config)
	if err != nil {
		return nil, err
	}
	resolved, err := ResolveConfig(cfg, ResolveOptions{})
	if err != nil {
		return nil, err
	}
	if resolved.Mode == ModeNone {
		return &Services{Config: resolved}, nil
	}

	stores, err := BuildStores(ctx, resolved.Stores)
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			_ = stores.Close(ctx)
		}
	}()

	sessionManager, err := BuildSessionManager(resolved.Session, stores.Session, b.options.ActorLoader, b.options.Now)
	if err != nil {
		return nil, err
	}
	auditSink := audit.Sink{Store: stores.Audit}
	authOptions := BuildAuthOptions(sessionManager, stores, auditSink)
	nativeHandlers, err := BuildNativeHandlers(ctx, resolved, sessionManager, stores, b.options.OIDCHTTPClient)
	if err != nil {
		return nil, err
	}
	services := &Services{
		Config:         resolved,
		AuthOptions:    authOptions,
		SessionManager: sessionManager,
		SessionStore:   stores.Session,
		AuditSink:      auditSink,
		AuditStore:     stores.Audit,
		AppAuth:        stores.AppAuth,
		Capability:     stores.Capability,
		NativeHandlers: nativeHandlers,
		Closers:        stores.Closers,
	}
	success = true
	return services, nil
}

// BuildNativeHandlers maps resolved auth config into Go-owned HTTP handlers
// mounted by xgoja serve before the JavaScript app host fallback.
func BuildNativeHandlers(ctx context.Context, cfg ResolvedConfig, sessionManager *sessionauth.Manager, stores *StoreBundle, oidcHTTPClient *http.Client) ([]NativeHandler, error) {
	if cfg.Mode != ModeOIDC {
		return nil, nil
	}
	if sessionManager == nil {
		return nil, configError("auth.session", errors.New("session manager is required for auth.mode=oidc"))
	}
	if stores == nil || stores.AppAuth.Users == nil {
		return nil, configError("auth.stores.appauth", errors.New("app auth user store is required for auth.mode=oidc"))
	}
	handlers, err := oidcauth.New(ctx, oidcauth.Config{
		IssuerURL:      cfg.OIDC.IssuerURL,
		ClientID:       cfg.OIDC.ClientID,
		ClientSecret:   cfg.OIDC.ClientSecret,
		RedirectURL:    cfg.OIDC.RedirectURL,
		Scopes:         cfg.OIDC.Scopes,
		AfterLoginURL:  cfg.OIDC.AfterLoginURL,
		AfterLogoutURL: cfg.OIDC.AfterLogoutURL,
		SessionManager: sessionManager,
		UserNormalizer: DefaultOIDCUserNormalizer(stores),
		HTTPClient:     oidcHTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return []NativeHandler{
		{Method: "GET", Path: "/auth/login", Handler: handlers.LoginHandler()},
		{Method: "GET", Path: "/auth/callback", Handler: handlers.CallbackHandler()},
		{Method: "POST", Path: "/auth/logout", Handler: handlers.LogoutHandler()},
		{Method: "GET", Path: "/auth/session", Handler: sessionInfoHandler(sessionManager)},
	}, nil
}

func sessionInfoHandler(sessionManager *sessionauth.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionManager.SessionFromRequest(r.Context(), r)
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		writeJSON(w, map[string]any{
			"userId":    session.UserID,
			"csrfToken": session.CSRFToken,
			"tenantIds": append([]string(nil), session.TenantIDs...),
		})
	})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, "encode json", http.StatusInternalServerError)
	}
}

// DefaultOIDCUserNormalizer upserts an app user by stable OIDC subject and
// projects existing app memberships into the application session. It does not
// grant roles or seed tenants; application seeding remains outside generic
// hostauth.
func DefaultOIDCUserNormalizer(stores *StoreBundle) oidcauth.UserNormalizer {
	return oidcauth.UserNormalizerFunc(func(ctx context.Context, claims oidcauth.OIDCClaims) (oidcauth.UserSession, error) {
		user, err := stores.AppAuth.Users.UpsertFromOIDC(ctx, claims.Issuer, claims.Subject, claims.Email, claims.EmailVerified)
		if err != nil {
			return oidcauth.UserSession{}, err
		}
		tenantIDs := []string(nil)
		if stores.AppAuth.Memberships != nil {
			memberships, err := stores.AppAuth.Memberships.MembershipsForUser(ctx, user.ID)
			if err != nil {
				return oidcauth.UserSession{}, err
			}
			seen := map[string]struct{}{}
			for _, membership := range memberships {
				if membership.RevokedAt != nil || membership.TenantID == "" {
					continue
				}
				if _, ok := seen[membership.TenantID]; ok {
					continue
				}
				seen[membership.TenantID] = struct{}{}
				tenantIDs = append(tenantIDs, membership.TenantID)
			}
		}
		return oidcauth.UserSession{
			UserID:        user.ID,
			Email:         user.Email,
			EmailVerified: user.EmailVerified,
			TenantIDs:     tenantIDs,
			Claims: map[string]any{
				"oidcIssuer":        claims.Issuer,
				"oidcSubject":       claims.Subject,
				"preferredUsername": claims.PreferredUsername,
				"name":              claims.Name,
				"groups":            append([]string(nil), claims.Groups...),
			},
		}, nil
	})
}

// BuildSessionManager maps resolved generated-host config into sessionauth.
func BuildSessionManager(cfg ResolvedSessionConfig, store sessionauth.Store, actorLoader sessionauth.ActorLoader, now func() time.Time) (*sessionauth.Manager, error) {
	return sessionauth.New(sessionauth.Config{
		Store:             store,
		ActorLoader:       actorLoader,
		CookieName:        cfg.Cookie.Name,
		Path:              cfg.Cookie.Path,
		SameSite:          cfg.Cookie.SameSite,
		IdleTimeout:       cfg.IdleTimeout,
		AbsoluteTimeout:   cfg.AbsoluteTimeout,
		AllowInsecureHTTP: cfg.Cookie.AllowInsecureHTTP,
		Now:               now,
	})
}

// BuildAuthOptions wires a session manager and built auth stores into
// gojahttp's host-owned auth interfaces.
func BuildAuthOptions(sessionManager *sessionauth.Manager, stores *StoreBundle, auditSink gojahttp.AuditSink) gojahttp.AuthOptions {
	var options gojahttp.AuthOptions
	if sessionManager != nil {
		options.Authenticator = sessionManager
		options.CSRF = sessionManager
	}
	if auditSink != nil {
		options.Audit = auditSink
	}
	if stores != nil {
		if stores.AppAuth.Resources != nil {
			options.Resources = appauth.Resolver{Store: stores.AppAuth.Resources}
		}
		if stores.AppAuth.Memberships != nil {
			options.Authorizer = appauth.Authorizer{Memberships: stores.AppAuth.Memberships}
		}
	}
	return options
}

func errNilBuilder() error { return &ConfigError{Path: "auth", Err: errServiceFactoryNil} }
