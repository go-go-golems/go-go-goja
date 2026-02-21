package gojaperf_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"

	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/modules"
)

var silenceLogsOnce sync.Once

func silenceBenchLogs() {
	silenceLogsOnce.Do(func() {
		log.SetOutput(io.Discard)
	})
}

func newRuntime(tb testing.TB, opts ...require.Option) (*goja.Runtime, *require.RequireModule, func()) {
	tb.Helper()
	factory, err := engine.NewBuilder().
		WithRequireOptions(opts...).
		WithModules(engine.DefaultRegistryModules()).
		Build()
	if err != nil {
		tb.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		tb.Fatalf("new runtime: %v", err)
	}
	return rt.VM, rt.Require, func() {
		_ = rt.Close(context.Background())
	}
}

func compile(tb testing.TB, name, src string) *goja.Program {
	tb.Helper()
	prg, err := goja.Compile(name, src, false)
	if err != nil {
		tb.Fatalf("compile %s: %v", name, err)
	}
	return prg
}

func BenchmarkRuntimeSpawn(b *testing.B) {
	silenceBenchLogs()
	b.ReportAllocs()

	b.Run("GojaNew", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if vm := goja.New(); vm == nil {
				b.Fatal("nil runtime")
			}
		}
	})

	b.Run("EngineNew", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			vm, req, closeRuntime := newRuntime(b)
			if vm == nil || req == nil {
				b.Fatal("nil runtime or require module")
			}
			closeRuntime()
		}
	})

	b.Run("EngineFactory", func(b *testing.B) {
		factory, err := engine.NewBuilder().
			WithModules(engine.DefaultRegistryModules()).
			Build()
		if err != nil {
			b.Fatalf("build factory: %v", err)
		}
		for i := 0; i < b.N; i++ {
			rt, err := factory.NewRuntime(context.Background())
			if err != nil {
				b.Fatalf("new runtime: %v", err)
			}
			vm, req := rt.VM, rt.Require
			if vm == nil || req == nil {
				b.Fatal("nil runtime or require module")
			}
			_ = rt.Close(context.Background())
		}
	})
}

func BenchmarkRuntimeSpawnAndExecute(b *testing.B) {
	silenceBenchLogs()

	const script = "var n = 41; n + 1;"
	prg := compile(b, "runtime-spawn.js", script)

	b.ReportAllocs()

	b.Run("RunString_FreshRuntime", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			vm, _, closeRuntime := newRuntime(b)
			v, err := vm.RunString(script)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("unexpected result: %d", got)
			}
			closeRuntime()
		}
	})

	b.Run("RunProgram_FreshRuntime", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			vm, _, closeRuntime := newRuntime(b)
			v, err := vm.RunProgram(prg)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("unexpected result: %d", got)
			}
			closeRuntime()
		}
	})
}

func BenchmarkRuntimeReuse(b *testing.B) {
	silenceBenchLogs()

	const script = "(40 + 2)"
	prg := compile(b, "runtime-reuse.js", script)
	vm, _, closeRuntime := newRuntime(b)
	defer closeRuntime()

	b.ReportAllocs()

	b.Run("RunString_ReusedRuntime", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v, err := vm.RunString(script)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("unexpected result: %d", got)
			}
		}
	})

	b.Run("RunProgram_ReusedRuntime", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prg)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("unexpected result: %d", got)
			}
		}
	})
}

func BenchmarkJSLoading(b *testing.B) {
	silenceBenchLogs()

	scripts := []struct {
		name string
		src  string
	}{
		{name: "small", src: buildArithmeticScript(32)},
		{name: "medium", src: buildArithmeticScript(512)},
		{name: "large", src: buildArithmeticScript(4096)},
	}

	for _, tc := range scripts {
		tc := tc
		b.Run(fmt.Sprintf("Compile_%s", tc.name), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := goja.Compile(tc.name+".js", tc.src, false); err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("RunString_FreshRuntime_%s", tc.name), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				vm, _, closeRuntime := newRuntime(b)
				if _, err := vm.RunString(tc.src); err != nil {
					b.Fatal(err)
				}
				closeRuntime()
			}
		})

		b.Run(fmt.Sprintf("RunProgram_ReusedRuntime_%s", tc.name), func(b *testing.B) {
			b.ReportAllocs()
			vm, _, closeRuntime := newRuntime(b)
			defer closeRuntime()
			prg := compile(b, tc.name+".js", tc.src)
			for i := 0; i < b.N; i++ {
				if _, err := vm.RunProgram(prg); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkJSCallingGo(b *testing.B) {
	silenceBenchLogs()

	nativeAdd := func(a, b int) int {
		return a + b
	}

	b.ReportAllocs()

	b.Run("GoDirect", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if got := nativeAdd(40, 2); got != 42 {
				b.Fatalf("unexpected result: %d", got)
			}
		}
	})

	b.Run("JS_vm_Set", func(b *testing.B) {
		vm := goja.New()
		if err := vm.Set("add", nativeAdd); err != nil {
			b.Fatal(err)
		}
		prg := compile(b, "js-vm-set.js", "add(40, 2)")

		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prg)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("unexpected result: %d", got)
			}
		}
	})

	b.Run("JS_module_SetExport", func(b *testing.B) {
		vm := goja.New()
		exports := vm.NewObject()
		modules.SetExport(exports, "bench", "add", nativeAdd)
		if err := vm.Set("bench", exports); err != nil {
			b.Fatal(err)
		}
		prg := compile(b, "js-module-setexport.js", "bench.add(40, 2)")

		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prg)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 42 {
				b.Fatalf("unexpected result: %d", got)
			}
		}
	})
}

func BenchmarkGoCallingJS(b *testing.B) {
	silenceBenchLogs()

	directAdd := func(a, c int64) int64 {
		return a + c
	}

	vm := goja.New()
	if _, err := vm.RunString(`function add(a, b) { return a + b; }`); err != nil {
		b.Fatal(err)
	}
	addVal := vm.Get("add")
	addFn, ok := goja.AssertFunction(addVal)
	if !ok {
		b.Fatal("add is not callable")
	}

	thisVal := goja.Undefined()
	a := vm.ToValue(40)
	c := vm.ToValue(2)

	b.ReportAllocs()

	b.Run("GoDirect", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if got := directAdd(40, 2); got != 42 {
				b.Fatalf("unexpected result: %d", got)
			}
		}
	})

	b.Run("GojaAssertFunction", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result, err := addFn(thisVal, a, c)
			if err != nil {
				b.Fatal(err)
			}
			if got := result.ToInteger(); got != 42 {
				b.Fatalf("unexpected result: %d", got)
			}
		}
	})
}

func BenchmarkRequireLoading(b *testing.B) {
	silenceBenchLogs()

	loader := sourceLoader(map[string]string{
		"entry.js": `
			var math = require('./math.js');
			module.exports = function run() { return math.add(40, 2); };
		`,
		"math.js": `
			module.exports = {
				add: function(a, b) { return a + b; }
			};
		`,
	})

	b.ReportAllocs()

	b.Run("ColdRequire_NewRuntime", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			vm, req, closeRuntime := newRuntime(b, require.WithLoader(loader))
			mod, err := req.Require("./entry.js")
			if err != nil {
				b.Fatal(err)
			}
			runVal := mod.ToObject(vm)
			runFn, ok := goja.AssertFunction(runVal)
			if !ok {
				b.Fatal("entry.js export is not callable")
			}
			out, err := runFn(goja.Undefined())
			if err != nil {
				b.Fatal(err)
			}
			if got := out.ToInteger(); got != 42 {
				b.Fatalf("unexpected result: %d", got)
			}
			closeRuntime()
		}
	})

	b.Run("WarmRequire_ReusedRuntime", func(b *testing.B) {
		vm, req, closeRuntime := newRuntime(b, require.WithLoader(loader))
		defer closeRuntime()
		for i := 0; i < b.N; i++ {
			mod, err := req.Require("./entry.js")
			if err != nil {
				b.Fatal(err)
			}
			runVal := mod.ToObject(vm)
			runFn, ok := goja.AssertFunction(runVal)
			if !ok {
				b.Fatal("entry.js export is not callable")
			}
			out, err := runFn(goja.Undefined())
			if err != nil {
				b.Fatal(err)
			}
			if got := out.ToInteger(); got != 42 {
				b.Fatalf("unexpected result: %d", got)
			}
		}
	})
}

func buildArithmeticScript(lines int) string {
	if lines < 1 {
		lines = 1
	}

	var sb strings.Builder
	sb.Grow(lines * 16)
	sb.WriteString("var total = 0;\n")
	for i := 0; i < lines; i++ {
		sb.WriteString(fmt.Sprintf("total = total + %d;\n", i%13))
	}
	sb.WriteString("total;")
	return sb.String()
}

func sourceLoader(modules map[string]string) func(path string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		trimmed := strings.TrimPrefix(path, "./")
		trimmed = strings.TrimPrefix(trimmed, "/")
		src, ok := modules[trimmed]
		if !ok {
			return nil, require.ModuleFileDoesNotExistError
		}
		return []byte(src), nil
	}
}
