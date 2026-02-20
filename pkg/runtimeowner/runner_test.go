package runtimeowner

import (
	"context"
	"errors"
	"sync"
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
