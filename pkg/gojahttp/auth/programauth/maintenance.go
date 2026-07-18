package programauth

import (
	"context"
	"fmt"
	"time"
)

// MaintenanceService owns bounded credential-retention operations. Callers
// should invoke it from a scheduled operator job, never from JavaScript.
type MaintenanceService struct{ Tokens OAuthTokenService }

func (s MaintenanceService) PurgeExpired(ctx context.Context, before time.Time) (int, error) {
	if s.Tokens.AccessTokens == nil || s.Tokens.RefreshTokens == nil {
		return 0, fmt.Errorf("maintenance token stores are required")
	}
	return s.Tokens.PurgeExpiredCredentials(ctx, before)
}
