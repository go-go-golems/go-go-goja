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
	sources         []JSVerbSourceSpec
}

func newJSVerbSourceSet(providers *providerapi.ProviderRegistry, embeddedJSVerbs fs.FS, sources []JSVerbSourceSpec) *jsVerbSourceSet {
	return &jsVerbSourceSet{
		providers:       providers,
		embeddedJSVerbs: embeddedJSVerbs,
		sources:         append([]JSVerbSourceSpec(nil), sources...),
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
			Package:    source.Package,
			Source:     source.Source,
			Include:    append([]string(nil), source.Include...),
			Exclude:    append([]string(nil), source.Exclude...),
			Extensions: append([]string(nil), source.Extensions...),
		})
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
		return scanVerbSource(s.providers, s.embeddedJSVerbs, source)
	}
	return nil, fmt.Errorf("unknown jsverb source %q", id)
}

func (s *jsVerbSourceSet) ScanAllJSVerbSources() ([]*jsverbs.Registry, error) {
	if s == nil || len(s.sources) == 0 {
		return nil, nil
	}
	registries := make([]*jsverbs.Registry, 0, len(s.sources))
	for _, source := range s.sources {
		registry, err := scanVerbSource(s.providers, s.embeddedJSVerbs, source)
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
