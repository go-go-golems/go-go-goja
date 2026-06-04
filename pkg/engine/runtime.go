package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"

	// Blank imports ensure module init() functions run so they can register
	// themselves in modules.DefaultRegistry. A plain NewRuntimeFactoryBuilder().Build() exposes
	// this default registry; callers can restrict it with UseModuleMiddleware
	// (e.g. MiddlewareSafe or MiddlewareOnly).
	_ "github.com/go-go-golems/go-go-goja/modules/crypto"
	_ "github.com/go-go-golems/go-go-goja/modules/database"
	_ "github.com/go-go-golems/go-go-goja/modules/events"
	_ "github.com/go-go-golems/go-go-goja/modules/exec"
	_ "github.com/go-go-golems/go-go-goja/modules/fs"
	_ "github.com/go-go-golems/go-go-goja/modules/os"
	_ "github.com/go-go-golems/go-go-goja/modules/path"
	_ "github.com/go-go-golems/go-go-goja/modules/time"
	_ "github.com/go-go-golems/go-go-goja/modules/timer"
	_ "github.com/go-go-golems/go-go-goja/modules/yaml"
)

// Runtime is an owned runtime instance with explicit lifecycle.
type Runtime struct {
	VM      *goja.Runtime
	Require *require.RequireModule
	Loop    *eventloop.EventLoop
	Owner   runtimeowner.RuntimeOwner
	Values  map[string]any

	runtimeCtx       context.Context
	runtimeCtxCancel context.CancelFunc

	closeOnce sync.Once
	closerMu  sync.Mutex
	closers   []func(context.Context) error
	closing   bool
}

// Value returns runtime-scoped data produced during runtime setup.
func (r *Runtime) Value(key string) (any, bool) {
	if r == nil || r.Values == nil || key == "" {
		return nil, false
	}
	value, ok := r.Values[key]
	return value, ok
}

// Context returns the runtime-owned lifecycle context.
func (r *Runtime) Context() context.Context {
	if r == nil || r.runtimeCtx == nil {
		return context.Background()
	}
	return r.runtimeCtx
}

// AddCloser registers a cleanup hook that is executed before the runtime owner
// and event loop are shut down.
func (r *Runtime) AddCloser(fn func(context.Context) error) error {
	if r == nil {
		return fmt.Errorf("runtime is nil")
	}
	if fn == nil {
		return fmt.Errorf("runtime closer is nil")
	}

	r.closerMu.Lock()
	defer r.closerMu.Unlock()

	if r.closing {
		return fmt.Errorf("runtime is closing or closed")
	}
	r.closers = append(r.closers, fn)
	return nil
}

// Close shuts down runtime-owned resources.
func (r *Runtime) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}

	var retErr error
	r.closeOnce.Do(func() {
		r.closerMu.Lock()
		r.closing = true
		closers := append([]func(context.Context) error(nil), r.closers...)
		r.closers = nil
		r.closerMu.Unlock()

		if r.runtimeCtxCancel != nil {
			r.runtimeCtxCancel()
		}
		if r.Owner != nil {
			if err := r.waitOwnerIdleOrInterrupt(ctx); err != nil {
				retErr = errors.Join(retErr, err)
			}
		}

		for i := len(closers) - 1; i >= 0; i-- {
			if err := closers[i](ctx); err != nil {
				retErr = errors.Join(retErr, err)
			}
		}
		if r.VM != nil {
			runtimebridge.Delete(r.VM)
		}
		if r.Owner != nil {
			if err := r.Owner.Shutdown(ctx); err != nil {
				retErr = errors.Join(retErr, err)
			}
		}
		if r.Loop != nil {
			r.Loop.Stop()
		}
	})

	return retErr
}

func (r *Runtime) waitOwnerIdleOrInterrupt(ctx context.Context) error {
	if r == nil || r.Owner == nil {
		return nil
	}
	waitCtx, cancel := closeWaitContext(ctx)
	defer cancel()
	if err := r.Owner.WaitIdle(waitCtx); err == nil {
		return nil
	}
	if r.VM == nil {
		return nil
	}
	interruptErr := fmt.Errorf("runtime close interrupted active JavaScript")
	r.VM.Interrupt(interruptErr)
	defer r.VM.ClearInterrupt()

	waitCtx, cancel = closeWaitContext(ctx)
	defer cancel()
	if err := r.Owner.WaitIdle(waitCtx); err != nil {
		return err
	}
	return nil
}

func closeWaitContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, 250*time.Millisecond)
}
