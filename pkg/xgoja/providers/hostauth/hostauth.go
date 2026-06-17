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
// not expose raw auth database handles; callers get narrow APIs such as
// auth.audit.query(...).
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
		modules.SetExport(auditObj, "auth.audit", "query", func(call goja.FunctionCall) goja.Value {
			query, err := queryFromValue(vm, call.Argument(0), maxLimit)
			if err != nil {
				panic(vm.NewTypeError(err.Error()))
			}
			records, err := queryStore.QueryAuditRecords(runtimebridge.CurrentOwnerContext(vm), query)
			if err != nil {
				panic(vm.NewGoError(err))
			}
			return vm.ToValue(recordsForJS(records))
		})
		modules.SetExport(exports, "auth", "audit", auditObj)
	}
}

func queryFromValue(vm *goja.Runtime, value goja.Value, maxLimit int) (audit.Query, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return audit.NormalizeQuery(audit.Query{}, maxLimit), nil
	}
	obj := value.ToObject(vm)
	if obj == nil {
		return audit.Query{}, fmt.Errorf("auth.audit.query expects an object")
	}
	query := audit.Query{
		TenantID:     optionalString(obj, "tenantId"),
		Outcome:      optionalString(obj, "outcome"),
		ActorID:      optionalString(obj, "actorId"),
		ResourceType: optionalString(obj, "resourceType"),
		ResourceID:   optionalString(obj, "resourceId"),
		Limit:        optionalInt(obj, "limit"),
		Offset:       optionalInt(obj, "offset"),
	}
	return audit.NormalizeQuery(query, maxLimit), nil
}

func optionalString(obj *goja.Object, name string) string {
	value := obj.Get(name)
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return ""
	}
	return value.String()
}

func optionalInt(obj *goja.Object, name string) int {
	value := obj.Get(name)
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return 0
	}
	return int(value.ToInteger())
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
