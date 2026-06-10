package dtsgen

import (
	"strings"
	"testing"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestRenderRuntimeSpecUsesAliasesAndDoesNotMutateProviderDescriptor(t *testing.T) {
	t.Parallel()

	descriptor := &spec.Module{
		Name: "fs",
		Functions: []spec.Function{{
			Name:    "readFileSync",
			Returns: spec.String(),
		}},
	}
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("host", providerapi.Module{
		Name:       "fs",
		DefaultAs:  "fs",
		TypeScript: descriptor,
		NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			return func() require.ModuleLoader { return nil }(), nil
		},
	}); err != nil {
		t.Fatalf("register package: %v", err)
	}

	result, err := RenderModules(registry, []ModuleInstance{{Package: "host", Name: "fs", As: "fs:assets"}}, Options{})
	if err != nil {
		t.Fatalf("render runtime spec: %v", err)
	}
	if !strings.Contains(result.DTS, `declare module "fs:assets"`) {
		t.Fatalf("expected aliased declaration, got:\n%s", result.DTS)
	}
	if descriptor.Name != "fs" {
		t.Fatalf("provider descriptor was mutated: %q", descriptor.Name)
	}
}

func TestRenderRuntimeSpecFallsBackToRuntimeModuleName(t *testing.T) {
	t.Parallel()

	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("pkg", providerapi.Module{
		Name:             "actual-name",
		DefaultAs:        "default-alias",
		TypeScript:       moduleDescriptor("default-alias"),
		NewModuleFactory: noopFactory,
	}); err != nil {
		t.Fatalf("register package: %v", err)
	}
	result, err := RenderModules(registry, []ModuleInstance{{Package: "pkg", Name: "actual-name"}}, Options{})
	if err != nil {
		t.Fatalf("render runtime spec: %v", err)
	}
	if !strings.Contains(result.DTS, `declare module "actual-name"`) {
		t.Fatalf("expected runtime module name declaration, got:\n%s", result.DTS)
	}
	if strings.Contains(result.DTS, `declare module "default-alias"`) {
		t.Fatalf("unexpected provider default alias declaration, got:\n%s", result.DTS)
	}
}

func TestRenderRuntimeSpecReportsMissingDescriptorsInNonStrictMode(t *testing.T) {
	t.Parallel()

	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("pkg", providerapi.Module{Name: "untyped", DefaultAs: "untyped", NewModuleFactory: noopFactory}); err != nil {
		t.Fatalf("register package: %v", err)
	}
	result, err := RenderModules(registry, []ModuleInstance{{Package: "pkg", Name: "untyped"}}, Options{})
	if err != nil {
		t.Fatalf("render runtime spec: %v", err)
	}
	if len(result.Missing) != 1 {
		t.Fatalf("missing len = %d, want 1", len(result.Missing))
	}
	if result.Missing[0].Alias != "untyped" {
		t.Fatalf("missing alias = %q", result.Missing[0].Alias)
	}
}

func TestRenderRuntimeSpecStrictModeFailsOnMissingDescriptor(t *testing.T) {
	t.Parallel()

	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("pkg", providerapi.Module{Name: "untyped", DefaultAs: "untyped", NewModuleFactory: noopFactory}); err != nil {
		t.Fatalf("register package: %v", err)
	}
	_, err := RenderModules(registry, []ModuleInstance{{Package: "pkg", Name: "untyped"}}, Options{Strict: true})
	if err == nil || !strings.Contains(err.Error(), "has no TypeScript descriptor") {
		t.Fatalf("expected missing descriptor error, got %v", err)
	}
}

func TestRenderRuntimeSpecRejectsDuplicateRequireAliases(t *testing.T) {
	t.Parallel()

	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("pkg",
		providerapi.Module{Name: "a", TypeScript: moduleDescriptor("a"), NewModuleFactory: noopFactory},
		providerapi.Module{Name: "b", TypeScript: moduleDescriptor("b"), NewModuleFactory: noopFactory},
	); err != nil {
		t.Fatalf("register package: %v", err)
	}
	_, err := RenderModules(registry, []ModuleInstance{{Package: "pkg", Name: "a", As: "dup"}, {Package: "pkg", Name: "b", As: "dup"}}, Options{})
	if err == nil || !strings.Contains(err.Error(), `duplicate require module alias "dup"`) {
		t.Fatalf("expected duplicate alias error, got %v", err)
	}
}

func noopFactory(providerapi.ModuleSetupContext) (require.ModuleLoader, error) { return nil, nil }

func moduleDescriptor(name string) *spec.Module {
	return &spec.Module{Name: name, Functions: []spec.Function{{Name: "ok", Returns: spec.Boolean()}}}
}
