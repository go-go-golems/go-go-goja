// Package capability provides narrow bearer-capability token helpers for flows
// such as organization invites, email verification, password reset, temporary
// download links, and scoped API tokens.
package capability

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

var (
	ErrNotFound     = errors.New("capability not found")
	ErrWrongPurpose = errors.New("capability purpose mismatch")
	ErrExpired      = errors.New("capability expired")
	ErrRevoked      = errors.New("capability revoked")
	ErrUsed         = errors.New("capability already used")
)

// Capability is the stored, hashed-token representation. Raw tokens are
// returned once from Issue and must never be stored or logged.
type Capability struct {
	ID           string
	Purpose      string
	SubjectID    string
	ResourceType string
	ResourceID   string
	Claims       map[string]string
	TokenHash    []byte
	ExpiresAt    time.Time
	SingleUse    bool
	UsedAt       *time.Time
	RevokedAt    *time.Time
	CreatedBy    string
	CreatedAt    time.Time
}

// IssueSpec describes a capability to issue.
type IssueSpec struct {
	Purpose      string
	SubjectID    string
	ResourceType string
	ResourceID   string
	Claims       map[string]string
	ExpiresAt    time.Time
	TTL          time.Duration
	SingleUse    bool
	CreatedBy    string
}

// IssueResult returns the stored capability plus the raw token exactly once.
type IssueResult struct {
	Capability Capability
	Token      string
}

// Store persists capabilities by token hash.
type Store interface {
	Create(ctx context.Context, capability Capability) error
	Redeem(ctx context.Context, tokenHash []byte, purpose string, now time.Time) (*Capability, error)
	Revoke(ctx context.Context, id string, now time.Time) error
	ByID(ctx context.Context, id string) (*Capability, error)
}

// Service issues, redeems, and revokes capability tokens.
type Service struct {
	Store Store
	Audit gojahttp.AuditSink
	Now   func() time.Time
}

func (s Service) Issue(ctx context.Context, spec IssueSpec) (IssueResult, error) {
	if s.Store == nil {
		return IssueResult{}, fmt.Errorf("capability: store is required")
	}
	now := s.now()
	if spec.Purpose == "" {
		return IssueResult{}, fmt.Errorf("capability: purpose is required")
	}
	if spec.SubjectID == "" && (spec.ResourceType == "" || spec.ResourceID == "") {
		return IssueResult{}, fmt.Errorf("capability: subject or resource is required")
	}
	if spec.ExpiresAt.IsZero() {
		if spec.TTL <= 0 {
			return IssueResult{}, fmt.Errorf("capability: expiry or TTL is required")
		}
		spec.ExpiresAt = now.Add(spec.TTL)
	}
	id, err := randomToken()
	if err != nil {
		return IssueResult{}, err
	}
	token, err := randomToken()
	if err != nil {
		return IssueResult{}, err
	}
	capability := Capability{ID: id, Purpose: spec.Purpose, SubjectID: spec.SubjectID, ResourceType: spec.ResourceType, ResourceID: spec.ResourceID, Claims: cloneStringMap(spec.Claims), TokenHash: HashToken(token), ExpiresAt: spec.ExpiresAt, SingleUse: spec.SingleUse, CreatedBy: spec.CreatedBy, CreatedAt: now}
	if err := s.Store.Create(ctx, capability); err != nil {
		return IssueResult{}, err
	}
	s.record(ctx, "capability.issued", "completed", capability, nil)
	return IssueResult{Capability: redactCapability(capability), Token: token}, nil
}

func (s Service) Redeem(ctx context.Context, purpose, token string) (*Capability, error) {
	if s.Store == nil {
		return nil, fmt.Errorf("capability: store is required")
	}
	capability, err := s.Store.Redeem(ctx, HashToken(token), purpose, s.now())
	if err != nil {
		s.record(ctx, "capability.redeemed", "denied", Capability{Purpose: purpose}, err)
		return nil, err
	}
	s.record(ctx, "capability.redeemed", "completed", *capability, nil)
	redacted := redactCapability(*capability)
	return &redacted, nil
}

func (s Service) Revoke(ctx context.Context, id string) error {
	if s.Store == nil {
		return fmt.Errorf("capability: store is required")
	}
	err := s.Store.Revoke(ctx, id, s.now())
	capability := Capability{ID: id}
	if loaded, loadErr := s.Store.ByID(ctx, id); loadErr == nil && loaded != nil {
		capability = *loaded
	}
	outcome := "completed"
	if err != nil {
		outcome = "denied"
	}
	s.record(ctx, "capability.revoked", outcome, capability, err)
	return err
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s Service) record(ctx context.Context, event, outcome string, capability Capability, err error) {
	if s.Audit == nil {
		return
	}
	reason := ""
	if err != nil {
		reason = err.Error()
	}
	_ = s.Audit.RecordAudit(ctx, gojahttp.AuditEvent{Event: event, Outcome: outcome, Reason: reason, Resource: &gojahttp.ResourceRef{Type: capability.ResourceType, ID: capability.ResourceID}, Attributes: map[string]any{"capabilityId": capability.ID, "purpose": capability.Purpose, "subjectId": capability.SubjectID, "singleUse": capability.SingleUse}})
}

// HashToken returns the storage hash for a raw token.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	out := make([]byte, len(sum))
	copy(out, sum[:])
	return out
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate capability token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// MemoryStore is an in-memory Store for tests and simple demos.
type MemoryStore struct {
	mu     sync.Mutex
	byID   map[string]Capability
	byHash map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byID: map[string]Capability{}, byHash: map[string]string{}}
}

func (s *MemoryStore) Create(_ context.Context, capability Capability) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byID == nil {
		s.byID = map[string]Capability{}
	}
	if s.byHash == nil {
		s.byHash = map[string]string{}
	}
	if capability.ID == "" || len(capability.TokenHash) == 0 {
		return fmt.Errorf("capability id and token hash are required")
	}
	clone := cloneCapability(capability)
	s.byID[clone.ID] = clone
	s.byHash[hashKey(clone.TokenHash)] = clone.ID
	return nil
}

func (s *MemoryStore) Redeem(_ context.Context, tokenHash []byte, purpose string, now time.Time) (*Capability, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.byHash[hashKey(tokenHash)]
	if !ok {
		return nil, ErrNotFound
	}
	capability := s.byID[id]
	if subtle.ConstantTimeCompare(capability.TokenHash, tokenHash) != 1 {
		return nil, ErrNotFound
	}
	if capability.Purpose != purpose {
		return nil, ErrWrongPurpose
	}
	if capability.RevokedAt != nil {
		return nil, ErrRevoked
	}
	if !capability.ExpiresAt.IsZero() && now.After(capability.ExpiresAt) {
		return nil, ErrExpired
	}
	if capability.SingleUse && capability.UsedAt != nil {
		return nil, ErrUsed
	}
	if capability.SingleUse {
		usedAt := now
		capability.UsedAt = &usedAt
		s.byID[id] = capability
	}
	clone := redactCapability(capability)
	return &clone, nil
}

func (s *MemoryStore) Revoke(_ context.Context, id string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	capability, ok := s.byID[id]
	if !ok {
		return ErrNotFound
	}
	revokedAt := now
	capability.RevokedAt = &revokedAt
	s.byID[id] = capability
	return nil
}

func (s *MemoryStore) ByID(_ context.Context, id string) (*Capability, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	capability, ok := s.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	clone := cloneCapability(capability)
	return &clone, nil
}

func hashKey(hash []byte) string { return base64.RawURLEncoding.EncodeToString(hash) }

func cloneCapability(in Capability) Capability {
	out := in
	out.TokenHash = append([]byte(nil), in.TokenHash...)
	out.Claims = cloneStringMap(in.Claims)
	return out
}

func redactCapability(in Capability) Capability {
	out := cloneCapability(in)
	out.TokenHash = nil
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := map[string]string{}
	for key, value := range in {
		out[key] = value
	}
	return out
}
