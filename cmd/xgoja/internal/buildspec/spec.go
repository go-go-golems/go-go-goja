package buildspec

import "fmt"

type Spec struct {
	Name     string             `yaml:"name"`
	Go       GoSpec             `yaml:"go"`
	Target   TargetSpec         `yaml:"target"`
	Packages []PackageSpec      `yaml:"packages"`
	Runtimes map[string]Runtime `yaml:"runtimes"`
	Commands CommandsSpec       `yaml:"commands"`
	JSVerbs  []JSVerbSourceSpec `yaml:"jsverbs"`
	BaseDir  string             `yaml:"-"`
}

type GoSpec struct {
	Version string   `yaml:"version"`
	Module  string   `yaml:"module"`
	Tags    []string `yaml:"tags"`
	LDFlags []string `yaml:"ldflags"`
}

type TargetSpec struct {
	Kind    string `yaml:"kind"`
	Import  string `yaml:"import"`
	Version string `yaml:"version"`
	Root    string `yaml:"root"`
	Output  string `yaml:"output"`
}

type PackageSpec struct {
	ID       string `yaml:"id"`
	Import   string `yaml:"import"`
	Version  string `yaml:"version"`
	Register string `yaml:"register"`
	Replace  string `yaml:"replace"`
}

type Runtime struct {
	Modules []ModuleInstance `yaml:"modules"`
}

type ModuleInstance struct {
	Package string         `yaml:"package"`
	Name    string         `yaml:"name"`
	As      string         `yaml:"as"`
	Config  map[string]any `yaml:"config"`
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
	Repl    CommandSpec `yaml:"repl"`
	JSVerbs CommandSpec `yaml:"jsverbs"`
}

type CommandSpec struct {
	Enabled bool   `yaml:"enabled"`
	Runtime string `yaml:"runtime"`
	Name    string `yaml:"name"`
	Mount   string `yaml:"mount"`
}

type JSVerbSourceSpec struct {
	ID      string `yaml:"id"`
	Path    string `yaml:"path"`
	Embed   bool   `yaml:"embed"`
	Package string `yaml:"package"`
	Source  string `yaml:"source"`
}
