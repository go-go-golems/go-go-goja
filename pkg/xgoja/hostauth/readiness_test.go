package hostauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeHealth struct {
	name string
	err  error
}

func (h fakeHealth) Name() string                      { return h.name }
func (h fakeHealth) CheckHealth(context.Context) error { return h.err }

type toggleHealth struct{ err error }

func (h *toggleHealth) Name() string                      { return "sql" }
func (h *toggleHealth) CheckHealth(context.Context) error { return h.err }

func TestReadinessChecksDependenciesAndKeepsLivenessSeparate(t *testing.T) {
	report := BuildReadinessReport(ResolvedConfig{})
	ready := readinessHandler(report, []DependencyHealth{fakeHealth{name: "sql", err: errors.New("down")}})
	recorder := httptest.NewRecorder()
	ready.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/auth/readyz", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready status=%d", recorder.Code)
	}
	live := httptest.NewRecorder()
	livenessHandler().ServeHTTP(live, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if live.Code != http.StatusOK {
		t.Fatalf("live status=%d", live.Code)
	}
}

func TestReadinessRecoversWithoutRestart(t *testing.T) {
	health := &toggleHealth{err: errors.New("down")}
	handler := readinessHandler(BuildReadinessReport(ResolvedConfig{}), []DependencyHealth{health})
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/auth/readyz", nil))
	if first.Code != http.StatusServiceUnavailable {
		t.Fatalf("outage status=%d", first.Code)
	}
	health.err = nil
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/auth/readyz", nil))
	if second.Code != http.StatusOK {
		t.Fatalf("recovery status=%d", second.Code)
	}
}
