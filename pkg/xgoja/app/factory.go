package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type JSRuntime struct {
	VM      *goja.Runtime
	Require *require.RequireModule
}

type RuntimeFactory struct {
	providers *providerapi.Registry
	spec      *Spec
}

func NewRuntimeFactory(providers *providerapi.Registry, spec *Spec) *RuntimeFactory {
	return &RuntimeFactory{providers: providers, spec: spec}
}

func (f *RuntimeFactory) NewRuntime(ctx context.Context, profile string) (*JSRuntime, error) {
	if f == nil || f.providers == nil || f.spec == nil {
		return nil, fmt.Errorf("xgoja runtime factory is not initialized")
	}
	runtime, ok := f.spec.Runtimes[profile]
	if !ok {
		return nil, fmt.Errorf("unknown runtime profile %q", profile)
	}
	vm := goja.New()
	registry := require.NewRegistry()
	for _, instance := range runtime.Modules {
		module, ok := f.providers.ResolveModule(instance.Package, instance.Name)
		if !ok {
			return nil, fmt.Errorf("runtime %s references unknown provider module %s.%s", profile, instance.Package, instance.Name)
		}
		config, err := json.Marshal(instance.Config)
		if err != nil {
			return nil, fmt.Errorf("marshal config for %s.%s: %w", instance.Package, instance.Name, err)
		}
		loader, err := module.New(providerapi.ModuleContext{
			Context: ctx,
			Name:    instance.Name,
			As:      instance.Alias(),
			Config:  config,
		})
		if err != nil {
			return nil, fmt.Errorf("create module %s.%s: %w", instance.Package, instance.Name, err)
		}
		registry.RegisterNativeModule(instance.Alias(), loader)
	}
	return &JSRuntime{VM: vm, Require: registry.Enable(vm)}, nil
}
