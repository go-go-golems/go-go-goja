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
	Runtimes         map[string]RuntimeSpec        `yaml:"runtimes"`
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
	Version string   `yaml:"version" json:"version"`
	Module  string   `yaml:"module" json:"module"`
	Tags    []string `yaml:"tags" json:"tags,omitempty"`
	LDFlags []string `yaml:"ldflags" json:"ldflags,omitempty"`
}

type TargetSpec struct {
	Kind    string `yaml:"kind" json:"kind"`
	Import  string `yaml:"import" json:"import,omitempty"`
	Version string `yaml:"version" json:"version,omitempty"`
	Root    string `yaml:"root" json:"root,omitempty"`
	Output  string `yaml:"output" json:"output"`
}

type PackageSpec struct {
	ID       string `yaml:"id" json:"id"`
	Import   string `yaml:"import" json:"import"`
	Version  string `yaml:"version" json:"version,omitempty"`
	Register string `yaml:"register" json:"register"`
	Replace  string `yaml:"replace" json:"replace,omitempty"`
}

// RuntimeSpec is a declarative runtime profile from xgoja.yaml.
// It lists selected provider module instances; it is not a concrete Goja
// runtime. Concrete runtimes are engine.Runtime values created by the generated
// app at execution time.
type RuntimeSpec struct {
	Modules []ModuleInstanceSpec `yaml:"modules" json:"modules"`
}

// ModuleInstanceSpec selects one provider module inside a runtime profile.
// Its Config field is static module config from xgoja.yaml; it is declarative
// data that is later marshaled or merged into providerapi.ModuleContext.Config.
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
	Runtime string `yaml:"runtime" json:"runtime,omitempty"`
	Name    string `yaml:"name" json:"name,omitempty"`
	Mount   string `yaml:"mount" json:"mount,omitempty"`
}

// CommandProviderInstanceSpec selects one provider-owned command set for the
// generated binary. It describes which provider command factory to mount and
// with which static config; it is not the provider's CommandSetProvider
// definition itself.
type CommandProviderInstanceSpec struct {
	ID             string         `yaml:"id" json:"id"`
	Package        string         `yaml:"package" json:"package"`
	Name           string         `yaml:"name" json:"name"`
	Mount          string         `yaml:"mount" json:"mount,omitempty"`
	RuntimeProfile string         `yaml:"runtimeProfile" json:"runtimeProfile,omitempty"`
	Modules        []string       `yaml:"modules" json:"modules,omitempty"`
	Config         map[string]any `yaml:"config" json:"config,omitempty"`
	Lazy           bool           `yaml:"lazy" json:"lazy,omitempty"`
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
