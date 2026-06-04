// Package app defines the runtime-side schema and wiring used by generated
// xgoja binaries.
//
// The Spec types in this file are declarative runtime DTOs decoded from the
// embedded JSON produced by cmd/xgoja/internal/generate. They intentionally omit
// build-only fields such as Go module versions, provider import paths, replace
// directives, target build roots, and source BaseDir. They describe what the
// generated binary should expose at runtime; concrete VM lifecycles live in
// engine.Runtime and app.RuntimeFactory.
package app

// Spec is the normalized embedded runtime spec decoded by a generated xgoja
// binary. It is derived from buildspec.Spec during code generation and contains
// only runtime-relevant command, module, config, help, jsverb, and asset data.
type Spec struct {
	Name             string                        `json:"name"`
	AppName          string                        `json:"appName,omitempty"`
	EnvPrefix        string                        `json:"envPrefix,omitempty"`
	Config           *ConfigSpec                   `json:"config,omitempty"`
	Target           TargetSpec                    `json:"target"`
	Packages         []PackageSpec                 `json:"packages"`
	Runtimes         map[string]RuntimeSpec        `json:"runtimes"`
	Commands         CommandsSpec                  `json:"commands"`
	CommandProviders []CommandProviderInstanceSpec `json:"commandProviders,omitempty"`
	JSVerbs          []JSVerbSourceSpec            `json:"jsverbs,omitempty"`
	Help             HelpSpec                      `json:"help,omitempty"`
	Assets           []AssetSourceSpec             `json:"assets,omitempty"`
}

type ConfigSpec struct {
	Enabled  bool     `json:"enabled"`
	Layers   []string `json:"layers,omitempty"`
	FileName string   `json:"fileName,omitempty"`
}

type TargetSpec struct {
	Kind   string `json:"kind"`
	Output string `json:"output"`
}

type PackageSpec struct {
	ID string `json:"id"`
}

// RuntimeSpec is a declarative runtime profile in the embedded app spec.
// It lists provider module instances selected for a named profile; it is not a
// concrete engine.Runtime.
type RuntimeSpec struct {
	Modules []ModuleInstanceSpec `json:"modules"`
}

// ModuleInstanceSpec selects one provider module for a runtime profile.
// The generated app resolves Package+Name through providerapi.Registry and uses
// As as the require() alias. Config is static module config carried from the
// generated spec.
type ModuleInstanceSpec struct {
	Package string         `json:"package"`
	Name    string         `json:"name"`
	As      string         `json:"as,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

func (m ModuleInstanceSpec) Alias() string {
	if m.As != "" {
		return m.As
	}
	return m.Name
}

type CommandsSpec struct {
	Eval    CommandSpec `json:"eval"`
	Run     CommandSpec `json:"run"`
	Repl    CommandSpec `json:"repl"`
	JSVerbs CommandSpec `json:"jsverbs"`
}

type CommandSpec struct {
	Enabled bool   `json:"enabled"`
	Runtime string `json:"runtime,omitempty"`
	Name    string `json:"name,omitempty"`
	Mount   string `json:"mount,omitempty"`
}

// CommandProviderInstanceSpec selects one provider-owned command set to attach
// to the generated Cobra root. It is the runtime DTO counterpart to the
// providerapi.CommandSetProvider definition stored in providerapi.Registry.
type CommandProviderInstanceSpec struct {
	ID             string         `json:"id"`
	Package        string         `json:"package"`
	Name           string         `json:"name"`
	Mount          string         `json:"mount,omitempty"`
	RuntimeProfile string         `json:"runtimeProfile,omitempty"`
	Modules        []string       `json:"modules,omitempty"`
	Config         map[string]any `json:"config,omitempty"`
	Lazy           bool           `json:"lazy,omitempty"`
}

type JSVerbSourceSpec struct {
	ID      string `json:"id"`
	Path    string `json:"path,omitempty"`
	Embed   bool   `json:"embed"`
	Package string `json:"package,omitempty"`
	Source  string `json:"source,omitempty"`
}

type HelpSpec struct {
	Sources []HelpSourceSpec `json:"sources,omitempty"`
}

type HelpSourceSpec struct {
	ID      string `json:"id"`
	Path    string `json:"path,omitempty"`
	Embed   bool   `json:"embed"`
	Package string `json:"package,omitempty"`
	Source  string `json:"source,omitempty"`
}

type AssetSourceSpec struct {
	ID          string `json:"id"`
	Path        string `json:"path,omitempty"`
	Embed       bool   `json:"embed"`
	Description string `json:"description,omitempty"`
}
