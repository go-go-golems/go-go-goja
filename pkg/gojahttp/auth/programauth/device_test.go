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

func TestDeviceAuthorizationPendingSlowDownApproveAndPoll(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC)
	current := now
	service := newDeviceTestService(func() time.Time { return current })
	started, err := service.StartDeviceAuthorization(ctx, programauth.DeviceStartSpec{ClientName: "goja-cli", TenantID: "o1", VerificationURI: "https://app.example.test/device", Grants: mustGrantSet(t, gojahttp.Grant{Action: "report.read", TenantID: "o1"})})
	if err != nil {
		t.Fatalf("StartDeviceAuthorization: %v", err)
	}
	if started.DeviceCode == "" || started.UserCode == "" || started.Device.VerificationURIComplete == "" || started.Device.PollIntervalSeconds != 5 {
		t.Fatalf("started = %#v", started)
	}
	if _, err := service.PollDeviceAuthorization(ctx, started.DeviceCode); !errors.Is(err, programauth.ErrDeviceAuthorizationPending) {
		t.Fatalf("first poll should be pending, err=%v", err)
	}
	if _, err := service.PollDeviceAuthorization(ctx, started.DeviceCode); !errors.Is(err, programauth.ErrDeviceSlowDown) {
		t.Fatalf("second immediate poll should slow down, err=%v", err)
	}
	approved, err := service.ApproveDeviceAuthorization(ctx, programauth.DeviceApprovalSpec{UserCode: started.UserCode, SubjectUserID: "u1"})
	if err != nil {
		t.Fatalf("ApproveDeviceAuthorization: %v", err)
	}
	if approved.AgentID == "" || approved.SubjectUserID != "u1" || approved.TenantID != "o1" {
		t.Fatalf("approved = %#v", approved)
	}
	current = current.Add(20 * time.Second)
	issued, err := service.PollDeviceAuthorization(ctx, started.DeviceCode)
	if err != nil {
		t.Fatalf("approved poll: %v", err)
	}
	if issued.AccessValue == "" || issued.RefreshValue == "" || issued.AccessToken.AgentID != approved.AgentID {
		t.Fatalf("issued = %#v", issued)
	}
	result, err := service.OAuthTokens.AuthenticateBearer(ctx, issued.AccessValue, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser})
	if err != nil {
		t.Fatalf("AuthenticateBearer(access): %v", err)
	}
	if result.Method != gojahttp.AuthMethodAccessToken || result.PrincipalKind != gojahttp.PrincipalKindAgent || result.PrincipalID != approved.AgentID {
		t.Fatalf("result = %#v", result)
	}
	if _, err := service.PollDeviceAuthorization(ctx, started.DeviceCode); !errors.Is(err, programauth.ErrDeviceConsumed) {
		t.Fatalf("consumed device code should fail, err=%v", err)
	}
}

func TestDeviceAuthorizationExpiryAndDeny(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 21, 1, 0, 0, 0, time.UTC)
	current := now
	service := newDeviceTestService(func() time.Time { return current })
	expired, err := service.StartDeviceAuthorization(ctx, programauth.DeviceStartSpec{ClientName: "expired", ExpiresIn: time.Minute, Grants: mustGrantSet(t, gojahttp.Grant{Action: "read"})})
	if err != nil {
		t.Fatalf("StartDeviceAuthorization(expired): %v", err)
	}
	current = current.Add(2 * time.Minute)
	if _, err := service.ApproveDeviceAuthorization(ctx, programauth.DeviceApprovalSpec{UserCode: expired.UserCode, SubjectUserID: "u1"}); !errors.Is(err, programauth.ErrDeviceExpired) {
		t.Fatalf("expired approval should fail, err=%v", err)
	}
	current = now
	denied, err := service.StartDeviceAuthorization(ctx, programauth.DeviceStartSpec{ClientName: "denied", Grants: mustGrantSet(t, gojahttp.Grant{Action: "read"})})
	if err != nil {
		t.Fatalf("StartDeviceAuthorization(denied): %v", err)
	}
	if _, err := service.DenyDeviceAuthorization(ctx, denied.UserCode); err != nil {
		t.Fatalf("DenyDeviceAuthorization: %v", err)
	}
	if _, err := service.PollDeviceAuthorization(ctx, denied.DeviceCode); !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("denied device poll should be unauthenticated, err=%v", err)
	}
}

func newDeviceTestService(now func() time.Time) programauth.DeviceService {
	var mu sync.Mutex
	ids := map[string]int{}
	randomByte := byte(1)
	agentStore := programauth.NewMemoryAgentStore()
	agents := programauth.AgentService{Store: agentStore, Now: now, NewID: func() (string, error) {
		mu.Lock()
		defer mu.Unlock()
		ids["agt"]++
		return fmt.Sprintf("agt_device_%d", ids["agt"]), nil
	}}
	oauth := programauth.OAuthTokenService{
		AccessTokens:  programauth.NewMemoryAccessTokenStore(),
		RefreshTokens: programauth.NewMemoryRefreshTokenStore(),
		Agents:        agents,
		Now:           now,
		NewID: func(prefix string) (string, error) {
			mu.Lock()
			defer mu.Unlock()
			ids[prefix]++
			return fmt.Sprintf("%s_device_%d", prefix, ids[prefix]), nil
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
	return programauth.DeviceService{
		Store:           programauth.NewMemoryDeviceAuthorizationStore(),
		Agents:          agents,
		OAuthTokens:     oauth,
		Now:             now,
		VerificationURI: "https://app.example.test/device",
		NewID: func(prefix string) (string, error) {
			mu.Lock()
			defer mu.Unlock()
			ids[prefix]++
			return fmt.Sprintf("%s_device_%d", prefix, ids[prefix]), nil
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
