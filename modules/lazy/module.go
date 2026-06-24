package lazy

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
)

// module is the require("lazy") NativeModule. It exposes JS builders that
// construct lazy Proxies from a { keys, get } / { length, get } configuration.
type module struct{}

var _ modules.NativeModule = (*module)(nil)
var _ modules.TypeScriptDeclarer = (*module)(nil)

func (module) Name() string { return "lazy" }

func (module) Doc() string {
	return `lazy — lazily-materialised objects backed by Go via goja Proxy.

Builders:
  lazy.map({ keys: string[], get: (k) => any, mutable?: boolean }) -> object
  lazy.array({ length: number, get: (i) => any }) -> array-like
  lazy.record({ keys: () => string[], get: (k) => any, has?: (k) => boolean, mutable?: boolean }) -> object

Values are produced on demand the first time JS reads a key/index, then cached.
Object.keys() / for..in enumerate keys without calling get. JSON.stringify and
spread/destructure read values (unavoidably). See the GOJA-LAZY-DS design doc.`
}

// Loader attaches the builders to module.exports.
func (mod module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)

	// lazy.map({ keys, get, mutable? }) -> Proxy
	modules.SetExport(exports, mod.Name(), "map", func(call goja.FunctionCall) goja.Value {
		opts := optsObject(vm, call.Argument(0))
		keys := toStringSlice(vm, opts.Get("keys"))
		getter := callable(vm, opts.Get("get"), "lazy.map: get")
		mutable := boolField(opts.Get("mutable"))
		m := &Map{
			Keys:     keys,
			Provider: jsProvider(vm, getter),
			Mutable:  mutable,
		}
		return m.ToValue(vm)
	})

	// lazy.array({ length, get }) -> Proxy
	modules.SetExport(exports, mod.Name(), "array", func(call goja.FunctionCall) goja.Value {
		opts := optsObject(vm, call.Argument(0))
		length := int(fieldInt(opts.Get("length"), 0))
		if length < 0 {
			panic(vm.NewTypeError("lazy.array: length must be >= 0"))
		}
		getter := callable(vm, opts.Get("get"), "lazy.array: get")
		a := &Array{
			Len:      length,
			Provider: jsIndexProvider(vm, getter),
		}
		return a.ToValue(vm)
	})

	// lazy.record({ keys, get, has?, mutable? }) -> Proxy
	modules.SetExport(exports, mod.Name(), "record", func(call goja.FunctionCall) goja.Value {
		opts := optsObject(vm, call.Argument(0))
		keysFn := callable(vm, opts.Get("keys"), "lazy.record: keys")
		getter := callable(vm, opts.Get("get"), "lazy.record: get")
		mutable := boolField(opts.Get("mutable"))
		var hasFn goja.Callable
		if h := opts.Get("has"); !goja.IsUndefined(h) && !goja.IsNull(h) {
			hasFn = callable(vm, h, "lazy.record: has")
		}
		r := &Record{
			Keys:     jsKeysProvider(vm, keysFn),
			Provider: jsProvider(vm, getter),
			Mutable:  mutable,
		}
		if hasFn != nil {
			r.Has = jsHasProvider(vm, hasFn)
		}
		return r.ToValue(vm)
	})
}

// --- JS↔Go provider adapters -------------------------------------------------

// jsProvider adapts a JS (key)->any getter into a Go func(string) any. The
// returned value is already a goja.Value, so toValueAny returns it as-is,
// preserving object identity across reads.
func jsProvider(vm *goja.Runtime, getter goja.Callable) func(string) any {
	return func(key string) any {
		v, err := getter(goja.Undefined(), vm.ToValue(key))
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return v
	}
}

func jsIndexProvider(vm *goja.Runtime, getter goja.Callable) func(int) any {
	return func(i int) any {
		v, err := getter(goja.Undefined(), vm.ToValue(i))
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return v
	}
}

func jsKeysProvider(vm *goja.Runtime, getter goja.Callable) func() []string {
	return func() []string {
		v, err := getter(goja.Undefined())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return toStringSlice(vm, v)
	}
}

func jsHasProvider(vm *goja.Runtime, getter goja.Callable) func(string) bool {
	return func(key string) bool {
		v, err := getter(goja.Undefined(), vm.ToValue(key))
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return v.ToBoolean()
	}
}

// --- option decoding helpers -------------------------------------------------

func optsObject(vm *goja.Runtime, v goja.Value) *goja.Object {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		panic(vm.NewTypeError("lazy: expected an options object"))
	}
	obj := v.ToObject(vm)
	if obj == nil {
		panic(vm.NewTypeError("lazy: expected an options object"))
	}
	return obj
}

func callable(vm *goja.Runtime, v goja.Value, what string) goja.Callable {
	fn, ok := goja.AssertFunction(v)
	if !ok {
		panic(vm.NewTypeError("%s must be a function", what))
	}
	return fn
}

func toStringSlice(vm *goja.Runtime, v goja.Value) []string {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	obj := v.ToObject(vm)
	length := int(obj.Get("length").ToInteger())
	out := make([]string, 0, length)
	for i := 0; i < length; i++ {
		out = append(out, obj.Get(fmt.Sprintf("%d", i)).String())
	}
	return out
}

func boolField(v goja.Value) bool {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return false
	}
	return v.ToBoolean()
}

func fieldInt(v goja.Value, def int64) int64 {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return def
	}
	return v.ToInteger()
}

func init() {
	modules.Register(module{})
}
