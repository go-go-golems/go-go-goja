package runtimebridge

import (
	"context"
	"testing"
	"time"

	"github.com/dop251/goja"
)

type contextKey string

type fakeRuntimeOwner struct{}

func (fakeRuntimeOwner) Call(ctx context.Context, _ string, fn func(context.Context, *goja.Runtime) (any, error)) (any, error) {
	return fn(ctx, goja.New())
}

func (fakeRuntimeOwner) Post(ctx context.Context, _ string, fn func(context.Context, *goja.Runtime)) error {
	fn(ctx, goja.New())
	return nil
}

func TestCurrentOwnerContextFallsBackToLifetimeContext(t *testing.T) {
	vm := goja.New()
	defer Delete(vm)

	ctx := context.WithValue(context.Background(), contextKey("lifecycle"), "runtime")
	Store(vm, RuntimeServices{LifetimeContext: ctx})

	got := CurrentOwnerContext(vm)
	if got.Value(contextKey("lifecycle")) != "runtime" {
		t.Fatalf("CurrentOwnerContext() = %#v, want lifetime context", got)
	}
}

func TestWithCallContextPushesAndRestoresNestedContext(t *testing.T) {
	vm := goja.New()
	defer Delete(vm)

	outer := context.WithValue(context.Background(), contextKey("request"), "outer")
	inner := context.WithValue(context.Background(), contextKey("request"), "inner")

	_, err := WithCallContext(vm, outer, func() (any, error) {
		if got := CurrentOwnerContext(vm).Value(contextKey("request")); got != "outer" {
			t.Fatalf("outer CurrentOwnerContext() = %#v, want outer", got)
		}
		_, err := WithCallContext(vm, inner, func() (any, error) {
			if got := CurrentOwnerContext(vm).Value(contextKey("request")); got != "inner" {
				t.Fatalf("inner CurrentOwnerContext() = %#v, want inner", got)
			}
			return nil, nil
		})
		if err != nil {
			return nil, err
		}
		if got := CurrentOwnerContext(vm).Value(contextKey("request")); got != "outer" {
			t.Fatalf("restored CurrentOwnerContext() = %#v, want outer", got)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("WithCallContext() returned error: %v", err)
	}
}

func TestRuntimeServicesCallWithCustomContextCancelsWhenLifetimeCancels(t *testing.T) {
	lifetimeCtx, cancelLifetime := context.WithCancel(context.Background())
	svc := RuntimeServices{
		LifetimeContext: lifetimeCtx,
		Owner:           fakeRuntimeOwner{},
	}

	customCtx := context.WithValue(context.Background(), contextKey("request"), "custom")
	_, err := svc.CallWithCustomContext(customCtx, "test.cancel", func(ctx context.Context, _ *goja.Runtime) (any, error) {
		if got := ctx.Value(contextKey("request")); got != "custom" {
			t.Fatalf("linked context value = %#v, want custom", got)
		}
		cancelLifetime()
		select {
		case <-ctx.Done():
			return nil, nil
		case <-time.After(2 * time.Second):
			t.Fatalf("linked context did not cancel when lifetime canceled")
			return nil, nil
		}
	})
	if err != nil {
		t.Fatalf("CallWithCustomContext() returned error: %v", err)
	}
}

func TestRuntimeServicesCallWithCustomContextCancelsLinkedContextAfterCall(t *testing.T) {
	svc := RuntimeServices{
		LifetimeContext: context.Background(),
		Owner:           fakeRuntimeOwner{},
	}

	var callCtx context.Context
	customCtx := context.WithValue(context.Background(), contextKey("request"), "call")
	_, err := svc.CallWithCustomContext(customCtx, "test.cleanup.call", func(ctx context.Context, _ *goja.Runtime) (any, error) {
		callCtx = ctx
		return nil, nil
	})
	if err != nil {
		t.Fatalf("CallWithCustomContext() returned error: %v", err)
	}
	select {
	case <-callCtx.Done():
	case <-time.After(2 * time.Second):
		t.Fatalf("linked call context was not canceled after call returned")
	}
}

func TestRuntimeServicesPostWithNilContextUsesLifetimeContext(t *testing.T) {
	lifetimeCtx := context.WithValue(context.Background(), contextKey("lifecycle"), "runtime")
	svc := RuntimeServices{
		LifetimeContext: lifetimeCtx,
		Owner:           fakeRuntimeOwner{},
	}

	var nilCtx context.Context
	if err := svc.PostWithCustomContext(nilCtx, "test.nil", func(ctx context.Context, _ *goja.Runtime) {
		if got := ctx.Value(contextKey("lifecycle")); got != "runtime" {
			t.Fatalf("post context value = %#v, want runtime", got)
		}
	}); err != nil {
		t.Fatalf("PostWithCustomContext() returned error: %v", err)
	}
}

func TestWithCallContextPopsAfterPanic(t *testing.T) {
	vm := goja.New()
	defer Delete(vm)

	lifecycle := context.WithValue(context.Background(), contextKey("scope"), "lifecycle")
	call := context.WithValue(context.Background(), contextKey("scope"), "call")
	Store(vm, RuntimeServices{LifetimeContext: lifecycle})

	func() {
		defer func() { _ = recover() }()
		_, _ = WithCallContext(vm, call, func() (any, error) {
			panic("boom")
		})
	}()

	if got := CurrentOwnerContext(vm).Value(contextKey("scope")); got != "lifecycle" {
		t.Fatalf("CurrentOwnerContext() after panic = %#v, want lifecycle", got)
	}
}
