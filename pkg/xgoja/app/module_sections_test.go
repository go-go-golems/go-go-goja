package app

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestRuntimeFactoryCollectsSectionsForRuntimeProfile(t *testing.T) {
	factory := newSectionTestFactory(t,
		providerapi.WithPackageCapability(sectionCapability{id: "alpha", slug: "alpha"}),
		providerapi.WithPackageCapability(sectionCapability{id: "beta", slug: "beta"}),
	)
	sections, descriptors, err := factory.sectionsForRuntimeProfile("run", "main")
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("descriptors = %#v", descriptors)
	}
	if descriptors[0].PackageID != "fixture" || descriptors[0].ModuleID != "mod" || descriptors[0].As != "alias" {
		t.Fatalf("descriptor = %#v", descriptors[0])
	}
	if got := sectionSlugs(sections); strings.Join(got, ",") != "alpha,beta" {
		t.Fatalf("section slugs = %v", got)
	}
}

func TestRuntimeFactoryRejectsDuplicateSectionSlugs(t *testing.T) {
	factory := newSectionTestFactory(t,
		providerapi.WithPackageCapability(sectionCapability{id: "one", slug: "dup"}),
		providerapi.WithPackageCapability(sectionCapability{id: "two", slug: "dup"}),
	)
	_, _, err := factory.sectionsForRuntimeProfile("run", "main")
	if err == nil || !strings.Contains(err.Error(), "duplicate config section slug") {
		t.Fatalf("expected duplicate section error, got %v", err)
	}
}

func TestRuntimeFactoryAttachesPackageCapabilitiesToEverySelectedModule(t *testing.T) {
	capability := sectionCapability{id: "settings", slug: "fixture"}
	registry := providerapi.NewRegistry()
	if err := registry.Package("fixture",
		providerapi.Module{Name: "first", New: noopSectionModule},
		providerapi.Module{Name: "second", New: noopSectionModule},
		providerapi.WithPackageCapability(capability),
	); err != nil {
		t.Fatalf("register fixture provider: %v", err)
	}
	factory := NewRuntimeFactory(registry, &Spec{Runtimes: map[string]Runtime{
		"main": {Modules: []ModuleInstance{
			{Package: "fixture", Name: "first", As: "first"},
			{Package: "fixture", Name: "second", As: "second"},
		}},
	}})
	descriptors, err := factory.selectedModuleDescriptors("main")
	if err != nil {
		t.Fatalf("selected descriptors: %v", err)
	}
	if len(descriptors) != 2 {
		t.Fatalf("descriptors = %#v", descriptors)
	}
	for _, descriptor := range descriptors {
		if len(descriptor.PackageCapabilities) != 1 {
			t.Fatalf("descriptor %s capabilities = %#v", descriptor.ModuleID, descriptor.PackageCapabilities)
		}
	}
	sections, _, err := factory.sectionsForRuntimeProfile("run", "main")
	if err != nil {
		t.Fatalf("sections should dedupe same package capability: %v", err)
	}
	if got := sectionSlugs(sections); strings.Join(got, ",") != "fixture" {
		t.Fatalf("section slugs = %v", got)
	}
}

func TestInitRuntimeFromSectionsCallsRuntimeInitializers(t *testing.T) {
	called := false
	capability := runtimeInitCapability{
		id: "init",
		fn: func(ctx context.Context, vals *values.Values, handle providerapi.RuntimeHandle) error {
			called = true
			if handle.Runtime() == nil {
				t.Fatalf("expected goja runtime handle")
			}
			return nil
		},
	}
	descriptors := []providerapi.ModuleDescriptor{{
		PackageID:           "fixture",
		ModuleID:            "mod",
		PackageCapabilities: []providerapi.PackageCapability{capability},
	}}
	rt := &JSRuntime{VM: goja.New()}
	if err := initRuntimeFromSections(context.Background(), values.New(), rt, descriptors); err != nil {
		t.Fatalf("init runtime: %v", err)
	}
	if !called {
		t.Fatal("expected initializer to be called")
	}
}

func TestInitRuntimeFromSectionsWrapsInitializerErrors(t *testing.T) {
	descriptors := []providerapi.ModuleDescriptor{{
		PackageID: "fixture",
		ModuleID:  "mod",
		PackageCapabilities: []providerapi.PackageCapability{runtimeInitCapability{
			id: "init",
			fn: func(context.Context, *values.Values, providerapi.RuntimeHandle) error {
				return fmt.Errorf("boom")
			},
		}},
	}}
	rt := &JSRuntime{VM: goja.New()}
	err := initRuntimeFromSections(context.Background(), values.New(), rt, descriptors)
	if err == nil || !strings.Contains(err.Error(), "fixture.mod capability init") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped initializer error, got %v", err)
	}
}

func newSectionTestFactory(t *testing.T, entries ...providerapi.Entry) *RuntimeFactory {
	t.Helper()
	registry := providerapi.NewRegistry()
	allEntries := []providerapi.Entry{providerapi.Module{Name: "mod", New: noopSectionModule}}
	allEntries = append(allEntries, entries...)
	if err := registry.Package("fixture", allEntries...); err != nil {
		t.Fatalf("register fixture provider: %v", err)
	}
	return NewRuntimeFactory(registry, &Spec{
		Runtimes: map[string]Runtime{
			"main": {Modules: []ModuleInstance{{Package: "fixture", Name: "mod", As: "alias"}}},
		},
	})
}

func noopSectionModule(providerapi.ModuleContext) (require.ModuleLoader, error) {
	return func(vm *goja.Runtime, module *goja.Object) {}, nil
}

type sectionCapability struct {
	id   string
	slug string
}

func (c sectionCapability) CapabilityID() string { return c.id }

func (c sectionCapability) ConfigSections(providerapi.SectionContext) ([]schema.Section, error) {
	section, err := schema.NewSection(c.slug, c.slug, schema.WithFields(fields.New("value", fields.TypeString)))
	if err != nil {
		return nil, err
	}
	return []schema.Section{section}, nil
}

type runtimeInitCapability struct {
	id string
	fn func(context.Context, *values.Values, providerapi.RuntimeHandle) error
}

func (c runtimeInitCapability) CapabilityID() string { return c.id }

func (c runtimeInitCapability) InitRuntimeFromSections(ctx context.Context, vals *values.Values, handle providerapi.RuntimeHandle) error {
	return c.fn(ctx, vals, handle)
}

func sectionSlugs(sections []schema.Section) []string {
	out := make([]string, 0, len(sections))
	for _, section := range sections {
		out = append(out, section.GetSlug())
	}
	return out
}
