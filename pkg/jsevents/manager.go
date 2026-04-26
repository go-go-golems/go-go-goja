package jsevents

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/engine"
	eventsmodule "github.com/go-go-golems/go-go-goja/modules/events"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

const RuntimeValueKey = "jsevents.manager"

// ErrorHandler receives asynchronous listener or scheduling errors.
type ErrorHandler func(error)

// ValueBuilder builds owner-thread JavaScript values for an emitted event.
type ValueBuilder func(vm *goja.Runtime) ([]goja.Value, error)

// Manager owns connected EventEmitter references for one goja runtime.
type Manager struct {
	ctx     context.Context
	owner   runtimeowner.Runner
	onError ErrorHandler

	nextID atomic.Uint64

	mu     sync.RWMutex
	refs   map[string]*EmitterRef
	closed bool
}

// EmitterRef is a Go handle to one Go-native EventEmitter that JavaScript owns
// or received. It is safe to hold from background goroutines, but all emission
// is scheduled onto the owning runtime.
type EmitterRef struct {
	manager *Manager
	id      string
	emitter *eventsmodule.EventEmitter
	object  *goja.Object

	closeOnce sync.Once
	cancelMu  sync.Mutex
	cancel    context.CancelFunc

	closed atomic.Bool
}

type options struct {
	onError ErrorHandler
}

// Option configures the connected-emitter manager.
type Option func(*options)

// WithErrorHandler installs a callback for asynchronous errors.
func WithErrorHandler(fn ErrorHandler) Option {
	return func(opts *options) {
		opts.onError = fn
	}
}

type installer struct {
	opts options
}

// Install returns a runtime initializer that installs a connected-emitter
// manager. It does not create any emitters or connect to any Go-side resource by
// itself.
func Install(opts ...Option) engine.RuntimeInitializer {
	cfg := options{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &installer{opts: cfg}
}

func (i *installer) ID() string { return "jsevents.manager" }

func (i *installer) InitRuntime(ctx *engine.RuntimeContext) error {
	if ctx == nil || ctx.Owner == nil {
		return fmt.Errorf("jsevents: incomplete runtime context")
	}
	manager := &Manager{
		ctx:     ctx.Context,
		owner:   ctx.Owner,
		onError: i.opts.onError,
		refs:    map[string]*EmitterRef{},
	}
	ctx.SetValue(RuntimeValueKey, manager)
	return nil
}

// FromRuntime returns the connected-emitter manager installed in rt.
func FromRuntime(rt *engine.Runtime) (*Manager, bool) {
	if rt == nil {
		return nil, false
	}
	value, ok := rt.Value(RuntimeValueKey)
	if !ok {
		return nil, false
	}
	manager, ok := value.(*Manager)
	return manager, ok && manager != nil
}

// AdoptEmitterOnOwner adopts a JavaScript-created Go-native EventEmitter value.
// It must be called on the owning runtime goroutine.
func (m *Manager) AdoptEmitterOnOwner(value goja.Value) (*EmitterRef, error) {
	if m == nil {
		return nil, fmt.Errorf("jsevents: nil manager")
	}
	emitter, object, ok := eventsmodule.FromValue(value)
	if !ok {
		return nil, fmt.Errorf("jsevents: value is not an events.EventEmitter")
	}
	return m.registerEmitter(emitter, object)
}

func (m *Manager) registerEmitter(emitter *eventsmodule.EventEmitter, object *goja.Object) (*EmitterRef, error) {
	if emitter == nil || object == nil {
		return nil, fmt.Errorf("jsevents: nil emitter")
	}
	id := fmt.Sprintf("emitter-%d", m.nextID.Add(1))
	ref := &EmitterRef{
		manager: m,
		id:      id,
		emitter: emitter,
		object:  object,
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, fmt.Errorf("jsevents: manager is closed")
	}
	m.refs[id] = ref
	return ref, nil
}

// ID returns the stable Go-side identifier for this connected emitter.
func (r *EmitterRef) ID() string {
	if r == nil {
		return ""
	}
	return r.id
}

// SetCancel registers resource cleanup for Close. It is safe to call at most a
// few times during setup; the latest cancel function wins until Close runs.
func (r *EmitterRef) SetCancel(cancel context.CancelFunc) {
	if r == nil {
		return
	}
	r.cancelMu.Lock()
	defer r.cancelMu.Unlock()
	r.cancel = cancel
}

// Emit schedules asynchronous delivery of one event to the connected emitter.
func (r *EmitterRef) Emit(ctx context.Context, name string, args ...any) error {
	copied := append([]any(nil), args...)
	return r.EmitWithBuilder(ctx, name, func(vm *goja.Runtime) ([]goja.Value, error) {
		values := make([]goja.Value, 0, len(copied))
		for _, arg := range copied {
			values = append(values, vm.ToValue(arg))
		}
		return values, nil
	})
}

// EmitWithBuilder schedules asynchronous delivery using owner-thread value
// construction.
func (r *EmitterRef) EmitWithBuilder(ctx context.Context, name string, builder ValueBuilder) error {
	if r == nil || r.manager == nil {
		return fmt.Errorf("jsevents: nil emitter ref")
	}
	if builder == nil {
		return fmt.Errorf("jsevents: nil value builder")
	}
	if ctx == nil {
		ctx = r.manager.ctx
	}
	if r.closed.Load() {
		return fmt.Errorf("jsevents: emitter %s is closed", r.id)
	}
	return r.manager.owner.Post(ctx, "jsevents.emit."+r.id+"."+name, func(_ context.Context, vm *goja.Runtime) {
		if r.closed.Load() {
			return
		}
		args, err := builder(vm)
		if err == nil {
			_, err = r.emitter.Emit(name, args...)
		}
		if err != nil {
			r.manager.report(err)
		}
	})
}

// EmitSync emits one event and waits until listener dispatch completes.
func (r *EmitterRef) EmitSync(ctx context.Context, name string, args ...any) (bool, error) {
	copied := append([]any(nil), args...)
	return r.EmitWithBuilderSync(ctx, name, func(vm *goja.Runtime) ([]goja.Value, error) {
		values := make([]goja.Value, 0, len(copied))
		for _, arg := range copied {
			values = append(values, vm.ToValue(arg))
		}
		return values, nil
	})
}

// EmitWithBuilderSync emits one event with owner-thread value construction and
// returns the EventEmitter.emit boolean result.
func (r *EmitterRef) EmitWithBuilderSync(ctx context.Context, name string, builder ValueBuilder) (bool, error) {
	if r == nil || r.manager == nil {
		return false, fmt.Errorf("jsevents: nil emitter ref")
	}
	if builder == nil {
		return false, fmt.Errorf("jsevents: nil value builder")
	}
	if ctx == nil {
		ctx = r.manager.ctx
	}
	if r.closed.Load() {
		return false, fmt.Errorf("jsevents: emitter %s is closed", r.id)
	}
	ret, err := r.manager.owner.Call(ctx, "jsevents.emitSync."+r.id+"."+name, func(_ context.Context, vm *goja.Runtime) (any, error) {
		if r.closed.Load() {
			return false, fmt.Errorf("jsevents: emitter %s is closed", r.id)
		}
		args, err := builder(vm)
		if err != nil {
			return false, err
		}
		return r.emitter.Emit(name, args...)
	})
	if err != nil {
		return false, err
	}
	delivered, _ := ret.(bool)
	return delivered, nil
}

// Close cancels the Go-side resource and unregisters the connected emitter.
func (r *EmitterRef) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	var err error
	r.closeOnce.Do(func() {
		r.closed.Store(true)
		r.cancelMu.Lock()
		cancel := r.cancel
		r.cancel = nil
		r.cancelMu.Unlock()
		if cancel != nil {
			cancel()
		}
		if r.manager == nil {
			return
		}
		r.manager.mu.Lock()
		delete(r.manager.refs, r.id)
		r.manager.mu.Unlock()
	})
	return err
}

func (m *Manager) report(err error) {
	if err != nil && m != nil && m.onError != nil {
		m.onError(err)
	}
}
