package hostauth

import (
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

const SectionSlug = "auth"

// GlazedSettings is the flat, public command/config/env shape for generated-host
// auth settings. It intentionally mirrors CLI field names rather than the nested
// hostauth.Config shape so generated commands can expose clear --auth-* flags.
type GlazedSettings struct {
	Mode string `glazed:"mode"`

	SessionCookieAllowInsecureHTTP bool   `glazed:"session-cookie-allow-insecure-http"`
	SessionCookieName              string `glazed:"session-cookie-name"`
	SessionCookieSameSite          string `glazed:"session-cookie-same-site"`
	SessionCookiePath              string `glazed:"session-cookie-path"`
	SessionIdleTimeout             string `glazed:"session-idle-timeout"`
	SessionAbsoluteTimeout         string `glazed:"session-absolute-timeout"`

	DefaultStoreDriver      string `glazed:"default-store-driver"`
	DefaultStoreDSN         string `glazed:"default-store-dsn"`
	DefaultStoreApplySchema bool   `glazed:"default-store-apply-schema"`

	SessionStoreDriver      string `glazed:"session-store-driver"`
	SessionStoreDSN         string `glazed:"session-store-dsn"`
	SessionStoreApplySchema bool   `glazed:"session-store-apply-schema"`

	AuditStoreDriver      string `glazed:"audit-store-driver"`
	AuditStoreDSN         string `glazed:"audit-store-dsn"`
	AuditStoreApplySchema bool   `glazed:"audit-store-apply-schema"`

	AppAuthStoreDriver      string `glazed:"appauth-store-driver"`
	AppAuthStoreDSN         string `glazed:"appauth-store-dsn"`
	AppAuthStoreApplySchema bool   `glazed:"appauth-store-apply-schema"`

	CapabilityStoreDriver      string `glazed:"capability-store-driver"`
	CapabilityStoreDSN         string `glazed:"capability-store-dsn"`
	CapabilityStoreApplySchema bool   `glazed:"capability-store-apply-schema"`
}

// ConfigDefaultsProvider is implemented by service factories that can expose
// their base auth config for command help/default generation without opening
// stores or resolving runtime settings.
type ConfigDefaultsProvider interface {
	AuthConfigDefaults() Config
}

func GlazedConfigSection(base Config, opts ...schema.SectionOption) (schema.Section, error) {
	defaults := FlattenConfig(base)
	fieldDefs := []schema.SectionOption{schema.WithFields(
		fields.New("mode", fields.TypeChoice, fields.WithChoices(string(ModeNone), string(ModeDev), string(ModeOIDC)), fields.WithDefault(defaults.Mode), fields.WithHelp("Generated-host auth mode")),
		fields.New("session-cookie-allow-insecure-http", fields.TypeBool, fields.WithDefault(defaults.SessionCookieAllowInsecureHTTP), fields.WithHelp("Allow non-Secure auth session cookies for local HTTP demos")),
		fields.New("session-cookie-name", fields.TypeString, fields.WithDefault(defaults.SessionCookieName), fields.WithHelp("Auth session cookie name; empty uses the session manager default")),
		fields.New("session-cookie-same-site", fields.TypeChoice, fields.WithChoices("", "lax", "strict", "none", "default"), fields.WithDefault(defaults.SessionCookieSameSite), fields.WithHelp("Auth session cookie SameSite mode")),
		fields.New("session-cookie-path", fields.TypeString, fields.WithDefault(defaults.SessionCookiePath), fields.WithHelp("Auth session cookie path")),
		fields.New("session-idle-timeout", fields.TypeString, fields.WithDefault(defaults.SessionIdleTimeout), fields.WithHelp("Idle timeout for app sessions as a Go duration")),
		fields.New("session-absolute-timeout", fields.TypeString, fields.WithDefault(defaults.SessionAbsoluteTimeout), fields.WithHelp("Absolute timeout for app sessions as a Go duration")),
	)}
	opts = append(opts, fieldDefs...)
	opts = append(opts, storeFields("default", defaults.DefaultStoreDriver, defaults.DefaultStoreDSN, defaults.DefaultStoreApplySchema)...)
	opts = append(opts, storeFields("session", defaults.SessionStoreDriver, defaults.SessionStoreDSN, defaults.SessionStoreApplySchema)...)
	opts = append(opts, storeFields("audit", defaults.AuditStoreDriver, defaults.AuditStoreDSN, defaults.AuditStoreApplySchema)...)
	opts = append(opts, storeFields("appauth", defaults.AppAuthStoreDriver, defaults.AppAuthStoreDSN, defaults.AppAuthStoreApplySchema)...)
	opts = append(opts, storeFields("capability", defaults.CapabilityStoreDriver, defaults.CapabilityStoreDSN, defaults.CapabilityStoreApplySchema)...)
	return schema.NewSection(SectionSlug, "Generated host auth", opts...)
}

func storeFields(prefix, driver, dsn string, applySchema bool) []schema.SectionOption {
	return []schema.SectionOption{schema.WithFields(
		fields.New(prefix+"-store-driver", fields.TypeChoice, fields.WithChoices("", string(StoreDriverMemory), string(StoreDriverSQLite), string(StoreDriverPostgres)), fields.WithDefault(driver), fields.WithHelp("Auth "+prefix+" store driver")),
		fields.New(prefix+"-store-dsn", fields.TypeString, fields.WithDefault(dsn), fields.WithHelp("Auth "+prefix+" store DSN for SQL stores")),
		fields.New(prefix+"-store-apply-schema", fields.TypeBool, fields.WithDefault(applySchema), fields.WithHelp("Apply the built-in schema for the auth "+prefix+" store")),
	)}
}

func FlattenConfig(cfg Config) GlazedSettings {
	defaults := cfg.Stores.Default
	if strings.TrimSpace(defaults.Driver) == "" {
		defaults.Driver = string(StoreDriverMemory)
	}
	session := cfg.Stores.Session
	audit := cfg.Stores.Audit
	appauth := cfg.Stores.AppAuth
	capability := cfg.Stores.Capability
	return GlazedSettings{
		Mode: cfgModeDefault(cfg.Mode),

		SessionCookieAllowInsecureHTTP: cfg.Session.Cookie.AllowInsecureHTTP,
		SessionCookieName:              strings.TrimSpace(cfg.Session.Cookie.Name),
		SessionCookieSameSite:          strings.TrimSpace(cfg.Session.Cookie.SameSite),
		SessionCookiePath:              cookiePathDefault(cfg.Session.Cookie.Path),
		SessionIdleTimeout:             strings.TrimSpace(cfg.Session.IdleTimeout),
		SessionAbsoluteTimeout:         strings.TrimSpace(cfg.Session.AbsoluteTimeout),

		DefaultStoreDriver:      storeDriverDefault(defaults.Driver),
		DefaultStoreDSN:         strings.TrimSpace(defaults.DSN),
		DefaultStoreApplySchema: boolValue(defaults.ApplySchema),

		SessionStoreDriver:      strings.TrimSpace(session.Driver),
		SessionStoreDSN:         strings.TrimSpace(session.DSN),
		SessionStoreApplySchema: boolValue(session.ApplySchema),

		AuditStoreDriver:      strings.TrimSpace(audit.Driver),
		AuditStoreDSN:         strings.TrimSpace(audit.DSN),
		AuditStoreApplySchema: boolValue(audit.ApplySchema),

		AppAuthStoreDriver:      strings.TrimSpace(appauth.Driver),
		AppAuthStoreDSN:         strings.TrimSpace(appauth.DSN),
		AppAuthStoreApplySchema: boolValue(appauth.ApplySchema),

		CapabilityStoreDriver:      strings.TrimSpace(capability.Driver),
		CapabilityStoreDSN:         strings.TrimSpace(capability.DSN),
		CapabilityStoreApplySchema: boolValue(capability.ApplySchema),
	}
}

func ConfigFromValues(vals *values.Values, base Config) (Config, error) {
	if vals == nil || !valuesContainAuthSection(vals) {
		return base, nil
	}
	settings := GlazedSettings{}
	if err := vals.DecodeSectionInto(SectionSlug, &settings); err != nil {
		return Config{}, err
	}
	return settings.ToConfig(), nil
}

func valuesContainAuthSection(vals *values.Values) bool {
	if vals == nil {
		return false
	}
	found := false
	vals.ForEach(func(slug string, _ *values.SectionValues) {
		if slug == SectionSlug {
			found = true
		}
	})
	return found
}

func (s GlazedSettings) ToConfig() Config {
	return Config{
		Mode: Mode(strings.TrimSpace(s.Mode)),
		Session: SessionConfig{
			Cookie: CookieConfig{
				AllowInsecureHTTP: s.SessionCookieAllowInsecureHTTP,
				Name:              strings.TrimSpace(s.SessionCookieName),
				SameSite:          strings.TrimSpace(s.SessionCookieSameSite),
				Path:              strings.TrimSpace(s.SessionCookiePath),
			},
			IdleTimeout:     strings.TrimSpace(s.SessionIdleTimeout),
			AbsoluteTimeout: strings.TrimSpace(s.SessionAbsoluteTimeout),
		},
		Stores: StoresConfig{
			Default:    storeConfigFromGlazed(s.DefaultStoreDriver, s.DefaultStoreDSN, s.DefaultStoreApplySchema),
			Session:    storeConfigFromGlazed(s.SessionStoreDriver, s.SessionStoreDSN, s.SessionStoreApplySchema),
			Audit:      storeConfigFromGlazed(s.AuditStoreDriver, s.AuditStoreDSN, s.AuditStoreApplySchema),
			AppAuth:    storeConfigFromGlazed(s.AppAuthStoreDriver, s.AppAuthStoreDSN, s.AppAuthStoreApplySchema),
			Capability: storeConfigFromGlazed(s.CapabilityStoreDriver, s.CapabilityStoreDSN, s.CapabilityStoreApplySchema),
		},
	}
}

func storeConfigFromGlazed(driver, dsn string, applySchema bool) StoreConfig {
	return StoreConfig{Driver: strings.TrimSpace(driver), DSN: strings.TrimSpace(dsn), ApplySchema: boolPtr(applySchema)}
}

func cfgModeDefault(mode Mode) string {
	mode = Mode(strings.TrimSpace(string(mode)))
	if mode == "" {
		return string(ModeNone)
	}
	return string(mode)
}

func storeDriverDefault(driver string) string {
	driver = strings.TrimSpace(driver)
	if driver == "" {
		return string(StoreDriverMemory)
	}
	return driver
}

func cookiePathDefault(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	return path
}

func boolValue(v *bool) bool {
	return v != nil && *v
}

func boolPtr(v bool) *bool { return &v }
