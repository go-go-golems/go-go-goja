package jsdoc

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/docaccess"
	jsdocmodel "github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
)

const (
	EntryKindPackage = "package"
	EntryKindSymbol  = "symbol"
	EntryKindExample = "example"
)

type Provider struct {
	sourceID string
	title    string
	summary  string
	store    *jsdocmodel.DocStore
}

func NewProvider(sourceID, title, summary string, store *jsdocmodel.DocStore) (*Provider, error) {
	if store == nil {
		return nil, fmt.Errorf("jsdoc store is nil")
	}
	if sourceID == "" {
		sourceID = "jsdoc"
	}
	if title == "" {
		title = "JavaScript Docs"
	}
	return &Provider{
		sourceID: sourceID,
		title:    title,
		summary:  summary,
		store:    store,
	}, nil
}

func (p *Provider) Descriptor() docaccess.SourceDescriptor {
	return docaccess.SourceDescriptor{
		ID:      p.sourceID,
		Kind:    docaccess.SourceKindJSDoc,
		Title:   p.title,
		Summary: p.summary,
	}
}

func (p *Provider) List(ctx context.Context) ([]docaccess.EntryRef, error) {
	entries, err := p.Search(ctx, docaccess.Query{})
	if err != nil {
		return nil, err
	}
	out := make([]docaccess.EntryRef, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Ref)
	}
	return out, nil
}

func (p *Provider) Get(_ context.Context, ref docaccess.EntryRef) (*docaccess.Entry, error) {
	if ref.SourceID != p.sourceID {
		return nil, docaccess.ErrEntryNotFound
	}

	switch ref.Kind {
	case EntryKindPackage:
		pkg := p.store.ByPackage[ref.ID]
		if pkg == nil {
			return nil, docaccess.ErrEntryNotFound
		}
		entry := packageEntry(p.sourceID, pkg)
		return &entry, nil
	case EntryKindSymbol:
		sym := p.store.BySymbol[ref.ID]
		if sym == nil {
			return nil, docaccess.ErrEntryNotFound
		}
		entry := symbolEntry(p.sourceID, sym)
		return &entry, nil
	case EntryKindExample:
		example := p.store.ByExample[ref.ID]
		if example == nil {
			return nil, docaccess.ErrEntryNotFound
		}
		entry := exampleEntry(p.sourceID, example)
		return &entry, nil
	default:
		return nil, docaccess.ErrEntryNotFound
	}
}

func (p *Provider) Search(_ context.Context, _ docaccess.Query) ([]docaccess.Entry, error) {
	out := make([]docaccess.Entry, 0, len(p.store.ByPackage)+len(p.store.BySymbol)+len(p.store.ByExample))
	for _, pkg := range p.store.ByPackage {
		out = append(out, packageEntry(p.sourceID, pkg))
	}
	for _, sym := range p.store.BySymbol {
		out = append(out, symbolEntry(p.sourceID, sym))
	}
	for _, example := range p.store.ByExample {
		out = append(out, exampleEntry(p.sourceID, example))
	}
	return out, nil
}

func packageEntry(sourceID string, pkg *jsdocmodel.Package) docaccess.Entry {
	return docaccess.Entry{
		Ref:       docaccess.EntryRef{SourceID: sourceID, Kind: EntryKindPackage, ID: pkg.Name},
		Title:     pkg.Title,
		Summary:   docaccessSummary(pkg.Description, pkg.Prose),
		Body:      pkg.Prose,
		Path:      pkg.SourceFile,
		KindLabel: "Package",
		Metadata: map[string]any{
			"name":        pkg.Name,
			"category":    pkg.Category,
			"guide":       pkg.Guide,
			"version":     pkg.Version,
			"description": pkg.Description,
			"seeAlso":     append([]string(nil), pkg.SeeAlso...),
		},
	}
}

func symbolEntry(sourceID string, sym *jsdocmodel.SymbolDoc) docaccess.Entry {
	related := make([]docaccess.EntryRef, 0, len(sym.Related))
	for _, item := range sym.Related {
		related = append(related, docaccess.EntryRef{SourceID: sourceID, Kind: EntryKindSymbol, ID: item})
	}

	return docaccess.Entry{
		Ref:       docaccess.EntryRef{SourceID: sourceID, Kind: EntryKindSymbol, ID: sym.Name},
		Title:     sym.Name,
		Summary:   docaccessSummary(sym.Summary, sym.Prose),
		Body:      sym.Prose,
		Topics:    append([]string(nil), sym.Concepts...),
		Tags:      append([]string(nil), sym.Tags...),
		Path:      sym.SourceFile,
		KindLabel: "Symbol",
		Related:   related,
		Metadata: map[string]any{
			"params":   sym.Params,
			"returns":  sym.Returns,
			"docpage":  sym.DocPage,
			"line":     sym.Line,
			"concepts": append([]string(nil), sym.Concepts...),
		},
	}
}

func exampleEntry(sourceID string, example *jsdocmodel.Example) docaccess.Entry {
	related := make([]docaccess.EntryRef, 0, len(example.Symbols))
	for _, item := range example.Symbols {
		related = append(related, docaccess.EntryRef{SourceID: sourceID, Kind: EntryKindSymbol, ID: item})
	}

	return docaccess.Entry{
		Ref:       docaccess.EntryRef{SourceID: sourceID, Kind: EntryKindExample, ID: example.ID},
		Title:     example.Title,
		Summary:   docaccessSummary(example.Title, example.Body),
		Body:      example.Body,
		Topics:    append([]string(nil), example.Concepts...),
		Tags:      append([]string(nil), example.Tags...),
		Path:      example.SourceFile,
		KindLabel: "Example",
		Related:   related,
		Metadata: map[string]any{
			"symbols":  append([]string(nil), example.Symbols...),
			"docpage":  example.DocPage,
			"line":     example.Line,
			"concepts": append([]string(nil), example.Concepts...),
		},
	}
}

func docaccessSummary(summary, body string) string {
	return summaryFromBody(summary, body)
}

func summaryFromBody(summary, body string) string {
	summary = strings.TrimSpace(summary)
	if summary != "" {
		return summary
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	if idx := strings.Index(body, "\n\n"); idx >= 0 {
		return strings.TrimSpace(body[:idx])
	}
	if idx := strings.IndexByte(body, '\n'); idx >= 0 {
		return strings.TrimSpace(body[:idx])
	}
	return body
}
