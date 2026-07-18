package programauth

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type MemoryAccessTokenStore struct {
	mu     sync.Mutex
	tokens map[string]AccessToken
}

func NewMemoryAccessTokenStore() *MemoryAccessTokenStore {
	return &MemoryAccessTokenStore{tokens: map[string]AccessToken{}}
}

func (s *MemoryAccessTokenStore) CreateAccessToken(_ context.Context, token AccessToken) (AccessToken, error) {
	if s == nil {
		return AccessToken{}, fmt.Errorf("programauth memory access token store is nil")
	}
	token = cloneAccessToken(token)
	if token.ID == "" {
		return AccessToken{}, fmt.Errorf("access token id is required")
	}
	if token.TokenPrefix == "" || len(token.TokenHash) == 0 {
		return AccessToken{}, fmt.Errorf("access token hash and prefix are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tokens == nil {
		s.tokens = map[string]AccessToken{}
	}
	if _, exists := s.tokens[token.ID]; exists {
		return AccessToken{}, fmt.Errorf("access token %q already exists", token.ID)
	}
	s.tokens[token.ID] = token
	return cloneAccessToken(token), nil
}

func (s *MemoryAccessTokenStore) DeleteAccessToken(_ context.Context, id string) error {
	if s == nil {
		return fmt.Errorf("programauth memory access token store is nil")
	}
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tokens[id]; !ok {
		return ErrAccessTokenNotFound
	}
	delete(s.tokens, id)
	return nil
}

func (s *MemoryAccessTokenStore) FindAccessTokenByPrefix(_ context.Context, prefix string) ([]AccessToken, error) {
	if s == nil {
		return nil, fmt.Errorf("programauth memory access token store is nil")
	}
	prefix = strings.TrimSpace(prefix)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []AccessToken{}
	for _, token := range s.tokens {
		if token.TokenPrefix == prefix {
			out = append(out, cloneAccessToken(token))
		}
	}
	return out, nil
}

func (s *MemoryAccessTokenStore) TouchAccessToken(_ context.Context, id string, usedAt time.Time) error {
	if s == nil {
		return fmt.Errorf("programauth memory access token store is nil")
	}
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.tokens[id]
	if !ok {
		return ErrAccessTokenNotFound
	}
	usedAt = usedAt.UTC()
	token.LastUsedAt = &usedAt
	token.UpdatedAt = usedAt
	s.tokens[id] = token
	return nil
}

type MemoryRefreshTokenStore struct {
	mu     sync.Mutex
	tokens map[string]RefreshToken
}

func NewMemoryRefreshTokenStore() *MemoryRefreshTokenStore {
	return &MemoryRefreshTokenStore{tokens: map[string]RefreshToken{}}
}

func (s *MemoryRefreshTokenStore) CreateRefreshToken(_ context.Context, token RefreshToken) (RefreshToken, error) {
	if s == nil {
		return RefreshToken{}, fmt.Errorf("programauth memory refresh token store is nil")
	}
	token = cloneRefreshToken(token)
	if token.ID == "" {
		return RefreshToken{}, fmt.Errorf("refresh token id is required")
	}
	if token.FamilyID == "" {
		return RefreshToken{}, fmt.Errorf("refresh token family id is required")
	}
	if token.TokenPrefix == "" || len(token.TokenHash) == 0 {
		return RefreshToken{}, fmt.Errorf("refresh token hash and prefix are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tokens == nil {
		s.tokens = map[string]RefreshToken{}
	}
	if _, exists := s.tokens[token.ID]; exists {
		return RefreshToken{}, fmt.Errorf("refresh token %q already exists", token.ID)
	}
	s.tokens[token.ID] = token
	return cloneRefreshToken(token), nil
}

func (s *MemoryRefreshTokenStore) ListRefreshTokens(_ context.Context, query RefreshTokenQuery) ([]RefreshToken, error) {
	if s == nil {
		return nil, fmt.Errorf("programauth memory refresh token store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []RefreshToken{}
	for _, token := range s.tokens {
		if (query.SubjectUserID == "" || token.SubjectUserID == query.SubjectUserID) && (query.FamilyID == "" || token.FamilyID == query.FamilyID) {
			out = append(out, cloneRefreshToken(token))
		}
	}
	return out, nil
}

func (s *MemoryRefreshTokenStore) FindRefreshTokenByPrefix(_ context.Context, prefix string) ([]RefreshToken, error) {
	if s == nil {
		return nil, fmt.Errorf("programauth memory refresh token store is nil")
	}
	prefix = strings.TrimSpace(prefix)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []RefreshToken{}
	for _, token := range s.tokens {
		if token.TokenPrefix == prefix {
			out = append(out, cloneRefreshToken(token))
		}
	}
	return out, nil
}

func (s *MemoryRefreshTokenStore) RotateRefreshToken(_ context.Context, currentID string, next RefreshToken, usedAt time.Time) (RefreshToken, RefreshToken, error) {
	if s == nil {
		return RefreshToken{}, RefreshToken{}, fmt.Errorf("programauth memory refresh token store is nil")
	}
	currentID = strings.TrimSpace(currentID)
	next = cloneRefreshToken(next)
	if next.ID == "" {
		return RefreshToken{}, RefreshToken{}, fmt.Errorf("replacement refresh token id is required")
	}
	if next.TokenPrefix == "" || len(next.TokenHash) == 0 {
		return RefreshToken{}, RefreshToken{}, fmt.Errorf("replacement refresh token hash and prefix are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tokens == nil {
		s.tokens = map[string]RefreshToken{}
	}
	current, ok := s.tokens[currentID]
	if !ok {
		return RefreshToken{}, RefreshToken{}, ErrRefreshTokenNotFound
	}
	if current.Revoked() {
		return RefreshToken{}, RefreshToken{}, ErrRefreshTokenRevoked
	}
	if current.Used() {
		return RefreshToken{}, RefreshToken{}, ErrRefreshTokenUsed
	}
	if _, exists := s.tokens[next.ID]; exists {
		return RefreshToken{}, RefreshToken{}, fmt.Errorf("replacement refresh token %q already exists", next.ID)
	}
	usedAt = usedAt.UTC()
	current.UsedAt = &usedAt
	current.ReplacedByID = next.ID
	current.UpdatedAt = usedAt
	next.FamilyID = current.FamilyID
	s.tokens[current.ID] = current
	s.tokens[next.ID] = next
	return cloneRefreshToken(current), cloneRefreshToken(next), nil
}

func (s *MemoryRefreshTokenStore) RevokeRefreshTokenFamily(_ context.Context, familyID string, revokedAt time.Time) error {
	if s == nil {
		return fmt.Errorf("programauth memory refresh token store is nil")
	}
	familyID = strings.TrimSpace(familyID)
	if familyID == "" {
		return fmt.Errorf("refresh token family id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	revokedAt = revokedAt.UTC()
	for id, token := range s.tokens {
		if token.FamilyID != familyID || token.Revoked() {
			continue
		}
		token.RevokedAt = &revokedAt
		token.UpdatedAt = revokedAt
		s.tokens[id] = token
	}
	return nil
}
