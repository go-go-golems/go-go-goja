package hostauth

import (
	"net/http"
	"net/netip"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

// Mode selects the generated-host authentication infrastructure shape.
type Mode string

const (
	ModeNone Mode = "none"
	ModeDev  Mode = "dev"
	ModeOIDC Mode = "oidc"
)

// DeploymentProfile states the operational contract under which a generated
// host is started. Development keeps tutorial-friendly defaults. SingleNode is
// an explicit production contract for exactly one serving process.
type DeploymentProfile string

const (
	DeploymentProfileDevelopment DeploymentProfile = "development"
	DeploymentProfileSingleNode  DeploymentProfile = "single-node"
)

// RateLimiterDriver selects the host-wide limiter implementation. Memory is
// safe only for DeploymentProfileSingleNode because its counters are local to
// one process.
type RateLimiterDriver string

const (
	RateLimiterDriverMemory RateLimiterDriver = "memory"
)

// StoreDriver selects the persistence backend for a host auth store.
type StoreDriver string

const (
	StoreDriverMemory   StoreDriver = "memory"
	StoreDriverSQLite   StoreDriver = "sqlite"
	StoreDriverPostgres StoreDriver = "postgres"
)

// Config is the generated-host auth infrastructure configuration. It is host
// config, not JavaScript route config and not an authorization policy DSL.
type Config struct {
	Mode        Mode              `yaml:"mode" json:"mode"`
	Deployment  DeploymentConfig  `yaml:"deployment" json:"deployment"`
	Session     SessionConfig     `yaml:"session" json:"session"`
	Stores      StoresConfig      `yaml:"stores" json:"stores"`
	OIDC        OIDCConfig        `yaml:"oidc" json:"oidc"`
	RateLimiter RateLimiterConfig `yaml:"rate-limiter" json:"rate-limiter"`
	Proxy       ProxyConfig       `yaml:"proxy" json:"proxy"`
	Device      DeviceConfig      `yaml:"device" json:"device"`
}

// DeploymentConfig controls the explicit operational profile of the host.
type DeploymentConfig struct {
	Profile DeploymentProfile `yaml:"profile" json:"profile"`
}

// RateLimiterConfig controls the host-wide limiter. A future distributed
// implementation must use a distinct driver rather than silently changing the
// semantics of memory.
type RateLimiterConfig struct {
	Driver RateLimiterDriver `yaml:"driver" json:"driver"`
}

// ProxyConfig defines the only forwarding-header trust policy supported by a
// generated host. Trusted CIDRs describe direct reverse-proxy peers, never
// client addresses advertised in a header.
type ProxyConfig struct {
	Mode         gojahttp.ProxyMode `yaml:"mode" json:"mode"`
	TrustedCIDRs []string           `yaml:"trusted-cidrs" json:"trusted-cidrs"`
}

type DeviceConfig struct {
	AllowedActions  []string `yaml:"allowed-actions" json:"allowed-actions"`
	MaxActions      int      `yaml:"max-actions" json:"max-actions"`
	VerificationURI string   `yaml:"verification-uri" json:"verification-uri"`
}

// OIDCConfig controls generated-host browser login when Mode is oidc.
// PublicBaseURL is the preferred production input; RedirectURL is an advanced
// explicit callback override.
type OIDCConfig struct {
	IssuerURL      string   `yaml:"issuer-url" json:"issuer-url"`
	ClientID       string   `yaml:"client-id" json:"client-id"`
	ClientSecret   string   `yaml:"client-secret" json:"client-secret"`
	PublicBaseURL  string   `yaml:"public-base-url" json:"public-base-url"`
	RedirectURL    string   `yaml:"redirect-url" json:"redirect-url"`
	Scopes         []string `yaml:"scopes" json:"scopes"`
	AfterLoginURL  string   `yaml:"after-login-url" json:"after-login-url"`
	AfterLogoutURL string   `yaml:"after-logout-url" json:"after-logout-url"`
}

// SessionConfig controls server-side app session behavior.
type SessionConfig struct {
	Cookie          CookieConfig `yaml:"cookie" json:"cookie"`
	IdleTimeout     string       `yaml:"idle-timeout" json:"idle-timeout"`
	AbsoluteTimeout string       `yaml:"absolute-timeout" json:"absolute-timeout"`
}

// CookieConfig controls the app session cookie. Empty Name delegates to
// sessionauth.New's secure default cookie name.
type CookieConfig struct {
	AllowInsecureHTTP bool   `yaml:"allow-insecure-http" json:"allow-insecure-http"`
	Name              string `yaml:"name" json:"name"`
	SameSite          string `yaml:"same-site" json:"same-site"`
	Path              string `yaml:"path" json:"path"`
}

// StoresConfig configures the persistent stores used by host-owned auth
// infrastructure. Per-store blocks inherit from Default field-by-field.
type StoresConfig struct {
	Default     StoreConfig `yaml:"default" json:"default"`
	Session     StoreConfig `yaml:"session" json:"session"`
	Audit       StoreConfig `yaml:"audit" json:"audit"`
	AppAuth     StoreConfig `yaml:"appauth" json:"appauth"`
	Capability  StoreConfig `yaml:"capability" json:"capability"`
	ProgramAuth StoreConfig `yaml:"programauth" json:"programauth"`
	// OIDCTransaction stores short-lived state, nonce, and PKCE verifier
	// material. It is intentionally separate from durable application sessions.
	OIDCTransaction StoreConfig `yaml:"oidc-transaction" json:"oidc-transaction"`
}

// StoreConfig configures one store. ApplySchema is a pointer so inheritance can
// distinguish an omitted value from an explicit false override.
type StoreConfig struct {
	Driver      string `yaml:"driver" json:"driver"`
	DSN         string `yaml:"dsn" json:"dsn"`
	ApplySchema *bool  `yaml:"apply-schema" json:"apply-schema"`
}

// ResolvedConfig is the fully parsed and defaulted configuration used by
// builders. It contains no unresolved env references.
type ResolvedConfig struct {
	Mode        Mode
	Deployment  ResolvedDeploymentConfig
	Session     ResolvedSessionConfig
	Stores      ResolvedStoresConfig
	OIDC        ResolvedOIDCConfig
	RateLimiter ResolvedRateLimiterConfig
	Proxy       ResolvedProxyConfig
	Device      ResolvedDeviceConfig
}

type ResolvedDeploymentConfig struct {
	Profile DeploymentProfile
}

type ResolvedRateLimiterConfig struct {
	Driver RateLimiterDriver
}

type ResolvedProxyConfig struct {
	Mode            gojahttp.ProxyMode
	TrustedPrefixes []netip.Prefix
}

type ResolvedDeviceConfig struct {
	AllowedActions  map[string]struct{}
	MaxActions      int
	VerificationURI string
}

// ResolvedOIDCConfig contains validated OIDC settings. RedirectURL is always
// concrete when mode=oidc.
type ResolvedOIDCConfig struct {
	IssuerURL      string
	ClientID       string
	ClientSecret   string
	RedirectURL    string
	Scopes         []string
	AfterLoginURL  string
	AfterLogoutURL string
}

type ResolvedSessionConfig struct {
	Cookie          ResolvedCookieConfig
	IdleTimeout     time.Duration
	AbsoluteTimeout time.Duration
}

type ResolvedCookieConfig struct {
	AllowInsecureHTTP bool
	Name              string
	SameSite          http.SameSite
	Path              string
}

type ResolvedStoresConfig struct {
	Session         ResolvedStoreConfig
	Audit           ResolvedStoreConfig
	AppAuth         ResolvedStoreConfig
	Capability      ResolvedStoreConfig
	ProgramAuth     ResolvedStoreConfig
	OIDCTransaction ResolvedStoreConfig
}

type ResolvedStoreConfig struct {
	Name        string
	Driver      StoreDriver
	DSN         string
	ApplySchema bool
}
