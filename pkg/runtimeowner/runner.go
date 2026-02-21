package runtimeowner

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/dop251/goja"
)

type ownerCtxKey struct{}

type ownerCtxValue struct {
	r *runner
}

type callResult struct {
	value any
	err   error
}

type runner struct {
	vm        *goja.Runtime
	scheduler Scheduler
	opts      Options
	closed    atomic.Bool
}

func NewRunner(vm *goja.Runtime, scheduler Scheduler, opts Options) Runner {
	if vm == nil {
		panic("runtimeowner: vm is nil")
	}
	if scheduler == nil {
		panic("runtimeowner: scheduler is nil")
	}
	if opts.Name == "" {
		opts.Name = "runtime"
	}
	return &runner{vm: vm, scheduler: scheduler, opts: opts}
}

func (r *runner) IsClosed() bool {
	if r == nil {
		return true
	}
	return r.closed.Load()
}

func (r *runner) Shutdown(context.Context) error {
	if r == nil {
		return nil
	}
	r.closed.Store(true)
	return nil
}

func (r *runner) Call(ctx context.Context, op string, fn CallFunc) (any, error) {
	if r == nil || fn == nil {
		return nil, fmt.Errorf("runtimeowner %s: nil runner or function", op)
	}
	if r.closed.Load() {
		return nil, ErrClosed
	}
	ctx, cancel := r.normalizeContext(ctx)
	if cancel != nil {
		defer cancel()
	}

	if r.isOwnerContext(ctx) {
		return r.invoke(ctx, op, fn)
	}

	resultCh := make(chan callResult, 1)
	accepted := r.scheduler.RunOnLoop(func(vm *goja.Runtime) {
		ownerCtx := r.withOwnerContext(ctx)
		v, err := r.invoke(ownerCtx, op, fn)
		resultCh <- callResult{value: v, err: err}
	})
	if !accepted {
		return nil, fmt.Errorf("runtimeowner %s: %w", op, ErrScheduleRejected)
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("runtimeowner %s: %w: %v", op, ErrCanceled, ctx.Err())
	case res := <-resultCh:
		return res.value, res.err
	}
}

func (r *runner) Post(ctx context.Context, op string, fn PostFunc) error {
	if r == nil || fn == nil {
		return fmt.Errorf("runtimeowner %s: nil runner or function", op)
	}
	if r.closed.Load() {
		return ErrClosed
	}
	ctx, cancel := r.normalizeContext(ctx)
	if cancel != nil {
		defer cancel()
	}
	select {
	case <-ctx.Done():
		return fmt.Errorf("runtimeowner %s: %w: %v", op, ErrCanceled, ctx.Err())
	default:
	}

	if r.isOwnerContext(ctx) {
		r.invokePost(ctx, op, fn)
		return nil
	}

	accepted := r.scheduler.RunOnLoop(func(vm *goja.Runtime) {
		r.invokePost(r.withOwnerContext(ctx), op, fn)
	})
	if !accepted {
		return fmt.Errorf("runtimeowner %s: %w", op, ErrScheduleRejected)
	}
	return nil
}

func (r *runner) invoke(ctx context.Context, op string, fn CallFunc) (any, error) {
	if !r.opts.RecoverPanics {
		return fn(ctx, r.vm)
	}

	var (
		ret any
		err error
	)
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				err = fmt.Errorf("runtimeowner %s: %w: %v", op, ErrPanicked, rec)
				ret = nil
			}
		}()
		ret, err = fn(ctx, r.vm)
	}()
	return ret, err
}

func (r *runner) invokePost(ctx context.Context, op string, fn PostFunc) {
	if r.opts.RecoverPanics {
		defer func() {
			_ = recover()
		}()
	}
	fn(ctx, r.vm)
}

func (r *runner) normalizeContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if r.opts.MaxWait > 0 {
		if _, ok := ctx.Deadline(); !ok {
			timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(r.opts.MaxWait)*time.Millisecond)
			return timeoutCtx, cancel
		}
	}
	return ctx, nil
}

func (r *runner) withOwnerContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ownerCtxKey{}, ownerCtxValue{r: r})
}

func (r *runner) isOwnerContext(ctx context.Context) bool {
	v, ok := ctx.Value(ownerCtxKey{}).(ownerCtxValue)
	return ok && v.r == r
}
