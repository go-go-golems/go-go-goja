package providerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

func TestRegistryPackageRegistersModulesAndVerbSources(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Package("core",
		Module{Name: "fs", DefaultAs: "fs", New: noopFactory},
		Module{Name: "yaml", DefaultAs: "yaml", New: noopFactory},
		VerbSource{Name: "builtin", Root: "verbs"},
	); err != nil {
		t.Fatalf("register package: %v", err)
	}

	mod, ok := registry.ResolveModule("core", "yaml")
	if !ok {
		t.Fatal("expected core.yaml module")
	}
	if mod.DefaultAs != "yaml" {
		t.Fatalf("default alias = %q", mod.DefaultAs)
	}
	source, ok := registry.ResolveVerbSource("core", "builtin")
	if !ok {
		t.Fatal("expected core builtin verb source")
	}
	if source.Root != "verbs" {
		t.Fatalf("verb source root = %q", source.Root)
	}
	packages := registry.Packages()
	if len(packages) != 1 || packages[0].ID != "core" {
		t.Fatalf("packages = %#v", packages)
	}
}

func TestRegistryRejectsDuplicates(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Package("core", Module{Name: "fs", New: noopFactory}); err != nil {
		t.Fatalf("register package: %v", err)
	}
	if err := registry.Package("core"); err == nil || !strings.Contains(err.Error(), "duplicate provider package") {
		t.Fatalf("expected duplicate package error, got %v", err)
	}

	registry = NewRegistry()
	err := registry.Package("web", Module{Name: "fetch", New: noopFactory}, Module{Name: "fetch", New: noopFactory})
	if err == nil || !strings.Contains(err.Error(), "duplicate module") {
		t.Fatalf("expected duplicate module error, got %v", err)
	}

	err = NewRegistry().Package("verbs", VerbSource{Name: "default"}, VerbSource{Name: "default"})
	if err == nil || !strings.Contains(err.Error(), "duplicate verb source") {
		t.Fatalf("expected duplicate verb source error, got %v", err)
	}
}

func TestRegistryRejectsInvalidEntries(t *testing.T) {
	if err := NewRegistry().Package("", Module{Name: "fs", New: noopFactory}); err == nil {
		t.Fatal("expected empty package id error")
	}
	if err := NewRegistry().Package("core", Module{Name: ""}); err == nil || !strings.Contains(err.Error(), "module name") {
		t.Fatalf("expected module name error, got %v", err)
	}
	if err := NewRegistry().Package("core", Module{Name: "fs"}); err == nil || !strings.Contains(err.Error(), "factory") {
		t.Fatalf("expected factory error, got %v", err)
	}
	if err := NewRegistry().Package("core", VerbSource{Name: ""}); err == nil || !strings.Contains(err.Error(), "verb source name") {
		t.Fatalf("expected verb source name error, got %v", err)
	}
}

func noopFactory(ModuleContext) (require.ModuleLoader, error) {
	return func(vm *goja.Runtime, module *goja.Object) {}, nil
}

func TestModuleFactoryReceivesContextShape(t *testing.T) {
	loader, err := noopFactory(ModuleContext{Context: context.Background(), Name: "fs", As: "fs"})
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	if loader == nil {
		t.Fatal("expected loader")
	}
}
