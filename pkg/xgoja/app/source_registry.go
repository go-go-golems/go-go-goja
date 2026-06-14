package app

import (
	"io/fs"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type SourceRegistry struct {
	providers       *providerapi.ProviderRegistry
	embeddedJSVerbs fs.FS
	sources         []SourcePlan
	runtimeAliases  []string
}

var _ providerapi.SourceRegistry = (*SourceRegistry)(nil)

func NewSourceRegistry(providers *providerapi.ProviderRegistry, embeddedJSVerbs fs.FS, sources []SourcePlan) *SourceRegistry {
	return NewSourceRegistryWithRuntimeAliases(providers, embeddedJSVerbs, sources, nil)
}

func NewSourceRegistryWithRuntimeAliases(providers *providerapi.ProviderRegistry, embeddedJSVerbs fs.FS, sources []SourcePlan, runtimeAliases []string) *SourceRegistry {
	return &SourceRegistry{providers: providers, embeddedJSVerbs: embeddedJSVerbs, sources: append([]SourcePlan(nil), sources...), runtimeAliases: appendUniqueStrings(nil, runtimeAliases...)}
}

func (r *SourceRegistry) ListSources() []providerapi.RuntimeSourceDescriptor {
	if r == nil || len(r.sources) == 0 {
		return nil
	}
	out := make([]providerapi.RuntimeSourceDescriptor, 0, len(r.sources))
	for _, source := range r.sources {
		out = append(out, runtimeSourceDescriptor(source))
	}
	return out
}

func (r *SourceRegistry) ListSourcesByKind(kind providerapi.RuntimeSourceKind) []providerapi.RuntimeSourceDescriptor {
	if r == nil || len(r.sources) == 0 {
		return nil
	}
	out := make([]providerapi.RuntimeSourceDescriptor, 0, len(r.sources))
	for _, source := range r.sources {
		if providerSourceKind(source.Kind) == kind {
			out = append(out, runtimeSourceDescriptor(source))
		}
	}
	return out
}

func (r *SourceRegistry) SourceByID(id string) (providerapi.RuntimeSourceDescriptor, bool) {
	id = strings.TrimSpace(id)
	if r == nil || id == "" {
		return providerapi.RuntimeSourceDescriptor{}, false
	}
	for _, source := range r.sources {
		if source.ID == id {
			return runtimeSourceDescriptor(source), true
		}
	}
	return providerapi.RuntimeSourceDescriptor{}, false
}

func (r *SourceRegistry) JSVerbs() providerapi.JSVerbSourceSet {
	if r == nil {
		return nil
	}
	return newJSVerbSourceSet(r.providers, r.embeddedJSVerbs, filterSourcesByKind(r.sources, SourceKindJSVerbs), r.runtimeAliases)
}

func (r *SourceRegistry) scanJSVerbSource(id string, runtimeAliases []string) (*jsverbs.Registry, error) {
	if r == nil {
		return nil, nil
	}
	for _, source := range r.sources {
		if source.ID == id && source.Kind == SourceKindJSVerbs {
			return scanVerbSource(r.providers, r.embeddedJSVerbs, source, runtimeAliases)
		}
	}
	return nil, nil
}

func (r *SourceRegistry) Scoped(sourceIDs []string) *SourceRegistry {
	if r == nil {
		return nil
	}
	wanted := map[string]struct{}{}
	for _, id := range sourceIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			wanted[id] = struct{}{}
		}
	}
	filtered := make([]SourcePlan, 0, len(r.sources))
	for _, source := range r.sources {
		if _, ok := wanted[source.ID]; ok {
			filtered = append(filtered, source)
		}
	}
	return NewSourceRegistryWithRuntimeAliases(r.providers, r.embeddedJSVerbs, filtered, r.runtimeAliases)
}

func (r *SourceRegistry) ScopedWithRuntimeAliases(sourceIDs []string, runtimeAliases []string) *SourceRegistry {
	scoped := r.Scoped(sourceIDs)
	if scoped == nil {
		return nil
	}
	scoped.runtimeAliases = appendUniqueStrings(nil, runtimeAliases...)
	return scoped
}

func filterSourcesByKind(sources []SourcePlan, kind SourceKind) []SourcePlan {
	out := make([]SourcePlan, 0, len(sources))
	for _, source := range sources {
		if source.Kind == kind {
			out = append(out, source)
		}
	}
	return out
}

func sourcePlansFromDescriptors(descriptors []providerapi.RuntimeSourceDescriptor) []SourcePlan {
	out := make([]SourcePlan, 0, len(descriptors))
	for _, descriptor := range descriptors {
		out = append(out, SourcePlan{ID: descriptor.ID, Kind: appSourceKind(descriptor.Kind), Path: descriptor.Path, Embed: descriptor.Embed, Provider: descriptor.Provider, Source: descriptor.Source, Include: append([]string(nil), descriptor.Include...), Exclude: append([]string(nil), descriptor.Exclude...), Extensions: append([]string(nil), descriptor.Extensions...), TypeScript: appTypeScriptPlan(descriptor.TypeScript)})
	}
	return out
}

func runtimeSourceDescriptor(source SourcePlan) providerapi.RuntimeSourceDescriptor {
	return providerapi.RuntimeSourceDescriptor{
		ID:         source.ID,
		Kind:       providerSourceKind(source.Kind),
		Path:       source.Path,
		Embed:      source.Embed,
		Provider:   source.ProviderID(),
		Source:     source.Source,
		Include:    append([]string(nil), source.Include...),
		Exclude:    append([]string(nil), source.Exclude...),
		Extensions: append([]string(nil), source.Extensions...),
		TypeScript: providerTypeScriptDescriptor(source.TypeScript),
	}
}

func appTypeScriptPlan(spec *providerapi.TypeScriptDescriptor) *TypeScriptPlan {
	if spec == nil {
		return nil
	}
	return &TypeScriptPlan{Enabled: spec.Enabled, Bundle: spec.Bundle, Target: spec.Target, Format: spec.Format, Platform: spec.Platform, Tsconfig: spec.Tsconfig, Sourcemap: spec.Sourcemap, External: append([]string(nil), spec.External...), Define: cloneStringMap(spec.Define), CheckCommand: append([]string(nil), spec.CheckCommand...)}
}

func appSourceKind(kind providerapi.RuntimeSourceKind) SourceKind {
	switch kind {
	case providerapi.RuntimeSourceKindJSVerbs:
		return SourceKindJSVerbs
	case providerapi.RuntimeSourceKindScript:
		return SourceKindScript
	case providerapi.RuntimeSourceKindAssets:
		return SourceKindAssets
	case providerapi.RuntimeSourceKindHelp:
		return SourceKindHelp
	default:
		return SourceKind(kind)
	}
}

func providerSourceKind(kind SourceKind) providerapi.RuntimeSourceKind {
	switch kind {
	case SourceKindJSVerbs:
		return providerapi.RuntimeSourceKindJSVerbs
	case SourceKindScript:
		return providerapi.RuntimeSourceKindScript
	case SourceKindAssets:
		return providerapi.RuntimeSourceKindAssets
	case SourceKindHelp:
		return providerapi.RuntimeSourceKindHelp
	default:
		return providerapi.RuntimeSourceKind(kind)
	}
}
