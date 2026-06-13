package app

import (
	"io/fs"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type SourceRegistry struct {
	providers       *providerapi.ProviderRegistry
	embeddedJSVerbs fs.FS
	sources         []SourcePlan
}

var _ providerapi.SourceRegistry = (*SourceRegistry)(nil)

func NewSourceRegistry(providers *providerapi.ProviderRegistry, embeddedJSVerbs fs.FS, sources []SourcePlan) *SourceRegistry {
	return &SourceRegistry{providers: providers, embeddedJSVerbs: embeddedJSVerbs, sources: append([]SourcePlan(nil), sources...)}
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
	return newJSVerbSourceSet(r.providers, r.embeddedJSVerbs, filterSourcesByKind(r.sources, SourceKindJSVerbs))
}

func (r *SourceRegistry) Scoped(sourceIDs []string) *SourceRegistry {
	if r == nil {
		return nil
	}
	if len(sourceIDs) == 0 {
		return NewSourceRegistry(r.providers, r.embeddedJSVerbs, r.sources)
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
	return NewSourceRegistry(r.providers, r.embeddedJSVerbs, filtered)
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
