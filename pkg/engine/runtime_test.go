package engine

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
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

func TestNewRuntimeUsesStartupContextForInitializers(t *testing.T) {
	startupCtx := context.WithValue(context.Background(), contextKey("startup"), "yes")
	factory, err := NewBuilder().WithRuntimeInitializers(testRuntimeInitializer{
		id: "startup-context",
		fn: func(ctx *RuntimeContext) error {
			if got := ctx.Context.Value(contextKey("startup")); got != "yes" {
				return fmt.Errorf("startup context value = %#v, want yes", got)
			}
			return nil
		},
	}).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(WithStartupContext(startupCtx), WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
}

func TestNewRuntimeRejectsCanceledStartupContext(t *testing.T) {
	startupCtx, cancel := context.WithCancel(context.Background())
	cancel()
	factory, err := NewBuilder().Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	if _, err := factory.NewRuntime(WithStartupContext(startupCtx), WithLifetimeContext(context.Background())); err == nil {
		t.Fatalf("expected canceled startup context error")
	}
}

func TestRuntimeContextUsesLifetimeContext(t *testing.T) {
	lifetimeCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	factory, err := NewBuilder().Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(WithStartupContext(context.Background()), WithLifetimeContext(lifetimeCtx))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	runtimeCtx := rt.Context()
	cancel()

	select {
	case <-runtimeCtx.Done():
	case <-time.After(2 * time.Second):
		t.Fatalf("runtime context did not follow lifetime context cancellation")
	}
}

func TestRuntimeCloseRunsClosersBeforeDeletingRuntimeServices(t *testing.T) {
	factory, err := NewBuilder().Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(WithStartupContext(context.Background()), WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	closerSawServices := false
	if err := rt.AddCloser(func(context.Context) error {
		services, ok := runtimebridge.Lookup(rt.VM)
		closerSawServices = ok && services.Owner != nil
		return nil
	}); err != nil {
		t.Fatalf("add closer: %v", err)
	}

	if err := rt.Close(context.Background()); err != nil {
		t.Fatalf("close runtime: %v", err)
	}
	if !closerSawServices {
		t.Fatalf("closer did not see runtime services before cleanup")
	}
	if _, ok := runtimebridge.Lookup(rt.VM); ok {
		t.Fatalf("runtime services still registered after close")
	}
}

func TestRuntimeContextCancelsOnClose(t *testing.T) {
	factory, err := NewBuilder().Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(WithStartupContext(context.Background()), WithLifetimeContext(context.Background()))
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

type contextKey string

type testRuntimeInitializer struct {
	id string
	fn func(*RuntimeContext) error
}

func (i testRuntimeInitializer) ID() string { return i.id }

func (i testRuntimeInitializer) InitRuntime(ctx *RuntimeContext) error {
	if i.fn == nil {
		return nil
	}
	return i.fn(ctx)
}
