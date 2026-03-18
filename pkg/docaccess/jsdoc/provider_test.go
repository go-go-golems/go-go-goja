package jsdoc

import (
	"context"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/docaccess"
	jsdocmodel "github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
)

func TestProviderSearchAndGet(t *testing.T) {
	store := jsdocmodel.NewDocStore()
	store.AddFile(&jsdocmodel.FileDoc{
		FilePath: "math.js",
		Package: &jsdocmodel.Package{
			Name:        "math",
			Title:       "Math",
			Description: "Math helpers",
			Prose:       "Long package prose",
			SourceFile:  "math.js",
		},
		Symbols: []*jsdocmodel.SymbolDoc{{
			Name:       "smoothstep",
			Summary:    "Smooth interpolation",
			Prose:      "Detailed symbol prose",
			Concepts:   []string{"interpolation"},
			SourceFile: "math.js",
			Line:       10,
		}},
	})

	provider, err := NewProvider("workspace-jsdoc", "Workspace JS Docs", "", store)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	entries, err := provider.Search(context.Background(), docaccess.Query{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %#v", entries)
	}

	entry, err := provider.Get(context.Background(), entries[0].Ref)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if entry.Title == "" {
		t.Fatalf("empty title for entry %#v", entry)
	}
}
