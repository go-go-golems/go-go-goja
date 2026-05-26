package runtimebridge

import (
	"context"
	"testing"

	"github.com/dop251/goja"
)

type contextKey string

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
