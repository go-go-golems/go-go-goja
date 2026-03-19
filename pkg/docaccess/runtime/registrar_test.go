package runtime

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/glazed/pkg/help"
	helpmodel "github.com/go-go-golems/glazed/pkg/help/model"
	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/docaccess"
	pluginprovider "github.com/go-go-golems/go-go-goja/pkg/docaccess/plugin"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
	jsdocmodel "github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
)

func TestRegistrarRegistersDocsModuleWithHelpAndJSDocSources(t *testing.T) {
	helpSystem := help.NewHelpSystem()
	helpSystem.AddSection(&help.Section{
		Section: &helpmodel.Section{
			Slug:        "repl-usage",
			Title:       "REPL Usage",
			Short:       "How to use the REPL",
			Content:     "Detailed help",
			Topics:      []string{"goja", "repl"},
			SectionType: helpmodel.SectionGeneralTopic,
		},
		HelpSystem: helpSystem,
	})

	store := jsdocmodel.NewDocStore()
	store.AddFile(&jsdocmodel.FileDoc{
		FilePath: "math.js",
		Symbols: []*jsdocmodel.SymbolDoc{{
			Name:       "smoothstep",
			Summary:    "Smooth interpolation",
			SourceFile: "math.js",
		}},
	})

	factory, err := engine.NewBuilder().
		WithRuntimeModuleRegistrars(NewRegistrar(Config{
			HelpSources: []HelpSource{{
				ID:      "default-help",
				Title:   "Default Help",
				Summary: "Embedded help pages",
				System:  helpSystem,
			}},
			JSDocSources: []JSDocSource{{
				ID:      "workspace-jsdoc",
				Title:   "Workspace JS Docs",
				Summary: "Extracted JavaScript docs",
				Store:   store,
			}},
		})).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() {
		if err := rt.Close(context.Background()); err != nil {
			t.Fatalf("close runtime: %v", err)
		}
	}()

	hubValue, ok := rt.Value(RuntimeHubContextKey)
	if !ok {
		t.Fatalf("runtime docs hub missing")
	}
	if _, ok := hubValue.(*docaccess.Hub); !ok {
		t.Fatalf("runtime docs hub type = %T, want *docaccess.Hub", hubValue)
	}

	value, err := rt.Require.Require("docs")
	if err != nil {
		t.Fatalf("require docs: %v", err)
	}
	if value == nil {
		t.Fatalf("docs module is nil")
	}

	sourcesValue, err := rt.VM.RunString(`
		const docs = require("docs")
		docs.sources()
	`)
	if err != nil {
		t.Fatalf("docs.sources(): %v", err)
	}
	sources, ok := sourcesValue.Export().([]map[string]any)
	if !ok || len(sources) != 2 {
		t.Fatalf("sources = %#v", sourcesValue.Export())
	}

	entryValue, err := rt.VM.RunString(`require("docs").bySymbol("workspace-jsdoc", "smoothstep")`)
	if err != nil {
		t.Fatalf("docs.bySymbol(): %v", err)
	}
	entryMap, ok := entryValue.Export().(map[string]any)
	if !ok {
		t.Fatalf("entry = %#v", entryValue.Export())
	}
	if got := entryMap["title"]; got != "smoothstep" {
		t.Fatalf("title = %#v", got)
	}
}

func TestRegistrarExposesPluginMethodDocs(t *testing.T) {
	binDir := t.TempDir()
	buildPlugin(t, filepath.Join(binDir, "goja-plugin-examples-kv"), "./plugins/examples/kv")

	factory, err := engine.NewBuilder().
		WithRuntimeModuleRegistrars(
			host.NewRegistrar(host.Config{Directories: []string{binDir}}),
			NewRegistrar(Config{
				HelpSources: []HelpSource{{
					ID:     "default-help",
					Title:  "Default Help",
					System: help.NewHelpSystem(),
				}},
			}),
		).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() {
		if err := rt.Close(context.Background()); err != nil {
			t.Fatalf("close runtime: %v", err)
		}
	}()

	entryValue, err := rt.VM.RunString(`require("docs").byID("` + pluginprovider.DefaultSourceID + `", "plugin-method", "plugin:examples:kv/store.get")`)
	if err != nil {
		t.Fatalf("docs.byID(): %v", err)
	}
	entryMap, ok := entryValue.Export().(map[string]any)
	if !ok {
		t.Fatalf("entry = %#v", entryValue.Export())
	}
	if got := entryMap["body"]; got == nil || got == "" {
		t.Fatalf("body = %#v, want non-empty", got)
	}
}

func buildPlugin(t *testing.T, outputPath, packagePath string) {
	t.Helper()

	cmd := exec.Command("go", "build", "-o", outputPath, packagePath)
	cmd.Dir = repoRoot(t)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build plugin %s: %v\n%s", packagePath, err, output)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}
