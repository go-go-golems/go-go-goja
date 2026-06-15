package hostauth

import (
	"net/http"
	"time"
)

// Mode selects the generated-host authentication infrastructure shape.
type Mode string

const (
	ModeNone Mode = "none"
	ModeDev  Mode = "dev"
	ModeOIDC Mode = "oidc"
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
	Mode    Mode          `yaml:"mode" json:"mode"`
	Session SessionConfig `yaml:"session" json:"session"`
	Stores  StoresConfig  `yaml:"stores" json:"stores"`
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
	Default    StoreConfig `yaml:"default" json:"default"`
	Session    StoreConfig `yaml:"session" json:"session"`
	Audit      StoreConfig `yaml:"audit" json:"audit"`
	AppAuth    StoreConfig `yaml:"appauth" json:"appauth"`
	Capability StoreConfig `yaml:"capability" json:"capability"`
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
	Mode    Mode
	Session ResolvedSessionConfig
	Stores  ResolvedStoresConfig
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
	Session    ResolvedStoreConfig
	Audit      ResolvedStoreConfig
	AppAuth    ResolvedStoreConfig
	Capability ResolvedStoreConfig
}

type ResolvedStoreConfig struct {
	Name        string
	Driver      StoreDriver
	DSN         string
	ApplySchema bool
}
