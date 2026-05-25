package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func (f *RuntimeFactory) selectedModuleDescriptors(profile string) ([]providerapi.ModuleDescriptor, error) {
	if f == nil || f.providers == nil || f.spec == nil {
		return nil, fmt.Errorf("xgoja runtime factory is not initialized")
	}
	runtime, ok := f.spec.Runtimes[profile]
	if !ok {
		return nil, fmt.Errorf("unknown runtime profile %q", profile)
	}
	descriptors := make([]providerapi.ModuleDescriptor, 0, len(runtime.Modules))
	capabilitiesUsed := map[string]struct{}{}
	for _, instance := range runtime.Modules {
		module, ok := f.providers.ResolveModule(instance.Package, instance.Name)
		if !ok {
			return nil, fmt.Errorf("runtime %s references unknown provider module %s.%s", profile, instance.Package, instance.Name)
		}
		capabilities := []providerapi.ModuleCapability(nil)
		if _, ok := capabilitiesUsed[instance.Package]; !ok {
			capabilities, _ = f.providers.ResolveCapabilities(instance.Package)
			capabilitiesUsed[instance.Package] = struct{}{}
		}
		descriptors = append(descriptors, providerapi.ModuleDescriptor{
			PackageID:    instance.Package,
			ModuleID:     instance.Name,
			As:           instance.Alias(),
			Module:       module,
			Capabilities: capabilities,
		})
	}
	return descriptors, nil
}

func (f *RuntimeFactory) sectionsForRuntimeProfile(commandName, profile string) ([]schema.Section, []providerapi.ModuleDescriptor, error) {
	descriptors, err := f.selectedModuleDescriptors(profile)
	if err != nil {
		return nil, nil, err
	}
	sections := []schema.Section{}
	seen := map[string]string{}
	for _, descriptor := range descriptors {
		for _, capability := range descriptor.Capabilities {
			sectionCapability, ok := capability.(providerapi.ConfigSectionCapability)
			if !ok {
				continue
			}
			moduleSections, err := sectionCapability.ConfigSections(providerapi.SectionContext{
				CommandName:    commandName,
				RuntimeProfile: profile,
				PackageID:      descriptor.PackageID,
				ModuleID:       descriptor.ModuleID,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("collect config sections for %s.%s capability %s: %w", descriptor.PackageID, descriptor.ModuleID, capability.CapabilityID(), err)
			}
			if err := appendUniqueSections(&sections, seen, moduleSections, fmt.Sprintf("%s.%s capability %s", descriptor.PackageID, descriptor.ModuleID, capability.CapabilityID())); err != nil {
				return nil, nil, err
			}
		}
	}
	return sections, descriptors, nil
}

func appendUniqueSections(out *[]schema.Section, seen map[string]string, sections []schema.Section, source string) error {
	for _, section := range sections {
		if section == nil {
			return fmt.Errorf("%s returned nil config section", source)
		}
		slug := strings.TrimSpace(section.GetSlug())
		if slug == "" {
			return fmt.Errorf("%s returned config section with empty slug", source)
		}
		if previous, ok := seen[slug]; ok {
			return fmt.Errorf("duplicate config section slug %q from %s; already provided by %s", slug, source, previous)
		}
		seen[slug] = source
		*out = append(*out, section)
	}
	return nil
}

func addSectionsToCommandDescription(desc *cmds.CommandDescription, sections []schema.Section, source string) error {
	if desc == nil {
		return fmt.Errorf("command description is nil")
	}
	seen := map[string]string{}
	if desc.Schema != nil {
		desc.Schema.ForEach(func(slug string, _ schema.Section) {
			seen[slug] = "command schema"
		})
	}
	return appendSectionsToCommandDescription(desc, seen, sections, source)
}

func appendSectionsToCommandDescription(desc *cmds.CommandDescription, seen map[string]string, sections []schema.Section, source string) error {
	collected := []schema.Section{}
	if err := appendUniqueSections(&collected, seen, sections, source); err != nil {
		return err
	}
	desc.SetSections(collected...)
	return nil
}

func initRuntimeFromSections(ctx context.Context, vals *values.Values, rt *JSRuntime, descriptors []providerapi.ModuleDescriptor) error {
	if rt == nil {
		return fmt.Errorf("runtime is nil")
	}
	handle := runtimeHandle{rt: rt}
	for _, descriptor := range descriptors {
		for _, capability := range descriptor.Capabilities {
			initializer, ok := capability.(providerapi.RuntimeInitializerCapability)
			if !ok {
				continue
			}
			if err := initializer.InitRuntimeFromSections(ctx, vals, handle); err != nil {
				return fmt.Errorf("initialize runtime from sections for %s.%s capability %s: %w", descriptor.PackageID, descriptor.ModuleID, capability.CapabilityID(), err)
			}
		}
	}
	return nil
}

type runtimeHandle struct {
	rt *JSRuntime
}

func (h runtimeHandle) Runtime() *goja.Runtime {
	if h.rt == nil {
		return nil
	}
	return h.rt.VM
}

func (h runtimeHandle) Close(ctx context.Context) error {
	if h.rt == nil {
		return nil
	}
	return h.rt.Close(ctx)
}
