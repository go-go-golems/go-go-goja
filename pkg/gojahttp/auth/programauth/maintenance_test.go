package programauth

import (
	"context"
	"testing"
	"time"
)

func TestMaintenancePurgesExpiredCredentials(t *testing.T) {
	access := NewMemoryAccessTokenStore()
	refresh := NewMemoryRefreshTokenStore()
	now := time.Now()
	_, _ = access.CreateAccessToken(context.Background(), AccessToken{ID: "expired", TokenPrefix: "x", TokenHash: []byte("h"), CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour), ExpiresAt: now.Add(-time.Minute)})
	_, _ = refresh.CreateRefreshToken(context.Background(), RefreshToken{ID: "expired", FamilyID: "f", TokenPrefix: "x", TokenHash: []byte("h"), CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour), ExpiresAt: now.Add(-time.Minute)})
	n, err := (MaintenanceService{Tokens: OAuthTokenService{AccessTokens: access, RefreshTokens: refresh}}).PurgeExpired(context.Background(), now)
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
