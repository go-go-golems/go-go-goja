package appauth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func TestResourceResolver(t *testing.T) {
	ctx := context.Background()
	store := seededStore()
	resolver := Resolver{Store: store}
	resource, err := resolver.ResolveResource(ctx, gojahttp.ResourceRequest{Spec: gojahttp.ResourceSpec{Name: "project", Type: "project"}, ID: "p1", TenantID: "o1"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resource.ID != "p1" || resource.TenantID != "o1" || resource.Claims["name"] != "Project One" {
		t.Fatalf("unexpected resource: %#v", resource)
	}
	_, err = resolver.ResolveResource(ctx, gojahttp.ResourceRequest{Spec: gojahttp.ResourceSpec{Name: "project", Type: "project"}, ID: "p1", TenantID: "other"})
	if !errors.Is(err, gojahttp.ErrNotFound) {
		t.Fatalf("tenant mismatch err=%v", err)
	}
	_, err = resolver.ResolveResource(ctx, gojahttp.ResourceRequest{Spec: gojahttp.ResourceSpec{Name: "project", Type: "project"}, ID: "missing", TenantID: "o1"})
	if !errors.Is(err, gojahttp.ErrNotFound) {
		t.Fatalf("missing err=%v", err)
	}
}

func TestAuthorizerAllowsExpectedActions(t *testing.T) {
	ctx := context.Background()
	authorizer := Authorizer{Memberships: seededStore()}
	actor := &gojahttp.Actor{ID: "u1", Kind: "user"}
	project := &gojahttp.ResourceRef{Type: "project", ID: "p1", TenantID: "o1"}
	org := &gojahttp.ResourceRef{Type: "org", ID: "o1"}
	user := &gojahttp.ResourceRef{Type: "user", ID: "u1"}
	for _, req := range []gojahttp.AuthorizationRequest{
		{Actor: actor, Action: ActionUserSelfRead},
		{Actor: actor, Action: ActionUserSelfUpdate},
		{Actor: actor, Action: ActionUserSelfUpdate, Resource: user},
		{Actor: actor, Action: ActionProjectRead, Resource: project},
		{Actor: actor, Action: ActionProjectUpdate, Resource: project},
		{Actor: actor, Action: ActionOrgInvite, Resource: org},
		{Actor: actor, Action: ActionAuditRead, Resource: org},
	} {
		decision, err := authorizer.Authorize(ctx, req)
		if err != nil {
			t.Fatalf("authorize %s: %v", req.Action, err)
		}
		if !decision.Allowed {
			t.Fatalf("expected %s allowed: %#v", req.Action, decision)
		}
	}
}

func TestAuthorizerDeniesNegativeCases(t *testing.T) {
	ctx := context.Background()
	authorizer := Authorizer{Memberships: seededStore()}
	actor := &gojahttp.Actor{ID: "u1", Kind: "user"}
	nonMember := &gojahttp.Actor{ID: "u2", Kind: "user"}
	projectOtherTenant := &gojahttp.ResourceRef{Type: "project", ID: "p2", TenantID: "o2"}
	otherUser := &gojahttp.ResourceRef{Type: "user", ID: "u2"}
	tests := []gojahttp.AuthorizationRequest{
		{Actor: nil, Action: ActionUserSelfRead},
		{Actor: actor, Action: "unknown.action"},
		{Actor: actor, Action: ActionUserSelfUpdate, Resource: otherUser},
		{Actor: nonMember, Action: ActionProjectRead, Resource: projectOtherTenant},
		{Actor: nonMember, Action: ActionProjectUpdate, Resource: projectOtherTenant},
		{Actor: actor, Action: ActionProjectUpdate},
		{Actor: actor, Action: ActionAuditRead, Resource: projectOtherTenant},
	}
	for _, req := range tests {
		decision, err := authorizer.Authorize(ctx, req)
		if err != nil {
			t.Fatalf("authorize %s: %v", req.Action, err)
		}
		if decision.Allowed || decision.Reason == "" {
			t.Fatalf("expected denial with reason for %#v, got %#v", req, decision)
		}
	}
}

func TestUpsertFromOIDCRejectsDisabledExistingUser(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	disabledAt := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	store.AddUser(User{ID: "u-disabled", OIDCIssuer: "https://issuer.example.test", OIDCSubject: "kc-disabled", Email: "old@example.test", DisabledAt: &disabledAt})

	_, err := store.UpsertFromOIDC(ctx, "https://issuer.example.test", "kc-disabled", "new@example.test", true)
	if !errors.Is(err, gojahttp.ErrNotFound) {
		t.Fatalf("expected disabled user upsert to be rejected, got %v", err)
	}
	_, err = store.ByOIDCIdentity(ctx, "https://issuer.example.test", "kc-disabled")
	if !errors.Is(err, gojahttp.ErrNotFound) {
		t.Fatalf("disabled user should remain hidden, got %v", err)
	}
}

func TestUserAndMembershipStore(t *testing.T) {
	ctx := context.Background()
	store := seededStore()
	user, err := store.ByOIDCIdentity(ctx, "https://issuer.example.test", "kc-u1")
	if err != nil {
		t.Fatalf("by sub: %v", err)
	}
	if user.ID != "u1" {
		t.Fatalf("unexpected user: %#v", user)
	}
	created, err := store.UpsertFromOIDC(ctx, "https://issuer.example.test", "kc-new", "new@example.test", true)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if created.ID == "" || !created.EmailVerified {
		t.Fatalf("unexpected created user: %#v", created)
	}
	memberships, err := store.MembershipsForUser(ctx, "u1")
	if err != nil {
		t.Fatalf("memberships: %v", err)
	}
	if len(memberships) != 1 {
		t.Fatalf("expected one active membership, got %d", len(memberships))
	}
	ok, err := store.HasRole(ctx, "u1", "o1", "admin")
	if err != nil || !ok {
		t.Fatalf("expected admin role ok=%v err=%v", ok, err)
	}
}

func TestOIDCIdentityScopesSubjectByIssuer(t *testing.T) {
	store := NewMemoryStore()
	first, err := store.UpsertFromOIDC(context.Background(), "https://issuer-a.example.test", "shared-subject", "a@example.test", true)
	if err != nil {
		t.Fatalf("first issuer: %v", err)
	}
	second, err := store.UpsertFromOIDC(context.Background(), "https://issuer-b.example.test", "shared-subject", "b@example.test", true)
	if err != nil {
		t.Fatalf("second issuer: %v", err)
	}
	if first.ID == second.ID {
		t.Fatalf("issuer-scoped identities collided: %#v %#v", first, second)
	}
	if first.OIDCIssuer == second.OIDCIssuer {
		t.Fatalf("issuers were not preserved: %#v %#v", first, second)
	}
	if _, err := store.ByOIDCIdentity(context.Background(), "https://issuer-a.example.test", "shared-subject"); err != nil {
		t.Fatalf("lookup first identity: %v", err)
	}
}

func TestOIDCUserIDRejectsIncompleteOrAmbiguousIdentity(t *testing.T) {
	for _, identity := range [][2]string{{"", "subject"}, {"https://issuer.example.test", ""}, {"https://issuer.example.test", "bad\x00subject"}} {
		if _, err := OIDCUserID(identity[0], identity[1]); err == nil {
			t.Fatalf("OIDCUserID(%q, %q) succeeded", identity[0], identity[1])
		}
	}
}

func TestAuthorizerPropagatesBackendErrors(t *testing.T) {
	decision, err := (Authorizer{Memberships: failingMembershipStore{}}).Authorize(context.Background(), gojahttp.AuthorizationRequest{Actor: &gojahttp.Actor{ID: "u1"}, Action: ActionProjectRead, Resource: &gojahttp.ResourceRef{Type: "project", TenantID: "o1"}})
	if err == nil {
		t.Fatalf("expected backend error")
	}
	if decision.Allowed || decision.Reason == "" {
		t.Fatalf("expected denied decision on backend error: %#v", decision)
	}
}

func seededStore() *MemoryStore {
	store := NewMemoryStore()
	store.AddUser(User{ID: "u1", OIDCIssuer: "https://issuer.example.test", OIDCSubject: "kc-u1", Email: "u1@example.test", EmailVerified: true})
	store.AddUser(User{ID: "u2", OIDCIssuer: "https://issuer.example.test", OIDCSubject: "kc-u2", Email: "u2@example.test"})
	store.AddMembership(Membership{UserID: "u1", TenantID: "o1", Role: "admin"})
	store.AddResource(Resource{Type: "project", ID: "p1", TenantID: "o1", Claims: map[string]any{"name": "Project One"}})
	store.AddResource(Resource{Type: "project", ID: "p2", TenantID: "o2"})
	store.AddResource(Resource{Type: "org", ID: "o1"})
	store.AddResource(Resource{Type: "user", ID: "u1"})
	store.AddResource(Resource{Type: "user", ID: "u2"})
	return store
}

type failingMembershipStore struct{}

func (failingMembershipStore) MembershipsForUser(context.Context, string) ([]Membership, error) {
	return nil, errors.New("backend unavailable")
}
func (failingMembershipStore) IsMember(context.Context, string, string) (bool, error) {
	return false, errors.New("backend unavailable")
}
func (failingMembershipStore) HasRole(context.Context, string, string, ...string) (bool, error) {
	return false, errors.New("backend unavailable")
}
