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
					Name:    "get",
					Summary: "Return the current value",
					Doc:     "Return the current value",
					Tags:    []string{"lookup", "kv"},
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
	if entry.Summary != "Return the current value" {
		t.Fatalf("summary = %q", entry.Summary)
	}
	if len(entry.Tags) != 2 || entry.Tags[0] != "lookup" || entry.Tags[1] != "kv" {
		t.Fatalf("tags = %#v", entry.Tags)
	}
}
