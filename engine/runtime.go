package engine

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"

	// Blank imports ensure module init() functions run so they can register themselves.
	_ "github.com/go-go-golems/go-go-goja/modules/database"
	_ "github.com/go-go-golems/go-go-goja/modules/exec"
	_ "github.com/go-go-golems/go-go-goja/modules/fs"
	_ "github.com/go-go-golems/go-go-goja/modules/glazehelp"
)

// New creates a fresh goja.Runtime ready for execution with Node-style
// `require()` enabled and all native modules registered.
//
// The returned *goja.Runtime can be used directly with RunString/run etc. The
// second return value is the require.Module instance so that callers can load
// entry-point JavaScript files via req.Require(path).
func New() (*goja.Runtime, *require.RequireModule) {
	return Open()
}

// NewWithOptions creates a runtime like New, but allows callers to customize
// the goja_nodejs/require registry (for example, to install a custom loader).
func NewWithOptions(opts ...require.Option) (*goja.Runtime, *require.RequireModule) {
	return Open(WithRequireOptions(opts...))
}

// Open creates a fresh runtime using option-driven configuration.
func Open(opts ...Option) (*goja.Runtime, *require.RequireModule) {
	return NewFactory(opts...).NewRuntime()
}
