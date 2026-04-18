package glazed

import (
	"context"
	"testing"

	"github.com/go-go-golems/glazed/pkg/help"
	helpmodel "github.com/go-go-golems/glazed/pkg/help/model"
	"github.com/go-go-golems/go-go-goja/pkg/docaccess"
)

func TestProviderSearchAndGet(t *testing.T) {
	hs := help.NewHelpSystem()
	hs.AddSection(&helpmodel.Section{
		Slug:        "repl-usage",
		Title:       "REPL Usage",
		Short:       "How to use the REPL",
		Content:     "Body",
		Topics:      []string{"goja", "repl"},
		Commands:    []string{"repl"},
		SectionType: helpmodel.SectionGeneralTopic,
	})

	provider, err := NewProvider("default-help", "Default Help", "", hs)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	entries, err := provider.Search(context.Background(), docaccess.Query{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(entries) != 1 || entries[0].Ref.ID != "repl-usage" {
		t.Fatalf("entries = %#v", entries)
	}

	entry, err := provider.Get(context.Background(), entries[0].Ref)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if entry.Title != "REPL Usage" {
		t.Fatalf("title = %q", entry.Title)
	}
}
