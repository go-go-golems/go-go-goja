package engine

import (
	"context"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

// RuntimeModuleRegistrar registers one or more require() modules for a concrete
// runtime instance. All engine modules are runtime-aware: they receive the VM,
// event loop, owner, startup context, closer registry, and value bag before the
// require registry is enabled. Runtime lifetime is available through
// runtimebridge.RuntimeServices once a module loader runs against a VM.
type RuntimeModuleRegistrar interface {
	ID() string
	RegisterRuntimeModule(ctx *RuntimeModuleRegistrationContext, reg *require.Registry) error
}

// RuntimeModuleRegistrationContext exposes runtime-scoped objects to module specs.
type RuntimeModuleRegistrationContext struct {
	Context   context.Context
	VM        *goja.Runtime
	Loop      *eventloop.EventLoop
	Owner     runtimeowner.RuntimeOwner
	AddCloser func(func(context.Context) error) error
	Values    map[string]any
}

func (ctx *RuntimeModuleRegistrationContext) SetValue(key string, value any) {
	if ctx == nil || key == "" {
		return
	}
	if ctx.Values == nil {
		ctx.Values = map[string]any{}
	}
	ctx.Values[key] = value
}

func (ctx *RuntimeModuleRegistrationContext) Value(key string) (any, bool) {
	if ctx == nil || ctx.Values == nil || key == "" {
		return nil, false
	}
	value, ok := ctx.Values[key]
	return value, ok
}
