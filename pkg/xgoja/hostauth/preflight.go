package hostauth

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func resolveDeploymentProfile(cfg DeploymentConfig) (ResolvedDeploymentConfig, error) {
	switch profile := DeploymentProfile(strings.ToLower(strings.TrimSpace(string(cfg.Profile)))); profile {
	case "", DeploymentProfileDevelopment:
		return ResolvedDeploymentConfig{Profile: DeploymentProfileDevelopment}, nil
	case DeploymentProfileSingleNode:
		return ResolvedDeploymentConfig{Profile: profile}, nil
	default:
		return ResolvedDeploymentConfig{}, fmt.Errorf("unsupported deployment profile %q", cfg.Profile)
	}
}

func resolveDeviceConfig(cfg DeviceConfig) (ResolvedDeviceConfig, error) {
	if cfg.MaxActions < 0 {
		return ResolvedDeviceConfig{}, fmt.Errorf("max-actions must not be negative")
	}
	actions := map[string]struct{}{}
	for _, raw := range cfg.AllowedActions {
		action := strings.TrimSpace(raw)
		if action == "" {
			return ResolvedDeviceConfig{}, fmt.Errorf("allowed-actions contains an empty action")
		}
		if _, ok := actions[action]; ok {
			return ResolvedDeviceConfig{}, fmt.Errorf("duplicate allowed action %q", action)
		}
		actions[action] = struct{}{}
	}
	return ResolvedDeviceConfig{AllowedActions: actions, MaxActions: cfg.MaxActions, VerificationURI: strings.TrimSpace(cfg.VerificationURI)}, nil
}

func resolveProxyConfig(cfg ProxyConfig) (ResolvedProxyConfig, error) {
	mode := gojahttp.ProxyMode(strings.ToLower(strings.TrimSpace(string(cfg.Mode))))
	switch mode {
	case "", gojahttp.ProxyModeDirect:
		if len(cfg.TrustedCIDRs) != 0 {
			return ResolvedProxyConfig{}, fmt.Errorf("trusted-cidrs requires mode=trusted-forwarded")
		}
		return ResolvedProxyConfig{Mode: gojahttp.ProxyModeDirect}, nil
	case gojahttp.ProxyModeTrustedForwarded:
		if len(cfg.TrustedCIDRs) == 0 {
			return ResolvedProxyConfig{}, fmt.Errorf("trusted-forwarded requires at least one trusted CIDR")
		}
		prefixes := make([]netip.Prefix, 0, len(cfg.TrustedCIDRs))
		seen := map[netip.Prefix]struct{}{}
		for _, raw := range cfg.TrustedCIDRs {
			prefix, err := netip.ParsePrefix(strings.TrimSpace(raw))
			if err != nil {
				return ResolvedProxyConfig{}, fmt.Errorf("invalid trusted CIDR %q: %w", raw, err)
			}
			prefix = prefix.Masked()
			if _, ok := seen[prefix]; ok {
				return ResolvedProxyConfig{}, fmt.Errorf("duplicate trusted CIDR %q", raw)
			}
			seen[prefix] = struct{}{}
			prefixes = append(prefixes, prefix)
		}
		return ResolvedProxyConfig{Mode: mode, TrustedPrefixes: prefixes}, nil
	default:
		return ResolvedProxyConfig{}, fmt.Errorf("unsupported proxy mode %q", cfg.Mode)
	}
}

func resolveRateLimiterConfig(cfg RateLimiterConfig) (ResolvedRateLimiterConfig, error) {
	switch driver := RateLimiterDriver(strings.ToLower(strings.TrimSpace(string(cfg.Driver)))); driver {
	case "", RateLimiterDriverMemory:
		return ResolvedRateLimiterConfig{Driver: RateLimiterDriverMemory}, nil
	default:
		return ResolvedRateLimiterConfig{}, fmt.Errorf("unsupported rate limiter driver %q", cfg.Driver)
	}
}

// validateDeploymentPreflight rejects configurations which are convenient for
// a tutorial but would silently weaken the declared single-node production
// topology. The resulting profile intentionally permits the local memory rate
// limiter only when exactly one serving process is operated.
func validateDeploymentPreflight(cfg ResolvedConfig) error {
	if cfg.Deployment.Profile != DeploymentProfileSingleNode {
		return nil
	}
	if cfg.Mode != ModeOIDC {
		return configError("auth.mode", fmt.Errorf("must be oidc for deployment.profile=single-node"))
	}
	if cfg.Session.Cookie.AllowInsecureHTTP {
		return configError("auth.session.cookie.allow-insecure-http", fmt.Errorf("must be false for deployment.profile=single-node"))
	}
	for _, store := range cfg.Stores.all() {
		path := "auth.stores." + store.Name
		if store.Driver == StoreDriverMemory {
			return configError(path+".driver", fmt.Errorf("memory storage is not allowed for deployment.profile=single-node"))
		}
		if store.ApplySchema {
			return configError(path+".apply-schema", fmt.Errorf("must be false for deployment.profile=single-node; run migrations before startup"))
		}
	}
	if len(cfg.Device.AllowedActions) == 0 {
		return configError("auth.device.allowed-actions", fmt.Errorf("must be non-empty for deployment.profile=single-node"))
	}
	if cfg.Device.VerificationURI == "" {
		return configError("auth.device.verification-uri", fmt.Errorf("must be configured for deployment.profile=single-node"))
	}
	if cfg.RateLimiter.Driver != RateLimiterDriverMemory {
		return configError("auth.rate-limiter.driver", fmt.Errorf("unsupported driver %q", cfg.RateLimiter.Driver))
	}
	return nil
}

func (c ResolvedStoresConfig) all() []ResolvedStoreConfig {
	return []ResolvedStoreConfig{c.Session, c.Audit, c.AppAuth, c.Capability, c.ProgramAuth, c.OIDCTransaction}
}
