// Package appauth provides small app-owned user, tenant, membership,
// resource, and authorization helpers for gojahttp planned routes. It is not a
// policy engine; it is a boring explicit-Go starting point that denies by
// default and can later be replaced by application-specific logic.
package appauth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

const (
	ActionUserSelfRead   = "user.self.read"
	ActionUserSelfUpdate = "user.self.update"
	ActionProjectRead    = "project.read"
	ActionProjectUpdate  = "project.update"
	ActionOrgInvite      = "org.member.invite"
)

// User is the minimal app-owned user model used by helpers.
type User struct {
	ID            string
	KeycloakSub   string
	Email         string
	DisplayName   string
	EmailVerified bool
	DisabledAt    *time.Time
}

// Tenant is the minimal app-owned tenant model used by helpers.
type Tenant struct {
	ID         string
	Slug       string
	Name       string
	DisabledAt *time.Time
}

// Membership binds a user to a tenant role.
type Membership struct {
	UserID    string
	TenantID  string
	Role      string
	RevokedAt *time.Time
}

// Resource is the app-owned resource handle projected into gojahttp.ResourceRef.
type Resource struct {
	Name     string
	Type     string
	ID       string
	TenantID string
	OwnerID  string
	Claims   map[string]any
}

// UserStore loads app users.
type UserStore interface {
	ByID(ctx context.Context, id string) (*User, error)
	ByKeycloakSub(ctx context.Context, sub string) (*User, error)
	UpsertFromOIDC(ctx context.Context, sub, email string, emailVerified bool) (*User, error)
}

// MembershipStore answers tenant membership/role questions.
type MembershipStore interface {
	MembershipsForUser(ctx context.Context, userID string) ([]Membership, error)
	IsMember(ctx context.Context, userID, tenantID string) (bool, error)
	HasRole(ctx context.Context, userID, tenantID string, roles ...string) (bool, error)
}

// ResourceStore loads resources by type and ID.
type ResourceStore interface {
	GetResource(ctx context.Context, typ, id string) (*Resource, error)
}

// Resolver implements gojahttp.ResourceResolver using a ResourceStore.
type Resolver struct{ Store ResourceStore }

func (r Resolver) ResolveResource(ctx context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
	if r.Store == nil {
		return nil, fmt.Errorf("appauth: resource store is required")
	}
	resource, err := r.Store.GetResource(ctx, req.Spec.Type, req.ID)
	if err != nil {
		return nil, err
	}
	if resource == nil {
		return nil, gojahttp.ErrNotFound
	}
	if req.TenantID != "" && resource.TenantID != req.TenantID {
		return nil, gojahttp.ErrNotFound
	}
	name := req.Spec.Name
	if name == "" {
		name = resource.Name
	}
	if name == "" {
		name = resource.Type
	}
	return &gojahttp.ResourceRef{Name: name, Type: resource.Type, ID: resource.ID, TenantID: resource.TenantID, Claims: cloneClaims(resource.Claims)}, nil
}

// Authorizer implements a small deny-by-default action switch suitable as a
// starting point for monolith apps and demos.
type Authorizer struct{ Memberships MembershipStore }

func (a Authorizer) Authorize(ctx context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
	if req.Actor == nil {
		return deny("missing actor"), nil
	}
	switch req.Action {
	case ActionUserSelfRead:
		return allow(), nil
	case ActionUserSelfUpdate:
		if req.Resource == nil || req.Resource.Type != "user" {
			return deny("missing user resource"), nil
		}
		if req.Resource.ID == req.Actor.ID {
			return allow(), nil
		}
		return deny("cannot update another user"), nil
	case ActionProjectRead:
		if req.Resource == nil || req.Resource.Type != "project" {
			return deny("missing project resource"), nil
		}
		return a.memberDecision(ctx, req.Actor.ID, req.Resource.TenantID)
	case ActionProjectUpdate:
		if req.Resource == nil || req.Resource.Type != "project" {
			return deny("missing project resource"), nil
		}
		return a.roleDecision(ctx, req.Actor.ID, req.Resource.TenantID, "admin", "editor")
	case ActionOrgInvite:
		if req.Resource == nil {
			return deny("missing organization resource"), nil
		}
		tenantID := req.Resource.TenantID
		if req.Resource.Type == "org" && tenantID == "" {
			tenantID = req.Resource.ID
		}
		return a.roleDecision(ctx, req.Actor.ID, tenantID, "admin")
	default:
		return deny("unknown action"), nil
	}
}

func (a Authorizer) memberDecision(ctx context.Context, userID, tenantID string) (gojahttp.AuthorizationDecision, error) {
	if a.Memberships == nil {
		return deny("membership store is required"), nil
	}
	ok, err := a.Memberships.IsMember(ctx, userID, tenantID)
	if err != nil {
		return deny("membership lookup failed"), err
	}
	if ok {
		return allow(), nil
	}
	return deny("tenant membership required"), nil
}

func (a Authorizer) roleDecision(ctx context.Context, userID, tenantID string, roles ...string) (gojahttp.AuthorizationDecision, error) {
	if a.Memberships == nil {
		return deny("membership store is required"), nil
	}
	ok, err := a.Memberships.HasRole(ctx, userID, tenantID, roles...)
	if err != nil {
		return deny("role lookup failed"), err
	}
	if ok {
		return allow(), nil
	}
	return deny("required tenant role missing"), nil
}

func allow() gojahttp.AuthorizationDecision { return gojahttp.AuthorizationDecision{Allowed: true} }
func deny(reason string) gojahttp.AuthorizationDecision {
	return gojahttp.AuthorizationDecision{Allowed: false, Reason: reason}
}

// MemoryStore is a small in-memory store for tests and examples.
type MemoryStore struct {
	mu          sync.Mutex
	users       map[string]User
	usersBySub  map[string]string
	memberships []Membership
	resources   map[string]Resource
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{users: map[string]User{}, usersBySub: map[string]string{}, resources: map[string]Resource{}}
}

func (s *MemoryStore) AddUser(user User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user = cloneUser(user)
	s.users[user.ID] = user
	if user.KeycloakSub != "" {
		s.usersBySub[user.KeycloakSub] = user.ID
	}
}

func (s *MemoryStore) AddMembership(membership Membership) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memberships = append(s.memberships, cloneMembership(membership))
}

func (s *MemoryStore) AddResource(resource Resource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resource = cloneResource(resource)
	s.resources[resourceKey(resource.Type, resource.ID)] = resource
}

func (s *MemoryStore) ByID(_ context.Context, id string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok || user.DisabledAt != nil {
		return nil, gojahttp.ErrNotFound
	}
	user = cloneUser(user)
	return &user, nil
}

func (s *MemoryStore) ByKeycloakSub(ctx context.Context, sub string) (*User, error) {
	s.mu.Lock()
	id, ok := s.usersBySub[sub]
	s.mu.Unlock()
	if !ok {
		return nil, gojahttp.ErrNotFound
	}
	return s.ByID(ctx, id)
}

func (s *MemoryStore) UpsertFromOIDC(_ context.Context, sub, email string, emailVerified bool) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.usersBySub[sub]; ok {
		user := s.users[id]
		if user.DisabledAt != nil {
			return nil, gojahttp.ErrNotFound
		}
		user.Email = email
		user.EmailVerified = emailVerified
		s.users[id] = user
		user = cloneUser(user)
		return &user, nil
	}
	id := "user:" + sub
	user := User{ID: id, KeycloakSub: sub, Email: email, EmailVerified: emailVerified}
	s.users[id] = user
	s.usersBySub[sub] = id
	return &user, nil
}

func (s *MemoryStore) MembershipsForUser(_ context.Context, userID string) ([]Membership, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Membership{}
	for _, membership := range s.memberships {
		if membership.UserID == userID && membership.RevokedAt == nil {
			out = append(out, cloneMembership(membership))
		}
	}
	return out, nil
}

func (s *MemoryStore) IsMember(ctx context.Context, userID, tenantID string) (bool, error) {
	memberships, err := s.MembershipsForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, membership := range memberships {
		if membership.TenantID == tenantID {
			return true, nil
		}
	}
	return false, nil
}

func (s *MemoryStore) HasRole(ctx context.Context, userID, tenantID string, roles ...string) (bool, error) {
	memberships, err := s.MembershipsForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, membership := range memberships {
		if membership.TenantID != tenantID {
			continue
		}
		for _, role := range roles {
			if membership.Role == role {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *MemoryStore) GetResource(_ context.Context, typ, id string) (*Resource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resource, ok := s.resources[resourceKey(typ, id)]
	if !ok {
		return nil, gojahttp.ErrNotFound
	}
	resource = cloneResource(resource)
	return &resource, nil
}

func resourceKey(typ, id string) string { return typ + ":" + id }

func cloneUser(in User) User {
	out := in
	if in.DisabledAt != nil {
		disabledAt := *in.DisabledAt
		out.DisabledAt = &disabledAt
	}
	return out
}

func cloneMembership(in Membership) Membership {
	out := in
	if in.RevokedAt != nil {
		revokedAt := *in.RevokedAt
		out.RevokedAt = &revokedAt
	}
	return out
}

func cloneResource(in Resource) Resource {
	out := in
	out.Claims = cloneClaims(in.Claims)
	return out
}

func cloneClaims(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := map[string]any{}
	for key, value := range in {
		out[key] = value
	}
	return out
}
