package docaccess

import (
	"context"
	"testing"
)

type stubProvider struct {
	sourceID string
	entries  []Entry
}

func (p *stubProvider) Descriptor() SourceDescriptor {
	return SourceDescriptor{ID: p.sourceID, Kind: SourceKindPlugin}
}

func (p *stubProvider) List(context.Context) ([]EntryRef, error) {
	out := make([]EntryRef, 0, len(p.entries))
	for _, entry := range p.entries {
		out = append(out, entry.Ref)
	}
	return out, nil
}

func (p *stubProvider) Get(_ context.Context, ref EntryRef) (*Entry, error) {
	for _, entry := range p.entries {
		if entry.Ref == ref {
			copy_ := entry
			return &copy_, nil
		}
	}
	return nil, ErrEntryNotFound
}

func (p *stubProvider) Search(context.Context, Query) ([]Entry, error) {
	return append([]Entry(nil), p.entries...), nil
}

func TestHubRegisterAndSearch(t *testing.T) {
	hub := NewHub()
	if err := hub.Register(&stubProvider{
		sourceID: "test",
		entries: []Entry{
			{
				Ref:     EntryRef{SourceID: "test", Kind: "symbol", ID: "alpha"},
				Title:   "Alpha",
				Summary: "First symbol",
				Topics:  []string{"example"},
			},
			{
				Ref:     EntryRef{SourceID: "test", Kind: "symbol", ID: "beta"},
				Title:   "Beta",
				Summary: "Second symbol",
			},
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	results, err := hub.Search(context.Background(), Query{Text: "first"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 || results[0].Ref.ID != "alpha" {
		t.Fatalf("results = %#v", results)
	}

	entry, err := hub.FindByID("test", "symbol", "beta")
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if entry.Title != "Beta" {
		t.Fatalf("entry title = %q, want Beta", entry.Title)
	}
}
