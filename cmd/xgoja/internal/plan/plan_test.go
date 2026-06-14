package plan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/workspace"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestCompileBuildsProviderSourceAndWorkspacePlans(t *testing.T) {
	dir := t.TempDir()
	verbsDir := filepath.Join(dir, "verbs")
	if err := os.MkdirAll(verbsDir, 0o755); err != nil {
		t.Fatalf("mkdir verbs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(verbsDir, "site.js"), []byte(`const hello = require("hello")`), 0o644); err != nil {
		t.Fatalf("write site: %v", err)
	}
	registry := providerRegistry(t)

	p, err := Compile(Options{Config: specv2.Config{
		Schema:    specv2.Schema,
		Name:      "fixture",
		BaseDir:   dir,
		Go:        specv2.GoSpec{Module: "example.com/fixture", Version: "1.26"},
		Workspace: specv2.WorkspaceSpec{Mode: string(workspace.ModeOff)},
		Providers: []specv2.ProviderSpec{{
			ID:       "fixture",
			Import:   "example.com/fixture/provider",
			Register: "Register",
			Module:   specv2.ProviderModuleSpec{Version: "v1.2.3"},
		}},
		Runtime: specv2.RuntimeSpec{Modules: []specv2.RuntimeModuleSpec{{Provider: "fixture", Name: "hello", As: "hello"}}},
		Sources: []specv2.SourceSpec{{
			ID:         "verbs",
			Kind:       specv2.SourceKindJSVerbs,
			From:       specv2.SourceFromSpec{Dir: "verbs"},
			Extensions: []string{".js"},
		}},
		Commands:  []specv2.CommandSurfaceSpec{{ID: "verbs", Type: "builtin.jsverbs", Sources: []string{"verbs"}}},
		Artifacts: []specv2.ArtifactSpec{{ID: "bin", Type: "binary", Output: "dist/fixture", Sources: []string{"verbs"}}},
	}, Providers: registry, StartDir: dir})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if got := p.ProviderGraph.RuntimeModuleAliases(); len(got) != 1 || got[0] != "hello" {
		t.Fatalf("runtime aliases = %#v", got)
	}
	if len(p.SourceGraph.FilesForSourceSet("verbs")) != 1 {
		t.Fatalf("source files = %#v", p.SourceGraph.FilesForSourceSet("verbs"))
	}
	if len(p.GoModules.Modules) != 1 || p.GoModules.Modules[0].ModulePath != "example.com/fixture/provider" || p.GoModules.Modules[0].Version != "v1.2.3" {
		t.Fatalf("go module plan = %#v", p.GoModules.Modules)
	}
	if len(p.Commands) != 1 || len(p.Artifacts) != 1 {
		t.Fatalf("commands/artifacts = %#v %#v", p.Commands, p.Artifacts)
	}
}

func TestCompileAllowsColonRuntimeAliasImports(t *testing.T) {
	dir := t.TempDir()
	verbsDir := filepath.Join(dir, "verbs")
	if err := os.MkdirAll(verbsDir, 0o755); err != nil {
		t.Fatalf("mkdir verbs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(verbsDir, "site.js"), []byte(`const assets = require("fs:assets")`), 0o644); err != nil {
		t.Fatalf("write site: %v", err)
	}
	p, err := Compile(Options{Config: specv2.Config{
		Schema:    specv2.Schema,
		Name:      "fixture",
		BaseDir:   dir,
		Go:        specv2.GoSpec{Module: "example.com/fixture", Version: "1.26"},
		Workspace: specv2.WorkspaceSpec{Mode: string(workspace.ModeOff)},
		Providers: []specv2.ProviderSpec{{
			ID:       "fixture",
			Import:   "example.com/fixture/provider",
			Register: "Register",
			Module:   specv2.ProviderModuleSpec{Version: "v1.2.3"},
		}},
		Runtime: specv2.RuntimeSpec{Modules: []specv2.RuntimeModuleSpec{{Provider: "fixture", Name: "hello", As: "fs:assets"}}},
		Sources: []specv2.SourceSpec{{
			ID:         "verbs",
			Kind:       specv2.SourceKindJSVerbs,
			From:       specv2.SourceFromSpec{Dir: "verbs"},
			Extensions: []string{".js"},
		}},
		Commands:  []specv2.CommandSurfaceSpec{{ID: "verbs", Type: "builtin.jsverbs", Sources: []string{"verbs"}}},
		Artifacts: []specv2.ArtifactSpec{{ID: "bin", Type: "binary", Output: "dist/fixture", Sources: []string{"verbs"}}},
	}, Providers: providerRegistry(t), StartDir: dir})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if got := p.RuntimeAliases; len(got) != 1 || got[0] != "fs:assets" {
		t.Fatalf("runtime aliases = %#v, want fs:assets", got)
	}
}

func TestCompileRejectsUnknownBareImport(t *testing.T) {
	dir := t.TempDir()
	verbsDir := filepath.Join(dir, "verbs")
	if err := os.MkdirAll(verbsDir, 0o755); err != nil {
		t.Fatalf("mkdir verbs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(verbsDir, "site.js"), []byte(`const leftpad = require("left-pad")`), 0o644); err != nil {
		t.Fatalf("write site: %v", err)
	}
	_, err := Compile(Options{Config: specv2.Config{
		Schema:    specv2.Schema,
		Name:      "fixture",
		BaseDir:   dir,
		Go:        specv2.GoSpec{Module: "example.com/fixture", Version: "1.26"},
		Workspace: specv2.WorkspaceSpec{Mode: string(workspace.ModeOff)},
		Sources: []specv2.SourceSpec{{
			ID:         "verbs",
			Kind:       specv2.SourceKindJSVerbs,
			From:       specv2.SourceFromSpec{Dir: "verbs"},
			Extensions: []string{".js"},
		}},
		Commands:  []specv2.CommandSurfaceSpec{{ID: "verbs", Type: "builtin.jsverbs", Sources: []string{"verbs"}}},
		Artifacts: []specv2.ArtifactSpec{{ID: "bin", Type: "binary", Output: "dist/fixture", Sources: []string{"verbs"}}},
	}, Providers: providerapi.NewProviderRegistry(), StartDir: dir})
	if err == nil {
		t.Fatalf("expected unknown import error")
	}
}

func providerRegistry(t *testing.T) *providerapi.ProviderRegistry {
	t.Helper()
	registry := providerapi.NewProviderRegistry()
	loader := func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
		return func(_ *goja.Runtime, _ *goja.Object) {}, nil
	}
	if err := registry.Package("fixture", providerapi.Module{Name: "hello", DefaultAs: "hello", NewModuleFactory: loader}, providerapi.CommandSetProvider{Name: "tools", NewCommandSet: func(providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
		return &providerapi.CommandSet{}, nil
	}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	return registry
}
