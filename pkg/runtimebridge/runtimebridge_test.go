package runtimebridge

import (
	"context"
	"testing"

	"github.com/dop251/goja"
)

type contextKey string

func TestCurrentContextFallsBackToLifecycleContext(t *testing.T) {
	vm := goja.New()
	defer Delete(vm)

	ctx := context.WithValue(context.Background(), contextKey("lifecycle"), "runtime")
	Store(vm, Bindings{Context: ctx})

	got := CurrentContext(vm)
	if got.Value(contextKey("lifecycle")) != "runtime" {
		t.Fatalf("CurrentContext() = %#v, want lifecycle context", got)
	}
}

func TestWithCallContextPushesAndRestoresNestedContext(t *testing.T) {
	vm := goja.New()
	defer Delete(vm)

	outer := context.WithValue(context.Background(), contextKey("request"), "outer")
	inner := context.WithValue(context.Background(), contextKey("request"), "inner")

	_, err := WithCallContext(vm, outer, func() (any, error) {
		if got := CurrentContext(vm).Value(contextKey("request")); got != "outer" {
			t.Fatalf("outer CurrentContext() = %#v, want outer", got)
		}
		_, err := WithCallContext(vm, inner, func() (any, error) {
			if got := CurrentContext(vm).Value(contextKey("request")); got != "inner" {
				t.Fatalf("inner CurrentContext() = %#v, want inner", got)
			}
			return nil, nil
		})
		if err != nil {
			return nil, err
		}
		if got := CurrentContext(vm).Value(contextKey("request")); got != "outer" {
			t.Fatalf("restored CurrentContext() = %#v, want outer", got)
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
	Store(vm, Bindings{Context: lifecycle})

	func() {
		defer func() { _ = recover() }()
		_, _ = WithCallContext(vm, call, func() (any, error) {
			panic("boom")
		})
	}()

	if got := CurrentContext(vm).Value(contextKey("scope")); got != "lifecycle" {
		t.Fatalf("CurrentContext() after panic = %#v, want lifecycle", got)
	}
}
