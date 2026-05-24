package testprovider

import (
	"context"
	"embed"
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

//go:embed verbs/*.js
var verbsFS embed.FS

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
		providerapi.Module{
			Name:        "owner-check",
			DefaultAs:   "owner-check",
			Description: "Fixture module that requires xgoja runtime owner bindings",
			New: func(providerapi.ModuleContext) (require.ModuleLoader, error) {
				return func(vm *goja.Runtime, module *goja.Object) {
					exports := module.Get("exports").(*goja.Object)
					_ = exports.Set("hasOwner", func() bool {
						bindings, ok := runtimebridge.Lookup(vm)
						return ok && bindings.Owner != nil && bindings.Loop != nil
					})
					_ = exports.Set("pingAsync", func() goja.Value {
						promise, resolve, reject := vm.NewPromise()
						bindings, ok := runtimebridge.Lookup(vm)
						if !ok || bindings.Owner == nil {
							_ = reject(vm.ToValue("missing runtime owner bindings"))
							return vm.ToValue(promise)
						}
						callCtx := runtimebridge.CurrentContext(vm)
						go func() {
							if err := bindings.Owner.Post(callCtx, "owner-check.ping", func(context.Context, *goja.Runtime) {
								_ = resolve("pong")
							}); err != nil {
								_ = bindings.Owner.Post(context.Background(), "owner-check.ping.reject", func(context.Context, *goja.Runtime) {
									_ = reject(vm.ToValue(fmt.Sprintf("post failed: %v", err)))
								})
							}
						}()
						return vm.ToValue(promise)
					})
				}, nil
			},
		},
		providerapi.VerbSource{Name: "verbs", Root: "verbs", FS: verbsFS},
	)
}
