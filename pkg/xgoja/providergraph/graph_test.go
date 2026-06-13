package providergraph

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestBuildResolvesProvidersModulesAndCommandSets(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("http",
		providerapi.Module{
			Name:             "express",
			DefaultAs:        "express",
			TypeScript:       &spec.Module{Name: "express"},
			NewModuleFactory: dummyLoader,
		},
		providerapi.CommandSetProvider{
			Name:         "serve",
			DefaultMount: "serve",
			NewCommandSet: func(providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
				return &providerapi.CommandSet{}, nil
			},
		},
	); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	graph, err := Build(registry, Options{
		Providers:   []ProviderSelection{{ID: "http"}},
		Modules:     []RuntimeModuleSelection{{Provider: "http", Name: "express"}},
		CommandSets: []CommandSetSelection{{ID: "serve", Provider: "http", Name: "serve"}},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := graph.RuntimeModuleAliases(); len(got) != 1 || got[0] != "express" {
		t.Fatalf("aliases = %#v", got)
	}
	if _, ok := graph.ResolveAlias("express"); !ok {
		t.Fatal("expected express alias to resolve")
	}
	if got := graph.CommandSets(); len(got) != 1 || got[0].ID != "serve" {
		t.Fatalf("command sets = %#v", got)
	}
	modules, err := graph.TypeScriptModules(true)
	if err != nil {
		t.Fatalf("TypeScriptModules: %v", err)
	}
	if len(modules) != 1 || modules[0].Name != "express" {
		t.Fatalf("typescript modules = %#v", modules)
	}
}

func TestBuildRejectsDuplicateAliases(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("core",
		providerapi.Module{Name: "one", NewModuleFactory: dummyLoader},
		providerapi.Module{Name: "two", NewModuleFactory: dummyLoader},
	); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	_, err := Build(registry, Options{
		Providers: []ProviderSelection{{ID: "core"}},
		Modules: []RuntimeModuleSelection{
			{Provider: "core", Name: "one", As: "dup"},
			{Provider: "core", Name: "two", As: "dup"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate runtime module alias") {
		t.Fatalf("expected duplicate alias error, got %v", err)
	}
}

func TestBuildRejectsMissingSelections(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("core", providerapi.Module{Name: "path", NewModuleFactory: dummyLoader}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	_, err := Build(registry, Options{Providers: []ProviderSelection{{ID: "missing"}}})
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("expected unknown provider error, got %v", err)
	}
	_, err = Build(registry, Options{Providers: []ProviderSelection{{ID: "core"}}, Modules: []RuntimeModuleSelection{{Provider: "core", Name: "missing"}}})
	if err == nil || !strings.Contains(err.Error(), "unknown runtime module") {
		t.Fatalf("expected unknown module error, got %v", err)
	}
}

func TestTypeScriptModulesStrictRejectsMissingDescriptor(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("core", providerapi.Module{Name: "path", NewModuleFactory: dummyLoader}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	graph, err := Build(registry, Options{Providers: []ProviderSelection{{ID: "core"}}, Modules: []RuntimeModuleSelection{{Provider: "core", Name: "path"}}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	_, err = graph.TypeScriptModules(true)
	if err == nil || !strings.Contains(err.Error(), "no TypeScript descriptor") {
		t.Fatalf("expected descriptor error, got %v", err)
	}
}

func dummyLoader(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
	return func(*goja.Runtime, *goja.Object) {}, nil
}
