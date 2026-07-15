package hostauth

import (
	"strings"
	"testing"
)

func TestResolveConfigSingleNodePreflightRejectsUnsafeCombinations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		path   string
		want   string
	}{
		{
			name:   "memory store",
			mutate: func(cfg *Config) { cfg.Stores.Default.Driver = "memory" },
			path:   "auth.stores.session.driver",
			want:   "memory storage is not allowed",
		},
		{
			name: "insecure localhost",
			mutate: func(cfg *Config) {
				cfg.Session.Cookie.AllowInsecureHTTP = true
				cfg.OIDC.IssuerURL = "http://localhost:9443"
				cfg.OIDC.PublicBaseURL = "http://localhost:8787"
			},
			path: "auth.session.cookie.allow-insecure-http",
			want: "must be false",
		},
		{
			name: "automatic schema",
			mutate: func(cfg *Config) {
				applySchema := true
				cfg.Stores.Default.ApplySchema = &applySchema
			},
			path: "auth.stores.session.apply-schema",
			want: "run migrations before startup",
		},
		{
			name:   "non oidc mode",
			mutate: func(cfg *Config) { cfg.Mode = ModeDev },
			path:   "auth.mode",
			want:   "must be oidc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validSingleNodeConfig()
			tt.mutate(&cfg)
			_, err := ResolveConfig(cfg, ResolveOptions{})
			if err == nil {
				t.Fatal("expected preflight rejection")
			}
			assertConfigPath(t, err, tt.path)
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err, tt.want)
			}
		})
	}
}

func TestResolveConfigSingleNodePreflightAcceptsDurableHTTPSConfig(t *testing.T) {
	resolved, err := ResolveConfig(validSingleNodeConfig(), ResolveOptions{})
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if resolved.Deployment.Profile != DeploymentProfileSingleNode {
		t.Fatalf("profile = %q", resolved.Deployment.Profile)
	}
	if resolved.RateLimiter.Driver != RateLimiterDriverMemory {
		t.Fatalf("rate limiter = %q", resolved.RateLimiter.Driver)
	}
	for _, store := range resolved.Stores.all() {
		if store.Driver != StoreDriverSQLite || store.ApplySchema {
			t.Fatalf("store %s = %#v", store.Name, store)
		}
	}
}

func TestResolveConfigRejectsUnsupportedDeploymentAndRateLimiter(t *testing.T) {
	for _, tt := range []struct {
		cfg  Config
		path string
	}{
		{cfg: Config{Deployment: DeploymentConfig{Profile: "multi-region"}}, path: "auth.deployment.profile"},
		{cfg: Config{RateLimiter: RateLimiterConfig{Driver: "redis"}}, path: "auth.rate-limiter.driver"},
	} {
		_, err := ResolveConfig(tt.cfg, ResolveOptions{})
		if err == nil {
			t.Fatal("expected config error")
		}
		assertConfigPath(t, err, tt.path)
	}
}

func validSingleNodeConfig() Config {
	return Config{
		Mode:       ModeOIDC,
		Deployment: DeploymentConfig{Profile: DeploymentProfileSingleNode},
		Stores: StoresConfig{Default: StoreConfig{
			Driver: "sqlite",
			DSN:    "file:single-node.db",
		}},
		OIDC: OIDCConfig{
			IssuerURL:     "https://idp.example.test",
			ClientID:      "example-app",
			PublicBaseURL: "https://app.example.test",
		},
	}
}
