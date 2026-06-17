// Package app defines the v2-native runtime plan and wiring used by generated
// xgoja binaries.
//
// RuntimePlan is decoded from the embedded JSON produced by cmd/xgoja/internal/generate.
// It mirrors the public xgoja/v2 concepts that are still relevant at runtime:
// providers, selected runtime modules, sources, commands, artifacts, and app
// settings. Build-only fields such as Go module versions, provider import paths,
// replace directives, and source BaseDir are intentionally omitted.
package app

import (
	"encoding/json"
	"fmt"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth"
)

const RuntimePlanSchema = "xgoja/runtime/v2"

type RuntimePlan struct {
	Schema    string           `json:"schema"`
	Name      string           `json:"name"`
	App       AppPlan          `json:"app,omitempty"`
	Target    TargetPlan       `json:"target"`
	Providers []ProviderPlan   `json:"providers,omitempty"`
	Runtime   RuntimeSection   `json:"runtime,omitempty"`
	Auth      *hostauth.Config `json:"auth,omitempty"`
	Sources   []SourcePlan     `json:"sources,omitempty"`
	Commands  []CommandPlan    `json:"commands,omitempty"`
	Artifacts []ArtifactPlan   `json:"artifacts,omitempty"`
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
	Name     string         `json:"name"`
	As       string         `json:"as,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
}

func (m RuntimeModulePlan) ProviderID() string {
	return m.Provider
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
	Source     string          `json:"source,omitempty"`
	Include    []string        `json:"include,omitempty"`
	Exclude    []string        `json:"exclude,omitempty"`
	Extensions []string        `json:"extensions,omitempty"`
	TypeScript *TypeScriptPlan `json:"typescript,omitempty"`
}

func (s SourcePlan) ProviderID() string {
	return s.Provider
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

type CommandPlan struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Name     string         `json:"name,omitempty"`
	Mount    string         `json:"mount,omitempty"`
	Provider string         `json:"provider,omitempty"`
	Sources  []string       `json:"sources,omitempty"`
	Modules  []string       `json:"modules,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
	Lazy     bool           `json:"lazy,omitempty"`
}

func (c CommandPlan) ProviderID() string {
	return c.Provider
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
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	for _, key := range []string{"appName", "envPrefix", "configFile", "packages", "modules", "commandProviders", "jsverbs", "help", "assets"} {
		if _, ok := payload[key]; ok {
			return fmt.Errorf("runtime plan uses removed legacy key %q", key)
		}
	}
	type alias RuntimePlan
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*p = RuntimePlan(decoded)
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
	return p.Runtime.Modules
}

func (p *RuntimePlan) runtimeCommands() []CommandPlan {
	if p == nil {
		return nil
	}
	return append([]CommandPlan(nil), p.Commands...)
}

func (p *RuntimePlan) commandByType(commandType string) (CommandPlan, bool) {
	for _, command := range p.runtimeCommands() {
		if command.Type == commandType {
			return command, true
		}
	}
	return CommandPlan{}, false
}

func (p *RuntimePlan) allSources() []SourcePlan {
	if p == nil {
		return nil
	}
	return append([]SourcePlan(nil), p.Sources...)
}

func (p *RuntimePlan) sourcesByKind(kind SourceKind) []SourcePlan {
	if p == nil {
		return nil
	}
	out := make([]SourcePlan, 0)
	for _, source := range p.allSources() {
		if source.Kind == kind {
			out = append(out, source)
		}
	}
	return out
}
