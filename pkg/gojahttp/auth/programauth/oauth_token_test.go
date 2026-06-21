package programauth_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth"
)

func TestOAuthTokenPairIssueAuthenticateRefreshAndReuse(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 20, 20, 0, 0, 0, time.UTC)
	current := now
	agents := programauth.AgentService{Store: programauth.NewMemoryAgentStore(), Now: func() time.Time { return current }, NewID: func() (string, error) { return "agt_oauth", nil }}
	_, err := agents.CreateAgent(ctx, programauth.AgentCreateSpec{Name: "device", Kind: programauth.AgentKindDevice, TenantID: "o1", Policy: mustGrantSet(t, gojahttp.Grant{Action: "report.read", TenantID: "o1"})})
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}
	service := newOAuthTokenTestService(agents, func() time.Time { return current })
	issued, err := service.IssueTokenPair(ctx, programauth.OAuthTokenIssueSpec{AgentID: "agt_oauth", AccessTTL: time.Minute, RefreshTTL: time.Hour, Grants: mustGrantSet(t, gojahttp.Grant{Action: "report.read", TenantID: "o1"})})
	if err != nil {
		t.Fatalf("IssueTokenPair: %v", err)
	}
	if issued.AccessValue == "" || issued.RefreshValue == "" || issued.AccessToken.FamilyID == "" || issued.RefreshToken.FamilyID != issued.AccessToken.FamilyID {
		t.Fatalf("issued token pair missing values or family: %#v", issued)
	}
	result, err := service.AuthenticateBearer(ctx, issued.AccessValue, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser})
	if err != nil {
		t.Fatalf("AuthenticateBearer(access): %v", err)
	}
	if result.Method != gojahttp.AuthMethodAccessToken || result.PrincipalKind != gojahttp.PrincipalKindAgent || result.PrincipalID != "agt_oauth" || result.CSRFRequired {
		t.Fatalf("unexpected auth result: %#v", result)
	}
	if _, err := service.AuthenticateBearer(ctx, issued.RefreshValue, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}); !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("refresh token authenticated planned route, err=%v", err)
	}
	current = current.Add(30 * time.Second)
	refreshed, err := service.RefreshTokenPair(ctx, issued.RefreshValue, time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("RefreshTokenPair: %v", err)
	}
	if refreshed.RefreshToken.Generation != issued.RefreshToken.Generation+1 || refreshed.RefreshToken.FamilyID != issued.RefreshToken.FamilyID {
		t.Fatalf("unexpected rotated refresh metadata: %#v", refreshed.RefreshToken)
	}
	if _, err := service.AuthenticateBearer(ctx, refreshed.AccessValue, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}); err != nil {
		t.Fatalf("AuthenticateBearer(refreshed access): %v", err)
	}
	if _, err := service.RefreshTokenPair(ctx, issued.RefreshValue, time.Minute, time.Hour); !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("old refresh reuse should be unauthenticated, err=%v", err)
	}
	if _, err := service.RefreshTokenPair(ctx, refreshed.RefreshValue, time.Minute, time.Hour); !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("family revocation should reject replacement refresh token, err=%v", err)
	}
}

func TestOAuthTokenRefreshDoubleUseRevokesFamily(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 20, 21, 0, 0, 0, time.UTC)
	agents := programauth.AgentService{Store: programauth.NewMemoryAgentStore(), Now: func() time.Time { return now }, NewID: func() (string, error) { return "agt_race", nil }}
	_, err := agents.CreateAgent(ctx, programauth.AgentCreateSpec{Name: "race", Kind: programauth.AgentKindCI, Policy: mustGrantSet(t, gojahttp.Grant{Action: "job.run"})})
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}
	service := newOAuthTokenTestService(agents, func() time.Time { return now })
	issued, err := service.IssueTokenPair(ctx, programauth.OAuthTokenIssueSpec{AgentID: "agt_race", Grants: mustGrantSet(t, gojahttp.Grant{Action: "job.run"})})
	if err != nil {
		t.Fatalf("IssueTokenPair: %v", err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, refreshErr := service.RefreshTokenPair(ctx, issued.RefreshValue, time.Minute, time.Hour)
			results <- refreshErr
		}()
	}
	wg.Wait()
	close(results)
	successes := 0
	failures := 0
	for refreshErr := range results {
		if refreshErr == nil {
			successes++
			continue
		}
		if errors.Is(refreshErr, gojahttp.ErrUnauthenticated) {
			failures++
			continue
		}
		t.Fatalf("unexpected refresh error: %v", refreshErr)
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("expected one successful rotation and one reuse failure, got successes=%d failures=%d", successes, failures)
	}
}

func TestAccessTokenExpiresAndDisabledAgentFails(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 20, 22, 0, 0, 0, time.UTC)
	current := now
	agentStore := programauth.NewMemoryAgentStore()
	agents := programauth.AgentService{Store: agentStore, Now: func() time.Time { return current }, NewID: func() (string, error) { return "agt_exp_access", nil }}
	_, err := agents.CreateAgent(ctx, programauth.AgentCreateSpec{Name: "short", Policy: mustGrantSet(t, gojahttp.Grant{Action: "report.read"})})
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}
	service := newOAuthTokenTestService(agents, func() time.Time { return current })
	issued, err := service.IssueTokenPair(ctx, programauth.OAuthTokenIssueSpec{AgentID: "agt_exp_access", AccessTTL: time.Minute, Grants: mustGrantSet(t, gojahttp.Grant{Action: "report.read"})})
	if err != nil {
		t.Fatalf("IssueTokenPair: %v", err)
	}
	current = current.Add(2 * time.Minute)
	if _, err := service.AuthenticateBearer(ctx, issued.AccessValue, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}); !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("expired access token should be unauthenticated, err=%v", err)
	}
	current = now
	if _, err := agents.DisableAgent(ctx, "agt_exp_access"); err != nil {
		t.Fatalf("DisableAgent: %v", err)
	}
	if _, err := service.AuthenticateBearer(ctx, issued.AccessValue, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}); !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("disabled agent should reject access token, err=%v", err)
	}
}

func newOAuthTokenTestService(agents programauth.AgentService, now func() time.Time) programauth.OAuthTokenService {
	var mu sync.Mutex
	ids := map[string]int{}
	randomByte := byte(1)
	return programauth.OAuthTokenService{
		AccessTokens:  programauth.NewMemoryAccessTokenStore(),
		RefreshTokens: programauth.NewMemoryRefreshTokenStore(),
		Agents:        agents,
		Now:           now,
		NewID: func(prefix string) (string, error) {
			mu.Lock()
			defer mu.Unlock()
			ids[prefix]++
			return fmt.Sprintf("%s_test_%d", prefix, ids[prefix]), nil
		},
		Random: func(n int) ([]byte, error) {
			mu.Lock()
			defer mu.Unlock()
			out := make([]byte, n)
			for i := range out {
				out[i] = randomByte
				randomByte++
				if randomByte == 0 {
					randomByte = 1
				}
			}
			return out, nil
		},
	}
}
