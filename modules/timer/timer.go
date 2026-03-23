package timermod

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
)

type m struct{}

var _ modules.NativeModule = (*m)(nil)

func (m) Name() string { return "timer" }

func (m) Doc() string {
	return `
The timer module provides Promise-based timing helpers.

Functions:
  sleep(ms): Returns a Promise that resolves after the provided duration.
`
}

func (mod m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)

	modules.SetExport(exports, mod.Name(), "sleep", func(ms int64) goja.Value {
		promise, resolve, reject := vm.NewPromise()

		bindings, ok := runtimebridge.Lookup(vm)
		if !ok || bindings.Owner == nil {
			panic(vm.NewGoError(fmt.Errorf("timer module requires runtime owner bindings")))
		}

		go func() {
			if ms < 0 {
				_ = bindings.Owner.Post(bindings.Context, "timer.sleep.reject", func(context.Context, *goja.Runtime) {
					_ = reject(vm.ToValue("timer.sleep: duration must be >= 0"))
				})
				return
			}

			timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
			defer timer.Stop()

			select {
			case <-bindings.Context.Done():
				return
			case <-timer.C:
				_ = bindings.Owner.Post(bindings.Context, "timer.sleep.resolve", func(context.Context, *goja.Runtime) {
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
