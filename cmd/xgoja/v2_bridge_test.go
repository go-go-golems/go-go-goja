package main

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
)

func TestBuildSpecFromV2PlanMarksArtifactSourcesEmbedded(t *testing.T) {
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

	buildSpec := buildSpecFromV2Plan(compiled)
	if len(buildSpec.JSVerbs) != 1 || !buildSpec.JSVerbs[0].Embed {
		t.Fatalf("expected v2 binary source dependency to embed jsverbs, got %#v", buildSpec.JSVerbs)
	}
	if len(buildSpec.Help.Sources) != 1 || !buildSpec.Help.Sources[0].Embed {
		t.Fatalf("expected v2 binary source dependency to embed help, got %#v", buildSpec.Help.Sources)
	}
	if len(buildSpec.Assets) != 1 || !buildSpec.Assets[0].Embed {
		t.Fatalf("expected v2 embedded-assets artifact to embed assets, got %#v", buildSpec.Assets)
	}
}
