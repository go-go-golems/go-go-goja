package http

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestRegister(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, ok := registry.ResolveModule(PackageID, "express"); !ok {
		t.Fatal("expected express module")
	}
	caps, ok := registry.ResolvePackageCapabilities(PackageID)
	if !ok || len(caps) != 1 {
		t.Fatalf("capabilities = %#v ok=%v", caps, ok)
	}
}

func TestCapabilityProvidesHTTPSection(t *testing.T) {
	capability := newHTTPCapability()
	sections, err := capability.ConfigSections(providerapi.SectionContext{})
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	if len(sections) != 1 || sections[0].GetSlug() != "http" {
		t.Fatalf("sections = %#v", sections)
	}
	if sections[0].GetPrefix() != "http-" {
		t.Fatalf("prefix = %q", sections[0].GetPrefix())
	}
}

func TestCapabilityRejectsNilRuntimeHandle(t *testing.T) {
	capability := newHTTPCapability()
	if err := capability.InitRuntimeFromSections(context.Background(), nil, nil); err == nil {
		t.Fatal("expected nil runtime handle error")
	}
}

func TestCapabilityDisablesHTTPWhenValuesAreNil(t *testing.T) {
	capability := newHTTPCapability()
	vm := goja.New()
	if err := capability.InitRuntimeFromSections(context.Background(), nil, testRuntimeHandle{vm: vm}); err != nil {
		t.Fatalf("init runtime: %v", err)
	}
	entry := capability.entry(vm)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.settings.Enabled {
		t.Fatalf("expected nil values to keep HTTP disabled, got %#v", entry.settings)
	}
}

func TestCapabilityEnablesHTTPByDefaultWhenValuesArePresent(t *testing.T) {
	capability := newHTTPCapability()
	vm := goja.New()
	vals := httpValues(t, nil)
	if err := capability.InitRuntimeFromSections(context.Background(), vals, testRuntimeHandle{vm: vm}); err != nil {
		t.Fatalf("init runtime: %v", err)
	}
	entry := capability.entry(vm)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if !entry.settings.Enabled || entry.settings.Listen != "127.0.0.1:8787" {
		t.Fatalf("settings = %#v", entry.settings)
	}
}

func TestCapabilityAllowsExplicitHTTPDisable(t *testing.T) {
	capability := newHTTPCapability()
	vm := goja.New()
	vals := httpValues(t, map[string]any{"enabled": false, "listen": "127.0.0.1:9999"})
	if err := capability.InitRuntimeFromSections(context.Background(), vals, testRuntimeHandle{vm: vm}); err != nil {
		t.Fatalf("init runtime: %v", err)
	}
	entry := capability.entry(vm)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.settings.Enabled || entry.settings.Listen != "127.0.0.1:9999" {
		t.Fatalf("settings = %#v", entry.settings)
	}
}

func httpValues(t *testing.T, overrides map[string]any) *values.Values {
	t.Helper()
	capability := newHTTPCapability()
	sections, err := capability.ConfigSections(providerapi.SectionContext{})
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	section := sections[0]
	fieldValues := fields.NewFieldValues()
	for _, definition := range section.GetDefinitions().ToList() {
		if definition.Default != nil {
			fieldValues.Set(definition.Name, &fields.FieldValue{Definition: definition, Value: *definition.Default})
		}
	}
	for name, value := range overrides {
		definition, ok := section.GetDefinitions().Get(name)
		if !ok {
			t.Fatalf("unknown field %q", name)
		}
		fieldValues.Set(name, &fields.FieldValue{Definition: definition, Value: value})
	}
	sectionValues, err := values.NewSectionValues(section, values.WithFields(fieldValues))
	if err != nil {
		t.Fatalf("section values: %v", err)
	}
	return values.New(values.WithSectionValues("http", sectionValues))
}

type testRuntimeHandle struct {
	vm *goja.Runtime
}

func (h testRuntimeHandle) Runtime() *goja.Runtime      { return h.vm }
func (h testRuntimeHandle) Close(context.Context) error { return nil }
