package app

import (
	"io/fs"
	"path"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

var _ providerapi.AssetResolver = (*AssetStore)(nil)
var _ providerapi.HostServices = HostServices{}

type AssetStore struct {
	fsys   fs.FS
	assets map[string]AssetSourceSpec
}

type HostServices struct {
	Assets *AssetStore
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
