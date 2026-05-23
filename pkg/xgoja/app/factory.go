package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type JSRuntime struct {
	VM      *goja.Runtime
	Require *require.RequireModule
	Loop    *eventloop.EventLoop
	Owner   runtimeowner.Runner

	runtimeCtx       context.Context
	runtimeCtxCancel context.CancelFunc
	closeOnce        sync.Once
}

type runtimebridgeOwner struct {
	owner runtimeowner.Runner
}

func (o runtimebridgeOwner) Call(ctx context.Context, op string, fn func(context.Context, *goja.Runtime) (any, error)) (any, error) {
	return o.owner.Call(ctx, op, runtimeowner.CallFunc(fn))
}

func (o runtimebridgeOwner) Post(ctx context.Context, op string, fn func(context.Context, *goja.Runtime)) error {
	return o.owner.Post(ctx, op, runtimeowner.PostFunc(fn))
}

type RuntimeFactory struct {
	providers *providerapi.Registry
	spec      *Spec
}

func NewRuntimeFactory(providers *providerapi.Registry, spec *Spec) *RuntimeFactory {
	return &RuntimeFactory{providers: providers, spec: spec}
}

func (f *RuntimeFactory) NewRuntime(ctx context.Context, profile string, opts ...require.Option) (*JSRuntime, error) {
	if f == nil || f.providers == nil || f.spec == nil {
		return nil, fmt.Errorf("xgoja runtime factory is not initialized")
	}
	runtime, ok := f.spec.Runtimes[profile]
	if !ok {
		return nil, fmt.Errorf("unknown runtime profile %q", profile)
	}
	vm := goja.New()
	loop := eventloop.NewEventLoop()
	go loop.Start()
	owner := runtimeowner.NewRunner(vm, loop, runtimeowner.Options{
		Name:          "xgoja-runtime",
		RecoverPanics: true,
	})
	// #nosec G118 -- the generated runtime owns this cancel func and calls it on Close and setup failures.
	runtimeCtx, runtimeCtxCancel := context.WithCancel(context.Background())
	rt := &JSRuntime{
		VM:               vm,
		Loop:             loop,
		Owner:            owner,
		runtimeCtx:       runtimeCtx,
		runtimeCtxCancel: runtimeCtxCancel,
	}
	runtimebridge.Store(vm, runtimebridge.Bindings{
		Context: runtimeCtx,
		Loop:    loop,
		Owner:   runtimebridgeOwner{owner: owner},
	})
	registry := require.NewRegistry(opts...)
	for _, instance := range runtime.Modules {
		module, ok := f.providers.ResolveModule(instance.Package, instance.Name)
		if !ok {
			_ = rt.Close(context.Background())
			return nil, fmt.Errorf("runtime %s references unknown provider module %s.%s", profile, instance.Package, instance.Name)
		}
		config, err := json.Marshal(instance.Config)
		if err != nil {
			_ = rt.Close(context.Background())
			return nil, fmt.Errorf("marshal config for %s.%s: %w", instance.Package, instance.Name, err)
		}
		loader, err := module.New(providerapi.ModuleContext{
			Context: runtimeCtx,
			Name:    instance.Name,
			As:      instance.Alias(),
			Config:  config,
		})
		if err != nil {
			_ = rt.Close(context.Background())
			return nil, fmt.Errorf("create module %s.%s: %w", instance.Package, instance.Name, err)
		}
		registry.RegisterNativeModule(instance.Alias(), loader)
	}
	rt.Require = registry.Enable(vm)
	return rt, nil
}

func (r *JSRuntime) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var retErr error
	r.closeOnce.Do(func() {
		if r.runtimeCtxCancel != nil {
			r.runtimeCtxCancel()
		}
		if r.VM != nil {
			runtimebridge.Delete(r.VM)
		}
		if r.Owner != nil {
			retErr = errors.Join(retErr, r.Owner.Shutdown(ctx))
		}
		if r.Loop != nil {
			r.Loop.Stop()
		}
	})
	return retErr
}
