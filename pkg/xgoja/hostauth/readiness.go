package hostauth

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"
)

// ReadinessReport contains safe topology and live dependency outcomes. It
// intentionally omits DSNs, provider errors, tokens, and secrets.
type ReadinessReport struct {
	Ready       bool                 `json:"ready"`
	Mode        Mode                 `json:"mode"`
	Profile     DeploymentProfile    `json:"profile"`
	RateLimiter RateLimiterDriver    `json:"rateLimiter"`
	Stores      []ReadinessStore     `json:"stores"`
	Components  []ReadinessComponent `json:"components"`
}

type ReadinessStore struct {
	Name   string      `json:"name"`
	Driver StoreDriver `json:"driver"`
}

type ReadinessComponent struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
}

func BuildReadinessReport(cfg ResolvedConfig) ReadinessReport {
	stores := cfg.Stores.all()
	report := ReadinessReport{Ready: true, Mode: cfg.Mode, Profile: cfg.Deployment.Profile, RateLimiter: cfg.RateLimiter.Driver, Stores: make([]ReadinessStore, 0, len(stores))}
	for _, store := range stores {
		report.Stores = append(report.Stores, ReadinessStore{Name: store.Name, Driver: store.Driver})
	}
	return report
}

func readinessHandler(report ReadinessReport, health []DependencyHealth) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		out := report
		out.Components = checkDependencies(ctx, health)
		out.Ready = true
		for _, component := range out.Components {
			if !component.Healthy {
				out.Ready = false
				break
			}
		}
		if !out.Ready {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		writeJSON(w, out)
	})
}

func livenessHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, map[string]bool{"live": true}) })
}

func checkDependencies(ctx context.Context, health []DependencyHealth) []ReadinessComponent {
	out := make([]ReadinessComponent, len(health))
	var wg sync.WaitGroup
	for i, dependency := range health {
		wg.Add(1)
		go func(i int, dependency DependencyHealth) {
			defer wg.Done()
			out[i] = ReadinessComponent{Name: dependency.Name(), Healthy: dependency.CheckHealth(ctx) == nil}
		}(i, dependency)
	}
	wg.Wait()
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
