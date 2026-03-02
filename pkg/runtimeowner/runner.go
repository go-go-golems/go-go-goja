package runtimeowner

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/dop251/goja"
)

type ownerCtxKey struct{}

type ownerCtxValue struct {
	r   *runner
	gid uint64
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
	ctx = normalizeContext(ctx)
	var cancel context.CancelFunc
	if r.opts.MaxWait > 0 {
		if _, ok := ctx.Deadline(); !ok {
			ctx, cancel = context.WithTimeout(ctx, time.Duration(r.opts.MaxWait)*time.Millisecond)
			defer cancel()
		}
	}

	if r.isOwnerContext(ctx) {
		return r.invoke(ctx, op, fn)
	}

	resultCh := make(chan callResult, 1)
	accepted := r.scheduler.RunOnLoop(func(vm *goja.Runtime) {
		ownerCtx := r.withOwnerContext(ctx)
		select {
		case <-ownerCtx.Done():
			resultCh <- callResult{
				err: fmt.Errorf("runtimeowner %s: %w: %v", op, ErrCanceled, ownerCtx.Err()),
			}
			return
		default:
		}
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
	ctx = normalizeContext(ctx)
	var cancel context.CancelFunc
	if r.opts.MaxWait > 0 {
		if _, ok := ctx.Deadline(); !ok {
			ctx, cancel = context.WithTimeout(ctx, time.Duration(r.opts.MaxWait)*time.Millisecond)
		}
	}

	select {
	case <-ctx.Done():
		if cancel != nil {
			cancel()
		}
		return fmt.Errorf("runtimeowner %s: %w: %v", op, ErrCanceled, ctx.Err())
	default:
	}

	if r.isOwnerContext(ctx) {
		if cancel != nil {
			defer cancel()
		}
		r.invokePost(ctx, op, fn)
		return nil
	}

	accepted := r.scheduler.RunOnLoop(func(vm *goja.Runtime) {
		if cancel != nil {
			defer cancel()
		}
		ownerCtx := r.withOwnerContext(ctx)
		select {
		case <-ownerCtx.Done():
			return
		default:
		}
		r.invokePost(ownerCtx, op, fn)
	})
	if !accepted {
		if cancel != nil {
			cancel()
		}
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

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx
}

func (r *runner) withOwnerContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ownerCtxKey{}, ownerCtxValue{
		r:   r,
		gid: currentGoroutineID(),
	})
}

func (r *runner) isOwnerContext(ctx context.Context) bool {
	v, ok := ctx.Value(ownerCtxKey{}).(ownerCtxValue)
	if !ok || v.r != r || v.gid == 0 {
		return false
	}
	return v.gid == currentGoroutineID()
}

func currentGoroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	parts := bytes.Fields(buf[:n])
	// Expected prefix: "goroutine <id> ..."
	if len(parts) < 2 {
		return 0
	}
	id, err := strconv.ParseUint(string(parts[1]), 10, 64)
	if err != nil {
		return 0
	}
	return id
}
