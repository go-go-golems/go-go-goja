//go:build ignore
// This is a read-only SNAPSHOT of go-go-goja/perf/goja/phase4_lazy_bench_test.go,
// kept in the ticket for reproducibility. The //go:build ignore tag excludes it
// from `go test ./...` so it does not need the helpers (silenceBenchLogs,
// compile) defined in the live perf/goja package.

package gojaperf_test

// phase4_lazy_bench_test.go — performance of lazily-materialised Go data
// structures exposed to JavaScript via goja Proxy (the modules/lazy module).
//
// The motivating question, carried over from the GOJA-LAZY-DS design doc:
// the lazy module's flag-only GetOwnPropertyDescriptor makes Object.keys() and
// for...in enumeration free of *provider* work. How much does that enumeration
// actually cost, and how does it compare to (a) an eager object and (b) a naive
// lazy map whose descriptor materialises every value?
//
// The load-bearing signal here is provider_calls/op, reported via
// b.ReportMetric. It is the direct analogue of allocs/op in the
// binding-mechanism report: a deterministic code-path invariant, identical
// across every sample. Nanoseconds corroborate it; the provider call count is
// the thing that proves laziness.
//
// Run:
//
//   go test ./perf/goja -run '^$' \
//     -bench 'LazyRead|LazyEnumeration|LazyScale|LazyMaterialize' \
//     -benchtime=1s -count=5 -benchmem | tee raw.txt

import (
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/dop251/goja"

	"github.com/go-go-golems/go-go-goja/modules/lazy"
)

// lazyBenchProvider returns a provider that counts calls and serves pre-built
// values so per-call allocation noise is minimal (the measurement isolates
// trap/dispatch overhead, not string formatting).
func lazyBenchProvider(vals []string) (func(string) any, *int64) {
	var n int64
	p := func(key string) any {
		atomic.AddInt64(&n, 1)
		i, _ := strconv.Atoi(key[1:]) // keys are "k000".."k999"
		if i < 0 || i >= len(vals) {
			i = i % len(vals)
		}
		return vals[i]
	}
	return p, &n
}

// lazyBenchKeys returns ["k000", "k001", ...] of length n (3-digit zero-padded,
// so parsing in the provider is a single strconv.Atoi).
func lazyBenchKeys(n int) []string {
	ks := make([]string, n)
	for i := 0; i < n; i++ {
		ks[i] = "k" + pad3(i)
	}
	return ks
}

func lazyBenchVals(n int) []string {
	vs := make([]string, n)
	for i := 0; i < n; i++ {
		vs[i] = "val-" + pad3(i)
	}
	return vs
}

func pad3(i int) string {
	s := strconv.Itoa(i)
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}

// reportProviderPerOp divides the total provider calls by b.N.
func reportProviderPerOp(b *testing.B, n *int64) {
	b.ReportMetric(float64(atomic.LoadInt64(n))/float64(b.N), "provider/op")
}

// strAnySlice converts []string to []any for vm.NewArray.
func strAnySlice(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

// -----------------------------------------------------------------------------
// BenchmarkLazyRead — a single property read: obj.k000 -> "val-000".
// -----------------------------------------------------------------------------

func BenchmarkLazyRead(b *testing.B) {
	silenceBenchLogs()
	b.ReportAllocs()

	const key = "k000"
	vals := lazyBenchVals(1)
	prg := compile(b, "lazy-read.js", `obj.k000`)

	// Eager pre-built *Object — the read floor.
	b.Run("Eager_Object", func(b *testing.B) {
		vm := goja.New()
		obj := vm.NewObject()
		_ = obj.Set(key, vals[0])
		_ = vm.Set("obj", obj)
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prg)
			if err != nil {
				b.Fatal(err)
			}
			if v.String() != vals[0] {
				b.Fatalf("got %s", v)
			}
		}
	})

	// lazy.Map, steady-state cached read (the realistic hot path): the key is
	// pre-warmed once, so b.N reads all hit the cache. provider/op -> 0.
	b.Run("LazyMap_Cached", func(b *testing.B) {
		vm := goja.New()
		p, cnt := lazyBenchProvider(vals)
		m := &lazy.Map{Keys: []string{key}, Provider: p}
		_ = vm.Set("obj", m.ToValue(vm))
		_, _ = vm.RunProgram(prg) // warm the cache (untimed)
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prg)
			if err != nil {
				b.Fatal(err)
			}
			if v.String() != vals[0] {
				b.Fatalf("got %s", v)
			}
		}
		reportProviderPerOp(b, cnt)
	})

	// Cold read semantics: a Proxy whose Get trap has NO cache, so every read
	// calls the provider and re-converts. provider/op -> 1. This is the cost a
	// lazy read pays the first time, and what a non-caching lazy object pays
	// every time.
	b.Run("LazyMap_Cold_NoCache", func(b *testing.B) {
		vm := goja.New()
		p, cnt := lazyBenchProvider(vals)
		target := vm.NewObject()
		proxy := vm.NewProxy(target, &goja.ProxyTrapConfig{
			Get: func(t *goja.Object, prop string, receiver goja.Value) goja.Value {
				if prop == key {
					return vm.ToValue(p(prop))
				}
				return t.Get(prop)
			},
			Has: func(t *goja.Object, prop string) bool { return prop == key },
		})
		_ = vm.Set("obj", vm.ToValue(proxy))
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prg)
			if err != nil {
				b.Fatal(err)
			}
			if v.String() != vals[0] {
				b.Fatalf("got %s", v)
			}
		}
		reportProviderPerOp(b, cnt)
	})
}

// -----------------------------------------------------------------------------
// BenchmarkLazyEnumeration — Object.keys / for..in over a 100-key object.
// This is the regime where the flag-only descriptor earns its keep.
// -----------------------------------------------------------------------------

func BenchmarkLazyEnumeration(b *testing.B) {
	silenceBenchLogs()
	b.ReportAllocs()

	const n = 100
	keys := lazyBenchKeys(n)
	vals := lazyBenchVals(n)

	prgKeys := compile(b, "lazy-keys.js", `Object.keys(obj)`)
	prgForIn := compile(b, "lazy-forin.js", `for (var k in obj) { void k; }`)

	// Eager pre-built *Object — the enumeration floor.
	b.Run("Eager_Keys", func(b *testing.B) {
		vm := goja.New()
		obj := vm.NewObject()
		for i, k := range keys {
			_ = obj.Set(k, vals[i])
		}
		_ = vm.Set("obj", obj)
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prgKeys)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToObject(vm).Get("length").ToInteger(); got != n {
				b.Fatalf("got %d keys", got)
			}
		}
	})

	// lazy.Map — the module. flag-only descriptor: provider is NEVER called
	// during enumeration. provider/op -> 0. This is the win.
	b.Run("LazyMap_Keys_FlagOnly", func(b *testing.B) {
		vm := goja.New()
		p, cnt := lazyBenchProvider(vals)
		m := &lazy.Map{Keys: keys, Provider: p}
		_ = vm.Set("obj", m.ToValue(vm))
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prgKeys)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToObject(vm).Get("length").ToInteger(); got != n {
				b.Fatalf("got %d keys", got)
			}
		}
		reportProviderPerOp(b, cnt)
	})

	// Naive lazy map: GetOwnPropertyDescriptor builds the value (no cache), so
	// every enumeration materialises all values. provider/op -> n (100). This is
	// the sharp edge the flag-only descriptor exists to avoid.
	b.Run("NaiveLazy_Keys_Materializing", func(b *testing.B) {
		vm := goja.New()
		p, cnt := lazyBenchProvider(vals)
		target := vm.NewObject()
		proxy := vm.NewProxy(target, &goja.ProxyTrapConfig{
			Get: func(t *goja.Object, prop string, receiver goja.Value) goja.Value {
				return vm.ToValue(p(prop))
			},
			Has: func(t *goja.Object, prop string) bool { return true },
			GetOwnPropertyDescriptor: func(t *goja.Object, prop string) goja.PropertyDescriptor {
				return goja.PropertyDescriptor{
					Value:        vm.ToValue(p(prop)),
					Writable:     goja.FLAG_FALSE,
					Configurable: goja.FLAG_TRUE,
					Enumerable:   goja.FLAG_TRUE,
				}
			},
			OwnKeys: func(t *goja.Object) *goja.Object {
				return vm.NewArray(strAnySlice(keys)...)
			},
		})
		_ = vm.Set("obj", vm.ToValue(proxy))
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prgKeys)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToObject(vm).Get("length").ToInteger(); got != n {
				b.Fatalf("got %d keys", got)
			}
		}
		reportProviderPerOp(b, cnt)
	})

	// for..in on lazy.Map — also free of provider work (provider/op -> 0).
	b.Run("LazyMap_ForIn", func(b *testing.B) {
		vm := goja.New()
		p, cnt := lazyBenchProvider(vals)
		m := &lazy.Map{Keys: keys, Provider: p}
		_ = vm.Set("obj", m.ToValue(vm))
		for i := 0; i < b.N; i++ {
			if _, err := vm.RunProgram(prgForIn); err != nil {
				b.Fatal(err)
			}
		}
		reportProviderPerOp(b, cnt)
	})
}

// -----------------------------------------------------------------------------
// BenchmarkLazyScale — Object.keys on lazy.Map as the key count grows. Shows
// the enumeration is O(keys) in trap dispatch and 0 in provider work throughout.
// -----------------------------------------------------------------------------

func BenchmarkLazyScale(b *testing.B) {
	silenceBenchLogs()
	b.ReportAllocs()

	prg := compile(b, "lazy-scale.js", `Object.keys(obj)`)
	for _, n := range []int{10, 100, 1000} {
		keys := lazyBenchKeys(n)
		vals := lazyBenchVals(n)
		b.Run("LazyMap_Keys/N="+strconv.Itoa(n), func(b *testing.B) {
			vm := goja.New()
			p, cnt := lazyBenchProvider(vals)
			m := &lazy.Map{Keys: keys, Provider: p}
			_ = vm.Set("obj", m.ToValue(vm))
			for i := 0; i < b.N; i++ {
				v, err := vm.RunProgram(prg)
				if err != nil {
					b.Fatal(err)
				}
				if got := v.ToObject(vm).Get("length").ToInteger(); got != int64(n) {
					b.Fatalf("got %d keys", got)
				}
			}
			reportProviderPerOp(b, cnt)
		})
	}
}

// -----------------------------------------------------------------------------
// BenchmarkLazyMaterialize — JSON.stringify, which unavoidably reads every
// value. provider/op -> n. The contrast with LazyMap_Keys_FlagOnly (0) is the
// boundary between cheap and inherently materialising operations.
// -----------------------------------------------------------------------------

func BenchmarkLazyMaterialize(b *testing.B) {
	silenceBenchLogs()
	b.ReportAllocs()

	const n = 100
	keys := lazyBenchKeys(n)
	vals := lazyBenchVals(n)
	prg := compile(b, "lazy-stringify.js", `JSON.stringify(obj)`)

	b.Run("Eager_Stringify", func(b *testing.B) {
		vm := goja.New()
		obj := vm.NewObject()
		for i, k := range keys {
			_ = obj.Set(k, vals[i])
		}
		_ = vm.Set("obj", obj)
		for i := 0; i < b.N; i++ {
			if _, err := vm.RunProgram(prg); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LazyMap_Stringify", func(b *testing.B) {
		// Non-caching flag-only lazy proxy: Get calls the provider every time,
		// so the per-call materialisation cost is visible (provider/op -> n).
		// With the module's cache, only the FIRST stringify materialises; the
		// steady-state cost would be the same as reading cached values.
		vm := goja.New()
		p, cnt := lazyBenchProvider(vals)
		target := vm.NewObject()
		proxy := vm.NewProxy(target, &goja.ProxyTrapConfig{
			Get: func(t *goja.Object, prop string, receiver goja.Value) goja.Value {
				return vm.ToValue(p(prop))
			},
			Has: func(t *goja.Object, prop string) bool { return true },
			GetOwnPropertyDescriptor: func(t *goja.Object, prop string) goja.PropertyDescriptor {
				return goja.PropertyDescriptor{
					Enumerable:   goja.FLAG_TRUE,
					Configurable: goja.FLAG_TRUE,
				}
			},
			OwnKeys: func(t *goja.Object) *goja.Object {
				return vm.NewArray(strAnySlice(keys)...)
			},
		})
		_ = vm.Set("obj", vm.ToValue(proxy))
		for i := 0; i < b.N; i++ {
			if _, err := vm.RunProgram(prg); err != nil {
				b.Fatal(err)
			}
		}
		reportProviderPerOp(b, cnt)
	})
}
