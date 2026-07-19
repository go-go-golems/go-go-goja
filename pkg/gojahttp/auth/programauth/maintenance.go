package programauth

import (
	"context"
	"fmt"
	"time"
)

// ExpiredRecordCleaner is implemented by stores that remove expired retained
// records. It keeps host maintenance independent of a particular auth flow.
type ExpiredRecordCleaner interface {
	Cleanup(ctx context.Context) (int64, error)
}

// MaintenanceService owns bounded credential-retention operations. Callers
// should invoke it from a scheduled operator job, never from JavaScript.
type MaintenanceService struct {
	Tokens       OAuthTokenService
	Transactions ExpiredRecordCleaner
}

func (s MaintenanceService) PurgeExpired(ctx context.Context, before time.Time) (int, error) {
	if s.Tokens.AccessTokens == nil || s.Tokens.RefreshTokens == nil {
		return 0, fmt.Errorf("maintenance token stores are required")
	}
	count, err := s.Tokens.PurgeExpiredCredentials(ctx, before)
	if err != nil {
		return 0, err
	}
	if s.Transactions == nil {
		return count, nil
	}
	removed, err := s.Transactions.Cleanup(ctx)
	if err != nil {
		return 0, err
	}
	return count + int(removed), nil
}
