package replsession

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-go-golems/go-go-goja/pkg/repldb"
)

// SessionHealth describes whether a live VM may evaluate new JavaScript.
type SessionHealth string

const (
	SessionHealthHealthy  SessionHealth = "healthy"
	SessionHealthDegraded SessionHealth = "degraded"
	SessionHealthFenced   SessionHealth = "fenced"
)

var (
	// ErrSessionDegraded identifies a VM that executed work which was not durably committed.
	ErrSessionDegraded = errors.New("replsession: persistence is degraded")
	// ErrSessionFenced identifies a VM whose durable ownership was lost.
	ErrSessionFenced = errors.New("replsession: session ownership was lost")
	// ErrCommitFailed identifies an evaluation that executed but did not durably commit.
	ErrCommitFailed = errors.New("replsession: evaluation commit failed")
	// ErrNoPendingCommit indicates that exact retry is unavailable.
	ErrNoPendingCommit = errors.New("replsession: no pending evaluation commit")
)

// SessionHealthError rejects evaluation before JavaScript runs. SessionID and
// Health are safe for callers; Cause is retained for local diagnostics only.
type SessionHealthError struct {
	SessionID string
	Health    SessionHealth
	Cause     error
}

func (e *SessionHealthError) Error() string {
	if e == nil {
		return "replsession: unhealthy session"
	}
	return fmt.Sprintf("replsession: session %q is %s", e.SessionID, e.Health)
}

func (e *SessionHealthError) Unwrap() error {
	if e != nil && e.Health == SessionHealthFenced {
		return ErrSessionFenced
	}
	return ErrSessionDegraded
}

// CommitError reports that a cell result exists but its exact durable record
// was not committed. Evaluate returns the Cell alongside this error; callers
// must not rerun source. Use RetryPendingCommit or discard and recover the VM.
type CommitError struct {
	SessionID string
	CellID    int
	Cause     error
}

func (e *CommitError) Error() string {
	if e == nil {
		return ErrCommitFailed.Error()
	}
	return fmt.Sprintf("%s: session=%q cell=%d: %v", ErrCommitFailed, e.SessionID, e.CellID, e.Cause)
}

func (e *CommitError) Unwrap() error { return ErrCommitFailed }

// PersistenceCause returns the underlying local persistence failure.
func (e *CommitError) PersistenceCause() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type pendingCommit struct {
	record repldb.EvaluationRecord
	cell   *cellState
	fence  *repldb.WriteFence
}

func (s *sessionState) evaluationHealthError() error {
	switch s.health {
	case SessionHealthDegraded:
		return &SessionHealthError{SessionID: s.id, Health: s.health, Cause: s.healthCause}
	case SessionHealthFenced:
		return &SessionHealthError{SessionID: s.id, Health: s.health, Cause: s.healthCause}
	case SessionHealthHealthy:
		return nil
	default:
		return nil
	}
}

func (s *sessionState) markCommitFailure(cell *cellState, record *repldb.EvaluationRecord, fence *repldb.WriteFence, cause error) *CommitError {
	if isOwnershipError(cause) {
		s.health = SessionHealthFenced
	} else {
		s.health = SessionHealthDegraded
	}
	s.healthCause = cause
	if record != nil {
		s.pendingCommit = &pendingCommit{record: *record, cell: cell, fence: fence}
	} else {
		s.pendingCommit = nil
	}
	cellID := 0
	if cell != nil && cell.report != nil {
		cellID = cell.report.ID
	}
	return &CommitError{SessionID: s.id, CellID: cellID, Cause: cause}
}

func (s *sessionState) markFenced(cause error) {
	s.health = SessionHealthFenced
	s.healthCause = cause
}

func (s *sessionState) publishCommittedCell(cell *cellState) {
	if cell == nil || cell.report == nil {
		return
	}
	s.nextCellID = cell.report.ID
	s.cells = append(s.cells, cell)
	s.pendingCommit = nil
	s.health = SessionHealthHealthy
	s.healthCause = nil
}

// SessionHealth returns the current live-session health without requiring the
// session to be healthy.
func (s *Service) SessionHealth(ctx context.Context, sessionID string) (SessionHealth, error) {
	state, err := s.getSession(sessionID)
	if err != nil {
		return "", err
	}
	op, err := state.beginOperation(ctx)
	if err != nil {
		return "", err
	}
	defer op.Release()
	return state.health, nil
}

// RetryPendingCommit retries the exact retained EvaluationRecord without
// executing JavaScript or rebuilding persistence payloads.
func (s *Service) RetryPendingCommit(ctx context.Context, sessionID string) (*EvaluateResponse, error) {
	state, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}
	op, err := state.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer op.Release()
	ctx = op.Context()

	if state.health == SessionHealthFenced {
		return nil, state.evaluationHealthError()
	}
	pending := state.pendingCommit
	if state.health != SessionHealthDegraded || pending == nil || pending.cell == nil {
		return nil, ErrNoPendingCommit
	}

	response := &EvaluateResponse{Session: state.buildSummary(ctx), Cell: pending.cell.report}
	if _, err := s.renewSessionLease(ctx, state); err != nil {
		return nil, err
	}
	var persistErr error
	if pending.fence != nil && s.leaseStore != nil {
		persistErr = s.leaseStore.PersistEvaluationFenced(ctx, pending.record, *pending.fence, s.nowUTC())
	} else {
		persistErr = s.store.PersistEvaluation(ctx, pending.record)
	}
	if persistErr != nil {
		state.healthCause = persistErr
		if isOwnershipError(persistErr) {
			state.markFenced(persistErr)
			return nil, state.evaluationHealthError()
		}
		return response, &CommitError{SessionID: state.id, CellID: pending.record.CellID, Cause: persistErr}
	}
	state.publishCommittedCell(pending.cell)
	response.Session = state.buildSummary(ctx)
	return response, nil
}
