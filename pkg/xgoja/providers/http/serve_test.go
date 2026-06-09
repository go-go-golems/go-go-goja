package http

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestNewServeCommandSetRequiresJSVerbSources(t *testing.T) {
	_, ok := providerapi.NewProviderRegistry().ResolveCommandSetProvider(PackageID, "serve")
	if ok {
		t.Fatal("empty registry unexpectedly resolved serve provider")
	}
	_, err := newServeCommandSet(providerapi.CommandSetContext{RuntimeFactory: fakeRuntimeFactory{}})
	if err == nil {
		t.Fatal("expected missing jsverb source error")
	}
}

func TestNewServeCommandSetBuildsVerbCommandsWithHTTPSection(t *testing.T) {
	registry := scanServeTestRegistry(t)
	capability := newHTTPCapability()
	set, err := newServeCommandSet(providerapi.CommandSetContext{
		Name:           "serve",
		RuntimeFactory: fakeRuntimeFactory{},
		SelectedModules: []providerapi.ModuleDescriptor{{
			PackageID:           PackageID,
			ModuleID:            "express",
			As:                  "express",
			PackageCapabilities: []providerapi.PackageCapability{capability},
		}},
		JSVerbs: fakeJSVerbSourceSet{registries: []*jsverbs.Registry{registry}},
	})
	if err != nil {
		t.Fatalf("new serve command set: %v", err)
	}
	if len(set.Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(set.Commands))
	}
	desc := set.Commands[0].Description()
	if desc.Name != "demo" {
		t.Fatalf("command name = %q, want demo", desc.Name)
	}
	if desc.Parents[0] != "sites" {
		t.Fatalf("parents = %#v, want sites", desc.Parents)
	}
	if _, ok := desc.Schema.Get("http"); !ok {
		t.Fatalf("expected http section on serve command; schema=%#v", desc.Schema)
	}
	hotReloadSection, ok := desc.Schema.Get(serveHotReloadSectionSlug)
	if !ok {
		t.Fatalf("expected hot reload section on serve command; schema=%#v", desc.Schema)
	}
	for _, name := range []string{"hot-reload", "hot-reload-watch-root", "hot-reload-watch-ext", "hot-reload-smoke-path", "hot-reload-poll", "hot-reload-debounce", "hot-reload-close-grace", "hot-reload-status-path"} {
		if _, ok := hotReloadSection.GetDefinitions().Get(name); !ok {
			t.Fatalf("missing hot reload field %q", name)
		}
	}
}

func scanServeTestRegistry(t *testing.T) *jsverbs.Registry {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sites.js")
	source := `
__package__({ name: "sites" });
__verb__("demo", { name: "demo", short: "Serve demo", output: "text" });
function demo() {}
`
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("write verb: %v", err)
	}
	registry, err := jsverbs.ScanDir(dir)
	if err != nil {
		t.Fatalf("scan dir: %v", err)
	}
	return registry
}

type fakeRuntimeFactory struct{}

func (fakeRuntimeFactory) NewRuntime(context.Context, ...require.Option) (*engine.Runtime, error) {
	return nil, fmt.Errorf("not implemented")
}

func (fakeRuntimeFactory) NewRuntimeFromSections(context.Context, *values.Values, ...require.Option) (*engine.Runtime, error) {
	return nil, fmt.Errorf("not implemented")
}

type fakeJSVerbSourceSet struct {
	registries []*jsverbs.Registry
}

func (s fakeJSVerbSourceSet) ListJSVerbSources() []providerapi.JSVerbSourceDescriptor {
	return []providerapi.JSVerbSourceDescriptor{{ID: "fake"}}
}

func (s fakeJSVerbSourceSet) ScanJSVerbSource(id string) (*jsverbs.Registry, error) {
	if id != "fake" {
		return nil, fmt.Errorf("unknown fake source %q", id)
	}
	if len(s.registries) == 0 {
		return nil, nil
	}
	return s.registries[0], nil
}

func (s fakeJSVerbSourceSet) ScanAllJSVerbSources() ([]*jsverbs.Registry, error) {
	return s.registries, nil
}
