package programauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

var (
	ErrAgentNotFound = errors.New("programauth agent not found")
	ErrAgentDisabled = errors.New("programauth agent disabled")
)

// AgentKind describes the product category of an automation identity.
type AgentKind string

const (
	AgentKindPersonal    AgentKind = "personal"
	AgentKindService     AgentKind = "service"
	AgentKindDevice      AgentKind = "device"
	AgentKindCI          AgentKind = "ci"
	AgentKindIntegration AgentKind = "integration"
)

// Agent is a durable automation principal. Credentials such as API tokens prove
// possession; agents carry ownership, lifecycle, and grant policy.
type Agent struct {
	ID          string
	Name        string
	Kind        AgentKind
	OwnerUserID string
	TenantID    string
	DisabledAt  *time.Time
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Policy      gojahttp.GrantSet
}

func (a Agent) Disabled() bool { return a.DisabledAt != nil }

// Actor projects an agent into the planned-route actor shape.
func (a Agent) Actor() *gojahttp.Actor {
	claims := map[string]any{"name": a.Name, "kind": string(a.Kind)}
	if a.OwnerUserID != "" {
		claims["ownerUserId"] = a.OwnerUserID
	}
	if a.TenantID != "" {
		claims["tenantId"] = a.TenantID
	}
	return &gojahttp.Actor{ID: a.ID, Kind: string(gojahttp.PrincipalKindAgent), TenantIDs: tenantIDs(a.TenantID), Claims: claims}
}

type AgentCreateSpec struct {
	ID          string
	Name        string
	Kind        AgentKind
	OwnerUserID string
	TenantID    string
	CreatedBy   string
	Policy      gojahttp.GrantSet
}

type AgentQuery struct {
	OwnerUserID     string
	TenantID        string
	IncludeDisabled bool
}

type AgentStore interface {
	CreateAgent(ctx context.Context, agent Agent) (Agent, error)
	GetAgent(ctx context.Context, id string) (Agent, error)
	ListAgents(ctx context.Context, query AgentQuery) ([]Agent, error)
	DisableAgent(ctx context.Context, id string, disabledAt time.Time) (Agent, error)
}

type AgentService struct {
	Store AgentStore
	Now   func() time.Time
	NewID func() (string, error)
}

func (s AgentService) CreateAgent(ctx context.Context, spec AgentCreateSpec) (Agent, error) {
	if s.Store == nil {
		return Agent{}, fmt.Errorf("programauth agent store is required")
	}
	agent, err := normalizeAgentCreateSpec(spec, s.now())
	if err != nil {
		return Agent{}, err
	}
	if agent.ID == "" {
		agent.ID, err = s.newID()
		if err != nil {
			return Agent{}, err
		}
	}
	return s.Store.CreateAgent(ctx, agent)
}

func (s AgentService) GetAgent(ctx context.Context, id string) (Agent, error) {
	if s.Store == nil {
		return Agent{}, fmt.Errorf("programauth agent store is required")
	}
	agent, err := s.Store.GetAgent(ctx, strings.TrimSpace(id))
	if err != nil {
		return Agent{}, err
	}
	if agent.Disabled() {
		return Agent{}, ErrAgentDisabled
	}
	return agent, nil
}

func (s AgentService) ListAgents(ctx context.Context, query AgentQuery) ([]Agent, error) {
	if s.Store == nil {
		return nil, fmt.Errorf("programauth agent store is required")
	}
	query.OwnerUserID = strings.TrimSpace(query.OwnerUserID)
	query.TenantID = strings.TrimSpace(query.TenantID)
	return s.Store.ListAgents(ctx, query)
}

func (s AgentService) DisableAgent(ctx context.Context, id string) (Agent, error) {
	if s.Store == nil {
		return Agent{}, fmt.Errorf("programauth agent store is required")
	}
	return s.Store.DisableAgent(ctx, strings.TrimSpace(id), s.now())
}

// ListOwnedAgents and DisableOwnedAgent keep the owner predicate at the
// service boundary; a UI must not become the only cross-user protection.
func (s AgentService) ListOwnedAgents(ctx context.Context, ownerUserID string) ([]Agent, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return nil, fmt.Errorf("agent owner user id is required")
	}
	return s.ListAgents(ctx, AgentQuery{OwnerUserID: ownerUserID, IncludeDisabled: true})
}

func (s AgentService) DisableOwnedAgent(ctx context.Context, ownerUserID, id string) (Agent, error) {
	if s.Store == nil {
		return Agent{}, fmt.Errorf("programauth agent store is required")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	agent, err := s.Store.GetAgent(ctx, strings.TrimSpace(id))
	if err != nil || ownerUserID == "" || agent.OwnerUserID != ownerUserID {
		return Agent{}, ErrAgentNotFound
	}
	return s.Store.DisableAgent(ctx, agent.ID, s.now())
}

func normalizeAgentCreateSpec(spec AgentCreateSpec, now time.Time) (Agent, error) {
	agent := Agent{
		ID:          strings.TrimSpace(spec.ID),
		Name:        strings.TrimSpace(spec.Name),
		Kind:        spec.Kind,
		OwnerUserID: strings.TrimSpace(spec.OwnerUserID),
		TenantID:    strings.TrimSpace(spec.TenantID),
		CreatedBy:   strings.TrimSpace(spec.CreatedBy),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if agent.Name == "" {
		return Agent{}, fmt.Errorf("agent name is required")
	}
	if agent.Kind == "" {
		agent.Kind = AgentKindService
	}
	switch agent.Kind {
	case AgentKindPersonal, AgentKindService, AgentKindDevice, AgentKindCI, AgentKindIntegration:
	default:
		return Agent{}, fmt.Errorf("unsupported agent kind %q", agent.Kind)
	}
	policy, err := spec.Policy.Normalize()
	if err != nil {
		return Agent{}, err
	}
	agent.Policy = policy
	return agent, nil
}

func (s AgentService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s AgentService) newID() (string, error) {
	if s.NewID != nil {
		return s.NewID()
	}
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "agt_" + hex.EncodeToString(buf), nil
}

func tenantIDs(tenantID string) []string {
	if tenantID == "" {
		return nil
	}
	return []string{tenantID}
}

func cloneAgent(agent Agent) Agent {
	out := agent
	out.Policy = agent.Policy.Clone()
	if agent.DisabledAt != nil {
		disabledAt := *agent.DisabledAt
		out.DisabledAt = &disabledAt
	}
	return out
}
