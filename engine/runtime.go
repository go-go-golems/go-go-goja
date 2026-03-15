package engine

import (
	"context"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"

	// Blank imports ensure module init() functions run so they can register
	// themselves in modules.DefaultRegistry. Registration is still explicit:
	// callers must opt in via DefaultRegistryModules().
	_ "github.com/go-go-golems/go-go-goja/modules/database"
	_ "github.com/go-go-golems/go-go-goja/modules/exec"
	_ "github.com/go-go-golems/go-go-goja/modules/fs"
	_ "github.com/go-go-golems/go-go-goja/modules/glazehelp"
)

// Runtime is an owned runtime instance with explicit lifecycle.
type Runtime struct {
	VM      *goja.Runtime
	Require *require.RequireModule
	Loop    *eventloop.EventLoop
	Owner   runtimeowner.Runner

	closeOnce sync.Once
}

// Close shuts down runtime-owned resources.
func (r *Runtime) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}

	var retErr error
	r.closeOnce.Do(func() {
		if r.Owner != nil {
			if err := r.Owner.Shutdown(ctx); err != nil && retErr == nil {
				retErr = err
			}
		}
		if r.Loop != nil {
			r.Loop.Stop()
		}
	})

	return retErr
}
