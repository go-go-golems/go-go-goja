package timemod_test

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
)

func TestTimeModuleSmoke(t *testing.T) {
	factory, err := gggengine.NewBuilder().WithModules(gggengine.DefaultRegistryModules()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(context.Background(), "time-smoke", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			const time = require("time");
			const t0 = time.now();
			for (let i = 0; i < 10000; i++) {}
			const elapsed = time.since(t0);
			typeof t0 === "number" && typeof elapsed === "number" && elapsed >= 0 ? "ok" : "bad";
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run time smoke: %v", err)
	}
	if ret != "ok" {
		t.Fatalf("time smoke = %v, want ok", ret)
	}
}
