package providerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

func TestRegistryPackageRegistersModulesVerbSourcesAndCapabilities(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Package("core",
		Module{Name: "fs", DefaultAs: "fs", New: noopFactory},
		Module{Name: "yaml", DefaultAs: "yaml", New: noopFactory},
		VerbSource{Name: "builtin", Root: "verbs"},
		WithCapability(testCapability{id: "settings"}),
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
	capabilities, ok := registry.ResolveCapabilities("core")
	if !ok {
		t.Fatal("expected core capabilities")
	}
	if len(capabilities) != 1 || capabilities[0].CapabilityID() != "settings" {
		t.Fatalf("capabilities = %#v", capabilities)
	}
	packages := registry.Packages()
	if len(packages) != 1 || packages[0].ID != "core" {
		t.Fatalf("packages = %#v", packages)
	}
	if len(packages[0].Capabilities) != 1 {
		t.Fatalf("cloned package capabilities = %#v", packages[0].Capabilities)
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

	err = NewRegistry().Package("caps", WithCapability(testCapability{id: "settings"}), WithCapability(testCapability{id: "settings"}))
	if err == nil || !strings.Contains(err.Error(), "duplicate capability") {
		t.Fatalf("expected duplicate capability error, got %v", err)
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
	if err := NewRegistry().Package("core", WithCapability(nil)); err == nil || !strings.Contains(err.Error(), "capability is nil") {
		t.Fatalf("expected nil capability error, got %v", err)
	}
	if err := NewRegistry().Package("core", WithCapability(testCapability{id: ""})); err == nil || !strings.Contains(err.Error(), "capability id") {
		t.Fatalf("expected empty capability id error, got %v", err)
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

type testCapability struct {
	id string
}

func (c testCapability) CapabilityID() string { return c.id }

func (c testCapability) ConfigSections(SectionContext) ([]schema.Section, error) { return nil, nil }

func (c testCapability) InitRuntimeFromSections(context.Context, *values.Values, RuntimeHandle) error {
	return nil
}

func TestCapabilityInterfaces(t *testing.T) {
	var capability ModuleCapability = testCapability{id: "settings"}
	if capability.CapabilityID() != "settings" {
		t.Fatalf("capability id = %q", capability.CapabilityID())
	}
	var _ ConfigSectionCapability = testCapability{}
	var _ RuntimeInitializerCapability = testCapability{}
}
