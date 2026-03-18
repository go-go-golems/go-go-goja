package plugin

import (
	"context"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/docaccess"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
)

func TestProviderExposesMethodDocs(t *testing.T) {
	provider, err := NewProvider("plugin-manifests", "", "", []host.LoadedModuleInfo{{
		Path: "/plugins/goja-plugin-examples-kv",
		Manifest: &contract.ModuleManifest{
			ModuleName: "plugin:examples:kv",
			Doc:        "Stateful plugin",
			Exports: []*contract.ExportSpec{{
				Name: "store",
				Kind: contract.ExportKind_EXPORT_KIND_OBJECT,
				Doc:  "Store operations",
				MethodSpecs: []*contract.MethodSpec{{
					Name: "get",
					Doc:  "Return the current value",
				}},
			}},
		},
	}})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	entry, err := provider.Get(context.Background(), docaccess.EntryRef{
		SourceID: "plugin-manifests",
		Kind:     EntryKindPluginMethod,
		ID:       "plugin:examples:kv/store.get",
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if entry.Body != "Return the current value" {
		t.Fatalf("body = %q", entry.Body)
	}
}
