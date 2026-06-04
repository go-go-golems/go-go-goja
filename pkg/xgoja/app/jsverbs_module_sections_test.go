package app

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestJSVerbsCommandsIncludeRuntimeProfileModuleSections(t *testing.T) {
	registry := newJSVerbsSectionRegistry(t)
	runtimeSpec := jsverbsSectionSpec()
	embedded := jsverbsSectionFS()
	commands, err := buildVerbCommands(registry, NewRuntimeFactory(registry, runtimeSpec), runtimeSpec, embedded)
	if err != nil {
		t.Fatalf("build verb commands: %v", err)
	}
	if len(commands) != 1 {
		t.Fatalf("commands = %d", len(commands))
	}
	section, ok := commands[0].Description().Schema.Get("fixture")
	if !ok {
		t.Fatal("expected fixture section on jsverb command")
	}
	if section.GetPrefix() != "fixture-" {
		t.Fatalf("fixture prefix = %q", section.GetPrefix())
	}
}

func TestJSVerbsInitializeRuntimeFromModuleSections(t *testing.T) {
	registry := newJSVerbsSectionRegistry(t)
	specJSON := `{
  "name": "fixture",
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "runtimes": {"main": {"modules": [{"package": "fixture", "name": "mod", "as": "mod"}]}},
  "commands": {"jsverbs": {"enabled": true, "runtime": "main", "name": "verbs"}},
  "jsverbs": [{"id": "local", "path": "xgoja_embed/jsverbs/local", "embed": true}]
}`
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: specJSON, EmbeddedJSVerbs: jsverbsSectionFS()})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"verbs", "tools", "check-fixture", "--fixture-value", "ok"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute jsverb: %v", err)
	}
}

func newJSVerbsSectionRegistry(t *testing.T) *providerapi.Registry {
	t.Helper()
	registry := providerapi.NewRegistry()
	if err := registry.Package("fixture",
		providerapi.Module{Name: "mod", New: func(providerapi.ModuleContext) (require.ModuleLoader, error) {
			return func(vm *goja.Runtime, module *goja.Object) {}, nil
		}},
		providerapi.WithPackageCapability(runFixtureCapability{}),
	); err != nil {
		t.Fatalf("register fixture package: %v", err)
	}
	return registry
}

func jsverbsSectionSpec() *RuntimeSpec {
	return &RuntimeSpec{
		Runtimes: map[string]RuntimeProfileSpec{
			"main": {Modules: []ModuleInstanceSpec{{Package: "fixture", Name: "mod", As: "mod"}}},
		},
		Commands: CommandsSpec{JSVerbs: CommandSpec{Enabled: true, Runtime: "main", Name: "verbs"}},
		JSVerbs:  []JSVerbSourceSpec{{ID: "local", Path: "xgoja_embed/jsverbs/local", Embed: true}},
	}
}

func jsverbsSectionFS() fstest.MapFS {
	return fstest.MapFS{
		"xgoja_embed/jsverbs/local/tools.js": &fstest.MapFile{Data: []byte(`
__package__({ name: "tools" })
__verb__("checkFixture", {
  name: "check-fixture",
  output: "text"
})
function checkFixture() {
  if (globalThis.fixtureValue !== "ok") {
    throw new Error("fixtureValue=" + globalThis.fixtureValue)
  }
  return "ok"
}
`)},
	}
}
