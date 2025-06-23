package fs

import (
	"os"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
)

// m is the concrete implementation of the fs module.
// We use an empty struct because no internal state is required.
// The compile-time assertion guarantees that *m implements NativeModule.

type m struct{}

var _ modules.NativeModule = (*m)(nil)

func (m) Name() string { return "fs" }

// Loader attaches the exported Go functions to the JS `exports` object.
func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)

	// readFileSync(path) -> string | throws
	_ = exports.Set("readFileSync", func(path string) (string, error) {
		b, err := os.ReadFile(path)
		return string(b), err
	})

	// writeFileSync(path, data) -> void | throws
	_ = exports.Set("writeFileSync", func(path, data string) error {
		return os.WriteFile(path, []byte(data), 0o644)
	})
}

// Each module registers itself during package initialization.
func init() {
	modules.Register(&m{})
}
