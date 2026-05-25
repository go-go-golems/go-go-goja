package providerutil

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestCollectConfigSectionsRejectsDuplicateSlugs(t *testing.T) {
	descriptors := []providerapi.ModuleDescriptor{
		{PackageID: "pkg-a", ModuleID: "mod-a", PackageCapabilities: []providerapi.PackageCapability{sectionCapability{slug: "shared"}}},
		{PackageID: "pkg-b", ModuleID: "mod-b", PackageCapabilities: []providerapi.PackageCapability{sectionCapability{slug: "shared"}}},
	}
	_, err := CollectConfigSections(descriptors, providerapi.SectionContext{CommandName: "run"}, nil)
	if err == nil || !strings.Contains(err.Error(), "duplicate config section slug") {
		t.Fatalf("expected duplicate slug error, got %v", err)
	}
}

func TestCollectConfigSectionsRejectsNilSection(t *testing.T) {
	descriptors := []providerapi.ModuleDescriptor{{PackageID: "pkg", ModuleID: "mod", PackageCapabilities: []providerapi.PackageCapability{nilSectionCapability{}}}}
	_, err := CollectConfigSections(descriptors, providerapi.SectionContext{}, nil)
	if err == nil || !strings.Contains(err.Error(), "nil config section") {
		t.Fatalf("expected nil section error, got %v", err)
	}
}

func TestCollectConfigSectionsDedupesSamePackageCapability(t *testing.T) {
	capability := &countingSectionCapability{slug: "settings"}
	descriptors := []providerapi.ModuleDescriptor{
		{PackageID: "pkg", ModuleID: "first", PackageCapabilities: []providerapi.PackageCapability{capability}},
		{PackageID: "pkg", ModuleID: "second", PackageCapabilities: []providerapi.PackageCapability{capability}},
	}
	sections, err := CollectConfigSections(descriptors, providerapi.SectionContext{}, nil)
	if err != nil {
		t.Fatalf("collect config sections: %v", err)
	}
	if capability.calls != 1 {
		t.Fatalf("capability calls = %d", capability.calls)
	}
	if len(sections) != 1 || sections[0].GetSlug() != "settings" {
		t.Fatalf("sections = %#v", sections)
	}
}

func TestCollectConfigSectionsRejectsEmptySlug(t *testing.T) {
	descriptors := []providerapi.ModuleDescriptor{{PackageID: "pkg", ModuleID: "mod", PackageCapabilities: []providerapi.PackageCapability{emptySlugCapability{}}}}
	_, err := CollectConfigSections(descriptors, providerapi.SectionContext{}, nil)
	if err == nil || !strings.Contains(err.Error(), "empty slug") {
		t.Fatalf("expected empty slug error, got %v", err)
	}
}

func TestInitRuntimeFromSectionsCallsInitializers(t *testing.T) {
	initializer := &runtimeInitCapability{}
	handle := fakeRuntimeHandle{vm: goja.New()}
	vals := values.New()
	descriptors := []providerapi.ModuleDescriptor{{PackageID: "pkg", ModuleID: "mod", PackageCapabilities: []providerapi.PackageCapability{initializer}}}
	if err := InitRuntimeFromSections(context.Background(), vals, handle, descriptors); err != nil {
		t.Fatalf("init runtime: %v", err)
	}
	if !initializer.called || initializer.vals != vals || initializer.handle.Runtime() != handle.vm {
		t.Fatalf("initializer not called with expected values: %#v", initializer)
	}
}

func TestInitRuntimeFromSectionsWrapsErrors(t *testing.T) {
	boom := errors.New("boom")
	descriptors := []providerapi.ModuleDescriptor{{PackageID: "pkg", ModuleID: "mod", PackageCapabilities: []providerapi.PackageCapability{&runtimeInitCapability{err: boom}}}}
	err := InitRuntimeFromSections(context.Background(), values.New(), fakeRuntimeHandle{vm: goja.New()}, descriptors)
	if err == nil || !strings.Contains(err.Error(), "pkg.mod capability runtime-init") || !errors.Is(err, boom) {
		t.Fatalf("expected wrapped initializer error, got %v", err)
	}
}

func TestInitRuntimeFromSectionsNoopsWithoutInitializers(t *testing.T) {
	descriptors := []providerapi.ModuleDescriptor{{PackageID: "pkg", ModuleID: "mod", PackageCapabilities: []providerapi.PackageCapability{sectionCapability{slug: "section"}}}}
	if err := InitRuntimeFromSections(context.Background(), values.New(), fakeRuntimeHandle{vm: goja.New()}, descriptors); err != nil {
		t.Fatalf("expected no-op, got %v", err)
	}
}

func TestInitRuntimeFromSectionsDedupesSamePackageCapability(t *testing.T) {
	initializer := &runtimeInitCapability{}
	descriptors := []providerapi.ModuleDescriptor{
		{PackageID: "pkg", ModuleID: "first", PackageCapabilities: []providerapi.PackageCapability{initializer}},
		{PackageID: "pkg", ModuleID: "second", PackageCapabilities: []providerapi.PackageCapability{initializer}},
	}
	if err := InitRuntimeFromSections(context.Background(), values.New(), fakeRuntimeHandle{vm: goja.New()}, descriptors); err != nil {
		t.Fatalf("init runtime: %v", err)
	}
	if initializer.calls != 1 {
		t.Fatalf("initializer calls = %d", initializer.calls)
	}
}

type sectionCapability struct{ slug string }

func (c sectionCapability) CapabilityID() string { return "section" }
func (c sectionCapability) ConfigSections(providerapi.SectionContext) ([]schema.Section, error) {
	section, err := schema.NewSection(c.slug, "Section", schema.WithFields(fields.New("value", fields.TypeString)))
	if err != nil {
		return nil, err
	}
	return []schema.Section{section}, nil
}

type countingSectionCapability struct {
	slug  string
	calls int
}

func (c *countingSectionCapability) CapabilityID() string { return "counting-section" }
func (c *countingSectionCapability) ConfigSections(providerapi.SectionContext) ([]schema.Section, error) {
	c.calls++
	section, err := schema.NewSection(c.slug, "Section", schema.WithFields(fields.New("value", fields.TypeString)))
	if err != nil {
		return nil, err
	}
	return []schema.Section{section}, nil
}

type nilSectionCapability struct{}

func (nilSectionCapability) CapabilityID() string { return "nil-section" }
func (nilSectionCapability) ConfigSections(providerapi.SectionContext) ([]schema.Section, error) {
	return []schema.Section{nil}, nil
}

type emptySlugCapability struct{}

func (emptySlugCapability) CapabilityID() string { return "empty-section" }
func (emptySlugCapability) ConfigSections(providerapi.SectionContext) ([]schema.Section, error) {
	return []schema.Section{&schema.SectionImpl{}}, nil
}

type runtimeInitCapability struct {
	called bool
	calls  int
	vals   *values.Values
	handle providerapi.RuntimeHandle
	err    error
}

func (c runtimeInitCapability) CapabilityID() string { return "runtime-init" }
func (c *runtimeInitCapability) InitRuntimeFromSections(_ context.Context, vals *values.Values, handle providerapi.RuntimeHandle) error {
	c.called = true
	c.calls++
	c.vals = vals
	c.handle = handle
	return c.err
}

type fakeRuntimeHandle struct{ vm *goja.Runtime }

func (h fakeRuntimeHandle) Runtime() *goja.Runtime      { return h.vm }
func (h fakeRuntimeHandle) Close(context.Context) error { return nil }
