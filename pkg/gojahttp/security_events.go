package gojahttp

import (
	"context"
	"sync"
)

// SecurityEvent is a low-cardinality metric observation. It deliberately has
// no free-form attributes: credentials and protocol values belong neither in
// metric labels nor in operational logs.
type SecurityEvent struct {
	Name    string
	Outcome string
	Reason  string
}

// SecurityEventObserver receives security lifecycle observations. Production
// hosts may bridge this to their metrics system; MemorySecurityMetrics keeps
// the contract testable without imposing a metrics dependency.
type SecurityEventObserver interface {
	ObserveSecurityEvent(ctx context.Context, event SecurityEvent)
}

type SecurityMetricKey struct {
	Name    string
	Outcome string
	Reason  string
}

// MemorySecurityMetrics is a concurrency-safe counter implementation for
// tests, demos, and host integration checks.
type MemorySecurityMetrics struct {
	mu     sync.Mutex
	counts map[SecurityMetricKey]uint64
}

func (m *MemorySecurityMetrics) ObserveSecurityEvent(_ context.Context, event SecurityEvent) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.counts == nil {
		m.counts = map[SecurityMetricKey]uint64{}
	}
	m.counts[SecurityMetricKey(event)]++
}

func (m *MemorySecurityMetrics) Count(event SecurityEvent) uint64 {
	if m == nil {
		return 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counts[SecurityMetricKey(event)]
}
