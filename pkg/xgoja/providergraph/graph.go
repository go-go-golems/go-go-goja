package providergraph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type ProviderSelection struct {
	ID string
}

type RuntimeModuleSelection struct {
	Provider string
	Name     string
	As       string
}

func (m RuntimeModuleSelection) Alias() string {
	if strings.TrimSpace(m.As) != "" {
		return strings.TrimSpace(m.As)
	}
	return strings.TrimSpace(m.Name)
}

type CommandSetSelection struct {
	ID       string
	Provider string
	Name     string
}

type Graph struct {
	providers      map[string]providerapi.Package
	providerOrder  []string
	modules        []ResolvedRuntimeModule
	moduleAliases  map[string]ResolvedRuntimeModule
	commandSets    []ResolvedCommandSet
	commandSetByID map[string]ResolvedCommandSet
}

type ResolvedRuntimeModule struct {
	Provider string
	Name     string
	Alias    string
	Module   providerapi.Module
}

type ResolvedCommandSet struct {
	ID          string
	Provider    string
	Name        string
	ProviderDef providerapi.CommandSetProvider
}

type Options struct {
	Providers   []ProviderSelection
	Modules     []RuntimeModuleSelection
	CommandSets []CommandSetSelection
}

func Build(registry *providerapi.ProviderRegistry, opts Options) (*Graph, error) {
	if registry == nil {
		return nil, fmt.Errorf("provider registry is nil")
	}
	graph := &Graph{
		providers:      map[string]providerapi.Package{},
		moduleAliases:  map[string]ResolvedRuntimeModule{},
		commandSetByID: map[string]ResolvedCommandSet{},
	}
	available := map[string]providerapi.Package{}
	for _, pkg := range registry.Packages() {
		available[pkg.ID] = pkg
	}
	for _, selection := range opts.Providers {
		id := strings.TrimSpace(selection.ID)
		if id == "" {
			return nil, fmt.Errorf("provider id is required")
		}
		pkg, ok := available[id]
		if !ok {
			return nil, fmt.Errorf("unknown provider %q", id)
		}
		if _, exists := graph.providers[id]; exists {
			return nil, fmt.Errorf("duplicate provider %q", id)
		}
		graph.providers[id] = pkg
		graph.providerOrder = append(graph.providerOrder, id)
	}
	for _, selection := range opts.Modules {
		providerID := strings.TrimSpace(selection.Provider)
		moduleName := strings.TrimSpace(selection.Name)
		if _, ok := graph.providers[providerID]; !ok {
			return nil, fmt.Errorf("runtime module %s.%s references unselected provider %q", providerID, moduleName, providerID)
		}
		module, ok := registry.ResolveModule(providerID, moduleName)
		if !ok {
			return nil, fmt.Errorf("unknown runtime module %s.%s", providerID, moduleName)
		}
		alias := selection.Alias()
		if alias == "" {
			alias = firstNonEmpty(module.DefaultAs, moduleName)
		}
		if _, exists := graph.moduleAliases[alias]; exists {
			return nil, fmt.Errorf("duplicate runtime module alias %q", alias)
		}
		resolved := ResolvedRuntimeModule{Provider: providerID, Name: moduleName, Alias: alias, Module: module}
		graph.modules = append(graph.modules, resolved)
		graph.moduleAliases[alias] = resolved
	}
	for _, selection := range opts.CommandSets {
		id := strings.TrimSpace(selection.ID)
		if id == "" {
			id = strings.TrimSpace(selection.Provider) + "." + strings.TrimSpace(selection.Name)
		}
		providerID := strings.TrimSpace(selection.Provider)
		name := strings.TrimSpace(selection.Name)
		if _, ok := graph.providers[providerID]; !ok {
			return nil, fmt.Errorf("command set %s references unselected provider %q", id, providerID)
		}
		provider, ok := registry.ResolveCommandSetProvider(providerID, name)
		if !ok {
			return nil, fmt.Errorf("unknown command set %s.%s", providerID, name)
		}
		if _, exists := graph.commandSetByID[id]; exists {
			return nil, fmt.Errorf("duplicate command set id %q", id)
		}
		resolved := ResolvedCommandSet{ID: id, Provider: providerID, Name: name, ProviderDef: provider}
		graph.commandSets = append(graph.commandSets, resolved)
		graph.commandSetByID[id] = resolved
	}
	return graph, nil
}

func (g *Graph) Providers() []providerapi.Package {
	if g == nil {
		return nil
	}
	ids := append([]string(nil), g.providerOrder...)
	if len(ids) == 0 && len(g.providers) > 0 {
		for id := range g.providers {
			ids = append(ids, id)
		}
		sort.Strings(ids)
	}
	out := make([]providerapi.Package, 0, len(ids))
	for _, id := range ids {
		out = append(out, g.providers[id])
	}
	return out
}

func (g *Graph) RuntimeModules() []ResolvedRuntimeModule {
	if g == nil {
		return nil
	}
	return append([]ResolvedRuntimeModule(nil), g.modules...)
}

func (g *Graph) RuntimeModuleAliases() []string {
	if g == nil {
		return nil
	}
	aliases := make([]string, 0, len(g.moduleAliases))
	for alias := range g.moduleAliases {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

func (g *Graph) ResolveAlias(alias string) (ResolvedRuntimeModule, bool) {
	if g == nil {
		return ResolvedRuntimeModule{}, false
	}
	module, ok := g.moduleAliases[strings.TrimSpace(alias)]
	return module, ok
}

func (g *Graph) CommandSets() []ResolvedCommandSet {
	if g == nil {
		return nil
	}
	return append([]ResolvedCommandSet(nil), g.commandSets...)
}

func (g *Graph) TypeScriptModules(strict bool) ([]*spec.Module, error) {
	if g == nil {
		return nil, nil
	}
	out := []*spec.Module{}
	for _, module := range g.modules {
		if module.Module.TypeScript == nil {
			if strict {
				return nil, fmt.Errorf("runtime module %s.%s as %q has no TypeScript descriptor", module.Provider, module.Name, module.Alias)
			}
			continue
		}
		out = append(out, module.Module.TypeScript)
	}
	return out, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
