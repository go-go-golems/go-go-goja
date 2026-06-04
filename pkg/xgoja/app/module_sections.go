package app

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerutil"
)

func (f *RuntimeFactory) selectedModuleDescriptors(profile string) ([]providerapi.ModuleDescriptor, error) {
	if f == nil || f.providers == nil || f.runtimeSpec == nil {
		return nil, fmt.Errorf("xgoja runtime factory is not initialized")
	}
	runtime, ok := f.runtimeSpec.Runtimes[profile]
	if !ok {
		return nil, fmt.Errorf("unknown runtime profile %q", profile)
	}
	descriptors := make([]providerapi.ModuleDescriptor, 0, len(runtime.Modules))
	for _, instance := range runtime.Modules {
		module, ok := f.providers.ResolveModule(instance.Package, instance.Name)
		if !ok {
			return nil, fmt.Errorf("runtime %s references unknown provider module %s.%s", profile, instance.Package, instance.Name)
		}
		capabilities, _ := f.providers.ResolvePackageCapabilities(instance.Package)
		descriptors = append(descriptors, providerapi.ModuleDescriptor{
			PackageID:           instance.Package,
			ModuleID:            instance.Name,
			As:                  instance.Alias(),
			Module:              module,
			PackageCapabilities: capabilities,
		})
	}
	return descriptors, nil
}

func (f *RuntimeFactory) sectionsForRuntimeProfile(commandName, profile string) ([]schema.Section, []providerapi.ModuleDescriptor, error) {
	descriptors, err := f.selectedModuleDescriptors(profile)
	if err != nil {
		return nil, nil, err
	}
	sections, err := providerutil.CollectConfigSections(descriptors, providerapi.SectionContext{
		CommandName:    commandName,
		RuntimeProfile: profile,
	}, nil)
	if err != nil {
		return nil, nil, err
	}
	return sections, descriptors, nil
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
	if err := providerutil.AppendUniqueSections(&collected, seen, sections, source); err != nil {
		return err
	}
	desc.SetSections(collected...)
	return nil
}

func initRuntimeFromSections(ctx context.Context, vals *values.Values, rt *JSRuntime, descriptors []providerapi.ModuleDescriptor) error {
	if rt == nil {
		return fmt.Errorf("runtime is nil")
	}
	return providerutil.InitRuntimeFromSections(ctx, vals, runtimeHandle{rt: rt}, descriptors)
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

func (h runtimeHandle) AddCloser(fn func(context.Context) error) error {
	if h.rt == nil {
		return fmt.Errorf("runtime is nil")
	}
	return h.rt.AddCloser(fn)
}
