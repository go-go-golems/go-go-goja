package execmod

import (
	"os/exec"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
)

// m implements a minimal wrapper around os/exec for JavaScript.
type m struct{}

var _ modules.NativeModule = (*m)(nil)

func (m) Name() string { return "exec" }

// Doc returns the documentation for the module.
func (m) Doc() string {
	return "The exec module provides a simple way to run external commands."
}

// Loader attaches the exported Go functions to the JS module.exports object.
func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)

	// run(cmd, args[]) -> string
	_ = exports.Set("run", func(cmd string, args []string) (string, error) {
		out, err := exec.Command(cmd, args...).CombinedOutput()
		return string(out), err
	})
}

func init() {
	modules.Register(&m{})
}
