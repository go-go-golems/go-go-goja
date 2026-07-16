package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

func TestRuntimeCanCloseImmediatelyAfterCreation(t *testing.T) {
	factory, err := NewRuntimeFactoryBuilder().UseModuleMiddleware(MiddlewareSafe()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	for i := 0; i < 20; i++ {
		runtime, err := factory.NewRuntime()
		if err != nil {
			t.Fatalf("new runtime %d: %v", i, err)
		}
		if err := runtime.Close(context.Background()); err != nil {
			t.Fatalf("close runtime %d: %v", i, err)
		}
	}
}

func TestFactoryWithRequireOptions(t *testing.T) {
	loader := func(path string) ([]byte, error) {
		trimmed := strings.TrimPrefix(path, "./")
		if trimmed == "entry.js" {
			return []byte("module.exports = { ok: 42 };"), nil
		}
		return nil, require.ModuleFileDoesNotExistError
	}

	factory, err := NewRuntimeFactoryBuilder(
		WithRequireOptions(require.WithLoader(loader)),
	).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(WithStartupContext(context.Background()), WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() {
		_ = rt.Close(context.Background())
	}()

	val, err := rt.Require.Require("./entry.js")
	if err != nil {
		t.Fatalf("require entry.js: %v", err)
	}

	obj := val.ToObject(rt.VM)
	if got := obj.Get("ok").ToInteger(); got != 42 {
		t.Fatalf("ok = %d, want 42", got)
	}
}

func TestFactoryWithRecoveredPanicStack(t *testing.T) {
	factory, err := NewRuntimeFactoryBuilder(WithRecoveredPanicStack(true)).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(WithStartupContext(context.Background()), WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	_, err = rt.Owner.Call(context.Background(), "engine.panic.stack", func(context.Context, *goja.Runtime) (any, error) {
		panic("engine boom")
	})
	if err == nil {
		t.Fatalf("expected recovered panic error")
	}
	if !strings.Contains(err.Error(), "engine boom") || !strings.Contains(err.Error(), "runtime/debug.Stack") {
		t.Fatalf("expected recovered panic stack, got: %v", err)
	}
}
