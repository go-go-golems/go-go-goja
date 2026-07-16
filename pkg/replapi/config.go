package replapi

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
)

// Profile is the opinionated replapi preset selector.
type Profile string

const (
	ProfileRaw         Profile = "raw"
	ProfileInteractive Profile = "interactive"
	ProfilePersistent  Profile = "persistent"

	// DefaultLeaseTTL is renewed before persistent operations and periodically
	// during long evaluation/replay work.
	DefaultLeaseTTL = 30 * time.Second
)

var (
	// ErrUnknownProfile identifies an unsupported app or per-session profile.
	ErrUnknownProfile = errors.New("replapi: unknown profile")
	// ErrProfileMismatch identifies contradictory app and session-option labels.
	ErrProfileMismatch = errors.New("replapi: profile mismatch")
	// ErrInvalidSessionPolicy identifies a structurally inconsistent policy.
	ErrInvalidSessionPolicy = errors.New("replapi: invalid session policy")
)

// UnknownProfileError reports the unsupported profile text supplied by a caller.
type UnknownProfileError struct {
	Value string
}

func (e *UnknownProfileError) Error() string {
	return fmt.Sprintf("%s %q (want raw, interactive, or persistent)", ErrUnknownProfile, e.Value)
}

func (e *UnknownProfileError) Unwrap() error { return ErrUnknownProfile }

// ProfileMismatchError reports disagreement between app and session-option
// profile labels. A policy may deliberately replace a preset, but duplicate
// labels must still name the same preset.
type ProfileMismatchError struct {
	AppProfile     Profile
	SessionProfile Profile
}

func (e *ProfileMismatchError) Error() string {
	return fmt.Sprintf("%s: app=%q session-options=%q", ErrProfileMismatch, e.AppProfile, e.SessionProfile)
}

func (e *ProfileMismatchError) Unwrap() error { return ErrProfileMismatch }

// ParseProfile trims and canonicalizes one profile value.
func ParseProfile(raw string) (Profile, error) {
	profile := Profile(strings.ToLower(strings.TrimSpace(raw)))
	switch profile {
	case ProfileRaw, ProfileInteractive, ProfilePersistent:
		return profile, nil
	default:
		return "", &UnknownProfileError{Value: raw}
	}
}

// ValidateProfile verifies that profile names one supported preset.
func ValidateProfile(profile Profile) error {
	_, err := ParseProfile(string(profile))
	return err
}

// Clock supplies lease time. Production callers normally leave it nil; tests
// can inject a deterministic clock.
type Clock interface {
	Now() time.Time
}

// Config controls app-level replapi behavior.
type Config struct {
	Profile     Profile
	Store       *repldb.Store
	AutoRestore bool
	ownerID     string
	LeaseTTL    time.Duration
	Clock       Clock
	// SessionOptions are the default kernel/session options passed into
	// replsession. A non-zero Policy is an intentional full replacement of the
	// selected profile policy; it is not merged field by field.
	SessionOptions replsession.SessionOptions
}

// SessionOverrides are app-layer create-session overrides.
//
// Unlike replsession.SessionOptions, this type is intentionally sparse: it lets
// callers override profile/policy defaults at session-creation time without
// having to construct a full kernel policy object up front. Policy, when
// non-nil, is a complete replacement rather than a partial merge.
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
	return mustConfigForProfile(ProfilePersistent)
}

// ConfigForProfile returns the complete preset config for one named profile.
// Unknown values return an error instead of silently selecting persistence.
func ConfigForProfile(profile Profile) (Config, error) {
	canonical, err := ParseProfile(string(profile))
	if err != nil {
		return Config{}, err
	}

	switch canonical {
	case ProfileRaw:
		return Config{
			Profile:        ProfileRaw,
			LeaseTTL:       DefaultLeaseTTL,
			SessionOptions: replsession.RawSessionOptions(),
		}, nil
	case ProfileInteractive:
		return Config{
			Profile:        ProfileInteractive,
			LeaseTTL:       DefaultLeaseTTL,
			SessionOptions: replsession.InteractiveSessionOptions(),
		}, nil
	case ProfilePersistent:
		return Config{
			Profile:        ProfilePersistent,
			AutoRestore:    true,
			LeaseTTL:       DefaultLeaseTTL,
			SessionOptions: replsession.PersistentSessionOptions(),
		}, nil
	default:
		panic("unreachable validated replapi profile")
	}
}

func mustConfigForProfile(profile Profile) Config {
	config, err := ConfigForProfile(profile)
	if err != nil {
		panic(err)
	}
	return config
}

// RawConfig returns the preset config for near-straight-goja execution.
func RawConfig() Config {
	return mustConfigForProfile(ProfileRaw)
}

// InteractiveConfig returns the preset config for in-memory REPL behavior.
func InteractiveConfig() Config {
	return mustConfigForProfile(ProfileInteractive)
}

// PersistentConfig returns the preset config for durable restore-aware sessions.
func PersistentConfig(store *repldb.Store) Config {
	config := mustConfigForProfile(ProfilePersistent)
	config.Store = store
	return config
}

// WithProfile applies one named profile preset. Invalid values are retained in
// Config so NewWithConfig can return a typed validation error.
func WithProfile(profile Profile) Option {
	return func(config *Config) {
		if config == nil {
			return
		}
		next, err := ConfigForProfile(profile)
		if err != nil {
			config.Profile = profile
			return
		}
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

// withOwnerIDForTest injects deterministic identity without exposing a public
// impersonation mechanism. Production app owner IDs are always generated.
func withOwnerIDForTest(ownerID string) Option {
	return func(config *Config) {
		if config != nil {
			config.ownerID = strings.TrimSpace(ownerID)
		}
	}
}

// WithLeaseTTL configures the persistent ownership lease duration.
func WithLeaseTTL(ttl time.Duration) Option {
	return func(config *Config) {
		if config != nil {
			config.LeaseTTL = ttl
		}
	}
}

// WithClock injects the clock used for lease acquisition, renewal, and fencing.
func WithClock(clock Clock) Option {
	return func(config *Config) {
		if config != nil {
			config.Clock = clock
		}
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

// WithDefaultSessionOptions replaces the default per-session options. Its
// Profile label must agree with Config.Profile when both are present. A
// non-zero Policy is a full policy replacement.
func WithDefaultSessionOptions(opts replsession.SessionOptions) Option {
	return func(config *Config) {
		if config == nil {
			return
		}
		config.SessionOptions = opts
	}
}

// WithDefaultSessionPolicy fully replaces the selected profile's default
// policy. Zero-valued booleans disable their corresponding features.
func WithDefaultSessionPolicy(policy replsession.SessionPolicy) Option {
	return func(config *Config) {
		if config == nil {
			return
		}
		config.SessionOptions.Policy = replsession.NormalizeSessionPolicy(policy)
	}
}

func normalizeConfig(config Config) (Config, error) {
	requestedProfile := config.Profile
	sessionProfileText := strings.TrimSpace(config.SessionOptions.Profile)

	if strings.TrimSpace(string(requestedProfile)) == "" {
		if sessionProfileText != "" {
			parsed, err := ParseProfile(sessionProfileText)
			if err != nil {
				return Config{}, err
			}
			requestedProfile = parsed
		} else {
			requestedProfile = ProfilePersistent
		}
	}

	canonical, err := ParseProfile(string(requestedProfile))
	if err != nil {
		return Config{}, err
	}
	preset, err := ConfigForProfile(canonical)
	if err != nil {
		return Config{}, err
	}

	if sessionProfileText != "" {
		sessionProfile, err := ParseProfile(sessionProfileText)
		if err != nil {
			return Config{}, err
		}
		if sessionProfile != canonical {
			return Config{}, &ProfileMismatchError{
				AppProfile:     canonical,
				SessionProfile: sessionProfile,
			}
		}
	}

	inputOptions := config.SessionOptions
	optionsWereZero := inputOptions == (replsession.SessionOptions{})
	resolvedOptions := preset.SessionOptions
	if !optionsWereZero {
		if id := strings.TrimSpace(inputOptions.ID); id != "" {
			resolvedOptions.ID = id
		}
		if !inputOptions.CreatedAt.IsZero() {
			resolvedOptions.CreatedAt = inputOptions.CreatedAt.UTC()
		}
		if !inputOptions.Policy.IsZero() {
			resolvedOptions.Policy = replsession.NormalizeSessionPolicy(inputOptions.Policy)
		}
	}
	resolvedOptions.Profile = string(canonical)
	resolvedOptions = replsession.NormalizeSessionOptions(resolvedOptions)
	if err := validateSessionPolicy(resolvedOptions.Policy); err != nil {
		return Config{}, err
	}

	normalized := config
	normalized.Profile = canonical
	normalized.SessionOptions = resolvedOptions
	if normalized.LeaseTTL == 0 {
		normalized.LeaseTTL = preset.LeaseTTL
	}
	if optionsWereZero {
		// A bare Config{Profile: ...} means the complete named preset. Callers
		// that need to override AutoRestore should start from a preset helper or
		// use WithAutoRestore so SessionOptions carry explicit preset state.
		normalized.AutoRestore = preset.AutoRestore
	}
	return normalized, nil
}

func validateConfig(config Config) error {
	if err := ValidateProfile(config.Profile); err != nil {
		return err
	}
	sessionProfile, err := ParseProfile(config.SessionOptions.Profile)
	if err != nil {
		return err
	}
	if sessionProfile != config.Profile {
		return &ProfileMismatchError{
			AppProfile:     config.Profile,
			SessionProfile: sessionProfile,
		}
	}
	if err := validateSessionPolicy(config.SessionOptions.Policy); err != nil {
		return err
	}
	if config.LeaseTTL <= 0 {
		return errors.New("replapi: lease TTL must be positive")
	}
	if config.AutoRestore && config.Store == nil {
		return errors.New("replapi: auto-restore requires a store")
	}
	if config.SessionOptions.Policy.PersistenceEnabled() && config.Store == nil {
		return errors.New("replapi: persistent session profile requires a store")
	}
	return nil
}

func validateSessionPolicy(policy replsession.SessionPolicy) error {
	normalized := replsession.NormalizeSessionPolicy(policy)
	switch normalized.Eval.Mode {
	case replsession.EvalModeRaw, replsession.EvalModeInstrumented:
	default:
		return fmt.Errorf("%w: unsupported eval mode %q", ErrInvalidSessionPolicy, normalized.Eval.Mode)
	}
	if !normalized.Persist.Enabled && (normalized.Persist.Evaluations || normalized.Persist.BindingVersions || normalized.Persist.BindingDocs) {
		return fmt.Errorf("%w: persistence detail flags require persist.enabled", ErrInvalidSessionPolicy)
	}
	return nil
}

func resolveCreateSessionOptions(base Config, override SessionOverrides) (replsession.SessionOptions, error) {
	resolved := replsession.NormalizeSessionOptions(base.SessionOptions)
	if strings.TrimSpace(override.ID) != "" {
		resolved.ID = strings.TrimSpace(override.ID)
	}
	if !override.CreatedAt.IsZero() {
		resolved.CreatedAt = override.CreatedAt.UTC()
	}
	if override.Profile != nil && strings.TrimSpace(string(*override.Profile)) != "" {
		profileConfig, err := ConfigForProfile(*override.Profile)
		if err != nil {
			return replsession.SessionOptions{}, err
		}
		resolved = replsession.NormalizeSessionOptions(profileConfig.SessionOptions)
		resolved.ID = strings.TrimSpace(override.ID)
		if !override.CreatedAt.IsZero() {
			resolved.CreatedAt = override.CreatedAt.UTC()
		}
	}
	if override.Policy != nil {
		resolved.Policy = replsession.NormalizeSessionPolicy(*override.Policy)
	}
	resolved = replsession.NormalizeSessionOptions(resolved)
	if err := validateSessionPolicy(resolved.Policy); err != nil {
		return replsession.SessionOptions{}, err
	}
	return resolved, nil
}
