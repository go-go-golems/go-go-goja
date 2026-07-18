package programauth

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryAgentStore is a concurrency-safe in-memory agent store for tests,
// examples, and local generated hosts. Production hosts should use a durable
// store with the same AgentStore contract.
type MemoryAgentStore struct {
	mu     sync.Mutex
	agents map[string]Agent
}

func NewMemoryAgentStore() *MemoryAgentStore {
	return &MemoryAgentStore{agents: map[string]Agent{}}
}

func (s *MemoryAgentStore) CreateAgent(_ context.Context, agent Agent) (Agent, error) {
	if s == nil {
		return Agent{}, fmt.Errorf("programauth memory agent store is nil")
	}
	agent = cloneAgent(agent)
	if agent.ID == "" {
		return Agent{}, fmt.Errorf("agent id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.agents == nil {
		s.agents = map[string]Agent{}
	}
	if _, exists := s.agents[agent.ID]; exists {
		return Agent{}, fmt.Errorf("agent %q already exists", agent.ID)
	}
	s.agents[agent.ID] = agent
	return cloneAgent(agent), nil
}

func (s *MemoryAgentStore) GetAgent(_ context.Context, id string) (Agent, error) {
	if s == nil {
		return Agent{}, fmt.Errorf("programauth memory agent store is nil")
	}
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	agent, ok := s.agents[id]
	if !ok {
		return Agent{}, ErrAgentNotFound
	}
	return cloneAgent(agent), nil
}

func (s *MemoryAgentStore) ListAgents(_ context.Context, query AgentQuery) ([]Agent, error) {
	if s == nil {
		return nil, fmt.Errorf("programauth memory agent store is nil")
	}
	query.OwnerUserID = strings.TrimSpace(query.OwnerUserID)
	query.TenantID = strings.TrimSpace(query.TenantID)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Agent, 0, len(s.agents))
	for _, agent := range s.agents {
		if query.OwnerUserID != "" && agent.OwnerUserID != query.OwnerUserID {
			continue
		}
		if query.TenantID != "" && agent.TenantID != query.TenantID {
			continue
		}
		if !query.IncludeDisabled && agent.Disabled() {
			continue
		}
		out = append(out, cloneAgent(agent))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (s *MemoryAgentStore) DisableAgent(_ context.Context, id string, disabledAt time.Time) (Agent, error) {
	if s == nil {
		return Agent{}, fmt.Errorf("programauth memory agent store is nil")
	}
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	agent, ok := s.agents[id]
	if !ok {
		return Agent{}, ErrAgentNotFound
	}
	disabledAt = disabledAt.UTC()
	agent.DisabledAt = &disabledAt
	agent.UpdatedAt = disabledAt
	s.agents[id] = agent
	return cloneAgent(agent), nil
}
