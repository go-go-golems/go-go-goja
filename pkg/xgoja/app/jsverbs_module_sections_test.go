package app

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestJSVerbsCommandsIncludeRuntimeModuleSections(t *testing.T) {
	registry := newJSVerbsSectionRegistry(t)
	runtimePlan := jsverbsSectionSpec()
	embedded := jsverbsSectionFS()
	sources := NewSourceRegistry(registry, embedded, runtimePlan.allSources())
	commands, err := buildVerbCommands(sources, NewRuntimeFactory(registry, runtimePlan), runtimePlan)
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
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "app": {
    "name": "fixture"
  },
  "target": {
    "kind": "xgoja",
    "output": "dist/fixture"
  },
  "providers": [
    {
      "id": "fixture"
    }
  ],
  "runtime": {
    "modules": [
      {
        "provider": "fixture",
        "name": "mod",
        "as": "mod"
      }
    ]
  },
  "sources": [
    {
      "id": "local",
      "path": "xgoja_embed/jsverbs/local",
      "embed": true,
      "kind": "jsverbs"
    }
  ],
  "commands": [
    {
      "id": "jsverbs",
      "type": "builtin.jsverbs",
      "name": "verbs",
      "sources": [
        "local"
      ]
    }
  ]
}`
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: specJSON, EmbeddedJSVerbs: jsverbsSectionFS()})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"verbs", "tools", "check-fixture", "--fixture-value", "ok"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute jsverb: %v", err)
	}
}

func newJSVerbsSectionRegistry(t *testing.T) *providerapi.ProviderRegistry {
	t.Helper()
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("fixture",
		providerapi.Module{Name: "mod", NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			return func(vm *goja.Runtime, module *goja.Object) {}, nil
		}},
		providerapi.WithPackageCapability(runFixtureCapability{}),
	); err != nil {
		t.Fatalf("register fixture package: %v", err)
	}
	return registry
}

func jsverbsSectionSpec() *RuntimePlan {
	return &RuntimePlan{
		Runtime:  RuntimeSection{Modules: []RuntimeModulePlan{{Provider: "fixture", Name: "mod", As: "mod"}}},
		Commands: []CommandPlan{{ID: "verbs", Type: "builtin.jsverbs", Name: "verbs"}},
		Sources:  []SourcePlan{{ID: "local", Kind: SourceKindJSVerbs, Path: "xgoja_embed/jsverbs/local", Embed: true}},
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
