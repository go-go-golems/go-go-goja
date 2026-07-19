package programauth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth"
)

func TestAgentServiceCreatesNormalizesAndProjectsActor(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	policy := mustGrantSet(t, gojahttp.Grant{Action: "project.read", TenantID: "o1"})
	service := programauth.AgentService{
		Store: programauth.NewMemoryAgentStore(),
		Now:   func() time.Time { return now },
		NewID: func() (string, error) { return "agt_test", nil },
	}
	agent, err := service.CreateAgent(context.Background(), programauth.AgentCreateSpec{
		Name:        " Daily report bot ",
		Kind:        programauth.AgentKindCI,
		OwnerUserID: " u1 ",
		TenantID:    " o1 ",
		CreatedBy:   " u1 ",
		Policy:      policy,
	})
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}
	if agent.ID != "agt_test" || agent.Name != "Daily report bot" || agent.Kind != programauth.AgentKindCI || !agent.CreatedAt.Equal(now) {
		t.Fatalf("agent = %#v", agent)
	}
	if !agent.Policy.AllowsResource("project.read", "o1", "project", "p1") {
		t.Fatalf("agent policy did not allow expected action: %#v", agent.Policy)
	}
	actor := agent.Actor()
	if actor.ID != "agt_test" || actor.Kind != string(gojahttp.PrincipalKindAgent) || actor.TenantIDs[0] != "o1" || actor.Claims["ownerUserId"] != "u1" {
		t.Fatalf("actor = %#v", actor)
	}
}

func TestAgentServiceRestrictsOwnerManagement(t *testing.T) {
	service := programauth.AgentService{Store: programauth.NewMemoryAgentStore(), NewID: func() (string, error) { return "agt_owner", nil }}
	if _, err := service.CreateAgent(context.Background(), programauth.AgentCreateSpec{Name: "owner agent", OwnerUserID: "u1", Policy: mustGrantSet(t, gojahttp.Grant{Action: "report.read"})}); err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}
	if _, err := service.DisableOwnedAgent(context.Background(), "u2", "agt_owner"); !errors.Is(err, programauth.ErrAgentNotFound) {
		t.Fatalf("cross-owner disable error = %v", err)
	}
	agents, err := service.ListOwnedAgents(context.Background(), "u1")
	if err != nil || len(agents) != 1 {
		t.Fatalf("ListOwnedAgents = %#v, %v", agents, err)
	}
	if _, err := service.DisableOwnedAgent(context.Background(), "u1", "agt_owner"); err != nil {
		t.Fatalf("DisableOwnedAgent: %v", err)
	}
}

func TestMemoryAgentStoreClonesStoredAgents(t *testing.T) {
	store := programauth.NewMemoryAgentStore()
	service := programauth.AgentService{Store: store, NewID: func() (string, error) { return "agt_clone", nil }}
	agent, err := service.CreateAgent(context.Background(), programauth.AgentCreateSpec{Name: "clone", Policy: mustGrantSet(t, gojahttp.Grant{Action: "project.read"})})
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}
	agent.Policy.Grants[0].Action = "project.delete"
	got, err := service.GetAgent(context.Background(), "agt_clone")
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if got.Policy.Grants[0].Action != "project.read" {
		t.Fatalf("store did not clone policy: %#v", got.Policy)
	}
	got.Policy.Grants[0].Action = "project.delete"
	again, err := service.GetAgent(context.Background(), "agt_clone")
	if err != nil {
		t.Fatalf("GetAgent again: %v", err)
	}
	if again.Policy.Grants[0].Action != "project.read" {
		t.Fatalf("GetAgent did not return clone: %#v", again.Policy)
	}
}

func TestAgentServiceListAndDisable(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	ids := []string{"agt_1", "agt_2", "agt_3"}
	service := programauth.AgentService{Store: programauth.NewMemoryAgentStore(), Now: func() time.Time { return now }, NewID: func() (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	}}
	_, _ = service.CreateAgent(context.Background(), programauth.AgentCreateSpec{Name: "one", OwnerUserID: "u1", TenantID: "o1", Policy: mustGrantSet(t, gojahttp.Grant{Action: "project.read"})})
	_, _ = service.CreateAgent(context.Background(), programauth.AgentCreateSpec{Name: "two", OwnerUserID: "u1", TenantID: "o2", Policy: mustGrantSet(t, gojahttp.Grant{Action: "project.read"})})
	_, _ = service.CreateAgent(context.Background(), programauth.AgentCreateSpec{Name: "three", OwnerUserID: "u2", TenantID: "o1", Policy: mustGrantSet(t, gojahttp.Grant{Action: "project.read"})})
	listed, err := service.ListAgents(context.Background(), programauth.AgentQuery{OwnerUserID: "u1"})
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("owner list len=%d agents=%#v", len(listed), listed)
	}
	if _, err := service.DisableAgent(context.Background(), "agt_1"); err != nil {
		t.Fatalf("DisableAgent: %v", err)
	}
	if _, err := service.GetAgent(context.Background(), "agt_1"); !errors.Is(err, programauth.ErrAgentDisabled) {
		t.Fatalf("GetAgent disabled err=%v", err)
	}
	active, err := service.ListAgents(context.Background(), programauth.AgentQuery{OwnerUserID: "u1"})
	if err != nil {
		t.Fatalf("ListAgents active: %v", err)
	}
	if len(active) != 1 || active[0].ID != "agt_2" {
		t.Fatalf("active list = %#v", active)
	}
	all, err := service.ListAgents(context.Background(), programauth.AgentQuery{OwnerUserID: "u1", IncludeDisabled: true})
	if err != nil {
		t.Fatalf("ListAgents all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("all list = %#v", all)
	}
}

func TestAgentServiceValidation(t *testing.T) {
	service := programauth.AgentService{Store: programauth.NewMemoryAgentStore()}
	if _, err := service.CreateAgent(context.Background(), programauth.AgentCreateSpec{}); err == nil {
		t.Fatal("expected missing name error")
	}
	if _, err := service.CreateAgent(context.Background(), programauth.AgentCreateSpec{Name: "bad", Kind: "robot"}); err == nil {
		t.Fatal("expected bad kind error")
	}
	if _, err := service.GetAgent(context.Background(), "missing"); !errors.Is(err, programauth.ErrAgentNotFound) {
		t.Fatalf("missing agent err=%v", err)
	}
}

func mustGrantSet(t *testing.T, grants ...gojahttp.Grant) gojahttp.GrantSet {
	t.Helper()
	set, err := gojahttp.NewGrantSet(grants...)
	if err != nil {
		t.Fatalf("NewGrantSet: %v", err)
	}
	return set
}
