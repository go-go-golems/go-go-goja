package buildspec

import "fmt"

type Spec struct {
	Name             string                    `yaml:"name"`
	AppName          string                    `yaml:"appName"`
	EnvPrefix        string                    `yaml:"envPrefix"`
	Config           *ConfigSpec               `yaml:"config,omitempty"`
	Go               GoSpec                    `yaml:"go"`
	Target           TargetSpec                `yaml:"target"`
	Packages         []PackageSpec             `yaml:"packages"`
	Runtimes         map[string]Runtime        `yaml:"runtimes"`
	Commands         CommandsSpec              `yaml:"commands"`
	CommandProviders []CommandProviderInstance `yaml:"commandProviders"`
	JSVerbs          []JSVerbSourceSpec        `yaml:"jsverbs"`
	Help             HelpSpec                  `yaml:"help"`
	Assets           []AssetSourceSpec         `yaml:"assets"`
	BaseDir          string                    `yaml:"-"`
}

type ConfigSpec struct {
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

type Runtime struct {
	Modules []ModuleInstance `yaml:"modules" json:"modules"`
}

type ModuleInstance struct {
	Package string         `yaml:"package" json:"package"`
	Name    string         `yaml:"name" json:"name"`
	As      string         `yaml:"as" json:"as,omitempty"`
	Config  map[string]any `yaml:"config" json:"config,omitempty"`
}

func (m ModuleInstance) Alias() string {
	if m.As != "" {
		return m.As
	}
	return m.Name
}

func (m ModuleInstance) Ref() string {
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

type CommandProviderInstance struct {
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
