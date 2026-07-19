// Package appauthtest provides reusable conformance tests for appauth store
// implementations.
package appauthtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
)

// Harness exposes the appauth store interfaces plus seeding hooks required by
// the contract. SQL stores can implement the hooks with inserts or fixtures.
type Harness struct {
	Users       appauth.UserStore
	Memberships appauth.MembershipStore
	Resources   appauth.ResourceStore
	AddUser     func(appauth.User)
	AddMember   func(appauth.Membership)
	AddResource func(appauth.Resource)
}

// NewHarness constructs an empty store harness for a single contract test.
type NewHarness func(testing.TB) Harness

// RunStoreContract verifies user, membership, and resource semantics expected
// by Resolver, Authorizer, and Keycloak user normalization.
func RunStoreContract(t *testing.T, newHarness NewHarness) {
	t.Helper()

	t.Run("users by id sub and oidc upsert", func(t *testing.T) {
		h := requireHarness(t, newHarness(t))
		ctx := context.Background()
		disabledAt := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
		h.AddUser(appauth.User{ID: "u1", OIDCIssuer: "https://issuer.example.test", OIDCSubject: "kc-u1", Email: "old@example.test", EmailVerified: false})
		h.AddUser(appauth.User{ID: "disabled", OIDCIssuer: "https://issuer.example.test", OIDCSubject: "kc-disabled", Email: "disabled@example.test", DisabledAt: &disabledAt})

		byID, err := h.Users.ByID(ctx, "u1")
		if err != nil {
			t.Fatalf("by id: %v", err)
		}
		if byID.OIDCSubject != "kc-u1" {
			t.Fatalf("unexpected user by id: %#v", byID)
		}
		bySub, err := h.Users.ByExternalIdentity(ctx, "https://issuer.example.test", "kc-u1")
		if err != nil {
			t.Fatalf("by sub: %v", err)
		}
		if bySub.ID != "u1" {
			t.Fatalf("unexpected user by sub: %#v", bySub)
		}
		if _, err := h.Users.ByID(ctx, "disabled"); !errors.Is(err, gojahttp.ErrNotFound) {
			t.Fatalf("disabled user err=%v", err)
		}

		updated, err := h.Users.UpsertFromOIDC(ctx, "https://issuer.example.test", "kc-u1", "new@example.test", true)
		if err != nil {
			t.Fatalf("upsert existing: %v", err)
		}
		if updated.ID != "u1" || updated.Email != "new@example.test" || !updated.EmailVerified {
			t.Fatalf("unexpected updated user: %#v", updated)
		}
		created, err := h.Users.UpsertFromOIDC(ctx, "https://issuer.example.test", "kc-new", "created@example.test", true)
		if err != nil {
			t.Fatalf("upsert new: %v", err)
		}
		if created.ID == "" || created.OIDCSubject != "kc-new" || created.Email != "created@example.test" {
			t.Fatalf("unexpected created user: %#v", created)
		}
	})

	t.Run("memberships filter revoked and check roles", func(t *testing.T) {
		h := requireHarness(t, newHarness(t))
		ctx := context.Background()
		revokedAt := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
		h.AddMember(appauth.Membership{UserID: "u1", TenantID: "o1", Role: "admin"})
		h.AddMember(appauth.Membership{UserID: "u1", TenantID: "o2", Role: "viewer", RevokedAt: &revokedAt})

		memberships, err := h.Memberships.MembershipsForUser(ctx, "u1")
		if err != nil {
			t.Fatalf("memberships: %v", err)
		}
		if len(memberships) != 1 || memberships[0].TenantID != "o1" {
			t.Fatalf("expected only active membership, got %#v", memberships)
		}
		ok, err := h.Memberships.IsMember(ctx, "u1", "o1")
		if err != nil || !ok {
			t.Fatalf("expected active membership ok=%v err=%v", ok, err)
		}
		ok, err = h.Memberships.IsMember(ctx, "u1", "o2")
		if err != nil || ok {
			t.Fatalf("expected revoked membership denied ok=%v err=%v", ok, err)
		}
		ok, err = h.Memberships.HasRole(ctx, "u1", "o1", "editor", "admin")
		if err != nil || !ok {
			t.Fatalf("expected admin role ok=%v err=%v", ok, err)
		}
		ok, err = h.Memberships.HasRole(ctx, "u1", "o1", "viewer")
		if err != nil || ok {
			t.Fatalf("expected viewer role denied ok=%v err=%v", ok, err)
		}
	})

	t.Run("resources by type id tenant and clone isolation", func(t *testing.T) {
		h := requireHarness(t, newHarness(t))
		ctx := context.Background()
		resource := appauth.Resource{Type: "project", ID: "p1", TenantID: "o1", OwnerID: "u1", Claims: map[string]any{"name": "Project One"}}
		h.AddResource(resource)
		resource.Claims["name"] = "mutated"

		got, err := h.Resources.GetResource(ctx, "project", "p1")
		if err != nil {
			t.Fatalf("get resource: %v", err)
		}
		if got.TenantID != "o1" || got.Claims["name"] != "Project One" {
			t.Fatalf("resource mutated through caller-owned input: %#v", got)
		}
		got.Claims["name"] = "changed-through-get"
		again, err := h.Resources.GetResource(ctx, "project", "p1")
		if err != nil {
			t.Fatalf("get resource again: %v", err)
		}
		if again.Claims["name"] != "Project One" {
			t.Fatalf("resource mutated through returned value: %#v", again)
		}
		if _, err := h.Resources.GetResource(ctx, "project", "missing"); !errors.Is(err, gojahttp.ErrNotFound) {
			t.Fatalf("missing resource err=%v", err)
		}
	})
}

func requireHarness(t *testing.T, h Harness) Harness {
	t.Helper()
	if h.Users == nil || h.Memberships == nil || h.Resources == nil || h.AddUser == nil || h.AddMember == nil || h.AddResource == nil {
		t.Fatalf("incomplete appauth store harness: %#v", h)
	}
	return h
}
