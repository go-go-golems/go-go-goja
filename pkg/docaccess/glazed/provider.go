package glazed

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/help"
	helpmodel "github.com/go-go-golems/glazed/pkg/help/model"
	"github.com/go-go-golems/go-go-goja/pkg/docaccess"
)

const EntryKindHelpSection = "help-section"

type Provider struct {
	sourceID string
	title    string
	summary  string
	hs       *help.HelpSystem
}

func NewProvider(sourceID, title, summary string, hs *help.HelpSystem) (*Provider, error) {
	if hs == nil {
		return nil, fmt.Errorf("glazed help system is nil")
	}
	if sourceID == "" {
		sourceID = "default-help"
	}
	if title == "" {
		title = "Glazed Help"
	}
	return &Provider{
		sourceID: sourceID,
		title:    title,
		summary:  summary,
		hs:       hs,
	}, nil
}

func (p *Provider) Descriptor() docaccess.SourceDescriptor {
	return docaccess.SourceDescriptor{
		ID:      p.sourceID,
		Kind:    docaccess.SourceKindGlazedHelp,
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
	if ref.SourceID != p.sourceID || ref.Kind != EntryKindHelpSection {
		return nil, docaccess.ErrEntryNotFound
	}
	section, err := p.hs.GetSectionWithSlug(ref.ID)
	if err != nil || section == nil {
		return nil, docaccess.ErrEntryNotFound
	}
	entry := sectionEntry(p.sourceID, section)
	return &entry, nil
}

func (p *Provider) Search(_ context.Context, _ docaccess.Query) ([]docaccess.Entry, error) {
	sections, err := p.hs.QuerySections("")
	if err != nil {
		return nil, err
	}
	out := make([]docaccess.Entry, 0, len(sections))
	for _, section := range sections {
		if section == nil {
			continue
		}
		out = append(out, sectionEntry(p.sourceID, section))
	}
	return out, nil
}

func sectionEntry(sourceID string, section *helpmodel.Section) docaccess.Entry {
	return docaccess.Entry{
		Ref: docaccess.EntryRef{
			SourceID: sourceID,
			Kind:     EntryKindHelpSection,
			ID:       section.Slug,
		},
		Title:     section.Title,
		Summary:   section.Short,
		Body:      section.Content,
		Topics:    append([]string(nil), section.Topics...),
		KindLabel: "Help Section",
		Metadata: map[string]any{
			"slug":           section.Slug,
			"commands":       append([]string(nil), section.Commands...),
			"flags":          append([]string(nil), section.Flags...),
			"sectionType":    section.SectionType.String(),
			"isTopLevel":     section.IsTopLevel,
			"isTemplate":     section.IsTemplate,
			"showPerDefault": section.ShowPerDefault,
			"order":          section.Order,
		},
	}
}
