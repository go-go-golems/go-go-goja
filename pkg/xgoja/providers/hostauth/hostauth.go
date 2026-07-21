package hostauth

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/membershipinvite"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	hostauthsvc "github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const PackageID = "go-go-goja-hostauth"

type Config struct {
	Audit AuditConfig `json:"audit"`
}

type AuditConfig struct {
	MaxLimit      int `json:"maxLimit,omitempty"`
	MaxLimitKebab int `json:"max-limit,omitempty"`
}

// Register exposes safe JavaScript access to host-owned auth services. It does
// not expose raw auth database handles; callers get narrow builder APIs such as
// auth.audit.query().tenantId(...).run().
func Register(registry *providerapi.ProviderRegistry) error {
	return registry.Package(PackageID,
		authModule(),
		providerapi.CommandSetProvider{
			Name:          "operator",
			DefaultMount:  "operator",
			Description:   "Offline generated-host authentication operations",
			NewCommandSet: newOperatorCommandSet,
		},
	)
}

func authModule() providerapi.Module {
	return providerapi.Module{
		Name:        "auth",
		DefaultAs:   "auth",
		Description: "Safe JavaScript access to generated host auth services.",
		ConfigSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "audit": {
      "type": "object",
      "properties": {
        "maxLimit": {"type": "integer", "minimum": 1, "maximum": 100, "description": "Maximum records auth.audit.query may return."},
        "max-limit": {"type": "integer", "minimum": 1, "maximum": 100, "description": "YAML-friendly alias for maxLimit."}
      }
    }
  }
}`),
		NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			cfg := Config{}
			if err := decodeConfig(ctx.Config, &cfg); err != nil {
				return nil, fmt.Errorf("auth config: %w", err)
			}
			services, ok, err := hostauthsvc.LookupServices(ctx.Host)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("auth module requires hostauth services")
			}
			if services.AuditStore == nil {
				return nil, fmt.Errorf("auth module requires an audit store")
			}
			queryStore, ok := services.AuditStore.(audit.QueryStore)
			if !ok {
				return nil, fmt.Errorf("auth audit store %T does not support query access", services.AuditStore)
			}
			if services.Capability == nil {
				return nil, fmt.Errorf("auth module requires a capability store")
			}
			capabilityService := capability.Service{Store: services.Capability, Audit: services.AuditSink}
			maxLimit := effectiveMaxLimit(cfg.Audit)
			return newLoader(queryStore, maxLimit, capabilityService, services.MembershipInvites, services.Agents, services.APITokens), nil
		},
	}
}

func newLoader(queryStore audit.QueryStore, maxLimit int, capabilityService capability.Service, membershipInvites membershipinvite.Service, agentService programauth.AgentService, apiTokenService programauth.APITokenService) require.ModuleLoader {
	return func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)
		auditObj := vm.NewObject()
		modules.SetExport(auditObj, "auth.audit", "query", func() *goja.Object {
			return newAuditQueryBuilder(vm, queryStore, maxLimit)
		})
		modules.SetExport(exports, "auth", "audit", auditObj)

		capabilitiesObj := vm.NewObject()
		modules.SetExport(capabilitiesObj, "auth.capabilities", "issue", func(purpose string) *goja.Object {
			return newCapabilityIssueBuilder(vm, capabilityService, purpose)
		})
		modules.SetExport(capabilitiesObj, "auth.capabilities", "validate", func(token string) *goja.Object {
			return newCapabilityValidateBuilder(vm, capabilityService, token, false)
		})
		modules.SetExport(capabilitiesObj, "auth.capabilities", "consume", func(token string) *goja.Object {
			return newCapabilityValidateBuilder(vm, capabilityService, token, true)
		})
		modules.SetExport(capabilitiesObj, "auth.capabilities", "revoke", func() *goja.Object {
			return newCapabilityRevokeBuilder(vm, capabilityService)
		})
		modules.SetExport(exports, "auth", "capabilities", capabilitiesObj)

		membershipInvitesObj := vm.NewObject()
		modules.SetExport(membershipInvitesObj, "auth.membershipInvites", "begin", func(token string) *goja.Object {
			return newMembershipInviteBeginBuilder(vm, membershipInvites, token)
		})
		modules.SetExport(membershipInvitesObj, "auth.membershipInvites", "accept", func(token string) *goja.Object {
			return newMembershipInviteAcceptBuilder(vm, membershipInvites, token)
		})
		modules.SetExport(membershipInvitesObj, "auth.membershipInvites", "acceptPending", func(handle string) *goja.Object {
			return newMembershipInviteAcceptPendingBuilder(vm, membershipInvites, handle)
		})
		modules.SetExport(exports, "auth", "membershipInvites", membershipInvitesObj)

		programmatic := newProgrammaticExports(vm, agentService, apiTokenService)
		modules.SetExport(exports, "auth", "grants", programmatic.grants)
		modules.SetExport(exports, "auth", "agents", programmatic.agents)
		modules.SetExport(exports, "auth", "tokens", programmatic.tokens)
	}
}

func newMembershipInviteBeginBuilder(vm *goja.Runtime, service membershipinvite.Service, token string) *goja.Object {
	obj := vm.NewObject()
	modules.SetExport(obj, "auth.membershipInvites.begin", "run", func() goja.Value {
		pending, err := service.Begin(runtimebridge.CurrentOwnerContext(vm), token)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(map[string]any{"handle": pending.Handle, "capabilityId": pending.CapabilityID, "orgId": pending.TenantID, "email": pending.Email, "role": pending.Role, "expiresAt": pending.ExpiresAt})
	})
	return obj
}

func newMembershipInviteAcceptPendingBuilder(vm *goja.Runtime, service membershipinvite.Service, handle string) *goja.Object {
	actorID := ""
	obj := vm.NewObject()
	modules.SetExport(obj, "auth.membershipInvites.acceptPending", "actor", func(id string) *goja.Object { actorID = strings.TrimSpace(id); return obj })
	modules.SetExport(obj, "auth.membershipInvites.acceptPending", "run", func() goja.Value {
		result, err := service.AcceptPending(runtimebridge.CurrentOwnerContext(vm), handle, actorID)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(map[string]any{"capabilityId": result.CapabilityID, "userId": result.UserID, "orgId": result.TenantID, "role": result.Role})
	})
	return obj
}

func newMembershipInviteAcceptBuilder(vm *goja.Runtime, service membershipinvite.Service, token string) *goja.Object {
	actorID := ""
	obj := vm.NewObject()
	modules.SetExport(obj, "auth.membershipInvites.accept", "actor", func(id string) *goja.Object {
		actorID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.membershipInvites.accept", "run", func() goja.Value {
		result, err := service.Accept(runtimebridge.CurrentOwnerContext(vm), token, actorID)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(map[string]any{"capabilityId": result.CapabilityID, "userId": result.UserID, "orgId": result.TenantID, "role": result.Role})
	})
	return obj
}

func newAuditQueryBuilder(vm *goja.Runtime, queryStore audit.QueryStore, maxLimit int) *goja.Object {
	query := audit.Query{}
	obj := vm.NewObject()
	modules.SetExport(obj, "auth.audit.query", "tenantId", func(id string) *goja.Object {
		query.TenantID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.audit.query", "outcome", func(outcome string) *goja.Object {
		query.Outcome = strings.TrimSpace(outcome)
		return obj
	})
	modules.SetExport(obj, "auth.audit.query", "actorId", func(id string) *goja.Object {
		query.ActorID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.audit.query", "resource", func(typ, id string) *goja.Object {
		query.ResourceType = strings.TrimSpace(typ)
		query.ResourceID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.audit.query", "resourceType", func(typ string) *goja.Object {
		query.ResourceType = strings.TrimSpace(typ)
		return obj
	})
	modules.SetExport(obj, "auth.audit.query", "resourceId", func(id string) *goja.Object {
		query.ResourceID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.audit.query", "limit", func(limit int) *goja.Object {
		query.Limit = limit
		return obj
	})
	modules.SetExport(obj, "auth.audit.query", "offset", func(offset int) *goja.Object {
		query.Offset = offset
		return obj
	})
	modules.SetExport(obj, "auth.audit.query", "run", func() goja.Value {
		normalized := audit.NormalizeQuery(query, maxLimit)
		records, err := queryStore.QueryAuditRecords(runtimebridge.CurrentOwnerContext(vm), normalized)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(recordsForJS(records))
	})
	return obj
}

func newCapabilityIssueBuilder(vm *goja.Runtime, service capability.Service, purpose string) *goja.Object {
	spec := capability.IssueSpec{Purpose: strings.TrimSpace(purpose)}
	obj := vm.NewObject()
	modules.SetExport(obj, "auth.capabilities.issue", "subject", func(kind, id string) *goja.Object {
		kind = strings.TrimSpace(kind)
		id = strings.TrimSpace(id)
		if kind == "" || kind == "id" || kind == "user" {
			spec.SubjectID = id
		} else if id != "" {
			if spec.Claims == nil {
				spec.Claims = map[string]string{}
			}
			spec.Claims["subject."+kind] = id
		}
		return obj
	})
	modules.SetExport(obj, "auth.capabilities.issue", "subjectId", func(id string) *goja.Object {
		spec.SubjectID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.capabilities.issue", "resource", func(typ, id string) *goja.Object {
		spec.ResourceType = strings.TrimSpace(typ)
		spec.ResourceID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.capabilities.issue", "tenantId", func(id string) *goja.Object {
		if spec.Claims == nil {
			spec.Claims = map[string]string{}
		}
		spec.Claims["tenantId"] = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.capabilities.issue", "claimString", func(key, value string) *goja.Object {
		key = strings.TrimSpace(key)
		if key != "" {
			if spec.Claims == nil {
				spec.Claims = map[string]string{}
			}
			spec.Claims[key] = value
		}
		return obj
	})
	modules.SetExport(obj, "auth.capabilities.issue", "ttlSeconds", func(seconds int) *goja.Object {
		if seconds > 0 {
			spec.TTL = time.Duration(seconds) * time.Second
		}
		return obj
	})
	modules.SetExport(obj, "auth.capabilities.issue", "expiresAt", func(value string) *goja.Object {
		if strings.TrimSpace(value) == "" {
			return obj
		}
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
		if err != nil {
			panic(vm.NewGoError(err))
		}
		spec.ExpiresAt = parsed
		return obj
	})
	modules.SetExport(obj, "auth.capabilities.issue", "singleUse", func(singleUse bool) *goja.Object {
		spec.SingleUse = singleUse
		return obj
	})
	modules.SetExport(obj, "auth.capabilities.issue", "createdBy", func(id string) *goja.Object {
		spec.CreatedBy = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.capabilities.issue", "run", func() goja.Value {
		issued, err := service.Issue(runtimebridge.CurrentOwnerContext(vm), spec)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(map[string]any{"token": issued.Token, "capability": capabilityForJS(issued.Capability)})
	})
	return obj
}

func newCapabilityValidateBuilder(vm *goja.Runtime, service capability.Service, token string, consume bool) *goja.Object {
	purpose := ""
	expectedResourceType := ""
	expectedResourceID := ""
	obj := vm.NewObject()
	modules.SetExport(obj, "auth.capabilities.validate", "expectedType", func(value string) *goja.Object {
		purpose = strings.TrimSpace(value)
		return obj
	})
	modules.SetExport(obj, "auth.capabilities.validate", "expectedResource", func(typ, id string) *goja.Object {
		expectedResourceType = strings.TrimSpace(typ)
		expectedResourceID = strings.TrimSpace(id)
		return obj
	})
	modules.SetExport(obj, "auth.capabilities.validate", "run", func() goja.Value {
		ctx := runtimebridge.CurrentOwnerContext(vm)
		capabilityRecord, err := service.Validate(ctx, purpose, token)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		if expectedResourceType != "" && capabilityRecord.ResourceType != expectedResourceType {
			panic(vm.NewGoError(fmt.Errorf("capability: resource type mismatch")))
		}
		if expectedResourceID != "" && capabilityRecord.ResourceID != expectedResourceID {
			panic(vm.NewGoError(fmt.Errorf("capability: resource id mismatch")))
		}
		if consume {
			capabilityRecord, err = service.Consume(ctx, purpose, token)
			if err != nil {
				panic(vm.NewGoError(err))
			}
		}
		return vm.ToValue(capabilityForJS(*capabilityRecord))
	})
	return obj
}

func newCapabilityRevokeBuilder(vm *goja.Runtime, service capability.Service) *goja.Object {
	id := ""
	obj := vm.NewObject()
	modules.SetExport(obj, "auth.capabilities.revoke", "id", func(value string) *goja.Object {
		id = strings.TrimSpace(value)
		return obj
	})
	modules.SetExport(obj, "auth.capabilities.revoke", "reason", func(_ string) *goja.Object {
		return obj
	})
	modules.SetExport(obj, "auth.capabilities.revoke", "run", func() goja.Value {
		if err := service.Revoke(runtimebridge.CurrentOwnerContext(vm), id); err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(map[string]any{"revoked": true, "id": id})
	})
	return obj
}

func capabilityForJS(record capability.Capability) map[string]any {
	out := map[string]any{
		"id":           record.ID,
		"purpose":      record.Purpose,
		"resourceType": record.ResourceType,
		"resourceId":   record.ResourceID,
		"singleUse":    record.SingleUse,
		"expiresAt":    record.ExpiresAt,
		"createdAt":    record.CreatedAt,
	}
	setString(out, "subjectId", record.SubjectID)
	setString(out, "createdBy", record.CreatedBy)
	if record.UsedAt != nil {
		out["usedAt"] = *record.UsedAt
	}
	if record.RevokedAt != nil {
		out["revokedAt"] = *record.RevokedAt
	}
	if len(record.Claims) > 0 {
		out["claims"] = record.Claims
	}
	return out
}

func recordsForJS(records []audit.Record) []map[string]any {
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		item := map[string]any{
			"event":     record.Event,
			"outcome":   record.Outcome,
			"method":    record.Method,
			"pattern":   record.Pattern,
			"createdAt": record.CreatedAt,
		}
		setString(item, "reason", record.Reason)
		setInt(item, "statusCode", record.StatusCode)
		setString(item, "routeName", record.RouteName)
		setString(item, "action", record.Action)
		setString(item, "actorId", record.ActorID)
		setString(item, "actorKind", record.ActorKind)
		setString(item, "tenantId", record.TenantID)
		setString(item, "resourceType", record.ResourceType)
		setString(item, "resourceId", record.ResourceID)
		setString(item, "requestId", record.RequestID)
		setString(item, "ipHash", record.IPHash)
		setString(item, "userAgent", record.UserAgent)
		if len(record.Attributes) > 0 {
			item["attributes"] = record.Attributes
		}
		out = append(out, item)
	}
	return out
}

func setString(out map[string]any, key, value string) {
	if value != "" {
		out[key] = value
	}
}

func setInt(out map[string]any, key string, value int) {
	if value != 0 {
		out[key] = value
	}
}

func effectiveMaxLimit(cfg AuditConfig) int {
	maxLimit := cfg.MaxLimit
	if cfg.MaxLimitKebab > 0 {
		maxLimit = cfg.MaxLimitKebab
	}
	if maxLimit <= 0 || maxLimit > audit.MaxQueryLimit {
		return audit.MaxQueryLimit
	}
	return maxLimit
}

func decodeConfig(data json.RawMessage, out any) error {
	if len(data) == 0 || strings.TrimSpace(string(data)) == "null" {
		return nil
	}
	return json.Unmarshal(data, out)
}
