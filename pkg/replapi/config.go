package replapi

import (
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/pkg/errors"
)

// Profile is the opinionated replapi preset selector.
type Profile string

const (
	ProfileRaw         Profile = "raw"
	ProfileInteractive Profile = "interactive"
	ProfilePersistent  Profile = "persistent"
)

// Config controls app-level replapi behavior.
type Config struct {
	Profile     Profile
	Store       *repldb.Store
	AutoRestore bool
	// SessionOptions are the default kernel/session options passed into replsession.
	SessionOptions replsession.SessionOptions
}

// SessionOverrides are app-layer create-session overrides.
//
// Unlike replsession.SessionOptions, this type is intentionally sparse: it lets
// callers override profile/policy defaults at session-creation time without
// having to construct a full kernel policy object up front.
type SessionOverrides struct {
	ID        string
	CreatedAt time.Time
	Profile   *Profile
	Policy    *replsession.SessionPolicy
}

// Option mutates the app config during construction.
type Option func(*Config)

// DefaultConfig preserves the prior replapi behavior: persistent and restore-aware.
func DefaultConfig() Config {
	return ConfigForProfile(ProfilePersistent)
}

// ConfigForProfile returns the preset config for one named profile.
func ConfigForProfile(profile Profile) Config {
	switch profile {
	case ProfileRaw:
		return Config{
			Profile:        ProfileRaw,
			SessionOptions: replsession.RawSessionOptions(),
		}
	case ProfileInteractive:
		return Config{
			Profile:        ProfileInteractive,
			SessionOptions: replsession.InteractiveSessionOptions(),
		}
	case ProfilePersistent:
		fallthrough
	default:
		return Config{
			Profile:        ProfilePersistent,
			AutoRestore:    true,
			SessionOptions: replsession.PersistentSessionOptions(),
		}
	}
}

// RawConfig returns the preset config for near-straight-goja execution.
func RawConfig() Config {
	return ConfigForProfile(ProfileRaw)
}

// InteractiveConfig returns the preset config for in-memory REPL behavior.
func InteractiveConfig() Config {
	return ConfigForProfile(ProfileInteractive)
}

// PersistentConfig returns the preset config for durable restore-aware sessions.
func PersistentConfig(store *repldb.Store) Config {
	config := ConfigForProfile(ProfilePersistent)
	config.Store = store
	return config
}

// WithProfile applies one named profile preset.
func WithProfile(profile Profile) Option {
	return func(config *Config) {
		if config == nil {
			return
		}
		next := ConfigForProfile(profile)
		next.Store = config.Store
		config.Profile = next.Profile
		config.AutoRestore = next.AutoRestore
		config.SessionOptions = next.SessionOptions
	}
}

// WithStore configures the durable SQLite store used by persistent sessions.
func WithStore(store *repldb.Store) Option {
	return func(config *Config) {
		if config == nil {
			return
		}
		config.Store = store
	}
}

// WithAutoRestore overrides the profile default for on-demand restore behavior.
func WithAutoRestore(enabled bool) Option {
	return func(config *Config) {
		if config == nil {
			return
		}
		config.AutoRestore = enabled
	}
}

// WithDefaultSessionOptions overrides the default per-session behavior.
func WithDefaultSessionOptions(opts replsession.SessionOptions) Option {
	return func(config *Config) {
		if config == nil {
			return
		}
		config.SessionOptions = replsession.NormalizeSessionOptions(opts)
	}
}

// WithDefaultSessionPolicy overrides only the default session policy.
func WithDefaultSessionPolicy(policy replsession.SessionPolicy) Option {
	return func(config *Config) {
		if config == nil {
			return
		}
		config.SessionOptions.Policy = replsession.NormalizeSessionPolicy(policy)
	}
}

func normalizeConfig(config Config) Config {
	normalized := config
	if normalized.Profile == "" {
		defaults := DefaultConfig()
		normalized.Profile = defaults.Profile
		normalized.AutoRestore = defaults.AutoRestore
		normalized.SessionOptions = defaults.SessionOptions
	}
	// Propagate Config.Profile into SessionOptions before normalizing
	// so that NormalizeSessionOptions sees the correct profile and does
	// not default it to "interactive".
	if strings.TrimSpace(normalized.SessionOptions.Profile) == "" {
		normalized.SessionOptions.Profile = string(normalized.Profile)
	}
	normalized.SessionOptions = replsession.NormalizeSessionOptions(normalized.SessionOptions)
	return normalized
}

func validateConfig(config Config) error {
	normalized := normalizeConfig(config)
	if normalized.AutoRestore && normalized.Store == nil {
		return errors.New("replapi: auto-restore requires a store")
	}
	if normalized.SessionOptions.Policy.PersistenceEnabled() && normalized.Store == nil {
		return errors.New("replapi: persistent session profile requires a store")
	}
	return nil
}

func resolveCreateSessionOptions(base Config, override SessionOverrides) replsession.SessionOptions {
	resolved := replsession.NormalizeSessionOptions(base.SessionOptions)
	if strings.TrimSpace(override.ID) != "" {
		resolved.ID = strings.TrimSpace(override.ID)
	}
	if !override.CreatedAt.IsZero() {
		resolved.CreatedAt = override.CreatedAt.UTC()
	}
	if override.Profile != nil && strings.TrimSpace(string(*override.Profile)) != "" {
		profileConfig := ConfigForProfile(*override.Profile)
		resolved = replsession.NormalizeSessionOptions(profileConfig.SessionOptions)
		resolved.ID = strings.TrimSpace(override.ID)
		if !override.CreatedAt.IsZero() {
			resolved.CreatedAt = override.CreatedAt.UTC()
		}
	}
	if override.Policy != nil {
		resolved.Policy = replsession.NormalizeSessionPolicy(*override.Policy)
	}
	return replsession.NormalizeSessionOptions(resolved)
}
