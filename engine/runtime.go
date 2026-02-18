package engine

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"

	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/calllog"
	// Blank imports ensure module init() functions run so they can register themselves.
	_ "github.com/go-go-golems/go-go-goja/modules/database"
	_ "github.com/go-go-golems/go-go-goja/modules/exec"
	_ "github.com/go-go-golems/go-go-goja/modules/fs"
	_ "github.com/go-go-golems/go-go-goja/modules/glazehelp"

	"log"
	goRuntime "runtime"
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

// NewWithConfig creates a runtime like New, but allows callers to configure
// call logging through the go-go-goja API.
func NewWithConfig(cfg RuntimeConfig, opts ...require.Option) (*goja.Runtime, *require.RequireModule) {
	openOpts := []Option{
		WithRequireOptions(opts...),
	}
	if cfg.CallLogEnabled {
		openOpts = append(openOpts, WithCallLog(cfg.CallLogPath))
	} else {
		openOpts = append(openOpts, WithCallLogDisabled())
	}
	return Open(openOpts...)
}

// Open creates a fresh runtime using option-driven configuration.
func Open(opts ...Option) (*goja.Runtime, *require.RequireModule) {
	settings := defaultOpenSettings()
	for _, opt := range opts {
		if opt != nil {
			opt(&settings)
		}
	}

	vm := goja.New()
	configureRuntimeCallLog(vm, settings)

	// Create a registry and register every known Go-backed module.
	reg := require.NewRegistry(settings.requireOptions...)
	modules.EnableAll(reg)

	log.Printf("go-go-goja: native modules enabled")

	// Attach the registry to the runtime. The returned object allows
	// programmatic require() calls from Go.
	reqMod := reg.Enable(vm)

	// Expose the global `console` object so that JS code can use console.log & friends.
	console.Enable(vm)

	return vm, reqMod
}

func configureRuntimeCallLog(vm *goja.Runtime, settings openSettings) {
	switch settings.callLogMode {
	case callLogModeEnabled:
		path := settings.callLogPath
		if path == "" {
			path = calllog.DefaultPath()
		}
		logger, err := calllog.New(path)
		if err != nil {
			log.Printf("go-go-goja: call log setup failed: %v", err)
			calllog.DisableRuntimeLogger(vm)
			break
		}
		calllog.BindOwnedRuntimeLogger(vm, logger)
	case callLogModeDisabled:
		fallthrough
	default:
		calllog.DisableRuntimeLogger(vm)
	}

	goRuntime.SetFinalizer(vm, func(r *goja.Runtime) {
		calllog.ReleaseRuntimeLogger(r)
	})
}
