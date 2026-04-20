package sandbox

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/engine"
	mssandbox "github.com/go-go-golems/go-go-goja/modules/sandbox"
)

// Config controls the runtime-scoped sandbox registrar.
type Config struct {
	ModuleName string
}

// Registrar installs the sandbox CommonJS module into a specific runtime.
type Registrar struct {
	config Config
}

var _ engine.RuntimeModuleRegistrar = (*Registrar)(nil)

// NewRegistrar creates a runtime-scoped sandbox module registrar.
func NewRegistrar(config Config) *Registrar {
	return &Registrar{config: config}
}

// ID returns a stable identifier for the registrar.
func (r *Registrar) ID() string {
	return "sandbox-registrar"
}

// RegisterRuntimeModules attaches the sandbox module loader to the runtime's
// require registry and seeds the runtime state object.
func (r *Registrar) RegisterRuntimeModules(ctx *engine.RuntimeModuleContext, reg *require.Registry) error {
	if reg == nil {
		return fmt.Errorf("require registry is nil")
	}

	moduleName := strings.TrimSpace(r.config.ModuleName)
	if moduleName == "" {
		moduleName = "sandbox"
	}
	state := mssandbox.NewRuntimeState(moduleName)

	if ctx != nil {
		ctx.SetValue(mssandbox.RuntimeStateContextKey, state)
		if ctx.VM != nil {
			mssandbox.RegisterRuntimeState(ctx.VM, state)
		}
		if ctx.AddCloser != nil && ctx.VM != nil {
			if err := ctx.AddCloser(func(context.Context) error {
				mssandbox.UnregisterRuntimeState(ctx.VM)
				return nil
			}); err != nil {
				return err
			}
		}
	}

	reg.RegisterNativeModule(state.ModuleName(), state.Loader)
	return nil
}

// Re-export the runtime-facing sandbox API from the host package.
type (
	DispatchRequest = mssandbox.DispatchRequest
	BotHandle       = mssandbox.BotHandle
	RuntimeState    = mssandbox.RuntimeState
)

var (
	CompileBot          = mssandbox.CompileBot
	ModuleNameOrDefault = mssandbox.ModuleNameOrDefault
)

const RuntimeStateContextKey = mssandbox.RuntimeStateContextKey
