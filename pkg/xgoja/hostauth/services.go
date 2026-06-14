package hostauth

import (
	"context"

	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
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

	SessionManager *sessionauth.Manager
	SessionStore   sessionauth.Store

	AuditSink  gojahttp.AuditSink
	AuditStore audit.Store

	AppAuth    AppAuthStores
	Capability capability.Store

	Closers []func(context.Context) error
}
