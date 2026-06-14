// Package capabilitytest provides reusable conformance tests for
// capability.Store implementations.
package capabilitytest

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
)

// NewStore constructs an empty Store for a single contract test.
type NewStore func(testing.TB) capability.Store

// RunStoreContract verifies token-hash persistence, redemption state changes,
// cloning, expiry, revocation, and single-use atomicity semantics.
func RunStoreContract(t *testing.T, newStore NewStore) {
	t.Helper()

	t.Run("create by id and clone isolation", func(t *testing.T) {
		store := newStore(t)
		createdAt := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
		expiresAt := createdAt.Add(time.Hour)
		capabilityRecord := capability.Capability{
			ID:        "cap-create",
			Purpose:   "email.verify",
			SubjectID: "u1",
			Claims:    map[string]string{"email": "u1@example.test"},
			TokenHash: capability.HashToken("raw-token"),
			ExpiresAt: expiresAt,
			SingleUse: true,
			CreatedBy: "admin",
			CreatedAt: createdAt,
		}
		if err := store.Create(context.Background(), capabilityRecord); err != nil {
			t.Fatalf("create: %v", err)
		}
		capabilityRecord.Claims["email"] = "mutated@example.test"
		capabilityRecord.TokenHash[0] ^= 0xff

		got, err := store.ByID(context.Background(), "cap-create")
		if err != nil {
			t.Fatalf("by id: %v", err)
		}
		if got.Claims["email"] != "u1@example.test" || bytes.Equal(got.TokenHash, capabilityRecord.TokenHash) {
			t.Fatalf("stored capability was mutated through caller-owned input: %#v", got)
		}

		got.Claims["email"] = "changed-through-get"
		got.TokenHash[0] ^= 0xff
		again, err := store.ByID(context.Background(), "cap-create")
		if err != nil {
			t.Fatalf("by id again: %v", err)
		}
		if again.Claims["email"] != "u1@example.test" || bytes.Equal(again.TokenHash, got.TokenHash) {
			t.Fatalf("stored capability was mutated through returned value: %#v", again)
		}
	})

	t.Run("redeem validates purpose expiry revocation and single use", func(t *testing.T) {
		store := newStore(t)
		now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
		tokenHash := capability.HashToken("redeem-token")
		record := capability.Capability{ID: "cap-redeem", Purpose: "invite.accept", ResourceType: "org", ResourceID: "o1", TokenHash: tokenHash, ExpiresAt: now.Add(time.Hour), SingleUse: true, CreatedAt: now}
		if err := store.Create(context.Background(), record); err != nil {
			t.Fatalf("create: %v", err)
		}
		if _, err := store.Redeem(context.Background(), tokenHash, "wrong", now); !errors.Is(err, capability.ErrWrongPurpose) {
			t.Fatalf("wrong purpose err=%v", err)
		}
		redeemed, err := store.Redeem(context.Background(), tokenHash, "invite.accept", now)
		if err != nil {
			t.Fatalf("redeem: %v", err)
		}
		if redeemed.ID != "cap-redeem" || redeemed.UsedAt == nil || !redeemed.UsedAt.Equal(now) {
			t.Fatalf("unexpected redeemed capability: %#v", redeemed)
		}
		if _, err := store.Redeem(context.Background(), tokenHash, "invite.accept", now); !errors.Is(err, capability.ErrUsed) {
			t.Fatalf("second redeem err=%v", err)
		}
	})

	t.Run("expired and revoked capabilities fail closed", func(t *testing.T) {
		store := newStore(t)
		now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
		expiredHash := capability.HashToken("expired-token")
		if err := store.Create(context.Background(), capability.Capability{ID: "cap-expired", Purpose: "p", SubjectID: "u1", TokenHash: expiredHash, ExpiresAt: now.Add(-time.Second), CreatedAt: now.Add(-time.Hour)}); err != nil {
			t.Fatalf("create expired: %v", err)
		}
		if _, err := store.Redeem(context.Background(), expiredHash, "p", now); !errors.Is(err, capability.ErrExpired) {
			t.Fatalf("expired err=%v", err)
		}

		revokedHash := capability.HashToken("revoked-token")
		if err := store.Create(context.Background(), capability.Capability{ID: "cap-revoked", Purpose: "p", SubjectID: "u1", TokenHash: revokedHash, ExpiresAt: now.Add(time.Hour), CreatedAt: now}); err != nil {
			t.Fatalf("create revoked: %v", err)
		}
		if err := store.Revoke(context.Background(), "cap-revoked", now); err != nil {
			t.Fatalf("revoke: %v", err)
		}
		if _, err := store.Redeem(context.Background(), revokedHash, "p", now); !errors.Is(err, capability.ErrRevoked) {
			t.Fatalf("revoked err=%v", err)
		}
		got, err := store.ByID(context.Background(), "cap-revoked")
		if err != nil {
			t.Fatalf("by id revoked: %v", err)
		}
		if got.RevokedAt == nil || !got.RevokedAt.Equal(now) {
			t.Fatalf("expected revoked timestamp: %#v", got)
		}
	})

	t.Run("single use redemption is atomic", func(t *testing.T) {
		store := newStore(t)
		now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
		tokenHash := capability.HashToken("atomic-token")
		if err := store.Create(context.Background(), capability.Capability{ID: "cap-atomic", Purpose: "p", SubjectID: "u1", TokenHash: tokenHash, ExpiresAt: now.Add(time.Hour), SingleUse: true, CreatedAt: now}); err != nil {
			t.Fatalf("create: %v", err)
		}

		const attempts = 16
		var wg sync.WaitGroup
		results := make(chan error, attempts)
		for range attempts {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := store.Redeem(context.Background(), tokenHash, "p", now)
				results <- err
			}()
		}
		wg.Wait()
		close(results)

		successes := 0
		used := 0
		for err := range results {
			switch {
			case err == nil:
				successes++
			case errors.Is(err, capability.ErrUsed):
				used++
			default:
				t.Fatalf("unexpected concurrent redeem err=%v", err)
			}
		}
		if successes != 1 || used != attempts-1 {
			t.Fatalf("expected one success and %d used errors, got successes=%d used=%d", attempts-1, successes, used)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := newStore(t)
		if _, err := store.ByID(context.Background(), "missing"); !errors.Is(err, capability.ErrNotFound) {
			t.Fatalf("by id missing err=%v", err)
		}
		if _, err := store.Redeem(context.Background(), capability.HashToken("missing"), "p", time.Now()); !errors.Is(err, capability.ErrNotFound) {
			t.Fatalf("redeem missing err=%v", err)
		}
		if err := store.Revoke(context.Background(), "missing", time.Now()); !errors.Is(err, capability.ErrNotFound) {
			t.Fatalf("revoke missing err=%v", err)
		}
	})
}
