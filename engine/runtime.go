package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"

	// Blank imports ensure module init() functions run so they can register
	// themselves in modules.DefaultRegistry. Registration is still explicit:
	// callers must opt in via DefaultRegistryModules().
	_ "github.com/go-go-golems/go-go-goja/modules/database"
	_ "github.com/go-go-golems/go-go-goja/modules/exec"
	_ "github.com/go-go-golems/go-go-goja/modules/fs"
	_ "github.com/go-go-golems/go-go-goja/modules/sandbox"
	_ "github.com/go-go-golems/go-go-goja/modules/timer"
)

// Runtime is an owned runtime instance with explicit lifecycle.
type Runtime struct {
	VM      *goja.Runtime
	Require *require.RequireModule
	Loop    *eventloop.EventLoop
	Owner   runtimeowner.Runner
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
		if r.VM != nil {
			runtimebridge.Delete(r.VM)
		}

		for i := len(closers) - 1; i >= 0; i-- {
			if err := closers[i](ctx); err != nil {
				retErr = errors.Join(retErr, err)
			}
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
