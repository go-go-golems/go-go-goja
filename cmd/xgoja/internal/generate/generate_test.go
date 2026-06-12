package generate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/workspace"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
)

func TestRenderGoModPlanDeterministic(t *testing.T) {
	compiled := fixturePlan(t)
	compiled.Config.Providers = append(compiled.Config.Providers, specv2.ProviderSpec{
		ID: "web", Import: "github.com/go-go-golems/web-stuff/xgoja", Register: "Register", Module: specv2.ProviderModuleSpec{Version: "v0.3.0", Replace: "../web-stuff"},
	})
	compiled.Config.Go.Imports = []specv2.GoImportSpec{{Import: "github.com/lib/pq", Alias: "_", Version: "v1.10.9"}}
	got := RenderGoModPlan(compiled, Options{XGojaModuleVersion: "v0.1.0", XGojaReplace: "../go-go-goja"})
	for _, want := range []string{
		"module xgoja.generated/fixture",
		"github.com/go-go-golems/go-go-goja v0.1.0",
		"github.com/go-go-golems/web-stuff v0.3.0",
		"github.com/lib/pq v1.10.9",
		"replace github.com/go-go-golems/go-go-goja => ../go-go-goja",
		"replace github.com/go-go-golems/web-stuff => ../web-stuff",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("go.mod missing %q:\n%s", want, got)
		}
	}
}

func TestRenderGoModPlanUsesWorkspaceModulePlan(t *testing.T) {
	compiled := fixturePlan(t)
	compiled.GoModules = &workspace.Plan{Modules: []workspace.GoModulePlan{{ModulePath: "github.com/example/local", Version: "", LocalDir: "/tmp/local"}}}
	got := RenderGoModPlan(compiled, Options{XGojaModuleVersion: "v0.1.0", GoModules: compiled.GoModules})
	if !strings.Contains(got, "github.com/example/local v0.0.0") {
		t.Fatalf("expected local module require:\n%s", got)
	}
	if !strings.Contains(got, "replace github.com/example/local => /tmp/local") {
		t.Fatalf("expected local module replace:\n%s", got)
	}
}

func TestRenderEmbeddedSpecFromPlanUsesRuntimeShapeAndEmbeddedRoots(t *testing.T) {
	compiled := fixturePlan(t)
	compiled.Config.Sources = []specv2.SourceSpec{
		{ID: "verbs", Kind: specv2.SourceKindJSVerbs, From: specv2.SourceFromSpec{Dir: "verbs"}, Language: "typescript", Compile: &specv2.CompileSpec{Bundle: true}},
		{ID: "docs", Kind: specv2.SourceKindHelp, From: specv2.SourceFromSpec{Dir: "docs/help"}},
		{ID: "assets", Kind: specv2.SourceKindAssets, From: specv2.SourceFromSpec{Dir: "assets"}},
	}
	compiled.Config.Artifacts = []specv2.ArtifactSpec{
		{ID: "binary", Type: "binary", Output: "dist/fixture", Sources: []string{"verbs", "docs"}},
		{ID: "assets", Type: "embedded-assets", Sources: []string{"assets"}},
	}
	var runtimeSpec app.RuntimeSpec
	if err := json.Unmarshal([]byte(RenderEmbeddedSpecFromPlan(compiled)), &runtimeSpec); err != nil {
		t.Fatalf("decode embedded runtime spec: %v", err)
	}
	if len(runtimeSpec.Packages) != 1 || runtimeSpec.Packages[0].ID != "fixture" {
		t.Fatalf("expected runtime provider ids only, got %#v", runtimeSpec.Packages)
	}
	if runtimeSpec.JSVerbs[0].Path != "xgoja_embed/jsverbs/verbs" || !runtimeSpec.JSVerbs[0].Embed {
		t.Fatalf("unexpected jsverb runtime source: %#v", runtimeSpec.JSVerbs[0])
	}
	if runtimeSpec.Help.Sources[0].Path != "xgoja_embed/help/docs" || !runtimeSpec.Help.Sources[0].Embed {
		t.Fatalf("unexpected help runtime source: %#v", runtimeSpec.Help.Sources[0])
	}
	if runtimeSpec.Assets[0].Path != "xgoja_embed/assets/assets" || !runtimeSpec.Assets[0].Embed {
		t.Fatalf("unexpected asset runtime source: %#v", runtimeSpec.Assets[0])
	}
}

func TestWriteAllPlanCopiesEmbeddedSources(t *testing.T) {
	base := t.TempDir()
	mustWrite(t, filepath.Join(base, "verbs", "hello.js"), "export const x = 1;\n")
	mustWrite(t, filepath.Join(base, "docs", "help", "index.md"), "# Help\n")
	mustWrite(t, filepath.Join(base, "assets", "app.css"), "body{}\n")
	mustWrite(t, filepath.Join(base, "assets", "node_modules", "skip.js"), "skip\n")

	compiled := fixturePlan(t)
	compiled.Config.BaseDir = base
	compiled.Config.Sources = []specv2.SourceSpec{
		{ID: "verbs", Kind: specv2.SourceKindJSVerbs, From: specv2.SourceFromSpec{Dir: "verbs"}},
		{ID: "docs", Kind: specv2.SourceKindHelp, From: specv2.SourceFromSpec{Dir: "docs/help"}},
		{ID: "assets", Kind: specv2.SourceKindAssets, From: specv2.SourceFromSpec{Dir: "assets"}},
	}
	compiled.Config.Artifacts = []specv2.ArtifactSpec{
		{ID: "binary", Type: "binary", Output: "dist/fixture", Sources: []string{"verbs", "docs"}},
		{ID: "assets", Type: "embedded-assets", Sources: []string{"assets"}},
	}
	out := t.TempDir()
	if err := WriteAllPlan(out, compiled, Options{XGojaModuleVersion: "v0.1.0"}); err != nil {
		t.Fatalf("WriteAllPlan: %v", err)
	}
	for _, path := range []string{
		"xgoja_embed/jsverbs/verbs/hello.js",
		"xgoja_embed/help/docs/index.md",
		"xgoja_embed/assets/assets/app.css",
		"go.mod",
		"main.go",
		"xgoja.gen.json",
	} {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(path))); err != nil {
			t.Fatalf("expected generated %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "xgoja_embed", "assets", "assets", "node_modules", "skip.js")); !os.IsNotExist(err) {
		t.Fatalf("expected node_modules asset to be skipped, got err=%v", err)
	}
}

func TestTemplateDataJSONFromPlan(t *testing.T) {
	got, err := TemplateDataJSONFromPlan(fixturePlan(t), "xgojaruntime")
	if err != nil {
		t.Fatalf("TemplateDataJSONFromPlan: %v", err)
	}
	for _, want := range []string{"\"PackageName\": \"xgojaruntime\"", "\"ProviderImports\""} {
		if !strings.Contains(got, want) {
			t.Fatalf("template data missing %q:\n%s", want, got)
		}
	}
}

func fixturePlan(t *testing.T) *plan.Plan {
	t.Helper()
	return &plan.Plan{Config: specv2.Config{
		Schema: specv2.Schema,
		Name:   "fixture",
		Go:     specv2.GoSpec{Version: "1.26", Module: "xgoja.generated/fixture"},
		Providers: []specv2.ProviderSpec{{
			ID: "fixture", Import: "github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider", Register: "Register",
		}},
		Runtime:   specv2.RuntimeSpec{Modules: []specv2.RuntimeModuleSpec{{Provider: "fixture", Name: "hello", As: "hello"}}},
		Commands:  []specv2.CommandSurfaceSpec{{ID: "eval", Type: "builtin.eval", Name: "eval"}},
		Artifacts: []specv2.ArtifactSpec{{ID: "binary", Type: "binary", Output: "dist/fixture"}},
	}}
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
