package app

type Spec struct {
	Name             string                    `json:"name"`
	AppName          string                    `json:"appName,omitempty"`
	EnvPrefix        string                    `json:"envPrefix,omitempty"`
	Config           *ConfigSpec               `json:"config,omitempty"`
	Target           TargetSpec                `json:"target"`
	Packages         []PackageSpec             `json:"packages"`
	Runtimes         map[string]Runtime        `json:"runtimes"`
	Commands         CommandsSpec              `json:"commands"`
	CommandProviders []CommandProviderInstance `json:"commandProviders,omitempty"`
	JSVerbs          []JSVerbSourceSpec        `json:"jsverbs,omitempty"`
	Help             HelpSpec                  `json:"help,omitempty"`
	Assets           []AssetSourceSpec         `json:"assets,omitempty"`
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

type Runtime struct {
	Modules []ModuleInstance `json:"modules"`
}

type ModuleInstance struct {
	Package string         `json:"package"`
	Name    string         `json:"name"`
	As      string         `json:"as,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

func (m ModuleInstance) Alias() string {
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

type CommandProviderInstance struct {
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
