package replapi

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/rs/zerolog"
)

func TestAppUnloadPreservesDurableHistoryAndDeleteRemovesIt(t *testing.T) {
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	app, err := New(
		context.Background(),
		newTestFactory(t),
		zerolog.Nop(),
		WithProfile(ProfilePersistent),
		WithStore(store),
	)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer func() { _ = app.Close(context.Background()) }()

	session, err := app.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := app.Evaluate(context.Background(), session.ID, `const durable = 42; durable`); err != nil {
		t.Fatalf("evaluate session: %v", err)
	}
	if err := app.UnloadSession(context.Background(), session.ID); err != nil {
		t.Fatalf("unload session: %v", err)
	}

	restored, err := app.Snapshot(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("auto-restore unloaded session: %v", err)
	}
	if restored.CellCount != 1 {
		t.Fatalf("expected durable history to survive unload, got %d cells", restored.CellCount)
	}
	if err := app.DeleteSession(context.Background(), session.ID); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	records, err := app.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected delete to hide durable session, got %d records", len(records))
	}
}

func TestAppCloseTimeoutIsRetryableAndLifecycleErrorsAreTyped(t *testing.T) {
	app, err := New(context.Background(), newTestFactory(t), zerolog.Nop(), WithProfile(ProfileInteractive))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	session, err := app.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	var closeCalls atomic.Int32
	holderDone := make(chan error, 1)
	go func() {
		holderDone <- app.WithRuntime(context.Background(), session.ID, func(_ context.Context, runtime *engine.Runtime) error {
			if err := runtime.AddCloser(func(context.Context) error {
				closeCalls.Add(1)
				return nil
			}); err != nil {
				return err
			}
			close(entered)
			<-release // Deliberately ignore op cancellation for the timeout path.
			return nil
		})
	}()
	<-entered

	closeCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := app.Close(closeCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected close deadline, got %v", err)
	}
	_, err = app.CreateSession(context.Background())
	if !errors.Is(err, ErrAppClosing) {
		t.Fatalf("expected ErrAppClosing after incomplete close, got %v", err)
	}
	var lifecycleErr *AppLifecycleError
	if !errors.As(err, &lifecycleErr) || lifecycleErr.Phase != AppPhaseClosing {
		t.Fatalf("expected typed closing error, got %#v", err)
	}

	close(release)
	if err := <-holderDone; err != nil {
		t.Fatalf("holder callback: %v", err)
	}
	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("retry app close: %v", err)
	}
	if got := closeCalls.Load(); got != 1 {
		t.Fatalf("expected close hook exactly once, got %d", got)
	}
	_, err = app.CreateSession(context.Background())
	if !errors.Is(err, ErrAppClosed) {
		t.Fatalf("expected ErrAppClosed after close, got %v", err)
	}
	if !errors.As(err, &lifecycleErr) || lifecycleErr.Phase != AppPhaseClosed {
		t.Fatalf("expected typed closed error, got %#v", err)
	}
	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("repeated close: %v", err)
	}
}

func TestAppCloseIsConcurrentAndReturnsStableAggregate(t *testing.T) {
	app, err := New(context.Background(), newTestFactory(t), zerolog.Nop(), WithProfile(ProfileInteractive))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	session, err := app.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	closerErr := errors.New("app-close-hook")
	var closeCalls atomic.Int32
	if err := app.WithRuntime(context.Background(), session.ID, func(_ context.Context, runtime *engine.Runtime) error {
		return runtime.AddCloser(func(context.Context) error {
			closeCalls.Add(1)
			return closerErr
		})
	}); err != nil {
		t.Fatalf("register closer: %v", err)
	}

	const callers = 8
	errs := make(chan error, callers)
	for range callers {
		go func() { errs <- app.Close(context.Background()) }()
	}
	for range callers {
		if err := <-errs; !errors.Is(err, closerErr) {
			t.Fatalf("expected shared close error, got %v", err)
		}
	}
	if got := closeCalls.Load(); got != 1 {
		t.Fatalf("expected app close hook once, got %d", got)
	}
	if err := app.Close(context.Background()); !errors.Is(err, closerErr) {
		t.Fatalf("expected repeated close error, got %v", err)
	}
}

func TestAppParentContextOwnsRuntimeLifetime(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	app, err := New(parent, newTestFactory(t), zerolog.Nop(), WithProfile(ProfileInteractive))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	session, err := app.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	var runtimeCtx context.Context
	if err := app.WithRuntime(context.Background(), session.ID, func(_ context.Context, runtime *engine.Runtime) error {
		runtimeCtx = runtime.Context()
		return nil
	}); err != nil {
		t.Fatalf("inspect runtime: %v", err)
	}
	cancelParent()
	select {
	case <-runtimeCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("runtime lifetime did not follow app parent")
	}
	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("close canceled-parent app: %v", err)
	}
}

// Promoted from the P0 red suite by Phase 2.
func TestHardeningCanceledWaiterDoesNotExecuteLate(t *testing.T) {
	ctx := context.Background()
	app, err := New(context.Background(), newTestFactory(t), zerolog.Nop(), WithProfile(ProfileInteractive))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer func() { _ = app.Close(context.Background()) }()
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	holderDone := make(chan error, 1)
	go func() {
		holderDone <- app.WithRuntime(ctx, session.ID, func(_ context.Context, _ *engine.Runtime) error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered

	waitCtx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	waiterDone := make(chan error, 1)
	go func() {
		_, snapshotErr := app.Snapshot(waitCtx, session.ID)
		waiterDone <- snapshotErr
	}()

	select {
	case snapshotErr := <-waiterDone:
		close(release)
		if holderErr := <-holderDone; holderErr != nil {
			t.Fatalf("holder operation: %v", holderErr)
		}
		if !errors.Is(snapshotErr, context.DeadlineExceeded) {
			t.Fatalf("expected queued snapshot deadline, got %v", snapshotErr)
		}
	case <-time.After(100 * time.Millisecond):
		close(release)
		<-holderDone
		t.Fatal("queued snapshot ignored cancellation")
	}
}

// Promoted from the P0 red suite by Phase 2.
func TestHardeningDeleteWaitsForActiveSessionOperation(t *testing.T) {
	app, err := New(context.Background(), newTestFactory(t), zerolog.Nop(), WithProfile(ProfileInteractive))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	session, err := app.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	holderDone := make(chan error, 1)
	go func() {
		holderDone <- app.WithRuntime(context.Background(), session.ID, func(_ context.Context, _ *engine.Runtime) error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered

	deleteDone := make(chan error, 1)
	go func() { deleteDone <- app.DeleteSession(context.Background(), session.ID) }()
	select {
	case err := <-deleteDone:
		t.Fatalf("delete completed while operation was active: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	if err := <-holderDone; err != nil {
		t.Fatalf("holder operation: %v", err)
	}
	if err := <-deleteDone; err != nil {
		t.Fatalf("delete session: %v", err)
	}
}
