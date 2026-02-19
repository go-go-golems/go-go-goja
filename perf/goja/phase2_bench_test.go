package gojaperf_test

import (
	"fmt"
	goruntime "runtime"
	"strconv"
	"testing"

	"github.com/dop251/goja"

	"github.com/go-go-golems/go-go-goja/modules"
)

func BenchmarkValueConversion(b *testing.B) {
	silenceBenchLogs()

	vm := goja.New()
	mapPayload := makeMapPayload(20)
	nestedPayload := makeNestedPayload(3, 3)

	type exportedShape struct {
		ID     int
		Name   string
		Active bool
		Scores []int
	}

	valueForExport := vm.ToValue(mapPayload)
	valueForExportTo, err := vm.RunString(`({ID: 42, Name: "goja", Active: true, Scores: [1,2,3,4]})`)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	b.Run("ToValue_Primitive_Int", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = vm.ToValue(42)
		}
	})

	b.Run("ToValue_Map20", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = vm.ToValue(mapPayload)
		}
	})

	b.Run("ToValue_Nested", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = vm.ToValue(nestedPayload)
		}
	})

	b.Run("Export_Map20", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = valueForExport.Export()
		}
	})

	b.Run("ExportTo_Struct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var out exportedShape
			if err := vm.ExportTo(valueForExportTo, &out); err != nil {
				b.Fatal(err)
			}
			if out.ID != 42 {
				b.Fatalf("unexpected id: %d", out.ID)
			}
		}
	})
}

func BenchmarkPayloadSweep(b *testing.B) {
	silenceBenchLogs()

	payloads := []struct {
		name      string
		jsLiteral string
		goValue   interface{}
	}{
		{
			name:      "tiny",
			jsLiteral: `{"a":1,"b":2}`,
			goValue:   map[string]interface{}{"a": 1, "b": 2},
		},
		{
			name:      "medium",
			jsLiteral: makeJSONLiteralMap(20),
			goValue:   makeMapPayload(20),
		},
		{
			name:      "large",
			jsLiteral: makeJSONLiteralNested(4, 4),
			goValue:   makeNestedPayload(4, 4),
		},
	}

	consumeGo := func(v interface{}) int {
		switch t := v.(type) {
		case map[string]interface{}:
			return len(t)
		case []interface{}:
			return len(t)
		default:
			return 1
		}
	}

	for _, tc := range payloads {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()

			b.Run("GoDirect", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_ = consumeGo(tc.goValue)
				}
			})

			b.Run("JS_vm_Set", func(b *testing.B) {
				vm := goja.New()
				if err := vm.Set("consume", consumeGo); err != nil {
					b.Fatal(err)
				}
				initScript := fmt.Sprintf(`var payload = %s;`, tc.jsLiteral)
				if _, err := vm.RunString(initScript); err != nil {
					b.Fatal(err)
				}
				prg := compile(b, "payload-vm-set.js", "consume(payload)")
				for i := 0; i < b.N; i++ {
					if _, err := vm.RunProgram(prg); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("JS_module_SetExport", func(b *testing.B) {
				vm := goja.New()
				exports := vm.NewObject()
				modules.SetExport(exports, "bench", "consume", consumeGo)
				if err := vm.Set("bench", exports); err != nil {
					b.Fatal(err)
				}
				initScript := fmt.Sprintf(`var payload = %s;`, tc.jsLiteral)
				if _, err := vm.RunString(initScript); err != nil {
					b.Fatal(err)
				}
				prg := compile(b, "payload-setexport.js", "bench.consume(payload)")
				for i := 0; i < b.N; i++ {
					if _, err := vm.RunProgram(prg); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("GojaAssertFunction", func(b *testing.B) {
				vm := goja.New()
				if _, err := vm.RunString(`
					function consume(payload) {
						if (Array.isArray(payload)) return payload.length;
						if (payload && typeof payload === "object") return Object.keys(payload).length;
						return 1;
					}
				`); err != nil {
					b.Fatal(err)
				}
				fnVal := vm.Get("consume")
				fn, ok := goja.AssertFunction(fnVal)
				if !ok {
					b.Fatal("consume not callable")
				}
				for i := 0; i < b.N; i++ {
					arg := vm.ToValue(tc.goValue)
					if _, err := fn(goja.Undefined(), arg); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

func BenchmarkGCSensitivity(b *testing.B) {
	silenceBenchLogs()

	b.ReportAllocs()

	b.Run("SpawnAndDiscardRuntime", func(b *testing.B) {
		const script = `(() => { let a=[]; for (let i=0;i<128;i++) { a.push({i, s:"abcd", n:[1,2,3,4]}); } return a.length; })()`
		for i := 0; i < b.N; i++ {
			vm, _ := newRuntime(b)
			v, err := vm.RunString(script)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 128 {
				b.Fatalf("unexpected result: %d", got)
			}
		}
	})

	b.Run("ReuseRuntime_AllocHeavy", func(b *testing.B) {
		vm, _ := newRuntime(b)
		prg := compile(b, "gc-heavy.js", `(() => { let a=[]; for (let i=0;i<128;i++) { a.push({i, s:"abcd", n:[1,2,3,4]}); } return a.length; })()`)
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prg)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 128 {
				b.Fatalf("unexpected result: %d", got)
			}
		}
	})

	b.Run("ReuseRuntime_AllocHeavy_WithPeriodicGC", func(b *testing.B) {
		vm, _ := newRuntime(b)
		prg := compile(b, "gc-heavy-periodic.js", `(() => { let a=[]; for (let i=0;i<128;i++) { a.push({i, s:"abcd", n:[1,2,3,4]}); } return a.length; })()`)
		for i := 0; i < b.N; i++ {
			v, err := vm.RunProgram(prg)
			if err != nil {
				b.Fatal(err)
			}
			if got := v.ToInteger(); got != 128 {
				b.Fatalf("unexpected result: %d", got)
			}
			if i%64 == 0 {
				goruntime.GC()
			}
		}
	})
}

func makeMapPayload(size int) map[string]interface{} {
	m := make(map[string]interface{}, size)
	for i := 0; i < size; i++ {
		m[fmt.Sprintf("key_%02d", i)] = i
	}
	return m
}

func makeNestedPayload(depth, width int) map[string]interface{} {
	if depth <= 1 {
		return makeMapPayload(width)
	}

	out := map[string]interface{}{}
	for i := 0; i < width; i++ {
		key := fmt.Sprintf("node_%d", i)
		out[key] = makeNestedPayload(depth-1, width)
	}
	return out
}

func makeJSONLiteralMap(size int) string {
	parts := make([]string, 0, size)
	for i := 0; i < size; i++ {
		parts = append(parts, fmt.Sprintf(`"k%s":%d`, strconv.Itoa(i), i))
	}
	return "{" + join(parts, ",") + "}"
}

func makeJSONLiteralNested(depth, width int) string {
	if depth <= 1 {
		return makeJSONLiteralMap(width)
	}
	parts := make([]string, 0, width)
	for i := 0; i < width; i++ {
		parts = append(parts, fmt.Sprintf(`"n%d":%s`, i, makeJSONLiteralNested(depth-1, width)))
	}
	return "{" + join(parts, ",") + "}"
}

func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep + parts[i]
	}
	return out
}
