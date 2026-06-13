// Package sessionauthtest provides reusable conformance tests for
// sessionauth.Store implementations.
package sessionauthtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

// NewStore constructs an empty Store for a single contract test.
type NewStore func(testing.TB) sessionauth.Store

// RunStoreContract verifies the storage semantics that all session stores must
// preserve before they are safe to use behind sessionauth.Manager.
func RunStoreContract(t *testing.T, newStore NewStore) {
	t.Helper()
	t.Run("create get and clone isolation", func(t *testing.T) {
		store := newStore(t)
		originalMFAAt := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
		mfaAt := originalMFAAt
		session := sessionauth.Session{
			ID:                "session-create",
			UserID:            "u1",
			KeycloakSub:       "kc-u1",
			Email:             "u1@example.test",
			EmailVerified:     true,
			TenantIDs:         []string{"o1", "o2"},
			CSRFToken:         "csrf-token",
			MFAAt:             &mfaAt,
			CreatedAt:         mfaAt,
			LastSeenAt:        mfaAt,
			IdleExpiresAt:     mfaAt.Add(30 * time.Minute),
			AbsoluteExpiresAt: mfaAt.Add(12 * time.Hour),
			Claims:            map[string]any{"role": "admin"},
		}
		if err := store.Create(context.Background(), session); err != nil {
			t.Fatalf("create: %v", err)
		}

		session.TenantIDs[0] = "mutated"
		session.Claims["role"] = "mutated"
		*mfaAtPointer(t, &session) = mfaAt.Add(time.Hour)

		got, err := store.Get(context.Background(), "session-create")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.UserID != "u1" || got.TenantIDs[0] != "o1" || got.Claims["role"] != "admin" || !got.MFAAt.Equal(originalMFAAt) {
			t.Fatalf("stored session was mutated through caller-owned input: %#v", got)
		}

		got.TenantIDs[0] = "changed-through-get"
		got.Claims["role"] = "changed-through-get"
		*mfaAtPointer(t, got) = mfaAt.Add(2 * time.Hour)

		again, err := store.Get(context.Background(), "session-create")
		if err != nil {
			t.Fatalf("get again: %v", err)
		}
		if again.TenantIDs[0] != "o1" || again.Claims["role"] != "admin" || !again.MFAAt.Equal(originalMFAAt) {
			t.Fatalf("stored session was mutated through returned value: %#v", again)
		}
	})

	t.Run("touch updates last seen and idle expiry", func(t *testing.T) {
		store := newStore(t)
		created := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
		if err := store.Create(context.Background(), sessionauth.Session{ID: "session-touch", UserID: "u1", CreatedAt: created, LastSeenAt: created, IdleExpiresAt: created.Add(time.Minute)}); err != nil {
			t.Fatalf("create: %v", err)
		}
		touched := created.Add(10 * time.Second)
		idleExpiresAt := touched.Add(30 * time.Minute)
		if err := store.Touch(context.Background(), "session-touch", touched, idleExpiresAt); err != nil {
			t.Fatalf("touch: %v", err)
		}
		got, err := store.Get(context.Background(), "session-touch")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if !got.LastSeenAt.Equal(touched) || !got.IdleExpiresAt.Equal(idleExpiresAt) {
			t.Fatalf("touch did not update timestamps: %#v", got)
		}
	})

	t.Run("rotate replaces old session atomically", func(t *testing.T) {
		store := newStore(t)
		now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
		if err := store.Create(context.Background(), sessionauth.Session{ID: "session-old", UserID: "u1", CreatedAt: now}); err != nil {
			t.Fatalf("create: %v", err)
		}
		next := sessionauth.Session{ID: "session-next", UserID: "u1", CSRFToken: "next-csrf", CreatedAt: now.Add(time.Second)}
		if err := store.Rotate(context.Background(), "session-old", next); err != nil {
			t.Fatalf("rotate: %v", err)
		}
		if _, err := store.Get(context.Background(), "session-old"); err == nil {
			t.Fatalf("old session should not be loadable after rotate")
		}
		got, err := store.Get(context.Background(), "session-next")
		if err != nil {
			t.Fatalf("next session missing: %v", err)
		}
		if got.UserID != "u1" || got.CSRFToken != "next-csrf" {
			t.Fatalf("unexpected next session: %#v", got)
		}
	})

	t.Run("revoke marks existing sessions revoked and ignores missing", func(t *testing.T) {
		store := newStore(t)
		now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
		if err := store.Create(context.Background(), sessionauth.Session{ID: "session-revoke", UserID: "u1", CreatedAt: now}); err != nil {
			t.Fatalf("create: %v", err)
		}
		if err := store.Revoke(context.Background(), "session-revoke"); err != nil {
			t.Fatalf("revoke existing: %v", err)
		}
		got, err := store.Get(context.Background(), "session-revoke")
		if err != nil {
			t.Fatalf("get revoked: %v", err)
		}
		if got.RevokedAt == nil {
			t.Fatalf("expected revoked timestamp: %#v", got)
		}
		if err := store.Revoke(context.Background(), "session-missing"); err != nil {
			t.Fatalf("revoke missing should be idempotent: %v", err)
		}
	})

	t.Run("missing session is an invalid cookie", func(t *testing.T) {
		store := newStore(t)
		_, err := store.Get(context.Background(), "missing")
		if !errors.Is(err, sessionauth.ErrInvalidCookie) {
			t.Fatalf("missing get err=%v", err)
		}
	})
}

func mfaAtPointer(t *testing.T, session *sessionauth.Session) *time.Time {
	t.Helper()
	if session.MFAAt == nil {
		t.Fatalf("session missing MFAAt")
	}
	return session.MFAAt
}
