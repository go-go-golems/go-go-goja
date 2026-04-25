package engine

import (
	"context"
	"testing"

	"github.com/dop251/goja"
)

func TestPerformanceNowAndConsoleTimersSmoke(t *testing.T) {
	factory, err := NewBuilder().Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(context.Background(), "performance-smoke", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			const t0 = performance.now();
			for (let i = 0; i < 10000; i++) {}
			const t1 = performance.now();
			console.time("work");
			console.timeLog("work");
			console.timeEnd("work");
			t1 >= t0 && typeof console.time === "function" && typeof console.timeEnd === "function" ? "ok" : "bad";
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run performance smoke: %v", err)
	}
	if ret != "ok" {
		t.Fatalf("performance smoke = %v, want ok", ret)
	}
}
