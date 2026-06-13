package app

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRuntimePlanRejectsRemovedLegacyKeys(t *testing.T) {
	for _, key := range []string{"appName", "envPrefix", "configFile", "packages", "modules", "commandProviders", "jsverbs", "help", "assets"} {
		t.Run(key, func(t *testing.T) {
			payload := `{"schema":"xgoja/runtime/v2","name":"fixture",` + jsonString(key) + `:null}`
			var plan RuntimePlan
			err := json.Unmarshal([]byte(payload), &plan)
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("expected removed-key error for %q, got %v", key, err)
			}
		})
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
