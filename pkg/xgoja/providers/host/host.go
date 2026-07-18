package host

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/modules"
	dbm "github.com/go-go-golems/go-go-goja/modules/database"
	_ "github.com/go-go-golems/go-go-goja/modules/exec"
	fetchmod "github.com/go-go-golems/go-go-goja/modules/fetch"
	fsmod "github.com/go-go-golems/go-go-goja/modules/fs"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
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

type FetchConfig struct {
	Allow            bool                   `json:"allow"`
	AllowedOrigins   []string               `json:"allowedOrigins,omitempty"`
	Timeout          string                 `json:"timeout,omitempty"`
	MaxResponseBytes int64                  `json:"maxResponseBytes,omitempty"`
	Credentials      FetchCredentialsConfig `json:"credentials,omitempty"`
}

type FetchCredentialsConfig struct {
	AllowEnv     bool     `json:"allowEnv,omitempty"`
	AllowFiles   bool     `json:"allowFiles,omitempty"`
	AllowedFiles []string `json:"allowedFiles,omitempty"`
}

type ExecConfig struct {
	Allow           bool     `json:"allow"`
	AllowedCommands []string `json:"allowedCommands,omitempty"`
}

type DatabaseConfig struct {
	AllowConfigure bool   `json:"allowConfigure"`
	DriverName     string `json:"driverName,omitempty"`
	DataSourceName string `json:"dataSourceName,omitempty"`
}

// Register exposes guarded host-capability modules. These modules can touch
// the host filesystem, process table, or databases and therefore require
// explicit per-module config in xgoja.yaml.
func Register(registry *providerapi.ProviderRegistry) error {
	return registry.Package(PackageID,
		fsModule("fs"),
		fsModule("node:fs"),
		fetchModule("fetch"),
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
		TypeScript:  fsmod.New(fsmod.WithName(name)).TypeScriptModule(),
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
		NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
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

func fetchModule(name string) providerapi.Module {
	return providerapi.Module{
		Name:        name,
		DefaultAs:   name,
		Description: "Guarded outbound HTTP client module. Requires config.allow=true and supports origin allow-lists, timeouts, response-size limits, and framework-native bearer credential sources.",
		TypeScript:  fetchmod.New(fetchmod.WithName(name)).TypeScriptModule(),
		ConfigSchema: json.RawMessage(`{
  "type": "object",
  "required": ["allow"],
  "properties": {
    "allow": {"type": "boolean", "const": true, "description": "Explicitly enable outbound HTTP from JavaScript."},
    "allowedOrigins": {"type": "array", "items": {"type": "string"}, "description": "Allowed URL origins. Exact origins are supported, plus development patterns such as http://127.0.0.1:* . Empty means any origin."},
    "timeout": {"type": "string", "description": "Default request timeout as a Go duration, for example 5s."},
    "maxResponseBytes": {"type": "integer", "minimum": 1, "description": "Maximum buffered response body size."},
    "credentials": {
      "type": "object",
      "properties": {
        "allowEnv": {"type": "boolean", "description": "Allow bearer credentials to be read from environment variables."},
        "allowFiles": {"type": "boolean", "description": "Allow bearer credentials to be read from files."},
        "allowedFiles": {"type": "array", "items": {"type": "string"}, "description": "Optional exact file paths allowed for credential file sources."}
      }
    }
  }
}`),
		NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			cfg := FetchConfig{}
			if err := decodeConfig(ctx.Config, &cfg); err != nil {
				return nil, fmt.Errorf("%s config: %w", name, err)
			}
			if !cfg.Allow {
				return nil, fmt.Errorf("%s module requires config.allow=true", name)
			}
			policy, err := fetchPolicyFromConfig(cfg)
			if err != nil {
				return nil, fmt.Errorf("%s config: %w", name, err)
			}
			requireName := ctx.As
			if strings.TrimSpace(requireName) == "" {
				requireName = ctx.Name
			}
			if strings.TrimSpace(requireName) == "" {
				requireName = name
			}
			return fetchmod.New(fetchmod.WithName(requireName), fetchmod.WithPolicy(policy)).Loader, nil
		},
	}
}

func fetchPolicyFromConfig(cfg FetchConfig) (fetchmod.Policy, error) {
	var timeout time.Duration
	if strings.TrimSpace(cfg.Timeout) != "" {
		parsed, err := time.ParseDuration(strings.TrimSpace(cfg.Timeout))
		if err != nil {
			return fetchmod.Policy{}, err
		}
		timeout = parsed
	}
	return fetchmod.Policy{
		AllowedOrigins:   append([]string(nil), cfg.AllowedOrigins...),
		Timeout:          timeout,
		MaxResponseBytes: cfg.MaxResponseBytes,
		Credentials: fetchmod.CredentialPolicy{
			AllowEnv:     cfg.Credentials.AllowEnv,
			AllowFiles:   cfg.Credentials.AllowFiles,
			AllowedFiles: append([]string(nil), cfg.Credentials.AllowedFiles...),
		},
	}, nil
}

func execModule() providerapi.Module {
	return providerapi.Module{
		Name:        "exec",
		DefaultAs:   "exec",
		Description: "Guarded process execution module. Requires config.allow=true and can optionally restrict command names.",
		TypeScript:  nativeModuleTypeScript("exec"),
		ConfigSchema: json.RawMessage(`{
  "type": "object",
  "required": ["allow"],
  "properties": {
    "allow": {"type": "boolean", "const": true, "description": "Explicitly enable process execution."},
    "allowedCommands": {"type": "array", "items": {"type": "string"}, "description": "Optional exact command allow-list. Empty means any command is allowed."}
  }
}`),
		NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
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
		TypeScript:  dbm.New(dbm.WithName(name)).TypeScriptModule(),
		ConfigSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "allowConfigure": {"type": "boolean", "description": "Allow JavaScript to call configure(driverName, dataSourceName). Ignored when driverName/dataSourceName preconfigure the module."},
    "driverName": {"type": "string", "description": "Optional SQL driver name used to preconfigure the module before JavaScript runs."},
    "dataSourceName": {"type": "string", "description": "Optional SQL data source name used with driverName to preconfigure the module before JavaScript runs."}
  }
}`),
		NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			cfg := DatabaseConfig{}
			if err := decodeConfig(ctx.Config, &cfg); err != nil {
				return nil, fmt.Errorf("database config: %w", err)
			}
			mod, err := databaseModuleFromConfig(name, cfg)
			if err != nil {
				return nil, err
			}
			return mod.Loader, nil
		},
	}
}

func databaseModuleFromConfig(name string, cfg DatabaseConfig) (*dbm.DBModule, error) {
	driverName := strings.TrimSpace(cfg.DriverName)
	dataSourceName := strings.TrimSpace(cfg.DataSourceName)
	if driverName == "" && dataSourceName == "" {
		return dbm.New(dbm.WithName(name), dbm.WithConfigureEnabled(cfg.AllowConfigure)), nil
	}
	if driverName == "" || dataSourceName == "" {
		return nil, fmt.Errorf("database config requires both driverName and dataSourceName for preconfigured modules")
	}
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open preconfigured database %q: %w", driverName, err)
	}
	return dbm.New(dbm.WithName(name), dbm.WithPreconfiguredDB(db), dbm.WithCloseFn(db.Close)), nil
}

func nativeModuleTypeScript(name string) *spec.Module {
	mod := modules.GetModule(name)
	if mod == nil {
		return nil
	}
	declarer, ok := mod.(modules.TypeScriptDeclarer)
	if !ok {
		return nil
	}
	return declarer.TypeScriptModule()
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
