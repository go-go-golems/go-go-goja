package engine

import (
	"context"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

// RuntimeModuleSpec registers one or more require() modules for a concrete
// runtime instance. All engine modules are runtime-aware: they receive the VM,
// event loop, owner, lifecycle context, closer registry, and value bag before
// the require registry is enabled.
type RuntimeModuleSpec interface {
	ID() string
	RegisterRuntimeModule(ctx *RuntimeModuleContext, reg *require.Registry) error
}

// RuntimeModuleContext exposes runtime-scoped objects to module specs.
type RuntimeModuleContext struct {
	Context   context.Context
	VM        *goja.Runtime
	Loop      *eventloop.EventLoop
	Owner     runtimeowner.Runner
	AddCloser func(func(context.Context) error) error
	Values    map[string]any
}

func (ctx *RuntimeModuleContext) SetValue(key string, value any) {
	if ctx == nil || key == "" {
		return
	}
	if ctx.Values == nil {
		ctx.Values = map[string]any{}
	}
	ctx.Values[key] = value
}

func (ctx *RuntimeModuleContext) Value(key string) (any, bool) {
	if ctx == nil || ctx.Values == nil || key == "" {
		return nil, false
	}
	value, ok := ctx.Values[key]
	return value, ok
}
