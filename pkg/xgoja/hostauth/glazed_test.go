package hostauth

import (
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

func TestGlazedConfigSectionDefaultsFromBaseConfig(t *testing.T) {
	applySchema := true
	section, err := GlazedConfigSection(Config{
		Mode: ModeDev,
		Session: SessionConfig{
			Cookie:      CookieConfig{AllowInsecureHTTP: true, Name: "demo_session", SameSite: "strict", Path: "/app"},
			IdleTimeout: "15m", AbsoluteTimeout: "8h",
		},
		Stores: StoresConfig{Default: StoreConfig{Driver: "sqlite", DSN: "file:auth.db", ApplySchema: &applySchema}},
		OIDC:   OIDCConfig{IssuerURL: "https://auth.example.test/realms/demo", ClientID: "goja-app", PublicBaseURL: "https://app.example.test", Scopes: []string{"profile", "email"}},
	})
	if err != nil {
		t.Fatalf("GlazedConfigSection: %v", err)
	}
	if section.GetSlug() != SectionSlug {
		t.Fatalf("slug = %q", section.GetSlug())
	}
	assertDefault(t, section, "auth-mode", string(ModeDev))
	assertDefault(t, section, "auth-session-cookie-allow-insecure-http", true)
	assertDefault(t, section, "auth-session-cookie-name", "demo_session")
	assertDefault(t, section, "auth-default-store-driver", "sqlite")
	assertDefault(t, section, "auth-default-store-dsn", "file:auth.db")
	assertDefault(t, section, "auth-session-store-apply-schema", false)
	assertDefault(t, section, "auth-programauth-store-apply-schema", false)
	assertDefault(t, section, "auth-oidc-issuer-url", "https://auth.example.test/realms/demo")
	assertDefault(t, section, "auth-oidc-client-id", "goja-app")
	assertDefault(t, section, "auth-oidc-public-base-url", "https://app.example.test")
}

func TestConfigFromValuesMapsAuthSectionToNestedConfig(t *testing.T) {
	section, err := GlazedConfigSection(Config{})
	if err != nil {
		t.Fatalf("GlazedConfigSection: %v", err)
	}
	sectionValues := sectionValuesWithDefaults(t, section, map[string]any{
		"auth-mode": string(ModeDev),
		"auth-session-cookie-allow-insecure-http": true,
		"auth-session-cookie-name":                "app_session",
		"auth-session-cookie-same-site":           "strict",
		"auth-session-cookie-path":                "/app",
		"auth-session-idle-timeout":               "15m",
		"auth-session-absolute-timeout":           "8h",
		"auth-default-store-driver":               "sqlite",
		"auth-default-store-dsn":                  "file:auth.db?mode=memory&cache=shared",
		"auth-default-store-apply-schema":         true,
		"auth-audit-store-driver":                 "memory",
		"auth-audit-store-dsn":                    "",
		"auth-audit-store-apply-schema":           false,
		"auth-programauth-store-driver":           "sqlite",
		"auth-programauth-store-dsn":              "file:programauth.db?mode=memory&cache=shared",
		"auth-programauth-store-apply-schema":     true,
		"auth-oidc-issuer-url":                    "https://auth.example.test/realms/demo",
		"auth-oidc-client-id":                     "goja-app",
		"auth-oidc-client-secret":                 "secret",
		"auth-oidc-public-base-url":               "https://app.example.test",
		"auth-oidc-scopes":                        []string{"profile", "email"},
	})
	cfg, err := ConfigFromValues(values.New(values.WithSectionValues(SectionSlug, sectionValues)), Config{Mode: ModeNone})
	if err != nil {
		t.Fatalf("ConfigFromValues: %v", err)
	}
	if cfg.Mode != ModeDev {
		t.Fatalf("mode = %q", cfg.Mode)
	}
	if !cfg.Session.Cookie.AllowInsecureHTTP || cfg.Session.Cookie.Name != "app_session" || cfg.Session.Cookie.SameSite != "strict" || cfg.Session.Cookie.Path != "/app" {
		t.Fatalf("session = %#v", cfg.Session)
	}
	if cfg.Session.IdleTimeout != "15m" || cfg.Session.AbsoluteTimeout != "8h" {
		t.Fatalf("timeouts = %#v", cfg.Session)
	}
	if cfg.Stores.Default.Driver != "sqlite" || cfg.Stores.Default.DSN == "" || cfg.Stores.Default.ApplySchema == nil || !*cfg.Stores.Default.ApplySchema {
		t.Fatalf("default store = %#v", cfg.Stores.Default)
	}
	if cfg.Stores.Audit.Driver != "memory" || cfg.Stores.Audit.DSN != "" || cfg.Stores.Audit.ApplySchema == nil || *cfg.Stores.Audit.ApplySchema {
		t.Fatalf("audit store = %#v", cfg.Stores.Audit)
	}
	if cfg.Stores.ProgramAuth.Driver != "sqlite" || cfg.Stores.ProgramAuth.DSN != "file:programauth.db?mode=memory&cache=shared" || cfg.Stores.ProgramAuth.ApplySchema == nil || !*cfg.Stores.ProgramAuth.ApplySchema {
		t.Fatalf("programauth store = %#v", cfg.Stores.ProgramAuth)
	}
	if cfg.OIDC.IssuerURL != "https://auth.example.test/realms/demo" || cfg.OIDC.ClientID != "goja-app" || cfg.OIDC.ClientSecret != "secret" || cfg.OIDC.PublicBaseURL != "https://app.example.test" {
		t.Fatalf("oidc = %#v", cfg.OIDC)
	}
	if len(cfg.OIDC.Scopes) != 2 || cfg.OIDC.Scopes[0] != "profile" || cfg.OIDC.Scopes[1] != "email" {
		t.Fatalf("oidc scopes = %#v", cfg.OIDC.Scopes)
	}
}

func TestConfigFromValuesReturnsBaseWhenAuthSectionMissing(t *testing.T) {
	base := Config{Mode: ModeDev, Stores: StoresConfig{Default: StoreConfig{Driver: "memory"}}}
	cfg, err := ConfigFromValues(values.New(), base)
	if err != nil {
		t.Fatalf("ConfigFromValues: %v", err)
	}
	if cfg.Mode != base.Mode || cfg.Stores.Default.Driver != base.Stores.Default.Driver {
		t.Fatalf("cfg = %#v, want base %#v", cfg, base)
	}
}

func TestServiceFactoryUsesParsedGlazedValues(t *testing.T) {
	section, err := GlazedConfigSection(Config{Mode: ModeDev})
	if err != nil {
		t.Fatalf("GlazedConfigSection: %v", err)
	}
	sectionValues := sectionValuesWithDefaults(t, section, map[string]any{
		"auth-mode":                               string(ModeDev),
		"auth-default-store-driver":               "sqlite",
		"auth-default-store-dsn":                  "file:hostauth-glazed?mode=memory&cache=shared",
		"auth-default-store-apply-schema":         true,
		"auth-session-cookie-allow-insecure-http": true,
	})
	services, err := NewServiceFactory(BuilderOptions{Config: Config{Mode: ModeNone}}).BuildHostAuthServices(t.Context(), values.New(values.WithSectionValues(SectionSlug, sectionValues)))
	if err != nil {
		t.Fatalf("BuildHostAuthServices: %v", err)
	}
	defer func() { _ = services.Close(t.Context()) }()
	if services.Config.Mode != ModeDev {
		t.Fatalf("mode = %q", services.Config.Mode)
	}
	if len(services.Closers) != 1 {
		t.Fatalf("closers = %d, want shared sqlite DB closer", len(services.Closers))
	}
}

func assertDefault(t *testing.T, section schema.Section, name string, want any) {
	t.Helper()
	definition, ok := section.GetDefinitions().Get(name)
	if !ok {
		t.Fatalf("missing field %q", name)
	}
	if definition.Default == nil {
		t.Fatalf("field %q missing default", name)
	}
	if got := *definition.Default; got != want {
		t.Fatalf("default %s = %#v, want %#v", name, got, want)
	}
}

func sectionValuesWithDefaults(t *testing.T, section schema.Section, overrides map[string]any) *values.SectionValues {
	t.Helper()
	fieldValues := fields.NewFieldValues()
	for _, definition := range section.GetDefinitions().ToList() {
		if definition.Default != nil {
			fieldValues.Set(definition.Name, &fields.FieldValue{Definition: definition, Value: *definition.Default})
		}
	}
	for name, value := range overrides {
		definition, ok := section.GetDefinitions().Get(name)
		if !ok {
			t.Fatalf("unknown field %q", name)
		}
		fieldValues.Set(name, &fields.FieldValue{Definition: definition, Value: value})
	}
	sectionValues, err := values.NewSectionValues(section, values.WithFields(fieldValues))
	if err != nil {
		t.Fatalf("section values: %v", err)
	}
	return sectionValues
}
