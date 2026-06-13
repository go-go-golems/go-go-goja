package sqlstore

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/internal/sessionauthtest"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

func TestSQLiteStoreContract(t *testing.T) {
	sessionauthtest.RunStoreContract(t, func(t testing.TB) sessionauth.Store {
		return newSQLiteStore(t)
	})
}

func TestSQLiteStorePersistsFullSessionProjection(t *testing.T) {
	store := newSQLiteStore(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC).Round(0)
	mfaAt := now.Add(-time.Minute)
	revokedAt := now.Add(time.Minute)
	session := sessionauth.Session{
		ID:                "full-session",
		UserID:            "u1",
		KeycloakSub:       "kc-u1",
		Email:             "u1@example.test",
		EmailVerified:     true,
		TenantIDs:         []string{"o1", "o2"},
		CSRFToken:         "csrf",
		MFAAt:             &mfaAt,
		CreatedAt:         now,
		LastSeenAt:        now,
		IdleExpiresAt:     now.Add(30 * time.Minute),
		AbsoluteExpiresAt: now.Add(12 * time.Hour),
		RevokedAt:         &revokedAt,
		Claims:            map[string]any{"email": "u1@example.test", "emailVerified": true},
	}
	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.UserID != "u1" || got.KeycloakSub != "kc-u1" || got.Email != "u1@example.test" || !got.EmailVerified {
		t.Fatalf("unexpected identity fields: %#v", got)
	}
	if len(got.TenantIDs) != 2 || got.TenantIDs[0] != "o1" || got.TenantIDs[1] != "o2" {
		t.Fatalf("unexpected tenant ids: %#v", got.TenantIDs)
	}
	if got.MFAAt == nil || !got.MFAAt.Equal(mfaAt) || got.RevokedAt == nil || !got.RevokedAt.Equal(revokedAt) {
		t.Fatalf("unexpected optional timestamps: %#v", got)
	}
	if got.Claims["email"] != "u1@example.test" || got.Claims["emailVerified"] != true {
		t.Fatalf("unexpected claims: %#v", got.Claims)
	}
}

func TestPostgresSchemaAndPlaceholders(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := New(Config{DB: db, Dialect: DialectPostgres})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if got := store.touchQuery(); !strings.Contains(got, "$1") || !strings.Contains(got, "$2") || !strings.Contains(got, "$3") {
		t.Fatalf("unexpected postgres touch query: %s", got)
	}
	if !strings.Contains(store.Schema(), "JSONB") || !strings.Contains(store.Schema(), "TIMESTAMPTZ") {
		t.Fatalf("postgres schema missing expected types: %s", store.Schema())
	}
}

func newSQLiteStore(t testing.TB) *Store {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:sessionauth-sqlstore?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	store, err := New(Config{DB: db, Dialect: DialectSQLite, Now: func() time.Time {
		return time.Date(2026, 6, 12, 13, 0, 0, 0, time.UTC)
	}})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := store.ApplySchema(context.Background()); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return store
}
