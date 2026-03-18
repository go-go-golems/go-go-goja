package engine

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

type testRuntimeModuleRegistrar struct {
	mu     sync.Mutex
	nextID int
	closed []int
}

func (r *testRuntimeModuleRegistrar) ID() string {
	return "test-runtime-registrar"
}

func (r *testRuntimeModuleRegistrar) RegisterRuntimeModules(ctx *RuntimeModuleContext, reg *require.Registry) error {
	r.mu.Lock()
	r.nextID++
	id := r.nextID
	r.mu.Unlock()

	reg.RegisterNativeModule("runtime-info", func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)
		_ = exports.Set("id", id)
	})

	return ctx.AddCloser(func(context.Context) error {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.closed = append(r.closed, id)
		return nil
	})
}

func (r *testRuntimeModuleRegistrar) closedIDs() []int {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]int, len(r.closed))
	copy(out, r.closed)
	return out
}

func TestRuntimeModuleRegistrarRegistersPerRuntime(t *testing.T) {
	registrar := &testRuntimeModuleRegistrar{}

	factory, err := NewBuilder().
		WithRuntimeModuleRegistrars(registrar).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt1, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime 1: %v", err)
	}
	defer func() {
		_ = rt1.Close(context.Background())
	}()

	rt2, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime 2: %v", err)
	}
	defer func() {
		_ = rt2.Close(context.Background())
	}()

	val1, err := rt1.Require.Require("runtime-info")
	if err != nil {
		t.Fatalf("require runtime-info in rt1: %v", err)
	}
	val2, err := rt2.Require.Require("runtime-info")
	if err != nil {
		t.Fatalf("require runtime-info in rt2: %v", err)
	}

	got1 := val1.ToObject(rt1.VM).Get("id").ToInteger()
	got2 := val2.ToObject(rt2.VM).Get("id").ToInteger()
	if got1 != 1 {
		t.Fatalf("runtime 1 id = %d, want 1", got1)
	}
	if got2 != 2 {
		t.Fatalf("runtime 2 id = %d, want 2", got2)
	}
}

func TestRuntimeCloseRunsClosersInReverseOrder(t *testing.T) {
	factory, err := NewBuilder().Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	var got []string
	if err := rt.AddCloser(func(context.Context) error {
		got = append(got, "first")
		return nil
	}); err != nil {
		t.Fatalf("add first closer: %v", err)
	}
	if err := rt.AddCloser(func(context.Context) error {
		got = append(got, "second")
		return nil
	}); err != nil {
		t.Fatalf("add second closer: %v", err)
	}

	if err := rt.Close(context.Background()); err != nil {
		t.Fatalf("close runtime: %v", err)
	}

	want := []string{"second", "first"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("close order = %v, want %v", got, want)
	}
}

func TestRuntimeCloseRunsRegistrarClosers(t *testing.T) {
	registrar := &testRuntimeModuleRegistrar{}
	factory, err := NewBuilder().
		WithRuntimeModuleRegistrars(registrar).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	if err := rt.Close(context.Background()); err != nil {
		t.Fatalf("close runtime: %v", err)
	}

	closed := registrar.closedIDs()
	if len(closed) != 1 || closed[0] != 1 {
		t.Fatalf("closed ids = %v, want [1]", closed)
	}
}
