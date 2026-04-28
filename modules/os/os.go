package osmod

import (
	"os"
	"runtime"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

type m struct{ name string }

var _ modules.NativeModule = (*m)(nil)
var _ modules.TypeScriptDeclarer = (*m)(nil)

func (m m) Name() string {
	if m.name != "" {
		return m.name
	}
	return "os"
}
func (m m) Doc() string { return `The os module exposes host operating-system helpers.` }
func (m m) TypeScriptModule() *spec.Module {
	return &spec.Module{Name: m.Name(), Functions: []spec.Function{
		{Name: "homedir", Returns: spec.String()}, {Name: "tmpdir", Returns: spec.String()},
		{Name: "platform", Returns: spec.String()}, {Name: "arch", Returns: spec.String()},
		{Name: "hostname", Returns: spec.String()}, {Name: "release", Returns: spec.String()},
		{Name: "type", Returns: spec.String()}, {Name: "cpus", Returns: spec.Array(spec.Object())},
	}}
}

func (mod m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	modules.SetExport(exports, mod.Name(), "homedir", os.UserHomeDir)
	modules.SetExport(exports, mod.Name(), "tmpdir", os.TempDir)
	modules.SetExport(exports, mod.Name(), "platform", func() string { return runtime.GOOS })
	modules.SetExport(exports, mod.Name(), "arch", func() string { return runtime.GOARCH })
	modules.SetExport(exports, mod.Name(), "hostname", os.Hostname)
	modules.SetExport(exports, mod.Name(), "release", func() string { return runtime.GOOS })
	modules.SetExport(exports, mod.Name(), "type", func() string { return runtime.GOOS })
	modules.SetExport(exports, mod.Name(), "cpus", func() []map[string]any {
		out := make([]map[string]any, runtime.NumCPU())
		for i := range out {
			out[i] = map[string]any{"model": "go runtime", "speed": 0, "times": map[string]int{"user": 0, "nice": 0, "sys": 0, "idle": 0, "irq": 0}}
		}
		return out
	})
	_ = exports.Set("EOL", "\n")
	if runtime.GOOS == "windows" {
		_ = exports.Set("EOL", "\r\n")
	}
}

func init() {
	modules.Register(&m{name: "os"})
	modules.Register(&m{name: "node:os"})
}
