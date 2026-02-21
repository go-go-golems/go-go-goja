package runtimeowner

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dop251/goja"
)

type queueScheduler struct {
	vm     *goja.Runtime
	jobs   chan func(*goja.Runtime)
	closed bool
	mu     sync.Mutex
	wg     sync.WaitGroup
}

func newQueueScheduler(vm *goja.Runtime) *queueScheduler {
	s := &queueScheduler{
		vm:   vm,
		jobs: make(chan func(*goja.Runtime), 2048),
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for job := range s.jobs {
			job(vm)
		}
	}()
	return s
}

func (s *queueScheduler) RunOnLoop(fn func(*goja.Runtime)) bool {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return false
	}
	s.jobs <- fn
	return true
}

func (s *queueScheduler) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	close(s.jobs)
	s.mu.Unlock()
	s.wg.Wait()
}

type rejectScheduler struct{}

func (rejectScheduler) RunOnLoop(func(*goja.Runtime)) bool { return false }

type manualScheduler struct {
	vm     *goja.Runtime
	jobs   chan func(*goja.Runtime)
	closed bool
	mu     sync.Mutex
}

func newManualScheduler(vm *goja.Runtime) *manualScheduler {
	return &manualScheduler{
		vm:   vm,
		jobs: make(chan func(*goja.Runtime), 128),
	}
}

func (s *manualScheduler) RunOnLoop(fn func(*goja.Runtime)) bool {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return false
	}
	s.jobs <- fn
	return true
}

func (s *manualScheduler) RunNext() bool {
	select {
	case job, ok := <-s.jobs:
		if !ok {
			return false
		}
		job(s.vm)
		return true
	default:
		return false
	}
}

func (s *manualScheduler) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	close(s.jobs)
	s.mu.Unlock()
}

func TestRunnerCallSuccess(t *testing.T) {
	vm := goja.New()
	s := newQueueScheduler(vm)
	defer s.Close()

	r := NewRunner(vm, s, Options{RecoverPanics: true})
	got, err := r.Call(context.Background(), "test.success", func(context.Context, *goja.Runtime) (any, error) {
		return 42, nil
	})
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if got.(int) != 42 {
		t.Fatalf("unexpected value: %v", got)
	}
}

func TestRunnerCallCanceled(t *testing.T) {
	vm := goja.New()
	s := newQueueScheduler(vm)
	defer s.Close()

	r := NewRunner(vm, s, Options{RecoverPanics: true})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := r.Call(ctx, "test.cancel", func(context.Context, *goja.Runtime) (any, error) {
		time.Sleep(100 * time.Millisecond)
		return 1, nil
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrCanceled) {
		t.Fatalf("expected ErrCanceled, got: %v", err)
	}
}

func TestRunnerCallScheduleRejected(t *testing.T) {
	vm := goja.New()
	r := NewRunner(vm, rejectScheduler{}, Options{RecoverPanics: true})
	_, err := r.Call(context.Background(), "test.reject", func(context.Context, *goja.Runtime) (any, error) {
		return nil, nil
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrScheduleRejected) {
		t.Fatalf("expected ErrScheduleRejected, got: %v", err)
	}
}

func TestRunnerCallPanicRecovered(t *testing.T) {
	vm := goja.New()
	s := newQueueScheduler(vm)
	defer s.Close()

	r := NewRunner(vm, s, Options{RecoverPanics: true})
	_, err := r.Call(context.Background(), "test.panic", func(context.Context, *goja.Runtime) (any, error) {
		panic("boom")
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrPanicked) {
		t.Fatalf("expected ErrPanicked, got: %v", err)
	}
}

func TestRunnerShutdown(t *testing.T) {
	vm := goja.New()
	s := newQueueScheduler(vm)
	defer s.Close()

	r := NewRunner(vm, s, Options{RecoverPanics: true})
	if err := r.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown error: %v", err)
	}
	if !r.IsClosed() {
		t.Fatalf("runner should be closed")
	}
	_, err := r.Call(context.Background(), "test.closed", func(context.Context, *goja.Runtime) (any, error) {
		return nil, nil
	})
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed, got: %v", err)
	}
}

func TestRunnerPost(t *testing.T) {
	vm := goja.New()
	s := newQueueScheduler(vm)
	defer s.Close()

	r := NewRunner(vm, s, Options{RecoverPanics: true})
	done := make(chan struct{}, 1)
	err := r.Post(context.Background(), "test.post", func(context.Context, *goja.Runtime) {
		done <- struct{}{}
	})
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatalf("post did not execute")
	}
}

func TestRunnerCallSkipsCanceledQueuedInvocation(t *testing.T) {
	vm := goja.New()
	s := newManualScheduler(vm)
	defer s.Close()

	r := NewRunner(vm, s, Options{RecoverPanics: true})

	var invoked atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := r.Call(ctx, "test.call.canceled-queued", func(context.Context, *goja.Runtime) (any, error) {
		invoked.Add(1)
		return "unexpected", nil
	})
	if err == nil {
		t.Fatalf("expected cancellation error")
	}
	if !errors.Is(err, ErrCanceled) {
		t.Fatalf("expected ErrCanceled, got: %v", err)
	}

	if !s.RunNext() {
		t.Fatalf("expected queued job")
	}
	if got := invoked.Load(); got != 0 {
		t.Fatalf("expected canceled queued call to skip invocation, invoked=%d", got)
	}
}

func TestRunnerPostKeepsTimeoutContextAliveUntilQueuedExecution(t *testing.T) {
	vm := goja.New()
	s := newManualScheduler(vm)
	defer s.Close()

	r := NewRunner(vm, s, Options{
		RecoverPanics: true,
		MaxWait:       1000,
	})

	done := make(chan error, 1)
	err := r.Post(context.Background(), "test.post.context-lifecycle", func(ctx context.Context, _ *goja.Runtime) {
		done <- ctx.Err()
	})
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}

	if !s.RunNext() {
		t.Fatalf("expected queued post job")
	}

	select {
	case ctxErr := <-done:
		if ctxErr != nil {
			t.Fatalf("expected queued post callback context to be active, got: %v", ctxErr)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("post callback did not execute")
	}
}

func TestRunnerCallWithLeakedOwnerContextStillSchedules(t *testing.T) {
	vm := goja.New()
	s := newQueueScheduler(vm)
	defer s.Close()

	r := NewRunner(vm, s, Options{RecoverPanics: true})

	innerReturned := make(chan struct{})
	innerInvoked := make(chan struct{}, 1)
	earlyReturn := make(chan bool, 1)

	_, err := r.Call(context.Background(), "test.call.leaked-owner-ctx", func(ctx context.Context, _ *goja.Runtime) (any, error) {
		go func(leaked context.Context) {
			_, _ = r.Call(leaked, "test.call.inner", func(context.Context, *goja.Runtime) (any, error) {
				innerInvoked <- struct{}{}
				return nil, nil
			})
			close(innerReturned)
		}(ctx)

		time.Sleep(20 * time.Millisecond)
		select {
		case <-innerReturned:
			earlyReturn <- true
		default:
			earlyReturn <- false
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("outer call failed: %v", err)
	}

	select {
	case early := <-earlyReturn:
		if early {
			t.Fatalf("inner call returned before owner callback finished; leaked owner context bypassed scheduler")
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for early-return check")
	}

	select {
	case <-innerInvoked:
	case <-time.After(1 * time.Second):
		t.Fatalf("inner invocation was never scheduled")
	}
}

func TestRunnerPostWithLeakedOwnerContextStillSchedules(t *testing.T) {
	vm := goja.New()
	s := newQueueScheduler(vm)
	defer s.Close()

	r := NewRunner(vm, s, Options{RecoverPanics: true})

	postDone := make(chan struct{})
	earlyPost := make(chan bool, 1)

	_, err := r.Call(context.Background(), "test.post.leaked-owner-ctx", func(ctx context.Context, _ *goja.Runtime) (any, error) {
		go func(leaked context.Context) {
			_ = r.Post(leaked, "test.post.inner", func(context.Context, *goja.Runtime) {
				close(postDone)
			})
		}(ctx)

		time.Sleep(20 * time.Millisecond)
		select {
		case <-postDone:
			earlyPost <- true
		default:
			earlyPost <- false
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("outer call failed: %v", err)
	}

	select {
	case early := <-earlyPost:
		if early {
			t.Fatalf("post callback executed before owner callback finished; leaked owner context bypassed scheduler")
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for post early-return check")
	}

	select {
	case <-postDone:
	case <-time.After(1 * time.Second):
		t.Fatalf("post callback was never scheduled")
	}
}
