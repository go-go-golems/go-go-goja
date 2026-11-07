package engine

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
)

func newTestRuntime(t *testing.T, builderOpts []Option, runtimeOpts ...RuntimeOption) *Runtime {
	t.Helper()
	opts := append([]Option{WithImplicitDefaultRegistryModules(false), WithDataOnlyDefaultRegistryModules(false)}, builderOpts...)
	factory, err := NewRuntimeFactoryBuilder(opts...).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(runtimeOpts...)
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	t.Cleanup(func() { _ = rt.Close(context.Background()) })
	return rt
}

func typeofGlobals(t *testing.T, rt *Runtime, exprs ...string) map[string]string {
	t.Helper()
	out := map[string]string{}
	_, err := rt.Owner.Call(context.Background(), "test.typeof", func(_ context.Context, vm *goja.Runtime) (any, error) {
		for _, expr := range exprs {
			v, err := vm.RunString("typeof " + expr)
			if err != nil {
				return nil, err
			}
			out[expr] = v.String()
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("evaluate typeof: %v", err)
	}
	return out
}

var defaultGlobalExprs = []string{"require", "console", "console.log", "console.time", "performance", "Buffer", "URL"}

func TestDefaultGlobalsInstalledByDefault(t *testing.T) {
	rt := newTestRuntime(t, nil)
	got := typeofGlobals(t, rt, defaultGlobalExprs...)
	want := map[string]string{
		"require": "function", "console": "object", "console.log": "function", "console.time": "function",
		"performance": "object", "Buffer": "function", "URL": "function",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("typeof %s = %q, want %q", k, got[k], v)
		}
	}
}

func TestWithoutDefaultGlobalsSkipsEveryGlobal(t *testing.T) {
	rt := newTestRuntime(t, []Option{WithoutDefaultGlobals(DefaultGlobals()...)})
	got := typeofGlobals(t, rt, "require", "console", "performance", "Buffer", "URL", "URLSearchParams")
	if got["require"] != "function" {
		t.Errorf("require must stay installed, typeof = %q", got["require"])
	}
	for _, k := range []string{"console", "performance", "Buffer", "URL", "URLSearchParams"} {
		if got[k] != "undefined" {
			t.Errorf("typeof %s = %q, want undefined", k, got[k])
		}
	}
}

func TestWithoutDefaultGlobalsSelective(t *testing.T) {
	rt := newTestRuntime(t, []Option{
		WithoutDefaultGlobals(GlobalConsoleTimers),
		WithoutDefaultGlobals(GlobalPerformance), // accumulates
	})
	got := typeofGlobals(t, rt, "console.log", "console.time", "console.timeEnd", "performance", "Buffer", "URL")
	want := map[string]string{
		"console.log": "function", "console.time": "undefined", "console.timeEnd": "undefined",
		"performance": "undefined", "Buffer": "function", "URL": "function",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("typeof %s = %q, want %q", k, got[k], v)
		}
	}
}

func TestWithoutConsoleAlsoSkipsConsoleTimers(t *testing.T) {
	// Console timers attach to the console global; without it they must not
	// fail runtime creation.
	rt := newTestRuntime(t, []Option{WithoutDefaultGlobals(GlobalConsole)})
	got := typeofGlobals(t, rt, "console", "performance")
	if got["console"] != "undefined" || got["performance"] != "object" {
		t.Fatalf("got %v", got)
	}
}

func TestWithoutDefaultGlobalsRejectsUnknownName(t *testing.T) {
	_, err := NewRuntimeFactoryBuilder(WithoutDefaultGlobals("setTimeout")).Build()
	if err == nil || !strings.Contains(err.Error(), `unknown default global "setTimeout"`) {
		t.Fatalf("expected unknown global error, got %v", err)
	}
}

// countingOwner wraps the base owner and counts what modules schedule.
type countingOwner struct {
	base  runtimebridge.RuntimeOwner
	calls atomic.Int32
	posts atomic.Int32
}

func (o *countingOwner) Call(ctx context.Context, op string, fn func(context.Context, *goja.Runtime) (any, error)) (any, error) {
	o.calls.Add(1)
	return o.base.Call(ctx, op, fn)
}

func (o *countingOwner) Post(ctx context.Context, op string, fn func(context.Context, *goja.Runtime)) error {
	o.posts.Add(1)
	return o.base.Post(ctx, op, fn)
}

// lookupRecorder is a module registrar that records the bridge owner it sees
// while registering, which happens before require is enabled.
type lookupRecorder struct {
	seen runtimebridge.RuntimeOwner
}

func (r *lookupRecorder) ID() string { return "lookup-recorder" }

func (r *lookupRecorder) RegisterRuntimeModule(ctx *RuntimeModuleRegistrationContext, reg *require.Registry) error {
	services, ok := runtimebridge.Lookup(ctx.VM)
	if ok {
		r.seen = services.Owner
	}
	reg.RegisterNativeModule("probe", func(vm *goja.Runtime, module *goja.Object) {
		exports := module.Get("exports").(*goja.Object)
		_ = exports.Set("settleLater", func(call goja.FunctionCall) goja.Value {
			// Capture the argument now: goja reuses the call's argument
			// storage after the function returns.
			value := call.Argument(0)
			promise, resolve, _ := vm.NewPromise()
			services, _ := runtimebridge.Lookup(vm)
			_ = services.Owner.Post(context.Background(), "probe.settle", func(context.Context, *goja.Runtime) {
				_ = resolve(value)
			})
			return vm.ToValue(promise)
		})
	})
	return nil
}

func TestWithRuntimeBridgeOwnerReplacesOwnerBeforeModulesRegister(t *testing.T) {
	var wrapper *countingOwner
	var wrappedVM *goja.Runtime
	rec := &lookupRecorder{}
	factory, err := NewRuntimeFactoryBuilder(
		WithImplicitDefaultRegistryModules(false),
		WithDataOnlyDefaultRegistryModules(false),
	).WithModules(rec).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	rt, err := factory.NewRuntime(WithRuntimeBridgeOwner(func(vm *goja.Runtime, base runtimebridge.RuntimeOwner) runtimebridge.RuntimeOwner {
		wrappedVM = vm
		wrapper = &countingOwner{base: base}
		return wrapper
	}))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	if wrappedVM != rt.VM {
		t.Fatalf("wrapper received a different VM")
	}
	if rec.seen != runtimebridge.RuntimeOwner(wrapper) {
		t.Fatalf("module registration saw %T, want the wrapper", rec.seen)
	}
	services, ok := runtimebridge.Lookup(rt.VM)
	if !ok || services.Owner != runtimebridge.RuntimeOwner(wrapper) {
		t.Fatalf("lookup owner = %T, want the wrapper", services.Owner)
	}

	// A module settling a promise goes through the wrapper's Post, and the
	// wrapper's delegation still runs the settle on the runtime's loop.
	result, err := rt.Owner.Call(context.Background(), "test.settle", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`globalThis.out = null; require("probe").settleLater(7).then(v => { globalThis.out = v; });`)
	})
	if err != nil || result == nil {
		t.Fatalf("run: %v", err)
	}
	if wrapper.posts.Load() != 1 {
		t.Fatalf("posts through wrapper = %d, want 1", wrapper.posts.Load())
	}
	if err := rt.Owner.WaitIdle(context.Background()); err != nil {
		t.Fatalf("wait idle: %v", err)
	}
	out, err := rt.Owner.Call(context.Background(), "test.read", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.Get("out").Export(), nil
	})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if out != int64(7) {
		t.Fatalf("out = %#v, want 7", out)
	}
}

func TestWithRuntimeBridgeOwnerNilResultKeepsDefault(t *testing.T) {
	rt := newTestRuntime(t, nil, WithRuntimeBridgeOwner(func(*goja.Runtime, runtimebridge.RuntimeOwner) runtimebridge.RuntimeOwner {
		return nil
	}))
	services, ok := runtimebridge.Lookup(rt.VM)
	if !ok || services.Owner == nil {
		t.Fatalf("expected the default owner")
	}
	if _, isWrapper := services.Owner.(*countingOwner); isWrapper {
		t.Fatalf("unexpected wrapper")
	}
}
