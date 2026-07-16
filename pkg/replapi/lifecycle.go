package replapi

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/go-go-golems/go-go-goja/pkg/replsession"
)

// AppPhase is the lifecycle phase of a replapi application.
type AppPhase string

const (
	AppPhaseOpen    AppPhase = "open"
	AppPhaseClosing AppPhase = "closing"
	AppPhaseClosed  AppPhase = "closed"
)

var (
	// ErrAppClosing is returned after application shutdown begins.
	ErrAppClosing = errors.New("replapi: app is closing")
	// ErrAppClosed is returned after application shutdown completes.
	ErrAppClosed = errors.New("replapi: app is closed")
)

// AppLifecycleError identifies the application phase that rejected an operation.
type AppLifecycleError struct {
	Phase AppPhase
}

func (e *AppLifecycleError) Error() string {
	return fmt.Sprintf("replapi: app is %s", e.Phase)
}

func (e *AppLifecycleError) Unwrap() error {
	if e != nil && e.Phase == AppPhaseClosed {
		return ErrAppClosed
	}
	return ErrAppClosing
}

type appLifecycle struct {
	mu           sync.Mutex
	phase        AppPhase
	closeAttempt chan struct{}
	closeErr     error
}

func (a *App) ensureOpen() error {
	if a == nil {
		return errors.New("replapi: app is nil")
	}
	a.lifecycle.mu.Lock()
	phase := a.lifecycle.phase
	a.lifecycle.mu.Unlock()
	if phase == AppPhaseOpen {
		return nil
	}
	return appPhaseError(phase)
}

func appPhaseError(phase AppPhase) error {
	if phase == AppPhaseClosed {
		return &AppLifecycleError{Phase: AppPhaseClosed}
	}
	return &AppLifecycleError{Phase: AppPhaseClosing}
}

func (a *App) translateLifecycleError(err error) error {
	if err == nil || a == nil {
		return err
	}
	if !errors.Is(err, replsession.ErrServiceClosing) &&
		!errors.Is(err, replsession.ErrServiceClosed) &&
		!errors.Is(err, replsession.ErrSessionClosing) &&
		!errors.Is(err, replsession.ErrSessionClosed) {
		return err
	}
	a.lifecycle.mu.Lock()
	phase := a.lifecycle.phase
	a.lifecycle.mu.Unlock()
	if phase != AppPhaseOpen {
		return appPhaseError(phase)
	}
	return err
}

// UnloadSession removes and closes a live runtime without soft-deleting durable
// history. For an in-memory profile this intentionally discards the only state.
func (a *App) UnloadSession(ctx context.Context, sessionID string) error {
	if err := a.ensureOpen(); err != nil {
		return err
	}
	return a.translateLifecycleError(a.service.UnloadSession(ctx, sessionID))
}

// Close non-destructively unloads all live runtimes. It does not close the
// configured store; hosts must close the app before closing the store.
//
// Concurrent callers share one active close attempt. If shutdown times out
// before an active session operation releases its gate, a later Close call can
// retry the reachable closing state.
func (a *App) Close(ctx context.Context) error {
	if a == nil {
		return nil
	}
	ctx = nonNilAppContext(ctx)
	for {
		a.lifecycle.mu.Lock()
		if a.lifecycle.phase == AppPhaseClosed {
			err := a.lifecycle.closeErr
			a.lifecycle.mu.Unlock()
			return err
		}
		if attempt := a.lifecycle.closeAttempt; attempt != nil {
			a.lifecycle.mu.Unlock()
			select {
			case <-attempt:
				continue
			case <-ctx.Done():
				return appContextError(ctx)
			}
		}

		a.lifecycle.phase = AppPhaseClosing
		a.cancel(ErrAppClosing)
		attempt := make(chan struct{})
		a.lifecycle.closeAttempt = attempt
		a.lifecycle.mu.Unlock()

		err := a.service.Close(ctx)
		terminal := a.service.Closed()

		a.lifecycle.mu.Lock()
		a.lifecycle.closeAttempt = nil
		if terminal {
			a.lifecycle.phase = AppPhaseClosed
			a.lifecycle.closeErr = err
		}
		close(attempt)
		finalErr := a.lifecycle.closeErr
		a.lifecycle.mu.Unlock()

		if terminal {
			return finalErr
		}
		return err
	}
}

func nonNilAppContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func appContextError(ctx context.Context) error {
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}
	return ctx.Err()
}
