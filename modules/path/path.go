package pathmod

import (
	"path/filepath"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

type m struct{}

var _ modules.NativeModule = (*m)(nil)
var _ modules.TypeScriptDeclarer = (*m)(nil)

func (m) Name() string { return "path" }

func (m) Doc() string {
	return `The path module provides host-platform filepath helpers: join, resolve, dirname, basename, extname, isAbsolute, relative, separator, and delimiter.`
}

func (m) TypeScriptModule() *spec.Module {
	return &spec.Module{Name: "path", Functions: []spec.Function{
		{Name: "join", Params: []spec.Param{{Name: "parts", Type: spec.String(), Variadic: true}}, Returns: spec.String()},
		{Name: "resolve", Params: []spec.Param{{Name: "parts", Type: spec.String(), Variadic: true}}, Returns: spec.String()},
		{Name: "dirname", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.String()},
		{Name: "basename", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.String()},
		{Name: "extname", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.String()},
		{Name: "isAbsolute", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Boolean()},
		{Name: "relative", Params: []spec.Param{{Name: "from", Type: spec.String()}, {Name: "to", Type: spec.String()}}, Returns: spec.String()},
	}}
}

func (mod m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	modules.SetExport(exports, mod.Name(), "join", func(parts ...string) string { return filepath.Join(parts...) })
	modules.SetExport(exports, mod.Name(), "resolve", func(parts ...string) (string, error) {
		if len(parts) == 0 {
			return filepath.Abs(".")
		}
		return filepath.Abs(filepath.Join(parts...))
	})
	modules.SetExport(exports, mod.Name(), "dirname", filepath.Dir)
	modules.SetExport(exports, mod.Name(), "basename", filepath.Base)
	modules.SetExport(exports, mod.Name(), "extname", filepath.Ext)
	modules.SetExport(exports, mod.Name(), "isAbsolute", filepath.IsAbs)
	modules.SetExport(exports, mod.Name(), "relative", filepath.Rel)
	_ = exports.Set("separator", string(filepath.Separator))
	_ = exports.Set("delimiter", string(filepath.ListSeparator))
}

func init() { modules.Register(&m{}) }
