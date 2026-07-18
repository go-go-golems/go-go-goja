package programauth

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryAPITokenStore is a concurrency-safe development/test API-token store.
type MemoryAPITokenStore struct {
	mu     sync.Mutex
	tokens map[string]APIToken
}

func NewMemoryAPITokenStore() *MemoryAPITokenStore {
	return &MemoryAPITokenStore{tokens: map[string]APIToken{}}
}

func (s *MemoryAPITokenStore) CreateAPIToken(_ context.Context, token APIToken) (APIToken, error) {
	if s == nil {
		return APIToken{}, fmt.Errorf("programauth memory api token store is nil")
	}
	token = cloneAPIToken(token)
	if token.ID == "" {
		return APIToken{}, fmt.Errorf("api token id is required")
	}
	if token.TokenPrefix == "" || len(token.TokenHash) == 0 {
		return APIToken{}, fmt.Errorf("api token hash and prefix are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tokens == nil {
		s.tokens = map[string]APIToken{}
	}
	if _, exists := s.tokens[token.ID]; exists {
		return APIToken{}, fmt.Errorf("api token %q already exists", token.ID)
	}
	s.tokens[token.ID] = token
	return cloneAPIToken(token), nil
}

func (s *MemoryAPITokenStore) GetAPITokenByID(_ context.Context, id string) (APIToken, error) {
	if s == nil {
		return APIToken{}, fmt.Errorf("programauth memory api token store is nil")
	}
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.tokens[id]
	if !ok {
		return APIToken{}, ErrAPITokenNotFound
	}
	return cloneAPIToken(token), nil
}

func (s *MemoryAPITokenStore) FindAPITokenByPrefix(_ context.Context, prefix string) ([]APIToken, error) {
	if s == nil {
		return nil, fmt.Errorf("programauth memory api token store is nil")
	}
	prefix = strings.TrimSpace(prefix)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []APIToken{}
	for _, token := range s.tokens {
		if token.TokenPrefix == prefix {
			out = append(out, cloneAPIToken(token))
		}
	}
	return out, nil
}

func (s *MemoryAPITokenStore) ListAPITokens(_ context.Context, query APITokenQuery) ([]APIToken, error) {
	if s == nil {
		return nil, fmt.Errorf("programauth memory api token store is nil")
	}
	query.AgentID = strings.TrimSpace(query.AgentID)
	query.SubjectUserID = strings.TrimSpace(query.SubjectUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]APIToken, 0, len(s.tokens))
	for _, token := range s.tokens {
		if query.AgentID != "" && token.AgentID != query.AgentID {
			continue
		}
		if query.SubjectUserID != "" && token.SubjectUserID != query.SubjectUserID {
			continue
		}
		if !query.IncludeRevoked && token.Revoked() {
			continue
		}
		out = append(out, cloneAPIToken(token))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (s *MemoryAPITokenStore) RevokeAPIToken(_ context.Context, id string, revokedAt time.Time) (APIToken, error) {
	if s == nil {
		return APIToken{}, fmt.Errorf("programauth memory api token store is nil")
	}
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.tokens[id]
	if !ok {
		return APIToken{}, ErrAPITokenNotFound
	}
	revokedAt = revokedAt.UTC()
	token.RevokedAt = &revokedAt
	token.UpdatedAt = revokedAt
	s.tokens[id] = token
	return cloneAPIToken(token), nil
}

func (s *MemoryAPITokenStore) TouchAPIToken(_ context.Context, id string, usedAt time.Time) error {
	if s == nil {
		return fmt.Errorf("programauth memory api token store is nil")
	}
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.tokens[id]
	if !ok {
		return ErrAPITokenNotFound
	}
	usedAt = usedAt.UTC()
	token.LastUsedAt = &usedAt
	token.UpdatedAt = usedAt
	s.tokens[id] = token
	return nil
}
