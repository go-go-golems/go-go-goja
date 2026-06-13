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

func TestV2PlanEmbeddedRuntimeSpecMarksArtifactSourcesEmbedded(t *testing.T) {
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

	runtimeSpec := app.RuntimeSpec{}
	if err := json.Unmarshal([]byte(generate.RenderEmbeddedSpecFromPlan(compiled)), &runtimeSpec); err != nil {
		t.Fatalf("decode runtime spec: %v", err)
	}
	if len(runtimeSpec.JSVerbs) != 1 || !runtimeSpec.JSVerbs[0].Embed || runtimeSpec.JSVerbs[0].Path != "xgoja_embed/jsverbs/verbs" {
		t.Fatalf("expected v2 binary source dependency to embed jsverbs, got %#v", runtimeSpec.JSVerbs)
	}
	if len(runtimeSpec.Help.Sources) != 1 || !runtimeSpec.Help.Sources[0].Embed || runtimeSpec.Help.Sources[0].Path != "xgoja_embed/help/docs" {
		t.Fatalf("expected v2 binary source dependency to embed help, got %#v", runtimeSpec.Help.Sources)
	}
	if len(runtimeSpec.Assets) != 1 || !runtimeSpec.Assets[0].Embed || runtimeSpec.Assets[0].Path != "xgoja_embed/assets/web" {
		t.Fatalf("expected v2 embedded-assets artifact to embed assets, got %#v", runtimeSpec.Assets)
	}
}
