package engine

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"

	"github.com/go-go-golems/go-go-goja/modules"
	// Blank imports ensure module init() functions run so they can register themselves.
	_ "github.com/go-go-golems/go-go-goja/modules/database"
	_ "github.com/go-go-golems/go-go-goja/modules/exec"
	_ "github.com/go-go-golems/go-go-goja/modules/fs"
	_ "github.com/go-go-golems/go-go-goja/modules/glazehelp"

	"log"
)

// New creates a fresh goja.Runtime ready for execution with Node-style
// `require()` enabled and all native modules registered.
//
// The returned *goja.Runtime can be used directly with RunString/run etc. The
// second return value is the require.Module instance so that callers can load
// entry-point JavaScript files via req.Require(path).
func New() (*goja.Runtime, *require.RequireModule) {
	return NewWithConfig(DefaultRuntimeConfig())
}

// NewWithOptions creates a runtime like New, but allows callers to customize
// the goja_nodejs/require registry (for example, to install a custom loader).
func NewWithOptions(opts ...require.Option) (*goja.Runtime, *require.RequireModule) {
	return NewWithConfig(DefaultRuntimeConfig(), opts...)
}

// NewWithConfig creates a runtime like New and accepts runtime configuration.
func NewWithConfig(cfg RuntimeConfig, opts ...require.Option) (*goja.Runtime, *require.RequireModule) {
	_ = cfg

	vm := goja.New()

	// Create a registry and register every known Go-backed module.
	reg := require.NewRegistry(opts...)
	modules.EnableAll(reg)

	log.Printf("go-go-goja: native modules enabled")

	// Attach the registry to the runtime. The returned object allows
	// programmatic require() calls from Go.
	reqMod := reg.Enable(vm)

	// Expose the global `console` object so that JS code can use console.log & friends.
	console.Enable(vm)

	return vm, reqMod
}
