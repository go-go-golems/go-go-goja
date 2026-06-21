package hostauth

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestResolveConfigDefaultsToNoAuthMemoryStoresAndSecureCookie(t *testing.T) {
	resolved, err := ResolveConfig(Config{}, ResolveOptions{})
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if resolved.Mode != ModeNone {
		t.Fatalf("mode = %q", resolved.Mode)
	}
	if resolved.Session.Cookie.AllowInsecureHTTP {
		t.Fatalf("AllowInsecureHTTP defaulted to true")
	}
	if resolved.Session.Cookie.Path != "/" {
		t.Fatalf("cookie path = %q", resolved.Session.Cookie.Path)
	}
	if resolved.Session.Cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("same-site = %v", resolved.Session.Cookie.SameSite)
	}
	if resolved.Session.IdleTimeout != 0 || resolved.Session.AbsoluteTimeout != 0 {
		t.Fatalf("timeouts = %s %s", resolved.Session.IdleTimeout, resolved.Session.AbsoluteTimeout)
	}
	for name, store := range map[string]ResolvedStoreConfig{
		"session":     resolved.Stores.Session,
		"audit":       resolved.Stores.Audit,
		"appauth":     resolved.Stores.AppAuth,
		"capability":  resolved.Stores.Capability,
		"programauth": resolved.Stores.ProgramAuth,
	} {
		if store.Name != name {
			t.Fatalf("store %s name = %q", name, store.Name)
		}
		if store.Driver != StoreDriverMemory {
			t.Fatalf("store %s driver = %q", name, store.Driver)
		}
		if store.DSN != "" {
			t.Fatalf("store %s DSN = %q", name, store.DSN)
		}
	}
}

func TestResolveConfigModeNoneIgnoresStoreDSNRequirements(t *testing.T) {
	resolved, err := ResolveConfig(Config{Mode: ModeNone, Stores: StoresConfig{Default: StoreConfig{Driver: "postgres"}}}, ResolveOptions{})
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if resolved.Mode != ModeNone || resolved.Stores.Session.Driver != StoreDriverMemory {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestResolveConfigParsesSessionFields(t *testing.T) {
	resolved, err := ResolveConfig(Config{
		Mode: ModeDev,
		Session: SessionConfig{
			Cookie: CookieConfig{
				AllowInsecureHTTP: true,
				Name:              " app_session ",
				SameSite:          "strict",
				Path:              "/app",
			},
			IdleTimeout:     "15m",
			AbsoluteTimeout: "8h",
		},
	}, ResolveOptions{})
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if resolved.Mode != ModeDev {
		t.Fatalf("mode = %q", resolved.Mode)
	}
	cookie := resolved.Session.Cookie
	if !cookie.AllowInsecureHTTP || cookie.Name != "app_session" || cookie.Path != "/app" || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie = %#v", cookie)
	}
	if resolved.Session.IdleTimeout != 15*time.Minute || resolved.Session.AbsoluteTimeout != 8*time.Hour {
		t.Fatalf("timeouts = %s %s", resolved.Session.IdleTimeout, resolved.Session.AbsoluteTimeout)
	}
}

func TestResolveConfigStoreInheritanceAndDSN(t *testing.T) {
	applyDefault := false
	applyAudit := true
	resolved, err := ResolveConfig(Config{Mode: ModeDev, Stores: StoresConfig{
		Default: StoreConfig{Driver: "postgres", DSN: " postgres://example/app ", ApplySchema: &applyDefault},
		Audit:   StoreConfig{ApplySchema: &applyAudit},
		Session: StoreConfig{Driver: "sqlite", DSN: "file:sessions.db"},
	}}, ResolveOptions{})
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if resolved.Stores.Session.Driver != StoreDriverSQLite || resolved.Stores.Session.DSN != "file:sessions.db" || resolved.Stores.Session.ApplySchema {
		t.Fatalf("session store = %#v", resolved.Stores.Session)
	}
	if resolved.Stores.Audit.Driver != StoreDriverPostgres || resolved.Stores.Audit.DSN != "postgres://example/app" || !resolved.Stores.Audit.ApplySchema {
		t.Fatalf("audit store = %#v", resolved.Stores.Audit)
	}
	if resolved.Stores.AppAuth.Driver != StoreDriverPostgres || resolved.Stores.AppAuth.DSN != "postgres://example/app" || resolved.Stores.AppAuth.ApplySchema {
		t.Fatalf("appauth store = %#v", resolved.Stores.AppAuth)
	}
	if resolved.Stores.Capability.Driver != StoreDriverPostgres || resolved.Stores.Capability.DSN != "postgres://example/app" || resolved.Stores.Capability.ApplySchema {
		t.Fatalf("capability store = %#v", resolved.Stores.Capability)
	}
	if resolved.Stores.ProgramAuth.Driver != StoreDriverPostgres || resolved.Stores.ProgramAuth.DSN != "postgres://example/app" || resolved.Stores.ProgramAuth.ApplySchema {
		t.Fatalf("programauth store = %#v", resolved.Stores.ProgramAuth)
	}
}

func TestResolveConfigExplicitMemoryStoreIgnoresInheritedDSN(t *testing.T) {
	resolved, err := ResolveConfig(Config{Mode: ModeDev, Stores: StoresConfig{
		Default: StoreConfig{Driver: "postgres", DSN: "postgres://example/app"},
		Audit:   StoreConfig{Driver: "memory"},
	}}, ResolveOptions{})
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if resolved.Stores.Audit.Driver != StoreDriverMemory || resolved.Stores.Audit.DSN != "" {
		t.Fatalf("audit store = %#v", resolved.Stores.Audit)
	}
}

func TestResolveConfigOIDCDerivesRedirectFromPublicBaseURL(t *testing.T) {
	resolved, err := ResolveConfig(Config{
		Mode:    ModeOIDC,
		Session: SessionConfig{Cookie: CookieConfig{AllowInsecureHTTP: true}},
		OIDC: OIDCConfig{
			IssuerURL:      "http://localhost:8080/realms/demo",
			ClientID:       "goja-app",
			PublicBaseURL:  "http://localhost:8787/",
			Scopes:         []string{"profile", "email"},
			AfterLoginURL:  "/dashboard",
			AfterLogoutURL: "/logged-out",
		},
	}, ResolveOptions{})
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if resolved.Mode != ModeOIDC || resolved.OIDC.RedirectURL != "http://localhost:8787/auth/callback" {
		t.Fatalf("resolved OIDC = %#v", resolved.OIDC)
	}
	if resolved.OIDC.AfterLoginURL != "/dashboard" || resolved.OIDC.AfterLogoutURL != "/logged-out" {
		t.Fatalf("after URLs = %#v", resolved.OIDC)
	}
	if len(resolved.OIDC.Scopes) != 2 || resolved.OIDC.Scopes[0] != "profile" || resolved.OIDC.Scopes[1] != "email" {
		t.Fatalf("scopes = %#v", resolved.OIDC.Scopes)
	}
}

func TestResolveConfigOIDCRedirectOverrideAndHTTPSPolicy(t *testing.T) {
	resolved, err := ResolveConfig(Config{
		Mode: ModeOIDC,
		OIDC: OIDCConfig{
			IssuerURL:   "https://auth.example.test/realms/demo",
			ClientID:    "goja-app",
			RedirectURL: "https://app.example.test/custom/callback",
		},
	}, ResolveOptions{})
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if resolved.OIDC.RedirectURL != "https://app.example.test/custom/callback" {
		t.Fatalf("redirect = %q", resolved.OIDC.RedirectURL)
	}

	_, err = ResolveConfig(Config{
		Mode: ModeOIDC,
		OIDC: OIDCConfig{IssuerURL: "https://auth.example.test/realms/demo", ClientID: "goja-app", PublicBaseURL: "http://app.example.test"},
	}, ResolveOptions{})
	if err == nil {
		t.Fatal("expected non-local http public base URL rejection")
	}
	assertConfigPath(t, err, "auth.oidc.public-base-url")
}

func TestResolveConfigRejectsInvalidValuesWithPaths(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		path string
		want string
	}{
		{name: "mode", cfg: Config{Mode: "magic"}, path: "auth.mode", want: "unsupported auth mode"},
		{name: "same-site", cfg: Config{Session: SessionConfig{Cookie: CookieConfig{SameSite: "weird"}}}, path: "auth.session.cookie.same-site", want: "unsupported SameSite"},
		{name: "path", cfg: Config{Session: SessionConfig{Cookie: CookieConfig{Path: "app"}}}, path: "auth.session.cookie.path", want: "must start with /"},
		{name: "idle", cfg: Config{Session: SessionConfig{IdleTimeout: "0s"}}, path: "auth.session.idle-timeout", want: "must be positive"},
		{name: "driver", cfg: Config{Mode: ModeDev, Stores: StoresConfig{Default: StoreConfig{Driver: "mysql"}}}, path: "auth.stores.session.driver", want: "unsupported store driver"},
		{name: "missing dsn", cfg: Config{Mode: ModeDev, Stores: StoresConfig{Default: StoreConfig{Driver: "postgres"}}}, path: "auth.stores.session.dsn", want: "dsn is required"},
		{name: "oidc issuer", cfg: Config{Mode: ModeOIDC, OIDC: OIDCConfig{ClientID: "goja-app", PublicBaseURL: "https://app.example.test"}}, path: "auth.oidc.issuer-url", want: "is required"},
		{name: "oidc client", cfg: Config{Mode: ModeOIDC, OIDC: OIDCConfig{IssuerURL: "https://auth.example.test/realms/demo", PublicBaseURL: "https://app.example.test"}}, path: "auth.oidc.client-id", want: "is required"},
		{name: "oidc callback", cfg: Config{Mode: ModeOIDC, OIDC: OIDCConfig{IssuerURL: "https://auth.example.test/realms/demo", ClientID: "goja-app"}}, path: "auth.oidc.public-base-url", want: "public-base-url or redirect-url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveConfig(tt.cfg, ResolveOptions{})
			if err == nil {
				t.Fatal("expected error")
			}
			assertConfigPath(t, err, tt.path)
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err, tt.want)
			}
		})
	}
}

func assertConfigPath(t *testing.T, err error, path string) {
	t.Helper()
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("error %T is not ConfigError: %v", err, err)
	}
	if cfgErr.Path != path {
		t.Fatalf("path = %q, want %q (err=%v)", cfgErr.Path, path, err)
	}
}
