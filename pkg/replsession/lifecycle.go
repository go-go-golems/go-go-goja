package replsession

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// SessionPhase is the lifecycle phase of one live session runtime.
type SessionPhase string

const (
	SessionPhaseActive  SessionPhase = "active"
	SessionPhaseClosing SessionPhase = "closing"
	SessionPhaseClosed  SessionPhase = "closed"
)

// ServicePhase is the lifecycle phase of the session service.
type ServicePhase string

const (
	ServicePhaseOpen    ServicePhase = "open"
	ServicePhaseClosing ServicePhase = "closing"
	ServicePhaseClosed  ServicePhase = "closed"
)

var (
	// ErrSessionClosing is returned when a session has begun shutdown.
	ErrSessionClosing = errors.New("replsession: session is closing")
	// ErrSessionClosed is returned when an operation retained a session that has closed.
	ErrSessionClosed = errors.New("replsession: session is closed")
	// ErrServiceClosing is returned when service shutdown has begun.
	ErrServiceClosing = errors.New("replsession: service is closing")
	// ErrServiceClosed is returned after service shutdown completes.
	ErrServiceClosed = errors.New("replsession: service is closed")
)

// SessionLifecycleError identifies a rejected operation and the session phase
// that rejected it. Callers should use errors.Is with ErrSessionClosing or
// ErrSessionClosed and errors.As when they need the session ID.
type SessionLifecycleError struct {
	SessionID string
	Phase     SessionPhase
}

func (e *SessionLifecycleError) Error() string {
	return fmt.Sprintf("replsession: session %q is %s", e.SessionID, e.Phase)
}

func (e *SessionLifecycleError) Unwrap() error {
	if e != nil && e.Phase == SessionPhaseClosed {
		return ErrSessionClosed
	}
	return ErrSessionClosing
}

// ServiceLifecycleError identifies a rejected operation and the service phase
// that rejected it.
type ServiceLifecycleError struct {
	Phase ServicePhase
}

func (e *ServiceLifecycleError) Error() string {
	return fmt.Sprintf("replsession: service is %s", e.Phase)
}

func (e *ServiceLifecycleError) Unwrap() error {
	if e != nil && e.Phase == ServicePhaseClosed {
		return ErrServiceClosed
	}
	return ErrServiceClosing
}

type operationToken struct {
	ctx     context.Context
	release func()
}

func (t *operationToken) Context() context.Context {
	if t == nil || t.ctx == nil {
		return context.Background()
	}
	return t.ctx
}

func (t *operationToken) Release() {
	if t != nil && t.release != nil {
		t.release()
		t.release = nil
	}
}

func newCapacityOneGate() chan struct{} {
	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	return gate
}

func acquireGate(ctx context.Context, gate chan struct{}) error {
	ctx = nonNilContext(ctx)
	select {
	case <-gate:
		return nil
	case <-ctx.Done():
		return contextError(ctx)
	}
}

func releaseGate(gate chan struct{}) {
	gate <- struct{}{}
}

// mergeOperationContext cancels when either the caller or session lifetime is
// canceled. context.AfterFunc avoids one waiting goroutine per queued operation.
func mergeOperationContext(caller context.Context, lifetime context.Context) (context.Context, func()) {
	caller = nonNilContext(caller)
	lifetime = nonNilContext(lifetime)
	ctx, cancel := context.WithCancelCause(caller)
	stop := context.AfterFunc(lifetime, func() {
		cancel(contextError(lifetime))
	})
	if err := contextError(lifetime); err != nil {
		cancel(err)
	}
	return ctx, func() {
		stop()
		cancel(context.Canceled)
	}
}

func (s *sessionState) beginOperation(ctx context.Context) (*operationToken, error) {
	opCtx, cleanup := mergeOperationContext(ctx, s.ctx)
	if err := acquireGate(opCtx, s.gate); err != nil {
		cleanup()
		return nil, err
	}

	s.lifecycleMu.Lock()
	phase := s.phase
	s.lifecycleMu.Unlock()
	if phase != SessionPhaseActive {
		releaseGate(s.gate)
		cleanup()
		return nil, sessionPhaseError(s.id, phase)
	}

	return &operationToken{
		ctx: opCtx,
		release: func() {
			releaseGate(s.gate)
			cleanup()
		},
	}, nil
}

func sessionPhaseError(sessionID string, phase SessionPhase) error {
	if phase == SessionPhaseClosed {
		return &SessionLifecycleError{SessionID: sessionID, Phase: SessionPhaseClosed}
	}
	return &SessionLifecycleError{SessionID: sessionID, Phase: SessionPhaseClosing}
}

func servicePhaseError(phase ServicePhase) error {
	if phase == ServicePhaseClosed {
		return &ServiceLifecycleError{Phase: ServicePhaseClosed}
	}
	return &ServiceLifecycleError{Phase: ServicePhaseClosing}
}

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}
	return ctx.Err()
}

type stopDisposition uint8

const (
	stopUnload stopDisposition = iota
	stopDelete
)

// UnloadSession removes and closes one live runtime without deleting durable
// history. Unloading an in-memory session intentionally discards its only state.
func (s *Service) UnloadSession(ctx context.Context, sessionID string) error {
	state, err := s.getSession(sessionID)
	if err != nil {
		return err
	}
	_, err = s.stopSessionState(ctx, state, stopUnload)
	return err
}

// DeleteSession closes one live runtime and soft-deletes durable history when
// persistence is enabled.
func (s *Service) DeleteSession(ctx context.Context, sessionID string) error {
	state, err := s.getSession(sessionID)
	if err != nil {
		return err
	}
	_, err = s.stopSessionState(ctx, state, stopDelete)
	return err
}

// stopSessionState returns terminal=false only when shutdown did not acquire the
// operation gate. In that case the closing state remains in the service map so
// a later unload/delete/close call can retry safely.
func (s *Service) stopSessionState(ctx context.Context, state *sessionState, disposition stopDisposition) (bool, error) {
	ctx = nonNilContext(ctx)
	if err := acquireGate(ctx, state.stopGate); err != nil {
		return false, err
	}
	defer releaseGate(state.stopGate)

	state.lifecycleMu.Lock()
	switch state.phase {
	case SessionPhaseClosed:
		err := state.closeErr
		state.lifecycleMu.Unlock()
		return true, err
	case SessionPhaseActive:
		state.phase = SessionPhaseClosing
		state.cancel(sessionPhaseError(state.id, SessionPhaseClosing))
	case SessionPhaseClosing:
		// A previous bounded shutdown attempt may have timed out. Retry below.
	default:
		state.phase = SessionPhaseClosing
		state.cancel(sessionPhaseError(state.id, SessionPhaseClosing))
	}
	state.lifecycleMu.Unlock()

	if err := acquireGate(ctx, state.gate); err != nil {
		return false, err
	}
	defer releaseGate(state.gate)

	var persistErr error
	if disposition == stopDelete && state.policy.Persist.Enabled {
		deletedAt := s.nowUTC()
		if s.ownershipEnabled(state.policy) {
			renewed, err := s.renewSessionLease(ctx, state)
			if err == nil {
				err = s.leaseStore.DeleteSessionFenced(ctx, state.id, renewed, deletedAt, deletedAt)
			}
			if err != nil {
				if isOwnershipError(err) {
					state.markFenced(err)
					err = state.evaluationHealthError()
				}
				persistErr = fmt.Errorf("persist fenced session deletion: %w", err)
			}
		} else if err := s.store.DeleteSession(ctx, state.id, deletedAt); err != nil {
			persistErr = fmt.Errorf("persist session deletion: %w", err)
		}
	}
	leaseErr := s.releaseSessionLease(ctx, state)
	closeErr := state.runtime.Close(ctx)
	closeStateErr := errors.Join(persistErr, leaseErr, closeErr)

	state.lifecycleMu.Lock()
	state.phase = SessionPhaseClosed
	state.closeErr = closeStateErr
	state.lifecycleMu.Unlock()

	s.mu.Lock()
	if s.sessions[state.id] == state {
		delete(s.sessions, state.id)
	}
	s.mu.Unlock()
	return true, closeStateErr
}

// Close non-destructively unloads every live runtime. Calls are idempotent;
// concurrent callers share an active attempt. If an attempt times out before a
// session gate is acquired, the service remains closing and a later call retries.
func (s *Service) Close(ctx context.Context) error {
	ctx = nonNilContext(ctx)
	for {
		s.mu.Lock()
		if s.phase == ServicePhaseClosed {
			err := s.closeErr
			s.mu.Unlock()
			return err
		}
		if attempt := s.closeAttempt; attempt != nil {
			s.mu.Unlock()
			select {
			case <-attempt:
				continue
			case <-ctx.Done():
				return contextError(ctx)
			}
		}

		s.phase = ServicePhaseClosing
		s.lifetimeCancel(ErrServiceClosing)
		attempt := make(chan struct{})
		s.closeAttempt = attempt
		states := make([]*sessionState, 0, len(s.sessions))
		for _, state := range s.sessions {
			states = append(states, state)
		}
		s.mu.Unlock()

		type stopResult struct {
			terminal bool
			err      error
		}
		results := make(chan stopResult, len(states))
		var wg sync.WaitGroup
		for _, state := range states {
			wg.Add(1)
			go func(state *sessionState) {
				defer wg.Done()
				terminal, err := s.stopSessionState(ctx, state, stopUnload)
				results <- stopResult{terminal: terminal, err: err}
			}(state)
		}
		wg.Wait()
		close(results)

		var terminalErr error
		var incompleteErr error
		incomplete := false
		for result := range results {
			if result.terminal {
				terminalErr = errors.Join(terminalErr, result.err)
				continue
			}
			incomplete = true
			incompleteErr = errors.Join(incompleteErr, result.err)
		}

		s.mu.Lock()
		s.closeAccum = errors.Join(s.closeAccum, terminalErr)
		s.closeAttempt = nil
		if !incomplete && len(s.sessions) == 0 {
			s.phase = ServicePhaseClosed
			s.closeErr = s.closeAccum
		}
		close(attempt)
		terminal := s.phase == ServicePhaseClosed
		finalErr := s.closeErr
		s.mu.Unlock()

		if terminal {
			return finalErr
		}
		return errors.Join(terminalErr, incompleteErr)
	}
}

// Closed reports whether service shutdown reached its terminal state.
func (s *Service) Closed() bool {
	if s == nil {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.phase == ServicePhaseClosed
}
