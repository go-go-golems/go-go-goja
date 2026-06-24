package gojaperf_test

// phase5_lazy_dyn_bench_test.go — head-to-head: Proxy-backed lazy.Map vs
// DynamicObject-backed lazy.DynMap. The motivating question: does the
// DynamicObject backend erase the per-key descriptor cost that made Proxy
// enumeration 15.8× an eager object?
//
//   go test ./perf/goja -run '^$' \
//     -bench 'LazyBackend' -benchtime=1s -count=5 -benchmem | tee raw.txt

import (
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/dop251/goja"

	"github.com/go-go-golems/go-go-goja/modules/lazy"
)

// dynProvider mirrors lazyBenchProvider but for the DynMap (same call counting).
func dynProvider(vals []string) (func(string) any, *int64) {
	var n int64
	p := func(key string) any {
		atomic.AddInt64(&n, 1)
		i, _ := strconv.Atoi(key[1:])
		if i < 0 || i >= len(vals) {
			i = i % len(vals)
		}
		return vals[i]
	}
	return p, &n
}

func reportProv(b *testing.B, n *int64) {
	b.ReportMetric(float64(atomic.LoadInt64(n))/float64(b.N), "provider/op")
}

// BenchmarkLazyBackendRead — single cached read: Proxy vs DynamicObject.
func BenchmarkLazyBackendRead(b *testing.B) {
	silenceBenchLogs()
	b.ReportAllocs()
	const key = "k000"
	vals := lazyBenchVals(1)
	prg := compile(b, "dyn-read.js", `obj.k000`)

	b.Run("Proxy", func(b *testing.B) {
		vm := goja.New()
		p, cnt := dynProvider(vals)
		m := &lazy.Map{Keys: []string{key}, Provider: p}
		_ = vm.Set("obj", m.ToValue(vm))
		_, _ = vm.RunProgram(prg) // warm cache
		for i := 0; i < b.N; i++ {
			if _, err := vm.RunProgram(prg); err != nil {
				b.Fatal(err)
			}
		}
		reportProv(b, cnt)
	})

	b.Run("DynamicObject", func(b *testing.B) {
		vm := goja.New()
		p, cnt := dynProvider(vals)
		m := &lazy.DynMap{Keys: []string{key}, Provider: p}
		_ = vm.Set("obj", m.ToValue(vm))
		_, _ = vm.RunProgram(prg) // warm cache
		for i := 0; i < b.N; i++ {
			if _, err := vm.RunProgram(prg); err != nil {
				b.Fatal(err)
			}
		}
		reportProv(b, cnt)
	})
}

// BenchmarkLazyBackendEnum — Object.keys over 100 keys: Proxy vs DynamicObject.
// provider/op is 0 for both; the differentiator is per-key descriptor machinery.
func BenchmarkLazyBackendEnum(b *testing.B) {
	silenceBenchLogs()
	b.ReportAllocs()
	const n = 100
	keys := lazyBenchKeys(n)
	vals := lazyBenchVals(n)
	prg := compile(b, "dyn-keys.js", `Object.keys(obj)`)

	b.Run("Proxy", func(b *testing.B) {
		vm := goja.New()
		p, cnt := dynProvider(vals)
		m := &lazy.Map{Keys: keys, Provider: p}
		_ = vm.Set("obj", m.ToValue(vm))
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prg)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToObject(vm).Get("length").ToInteger(); got != n {
				b.Fatalf("got %d keys", got)
			}
		}
		reportProv(b, cnt)
	})

	b.Run("DynamicObject", func(b *testing.B) {
		vm := goja.New()
		p, cnt := dynProvider(vals)
		m := &lazy.DynMap{Keys: keys, Provider: p}
		_ = vm.Set("obj", m.ToValue(vm))
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prg)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToObject(vm).Get("length").ToInteger(); got != n {
				b.Fatalf("got %d keys", got)
			}
		}
		reportProv(b, cnt)
	})
}

// BenchmarkLazyBackendScale — Object.keys on DynMap as N grows (provider/op=0).
func BenchmarkLazyBackendScale(b *testing.B) {
	silenceBenchLogs()
	b.ReportAllocs()
	prg := compile(b, "dyn-scale.js", `Object.keys(obj)`)
	for _, n := range []int{10, 100, 1000} {
		keys := lazyBenchKeys(n)
		vals := lazyBenchVals(n)
		b.Run("DynamicObject/N="+strconv.Itoa(n), func(b *testing.B) {
			vm := goja.New()
			p, cnt := dynProvider(vals)
			m := &lazy.DynMap{Keys: keys, Provider: p}
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
			reportProv(b, cnt)
		})
	}
}
