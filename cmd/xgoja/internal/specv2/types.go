// Package specv2 defines the native xgoja/v2 configuration schema.
//
// The v2 schema is intentionally an intent-level planner input. It describes
// providers, Go-backed runtime modules, goja-executed sources, command
// surfaces, and generated artifacts. It does not expose general-purpose
// browser/node bundler knobs; xgoja owns compiler defaults for code that runs
// inside goja.
package specv2

import "github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth"

// Schema is the required schema marker for native v2 xgoja specs.
const Schema = "xgoja/v2"

type Config struct {
	Schema    string                 `yaml:"schema" json:"schema"`
	Name      string                 `yaml:"name" json:"name"`
	App       AppSpec                `yaml:"app,omitempty" json:"app,omitempty"`
	Go        GoSpec                 `yaml:"go,omitempty" json:"go,omitempty"`
	Workspace WorkspaceSpec          `yaml:"workspace,omitempty" json:"workspace,omitempty"`
	Providers []ProviderSpec         `yaml:"providers,omitempty" json:"providers,omitempty"`
	Runtime   RuntimeSpec            `yaml:"runtime,omitempty" json:"runtime,omitempty"`
	Auth      *hostauth.Config       `yaml:"auth,omitempty" json:"auth,omitempty"`
	Sources   []SourceSpec           `yaml:"sources,omitempty" json:"sources,omitempty"`
	Commands  []CommandSurfaceSpec   `yaml:"commands,omitempty" json:"commands,omitempty"`
	Artifacts []ArtifactSpec         `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	Profiles  map[string]ProfileSpec `yaml:"profiles,omitempty" json:"profiles,omitempty"`
	BaseDir   string                 `yaml:"-" json:"-"`
}

type AppSpec struct {
	Name       string          `yaml:"name,omitempty" json:"name,omitempty"`
	EnvPrefix  string          `yaml:"envPrefix,omitempty" json:"envPrefix,omitempty"`
	ConfigFile *ConfigFileSpec `yaml:"configFile,omitempty" json:"configFile,omitempty"`
}

type ConfigFileSpec struct {
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	Layers   []string `yaml:"layers,omitempty" json:"layers,omitempty"`
	FileName string   `yaml:"fileName,omitempty" json:"fileName,omitempty"`
}

type GoSpec struct {
	Module  string            `yaml:"module,omitempty" json:"module,omitempty"`
	Version string            `yaml:"version,omitempty" json:"version,omitempty"`
	Tags    []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	LDFlags []string          `yaml:"ldflags,omitempty" json:"ldflags,omitempty"`
	Env     map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Imports []GoImportSpec    `yaml:"imports,omitempty" json:"imports,omitempty"`
}

type GoImportSpec struct {
	Import  string `yaml:"import" json:"import"`
	Alias   string `yaml:"alias,omitempty" json:"alias,omitempty"`
	Module  string `yaml:"module,omitempty" json:"module,omitempty"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

type WorkspaceSpec struct {
	Mode string `yaml:"mode,omitempty" json:"mode,omitempty"`
	File string `yaml:"file,omitempty" json:"file,omitempty"`
}

type ProviderSpec struct {
	ID       string             `yaml:"id" json:"id"`
	Import   string             `yaml:"import" json:"import"`
	Register string             `yaml:"register,omitempty" json:"register,omitempty"`
	Module   ProviderModuleSpec `yaml:"module,omitempty" json:"module,omitempty"`
}

type ProviderModuleSpec struct {
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
	Replace string `yaml:"replace,omitempty" json:"replace,omitempty"`
}

type RuntimeSpec struct {
	Modules []RuntimeModuleSpec `yaml:"modules,omitempty" json:"modules,omitempty"`
}

type RuntimeModuleSpec struct {
	Provider string         `yaml:"provider" json:"provider"`
	Name     string         `yaml:"name" json:"name"`
	As       string         `yaml:"as,omitempty" json:"as,omitempty"`
	Config   map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
}

func (m RuntimeModuleSpec) Alias() string {
	if m.As != "" {
		return m.As
	}
	return m.Name
}

type SourceKind string

const (
	SourceKindJSVerbs SourceKind = "jsverbs"
	SourceKindScript  SourceKind = "script"
	SourceKindAssets  SourceKind = "assets"
	SourceKindHelp    SourceKind = "help"
)

type SourceSpec struct {
	ID         string         `yaml:"id" json:"id"`
	Kind       SourceKind     `yaml:"kind" json:"kind"`
	From       SourceFromSpec `yaml:"from" json:"from"`
	Include    []string       `yaml:"include,omitempty" json:"include,omitempty"`
	Exclude    []string       `yaml:"exclude,omitempty" json:"exclude,omitempty"`
	Extensions []string       `yaml:"extensions,omitempty" json:"extensions,omitempty"`
	Language   string         `yaml:"language,omitempty" json:"language,omitempty"`
	Compile    *CompileSpec   `yaml:"compile,omitempty" json:"compile,omitempty"`
}

type SourceFromSpec struct {
	Dir       string              `yaml:"dir,omitempty" json:"dir,omitempty"`
	Provider  *ProviderSourceRef  `yaml:"provider,omitempty" json:"provider,omitempty"`
	Workspace *WorkspaceSourceRef `yaml:"workspace,omitempty" json:"workspace,omitempty"`
}

type ProviderSourceRef struct {
	Provider string `yaml:"provider" json:"provider"`
	Source   string `yaml:"source" json:"source"`
}

type WorkspaceSourceRef struct {
	Module string `yaml:"module" json:"module"`
	Path   string `yaml:"path" json:"path"`
}

type CompileSpec struct {
	Mode   string            `yaml:"mode,omitempty" json:"mode,omitempty"`
	Bundle bool              `yaml:"bundle,omitempty" json:"bundle,omitempty"`
	Check  *CompileCheckSpec `yaml:"check,omitempty" json:"check,omitempty"`
	Define map[string]string `yaml:"define,omitempty" json:"define,omitempty"`
}

type CompileCheckSpec struct {
	Command []string `yaml:"command,omitempty" json:"command,omitempty"`
}

type CommandSurfaceSpec struct {
	ID       string         `yaml:"id" json:"id"`
	Type     string         `yaml:"type" json:"type"`
	Name     string         `yaml:"name,omitempty" json:"name,omitempty"`
	Mount    string         `yaml:"mount,omitempty" json:"mount,omitempty"`
	Provider string         `yaml:"provider,omitempty" json:"provider,omitempty"`
	Sources  []string       `yaml:"sources,omitempty" json:"sources,omitempty"`
	Modules  []string       `yaml:"modules,omitempty" json:"modules,omitempty"`
	Config   map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
	Lazy     bool           `yaml:"lazy,omitempty" json:"lazy,omitempty"`
}

type ArtifactSpec struct {
	ID       string   `yaml:"id" json:"id"`
	Type     string   `yaml:"type" json:"type"`
	Output   string   `yaml:"output,omitempty" json:"output,omitempty"`
	Package  string   `yaml:"package,omitempty" json:"package,omitempty"`
	Import   string   `yaml:"import,omitempty" json:"import,omitempty"`
	Root     string   `yaml:"root,omitempty" json:"root,omitempty"`
	Template string   `yaml:"template,omitempty" json:"template,omitempty"`
	Sources  []string `yaml:"sources,omitempty" json:"sources,omitempty"`
	Strict   bool     `yaml:"strict,omitempty" json:"strict,omitempty"`
}

type ProfileSpec struct {
	Workspace *WorkspaceSpec `yaml:"workspace,omitempty" json:"workspace,omitempty"`
}
