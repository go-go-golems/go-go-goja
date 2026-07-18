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
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/keycloakauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

// BuilderOptions configures a generated-host auth service factory.
type BuilderOptions struct {
	Config         Config
	ActorLoader    sessionauth.ActorLoader
	Now            func() time.Time
	SecurityEvents gojahttp.SecurityEventObserver
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
	rateLimiter, err := buildRateLimiter(resolved.RateLimiter)
	if err != nil {
		return nil, err
	}
	agentService := programauth.AgentService{Store: stores.ProgramAuth.Agents, Now: b.options.Now}
	apiTokenService := programauth.APITokenService{Store: stores.ProgramAuth.APITokens, Agents: agentService, Now: b.options.Now}
	oauthTokenService := programauth.OAuthTokenService{AccessTokens: stores.ProgramAuth.AccessTokens, RefreshTokens: stores.ProgramAuth.RefreshTokens, PairStore: stores.ProgramAuth.OAuthTokenPairs, Agents: agentService, Now: b.options.Now}
	deviceService := programauth.DeviceService{Store: stores.ProgramAuth.Devices, Agents: agentService, OAuthTokens: oauthTokenService, Now: b.options.Now, VerificationURI: "/auth/device"}
	securityEvents := b.options.SecurityEvents
	if securityEvents == nil {
		securityEvents = &gojahttp.MemorySecurityMetrics{}
	}
	authOptions := BuildAuthOptions(sessionManager, stores, auditSink, rateLimiter, apiTokenService, oauthTokenService)
	authOptions.SecurityEvents = securityEvents
	nativeHandlers, err := BuildNativeHandlers(ctx, resolved, sessionManager, stores, deviceService, auditSink, securityEvents)
	if err != nil {
		return nil, err
	}
	services := &Services{
		Config:               resolved,
		AuthOptions:          authOptions,
		SessionManager:       sessionManager,
		SessionStore:         stores.Session,
		OIDCTransactionStore: stores.OIDCTransaction,
		AuditSink:            auditSink,
		AuditStore:           stores.Audit,
		RateLimiter:          rateLimiter,
		RequestIdentity:      gojahttp.TrustedProxyResolver{Mode: resolved.Proxy.Mode, TrustedPrefixes: resolved.Proxy.TrustedPrefixes},
		SecurityEvents:       securityEvents,
		AppAuth:              stores.AppAuth,
		Capability:           stores.Capability,
		AgentStore:           stores.ProgramAuth.Agents,
		APITokenStore:        stores.ProgramAuth.APITokens,
		AccessTokenStore:     stores.ProgramAuth.AccessTokens,
		RefreshTokenStore:    stores.ProgramAuth.RefreshTokens,
		DeviceStore:          stores.ProgramAuth.Devices,
		Agents:               agentService,
		APITokens:            apiTokenService,
		OAuthTokens:          oauthTokenService,
		Devices:              deviceService,
		NativeHandlers:       nativeHandlers,
		Closers:              stores.Closers,
	}
	success = true
	return services, nil
}

// BuildNativeHandlers maps resolved auth config into Go-owned HTTP handlers
// mounted by xgoja serve before the JavaScript app host fallback.
func BuildNativeHandlers(ctx context.Context, cfg ResolvedConfig, sessionManager *sessionauth.Manager, stores *StoreBundle, deviceService programauth.DeviceService, auditSink gojahttp.AuditSink, securityEvents gojahttp.SecurityEventObserver) ([]NativeHandler, error) {
	nativeHandlers := []NativeHandler{
		{Method: "GET", Path: "/auth/readyz", Handler: readinessHandler(BuildReadinessReport(cfg))},
	}
	if deviceService.Store != nil {
		deviceHandlers, err := programauth.NewDeviceHandlers(programauth.DeviceHandlersConfig{Service: deviceService, SessionManager: sessionManager, Audit: auditSink, SecurityEvents: securityEvents})
		if err != nil {
			return nil, err
		}
		nativeHandlers = append(nativeHandlers,
			NativeHandler{Method: "POST", Path: "/auth/device/start", Handler: deviceHandlers.StartHandler()},
			NativeHandler{Method: "POST", Path: "/auth/device/token", Handler: deviceHandlers.TokenHandler()},
			NativeHandler{Method: "POST", Path: "/auth/device/refresh", Handler: deviceHandlers.RefreshHandler()},
			NativeHandler{Method: "POST", Path: "/auth/device/revoke", Handler: deviceHandlers.RevokeHandler()},
			NativeHandler{Method: "POST", Path: "/auth/device/approve", Handler: deviceHandlers.ApproveHandler()},
		)
	}
	if cfg.Mode != ModeOIDC {
		return nativeHandlers, nil
	}
	if sessionManager == nil {
		return nil, configError("auth.session", errors.New("session manager is required for auth.mode=oidc"))
	}
	if stores == nil || stores.AppAuth.Users == nil {
		return nil, configError("auth.stores.appauth", errors.New("app auth user store is required for auth.mode=oidc"))
	}
	if stores.OIDCTransaction == nil {
		return nil, configError("auth.stores.oidc-transaction", errors.New("oidc transaction store is required for auth.mode=oidc"))
	}
	handlers, err := keycloakauth.New(ctx, keycloakauth.Config{
		IssuerURL:        cfg.OIDC.IssuerURL,
		ClientID:         cfg.OIDC.ClientID,
		ClientSecret:     cfg.OIDC.ClientSecret,
		RedirectURL:      cfg.OIDC.RedirectURL,
		Scopes:           cfg.OIDC.Scopes,
		AfterLoginURL:    cfg.OIDC.AfterLoginURL,
		AfterLogoutURL:   cfg.OIDC.AfterLogoutURL,
		SessionManager:   sessionManager,
		UserNormalizer:   DefaultOIDCUserNormalizer(stores),
		TransactionStore: stores.OIDCTransaction,
		Audit:            auditSink,
		SecurityEvents:   securityEvents,
	})
	if err != nil {
		return nil, err
	}
	nativeHandlers = append(nativeHandlers,
		NativeHandler{Method: "GET", Path: "/auth/login", Handler: handlers.LoginHandler()},
		NativeHandler{Method: "GET", Path: "/auth/callback", Handler: handlers.CallbackHandler()},
		NativeHandler{Method: "POST", Path: "/auth/logout", Handler: handlers.LogoutHandler()},
		NativeHandler{Method: "GET", Path: "/auth/logout", Handler: handlers.LogoutHandler()},
		NativeHandler{Method: "GET", Path: "/auth/session", Handler: sessionInfoHandler(sessionManager)},
	)
	return nativeHandlers, nil
}

func buildRateLimiter(cfg ResolvedRateLimiterConfig) (gojahttp.RateLimiter, error) {
	switch cfg.Driver {
	case RateLimiterDriverMemory:
		return gojahttp.NewMemoryRateLimiter(), nil
	default:
		return nil, configError("auth.rate-limiter.driver", errors.New("unsupported rate limiter driver"))
	}
}

func sessionInfoHandler(sessionManager *sessionauth.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionManager.SessionFromRequest(r.Context(), r)
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		writeJSON(w, map[string]any{
			"userId":        session.UserID,
			"email":         session.Email,
			"emailVerified": session.EmailVerified,
			"csrfToken":     session.CSRFToken,
			"tenantIds":     append([]string(nil), session.TenantIDs...),
			"claims":        cloneSessionClaims(session.Claims),
		})
	})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, "encode json", http.StatusInternalServerError)
	}
}

func cloneSessionClaims(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// DefaultOIDCUserNormalizer upserts an app user by stable OIDC subject and
// projects existing app memberships into the application session. It does not
// grant roles or seed tenants; application seeding remains outside generic
// hostauth.
func DefaultOIDCUserNormalizer(stores *StoreBundle) keycloakauth.UserNormalizer {
	return keycloakauth.UserNormalizerFunc(func(ctx context.Context, claims keycloakauth.OIDCClaims) (keycloakauth.UserSession, error) {
		user, err := stores.AppAuth.Users.UpsertFromOIDC(ctx, claims.Subject, claims.Email, claims.EmailVerified)
		if err != nil {
			return keycloakauth.UserSession{}, err
		}
		tenantIDs := []string(nil)
		if stores.AppAuth.Memberships != nil {
			memberships, err := stores.AppAuth.Memberships.MembershipsForUser(ctx, user.ID)
			if err != nil {
				return keycloakauth.UserSession{}, err
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
		return keycloakauth.UserSession{
			UserID:        user.ID,
			Email:         user.Email,
			EmailVerified: user.EmailVerified,
			TenantIDs:     tenantIDs,
			Claims: map[string]any{
				"keycloakSub":       claims.Subject,
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
func BuildAuthOptions(sessionManager *sessionauth.Manager, stores *StoreBundle, auditSink gojahttp.AuditSink, rateLimiter gojahttp.RateLimiter, apiTokens programauth.BearerAuthenticator, accessTokens programauth.BearerAuthenticator) gojahttp.AuthOptions {
	var options gojahttp.AuthOptions
	if sessionManager != nil {
		options.Authenticator = sessionManager
		options.CSRF = sessionManager
	}
	if apiTokens != nil || accessTokens != nil {
		options.Authenticator = programauth.CompositeAuthenticator{Session: sessionManager, APITokens: apiTokens, AccessTokens: accessTokens}
	}
	if auditSink != nil {
		options.Audit = auditSink
	}
	if rateLimiter != nil {
		options.RateLimiter = rateLimiter
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
