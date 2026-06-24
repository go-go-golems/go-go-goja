package lazy_test

import (
	"context"
	"strings"
	"testing"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/pkg/engine"
)

func newLazyRuntime(t *testing.T) (*gggengine.Runtime, context.CancelFunc) {
	t.Helper()
	factory, err := gggengine.NewRuntimeFactoryBuilder().
		UseModuleMiddleware(gggengine.MiddlewareOnly("lazy")).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	ctx := context.Background()
	rt, err := factory.NewRuntime(gggengine.WithStartupContext(ctx), gggengine.WithLifetimeContext(ctx))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	return rt, func() { _ = rt.Close(ctx) }
}

func runInRT(t *testing.T, rt *gggengine.Runtime, op string, fn func(*goja.Runtime) (string, error)) string {
	t.Helper()
	ret, err := rt.Owner.Call(context.Background(), op, func(_ context.Context, vm *goja.Runtime) (any, error) {
		s, err := fn(vm)
		return s, err
	})
	if err != nil {
		t.Fatalf("run %s: %v", op, err)
	}
	return ret.(string)
}

// require("lazy").map: lazy reads + free enumeration, end to end through the
// engine + require() plumbing and a JS-backed getter.
func TestLazyModule_MapLazyAndFreeEnumeration(t *testing.T) {
	rt, closeRT := newLazyRuntime(t)
	defer closeRT()

	out := runInRT(t, rt, "lazy.map", func(vm *goja.Runtime) (string, error) {
		v, err := vm.RunString(`
			const lazy = require("lazy");
			let calls = {};
			const device = lazy.map({
				keys: ["name", "wifis"],
				get: (k) => { calls[k] = (calls[k]||0)+1; return k === "wifis" ? ["a","b","c"] : "atom-3r"; }
			});
			// Object.keys must NOT call get:
			const keys = Object.keys(device);
			const enumCalls = (calls.name||0) + (calls.wifis||0);
			// A single read calls get once and returns the value:
			const v1 = device.wifis.length;
			const v2 = device.wifis.length; // cached
			JSON.stringify({ keys, enumCalls, v1, v2, wifisCalls: calls.wifis||0 });
		`)
		if err != nil {
			return "", err
		}
		return v.String(), nil
	})

	for _, want := range []string{
		`"keys":["name","wifis"]`,
		`"enumCalls":0`,  // enumeration did not call get
		`"v1":3`,         // device.wifis.length
		`"v2":3`,         // cached
		`"wifisCalls":1`, // read once, cached
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %s", want, out)
		}
	}
}

func TestLazyModule_Array(t *testing.T) {
	rt, closeRT := newLazyRuntime(t)
	defer closeRT()

	out := runInRT(t, rt, "lazy.array", func(vm *goja.Runtime) (string, error) {
		v, err := vm.RunString(`
			const lazy = require("lazy");
			const a = lazy.array({ length: 1000, get: (i) => i * i });
			JSON.stringify({ len: a.length, five: a[5], three: a[3], cached: a[5], oob: a[1001] === undefined });
		`)
		if err != nil {
			return "", err
		}
		return v.String(), nil
	})

	for _, want := range []string{
		`"len":1000`, `"five":25`, `"three":9`, `"cached":25`, `"oob":true`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %s", want, out)
		}
	}
}

func TestLazyModule_Record(t *testing.T) {
	rt, closeRT := newLazyRuntime(t)
	defer closeRT()

	out := runInRT(t, rt, "lazy.record", func(vm *goja.Runtime) (string, error) {
		v, err := vm.RunString(`
			const lazy = require("lazy");
			let fetches = 0;
			const env = lazy.record({
				keys: () => ["HOME", "EDITOR"],
				has:  (k) => k === "HOME" || k === "EDITOR",
				get:  (k) => { fetches++; return k === "HOME" ? "/home/x" : "nvim"; }
			});
			const keyCount = Object.keys(env).length; // no fetch
			const editor = env.EDITOR;                // one fetch
			JSON.stringify({ keyCount, editor, fetches });
		`)
		if err != nil {
			return "", err
		}
		return v.String(), nil
	})

	for _, want := range []string{`"keyCount":2`, `"editor":"nvim"`, `"fetches":1`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %s", want, out)
		}
	}
}
