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
	VM        *goja.Runtime
	Loop      *eventloop.EventLoop
	Owner     runtimeowner.Runner
	AddCloser func(func(context.Context) error) error
}
