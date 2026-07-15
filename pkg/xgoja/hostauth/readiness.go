package hostauth

import "net/http"

// ReadinessReport is a safe, machine-readable declaration of the resolved
// auth topology. It intentionally omits DSNs, client secrets, tokens, cookie
// values, and other credential material. It is a configuration/readiness
// assertion, not a replacement for a database liveness probe.
type ReadinessReport struct {
	Ready       bool              `json:"ready"`
	Mode        Mode              `json:"mode"`
	Profile     DeploymentProfile `json:"profile"`
	RateLimiter RateLimiterDriver `json:"rateLimiter"`
	Stores      []ReadinessStore  `json:"stores"`
}

type ReadinessStore struct {
	Name   string      `json:"name"`
	Driver StoreDriver `json:"driver"`
}

func BuildReadinessReport(cfg ResolvedConfig) ReadinessReport {
	stores := cfg.Stores.all()
	report := ReadinessReport{
		Ready:       true,
		Mode:        cfg.Mode,
		Profile:     cfg.Deployment.Profile,
		RateLimiter: cfg.RateLimiter.Driver,
		Stores:      make([]ReadinessStore, 0, len(stores)),
	}
	for _, store := range stores {
		report.Stores = append(report.Stores, ReadinessStore{Name: store.Name, Driver: store.Driver})
	}
	return report
}

func readinessHandler(report ReadinessReport) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, report)
	})
}
