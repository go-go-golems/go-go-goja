package providerutil

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

// CollectConfigSections collects Glazed config sections from the capabilities
// attached to selected module descriptors. The supplied base context is copied
// for each descriptor and enriched with PackageID and ModuleID before calling
// the provider capability.
func CollectGlazedConfigSections(descriptors []providerapi.ModuleDescriptor, base providerapi.SectionRequest, seen map[string]string) ([]schema.Section, error) {
	sections := []schema.Section{}
	if seen == nil {
		seen = map[string]string{}
	}
	applied := map[string]struct{}{}
	for _, descriptor := range descriptors {
		for _, capability := range descriptor.PackageCapabilities {
			sectionCapability, ok := capability.(providerapi.GlazedConfigSectionCapability)
			if !ok {
				continue
			}
			id := capability.CapabilityID()
			key := packageCapabilityKey(descriptor.PackageID, id)
			if _, ok := applied[key]; ok {
				continue
			}
			applied[key] = struct{}{}
			sectionContext := base
			sectionContext.PackageID = descriptor.PackageID
			sectionContext.ModuleID = descriptor.ModuleID
			moduleSections, err := sectionCapability.GlazedConfigSections(sectionContext)
			if err != nil {
				return nil, fmt.Errorf("collect config sections for %s.%s capability %s: %w", descriptor.PackageID, descriptor.ModuleID, id, err)
			}
			source := fmt.Sprintf("%s.%s capability %s", descriptor.PackageID, descriptor.ModuleID, id)
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

// ParseXGojaConfigMap parses a raw xgoja module config map through the
// provider's internal config section.
func ParseXGojaConfigMap(section schema.Section, config map[string]any) (*values.SectionValues, error) {
	if section == nil {
		return nil, fmt.Errorf("xgoja config section is nil")
	}
	fieldValues := fields.NewFieldValues()
	for name, value := range config {
		definition, ok := section.GetDefinitions().Get(name)
		if !ok {
			return nil, fmt.Errorf("unknown xgoja config field %q in section %q", name, section.GetSlug())
		}
		if err := fieldValues.UpdateValue(name, definition, value, fields.WithSource("xgoja.yaml")); err != nil {
			return nil, fmt.Errorf("parse xgoja config field %q: %w", name, err)
		}
	}
	return values.NewSectionValues(section, values.WithFields(fieldValues))
}

// MergeSectionValues returns a fresh SectionValues with static values first and
// override values applied last.
func MergeSectionValues(section schema.Section, staticValues, overrideValues *values.SectionValues) (*values.SectionValues, error) {
	if section == nil {
		return nil, fmt.Errorf("xgoja config section is nil")
	}
	merged, err := values.NewSectionValues(section)
	if err != nil {
		return nil, err
	}
	if staticValues != nil {
		if _, err := merged.Fields.Merge(staticValues.Fields.Clone()); err != nil {
			return nil, err
		}
	}
	if overrideValues != nil {
		if _, err := merged.Fields.Merge(overrideValues.Fields.Clone()); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

// SectionValuesToRawJSON converts final internal xgoja config values into the
// json.RawMessage currently passed through providerapi.ModuleSetupContext.Config.
func SectionValuesToRawJSON(sectionValues *values.SectionValues) (json.RawMessage, error) {
	if sectionValues == nil {
		return nil, nil
	}
	m, err := sectionValues.Fields.ToInterfaceMap()
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// InitRuntimeFromSections runs all runtime initializer capabilities attached to
// the selected module descriptors against one runtime handle.
func InitRuntimeFromSections(ctx context.Context, vals *values.Values, handle providerapi.RuntimeInitializerHandle, descriptors []providerapi.ModuleDescriptor) error {
	if handle == nil || handle.EngineRuntime() == nil {
		return fmt.Errorf("runtime handle is nil")
	}
	applied := map[string]struct{}{}
	for _, descriptor := range descriptors {
		for _, capability := range descriptor.PackageCapabilities {
			initializer, ok := capability.(providerapi.RuntimeInitializerCapability)
			if !ok {
				continue
			}
			id := capability.CapabilityID()
			key := packageCapabilityKey(descriptor.PackageID, id)
			if _, ok := applied[key]; ok {
				continue
			}
			applied[key] = struct{}{}
			if err := initializer.InitRuntimeFromSections(ctx, vals, handle); err != nil {
				return fmt.Errorf("initialize runtime from sections for %s.%s capability %s: %w", descriptor.PackageID, descriptor.ModuleID, id, err)
			}
		}
	}
	return nil
}

func packageCapabilityKey(packageID, capabilityID string) string {
	return packageID + "\x00" + capabilityID
}
