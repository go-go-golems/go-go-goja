package hostauth

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
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
	return registry.Package(PackageID, authModule())
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
			maxLimit := effectiveMaxLimit(cfg.Audit)
			return newLoader(queryStore, maxLimit), nil
		},
	}
}

func newLoader(queryStore audit.QueryStore, maxLimit int) require.ModuleLoader {
	return func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)
		auditObj := vm.NewObject()
		modules.SetExport(auditObj, "auth.audit", "query", func() *goja.Object {
			return newAuditQueryBuilder(vm, queryStore, maxLimit)
		})
		modules.SetExport(exports, "auth", "audit", auditObj)
	}
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
