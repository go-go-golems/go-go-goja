// Package app defines the v2-native runtime plan and wiring used by generated
// xgoja binaries.
//
// RuntimePlan is decoded from the embedded JSON produced by cmd/xgoja/internal/generate.
// It mirrors the public xgoja/v2 concepts that are still relevant at runtime:
// providers, selected runtime modules, sources, commands, artifacts, and app
// settings. Build-only fields such as Go module versions, provider import paths,
// replace directives, and source BaseDir are intentionally omitted.
package app

import "encoding/json"

const RuntimePlanSchema = "xgoja/runtime/v2"

type RuntimePlan struct {
	Schema    string         `json:"schema"`
	Name      string         `json:"name"`
	App       AppPlan        `json:"app,omitempty"`
	Target    TargetPlan     `json:"target"`
	Providers []ProviderPlan `json:"providers,omitempty"`
	Runtime   RuntimeSection `json:"runtime,omitempty"`
	Sources   []SourcePlan   `json:"sources,omitempty"`
	Commands  []CommandPlan  `json:"commands,omitempty"`
	Artifacts []ArtifactPlan `json:"artifacts,omitempty"`

	// Deprecated compatibility fields for old in-repository tests while the
	// runtime implementation is being moved to the v2 plan shape. These fields are
	// never emitted in generated runtime JSON.
	Modules          []RuntimeModulePlan `json:"-"`
	CommandProviders []CommandPlan       `json:"-"`
	JSVerbs          []SourcePlan        `json:"-"`
	Assets           []SourcePlan        `json:"-"`
	LegacyCommands   CommandsSpec        `json:"-"`
}

type AppPlan struct {
	Name       string          `json:"name,omitempty"`
	EnvPrefix  string          `json:"envPrefix,omitempty"`
	ConfigFile *ConfigFilePlan `json:"configFile,omitempty"`
}

type ConfigFilePlan struct {
	Enabled  bool     `json:"enabled"`
	Layers   []string `json:"layers,omitempty"`
	FileName string   `json:"fileName,omitempty"`
}

type TargetPlan struct {
	Kind   string `json:"kind"`
	Output string `json:"output"`
}

type ProviderPlan struct {
	ID string `json:"id"`
}

type RuntimeSection struct {
	Modules []RuntimeModulePlan `json:"modules,omitempty"`
}

// RuntimeModulePlan selects one provider module for the generated xgoja runtime.
type RuntimeModulePlan struct {
	Provider string         `json:"provider"`
	Package  string         `json:"-"`
	Name     string         `json:"name"`
	As       string         `json:"as,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
}

func (m *RuntimeModulePlan) UnmarshalJSON(data []byte) error {
	type alias RuntimeModulePlan
	var raw struct {
		alias
		Package string `json:"package,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*m = RuntimeModulePlan(raw.alias)
	if m.Provider == "" {
		m.Provider = raw.Package
	}
	return nil
}

func (m RuntimeModulePlan) ProviderID() string {
	if m.Provider != "" {
		return m.Provider
	}
	return m.Package
}

func (m RuntimeModulePlan) Alias() string {
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

type SourcePlan struct {
	ID         string          `json:"id"`
	Kind       SourceKind      `json:"kind"`
	Path       string          `json:"path,omitempty"`
	Embed      bool            `json:"embed,omitempty"`
	Provider   string          `json:"provider,omitempty"`
	Package    string          `json:"-"`
	Source     string          `json:"source,omitempty"`
	Include    []string        `json:"include,omitempty"`
	Exclude    []string        `json:"exclude,omitempty"`
	Extensions []string        `json:"extensions,omitempty"`
	TypeScript *TypeScriptPlan `json:"typescript,omitempty"`
}

func (s *SourcePlan) UnmarshalJSON(data []byte) error {
	type alias SourcePlan
	var raw struct {
		alias
		Package string `json:"package,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*s = SourcePlan(raw.alias)
	if s.Provider == "" {
		s.Provider = raw.Package
	}
	return nil
}

func (s SourcePlan) ProviderID() string {
	if s.Provider != "" {
		return s.Provider
	}
	return s.Package
}

type TypeScriptPlan struct {
	Enabled      bool              `json:"enabled"`
	Bundle       bool              `json:"bundle"`
	Target       string            `json:"target,omitempty"`
	Format       string            `json:"format,omitempty"`
	Platform     string            `json:"platform,omitempty"`
	Tsconfig     string            `json:"tsconfig,omitempty"`
	Sourcemap    string            `json:"sourcemap,omitempty"`
	External     []string          `json:"external,omitempty"`
	Define       map[string]string `json:"define,omitempty"`
	CheckCommand []string          `json:"checkCommand,omitempty"`
}

type CommandSpec struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name,omitempty"`
	Mount   string `json:"mount,omitempty"`
}

type CommandsSpec struct {
	Eval    CommandSpec `json:"eval"`
	Run     CommandSpec `json:"run"`
	Repl    CommandSpec `json:"repl"`
	JSVerbs CommandSpec `json:"jsverbs"`
}

type CommandProviderInstanceSpec = CommandPlan

type CommandPlan struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Name     string         `json:"name,omitempty"`
	Mount    string         `json:"mount,omitempty"`
	Provider string         `json:"provider,omitempty"`
	Package  string         `json:"-"`
	Sources  []string       `json:"sources,omitempty"`
	Modules  []string       `json:"modules,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
	Lazy     bool           `json:"lazy,omitempty"`
}

func (c *CommandPlan) UnmarshalJSON(data []byte) error {
	type alias CommandPlan
	var raw struct {
		alias
		Package string `json:"package,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = CommandPlan(raw.alias)
	if c.Provider == "" {
		c.Provider = raw.Package
	}
	return nil
}

func (c CommandPlan) ProviderID() string {
	if c.Provider != "" {
		return c.Provider
	}
	return c.Package
}

type ArtifactPlan struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Output  string   `json:"output,omitempty"`
	Package string   `json:"package,omitempty"`
	Import  string   `json:"import,omitempty"`
	Root    string   `json:"root,omitempty"`
	Sources []string `json:"sources,omitempty"`
	Strict  bool     `json:"strict,omitempty"`
}

func (p *RuntimePlan) UnmarshalJSON(data []byte) error {
	type runtimePlanAlias RuntimePlan
	var raw struct {
		*runtimePlanAlias
		AppName          string              `json:"appName,omitempty"`
		EnvPrefix        string              `json:"envPrefix,omitempty"`
		ConfigFile       *ConfigFilePlan     `json:"configFile,omitempty"`
		Modules          []RuntimeModulePlan `json:"modules,omitempty"`
		CommandsRaw      json.RawMessage     `json:"commands,omitempty"`
		CommandProviders []CommandPlan       `json:"commandProviders,omitempty"`
		JSVerbs          []SourcePlan        `json:"jsverbs,omitempty"`
		Help             struct {
			Sources []SourcePlan `json:"sources,omitempty"`
		} `json:"help,omitempty"`
		Assets []SourcePlan `json:"assets,omitempty"`
	}
	raw.runtimePlanAlias = (*runtimePlanAlias)(p)
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if p.App.Name == "" {
		p.App.Name = raw.AppName
	}
	if p.App.EnvPrefix == "" {
		p.App.EnvPrefix = raw.EnvPrefix
	}
	if p.App.ConfigFile == nil {
		p.App.ConfigFile = raw.ConfigFile
	}
	if len(p.Runtime.Modules) == 0 {
		p.Runtime.Modules = raw.Modules
	}
	if len(raw.CommandsRaw) > 0 {
		var commandList []CommandPlan
		if err := json.Unmarshal(raw.CommandsRaw, &commandList); err == nil {
			p.Commands = commandList
		} else {
			var commandObject CommandsSpec
			if err := json.Unmarshal(raw.CommandsRaw, &commandObject); err != nil {
				return err
			}
			for typ, command := range map[string]CommandSpec{"builtin.eval": commandObject.Eval, "builtin.run": commandObject.Run, "builtin.repl": commandObject.Repl, "builtin.jsverbs": commandObject.JSVerbs} {
				if command.Enabled {
					p.Commands = append(p.Commands, CommandPlan{Type: typ, Name: command.Name, Mount: command.Mount})
				}
			}
		}
	}
	p.Commands = append(p.Commands, raw.CommandProviders...)
	for i := range raw.JSVerbs {
		raw.JSVerbs[i].Kind = SourceKindJSVerbs
	}
	for i := range raw.Help.Sources {
		raw.Help.Sources[i].Kind = SourceKindHelp
	}
	for i := range raw.Assets {
		raw.Assets[i].Kind = SourceKindAssets
	}
	p.Sources = append(p.Sources, raw.JSVerbs...)
	p.Sources = append(p.Sources, raw.Help.Sources...)
	p.Sources = append(p.Sources, raw.Assets...)
	return nil
}

func (p *RuntimePlan) AppName() string {
	if p == nil {
		return ""
	}
	if p.App.Name != "" {
		return p.App.Name
	}
	return p.Name
}

func (p *RuntimePlan) runtimeModules() []RuntimeModulePlan {
	if p == nil {
		return nil
	}
	if len(p.Runtime.Modules) > 0 {
		return p.Runtime.Modules
	}
	return p.Modules
}

func (p *RuntimePlan) runtimeCommands() []CommandPlan {
	if p == nil {
		return nil
	}
	out := append([]CommandPlan(nil), p.Commands...)
	for _, command := range p.CommandProviders {
		if command.Type == "" {
			command.Type = "provider.command-set"
		}
		out = append(out, command)
	}
	for typ, command := range map[string]CommandSpec{
		"builtin.eval":    p.LegacyCommands.Eval,
		"builtin.run":     p.LegacyCommands.Run,
		"builtin.repl":    p.LegacyCommands.Repl,
		"builtin.jsverbs": p.LegacyCommands.JSVerbs,
	} {
		if command.Enabled {
			out = append(out, CommandPlan{Type: typ, Name: command.Name, Mount: command.Mount})
		}
	}
	return out
}

func (p *RuntimePlan) commandByType(commandType string) (CommandPlan, bool) {
	for _, command := range p.runtimeCommands() {
		if command.Type == commandType {
			return command, true
		}
	}
	return CommandPlan{}, false
}

func (p *RuntimePlan) sourcesByKind(kind SourceKind) []SourcePlan {
	if p == nil {
		return nil
	}
	out := make([]SourcePlan, 0)
	for _, source := range p.Sources {
		if source.Kind == kind {
			out = append(out, source)
		}
	}
	if kind == SourceKindJSVerbs {
		out = append(out, p.JSVerbs...)
	}
	if kind == SourceKindAssets {
		out = append(out, p.Assets...)
	}
	return out
}
