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

type testRegistrarFunc struct {
	id string
	fn func(ctx *RuntimeModuleRegistrationContext, reg *require.Registry) error
}

func (f testRegistrarFunc) ID() string {
	return f.id
}

func (f testRegistrarFunc) RegisterRuntimeModule(ctx *RuntimeModuleRegistrationContext, reg *require.Registry) error {
	return f.fn(ctx, reg)
}

type testRuntimeInitializerFunc struct {
	id string
	fn func(ctx *RuntimeInitializationContext) error
}

func (f testRuntimeInitializerFunc) ID() string {
	return f.id
}

func (f testRuntimeInitializerFunc) InitRuntime(ctx *RuntimeInitializationContext) error {
	return f.fn(ctx)
}

func (r *testRuntimeModuleRegistrar) ID() string {
	return "test-runtime-registrar"
}

func (r *testRuntimeModuleRegistrar) RegisterRuntimeModule(ctx *RuntimeModuleRegistrationContext, reg *require.Registry) error {
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

	factory, err := NewRuntimeFactoryBuilder().
		WithModules(registrar).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt1, err := factory.NewRuntime(WithStartupContext(context.Background()), WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime 1: %v", err)
	}
	defer func() {
		_ = rt1.Close(context.Background())
	}()

	rt2, err := factory.NewRuntime(WithStartupContext(context.Background()), WithLifetimeContext(context.Background()))
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
	factory, err := NewRuntimeFactoryBuilder().Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(WithStartupContext(context.Background()), WithLifetimeContext(context.Background()))
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
	factory, err := NewRuntimeFactoryBuilder().
		WithModules(registrar).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(WithStartupContext(context.Background()), WithLifetimeContext(context.Background()))
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

func TestRuntimePersistsRegistrarValues(t *testing.T) {
	registrar := &testRuntimeModuleRegistrar{}

	factory, err := NewRuntimeFactoryBuilder().
		WithModules(registrar, testRegistrarFunc{id: "value-registrar", fn: func(ctx *RuntimeModuleRegistrationContext, reg *require.Registry) error {
			ctx.SetValue("runtime-id", 42)
			return nil
		}}).
		Build()
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

	value, ok := rt.Value("runtime-id")
	if !ok {
		t.Fatalf("runtime value not found")
	}
	if value != 42 {
		t.Fatalf("runtime value = %#v, want 42", value)
	}
}

func TestRuntimeInitializersCanReadAndWriteRuntimeValues(t *testing.T) {
	factory, err := NewRuntimeFactoryBuilder().
		WithModules(testRegistrarFunc{id: "seed-values", fn: func(ctx *RuntimeModuleRegistrationContext, reg *require.Registry) error {
			ctx.SetValue("phase", "registered")
			return nil
		}}).
		WithRuntimeInitializers(testRuntimeInitializerFunc{id: "read-write-values", fn: func(ctx *RuntimeInitializationContext) error {
			value, ok := ctx.Value("phase")
			if !ok {
				return fmt.Errorf("phase value missing")
			}
			if value != "registered" {
				return fmt.Errorf("phase = %#v", value)
			}
			ctx.SetValue("initializer", "done")
			return nil
		}}).
		Build()
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

	if got, ok := rt.Value("phase"); !ok || got != "registered" {
		t.Fatalf("runtime phase = %#v, %v", got, ok)
	}
	if got, ok := rt.Value("initializer"); !ok || got != "done" {
		t.Fatalf("runtime initializer value = %#v, %v", got, ok)
	}
}

func TestRuntimeInitializersPersistValuesWithoutRegistrarState(t *testing.T) {
	factory, err := NewRuntimeFactoryBuilder().
		WithRuntimeInitializers(testRuntimeInitializerFunc{id: "seed-value", fn: func(ctx *RuntimeInitializationContext) error {
			ctx.SetValue("initializer-only", "present")
			return nil
		}}).
		Build()
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

	if got, ok := rt.Value("initializer-only"); !ok || got != "present" {
		t.Fatalf("runtime initializer-only = %#v, %v", got, ok)
	}
}

func TestBuilderCanDisableImplicitDefaultModules(t *testing.T) {
	factory, err := NewRuntimeFactoryBuilder(
		WithImplicitDefaultRegistryModules(false),
		WithDataOnlyDefaultRegistryModules(false),
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

	if _, err := rt.Require.Require("fs"); err == nil {
		t.Fatalf("require(fs) succeeded, want missing module")
	}
	if _, err := rt.Require.Require("path"); err == nil {
		t.Fatalf("require(path) succeeded, want missing module")
	}
}
