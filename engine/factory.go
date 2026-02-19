package engine

import (
	"log"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"

	"github.com/go-go-golems/go-go-goja/modules"
)

// Factory prebuilds reusable runtime bootstrap state and can create multiple
// fresh runtimes with lower per-runtime setup overhead.
type Factory struct {
	settings openSettings
	registry *require.Registry
}

// NewFactory creates a reusable runtime factory from engine options.
func NewFactory(opts ...Option) *Factory {
	settings := defaultOpenSettings()
	for _, opt := range opts {
		if opt != nil {
			opt(&settings)
		}
	}

	reg := require.NewRegistry(settings.requireOptions...)
	modules.EnableAll(reg)
	log.Printf("go-go-goja: native modules enabled")

	return &Factory{
		settings: settings,
		registry: reg,
	}
}

// NewRuntime creates a new runtime from this factory using the factory's
// preconfigured bootstrap state.
func (f *Factory) NewRuntime() (*goja.Runtime, *require.RequireModule) {
	vm := goja.New()
	reqMod := f.registry.Enable(vm)
	console.Enable(vm)
	return vm, reqMod
}
