package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type JSRuntime = engine.Runtime

type RuntimeFactory struct {
	providers   *providerapi.Registry
	runtimeSpec *RuntimeSpec
	services    providerapi.HostServices
}

type providerRuntimeModuleSpec struct {
	instance ModuleInstanceSpec
	module   providerapi.Module
	services providerapi.HostServices
}

func (s providerRuntimeModuleSpec) ID() string {
	return fmt.Sprintf("xgoja:%s.%s:%s", s.instance.Package, s.instance.Name, s.instance.Alias())
}

func (s providerRuntimeModuleSpec) RegisterRuntimeModule(ctx *engine.RuntimeModuleContext, reg *require.Registry) error {
	if reg == nil {
		return fmt.Errorf("require registry is nil")
	}
	config, err := json.Marshal(s.instance.Config)
	if err != nil {
		return fmt.Errorf("marshal config for %s.%s: %w", s.instance.Package, s.instance.Name, err)
	}
	loader, err := s.module.New(providerapi.ModuleContext{
		Context:      ctx.Context,
		Name:         s.instance.Name,
		As:           s.instance.Alias(),
		Config:       config,
		Host:         s.services,
		RuntimeOwner: ctx.Owner,
	})
	if err != nil {
		return fmt.Errorf("create module %s.%s: %w", s.instance.Package, s.instance.Name, err)
	}
	reg.RegisterNativeModule(s.instance.Alias(), loader)
	return nil
}

func NewRuntimeFactory(providers *providerapi.Registry, runtimeSpec *RuntimeSpec, services ...providerapi.HostServices) *RuntimeFactory {
	var hostServices providerapi.HostServices
	if len(services) > 0 {
		hostServices = services[0]
	}
	return &RuntimeFactory{providers: providers, runtimeSpec: runtimeSpec, services: hostServices}
}

func (f *RuntimeFactory) NewRuntime(ctx context.Context, profile string, opts ...require.Option) (*JSRuntime, error) {
	if f == nil || f.providers == nil || f.runtimeSpec == nil {
		return nil, fmt.Errorf("xgoja runtime factory is not initialized")
	}
	runtime, ok := f.runtimeSpec.Runtimes[profile]
	if !ok {
		return nil, fmt.Errorf("unknown runtime profile %q", profile)
	}
	modules := make([]engine.RuntimeModuleSpec, 0, len(runtime.Modules))
	for _, instance := range runtime.Modules {
		module, ok := f.providers.ResolveModule(instance.Package, instance.Name)
		if !ok {
			return nil, fmt.Errorf("runtime %s references unknown provider module %s.%s", profile, instance.Package, instance.Name)
		}
		modules = append(modules, providerRuntimeModuleSpec{instance: instance, module: module, services: f.services})
	}
	builder := engine.NewBuilder(
		engine.WithImplicitDefaultRegistryModules(false),
		engine.WithDataOnlyDefaultRegistryModules(false),
	).WithModules(modules...)
	if len(opts) > 0 {
		builder = builder.WithRequireOptions(opts...)
	}
	runtimeFactory, err := builder.Build()
	if err != nil {
		return nil, err
	}
	return runtimeFactory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))
}
