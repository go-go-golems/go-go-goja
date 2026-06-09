package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerutil"
)

type JSRuntime = engine.Runtime

type RuntimeFactory struct {
	providers   *providerapi.ProviderRegistry
	runtimeSpec *RuntimeSpec
	services    providerapi.HostServices
}

type providerRuntimeModuleRegistrar struct {
	instance ModuleInstanceSpec
	module   providerapi.Module
	config   json.RawMessage
	services providerapi.HostServices
}

func (s providerRuntimeModuleRegistrar) ID() string {
	return fmt.Sprintf("xgoja:%s.%s:%s", s.instance.Package, s.instance.Name, s.instance.Alias())
}

func (s providerRuntimeModuleRegistrar) RegisterRuntimeModule(ctx *engine.RuntimeModuleRegistrationContext, reg *require.Registry) error {
	if reg == nil {
		return fmt.Errorf("require registry is nil")
	}
	config := s.config
	if config == nil {
		var err error
		config, err = json.Marshal(s.instance.Config)
		if err != nil {
			return fmt.Errorf("marshal config for %s.%s: %w", s.instance.Package, s.instance.Name, err)
		}
	}
	loader, err := s.module.NewModuleFactory(providerapi.ModuleSetupContext{
		Context:      ctx.Context,
		Name:         s.instance.Name,
		As:           s.instance.Alias(),
		Config:       config,
		Host:         s.services,
		RuntimeOwner: ctx.Owner,
		AddCloser:    ctx.AddCloser,
	})
	if err != nil {
		return fmt.Errorf("create module %s.%s: %w", s.instance.Package, s.instance.Name, err)
	}
	reg.RegisterNativeModule(s.instance.Alias(), loader)
	return nil
}

func NewRuntimeFactory(providers *providerapi.ProviderRegistry, runtimeSpec *RuntimeSpec, services ...providerapi.HostServices) *RuntimeFactory {
	var hostServices providerapi.HostServices
	if len(services) > 0 {
		hostServices = services[0]
	}
	return &RuntimeFactory{providers: providers, runtimeSpec: runtimeSpec, services: hostServices}
}

func (f *RuntimeFactory) NewRuntime(ctx context.Context, opts ...require.Option) (*JSRuntime, error) {
	return f.NewRuntimeFromSections(ctx, nil, opts...)
}

func (f *RuntimeFactory) NewRuntimeFromSections(ctx context.Context, vals *values.Values, opts ...require.Option) (*JSRuntime, error) {
	return f.NewRuntimeFromSectionsWithHostServices(ctx, vals, nil, opts...)
}

func (f *RuntimeFactory) NewRuntimeFromSectionsWithHostServices(ctx context.Context, vals *values.Values, hostServices providerapi.HostServices, opts ...require.Option) (*JSRuntime, error) {
	if f == nil || f.providers == nil || f.runtimeSpec == nil {
		return nil, fmt.Errorf("xgoja runtime factory is not initialized")
	}
	descriptors, err := f.selectedModuleDescriptors()
	if err != nil {
		return nil, err
	}
	runtimeServices, err := f.hostServicesForRuntime(ctx, vals, descriptors, hostServices)
	if err != nil {
		return nil, err
	}
	closeUnregisteredHostServices := true
	defer func() {
		if closeUnregisteredHostServices {
			_ = closeHostServiceClosers(ctx, runtimeServices.closers)
		}
	}()
	descriptorsByInstance := map[string]providerapi.ModuleDescriptor{}
	for _, descriptor := range descriptors {
		descriptorsByInstance[moduleDescriptorKey(descriptor.PackageID, descriptor.ModuleID, descriptor.As)] = descriptor
	}
	modules := make([]engine.RuntimeModuleRegistrar, 0, len(f.runtimeSpec.Modules))
	for _, instance := range f.runtimeSpec.Modules {
		module, ok := f.providers.ResolveModule(instance.Package, instance.Name)
		if !ok {
			return nil, fmt.Errorf("runtime references unknown provider module %s.%s", instance.Package, instance.Name)
		}
		descriptor := descriptorsByInstance[moduleDescriptorKey(instance.Package, instance.Name, instance.Alias())]
		if descriptor.Module.Name == "" {
			descriptor = providerapi.ModuleDescriptor{PackageID: instance.Package, ModuleID: instance.Name, As: instance.Alias(), Module: module}
		}
		config, err := f.configForModuleInstance(ctx, instance, descriptor, vals)
		if err != nil {
			return nil, err
		}
		modules = append(modules, providerRuntimeModuleRegistrar{instance: instance, module: module, config: config, services: runtimeServices.services})
	}
	if len(runtimeServices.closers) > 0 {
		modules = append([]engine.RuntimeModuleRegistrar{hostServiceCloserRegistrar{closers: runtimeServices.closers}}, modules...)
	}
	includePanicStack, err := includeRecoveredPanicStack(vals)
	if err != nil {
		return nil, err
	}
	builder := engine.NewRuntimeFactoryBuilder(
		engine.WithImplicitDefaultRegistryModules(false),
		engine.WithDataOnlyDefaultRegistryModules(false),
		engine.WithRecoveredPanicStack(includePanicStack),
	).WithModules(modules...)
	if len(opts) > 0 {
		builder = builder.WithRequireOptions(opts...)
	}
	runtimeFactory, err := builder.Build()
	if err != nil {
		return nil, err
	}
	jsRuntime, err := runtimeFactory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))
	if err != nil {
		return nil, err
	}
	closeUnregisteredHostServices = false
	return jsRuntime, nil
}

func moduleDescriptorKey(packageID, moduleID, as string) string {
	return packageID + "\x00" + moduleID + "\x00" + as
}

func (f *RuntimeFactory) hostServicesForRuntime(ctx context.Context, vals *values.Values, descriptors []providerapi.ModuleDescriptor, runtimeServices providerapi.HostServices) (hostServicesForRuntime, error) {
	baseServices := f.services
	if runtimeServices != nil {
		baseServices = layeredHostServices{base: f.services, overlay: runtimeServices}
	}
	collector := newHostServiceCollector(baseServices)
	success := false
	defer func() {
		if !success {
			_ = closeHostServiceClosers(ctx, collector.closers)
		}
	}()
	seen := map[string]struct{}{}
	for _, descriptor := range descriptors {
		for _, capability := range descriptor.PackageCapabilities {
			hostContribution, ok := capability.(providerapi.HostServiceContributionCapability)
			if !ok {
				continue
			}
			key := descriptor.PackageID + "\x00" + capability.CapabilityID()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			if err := hostContribution.ContributeHostServices(ctx, providerapi.HostServiceContributionRequest{
				SectionRequest: providerapi.SectionRequest{PackageID: descriptor.PackageID, ModuleID: descriptor.ModuleID},
				Values:         vals,
				Modules:        descriptors,
			}, collector); err != nil {
				return hostServicesForRuntime{}, fmt.Errorf("contribute host services for %s capability %s: %w", descriptor.PackageID, capability.CapabilityID(), err)
			}
		}
	}
	success = true
	return collector.servicesForRuntime(), nil
}

func (f *RuntimeFactory) configForModuleInstance(ctx context.Context, instance ModuleInstanceSpec, descriptor providerapi.ModuleDescriptor, vals *values.Values) (json.RawMessage, error) {
	config, err := json.Marshal(instance.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal config for %s.%s: %w", instance.Package, instance.Name, err)
	}
	for _, capability := range descriptor.PackageCapabilities {
		xgojaConfig, ok := capability.(providerapi.XGojaConfigSectionCapability)
		if !ok {
			continue
		}
		sectionRequest := providerapi.SectionRequest{PackageID: descriptor.PackageID, ModuleID: descriptor.ModuleID}
		section, err := xgojaConfig.XGojaConfigSection(sectionRequest, descriptor)
		if err != nil {
			return nil, fmt.Errorf("xgoja config section for %s.%s capability %s: %w", instance.Package, instance.Name, capability.CapabilityID(), err)
		}
		staticValues, err := providerutil.ParseXGojaConfigMap(section, instance.Config)
		if err != nil {
			return nil, fmt.Errorf("parse xgoja config for %s.%s capability %s: %w", instance.Package, instance.Name, capability.CapabilityID(), err)
		}
		overrideValues, err := xgojaConfig.XGojaConfigFromGlazed(ctx, providerapi.XGojaConfigRequest{
			SectionRequest: sectionRequest,
			Descriptor:     descriptor,
			ConfigSection:  section,
			StaticConfig:   staticValues,
			GlazedValues:   vals,
		})
		if err != nil {
			return nil, fmt.Errorf("map glazed config for %s.%s capability %s: %w", instance.Package, instance.Name, capability.CapabilityID(), err)
		}
		mergedValues, err := providerutil.MergeSectionValues(section, staticValues, overrideValues)
		if err != nil {
			return nil, fmt.Errorf("merge xgoja config for %s.%s capability %s: %w", instance.Package, instance.Name, capability.CapabilityID(), err)
		}
		config, err = providerutil.SectionValuesToRawJSON(mergedValues)
		if err != nil {
			return nil, fmt.Errorf("encode xgoja config for %s.%s capability %s: %w", instance.Package, instance.Name, capability.CapabilityID(), err)
		}
	}
	return config, nil
}
