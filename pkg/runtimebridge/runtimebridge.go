package runtimebridge

import (
	"context"
	"errors"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

// RuntimeOwner is the owner-thread scheduling subset exposed to modules that
// need to execute JavaScript or settle asynchronous JavaScript values from
// background goroutines.
//
// It intentionally lives in runtimebridge instead of importing
// pkg/runtimeowner. That keeps runtimebridge usable from runtimeowner itself
// for current-call context tracking without creating an import cycle.
type RuntimeOwner interface {
	Call(ctx context.Context, op string, fn func(context.Context, *goja.Runtime) (any, error)) (any, error)
	Post(ctx context.Context, op string, fn func(context.Context, *goja.Runtime)) error
}

// RuntimeServices exposes runtime-owned scheduling primitives for native
// modules that need async owner-thread settlement.
type RuntimeServices struct {
	LifetimeContext context.Context
	Loop            *eventloop.EventLoop
	Owner           RuntimeOwner
}

// Lifetime returns the runtime lifetime context, or context.Background when no
// lifetime context was registered. It is the context for runtime-owned work,
// not a generic context for callbacks into JavaScript.
func (svc RuntimeServices) Lifetime() context.Context {
	if svc.LifetimeContext != nil {
		return svc.LifetimeContext
	}
	return context.Background()
}

// CallWithCurrentContext invokes fn through the runtime owner using the current
// owner-entry context for vm.
func (svc RuntimeServices) CallWithCurrentContext(vm *goja.Runtime, op string, fn func(context.Context, *goja.Runtime) (any, error)) (any, error) {
	if svc.Owner == nil {
		return nil, errors.New("runtimebridge: missing owner")
	}
	return svc.Owner.Call(CurrentOwnerContext(vm), op, fn)
}

// PostWithCurrentContext posts fn through the runtime owner using the current
// owner-entry context for vm.
func (svc RuntimeServices) PostWithCurrentContext(vm *goja.Runtime, op string, fn func(context.Context, *goja.Runtime)) error {
	if svc.Owner == nil {
		return errors.New("runtimebridge: missing owner")
	}
	return svc.Owner.Post(CurrentOwnerContext(vm), op, fn)
}

// CallWithLifetimeContext invokes fn with the runtime lifetime context.
func (svc RuntimeServices) CallWithLifetimeContext(op string, fn func(context.Context, *goja.Runtime) (any, error)) (any, error) {
	return svc.CallWithCustomContext(svc.Lifetime(), op, fn)
}

// PostWithLifetimeContext posts fn with the runtime lifetime context.
func (svc RuntimeServices) PostWithLifetimeContext(op string, fn func(context.Context, *goja.Runtime)) error {
	return svc.PostWithCustomContext(svc.Lifetime(), op, fn)
}

// CallWithCustomContext invokes fn with an explicit caller-provided context. A
// nil context falls back to the runtime lifetime context.
func (svc RuntimeServices) CallWithCustomContext(ctx context.Context, op string, fn func(context.Context, *goja.Runtime) (any, error)) (any, error) {
	if svc.Owner == nil {
		return nil, errors.New("runtimebridge: missing owner")
	}
	linked := svc.contextLinkedToLifetime(ctx)
	defer linked.cancel()
	return svc.Owner.Call(linked.ctx, op, fn)
}

// PostWithCustomContext posts fn with an explicit caller-provided context. A
// nil context falls back to the runtime lifetime context.
func (svc RuntimeServices) PostWithCustomContext(ctx context.Context, op string, fn func(context.Context, *goja.Runtime)) error {
	if svc.Owner == nil {
		return errors.New("runtimebridge: missing owner")
	}
	if fn == nil {
		return errors.New("runtimebridge: nil function")
	}
	linked := svc.contextLinkedToLifetime(ctx)
	posted := false
	defer func() {
		if !posted {
			linked.cancel()
		}
	}()
	if err := svc.Owner.Post(linked.ctx, op, func(ctx context.Context, vm *goja.Runtime) {
		defer linked.stop()
		fn(ctx, vm)
	}); err != nil {
		return err
	}
	posted = true
	return nil
}

type linkedContext struct {
	ctx    context.Context
	cancel context.CancelFunc
	stop   func()
}

func (svc RuntimeServices) contextLinkedToLifetime(ctx context.Context) linkedContext {
	lifetime := svc.Lifetime()
	if ctx == nil || ctx == lifetime {
		return linkedContext{ctx: lifetime, cancel: func() {}, stop: func() {}}
	}
	select {
	case <-lifetime.Done():
		return linkedContext{ctx: lifetime, cancel: func() {}, stop: func() {}}
	default:
	}
	linked, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(lifetime, cancel)
	return linkedContext{
		ctx: linked,
		cancel: func() {
			_ = stop()
			cancel()
		},
		stop: func() {
			_ = stop()
		},
	}
}

var servicesByVM sync.Map

// Store registers runtime services for a concrete VM.
func Store(vm *goja.Runtime, services RuntimeServices) {
	if vm == nil {
		return
	}
	servicesByVM.Store(vm, services)
}

// Lookup returns the services registered for a concrete VM.
func Lookup(vm *goja.Runtime) (RuntimeServices, bool) {
	if vm == nil {
		return RuntimeServices{}, false
	}
	value, ok := servicesByVM.Load(vm)
	if !ok {
		return RuntimeServices{}, false
	}
	services, ok := value.(RuntimeServices)
	if !ok {
		return RuntimeServices{}, false
	}
	return services, true
}

// Delete removes runtime services for a concrete VM.
func Delete(vm *goja.Runtime) {
	if vm == nil {
		return
	}
	servicesByVM.Delete(vm)
	callContextsByVM.Delete(vm)
}

type callContextStack struct {
	mu    sync.Mutex
	stack []context.Context
}

var callContextsByVM sync.Map

// LifetimeContext returns the registered runtime lifetime context for vm.
func LifetimeContext(vm *goja.Runtime) context.Context {
	if services, ok := Lookup(vm); ok {
		return services.Lifetime()
	}
	return context.Background()
}

// CurrentOwnerContext returns the context active for the current owner call on
// vm. If no owner call context is active, it falls back to the runtime lifetime
// context. Native modules should call this from JavaScript-exported functions
// to inherit request/command cancellation, deadlines, and tracing metadata.
func CurrentOwnerContext(vm *goja.Runtime) context.Context {
	if vm == nil {
		return context.Background()
	}
	if st, ok := lookupCallContextStack(vm); ok {
		if ctx, ok := st.peek(); ok && ctx != nil {
			return ctx
		}
	}
	return LifetimeContext(vm)
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
		ctx = CurrentOwnerContext(vm)
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
