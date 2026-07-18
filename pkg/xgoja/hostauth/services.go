package hostauth

import (
	"context"
	"net/http"

	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/keycloakauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

const (
	// ServicesKey stores concrete auth services built for a host/runtime/command.
	ServicesKey = "go-go-goja-auth.services"
	// ServiceFactoryKey stores a lazy auth service factory. Command providers use
	// this key during command construction, then build concrete services after
	// command values have been parsed.
	ServiceFactoryKey = "go-go-goja-auth.service-factory"
)

// ServiceFactory builds concrete host auth services at command execution time.
type ServiceFactory interface {
	BuildHostAuthServices(ctx context.Context, vals *values.Values) (*Services, error)
}

// NativeHandler describes a Go-owned HTTP route mounted by the command-owned
// server before the JavaScript app host fallback.
type NativeHandler struct {
	Method  string
	Path    string
	Handler http.Handler
}

// AppAuthStores groups the app-owned authorization data stores.
type AppAuthStores struct {
	Users       appauth.UserStore
	Memberships appauth.MembershipStore
	Resources   appauth.ResourceStore
}

// Services contains concrete host-owned auth infrastructure. It is intentionally
// a Go service payload for generated/custom hosts, not a JavaScript API.
type Services struct {
	Config      ResolvedConfig
	AuthOptions gojahttp.AuthOptions

	SessionManager       *sessionauth.Manager
	SessionStore         sessionauth.Store
	OIDCTransactionStore keycloakauth.TransactionStore

	AuditSink  gojahttp.AuditSink
	AuditStore audit.Store

	RateLimiter     gojahttp.RateLimiter
	RequestIdentity gojahttp.TrustedProxyResolver
	// SecurityEvents receives bounded lifecycle observations. BuilderOptions may
	// supply a production metrics bridge; otherwise the builder retains an
	// in-memory counter for diagnostics and integration tests.
	SecurityEvents gojahttp.SecurityEventObserver

	AppAuth    AppAuthStores
	Capability capability.Store

	AgentStore        programauth.AgentStore
	APITokenStore     programauth.APITokenStore
	AccessTokenStore  programauth.AccessTokenStore
	RefreshTokenStore programauth.RefreshTokenStore
	DeviceStore       programauth.DeviceAuthorizationStore
	Agents            programauth.AgentService
	APITokens         programauth.APITokenService
	OAuthTokens       programauth.OAuthTokenService
	Devices           programauth.DeviceService
	Maintenance       programauth.MaintenanceService

	NativeHandlers []NativeHandler

	Closers []func(context.Context) error
}

// Close closes resources owned by the services bundle.
func (s *Services) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	return closeAll(ctx, s.Closers)
}
