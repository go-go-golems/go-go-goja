package host

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/modules"
	dbm "github.com/go-go-golems/go-go-goja/modules/database"
	fsmod "github.com/go-go-golems/go-go-goja/modules/fs"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const PackageID = "go-go-goja-host"

// GuardConfig is the common enable switch for host-capability modules. Each
// host module requires an explicit {"allow": true} config block in the runtime
// profile before the loader is created.
type GuardConfig struct {
	Allow bool `json:"allow"`
}

type FSConfig struct {
	Allow    bool             `json:"allow"`
	Embedded EmbeddedFSConfig `json:"embedded"`
}

type EmbeddedFSConfig struct {
	Allow  bool         `json:"allow"`
	Mounts []AssetMount `json:"mounts"`
}

type AssetMount struct {
	Asset string `json:"asset"`
	Mount string `json:"mount"`
	Root  string `json:"root,omitempty"`
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
	return providerapi.Module{
		Name:        name,
		DefaultAs:   name,
		Description: "Configurable filesystem module. Use config.allow=true for host filesystem access, or config.embedded.allow=true with mounts for read-only embedded assets. Prefer separate aliases such as fs:host and fs:assets.",
		ConfigSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "allow": {"type": "boolean", "const": true, "description": "Explicitly enable host filesystem access. This does not sandbox paths."},
    "embedded": {
      "type": "object",
      "properties": {
        "allow": {"type": "boolean", "const": true, "description": "Enable read-only embedded asset mounts."},
        "mounts": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["asset", "mount"],
            "properties": {
              "asset": {"type": "string"},
              "mount": {"type": "string"},
              "root": {"type": "string"}
            }
          }
        }
      }
    }
  }
}`),
		New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
			cfg := FSConfig{}
			if err := decodeConfig(ctx.Config, &cfg); err != nil {
				return nil, fmt.Errorf("%s config: %w", name, err)
			}
			requireName := ctx.As
			if strings.TrimSpace(requireName) == "" {
				requireName = ctx.Name
			}
			if strings.TrimSpace(requireName) == "" {
				requireName = name
			}

			switch {
			case cfg.Embedded.Allow && cfg.Allow:
				return nil, fmt.Errorf("%s module config cannot combine allow=true and embedded.allow=true; register separate aliases such as fs:host and fs:assets", requireName)
			case cfg.Embedded.Allow:
				backend, err := embeddedBackendFromConfig(ctx.Host, cfg.Embedded)
				if err != nil {
					return nil, fmt.Errorf("%s embedded config: %w", requireName, err)
				}
				return fsmod.New(fsmod.WithName(requireName), fsmod.WithBackend(backend)).Loader, nil
			case cfg.Allow:
				return fsmod.New(fsmod.WithName(requireName), fsmod.WithBackend(fsmod.OSBackend{})).Loader, nil
			default:
				return nil, fmt.Errorf("%s module requires config.allow=true or config.embedded.allow=true", requireName)
			}
		},
	}
}

func embeddedBackendFromConfig(host providerapi.HostServices, cfg EmbeddedFSConfig) (*fsmod.ReadOnlyFSBackend, error) {
	if host == nil || host.AssetResolver() == nil {
		return nil, fmt.Errorf("host asset resolver is not configured")
	}
	if len(cfg.Mounts) == 0 {
		return nil, fmt.Errorf("at least one embedded mount is required")
	}
	resolver := host.AssetResolver()
	mounts := make([]fsmod.FSMount, 0, len(cfg.Mounts))
	for i, mount := range cfg.Mounts {
		assetID := strings.TrimSpace(mount.Asset)
		if assetID == "" {
			return nil, fmt.Errorf("mount %d asset is required", i)
		}
		mountPoint := strings.TrimSpace(mount.Mount)
		if mountPoint == "" {
			return nil, fmt.Errorf("mount %d mount path is required", i)
		}
		fsys, root, ok := resolver.ResolveAsset(assetID)
		if !ok {
			return nil, fmt.Errorf("unknown embedded asset %q", assetID)
		}
		if extraRoot := strings.TrimSpace(mount.Root); extraRoot != "" {
			root = path.Join(root, strings.TrimPrefix(extraRoot, "/"))
		}
		mounts = append(mounts, fsmod.FSMount{FS: fsys, Root: root, Mount: mountPoint})
	}
	return fsmod.NewReadOnlyFSBackend(mounts...), nil
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

func decodeConfig(data json.RawMessage, out any) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return err
	}
	return nil
}
