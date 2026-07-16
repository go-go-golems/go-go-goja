package replsession

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/repldb"
)

func (s *Service) ownershipEnabled(policy SessionPolicy) bool {
	return s != nil && s.leaseStore != nil && policy.Persist.Enabled
}

func (s *Service) acquireSessionLease(ctx context.Context, sessionID string, policy SessionPolicy) (*repldb.SessionLease, error) {
	if !s.ownershipEnabled(policy) {
		return nil, nil
	}
	lease, err := s.leaseStore.AcquireSessionLease(ctx, sessionID, s.ownerID, s.nowUTC(), s.leaseTTL)
	if err != nil {
		return nil, err
	}
	return &lease, nil
}

func (s *Service) releaseSessionLease(ctx context.Context, state *sessionState) error {
	if s == nil || s.leaseStore == nil || state == nil || state.lease == nil {
		return nil
	}
	lease := *state.lease
	if err := s.leaseStore.ReleaseSessionLease(ctx, lease); err != nil {
		return err
	}
	state.lease = nil
	return nil
}

func (s *Service) renewSessionLease(ctx context.Context, state *sessionState) (repldb.SessionLease, error) {
	if s == nil || s.leaseStore == nil || state == nil || state.lease == nil {
		return repldb.SessionLease{}, nil
	}
	renewed, err := s.leaseStore.RenewSessionLease(ctx, *state.lease, s.nowUTC(), s.leaseTTL)
	if err != nil {
		if callerErr := contextError(ctx); callerErr != nil {
			return repldb.SessionLease{}, callerErr
		}
		state.markFenced(err)
		return repldb.SessionLease{}, state.evaluationHealthError()
	}
	state.lease = &renewed
	return renewed, nil
}

func (s *Service) startSessionLeaseGuard(ctx context.Context, state *sessionState) (context.Context, func() error, error) {
	if s == nil || s.leaseStore == nil || state == nil || state.lease == nil {
		return ctx, func() error { return nil }, nil
	}
	lease, err := s.renewSessionLease(ctx, state)
	if err != nil {
		return nil, nil, err
	}
	guardCtx, stop := s.startLeaseGuard(ctx, lease)
	return guardCtx, stop, nil
}

func (s *Service) startLeaseGuard(ctx context.Context, lease repldb.SessionLease) (context.Context, func() error) {
	ctx = nonNilContext(ctx)
	guardCtx, cancel := context.WithCancelCause(ctx)
	interval := s.leaseTTL / 3
	if interval <= 0 {
		interval = time.Millisecond
	}
	stopCh := make(chan struct{})
	done := make(chan struct{})
	var (
		once     sync.Once
		errMu    sync.Mutex
		guardErr error
	)
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		current := lease
		for {
			select {
			case <-stopCh:
				return
			case <-guardCtx.Done():
				return
			case <-ticker.C:
				renewed, err := s.leaseStore.RenewSessionLease(guardCtx, current, s.nowUTC(), s.leaseTTL)
				if err != nil {
					if guardCtx.Err() != nil {
						return
					}
					errMu.Lock()
					guardErr = err
					errMu.Unlock()
					cancel(err)
					return
				}
				current = renewed
			}
		}
	}()
	return guardCtx, func() error {
		once.Do(func() { close(stopCh) })
		<-done
		cancel(context.Canceled)
		errMu.Lock()
		defer errMu.Unlock()
		return guardErr
	}
}

func (s *Service) nowUTC() time.Time {
	if s == nil || s.now == nil {
		return time.Now().UTC()
	}
	return s.now().UTC()
}

func leaseFence(lease *repldb.SessionLease, expectedCellID int) *repldb.WriteFence {
	if lease == nil {
		return nil
	}
	return &repldb.WriteFence{OwnerID: lease.OwnerID, Epoch: lease.Epoch, ExpectedCellID: expectedCellID}
}

func isOwnershipError(err error) bool {
	return errors.Is(err, repldb.ErrSessionOwned) || errors.Is(err, repldb.ErrLeaseLost) || errors.Is(err, repldb.ErrWriteConflict)
}
