package repldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrSessionOwned indicates that another unexpired owner holds the session lease.
	ErrSessionOwned = errors.New("repldb: session has another active owner")
	// ErrLeaseLost indicates that an owner/epoch no longer matches the durable lease.
	ErrLeaseLost = errors.New("repldb: session lease was lost")
	// ErrWriteConflict indicates that durable history is not at the expected cell.
	ErrWriteConflict = errors.New("repldb: durable cell head conflict")
)

// SessionLease is the durable ownership token for one persistent session.
type SessionLease struct {
	SessionID  string
	OwnerID    string
	Epoch      int64
	LeaseUntil time.Time
}

// WriteFence must match the current unexpired lease and next durable cell.
type WriteFence struct {
	OwnerID        string
	Epoch          int64
	ExpectedCellID int
}

// SessionOwnedError reports the conflicting owner and lease expiry.
type SessionOwnedError struct {
	SessionID  string
	OwnerID    string
	LeaseUntil time.Time
}

func (e *SessionOwnedError) Error() string {
	return fmt.Sprintf("%s: session=%q owner=%q lease_until=%s", ErrSessionOwned, e.SessionID, e.OwnerID, e.LeaseUntil.UTC().Format(time.RFC3339Nano))
}
func (e *SessionOwnedError) Unwrap() error { return ErrSessionOwned }

// LeaseLostError reports a stale or expired ownership token.
type LeaseLostError struct {
	SessionID string
	OwnerID   string
	Epoch     int64
	Reason    string
}

func (e *LeaseLostError) Error() string {
	return fmt.Sprintf("%s: session=%q owner=%q epoch=%d reason=%s", ErrLeaseLost, e.SessionID, e.OwnerID, e.Epoch, e.Reason)
}
func (e *LeaseLostError) Unwrap() error { return ErrLeaseLost }

// WriteConflictError reports disagreement about the next durable cell ID.
type WriteConflictError struct {
	SessionID string
	Expected  int
	Actual    int
}

func (e *WriteConflictError) Error() string {
	return fmt.Sprintf("%s: session=%q expected=%d actual=%d", ErrWriteConflict, e.SessionID, e.Expected, e.Actual)
}
func (e *WriteConflictError) Unwrap() error { return ErrWriteConflict }

// AcquireSessionLease atomically acquires, renews, or takes over an expired lease.
func (s *Store) AcquireSessionLease(ctx context.Context, sessionID string, ownerID string, now time.Time, ttl time.Duration) (SessionLease, error) {
	if err := validateLeaseInput(s, sessionID, ownerID, ttl); err != nil {
		return SessionLease{}, err
	}
	now = normalizeLeaseTime(now)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SessionLease{}, fmt.Errorf("acquire session lease: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var active int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE session_id = ? AND deleted_at IS NULL`, sessionID).Scan(&active); err != nil {
		return SessionLease{}, fmt.Errorf("acquire session lease: load session: %w", err)
	}
	if active != 1 {
		return SessionLease{}, ErrSessionNotFound
	}

	lease, found, err := loadSessionLeaseTx(ctx, tx, sessionID)
	if err != nil {
		return SessionLease{}, err
	}
	leaseUntil := now.Add(ttl).UTC()
	switch {
	case !found:
		lease = SessionLease{SessionID: sessionID, OwnerID: ownerID, Epoch: 1, LeaseUntil: leaseUntil}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO session_leases(session_id, owner_id, epoch, lease_until, updated_at)
			VALUES(?, ?, ?, ?, ?)
		`, sessionID, ownerID, lease.Epoch, formatLeaseTime(leaseUntil), formatLeaseTime(now)); err != nil {
			return SessionLease{}, fmt.Errorf("acquire session lease: insert: %w", err)
		}
	case lease.OwnerID == ownerID:
		lease.LeaseUntil = leaseUntil
		if _, err := tx.ExecContext(ctx, `UPDATE session_leases SET lease_until = ?, updated_at = ? WHERE session_id = ? AND owner_id = ? AND epoch = ?`, formatLeaseTime(leaseUntil), formatLeaseTime(now), sessionID, ownerID, lease.Epoch); err != nil {
			return SessionLease{}, fmt.Errorf("acquire session lease: renew same owner: %w", err)
		}
	case !lease.LeaseUntil.After(now):
		lease.OwnerID = ownerID
		lease.Epoch++
		lease.LeaseUntil = leaseUntil
		if _, err := tx.ExecContext(ctx, `UPDATE session_leases SET owner_id = ?, epoch = ?, lease_until = ?, updated_at = ? WHERE session_id = ?`, ownerID, lease.Epoch, formatLeaseTime(leaseUntil), formatLeaseTime(now), sessionID); err != nil {
			return SessionLease{}, fmt.Errorf("acquire session lease: take over expired lease: %w", err)
		}
	default:
		return SessionLease{}, &SessionOwnedError{SessionID: sessionID, OwnerID: lease.OwnerID, LeaseUntil: lease.LeaseUntil}
	}
	if err := tx.Commit(); err != nil {
		return SessionLease{}, fmt.Errorf("acquire session lease: commit: %w", err)
	}
	return lease, nil
}

// RenewSessionLease extends a matching owner/epoch lease. A stale epoch cannot renew.
func (s *Store) RenewSessionLease(ctx context.Context, lease SessionLease, now time.Time, ttl time.Duration) (SessionLease, error) {
	if err := validateLeaseInput(s, lease.SessionID, lease.OwnerID, ttl); err != nil {
		return SessionLease{}, err
	}
	if lease.Epoch <= 0 {
		return SessionLease{}, fmt.Errorf("renew session lease: epoch must be positive")
	}
	now = normalizeLeaseTime(now)
	leaseUntil := now.Add(ttl).UTC()
	result, err := s.db.ExecContext(ctx, `
		UPDATE session_leases
		SET lease_until = ?, updated_at = ?
		WHERE session_id = ? AND owner_id = ? AND epoch = ?
	`, formatLeaseTime(leaseUntil), formatLeaseTime(now), lease.SessionID, lease.OwnerID, lease.Epoch)
	if err != nil {
		return SessionLease{}, fmt.Errorf("renew session lease: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return SessionLease{}, fmt.Errorf("renew session lease: rows affected: %w", err)
	}
	if rows != 1 {
		return SessionLease{}, &LeaseLostError{SessionID: lease.SessionID, OwnerID: lease.OwnerID, Epoch: lease.Epoch, Reason: "owner or epoch no longer matches"}
	}
	lease.LeaseUntil = leaseUntil
	return lease, nil
}

// ReleaseSessionLease expires only an exact owner/epoch token while retaining
// the row so the next different owner receives a monotonically larger epoch.
// Stale release is a safe no-op.
func (s *Store) ReleaseSessionLease(ctx context.Context, lease SessionLease) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("release session lease: store is nil")
	}
	if strings.TrimSpace(lease.SessionID) == "" || strings.TrimSpace(lease.OwnerID) == "" || lease.Epoch <= 0 {
		return fmt.Errorf("release session lease: invalid lease")
	}
	releasedAt := time.Unix(0, 0).UTC()
	if _, err := s.db.ExecContext(ctx, `
		UPDATE session_leases SET lease_until = ?, updated_at = ?
		WHERE session_id = ? AND owner_id = ? AND epoch = ?
	`, formatLeaseTime(releasedAt), formatLeaseTime(releasedAt), lease.SessionID, lease.OwnerID, lease.Epoch); err != nil {
		return fmt.Errorf("release session lease: %w", err)
	}
	return nil
}

// PersistEvaluationFenced verifies ownership, expiry, and expected durable head
// in the same transaction that writes the evaluation and child rows.
func (s *Store) PersistEvaluationFenced(ctx context.Context, record EvaluationRecord, fence WriteFence, now time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("persist evaluation fenced: store is nil")
	}
	if strings.TrimSpace(record.SessionID) == "" || strings.TrimSpace(fence.OwnerID) == "" || fence.Epoch <= 0 || fence.ExpectedCellID <= 0 {
		return fmt.Errorf("persist evaluation fenced: invalid record or fence")
	}
	now = normalizeLeaseTime(now)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist evaluation fenced: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	lease, found, err := loadSessionLeaseTx(ctx, tx, record.SessionID)
	if err != nil {
		return err
	}
	if !found {
		return &LeaseLostError{SessionID: record.SessionID, OwnerID: fence.OwnerID, Epoch: fence.Epoch, Reason: "lease row is absent"}
	}
	if lease.OwnerID != fence.OwnerID || lease.Epoch != fence.Epoch {
		return &LeaseLostError{SessionID: record.SessionID, OwnerID: fence.OwnerID, Epoch: fence.Epoch, Reason: "owner or epoch no longer matches"}
	}
	if !lease.LeaseUntil.After(now) {
		return &LeaseLostError{SessionID: record.SessionID, OwnerID: fence.OwnerID, Epoch: fence.Epoch, Reason: "lease expired"}
	}

	var durableHead int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(cell_id), 0) FROM evaluations WHERE session_id = ?`, record.SessionID).Scan(&durableHead); err != nil {
		return fmt.Errorf("persist evaluation fenced: read durable head: %w", err)
	}
	actualNext := durableHead + 1
	if actualNext != fence.ExpectedCellID || record.CellID != fence.ExpectedCellID {
		return &WriteConflictError{SessionID: record.SessionID, Expected: fence.ExpectedCellID, Actual: actualNext}
	}
	if err := persistEvaluationTx(ctx, tx, record); err != nil {
		return err
	}
	if s.beforeEvaluationCommit != nil {
		if err := s.beforeEvaluationCommit(); err != nil {
			return fmt.Errorf("persist evaluation fenced: before commit: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist evaluation fenced: commit: %w", err)
	}
	return nil
}

func loadSessionLeaseTx(ctx context.Context, tx *sql.Tx, sessionID string) (SessionLease, bool, error) {
	var lease SessionLease
	var leaseUntilRaw string
	err := tx.QueryRowContext(ctx, `SELECT session_id, owner_id, epoch, lease_until FROM session_leases WHERE session_id = ?`, sessionID).Scan(&lease.SessionID, &lease.OwnerID, &lease.Epoch, &leaseUntilRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return SessionLease{}, false, nil
	}
	if err != nil {
		return SessionLease{}, false, fmt.Errorf("load session lease: %w", err)
	}
	lease.LeaseUntil = parseTime(leaseUntilRaw)
	return lease, true, nil
}

func validateLeaseInput(s *Store, sessionID string, ownerID string, ttl time.Duration) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("session lease: store is nil")
	}
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(ownerID) == "" {
		return fmt.Errorf("session lease: session and owner IDs are required")
	}
	if ttl <= 0 {
		return fmt.Errorf("session lease: ttl must be positive")
	}
	return nil
}

func normalizeLeaseTime(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}

func formatLeaseTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
