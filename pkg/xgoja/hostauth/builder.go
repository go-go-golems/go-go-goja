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
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/keycloakauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

// BuilderOptions configures a generated-host auth service factory.
type BuilderOptions struct {
	Config      Config
	ActorLoader sessionauth.ActorLoader
	Now         func() time.Time
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
	nativeHandlers, err := BuildNativeHandlers(ctx, resolved, sessionManager, stores)
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
func BuildNativeHandlers(ctx context.Context, cfg ResolvedConfig, sessionManager *sessionauth.Manager, stores *StoreBundle) ([]NativeHandler, error) {
	if cfg.Mode != ModeOIDC {
		return nil, nil
	}
	if sessionManager == nil {
		return nil, configError("auth.session", errors.New("session manager is required for auth.mode=oidc"))
	}
	if stores == nil || stores.AppAuth.Users == nil {
		return nil, configError("auth.stores.appauth", errors.New("app auth user store is required for auth.mode=oidc"))
	}
	handlers, err := keycloakauth.New(ctx, keycloakauth.Config{
		IssuerURL:      cfg.OIDC.IssuerURL,
		ClientID:       cfg.OIDC.ClientID,
		ClientSecret:   cfg.OIDC.ClientSecret,
		RedirectURL:    cfg.OIDC.RedirectURL,
		Scopes:         cfg.OIDC.Scopes,
		AfterLoginURL:  cfg.OIDC.AfterLoginURL,
		AfterLogoutURL: cfg.OIDC.AfterLogoutURL,
		SessionManager: sessionManager,
		UserNormalizer: DefaultOIDCUserNormalizer(stores),
	})
	if err != nil {
		return nil, err
	}
	capabilityService := capability.Service{Store: stores.Capability}
	return []NativeHandler{
		{Method: "GET", Path: "/auth/login", Handler: handlers.LoginHandler()},
		{Method: "GET", Path: "/auth/callback", Handler: handlers.CallbackHandler()},
		{Method: "POST", Path: "/auth/logout", Handler: handlers.LogoutHandler()},
		{Method: "GET", Path: "/auth/logout", Handler: handlers.LogoutHandler()},
		{Method: "GET", Path: "/auth/session", Handler: sessionInfoHandler(sessionManager)},
		{Method: "GET", Path: "/auth/audit", Handler: auditRecordsHandler(sessionManager, stores.Audit)},
		{Method: "POST", Path: "/orgs/o1/invites", Handler: issueDemoInviteHandler(sessionManager, stores.AppAuth.Memberships, capabilityService)},
		{Method: "POST", Path: "/org-invites/accept", Handler: acceptDemoInviteHandler(capabilityService)},
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

type auditSnapshotStore interface {
	Snapshot(context.Context) ([]audit.Record, error)
}

type auditMemorySnapshotStore interface {
	Snapshot() []audit.Record
}

func auditRecordsHandler(sessionManager *sessionauth.Manager, store audit.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionManager.SessionFromRequest(r.Context(), r); err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		var records []audit.Record
		switch s := store.(type) {
		case auditSnapshotStore:
			var err error
			records, err = s.Snapshot(r.Context())
			if err != nil {
				http.Error(w, "query audit records", http.StatusInternalServerError)
				return
			}
		case auditMemorySnapshotStore:
			records = s.Snapshot()
		default:
			http.Error(w, "audit store does not support snapshots", http.StatusNotImplemented)
			return
		}
		const limit = 50
		if len(records) > limit {
			records = records[len(records)-limit:]
		}
		writeJSON(w, map[string]any{"records": records, "count": len(records)})
	})
}

func issueDemoInviteHandler(sessionManager *sessionauth.Manager, memberships appauth.MembershipStore, capabilityService capability.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionManager.SessionFromRequest(r.Context(), r)
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		actor := &gojahttp.Actor{ID: session.UserID, Kind: "user"}
		if err := sessionManager.VerifyCSRF(r.Context(), gojahttp.CSRFRequest{HTTPRequest: r, Actor: actor}); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if memberships == nil {
			http.Error(w, "authorization failed", http.StatusInternalServerError)
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
		issued, err := capabilityService.IssueOrgInvite(r.Context(), capability.OrgInviteSpec{OrgID: "o1", Email: req.Email, Role: req.Role, TTL: 15 * time.Minute, CreatedBy: session.UserID})
		if err != nil {
			http.Error(w, "issue invite", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"token": issued.Token, "expiresAt": issued.Capability.ExpiresAt})
	})
}

func acceptDemoInviteHandler(capabilityService capability.Service) http.Handler {
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
			if errors.Is(err, capability.ErrUsed) {
				status = http.StatusConflict
			}
			http.Error(w, err.Error(), status)
			return
		}
		writeJSON(w, accepted)
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
