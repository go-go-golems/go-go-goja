// Package buildspec defines the build-time xgoja.yaml schema.
//
// The types in this file are declarative *Spec DTOs: they describe what the
// generator should build, import, embed, or expose, but they do not perform
// runtime work. They are loaded from xgoja.yaml, defaulted, validated, and then
// converted by cmd/xgoja/internal/generate into the smaller app.RuntimeSpec JSON that
// the generated binary embeds and reads at runtime.
package buildspec

import "fmt"

// BuildSpec is the top-level build-time xgoja.yaml document.
// It includes build-only fields such as Go module settings, provider import
// paths, replacement paths, target import/root data, and BaseDir for resolving
// local resources. Generated binaries should use app.RuntimeSpec instead.
type BuildSpec struct {
	Name             string                        `yaml:"name"`
	AppName          string                        `yaml:"appName"`
	EnvPrefix        string                        `yaml:"envPrefix"`
	ConfigFile       *ConfigFileSpec               `yaml:"configFile,omitempty"`
	Go               GoSpec                        `yaml:"go"`
	Target           TargetSpec                    `yaml:"target"`
	Packages         []PackageSpec                 `yaml:"packages"`
	Modules          []ModuleInstanceSpec          `yaml:"modules"`
	Commands         CommandsSpec                  `yaml:"commands"`
	CommandProviders []CommandProviderInstanceSpec `yaml:"commandProviders"`
	JSVerbs          []JSVerbSourceSpec            `yaml:"jsverbs"`
	Help             HelpSpec                      `yaml:"help"`
	Assets           []AssetSourceSpec             `yaml:"assets"`
	BaseDir          string                        `yaml:"-"`
}

type ConfigFileSpec struct {
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	Layers   []string `yaml:"layers,omitempty" json:"layers,omitempty"`
	FileName string   `yaml:"fileName,omitempty" json:"fileName,omitempty"`
}

type GoSpec struct {
	Version string            `yaml:"version" json:"version"`
	Module  string            `yaml:"module" json:"module"`
	Tags    []string          `yaml:"tags" json:"tags,omitempty"`
	LDFlags []string          `yaml:"ldflags" json:"ldflags,omitempty"`
	Env     map[string]string `yaml:"env" json:"env,omitempty"`
	Imports []GoImportSpec    `yaml:"imports" json:"imports,omitempty"`
}

// GoImportSpec describes an extra Go import that should be emitted into
// generated source. It is intended for side-effect imports such as SQL drivers,
// but can also name regular imports for custom templates.
type GoImportSpec struct {
	Import  string `yaml:"import" json:"import"`
	Alias   string `yaml:"alias" json:"alias,omitempty"`
	Module  string `yaml:"module" json:"module,omitempty"`
	Version string `yaml:"version" json:"version,omitempty"`
}

type TargetSpec struct {
	Kind     string `yaml:"kind" json:"kind"`
	Import   string `yaml:"import" json:"import,omitempty"`
	Version  string `yaml:"version" json:"version,omitempty"`
	Root     string `yaml:"root" json:"root,omitempty"`
	Output   string `yaml:"output" json:"output"`
	Package  string `yaml:"package" json:"package,omitempty"`
	Template string `yaml:"template" json:"template,omitempty"`
}

type PackageSpec struct {
	ID       string `yaml:"id" json:"id"`
	Import   string `yaml:"import" json:"import"`
	Version  string `yaml:"version" json:"version,omitempty"`
	Register string `yaml:"register" json:"register"`
	Replace  string `yaml:"replace" json:"replace,omitempty"`
}

// ModuleInstanceSpec selects one provider module for the generated xgoja runtime.
// Its Config field is static module config from xgoja.yaml; it is declarative
// data that is later marshaled or merged into providerapi.ModuleSetupContext.Config.
type ModuleInstanceSpec struct {
	Package string         `yaml:"package" json:"package"`
	Name    string         `yaml:"name" json:"name"`
	As      string         `yaml:"as" json:"as,omitempty"`
	Config  map[string]any `yaml:"config" json:"config,omitempty"`
}

func (m ModuleInstanceSpec) Alias() string {
	if m.As != "" {
		return m.As
	}
	return m.Name
}

func (m ModuleInstanceSpec) Ref() string {
	return fmt.Sprintf("%s.%s", m.Package, m.Name)
}

type CommandsSpec struct {
	Eval    CommandSpec `yaml:"eval" json:"eval"`
	Run     CommandSpec `yaml:"run" json:"run"`
	Repl    CommandSpec `yaml:"repl" json:"repl"`
	JSVerbs CommandSpec `yaml:"jsverbs" json:"jsverbs"`
}

type CommandSpec struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Name    string `yaml:"name" json:"name,omitempty"`
	Mount   string `yaml:"mount" json:"mount,omitempty"`
}

// CommandProviderInstanceSpec selects one provider-owned command set for the
// generated binary. It describes which provider command factory to mount and
// with which static config; it is not the provider's CommandSetProvider
// definition itself.
type CommandProviderInstanceSpec struct {
	ID      string         `yaml:"id" json:"id"`
	Package string         `yaml:"package" json:"package"`
	Name    string         `yaml:"name" json:"name"`
	Mount   string         `yaml:"mount" json:"mount,omitempty"`
	Modules []string       `yaml:"modules" json:"modules,omitempty"`
	Config  map[string]any `yaml:"config" json:"config,omitempty"`
	Lazy    bool           `yaml:"lazy" json:"lazy,omitempty"`
}

type JSVerbSourceSpec struct {
	ID      string `yaml:"id" json:"id"`
	Path    string `yaml:"path" json:"path,omitempty"`
	Embed   bool   `yaml:"embed" json:"embed"`
	Package string `yaml:"package" json:"package,omitempty"`
	Source  string `yaml:"source" json:"source,omitempty"`
}

type HelpSpec struct {
	Sources []HelpSourceSpec `yaml:"sources" json:"sources,omitempty"`
}

type HelpSourceSpec struct {
	ID      string `yaml:"id" json:"id"`
	Path    string `yaml:"path" json:"path,omitempty"`
	Embed   bool   `yaml:"embed" json:"embed"`
	Package string `yaml:"package" json:"package,omitempty"`
	Source  string `yaml:"source" json:"source,omitempty"`
}

type AssetSourceSpec struct {
	ID          string `yaml:"id" json:"id"`
	Path        string `yaml:"path" json:"path,omitempty"`
	Embed       bool   `yaml:"embed" json:"embed"`
	Description string `yaml:"description" json:"description,omitempty"`
}
