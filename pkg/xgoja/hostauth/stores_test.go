package hostauth

import (
	"context"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

func TestBuildStoresMemory(t *testing.T) {
	ctx := context.Background()
	stores, err := BuildStores(ctx, mustResolveStores(t, Config{}))
	if err != nil {
		t.Fatalf("BuildStores: %v", err)
	}
	defer func() {
		if err := stores.Close(ctx); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()
	if stores.Session == nil || stores.Audit == nil || stores.Capability == nil {
		t.Fatalf("stores missing: %#v", stores)
	}
	if stores.AppAuth.Users == nil || stores.AppAuth.Memberships == nil || stores.AppAuth.Resources == nil {
		t.Fatalf("appauth stores missing: %#v", stores.AppAuth)
	}
	if len(stores.Closers) != 0 {
		t.Fatalf("memory stores registered closers: %d", len(stores.Closers))
	}

	exerciseStores(t, ctx, stores)
}

func TestBuildStoresSQLiteSharedDBAndApplySchema(t *testing.T) {
	ctx := context.Background()
	applySchema := true
	stores, err := BuildStores(ctx, mustResolveStores(t, Config{Mode: ModeDev, Stores: StoresConfig{Default: StoreConfig{
		Driver:      "sqlite",
		DSN:         "file:hostauth-buildstores-shared?mode=memory&cache=shared",
		ApplySchema: &applySchema,
	}}}))
	if err != nil {
		t.Fatalf("BuildStores: %v", err)
	}
	defer func() {
		if err := stores.Close(ctx); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()
	if len(stores.Closers) != 1 {
		t.Fatalf("closers = %d, want one shared sqlite DB closer", len(stores.Closers))
	}
	exerciseStores(t, ctx, stores)
}

func TestBuildStoresSQLiteWithoutApplySchemaDoesNotCreateTables(t *testing.T) {
	ctx := context.Background()
	stores, err := BuildStores(ctx, mustResolveStores(t, Config{Mode: ModeDev, Stores: StoresConfig{Default: StoreConfig{
		Driver: "sqlite",
		DSN:    "file:hostauth-buildstores-no-schema?mode=memory&cache=shared",
	}}}))
	if err != nil {
		t.Fatalf("BuildStores: %v", err)
	}
	defer func() {
		if err := stores.Close(ctx); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()
	if err := stores.Session.Create(ctx, testSession(time.Now())); err == nil {
		t.Fatalf("Create session succeeded without schema")
	}
}

func TestBuildStoresPostgresConstructsWithoutConnectingWhenSchemaDisabled(t *testing.T) {
	ctx := context.Background()
	stores, err := BuildStores(ctx, mustResolveStores(t, Config{Mode: ModeDev, Stores: StoresConfig{Default: StoreConfig{
		Driver: "postgres",
		DSN:    "postgres://goja:goja@127.0.0.1:1/goja_auth?sslmode=disable",
	}}}))
	if err != nil {
		t.Fatalf("BuildStores: %v", err)
	}
	if len(stores.Closers) != 1 {
		t.Fatalf("closers = %d, want one shared postgres DB closer", len(stores.Closers))
	}
	if err := stores.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func mustResolveStores(t *testing.T, cfg Config) ResolvedStoresConfig {
	t.Helper()
	resolved, err := ResolveConfig(cfg, ResolveOptions{})
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	return resolved.Stores
}

func exerciseStores(t *testing.T, ctx context.Context, stores *StoreBundle) {
	t.Helper()
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	session := testSession(now)
	if err := stores.Session.Create(ctx, session); err != nil {
		t.Fatalf("Create session: %v", err)
	}
	loadedSession, err := stores.Session.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get session: %v", err)
	}
	if loadedSession.UserID != session.UserID || loadedSession.CSRFToken != session.CSRFToken {
		t.Fatalf("loaded session = %#v", loadedSession)
	}

	if err := stores.Audit.InsertAuditRecord(ctx, audit.Record{Event: "test.event", Outcome: "success", Method: "GET", Pattern: "/", CreatedAt: now}); err != nil {
		t.Fatalf("InsertAuditRecord: %v", err)
	}

	user, err := stores.AppAuth.Users.UpsertFromOIDC(ctx, "https://issuer.example.test", "sub-1", "demo@example.test", true)
	if err != nil {
		t.Fatalf("UpsertFromOIDC: %v", err)
	}
	loadedUser, err := stores.AppAuth.Users.ByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if loadedUser.Email != "demo@example.test" {
		t.Fatalf("loaded user = %#v", loadedUser)
	}

	capService := capability.Service{Store: stores.Capability, Now: func() time.Time { return now }}
	issued, err := capService.Issue(ctx, capability.IssueSpec{Purpose: "test", SubjectID: user.ID, TTL: time.Hour, SingleUse: true})
	if err != nil {
		t.Fatalf("Issue capability: %v", err)
	}
	redeemed, err := capService.Redeem(ctx, "test", issued.Token)
	if err != nil {
		t.Fatalf("Redeem capability: %v", err)
	}
	if redeemed.ID != issued.Capability.ID {
		t.Fatalf("redeemed = %#v", redeemed)
	}
}

func testSession(now time.Time) sessionauth.Session {
	return sessionauth.Session{
		ID:                "abcdefghijklmnopqrstuv",
		UserID:            "user-1",
		Email:             "demo@example.test",
		EmailVerified:     true,
		TenantIDs:         []string{"tenant-1"},
		CSRFToken:         "csrf-token",
		CreatedAt:         now,
		LastSeenAt:        now,
		IdleExpiresAt:     now.Add(30 * time.Minute),
		AbsoluteExpiresAt: now.Add(12 * time.Hour),
		Claims:            map[string]any{"role": "admin"},
	}
}

var _ appauth.UserStore = appauth.NewMemoryStore()
