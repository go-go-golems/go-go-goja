package replsession

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/rs/zerolog"
)

func TestServiceRuntimeLifetimeIsIndependentOfStartupContext(t *testing.T) {
	startupCtx, cancelStartup := context.WithCancel(context.Background())
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop())
	session, err := service.CreateSession(startupCtx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	var lifetime context.Context
	var closeCalls atomic.Int32
	if err := service.WithRuntime(context.Background(), session.ID, func(opCtx context.Context, runtime *engine.Runtime) error {
		if err := opCtx.Err(); err != nil {
			return err
		}
		lifetime = runtime.Context()
		return runtime.AddCloser(func(context.Context) error {
			closeCalls.Add(1)
			return nil
		})
	}); err != nil {
		t.Fatalf("inspect runtime: %v", err)
	}

	cancelStartup()
	if err := lifetime.Err(); err != nil {
		t.Fatalf("runtime lifetime followed startup context: %v", err)
	}
	if err := service.UnloadSession(context.Background(), session.ID); err != nil {
		t.Fatalf("unload session: %v", err)
	}
	if !errors.Is(lifetime.Err(), context.Canceled) {
		t.Fatalf("expected unload to cancel runtime lifetime, got %v", lifetime.Err())
	}
	if got := closeCalls.Load(); got != 1 {
		t.Fatalf("expected one close hook call, got %d", got)
	}
	if _, err := service.Snapshot(context.Background(), session.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected unloaded session to leave live map, got %v", err)
	}
}

func TestServiceUnloadTimeoutLeavesRetryableClosingSession(t *testing.T) {
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop())
	session, err := service.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	holderDone := make(chan error, 1)
	go func() {
		holderDone <- service.WithRuntime(context.Background(), session.ID, func(_ context.Context, _ *engine.Runtime) error {
			close(entered)
			<-release // Deliberately ignore operation cancellation to exercise retry.
			return nil
		})
	}()
	<-entered

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := service.UnloadSession(shutdownCtx, session.ID); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected bounded unload timeout, got %v", err)
	}

	_, err = service.Snapshot(context.Background(), session.ID)
	if !errors.Is(err, ErrSessionClosing) {
		t.Fatalf("expected reachable closing session, got %v", err)
	}
	var lifecycleErr *SessionLifecycleError
	if !errors.As(err, &lifecycleErr) || lifecycleErr.SessionID != session.ID {
		t.Fatalf("expected typed lifecycle error for %q, got %#v", session.ID, err)
	}

	close(release)
	if err := <-holderDone; err != nil {
		t.Fatalf("holder callback: %v", err)
	}
	if err := service.UnloadSession(context.Background(), session.ID); err != nil {
		t.Fatalf("retry unload: %v", err)
	}
	if _, err := service.Snapshot(context.Background(), session.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected retry to remove session, got %v", err)
	}
}

func TestServiceCloseIsConcurrentIdempotentAndAggregatesErrors(t *testing.T) {
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop())
	sessionA, err := service.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create session A: %v", err)
	}
	sessionB, err := service.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create session B: %v", err)
	}

	var closeA atomic.Int32
	var closeB atomic.Int32
	register := func(sessionID string, calls *atomic.Int32, message string) {
		t.Helper()
		if err := service.WithRuntime(context.Background(), sessionID, func(_ context.Context, runtime *engine.Runtime) error {
			return runtime.AddCloser(func(context.Context) error {
				calls.Add(1)
				return errors.New(message)
			})
		}); err != nil {
			t.Fatalf("register closer for %s: %v", sessionID, err)
		}
	}
	register(sessionA.ID, &closeA, "close-a")
	register(sessionB.ID, &closeB, "close-b")

	const callers = 8
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- service.Close(context.Background())
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err == nil || !strings.Contains(err.Error(), "close-a") || !strings.Contains(err.Error(), "close-b") {
			t.Fatalf("expected stable aggregate from concurrent close, got %v", err)
		}
	}
	if closeA.Load() != 1 || closeB.Load() != 1 {
		t.Fatalf("expected close hooks exactly once, got A=%d B=%d", closeA.Load(), closeB.Load())
	}
	if err := service.Close(context.Background()); err == nil || !strings.Contains(err.Error(), "close-a") || !strings.Contains(err.Error(), "close-b") {
		t.Fatalf("expected repeated close to return aggregate, got %v", err)
	}
	if _, err := service.CreateSession(context.Background()); !errors.Is(err, ErrServiceClosed) {
		t.Fatalf("expected create after close to return ErrServiceClosed, got %v", err)
	}
}

func TestServiceCloseCancelsWithRuntimeOperationContext(t *testing.T) {
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop())
	session, err := service.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	entered := make(chan struct{})
	holderDone := make(chan error, 1)
	go func() {
		holderDone <- service.WithRuntime(context.Background(), session.ID, func(opCtx context.Context, _ *engine.Runtime) error {
			close(entered)
			<-opCtx.Done()
			return context.Cause(opCtx)
		})
	}()
	<-entered

	if err := service.Close(context.Background()); err != nil {
		t.Fatalf("close service: %v", err)
	}
	if err := <-holderDone; !errors.Is(err, ErrServiceClosing) {
		t.Fatalf("expected callback operation context to report service closing, got %v", err)
	}
}

func TestServiceCloseInterruptsActiveEvaluation(t *testing.T) {
	started := make(chan struct{})
	initializer := &lifecycleTestInitializer{started: started}
	factory, err := engine.NewRuntimeFactoryBuilder().
		UseModuleMiddleware(engine.MiddlewareSafe()).
		WithRuntimeInitializers(initializer).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	service := NewService(factory, zerolog.Nop())
	session, err := service.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	evalDone := make(chan error, 1)
	go func() {
		_, err := service.Evaluate(context.Background(), session.ID, `phase2Started(); while (true) {}`)
		evalDone <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("evaluation did not enter JavaScript")
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := service.Close(closeCtx); err != nil {
		t.Fatalf("close service: %v", err)
	}
	select {
	case err := <-evalDone:
		if err == nil {
			t.Fatal("expected active evaluation to be canceled by close")
		}
	case <-time.After(time.Second):
		t.Fatal("active evaluation did not stop during close")
	}
}

type lifecycleTestInitializer struct {
	started chan struct{}
	once    sync.Once
}

func (i *lifecycleTestInitializer) ID() string { return "replsession-lifecycle-test" }

func (i *lifecycleTestInitializer) InitRuntime(ctx *engine.RuntimeInitializationContext) error {
	return ctx.VM.Set("phase2Started", func() {
		i.once.Do(func() { close(i.started) })
	})
}
