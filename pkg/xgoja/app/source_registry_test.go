package app

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestSourceRegistryScopedEmptyReturnsEmptyRegistry(t *testing.T) {
	registry := NewSourceRegistry(nil, nil, []SourcePlan{
		{ID: "public", Kind: SourceKindJSVerbs, Path: "public"},
		{ID: "admin", Kind: SourceKindJSVerbs, Path: "admin"},
	})

	scoped := registry.Scoped(nil)
	if scoped == nil {
		t.Fatal("expected scoped registry")
	}
	if got := scoped.ListSources(); len(got) != 0 {
		t.Fatalf("empty scope sources = %#v, want none", got)
	}
	if got := scoped.ListSourcesByKind(providerapi.RuntimeSourceKindJSVerbs); len(got) != 0 {
		t.Fatalf("empty scope jsverb sources = %#v, want none", got)
	}
	if got := scoped.JSVerbs().ListJSVerbSources(); len(got) != 0 {
		t.Fatalf("empty scope JSVerbs() sources = %#v, want none", got)
	}
}

func TestSourceRegistryScopedFiltersExplicitSourceIDs(t *testing.T) {
	registry := NewSourceRegistry(nil, nil, []SourcePlan{
		{ID: "public", Kind: SourceKindJSVerbs, Path: "public"},
		{ID: "admin", Kind: SourceKindJSVerbs, Path: "admin"},
	})

	scoped := registry.Scoped([]string{"public"})
	if _, ok := scoped.SourceByID("public"); !ok {
		t.Fatal("expected selected public source")
	}
	if _, ok := scoped.SourceByID("admin"); ok {
		t.Fatal("did not expect unselected admin source")
	}
	jsverbSources := scoped.JSVerbs().ListJSVerbSources()
	if len(jsverbSources) != 1 || jsverbSources[0].ID != "public" {
		t.Fatalf("jsverb sources = %#v, want only public", jsverbSources)
	}
}
