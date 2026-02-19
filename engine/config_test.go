package engine

import "testing"

func TestDefaultRuntimeConfig(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	if cfg != (RuntimeConfig{}) {
		t.Fatal("expected zero-value runtime config")
	}
}
