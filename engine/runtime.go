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

	"log"
)

// New creates a fresh goja.Runtime ready for execution with Node-style
// `require()` enabled and all native modules registered.
//
// The returned *goja.Runtime can be used directly with RunString/run etc. The
// second return value is the require.Module instance so that callers can load
// entry-point JavaScript files via req.Require(path).
func New() (*goja.Runtime, *require.RequireModule) {
	vm := goja.New()

	// Create a registry and register every known Go-backed module.
	reg := require.NewRegistry()
	modules.EnableAll(reg)

	log.Printf("go-go-goja: native modules enabled")

	// Attach the registry to the runtime. The returned object allows
	// programmatic require() calls from Go.
	reqMod := reg.Enable(vm)

	// Expose the global `console` object so that JS code can use console.log & friends.
	console.Enable(vm)

	return vm, reqMod
}
