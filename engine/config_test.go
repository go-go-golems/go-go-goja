package engine

import "testing"

func TestDefaultRuntimeConfig_DisablesCallLog(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	if cfg.CallLogEnabled {
		t.Fatal("expected call logging to be disabled by default")
	}
}
