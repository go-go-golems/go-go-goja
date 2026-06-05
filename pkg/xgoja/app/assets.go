package app

import (
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
	assets map[string]AssetSourceSpec
}

type HostServices struct {
	Assets   *AssetStore
	Services map[string][]any
}

func NewAssetStore(fsys fs.FS, runtimeSpec *RuntimeSpec) *AssetStore {
	store := &AssetStore{
		fsys:   fsys,
		assets: map[string]AssetSourceSpec{},
	}
	if runtimeSpec == nil {
		return store
	}
	for _, asset := range runtimeSpec.Assets {
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
	return s.Assets
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
