package app

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type jsVerbSourceSet struct {
	providers       *providerapi.ProviderRegistry
	embeddedJSVerbs fs.FS
	sources         []SourcePlan
	runtimeAliases  []string
}

func newJSVerbSourceSet(providers *providerapi.ProviderRegistry, embeddedJSVerbs fs.FS, sources []SourcePlan, runtimeAliases []string) *jsVerbSourceSet {
	return &jsVerbSourceSet{
		providers:       providers,
		embeddedJSVerbs: embeddedJSVerbs,
		sources:         append([]SourcePlan(nil), sources...),
		runtimeAliases:  appendUniqueStrings(nil, runtimeAliases...),
	}
}

func (s *jsVerbSourceSet) ListJSVerbSources() []providerapi.JSVerbSourceDescriptor {
	if s == nil || len(s.sources) == 0 {
		return nil
	}
	out := make([]providerapi.JSVerbSourceDescriptor, 0, len(s.sources))
	for _, source := range s.sources {
		out = append(out, providerapi.JSVerbSourceDescriptor{
			ID:         source.ID,
			Path:       source.Path,
			Embed:      source.Embed,
			Provider:   source.ProviderID(),
			Source:     source.Source,
			Include:    append([]string(nil), source.Include...),
			Exclude:    append([]string(nil), source.Exclude...),
			Extensions: append([]string(nil), source.Extensions...),
			TypeScript: providerTypeScriptDescriptor(source.TypeScript),
		})
	}
	return out
}

func providerTypeScriptDescriptor(spec *TypeScriptPlan) *providerapi.TypeScriptDescriptor {
	if spec == nil {
		return nil
	}
	return &providerapi.TypeScriptDescriptor{
		Enabled:      spec.Enabled,
		Bundle:       spec.Bundle,
		Target:       spec.Target,
		Format:       spec.Format,
		Platform:     spec.Platform,
		Tsconfig:     spec.Tsconfig,
		Sourcemap:    spec.Sourcemap,
		External:     append([]string(nil), spec.External...),
		Define:       cloneStringMap(spec.Define),
		CheckCommand: append([]string(nil), spec.CheckCommand...),
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (s *jsVerbSourceSet) ScanJSVerbSource(id string) (*jsverbs.Registry, error) {
	if s == nil {
		return nil, fmt.Errorf("jsverb source set is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("jsverb source id is required")
	}
	for _, source := range s.sources {
		if source.ID != id {
			continue
		}
		return scanVerbSource(s.providers, s.embeddedJSVerbs, source, sourceGraphRuntimeAliases(s.runtimeAliases))
	}
	return nil, fmt.Errorf("unknown jsverb source %q", id)
}

func (s *jsVerbSourceSet) ScanAllJSVerbSources() ([]*jsverbs.Registry, error) {
	if s == nil || len(s.sources) == 0 {
		return nil, nil
	}
	registries := make([]*jsverbs.Registry, 0, len(s.sources))
	for _, source := range s.sources {
		registry, err := scanVerbSource(s.providers, s.embeddedJSVerbs, source, sourceGraphRuntimeAliases(s.runtimeAliases))
		if err != nil {
			return nil, err
		}
		if registry == nil {
			continue
		}
		registries = append(registries, registry)
	}
	return registries, nil
}
