package gojaperf_test

// phase3_bindings_bench_test.go — Go -> JS / JS -> Go *binding mechanism* overhead.
//
// Where bench_test.go / phase2_bench_test.go measure runtime lifecycle and value
// conversion, this file answers a single question:
//
//   "How much does it cost to expose a Go object or function to JavaScript,
//    depending on *which* goja bridging mechanism you choose?"
//
// The mechanisms benchmarked (cheapest -> most expensive, roughly):
//
//   1. NativeFuncCall   — vm.Set("add", func(FunctionCall) Value{...})  (no reflection)
//   2. WrappedFunc      — vm.Set("add", func(a,b int) int{...})          (reflect.Call path)
//   3. SetExport        — modules.SetExport(exports, ..., func)          (same reflect path, nested)
//   4. ReflectStructMethod — vm.Set("svc", &GoStruct{}); svc.add(a,b)    (method lookup + reflect)
//   5. MapOfFuncs       — vm.Set("m", map[string]interface{}{"add": fn}) (live map view)
//   6. ReflectStructField — vm.Set("svc", &GoStruct{...}); svc.Field      (field read + reflect)
//   7. GoMapSimple      — vm.Set("m", map[string]interface{}{"v": 42})   (objectGoMapSimple view)
//   8. DynamicObject    — vm.NewDynamicObject(handler)                   (Get/Set/Has/Delete/Keys)
//   9. DynamicArray     — vm.NewDynamicArray(handler)                    (Len/Get/Set/SetLen)
//  10. Proxy            — vm.NewProxy(target, &ProxyTrapConfig{...})     (ES6 traps, MANUAL dispatch)
//  11. ProxyApply       — Proxy on a callable target with an Apply trap  (call through a proxy)
//
// Each sub-benchmark does the SAME logical work (return 42) so the numbers are
// directly comparable. Run:
//
//   go test ./perf/goja -run '^$' -bench 'Binding' -benchmem -count=5

import (
	"testing"

	"github.com/dop251/goja"

	"github.com/go-go-golems/go-go-goja/modules"
)

// -----------------------------------------------------------------------------
// Shared Go-side values used across benchmarks. All compute / return 42 so the
// result-check is identical regardless of mechanism.
// -----------------------------------------------------------------------------

// addReflect is an *arbitrary* Go func. goja wraps it with reflection
// (Runtime.wrapReflectFunc in runtime.go): every call reflects each argument
// into the target type, calls the func via reflect, then wraps the result.
func addReflect(a, b int) int { return a + b }

// nativeAdder returns the *lowest-level* goja callable for a given runtime: it
// receives the raw call frame and returns a Value. There is NO per-call
// reflection; the only cost is argument unboxing and the vmCall trampoline in
// func.go. Creating a Value requires a *Runtime, hence the factory.
func nativeAdder(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		a := call.Argument(0).ToInteger()
		b := call.Argument(1).ToInteger()
		return vm.ToValue(a + b)
	}
}

// calcService is a Go struct whose *methods* are reachable from JS once the
// struct is published via vm.Set. Method dispatch goes through objectGoReflect
// (a method-name -> reflect.Method index map) and then reflect.Call.
type calcService struct {
	Field int
}

func (c *calcService) Add(a, b int) int { return a + b }

// dynSvc implements goja.DynamicObject: a tiny manual-dispatch object with
// Get/Set/Has/Delete/Keys. Cheaper than a Proxy because the dispatch surface is
// a fixed 5-method Go interface rather than the full ES6 proxy trap matrix.
type dynSvc struct {
	store map[string]goja.Value
}

func (d *dynSvc) Get(key string) goja.Value { return d.store[key] }
func (d *dynSvc) Set(key string, val goja.Value) bool {
	d.store[key] = val
	return true
}
func (d *dynSvc) Has(key string) bool { _, ok := d.store[key]; return ok }
func (d *dynSvc) Delete(key string) bool {
	delete(d.store, key)
	return true
}
func (d *dynSvc) Keys() []string {
	out := make([]string, 0, len(d.store))
	for k := range d.store {
		out = append(out, k)
	}
	return out
}

// dynArr implements goja.DynamicArray.
type dynArr struct {
	data []goja.Value
}

func (a *dynArr) Len() int               { return len(a.data) }
func (a *dynArr) Get(idx int) goja.Value { return a.data[idx] }
func (a *dynArr) Set(idx int, v goja.Value) bool {
	if idx >= len(a.data) {
		return false
	}
	a.data[idx] = v
	return true
}
func (a *dynArr) SetLen(n int) bool { return false } // not exercised in read benchmarks

// -----------------------------------------------------------------------------
// BenchmarkBindingFunctionCall — JS invokes a Go function exposed N ways.
// -----------------------------------------------------------------------------

func BenchmarkBindingFunctionCall(b *testing.B) {
	silenceBenchLogs()

	const addSrc = "add(40, 2)"
	prgNative := compile(b, "fn-native.js", addSrc)
	prgWrapped := compile(b, "fn-wrapped.js", addSrc)
	prgSetExport := compile(b, "fn-setexport.js", "bench.add(40, 2)")
	prgStructMethod := compile(b, "fn-struct.js", "svc.Add(40, 2)")
	prgMapFuncs := compile(b, "fn-map.js", "m.add(40, 2)")
	prgProxyApply := compile(b, "fn-proxy-apply.js", "proxy(40, 2)")

	b.ReportAllocs()

	// 1) Native func — the floor (no reflection on the call path).
	b.Run("NativeFunc", func(b *testing.B) {
		vm := goja.New()
		_ = vm.Set("add", nativeAdder(vm))
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prgNative)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("got %d", got)
			}
		}
	})

	// 2) Wrapped arbitrary func — full reflect.Call argument marshalling.
	b.Run("WrappedFunc", func(b *testing.B) {
		vm := goja.New()
		_ = vm.Set("add", addReflect)
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prgWrapped)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("got %d", got)
			}
		}
	})

	// 3) modules.SetExport — same reflect path as (2), just nested one level.
	b.Run("SetExport", func(b *testing.B) {
		vm := goja.New()
		exports := vm.NewObject()
		modules.SetExport(exports, "bench", "add", addReflect)
		_ = vm.Set("bench", exports)
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prgSetExport)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("got %d", got)
			}
		}
	})

	// 4) Reflect struct method — svc.Add(a,b) via objectGoReflect.
	b.Run("ReflectStructMethod", func(b *testing.B) {
		vm := goja.New()
		_ = vm.Set("svc", &calcService{})
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prgStructMethod)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("got %d", got)
			}
		}
	})

	// 5) Map of funcs — live map[string]interface{} view (objectGoMapSimple),
	// value is itself a wrapped func.
	b.Run("MapOfFuncs", func(b *testing.B) {
		vm := goja.New()
		_ = vm.Set("m", map[string]interface{}{"add": addReflect})
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prgMapFuncs)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("got %d", got)
			}
		}
	})

	// 11) Proxy Apply — call a callable target through an Apply trap. The trap
	// itself adds one Go-callback hop on top of the underlying native call.
	b.Run("ProxyApply", func(b *testing.B) {
		vm := goja.New()
		adder := nativeAdder(vm)
		callableTarget := vm.ToValue(adder).ToObject(vm) // *Object backed by a native func
		proxy := vm.NewProxy(callableTarget, &goja.ProxyTrapConfig{
			Apply: func(target *goja.Object, this goja.Value, args []goja.Value) goja.Value {
				return adder(goja.FunctionCall{This: this, Arguments: args})
			},
		})
		_ = vm.Set("proxy", proxy)
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prgProxyApply)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("got %d", got)
			}
		}
	})
}

// -----------------------------------------------------------------------------
// BenchmarkBindingPropertyGet — JS reads a single property off a Go-backed
// object exposed N ways. This is where Proxy vs DynamicObject vs reflect diverge
// most sharply, because *every* read routes through the bridging mechanism.
// -----------------------------------------------------------------------------

func BenchmarkBindingPropertyGet(b *testing.B) {
	silenceBenchLogs()

	getSrc := "obj.v"
	prgPlain := compile(b, "get-plain.js", getSrc)
	// Reflect struct fields are exposed under their exported Go name ("Field"),
	// so this program differs from the "obj.v" used by the other mechanisms.
	prgReflectField := compile(b, "get-reflect-field.js", "obj.Field")
	prgMap := compile(b, "get-map.js", getSrc)
	prgDyn := compile(b, "get-dyn.js", getSrc)
	prgProxy := compile(b, "get-proxy.js", getSrc)
	prgPlainLit := compile(b, "get-plain-lit.js", "var obj = ({v: 42}); obj.v")

	b.ReportAllocs()

	// 0) Plain JS object literal baseline — re-creates the object each iteration.
	b.Run("PlainObject_Literal", func(b *testing.B) {
		vm := goja.New()
		for i := 0; i < b.N; i++ {
			if _, err := vm.RunProgram(prgPlainLit); err != nil {
				b.Fatal(err)
			}
		}
	})

	// Plain JS object, pre-created, reused (isolates the read cost).
	b.Run("PlainObject_Reuse", func(b *testing.B) {
		vm := goja.New()
		obj := vm.NewObject()
		_ = obj.Set("v", 42)
		_ = vm.Set("obj", obj)
		for i := 0; i < b.N; i++ {
			if _, err := vm.RunProgram(prgPlain); err != nil {
				b.Fatal(err)
			}
		}
	})

	// 6) Reflect struct field read.
	b.Run("ReflectStructField", func(b *testing.B) {
		vm := goja.New()
		_ = vm.Set("obj", &calcService{Field: 42})
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prgReflectField)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("got %d", got)
			}
		}
	})

	// 7) map[string]interface{} live view.
	b.Run("GoMapSimple", func(b *testing.B) {
		vm := goja.New()
		_ = vm.Set("obj", map[string]interface{}{"v": 42})
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prgMap)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("got %d", got)
			}
		}
	})

	// 8) DynamicObject — Get trap.
	b.Run("DynamicObject", func(b *testing.B) {
		vm := goja.New()
		d := &dynSvc{store: map[string]goja.Value{"v": vm.ToValue(42)}}
		_ = vm.Set("obj", vm.NewDynamicObject(d))
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prgDyn)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("got %d", got)
			}
		}
	})

	// 10) ES6 Proxy — Get trap (the "Go proxy object with manual dispatch").
	// The trap does a minimal string switch + returns a cached Value, so this
	// measures the proxy MACHINERY overhead plus a realistic dispatch hop.
	b.Run("Proxy_GetTrap", func(b *testing.B) {
		vm := goja.New()
		v42 := vm.ToValue(42)
		target := vm.NewObject()
		handler := &goja.ProxyTrapConfig{
			Get: func(target *goja.Object, property string, receiver goja.Value) goja.Value {
				switch property {
				case "v":
					return v42
				}
				return goja.Undefined()
			},
			Has: func(target *goja.Object, property string) bool { return property == "v" },
		}
		_ = vm.Set("obj", vm.NewProxy(target, handler))
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prgProxy)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("got %d", got)
			}
		}
	})
}

// -----------------------------------------------------------------------------
// BenchmarkBindingArrayIndexGet — JS reads arr[i] from a DynamicArray.
// -----------------------------------------------------------------------------

func BenchmarkBindingArrayIndexGet(b *testing.B) {
	silenceBenchLogs()

	const n = 16
	prg := compile(b, "arr-get.js", "arr[8]")

	b.Run("DynamicArray", func(b *testing.B) {
		vm := goja.New()
		data := make([]goja.Value, n)
		for i := range data {
			data[i] = vm.ToValue(i * 2)
		}
		_ = vm.Set("arr", vm.NewDynamicArray(&dynArr{data: data}))
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prg)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 16 {
				b.Fatalf("got %d", got)
			}
		}
	})
}

// -----------------------------------------------------------------------------
// BenchmarkProxyDispatchDetail — isolates the components of Proxy overhead so we
// can tell "machinery cost" apart from "dispatch logic cost". All variants read
// obj.v, so only the trap body changes.
// -----------------------------------------------------------------------------

func BenchmarkProxyDispatchDetail(b *testing.B) {
	silenceBenchLogs()
	b.ReportAllocs()

	prg := compile(b, "proxy-detail.js", "obj.v")

	// Variant A: empty Get trap returns undefined (pure machinery cost, no hit).
	b.Run("GetTrap_Miss_Undefined", func(b *testing.B) {
		vm := goja.New()
		target := vm.NewObject()
		handler := &goja.ProxyTrapConfig{
			Get: func(target *goja.Object, property string, receiver goja.Value) goja.Value {
				return goja.Undefined()
			},
		}
		_ = vm.Set("obj", vm.NewProxy(target, handler))
		for i := 0; i < b.N; i++ {
			if _, err := vm.RunProgram(prg); err != nil {
				b.Fatal(err)
			}
		}
	})

	// Variant B: Get trap does a string switch and returns a cached value on hit.
	b.Run("GetTrap_Switch_Hit", func(b *testing.B) {
		vm := goja.New()
		v42 := vm.ToValue(42)
		target := vm.NewObject()
		handler := &goja.ProxyTrapConfig{
			Get: func(target *goja.Object, property string, receiver goja.Value) goja.Value {
				switch property {
				case "v":
					return v42
				}
				return goja.Undefined()
			},
		}
		_ = vm.Set("obj", vm.NewProxy(target, handler))
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prg)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("got %d", got)
			}
		}
	})

	// Variant C: Get trap does a map lookup (the typical "manual dispatch" table)
	// and returns a cached value on hit. This is what a real proxy-backed module
	// would do: map[string]Value built once at registration time.
	b.Run("GetTrap_Map_Hit", func(b *testing.B) {
		vm := goja.New()
		v42 := vm.ToValue(42)
		target := vm.NewObject()
		table := map[string]goja.Value{"v": v42}
		handler := &goja.ProxyTrapConfig{
			Get: func(target *goja.Object, property string, receiver goja.Value) goja.Value {
				return table[property]
			},
			Has: func(target *goja.Object, property string) bool {
				_, ok := table[property]
				return ok
			},
		}
		_ = vm.Set("obj", vm.NewProxy(target, handler))
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prg)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("got %d", got)
			}
		}
	})
}
