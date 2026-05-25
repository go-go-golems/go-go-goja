package providerutil

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

// CollectConfigSections collects Glazed config sections from the capabilities
// attached to selected module descriptors. The supplied base context is copied
// for each descriptor and enriched with PackageID and ModuleID before calling
// the provider capability.
func CollectConfigSections(descriptors []providerapi.ModuleDescriptor, base providerapi.SectionContext, seen map[string]string) ([]schema.Section, error) {
	sections := []schema.Section{}
	if seen == nil {
		seen = map[string]string{}
	}
	for _, descriptor := range descriptors {
		for _, capability := range descriptor.Capabilities {
			sectionCapability, ok := capability.(providerapi.ConfigSectionCapability)
			if !ok {
				continue
			}
			sectionContext := base
			sectionContext.PackageID = descriptor.PackageID
			sectionContext.ModuleID = descriptor.ModuleID
			moduleSections, err := sectionCapability.ConfigSections(sectionContext)
			if err != nil {
				return nil, fmt.Errorf("collect config sections for %s.%s capability %s: %w", descriptor.PackageID, descriptor.ModuleID, capability.CapabilityID(), err)
			}
			source := fmt.Sprintf("%s.%s capability %s", descriptor.PackageID, descriptor.ModuleID, capability.CapabilityID())
			if err := AppendUniqueSections(&sections, seen, moduleSections, source); err != nil {
				return nil, err
			}
		}
	}
	return sections, nil
}

// AppendUniqueSections appends sections to out while rejecting nil sections,
// empty slugs, and duplicate slugs.
func AppendUniqueSections(out *[]schema.Section, seen map[string]string, sections []schema.Section, source string) error {
	if seen == nil {
		seen = map[string]string{}
	}
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

// InitRuntimeFromSections runs all runtime initializer capabilities attached to
// the selected module descriptors against one runtime handle.
func InitRuntimeFromSections(ctx context.Context, vals *values.Values, handle providerapi.RuntimeHandle, descriptors []providerapi.ModuleDescriptor) error {
	if handle == nil || handle.Runtime() == nil {
		return fmt.Errorf("runtime handle is nil")
	}
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
