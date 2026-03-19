package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja_nodejs/require"
)

func TestBuilderWithRequireOptions(t *testing.T) {
	loader := func(path string) ([]byte, error) {
		trimmed := strings.TrimPrefix(path, "./")
		if trimmed == "entry.js" {
			return []byte("module.exports = { ok: 42 };"), nil
		}
		return nil, require.ModuleFileDoesNotExistError
	}

	factory, err := NewBuilder(
		WithRequireOptions(require.WithLoader(loader)),
	).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(context.Background())
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

func TestRuntimeContextCancelsOnClose(t *testing.T) {
	factory, err := NewBuilder().Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	runtimeCtx := rt.Context()
	select {
	case <-runtimeCtx.Done():
		t.Fatalf("runtime context already canceled")
	default:
	}

	if err := rt.Close(context.Background()); err != nil {
		t.Fatalf("close runtime: %v", err)
	}

	select {
	case <-runtimeCtx.Done():
	case <-time.After(2 * time.Second):
		t.Fatalf("runtime context was not canceled on close")
	}
}
