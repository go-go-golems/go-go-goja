package sqlstore_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth/sqlstore"
)

func TestSQLStoreAgentsCreateListDisableAndClone(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	now := time.Date(2026, 6, 21, 19, 0, 0, 0, time.UTC)
	policy := mustGrantSet(t, gojahttp.Grant{Action: "report.read", TenantID: "o1"})
	created, err := store.CreateAgent(ctx, programauth.Agent{ID: "agt_sql", Name: "sql bot", Kind: programauth.AgentKindService, OwnerUserID: "u1", TenantID: "o1", CreatedBy: "test", CreatedAt: now, UpdatedAt: now, Policy: policy})
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}
	created.Policy.Grants[0].Action = "mutated"
	loaded, err := store.GetAgent(ctx, "agt_sql")
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if got, want := loaded.Policy.ScopeStrings(), []string{"tenant:o1:report.read"}; !sameStrings(got, want) {
		t.Fatalf("loaded policy = %#v, want %#v", got, want)
	}
	listed, err := store.ListAgents(ctx, programauth.AgentQuery{OwnerUserID: "u1", TenantID: "o1"})
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != "agt_sql" {
		t.Fatalf("listed = %#v", listed)
	}
	disabled, err := store.DisableAgent(ctx, "agt_sql", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("DisableAgent: %v", err)
	}
	if disabled.DisabledAt == nil {
		t.Fatalf("disabled agent missing DisabledAt: %#v", disabled)
	}
	listed, err = store.ListAgents(ctx, programauth.AgentQuery{OwnerUserID: "u1", TenantID: "o1"})
	if err != nil {
		t.Fatalf("ListAgents after disable: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("disabled agent should be hidden by default: %#v", listed)
	}
	listed, err = store.ListAgents(ctx, programauth.AgentQuery{OwnerUserID: "u1", TenantID: "o1", IncludeDisabled: true})
	if err != nil {
		t.Fatalf("ListAgents include disabled: %v", err)
	}
	if len(listed) != 1 || listed[0].DisabledAt == nil {
		t.Fatalf("include disabled listed = %#v", listed)
	}
}

func TestSQLStoreAPITokenServiceIssueAuthenticateListRevoke(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	now := time.Date(2026, 6, 21, 20, 0, 0, 0, time.UTC)
	agents := programauth.AgentService{Store: store, Now: func() time.Time { return now }, NewID: func() (string, error) { return "agt_token", nil }}
	_, err := agents.CreateAgent(ctx, programauth.AgentCreateSpec{Name: "sql token bot", Kind: programauth.AgentKindCI, TenantID: "o1", Policy: mustGrantSet(t, gojahttp.Grant{Action: "job.run", TenantID: "o1"})})
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}
	tokens := programauth.APITokenService{Store: store, Agents: agents, Now: func() time.Time { return now }, NewID: func() (string, error) { return "tok_sql", nil }, Random: deterministicRandom()}
	expiresAt := now.Add(time.Hour)
	issued, err := tokens.IssueAPIToken(ctx, programauth.APITokenIssueSpec{Name: "sql token", AgentID: "agt_token", CreatedBy: "test", ExpiresAt: &expiresAt, Grants: mustGrantSet(t, gojahttp.Grant{Action: "job.run", TenantID: "o1"})})
	if err != nil {
		t.Fatalf("IssueAPIToken: %v", err)
	}
	if issued.Value == "" || issued.Token.TokenPrefix == "" || issued.Token.CredentialHint == "" {
		t.Fatalf("issued = %#v", issued)
	}
	result, err := tokens.AuthenticateBearer(ctx, issued.Value, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser})
	if err != nil {
		t.Fatalf("AuthenticateBearer: %v", err)
	}
	if result.Method != gojahttp.AuthMethodAPIToken || result.PrincipalID != "agt_token" || result.CSRFRequired {
		t.Fatalf("auth result = %#v", result)
	}
	listed, err := tokens.ListAPITokens(ctx, programauth.APITokenQuery{AgentID: "agt_token"})
	if err != nil {
		t.Fatalf("ListAPITokens: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != "tok_sql" || listed[0].LastUsedAt == nil {
		t.Fatalf("listed tokens = %#v", listed)
	}
	if _, err := tokens.RevokeAPIToken(ctx, "tok_sql"); err != nil {
		t.Fatalf("RevokeAPIToken: %v", err)
	}
	if _, err := tokens.AuthenticateBearer(ctx, issued.Value, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}); !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("revoked token should be unauthenticated, err=%v", err)
	}
}

func TestSQLStoreOAuthTokenServiceIssueRefreshAndReuse(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	now := time.Date(2026, 6, 21, 21, 0, 0, 0, time.UTC)
	current := now
	agents := programauth.AgentService{Store: store, Now: func() time.Time { return current }, NewID: func() (string, error) { return "agt_oauth_sql", nil }}
	_, err := agents.CreateAgent(ctx, programauth.AgentCreateSpec{Name: "sql oauth bot", Kind: programauth.AgentKindDevice, TenantID: "o1", Policy: mustGrantSet(t, gojahttp.Grant{Action: "report.read", TenantID: "o1"})})
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}
	ids := map[string]int{}
	oauth := programauth.OAuthTokenService{
		AccessTokens:  store,
		RefreshTokens: store,
		Agents:        agents,
		Now:           func() time.Time { return current },
		NewID: func(prefix string) (string, error) {
			ids[prefix]++
			return fmt.Sprintf("%s_sql_%d", prefix, ids[prefix]), nil
		},
		Random: deterministicRandom(),
	}
	issued, err := oauth.IssueTokenPair(ctx, programauth.OAuthTokenIssueSpec{AgentID: "agt_oauth_sql", AccessTTL: time.Minute, RefreshTTL: time.Hour, Grants: mustGrantSet(t, gojahttp.Grant{Action: "report.read", TenantID: "o1"})})
	if err != nil {
		t.Fatalf("IssueTokenPair: %v", err)
	}
	if _, err := oauth.AuthenticateBearer(ctx, issued.AccessValue, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}); err != nil {
		t.Fatalf("AuthenticateBearer(access): %v", err)
	}
	if _, err := oauth.AuthenticateBearer(ctx, issued.RefreshValue, gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}); !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("refresh token authenticated planned route, err=%v", err)
	}
	current = current.Add(10 * time.Second)
	refreshed, err := oauth.RefreshTokenPair(ctx, issued.RefreshValue, time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("RefreshTokenPair: %v", err)
	}
	if refreshed.RefreshToken.Generation != issued.RefreshToken.Generation+1 || refreshed.RefreshToken.FamilyID != issued.RefreshToken.FamilyID {
		t.Fatalf("refreshed metadata = %#v", refreshed.RefreshToken)
	}
	if _, err := oauth.RefreshTokenPair(ctx, issued.RefreshValue, time.Minute, time.Hour); !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("old refresh token reuse should be unauthenticated, err=%v", err)
	}
	if _, err := oauth.RefreshTokenPair(ctx, refreshed.RefreshValue, time.Minute, time.Hour); !errors.Is(err, gojahttp.ErrUnauthenticated) {
		t.Fatalf("family revocation should reject replacement refresh token, err=%v", err)
	}
}

func TestSQLStoreDeviceAuthorizationServiceLifecycle(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	now := time.Date(2026, 6, 21, 22, 0, 0, 0, time.UTC)
	current := now
	agents := programauth.AgentService{Store: store, Now: func() time.Time { return current }, NewID: func() (string, error) { return "agt_device_sql", nil }}
	ids := map[string]int{}
	random := deterministicRandom()
	oauth := programauth.OAuthTokenService{
		AccessTokens:  store,
		RefreshTokens: store,
		Agents:        agents,
		Now:           func() time.Time { return current },
		NewID: func(prefix string) (string, error) {
			ids[prefix]++
			return fmt.Sprintf("%s_device_sql_%d", prefix, ids[prefix]), nil
		},
		Random: random,
	}
	devices := programauth.DeviceService{
		Store:           store,
		Agents:          agents,
		OAuthTokens:     oauth,
		Now:             func() time.Time { return current },
		VerificationURI: "https://app.example.test/device",
		NewID: func(prefix string) (string, error) {
			ids[prefix]++
			return fmt.Sprintf("%s_device_sql_%d", prefix, ids[prefix]), nil
		},
		Random: random,
	}
	started, err := devices.StartDeviceAuthorization(ctx, programauth.DeviceStartSpec{ClientName: "sql device", TenantID: "o1", Grants: mustGrantSet(t, gojahttp.Grant{Action: "report.read", TenantID: "o1"})})
	if err != nil {
		t.Fatalf("StartDeviceAuthorization: %v", err)
	}
	if _, err := devices.PollDeviceAuthorization(ctx, started.DeviceCode); !errors.Is(err, programauth.ErrDeviceAuthorizationPending) {
		t.Fatalf("first poll should be pending, err=%v", err)
	}
	if _, err := devices.PollDeviceAuthorization(ctx, started.DeviceCode); !errors.Is(err, programauth.ErrDeviceSlowDown) {
		t.Fatalf("second poll should slow down, err=%v", err)
	}
	approved, err := devices.ApproveDeviceAuthorization(ctx, programauth.DeviceApprovalSpec{UserCode: started.UserCode, SubjectUserID: "u1", TenantID: "o1", Grants: mustGrantSet(t, gojahttp.Grant{Action: "report.read", TenantID: "o1"})})
	if err != nil {
		t.Fatalf("ApproveDeviceAuthorization: %v", err)
	}
	if approved.AgentID == "" || approved.SubjectUserID != "u1" {
		t.Fatalf("approved = %#v", approved)
	}
	current = current.Add(20 * time.Second)
	issued, err := devices.PollDeviceAuthorization(ctx, started.DeviceCode)
	if err != nil {
		t.Fatalf("approved poll: %v", err)
	}
	if issued.AccessValue == "" || issued.RefreshValue == "" {
		t.Fatalf("issued = %#v", issued)
	}
	if _, err := devices.PollDeviceAuthorization(ctx, started.DeviceCode); !errors.Is(err, programauth.ErrDeviceConsumed) {
		t.Fatalf("consumed poll should fail, err=%v", err)
	}
}

func TestSQLStoreDeviceAuthorizationDenyAndDuplicateUserCode(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	now := time.Date(2026, 6, 21, 23, 0, 0, 0, time.UTC)
	device := programauth.DeviceAuthorization{ID: "dev_one", ClientName: "one", DeviceCodeHash: []byte("device-1"), DeviceCodePrefix: "aaaa", UserCodeHash: []byte("user-1"), UserCode: "AAAA-BBBB-CCCC", CreatedAt: now, UpdatedAt: now, ExpiresAt: now.Add(time.Minute), PollInterval: 5 * time.Second, Grants: mustGrantSet(t, gojahttp.Grant{Action: "read"})}
	if _, err := store.CreateDeviceAuthorization(ctx, device); err != nil {
		t.Fatalf("CreateDeviceAuthorization: %v", err)
	}
	device.ID = "dev_two"
	device.DeviceCodeHash = []byte("device-2")
	if _, err := store.CreateDeviceAuthorization(ctx, device); err == nil {
		t.Fatal("expected duplicate user code hash to fail")
	}
	denied, err := store.DenyDeviceAuthorization(ctx, "dev_one", now.Add(time.Second))
	if err != nil {
		t.Fatalf("DenyDeviceAuthorization: %v", err)
	}
	if denied.DeniedAt == nil {
		t.Fatalf("denied missing timestamp: %#v", denied)
	}
	if _, err := store.ConsumeDeviceAuthorization(ctx, "dev_one", now.Add(2*time.Second)); !errors.Is(err, programauth.ErrDeviceDenied) {
		t.Fatalf("denied consume should return ErrDeviceDenied, err=%v", err)
	}
}

func TestSQLStoreRejectsUnsupportedDialect(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := sqlstore.New(sqlstore.Config{DB: db, Dialect: "bogus"}); err == nil {
		t.Fatal("expected unsupported dialect error")
	}
}

func newTestStore(t *testing.T) *sqlstore.Store {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := sqlstore.New(sqlstore.Config{DB: db, Dialect: sqlstore.DialectSQLite})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := store.ApplySchema(context.Background()); err != nil {
		t.Fatalf("ApplySchema: %v", err)
	}
	return store
}

func mustGrantSet(t *testing.T, grants ...gojahttp.Grant) gojahttp.GrantSet {
	t.Helper()
	set, err := gojahttp.NewGrantSet(grants...)
	if err != nil {
		t.Fatalf("NewGrantSet: %v", err)
	}
	return set
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func deterministicRandom() func(int) ([]byte, error) {
	value := byte(1)
	return func(n int) ([]byte, error) {
		out := make([]byte, n)
		for i := range out {
			out[i] = value
			value++
			if value == 0 {
				value = 1
			}
		}
		return out, nil
	}
}

func ExampleStore_Schema() {
	fmt.Println(sqlstore.DialectSQLite)
	// Output: sqlite
}
