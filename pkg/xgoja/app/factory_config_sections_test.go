package app

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestRuntimeFactoryAppliesGlazedConfigBeforeModuleSetup(t *testing.T) {
	capability := configPatchCapability{}
	var captured []map[string]any
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("fixture",
		providerapi.Module{
			Name: "mod",
			NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
				var cfg map[string]any
				if err := json.Unmarshal(ctx.Config, &cfg); err != nil {
					return nil, err
				}
				captured = append(captured, cfg)
				return func(*goja.Runtime, *goja.Object) {}, nil
			},
		},
		providerapi.WithPackageCapability(capability),
	); err != nil {
		t.Fatalf("register package: %v", err)
	}
	factory := NewRuntimeFactory(registry, &RuntimeSpec{Runtimes: map[string]RuntimeProfileSpec{
		"main": {Modules: []ModuleInstanceSpec{{Package: "fixture", Name: "mod", As: "alias", Config: map[string]any{"message": "static"}}}},
	}})
	vals := configPatchValues(t, "cli")
	rt, err := factory.NewRuntimeFromSections(context.Background(), "main", vals)
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	if len(captured) != 1 {
		t.Fatalf("captured configs = %d", len(captured))
	}
	if got := captured[0]["message"]; got != "alias:cli" {
		t.Fatalf("message = %#v", got)
	}
}

func TestRuntimeFactoryKeepsIndependentConfigForRepeatedProvider(t *testing.T) {
	capability := configPatchCapability{}
	var captured []map[string]any
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("fixture",
		providerapi.Module{
			Name: "mod",
			NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
				var cfg map[string]any
				if err := json.Unmarshal(ctx.Config, &cfg); err != nil {
					return nil, err
				}
				captured = append(captured, cfg)
				return func(*goja.Runtime, *goja.Object) {}, nil
			},
		},
		providerapi.WithPackageCapability(capability),
	); err != nil {
		t.Fatalf("register package: %v", err)
	}
	factory := NewRuntimeFactory(registry, &RuntimeSpec{Runtimes: map[string]RuntimeProfileSpec{
		"main": {Modules: []ModuleInstanceSpec{
			{Package: "fixture", Name: "mod", As: "one", Config: map[string]any{"message": "static-one"}},
			{Package: "fixture", Name: "mod", As: "two", Config: map[string]any{"message": "static-two"}},
		}},
	}})
	vals := configPatchValues(t, "cli")
	rt, err := factory.NewRuntimeFromSections(context.Background(), "main", vals)
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	if len(captured) != 2 {
		t.Fatalf("captured configs = %d", len(captured))
	}
	if got := captured[0]["message"]; got != "one:cli" {
		t.Fatalf("first message = %#v", got)
	}
	if got := captured[1]["message"]; got != "two:cli" {
		t.Fatalf("second message = %#v", got)
	}
}

type configPatchCapability struct{}

func (configPatchCapability) CapabilityID() string { return "config" }

func (configPatchCapability) GlazedConfigSections(providerapi.SectionRequest) ([]schema.Section, error) {
	section, err := schema.NewSection("fixture", "Fixture", schema.WithFields(fields.New("value", fields.TypeString)))
	if err != nil {
		return nil, err
	}
	return []schema.Section{section}, nil
}

func (configPatchCapability) XGojaConfigSection(providerapi.SectionRequest, providerapi.ModuleDescriptor) (schema.Section, error) {
	return schema.NewSection("fixture-xgoja", "Fixture xgoja", schema.WithFields(fields.New("message", fields.TypeString)))
}

func (configPatchCapability) XGojaConfigFromGlazed(_ context.Context, req providerapi.XGojaConfigRequest) (*values.SectionValues, error) {
	out, err := values.NewSectionValues(req.ConfigSection)
	if err != nil {
		return nil, err
	}
	field, ok := req.GlazedValues.GetField("fixture", "value")
	if !ok {
		return out, nil
	}
	definition, _ := req.ConfigSection.GetDefinitions().Get("message")
	if err := out.Fields.UpdateWithLog("message", definition, req.Descriptor.As+":"+field.Value.(string), field.Log...); err != nil {
		return nil, err
	}
	return out, nil
}

func configPatchValues(t *testing.T, value string) *values.Values {
	t.Helper()
	capability := configPatchCapability{}
	sections, err := capability.GlazedConfigSections(providerapi.SectionRequest{})
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	sectionValues, err := values.NewSectionValues(sections[0], values.WithFieldValue("value", value, fields.WithSource("cobra")))
	if err != nil {
		t.Fatalf("section values: %v", err)
	}
	return values.New(values.WithSectionValues("fixture", sectionValues))
}
