package testprovider

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func Register(registry *providerapi.Registry) error {
	return registry.Package("fixture",
		providerapi.Module{
			Name:        "hello",
			DefaultAs:   "hello",
			Description: "Fixture module used by xgoja tests",
			New: func(providerapi.ModuleContext) (require.ModuleLoader, error) {
				return func(vm *goja.Runtime, module *goja.Object) {
					exports := module.Get("exports").(*goja.Object)
					_ = exports.Set("greet", func(name string) string { return "hello " + name })
				}, nil
			},
		},
		providerapi.VerbSource{Name: "verbs", Root: "verbs"},
	)
}
