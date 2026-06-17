package app

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

var _ providerapi.AssetResolver = (*AssetStore)(nil)
var _ providerapi.HostServices = HostServices{}
var _ providerapi.HostServiceLookup = HostServices{}

type AssetStore struct {
	fsys   fs.FS
	assets map[string]SourcePlan
}

type HostServices struct {
	Assets   *AssetStore
	Services map[string][]any
}

func NewAssetStore(fsys fs.FS, runtimePlan *RuntimePlan) *AssetStore {
	registry := NewSourceRegistry(nil, nil, runtimePlan.allSources())
	return NewAssetStoreFromSources(fsys, registry.ListSourcesByKind(providerapi.RuntimeSourceKindAssets))
}

func NewAssetStoreFromSources(fsys fs.FS, assets []providerapi.RuntimeSourceDescriptor) *AssetStore {
	store := &AssetStore{
		fsys:   fsys,
		assets: map[string]SourcePlan{},
	}
	for _, descriptor := range assets {
		asset := SourcePlan{ID: descriptor.ID, Kind: SourceKindAssets, Path: descriptor.Path, Embed: descriptor.Embed, Provider: descriptor.Provider, Source: descriptor.Source}
		id := strings.TrimSpace(asset.ID)
		if id == "" {
			continue
		}
		asset.ID = id
		asset.Path = cleanEmbeddedAssetPath(asset.Path)
		store.assets[id] = asset
	}
	return store
}

func (s *AssetStore) ResolveAsset(id string) (fs.FS, string, bool) {
	if s == nil || s.fsys == nil {
		return nil, "", false
	}
	asset, ok := s.assets[strings.TrimSpace(id)]
	if !ok || !asset.Embed || strings.TrimSpace(asset.Path) == "" {
		return nil, "", false
	}
	return s.fsys, asset.Path, true
}

func (s HostServices) AssetResolver() providerapi.AssetResolver {
	if s.Assets == nil {
		return nil
	}
	return s.Assets
}

func (s *HostServices) SetHostService(key string, value any) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("host service key is required")
	}
	if value == nil {
		return fmt.Errorf("host service %q value is nil", key)
	}
	if s.Services == nil {
		s.Services = map[string][]any{}
	}
	s.Services[key] = []any{value}
	return nil
}

func (s *HostServices) AddHostService(key string, value any) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("host service key is required")
	}
	if value == nil {
		return fmt.Errorf("host service %q value is nil", key)
	}
	if s.Services == nil {
		s.Services = map[string][]any{}
	}
	s.Services[key] = append(s.Services[key], value)
	return nil
}

func (s HostServices) HostService(key string) (any, bool) {
	values := s.HostServiceValues(key)
	switch len(values) {
	case 0:
		return nil, false
	case 1:
		return values[0], true
	default:
		return values, true
	}
}

func (s HostServices) HostServiceValues(key string) []any {
	key = strings.TrimSpace(key)
	if key == "" || s.Services == nil {
		return nil
	}
	values := s.Services[key]
	return append([]any(nil), values...)
}

func cleanEmbeddedAssetPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	cleaned := path.Clean(strings.TrimPrefix(raw, "/"))
	if cleaned == "." {
		return ""
	}
	return cleaned
}
