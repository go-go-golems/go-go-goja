package host

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/modules"
	dbm "github.com/go-go-golems/go-go-goja/modules/database"
	_ "github.com/go-go-golems/go-go-goja/modules/fs"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const PackageID = "go-go-goja-host"

// GuardConfig is the common enable switch for host-capability modules. Each
// host module requires an explicit {"allow": true} config block in the runtime
// profile before the loader is created.
type GuardConfig struct {
	Allow bool `json:"allow"`
}

type ExecConfig struct {
	Allow           bool     `json:"allow"`
	AllowedCommands []string `json:"allowedCommands,omitempty"`
}

type DatabaseConfig struct {
	AllowConfigure bool `json:"allowConfigure"`
}

// Register exposes guarded host-capability modules. These modules can touch
// the host filesystem, process table, or databases and therefore require
// explicit per-module config in xgoja.yaml.
func Register(registry *providerapi.Registry) error {
	return registry.Package(PackageID,
		fsModule("fs"),
		fsModule("node:fs"),
		execModule(),
		databaseModule("database"),
		databaseModule("db"),
	)
}

func fsModule(name string) providerapi.Module {
	mod := modules.GetModule(name)
	return providerapi.Module{
		Name:        name,
		DefaultAs:   name,
		Description: "Guarded host filesystem module. Requires config.allow=true and does not sandbox paths.",
		ConfigSchema: json.RawMessage(`{
  "type": "object",
  "required": ["allow"],
  "properties": {
    "allow": {"type": "boolean", "const": true, "description": "Explicitly enable filesystem access. This does not sandbox paths."}
  }
}`),
		New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
			if err := requireAllow(ctx.Config, name); err != nil {
				return nil, err
			}
			if mod == nil {
				return nil, fmt.Errorf("fs module %q is not registered", name)
			}
			return mod.Loader, nil
		},
	}
}

func execModule() providerapi.Module {
	return providerapi.Module{
		Name:        "exec",
		DefaultAs:   "exec",
		Description: "Guarded process execution module. Requires config.allow=true and can optionally restrict command names.",
		ConfigSchema: json.RawMessage(`{
  "type": "object",
  "required": ["allow"],
  "properties": {
    "allow": {"type": "boolean", "const": true, "description": "Explicitly enable process execution."},
    "allowedCommands": {"type": "array", "items": {"type": "string"}, "description": "Optional exact command allow-list. Empty means any command is allowed."}
  }
}`),
		New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
			cfg := ExecConfig{}
			if err := decodeConfig(ctx.Config, &cfg); err != nil {
				return nil, fmt.Errorf("exec config: %w", err)
			}
			if !cfg.Allow {
				return nil, fmt.Errorf("exec module requires config.allow=true")
			}
			allowed := map[string]struct{}{}
			for _, command := range cfg.AllowedCommands {
				command = strings.TrimSpace(command)
				if command != "" {
					allowed[command] = struct{}{}
				}
			}
			return func(vm *goja.Runtime, moduleObj *goja.Object) {
				exports := moduleObj.Get("exports").(*goja.Object)
				modules.SetExport(exports, "exec", "run", func(cmd string, args []string) (string, error) {
					if len(allowed) > 0 {
						if _, ok := allowed[cmd]; !ok {
							return "", fmt.Errorf("command %q is not allowed", cmd)
						}
					}
					// #nosec G204 -- guarded xgoja host provider explicitly exists to run configured trusted commands.
					out, err := exec.Command(cmd, args...).CombinedOutput()
					return string(out), err
				})
			}, nil
		},
	}
}

func databaseModule(name string) providerapi.Module {
	return providerapi.Module{
		Name:        name,
		DefaultAs:   name,
		Description: "Guarded database module. configure() is disabled unless config.allowConfigure=true.",
		ConfigSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "allowConfigure": {"type": "boolean", "description": "Allow JavaScript to call configure(driverName, dataSourceName)."}
  }
}`),
		New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
			cfg := DatabaseConfig{}
			if err := decodeConfig(ctx.Config, &cfg); err != nil {
				return nil, fmt.Errorf("database config: %w", err)
			}
			mod := dbm.New(dbm.WithName(name), dbm.WithConfigureEnabled(cfg.AllowConfigure))
			return mod.Loader, nil
		},
	}
}

func requireAllow(data json.RawMessage, moduleName string) error {
	cfg := GuardConfig{}
	if err := decodeConfig(data, &cfg); err != nil {
		return fmt.Errorf("%s config: %w", moduleName, err)
	}
	if !cfg.Allow {
		return fmt.Errorf("%s module requires config.allow=true", moduleName)
	}
	return nil
}

func decodeConfig(data json.RawMessage, out any) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return err
	}
	return nil
}
