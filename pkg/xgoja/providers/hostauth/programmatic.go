package hostauth

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
)

type programmaticExports struct {
	grants func() *goja.Object
	agents *goja.Object
	tokens *goja.Object
}

type grantObjectStore struct{ specs sync.Map }

type grantBuilderState struct {
	tenantID     string
	resourceType string
	resourceID   string
	actions      []string
	set          gojahttp.GrantSet
}

func newProgrammaticExports(vm *goja.Runtime, agentService programauth.AgentService, apiTokenService programauth.APITokenService) programmaticExports {
	grantStore := &grantObjectStore{}
	agentsObj := vm.NewObject()
	modules.SetExport(agentsObj, "auth.agents", "create", func(name string) *goja.Object {
		return newAgentCreateBuilder(vm, grantStore, agentService, apiTokenService, name)
	})

	apiObj := vm.NewObject()
	modules.SetExport(apiObj, "auth.tokens.api", "issue", func(name string) *goja.Object {
		return newAPITokenIssueBuilder(vm, grantStore, apiTokenService, name)
	})
	modules.SetExport(apiObj, "auth.tokens.api", "list", func() *goja.Object {
		return newAPITokenListBuilder(vm, apiTokenService)
	})
	modules.SetExport(apiObj, "auth.tokens.api", "revoke", func() *goja.Object {
		return newAPITokenRevokeBuilder(vm, apiTokenService)
	})
	tokensObj := vm.NewObject()
	modules.SetExport(tokensObj, "auth.tokens", "api", apiObj)

	return programmaticExports{
		grants: func() *goja.Object { return newGrantBuilder(vm, grantStore) },
		agents: agentsObj,
		tokens: tokensObj,
	}
}

func newGrantBuilder(vm *goja.Runtime, store *grantObjectStore) *goja.Object {
	state := &grantBuilderState{}
	obj := vm.NewObject()
	store.specs.Store(obj, state)
	modules.SetExport(obj, "auth.grants", "tenant", func(id string) *goja.Object {
		state.tenantID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.grants", "resource", func(typ, id string) *goja.Object {
		state.resourceType = strings.TrimSpace(typ)
		state.resourceID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.grants", "allow", func(action string) *goja.Object {
		action = strings.TrimSpace(action)
		if action != "" {
			state.actions = append(state.actions, action)
		}
		return obj
	})
	modules.SetExport(obj, "auth.grants", "done", func() goja.Value {
		set, err := state.toGrantSet()
		if err != nil {
			panic(vm.NewGoError(err))
		}
		state.set = set
		return obj
	})
	modules.SetExport(obj, "auth.grants", "toJSON", func() goja.Value {
		set, err := state.toGrantSet()
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(set.ScopeStrings())
	})
	return obj
}

func (s *grantBuilderState) toGrantSet() (gojahttp.GrantSet, error) {
	grants := make([]gojahttp.Grant, 0, len(s.actions))
	for _, action := range s.actions {
		grants = append(grants, gojahttp.Grant{Action: action, TenantID: s.tenantID, ResourceType: s.resourceType, ResourceID: s.resourceID})
	}
	return gojahttp.NewGrantSet(grants...)
}

func (s *grantObjectStore) grantSet(vm *goja.Runtime, value goja.Value) (gojahttp.GrantSet, error) {
	if value == nil || goja.IsNull(value) || goja.IsUndefined(value) {
		return gojahttp.GrantSet{}, fmt.Errorf("expected value returned by auth.grants()")
	}
	raw, ok := s.specs.Load(value.ToObject(vm))
	if !ok {
		return gojahttp.GrantSet{}, fmt.Errorf("expected value returned by auth.grants()")
	}
	state, ok := raw.(*grantBuilderState)
	if !ok || state == nil {
		return gojahttp.GrantSet{}, fmt.Errorf("internal grant builder state has invalid type")
	}
	if len(state.set.Grants) > 0 {
		return state.set.Clone(), nil
	}
	return state.toGrantSet()
}

func newAgentCreateBuilder(vm *goja.Runtime, grantStore *grantObjectStore, agentService programauth.AgentService, apiTokenService programauth.APITokenService, name string) *goja.Object {
	spec := programauth.AgentCreateSpec{Name: strings.TrimSpace(name)}
	issueToken := false
	tokenName := ""
	var tokenExpiresAt *time.Time
	obj := vm.NewObject()
	modules.SetExport(obj, "auth.agents.create", "kind", func(kind string) *goja.Object {
		spec.Kind = programauth.AgentKind(strings.TrimSpace(kind))
		return obj
	})
	modules.SetExport(obj, "auth.agents.create", "ownerUserId", func(id string) *goja.Object {
		spec.OwnerUserID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.agents.create", "tenant", func(id string) *goja.Object {
		spec.TenantID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.agents.create", "tenantId", func(id string) *goja.Object {
		spec.TenantID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.agents.create", "createdBy", func(id string) *goja.Object {
		spec.CreatedBy = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.agents.create", "allow", func(action string) *goja.Object {
		spec.Policy.Grants = append(spec.Policy.Grants, gojahttp.Grant{Action: strings.TrimSpace(action), TenantID: spec.TenantID})
		return obj
	})
	modules.SetExport(obj, "auth.agents.create", "grants", func(value goja.Value) *goja.Object {
		set, err := grantStore.grantSet(vm, value)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		spec.Policy = set
		return obj
	})
	modules.SetExport(obj, "auth.agents.create", "issueApiToken", func(optionalName ...string) *goja.Object {
		issueToken = true
		if len(optionalName) > 0 {
			tokenName = strings.TrimSpace(optionalName[0])
		}
		return obj
	})
	modules.SetExport(obj, "auth.agents.create", "expiresInDays", func(days int) *goja.Object {
		if days > 0 {
			expiresAt := time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour)
			tokenExpiresAt = &expiresAt
		}
		return obj
	})
	modules.SetExport(obj, "auth.agents.create", "run", func() goja.Value {
		ctx := runtimebridge.CurrentOwnerContext(vm)
		agent, err := agentService.CreateAgent(ctx, spec)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		out := map[string]any{"agent": agentForJS(agent)}
		if issueToken {
			if tokenName == "" {
				tokenName = agent.Name
			}
			issued, err := apiTokenService.IssueAPIToken(ctx, programauth.APITokenIssueSpec{Name: tokenName, AgentID: agent.ID, CreatedBy: spec.CreatedBy, ExpiresAt: tokenExpiresAt, Grants: agent.Policy})
			if err != nil {
				panic(vm.NewGoError(err))
			}
			out["token"] = issuedAPITokenForJS(issued)
		}
		return vm.ToValue(out)
	})
	return obj
}

func newAPITokenIssueBuilder(vm *goja.Runtime, grantStore *grantObjectStore, apiTokenService programauth.APITokenService, name string) *goja.Object {
	spec := programauth.APITokenIssueSpec{Name: strings.TrimSpace(name)}
	obj := vm.NewObject()
	modules.SetExport(obj, "auth.tokens.api.issue", "agent", func(id string) *goja.Object {
		spec.AgentID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.tokens.api.issue", "subjectUserId", func(id string) *goja.Object {
		spec.SubjectUserID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.tokens.api.issue", "createdBy", func(id string) *goja.Object {
		spec.CreatedBy = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.tokens.api.issue", "allow", func(action string) *goja.Object {
		spec.Grants.Grants = append(spec.Grants.Grants, gojahttp.Grant{Action: strings.TrimSpace(action)})
		return obj
	})
	modules.SetExport(obj, "auth.tokens.api.issue", "grants", func(value goja.Value) *goja.Object {
		set, err := grantStore.grantSet(vm, value)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		spec.Grants = set
		return obj
	})
	modules.SetExport(obj, "auth.tokens.api.issue", "expiresInDays", func(days int) *goja.Object {
		if days > 0 {
			expiresAt := time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour)
			spec.ExpiresAt = &expiresAt
		}
		return obj
	})
	modules.SetExport(obj, "auth.tokens.api.issue", "run", func() goja.Value {
		issued, err := apiTokenService.IssueAPIToken(runtimebridge.CurrentOwnerContext(vm), spec)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(issuedAPITokenForJS(issued))
	})
	return obj
}

func newAPITokenListBuilder(vm *goja.Runtime, apiTokenService programauth.APITokenService) *goja.Object {
	query := programauth.APITokenQuery{}
	obj := vm.NewObject()
	modules.SetExport(obj, "auth.tokens.api.list", "agent", func(id string) *goja.Object {
		query.AgentID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.tokens.api.list", "subjectUserId", func(id string) *goja.Object {
		query.SubjectUserID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.tokens.api.list", "includeRevoked", func(include bool) *goja.Object {
		query.IncludeRevoked = include
		return obj
	})
	modules.SetExport(obj, "auth.tokens.api.list", "run", func() goja.Value {
		views, err := apiTokenService.ListAPITokens(runtimebridge.CurrentOwnerContext(vm), query)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		out := make([]map[string]any, len(views))
		for i, view := range views {
			out[i] = apiTokenViewForJS(view)
		}
		return vm.ToValue(out)
	})
	return obj
}

func newAPITokenRevokeBuilder(vm *goja.Runtime, apiTokenService programauth.APITokenService) *goja.Object {
	id := ""
	obj := vm.NewObject()
	modules.SetExport(obj, "auth.tokens.api.revoke", "id", func(value string) *goja.Object {
		id = strings.TrimSpace(value)
		return obj
	})
	modules.SetExport(obj, "auth.tokens.api.revoke", "run", func() goja.Value {
		view, err := apiTokenService.RevokeAPIToken(runtimebridge.CurrentOwnerContext(vm), id)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(apiTokenViewForJS(view))
	})
	return obj
}

func agentForJS(agent programauth.Agent) map[string]any {
	out := map[string]any{"id": agent.ID, "name": agent.Name, "kind": string(agent.Kind), "createdAt": agent.CreatedAt, "updatedAt": agent.UpdatedAt, "scopes": agent.Policy.ScopeStrings()}
	setString(out, "ownerUserId", agent.OwnerUserID)
	setString(out, "tenantId", agent.TenantID)
	setString(out, "createdBy", agent.CreatedBy)
	if agent.DisabledAt != nil {
		out["disabledAt"] = *agent.DisabledAt
	}
	return out
}

func issuedAPITokenForJS(issued programauth.IssuedAPIToken) map[string]any {
	out := apiTokenViewForJS(issued.Token)
	out["value"] = issued.Value
	return out
}

func apiTokenViewForJS(view programauth.APITokenView) map[string]any {
	out := map[string]any{"id": view.ID, "name": view.Name, "tokenPrefix": view.TokenPrefix, "credentialHint": view.CredentialHint, "createdAt": view.CreatedAt, "updatedAt": view.UpdatedAt, "scopes": append([]string(nil), view.Scopes...)}
	setString(out, "agentId", view.AgentID)
	setString(out, "subjectUserId", view.SubjectUserID)
	setString(out, "createdBy", view.CreatedBy)
	if view.ExpiresAt != nil {
		out["expiresAt"] = *view.ExpiresAt
	}
	if view.LastUsedAt != nil {
		out["lastUsedAt"] = *view.LastUsedAt
	}
	if view.RevokedAt != nil {
		out["revokedAt"] = *view.RevokedAt
	}
	return out
}
