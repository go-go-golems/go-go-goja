package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
)

func TestLoadV2PlanRejectsLegacySpec(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte("name: legacy\n"), 0o644); err != nil {
		t.Fatalf("write legacy spec: %v", err)
	}
	_, err := loadV2Plan(specPath)
	if err == nil || !strings.Contains(err.Error(), "xgoja migrate-spec") {
		t.Fatalf("expected migration diagnostic, got %v", err)
	}
}

func TestV2PlanEmbeddedRuntimePlanMarksArtifactSourcesEmbedded(t *testing.T) {
	compiled := &plan.Plan{Config: specv2.Config{
		Schema: specv2.Schema,
		Name:   "embedded",
		Sources: []specv2.SourceSpec{
			{ID: "verbs", Kind: specv2.SourceKindJSVerbs, From: specv2.SourceFromSpec{Dir: "./verbs"}},
			{ID: "docs", Kind: specv2.SourceKindHelp, From: specv2.SourceFromSpec{Dir: "./docs"}},
			{ID: "web", Kind: specv2.SourceKindAssets, From: specv2.SourceFromSpec{Dir: "./web/dist"}},
		},
		Artifacts: []specv2.ArtifactSpec{
			{ID: "binary", Type: "binary", Output: "dist/embedded", Sources: []string{"verbs", "docs"}},
			{ID: "assets", Type: "embedded-assets", Sources: []string{"web"}},
		},
	}}

	runtimePlan := app.RuntimePlan{}
	if err := json.Unmarshal([]byte(generate.RenderRuntimePlanJSONFromPlan(compiled)), &runtimePlan); err != nil {
		t.Fatalf("decode runtime spec: %v", err)
	}
	assertEmbeddedSource(t, runtimePlan, app.SourceKindJSVerbs, "xgoja_embed/jsverbs/verbs")
	assertEmbeddedSource(t, runtimePlan, app.SourceKindHelp, "xgoja_embed/help/docs")
	assertEmbeddedSource(t, runtimePlan, app.SourceKindAssets, "xgoja_embed/assets/web")
}

func assertEmbeddedSource(t *testing.T, runtimePlan app.RuntimePlan, kind app.SourceKind, path string) {
	t.Helper()
	for _, source := range runtimePlan.Sources {
		if source.Kind == kind && source.Path == path && source.Embed {
			return
		}
	}
	t.Fatalf("expected embedded source kind=%s path=%s, got %#v", kind, path, runtimePlan.Sources)
}
