package timemod

import (
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

type m struct{}

var _ modules.NativeModule = (*m)(nil)
var _ modules.TypeScriptDeclarer = (*m)(nil)

func (m) Name() string { return "time" }

func (m) Doc() string {
	return `
The time module provides monotonic timing helpers for JavaScript-side
performance measurements.

Functions:
  now(): Returns elapsed milliseconds since module initialization.
  since(startMs): Returns elapsed milliseconds since a previous now() value.
`
}

func (m) TypeScriptModule() *spec.Module {
	return &spec.Module{
		Name: "time",
		Functions: []spec.Function{
			{
				Name:    "now",
				Returns: spec.Number(),
			},
			{
				Name: "since",
				Params: []spec.Param{
					{Name: "startMs", Type: spec.Number()},
				},
				Returns: spec.Number(),
			},
		},
	}
}

func (mod *m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	start := time.Now()
	exports := moduleObj.Get("exports").(*goja.Object)
	modules.SetExport(exports, mod.Name(), "now", func() float64 {
		return float64(time.Since(start).Nanoseconds()) / 1e6
	})
	modules.SetExport(exports, mod.Name(), "since", func(startMs float64) float64 {
		return float64(time.Since(start).Nanoseconds())/1e6 - startMs
	})
}

func init() {
	modules.Register(&m{})
}
