package app

import (
	"context"
	"io/fs"
	"path"
	"testing"
	"testing/fstest"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestAssetStoreResolveAsset(t *testing.T) {
	assetFS := fstest.MapFS{
		"xgoja_embed/assets/app/config.json": &fstest.MapFile{Data: []byte(`{"ok":true}`)},
	}
	store := NewAssetStore(assetFS, &Spec{Assets: []AssetSourceSpec{{ID: "app", Path: "/xgoja_embed/assets/app", Embed: true}}})

	fsys, root, ok := store.ResolveAsset("app")
	if !ok {
		t.Fatal("expected asset to resolve")
	}
	if root != "xgoja_embed/assets/app" {
		t.Fatalf("root = %q", root)
	}
	data, err := fs.ReadFile(fsys, path.Join(root, "config.json"))
	if err != nil {
		t.Fatalf("read resolved asset: %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Fatalf("asset data = %q", data)
	}
}

func TestRuntimeFactoryPassesRuntimeOwnerToModules(t *testing.T) {
	spec := &Spec{
		Runtimes: map[string]RuntimeSpec{
			"main": {Modules: []ModuleInstanceSpec{{Package: "fixture", Name: "owner-check", As: "owner-check"}}},
		},
	}
	seen := false
	registry := providerapi.NewRegistry()
	if err := registry.Package("fixture", providerapi.Module{
		Name: "owner-check",
		New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
			if ctx.RuntimeOwner == nil {
				t.Fatalf("expected module context runtime owner")
			}
			seen = true
			return func(vm *goja.Runtime, module *goja.Object) {}, nil
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	rt, err := NewRuntimeFactory(registry, spec).NewRuntime(context.Background(), "main")
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	if !seen {
		t.Fatal("module factory did not observe runtime owner")
	}
}

func TestRuntimeFactoryPassesHostServicesToModules(t *testing.T) {
	assetFS := fstest.MapFS{
		"xgoja_embed/assets/app/config.json": &fstest.MapFile{Data: []byte(`{"ok":true}`)},
	}
	spec := &Spec{
		Assets: []AssetSourceSpec{{ID: "app", Path: "xgoja_embed/assets/app", Embed: true}},
		Runtimes: map[string]RuntimeSpec{
			"main": {Modules: []ModuleInstanceSpec{{Package: "fixture", Name: "asset-check", As: "asset-check"}}},
		},
	}
	seen := false
	registry := providerapi.NewRegistry()
	if err := registry.Package("fixture", providerapi.Module{
		Name: "asset-check",
		New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
			resolver := ctx.Host.AssetResolver()
			fsys, root, ok := resolver.ResolveAsset("app")
			if !ok {
				t.Fatalf("expected module context host to resolve asset")
			}
			if _, err := fs.ReadFile(fsys, path.Join(root, "config.json")); err != nil {
				t.Fatalf("read asset through module context host: %v", err)
			}
			seen = true
			return func(vm *goja.Runtime, module *goja.Object) {}, nil
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	host := NewHostWithOptions(registry, spec, HostOptions{EmbeddedAssets: assetFS})
	rt, err := host.Factory.NewRuntime(context.Background(), "main")
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	if !seen {
		t.Fatal("module factory did not observe host services")
	}
}
