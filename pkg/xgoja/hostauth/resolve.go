package hostauth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var ErrOIDCNotImplemented = errors.New("hostauth: auth.mode=oidc is not implemented yet")

type ResolveOptions struct{}

type ConfigError struct {
	Path string
	Err  error
}

func (e *ConfigError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Path == "" {
		return e.Err.Error()
	}
	return e.Path + ": " + e.Err.Error()
}

func (e *ConfigError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func ResolveConfig(cfg Config, opts ResolveOptions) (ResolvedConfig, error) {
	mode, err := parseMode(cfg.Mode)
	if err != nil {
		return ResolvedConfig{}, configError("auth.mode", err)
	}
	if mode == ModeOIDC {
		return ResolvedConfig{}, configError("auth.mode", ErrOIDCNotImplemented)
	}

	session, err := resolveSessionConfig(cfg.Session)
	if err != nil {
		return ResolvedConfig{}, err
	}
	if mode == ModeNone {
		stores, err := resolveStoresConfig(StoresConfig{})
		if err != nil {
			return ResolvedConfig{}, err
		}
		return ResolvedConfig{Mode: mode, Session: session, Stores: stores}, nil
	}
	stores, err := resolveStoresConfig(cfg.Stores)
	if err != nil {
		return ResolvedConfig{}, err
	}
	return ResolvedConfig{Mode: mode, Session: session, Stores: stores}, nil
}

func configError(path string, err error) error {
	return &ConfigError{Path: path, Err: err}
}

func parseMode(mode Mode) (Mode, error) {
	switch normalized := Mode(strings.ToLower(strings.TrimSpace(string(mode)))); normalized {
	case "":
		return ModeNone, nil
	case ModeNone, ModeDev, ModeOIDC:
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported auth mode %q", mode)
	}
}

func resolveSessionConfig(cfg SessionConfig) (ResolvedSessionConfig, error) {
	sameSite, err := parseSameSite(cfg.Cookie.SameSite)
	if err != nil {
		return ResolvedSessionConfig{}, configError("auth.session.cookie.same-site", err)
	}
	idleTimeout, err := parseOptionalDuration("auth.session.idle-timeout", cfg.IdleTimeout)
	if err != nil {
		return ResolvedSessionConfig{}, err
	}
	absoluteTimeout, err := parseOptionalDuration("auth.session.absolute-timeout", cfg.AbsoluteTimeout)
	if err != nil {
		return ResolvedSessionConfig{}, err
	}
	path := strings.TrimSpace(cfg.Cookie.Path)
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		return ResolvedSessionConfig{}, configError("auth.session.cookie.path", fmt.Errorf("must start with /"))
	}
	return ResolvedSessionConfig{
		Cookie: ResolvedCookieConfig{
			AllowInsecureHTTP: cfg.Cookie.AllowInsecureHTTP,
			Name:              strings.TrimSpace(cfg.Cookie.Name),
			SameSite:          sameSite,
			Path:              path,
		},
		IdleTimeout:     idleTimeout,
		AbsoluteTimeout: absoluteTimeout,
	}, nil
}

func parseSameSite(value string) (http.SameSite, error) {
	switch normalized := strings.ToLower(strings.TrimSpace(value)); normalized {
	case "", "lax":
		return http.SameSiteLaxMode, nil
	case "strict":
		return http.SameSiteStrictMode, nil
	case "none":
		return http.SameSiteNoneMode, nil
	case "default":
		return http.SameSiteDefaultMode, nil
	default:
		return http.SameSiteDefaultMode, fmt.Errorf("unsupported SameSite mode %q", value)
	}
}

func parseOptionalDuration(path string, value string) (time.Duration, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, configError(path, err)
	}
	if d <= 0 {
		return 0, configError(path, fmt.Errorf("must be positive"))
	}
	return d, nil
}

func resolveStoresConfig(cfg StoresConfig) (ResolvedStoresConfig, error) {
	defaults := cfg.Default
	if strings.TrimSpace(defaults.Driver) == "" {
		defaults.Driver = string(StoreDriverMemory)
	}

	session, err := resolveStoreConfig("session", cfg.Session, defaults)
	if err != nil {
		return ResolvedStoresConfig{}, err
	}
	audit, err := resolveStoreConfig("audit", cfg.Audit, defaults)
	if err != nil {
		return ResolvedStoresConfig{}, err
	}
	appauth, err := resolveStoreConfig("appauth", cfg.AppAuth, defaults)
	if err != nil {
		return ResolvedStoresConfig{}, err
	}
	capability, err := resolveStoreConfig("capability", cfg.Capability, defaults)
	if err != nil {
		return ResolvedStoresConfig{}, err
	}
	return ResolvedStoresConfig{Session: session, Audit: audit, AppAuth: appauth, Capability: capability}, nil
}

func resolveStoreConfig(name string, specific StoreConfig, defaults StoreConfig) (ResolvedStoreConfig, error) {
	path := "auth.stores." + name
	merged := inheritStoreConfig(specific, defaults)
	driver, err := parseStoreDriver(merged.Driver)
	if err != nil {
		return ResolvedStoreConfig{}, configError(path+".driver", err)
	}

	applySchema := false
	if merged.ApplySchema != nil {
		applySchema = *merged.ApplySchema
	}
	if driver == StoreDriverMemory {
		return ResolvedStoreConfig{Name: name, Driver: driver, ApplySchema: applySchema}, nil
	}

	dsn, err := resolveDSN(path, merged)
	if err != nil {
		return ResolvedStoreConfig{}, err
	}
	return ResolvedStoreConfig{Name: name, Driver: driver, DSN: dsn, ApplySchema: applySchema}, nil
}

func inheritStoreConfig(specific StoreConfig, defaults StoreConfig) StoreConfig {
	out := defaults
	if strings.TrimSpace(specific.Driver) != "" {
		out.Driver = specific.Driver
	}
	if strings.TrimSpace(specific.DSN) != "" {
		out.DSN = specific.DSN
	}
	if specific.ApplySchema != nil {
		out.ApplySchema = specific.ApplySchema
	}
	return out
}

func parseStoreDriver(value string) (StoreDriver, error) {
	switch normalized := StoreDriver(strings.ToLower(strings.TrimSpace(value))); normalized {
	case "":
		return StoreDriverMemory, nil
	case StoreDriverMemory, StoreDriverSQLite, StoreDriverPostgres:
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported store driver %q", value)
	}
}

func resolveDSN(path string, cfg StoreConfig) (string, error) {
	dsn := strings.TrimSpace(cfg.DSN)
	if dsn == "" {
		return "", configError(path+".dsn", fmt.Errorf("dsn is required for non-memory stores"))
	}
	return dsn, nil
}
