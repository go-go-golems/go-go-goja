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
	Mode string `glazed:"auth-mode"`

	SessionCookieAllowInsecureHTTP bool   `glazed:"auth-session-cookie-allow-insecure-http"`
	SessionCookieName              string `glazed:"auth-session-cookie-name"`
	SessionCookieSameSite          string `glazed:"auth-session-cookie-same-site"`
	SessionCookiePath              string `glazed:"auth-session-cookie-path"`
	SessionIdleTimeout             string `glazed:"auth-session-idle-timeout"`
	SessionAbsoluteTimeout         string `glazed:"auth-session-absolute-timeout"`

	DefaultStoreDriver      string `glazed:"auth-default-store-driver"`
	DefaultStoreDSN         string `glazed:"auth-default-store-dsn"`
	DefaultStoreApplySchema bool   `glazed:"auth-default-store-apply-schema"`

	SessionStoreDriver      string `glazed:"auth-session-store-driver"`
	SessionStoreDSN         string `glazed:"auth-session-store-dsn"`
	SessionStoreApplySchema bool   `glazed:"auth-session-store-apply-schema"`

	AuditStoreDriver      string `glazed:"auth-audit-store-driver"`
	AuditStoreDSN         string `glazed:"auth-audit-store-dsn"`
	AuditStoreApplySchema bool   `glazed:"auth-audit-store-apply-schema"`

	AppAuthStoreDriver      string `glazed:"auth-appauth-store-driver"`
	AppAuthStoreDSN         string `glazed:"auth-appauth-store-dsn"`
	AppAuthStoreApplySchema bool   `glazed:"auth-appauth-store-apply-schema"`

	CapabilityStoreDriver      string `glazed:"auth-capability-store-driver"`
	CapabilityStoreDSN         string `glazed:"auth-capability-store-dsn"`
	CapabilityStoreApplySchema bool   `glazed:"auth-capability-store-apply-schema"`

	ProgramAuthStoreDriver      string `glazed:"auth-programauth-store-driver"`
	ProgramAuthStoreDSN         string `glazed:"auth-programauth-store-dsn"`
	ProgramAuthStoreApplySchema bool   `glazed:"auth-programauth-store-apply-schema"`

	OIDCTransactionStoreDriver      string `glazed:"auth-oidc-transaction-store-driver"`
	OIDCTransactionStoreDSN         string `glazed:"auth-oidc-transaction-store-dsn"`
	OIDCTransactionStoreApplySchema bool   `glazed:"auth-oidc-transaction-store-apply-schema"`

	OIDCIssuerURL      string   `glazed:"auth-oidc-issuer-url"`
	OIDCClientID       string   `glazed:"auth-oidc-client-id"`
	OIDCClientSecret   string   `glazed:"auth-oidc-client-secret"`
	OIDCPublicBaseURL  string   `glazed:"auth-oidc-public-base-url"`
	OIDCRedirectURL    string   `glazed:"auth-oidc-redirect-url"`
	OIDCScopes         []string `glazed:"auth-oidc-scopes"`
	OIDCAfterLoginURL  string   `glazed:"auth-oidc-after-login-url"`
	OIDCAfterLogoutURL string   `glazed:"auth-oidc-after-logout-url"`
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
		fields.New("auth-mode", fields.TypeChoice, fields.WithChoices(string(ModeNone), string(ModeDev), string(ModeOIDC)), fields.WithDefault(defaults.Mode), fields.WithHelp("Generated-host auth mode")),
		fields.New("auth-session-cookie-allow-insecure-http", fields.TypeBool, fields.WithDefault(defaults.SessionCookieAllowInsecureHTTP), fields.WithHelp("Allow non-Secure auth session cookies for local HTTP demos")),
		fields.New("auth-session-cookie-name", fields.TypeString, fields.WithDefault(defaults.SessionCookieName), fields.WithHelp("Auth session cookie name; empty uses the session manager default")),
		fields.New("auth-session-cookie-same-site", fields.TypeChoice, fields.WithChoices("", "lax", "strict", "none", "default"), fields.WithDefault(defaults.SessionCookieSameSite), fields.WithHelp("Auth session cookie SameSite mode")),
		fields.New("auth-session-cookie-path", fields.TypeString, fields.WithDefault(defaults.SessionCookiePath), fields.WithHelp("Auth session cookie path")),
		fields.New("auth-session-idle-timeout", fields.TypeString, fields.WithDefault(defaults.SessionIdleTimeout), fields.WithHelp("Idle timeout for app sessions as a Go duration")),
		fields.New("auth-session-absolute-timeout", fields.TypeString, fields.WithDefault(defaults.SessionAbsoluteTimeout), fields.WithHelp("Absolute timeout for app sessions as a Go duration")),
	)}
	opts = append(opts, fieldDefs...)
	opts = append(opts, storeFields("default", defaults.DefaultStoreDriver, defaults.DefaultStoreDSN, defaults.DefaultStoreApplySchema)...)
	opts = append(opts, storeFields("session", defaults.SessionStoreDriver, defaults.SessionStoreDSN, defaults.SessionStoreApplySchema)...)
	opts = append(opts, storeFields("audit", defaults.AuditStoreDriver, defaults.AuditStoreDSN, defaults.AuditStoreApplySchema)...)
	opts = append(opts, storeFields("appauth", defaults.AppAuthStoreDriver, defaults.AppAuthStoreDSN, defaults.AppAuthStoreApplySchema)...)
	opts = append(opts, storeFields("capability", defaults.CapabilityStoreDriver, defaults.CapabilityStoreDSN, defaults.CapabilityStoreApplySchema)...)
	opts = append(opts, storeFields("programauth", defaults.ProgramAuthStoreDriver, defaults.ProgramAuthStoreDSN, defaults.ProgramAuthStoreApplySchema)...)
	opts = append(opts, storeFields("oidc-transaction", defaults.OIDCTransactionStoreDriver, defaults.OIDCTransactionStoreDSN, defaults.OIDCTransactionStoreApplySchema)...)
	opts = append(opts, schema.WithFields(
		fields.New("auth-oidc-issuer-url", fields.TypeString, fields.WithDefault(defaults.OIDCIssuerURL), fields.WithHelp("OIDC issuer URL for auth.mode=oidc")),
		fields.New("auth-oidc-client-id", fields.TypeString, fields.WithDefault(defaults.OIDCClientID), fields.WithHelp("OIDC client ID for auth.mode=oidc")),
		fields.New("auth-oidc-client-secret", fields.TypeString, fields.WithDefault(defaults.OIDCClientSecret), fields.WithHelp("OIDC client secret for confidential clients")),
		fields.New("auth-oidc-public-base-url", fields.TypeString, fields.WithDefault(defaults.OIDCPublicBaseURL), fields.WithHelp("External browser-visible HTTPS origin; callback defaults to <public-base-url>/auth/callback")),
		fields.New("auth-oidc-redirect-url", fields.TypeString, fields.WithDefault(defaults.OIDCRedirectURL), fields.WithHelp("Advanced explicit OIDC callback URL override")),
		fields.New("auth-oidc-scopes", fields.TypeStringList, fields.WithDefault(defaults.OIDCScopes), fields.WithHelp("OIDC scopes; openid is added automatically")),
		fields.New("auth-oidc-after-login-url", fields.TypeString, fields.WithDefault(defaults.OIDCAfterLoginURL), fields.WithHelp("Relative URL to redirect to after login")),
		fields.New("auth-oidc-after-logout-url", fields.TypeString, fields.WithDefault(defaults.OIDCAfterLogoutURL), fields.WithHelp("Relative URL to redirect to after logout")),
	))
	return schema.NewSection(SectionSlug, "Generated host auth", opts...)
}

func storeFields(prefix, driver, dsn string, applySchema bool) []schema.SectionOption {
	return []schema.SectionOption{schema.WithFields(
		fields.New("auth-"+prefix+"-store-driver", fields.TypeChoice, fields.WithChoices("", string(StoreDriverMemory), string(StoreDriverSQLite), string(StoreDriverPostgres)), fields.WithDefault(driver), fields.WithHelp("Auth "+prefix+" store driver")),
		fields.New("auth-"+prefix+"-store-dsn", fields.TypeString, fields.WithDefault(dsn), fields.WithHelp("Auth "+prefix+" store DSN for SQL stores")),
		fields.New("auth-"+prefix+"-store-apply-schema", fields.TypeBool, fields.WithDefault(applySchema), fields.WithHelp("Apply the built-in schema for the auth "+prefix+" store")),
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
	programauth := cfg.Stores.ProgramAuth
	oidcTransaction := cfg.Stores.OIDCTransaction
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

		ProgramAuthStoreDriver:      strings.TrimSpace(programauth.Driver),
		ProgramAuthStoreDSN:         strings.TrimSpace(programauth.DSN),
		ProgramAuthStoreApplySchema: boolValue(programauth.ApplySchema),

		OIDCTransactionStoreDriver:      strings.TrimSpace(oidcTransaction.Driver),
		OIDCTransactionStoreDSN:         strings.TrimSpace(oidcTransaction.DSN),
		OIDCTransactionStoreApplySchema: boolValue(oidcTransaction.ApplySchema),

		OIDCIssuerURL:      strings.TrimSpace(cfg.OIDC.IssuerURL),
		OIDCClientID:       strings.TrimSpace(cfg.OIDC.ClientID),
		OIDCClientSecret:   strings.TrimSpace(cfg.OIDC.ClientSecret),
		OIDCPublicBaseURL:  strings.TrimSpace(cfg.OIDC.PublicBaseURL),
		OIDCRedirectURL:    strings.TrimSpace(cfg.OIDC.RedirectURL),
		OIDCScopes:         append([]string(nil), cfg.OIDC.Scopes...),
		OIDCAfterLoginURL:  strings.TrimSpace(cfg.OIDC.AfterLoginURL),
		OIDCAfterLogoutURL: strings.TrimSpace(cfg.OIDC.AfterLogoutURL),
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
			Default:         storeConfigFromGlazed(s.DefaultStoreDriver, s.DefaultStoreDSN, s.DefaultStoreApplySchema),
			Session:         storeConfigFromGlazed(s.SessionStoreDriver, s.SessionStoreDSN, s.SessionStoreApplySchema),
			Audit:           storeConfigFromGlazed(s.AuditStoreDriver, s.AuditStoreDSN, s.AuditStoreApplySchema),
			AppAuth:         storeConfigFromGlazed(s.AppAuthStoreDriver, s.AppAuthStoreDSN, s.AppAuthStoreApplySchema),
			Capability:      storeConfigFromGlazed(s.CapabilityStoreDriver, s.CapabilityStoreDSN, s.CapabilityStoreApplySchema),
			ProgramAuth:     storeConfigFromGlazed(s.ProgramAuthStoreDriver, s.ProgramAuthStoreDSN, s.ProgramAuthStoreApplySchema),
			OIDCTransaction: storeConfigFromGlazed(s.OIDCTransactionStoreDriver, s.OIDCTransactionStoreDSN, s.OIDCTransactionStoreApplySchema),
		},
		OIDC: OIDCConfig{
			IssuerURL:      strings.TrimSpace(s.OIDCIssuerURL),
			ClientID:       strings.TrimSpace(s.OIDCClientID),
			ClientSecret:   strings.TrimSpace(s.OIDCClientSecret),
			PublicBaseURL:  strings.TrimSpace(s.OIDCPublicBaseURL),
			RedirectURL:    strings.TrimSpace(s.OIDCRedirectURL),
			Scopes:         trimStringSlice(s.OIDCScopes),
			AfterLoginURL:  strings.TrimSpace(s.OIDCAfterLoginURL),
			AfterLogoutURL: strings.TrimSpace(s.OIDCAfterLogoutURL),
		},
	}
}

func trimStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func storeConfigFromGlazed(driver, dsn string, applySchema bool) StoreConfig {
	driver = strings.TrimSpace(driver)
	dsn = strings.TrimSpace(dsn)
	if driver == "" && dsn == "" && !applySchema {
		return StoreConfig{}
	}
	return StoreConfig{Driver: driver, DSN: dsn, ApplySchema: boolPtr(applySchema)}
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
