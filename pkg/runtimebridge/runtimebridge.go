package runtimebridge

import (
	"context"
	"errors"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

// OwnerRunner is the owner-thread scheduling subset exposed to modules that
// need to settle asynchronous JavaScript values from background goroutines.
//
// It intentionally lives in runtimebridge instead of importing
// pkg/runtimeowner. That keeps runtimebridge usable from runtimeowner itself
// for current-call context tracking without creating an import cycle.
type OwnerRunner interface {
	Post(ctx context.Context, op string, fn func(context.Context, *goja.Runtime)) error
}

// Bindings exposes runtime-owned scheduling primitives for modules that need
// async owner-thread settlement.
type Bindings struct {
	Context context.Context
	Loop    *eventloop.EventLoop
	Owner   OwnerRunner
}

var bindingsByVM sync.Map

// Store registers runtime bindings for a concrete VM.
func Store(vm *goja.Runtime, bindings Bindings) {
	if vm == nil {
		return
	}
	bindingsByVM.Store(vm, bindings)
}

// Lookup returns the bindings registered for a concrete VM.
func Lookup(vm *goja.Runtime) (Bindings, bool) {
	if vm == nil {
		return Bindings{}, false
	}
	value, ok := bindingsByVM.Load(vm)
	if !ok {
		return Bindings{}, false
	}
	bindings, ok := value.(Bindings)
	if !ok {
		return Bindings{}, false
	}
	return bindings, true
}

// Delete removes runtime bindings for a concrete VM.
func Delete(vm *goja.Runtime) {
	if vm == nil {
		return
	}
	bindingsByVM.Delete(vm)
	callContextsByVM.Delete(vm)
}

type callContextStack struct {
	mu    sync.Mutex
	stack []context.Context
}

var callContextsByVM sync.Map

// CurrentContext returns the context active for the current owner call on vm.
// If no owner call context is active, it falls back to the runtime lifecycle
// context stored in Bindings. If neither exists, it returns context.Background().
//
// Native modules can call this from JavaScript-exported functions to inherit
// HTTP cancellation, deadlines, and OpenTelemetry parent spans without exposing
// Go context objects to JavaScript authors.
func CurrentContext(vm *goja.Runtime) context.Context {
	if vm == nil {
		return context.Background()
	}
	if st, ok := lookupCallContextStack(vm); ok {
		if ctx, ok := st.peek(); ok && ctx != nil {
			return ctx
		}
	}
	if bindings, ok := Lookup(vm); ok && bindings.Context != nil {
		return bindings.Context
	}
	return context.Background()
}

// WithCallContext makes ctx the current owner-call context for vm while fn
// executes. Contexts are stored as a stack so nested owner calls restore the
// outer context even when fn returns an error or panics.
func WithCallContext(vm *goja.Runtime, ctx context.Context, fn func() (any, error)) (any, error) {
	if vm == nil {
		return nil, errors.New("runtimebridge: nil runtime")
	}
	if fn == nil {
		return nil, errors.New("runtimebridge: nil function")
	}
	if ctx == nil {
		ctx = CurrentContext(vm)
	}
	st := getCallContextStack(vm)
	st.push(ctx)
	defer st.pop()
	return fn()
}

// WithCallContextVoid is the fire-and-forget form of WithCallContext.
func WithCallContextVoid(vm *goja.Runtime, ctx context.Context, fn func() error) error {
	if fn == nil {
		return errors.New("runtimebridge: nil function")
	}
	_, err := WithCallContext(vm, ctx, func() (any, error) {
		return nil, fn()
	})
	return err
}

func getCallContextStack(vm *goja.Runtime) *callContextStack {
	value, _ := callContextsByVM.LoadOrStore(vm, &callContextStack{})
	return value.(*callContextStack)
}

func lookupCallContextStack(vm *goja.Runtime) (*callContextStack, bool) {
	value, ok := callContextsByVM.Load(vm)
	if !ok {
		return nil, false
	}
	st, ok := value.(*callContextStack)
	return st, ok
}

func (s *callContextStack) push(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stack = append(s.stack, ctx)
}

func (s *callContextStack) pop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.stack) == 0 {
		return
	}
	last := len(s.stack) - 1
	s.stack[last] = nil
	s.stack = s.stack[:last]
}

func (s *callContextStack) peek() (context.Context, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.stack) == 0 {
		return nil, false
	}
	return s.stack[len(s.stack)-1], true
}
