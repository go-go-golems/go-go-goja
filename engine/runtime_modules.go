package engine

import (
	"context"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

// RuntimeModuleRegistrar registers runtime-scoped modules into a require
// registry before it is enabled for a concrete VM instance.
type RuntimeModuleRegistrar interface {
	ID() string
	RegisterRuntimeModules(ctx *RuntimeModuleContext, reg *require.Registry) error
}

// RuntimeModuleContext exposes runtime-scoped objects to module registrars.
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
