package runtimeowner

import (
	"context"

	"github.com/dop251/goja"
)

// Scheduler serializes VM work onto the runtime owner context.
type Scheduler interface {
	RunOnLoop(fn func(*goja.Runtime)) bool
}

// CallFunc is executed against the runtime owner context.
type CallFunc func(context.Context, *goja.Runtime) (any, error)

// PostFunc is executed against the runtime owner context without a return value.
type PostFunc func(context.Context, *goja.Runtime)

// Runner provides safe request/response and fire-and-forget execution against a goja runtime.
type Runner interface {
	Call(ctx context.Context, op string, fn CallFunc) (any, error)
	Post(ctx context.Context, op string, fn PostFunc) error
	Shutdown(ctx context.Context) error
	IsClosed() bool
}

// Options configures a runner.
type Options struct {
	Name          string
	MaxWait       int64 // milliseconds; <=0 disables implicit timeout
	RecoverPanics bool
}
