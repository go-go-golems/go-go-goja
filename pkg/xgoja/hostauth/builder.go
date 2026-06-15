package hostauth

import (
	"context"
	"errors"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
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
	services := &Services{
		Config:         resolved,
		AuthOptions:    authOptions,
		SessionManager: sessionManager,
		SessionStore:   stores.Session,
		AuditSink:      auditSink,
		AuditStore:     stores.Audit,
		AppAuth:        stores.AppAuth,
		Capability:     stores.Capability,
		Closers:        stores.Closers,
	}
	success = true
	return services, nil
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
