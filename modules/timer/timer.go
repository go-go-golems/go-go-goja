package timermod

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

type m struct{}

var _ modules.NativeModule = (*m)(nil)
var _ modules.TypeScriptDeclarer = (*m)(nil)

func (m) Name() string { return "timer" }

func (m) Doc() string {
	return `
The timer module provides Promise-based timing helpers.

Functions:
  sleep(ms): Returns a Promise that resolves after the provided duration.
`
}

func (m) TypeScriptModule() *spec.Module {
	return &spec.Module{
		Name:        "timer",
		Description: "Promise-based timing helpers.",
		Functions: []spec.Function{
			{
				Name: "sleep",
				Params: []spec.Param{
					{Name: "ms", Type: spec.Number(), Description: "Duration in milliseconds."},
				},
				Returns: spec.Named("Promise<void>"),
			},
		},
	}
}

func (mod m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)

	modules.SetExport(exports, mod.Name(), "sleep", func(ms int64) goja.Value {
		promise, resolve, reject := vm.NewPromise()

		runtimeServices, ok := runtimebridge.Lookup(vm)
		if !ok || runtimeServices.Owner == nil {
			panic(vm.NewGoError(fmt.Errorf("timer module requires runtime services")))
		}

		callCtx := runtimebridge.CurrentOwnerContext(vm)
		runtimeCtx := runtimeServices.Lifetime()
		go func() {
			if ms < 0 {
				_ = runtimeServices.PostWithCustomContext(callCtx, "timer.sleep.reject", func(context.Context, *goja.Runtime) {
					_ = reject(vm.ToValue("timer.sleep: duration must be >= 0"))
				})
				return
			}

			timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
			defer timer.Stop()

			select {
			case <-callCtx.Done():
				return
			case <-runtimeCtx.Done():
				return
			case <-timer.C:
				_ = runtimeServices.PostWithCustomContext(callCtx, "timer.sleep.resolve", func(context.Context, *goja.Runtime) {
					_ = resolve(goja.Undefined())
				})
			}
		}()

		return vm.ToValue(promise)
	})
}

func init() {
	modules.Register(&m{})
}
