package hostauth

import (
	"fmt"
	"strings"
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
	if cfg.RateLimiter.Driver != RateLimiterDriverMemory {
		return configError("auth.rate-limiter.driver", fmt.Errorf("unsupported driver %q", cfg.RateLimiter.Driver))
	}
	return nil
}

func (c ResolvedStoresConfig) all() []ResolvedStoreConfig {
	return []ResolvedStoreConfig{c.Session, c.Audit, c.AppAuth, c.Capability, c.ProgramAuth, c.OIDCTransaction}
}
