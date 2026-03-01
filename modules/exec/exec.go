package execmod

import (
	"os/exec"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

// m implements a minimal wrapper around os/exec for JavaScript.
type m struct{}

var _ modules.NativeModule = (*m)(nil)
var _ modules.TypeScriptDeclarer = (*m)(nil)

func (m) Name() string { return "exec" }

func (m) TypeScriptModule() *spec.Module {
	return &spec.Module{
		Name: "exec",
		Functions: []spec.Function{
			{
				Name: "run",
				Params: []spec.Param{
					{Name: "cmd", Type: spec.String()},
					{Name: "args", Type: spec.Array(spec.String())},
				},
				Returns: spec.String(),
			},
		},
	}
}

// Doc returns the documentation for the module.
func (m) Doc() string {
	return "The exec module provides a simple way to run external commands."
}

// Loader attaches the exported Go functions to the JS module.exports object.
func (mod m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)

	// run(cmd, args[]) -> string
	modules.SetExport(exports, mod.Name(), "run", func(cmd string, args []string) (string, error) {
		out, err := exec.Command(cmd, args...).CombinedOutput()
		return string(out), err
	})
}

func init() {
	modules.Register(&m{})
}
