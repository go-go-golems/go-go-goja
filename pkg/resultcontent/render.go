package resultcontent

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
)

// Ref describes one content item without its bytes. Refs are safe to persist.
type Ref struct {
	// Index is the position of the item in Rendering.Content, or -1 when the
	// item was dropped or the provider produced nothing deliverable.
	Index int `json:"index"`
	// Provider is the 1-based provider number used in "[content N: ...]".
	Provider int    `json:"provider"`
	MIMEType string `json:"mimeType,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Bytes    int    `json:"bytes"`
	// Dropped explains why the item is not in Rendering.Content.
	Dropped string `json:"dropped,omitempty"`
}

// Rendering is the output of Render.
type Rendering struct {
	Content []Content
	Refs    []Ref
}

// Dropped returns the refs whose content was not delivered.
func (r Rendering) Dropped() []Ref {
	out := []Ref{}
	for _, ref := range r.Refs {
		if ref.Index < 0 {
			out = append(out, ref)
		}
	}
	return out
}

// Notes returns one human-readable line per dropped item, suitable for
// appending to a text result.
func (r Rendering) Notes() []string {
	dropped := r.Dropped()
	notes := make([]string, 0, len(dropped))
	for _, ref := range dropped {
		label := fmt.Sprintf("content %d", ref.Provider)
		if ref.MIMEType != "" {
			label += " (" + ref.MIMEType
			if ref.Bytes > 0 {
				label += fmt.Sprintf(", %d bytes", ref.Bytes)
			}
			label += ")"
		}
		notes = append(notes, label+" dropped: "+ref.Dropped)
	}
	return notes
}

// Render calls each provider in order and keeps the content that fits the
// item and byte limits. It must not run on the runtime owner goroutine, since
// providers may do expensive work. Providers after the item limit is reached
// are not called. Provider errors and panics are reported as dropped refs.
func Render(ctx context.Context, providers []Provider, limits Limits) Rendering {
	limits = limits.Normalize()
	out := Rendering{Content: []Content{}, Refs: []Ref{}}
	totalBytes := 0
	for i, p := range providers {
		n := i + 1
		if p == nil {
			continue
		}
		if len(out.Content) >= limits.MaxItems {
			out.Refs = append(out.Refs, Ref{Index: -1, Provider: n, Dropped: fmt.Sprintf("not rendered: item limit (%d) reached", limits.MaxItems)})
			continue
		}
		if err := ctx.Err(); err != nil {
			out.Refs = append(out.Refs, Ref{Index: -1, Provider: n, Dropped: "not rendered: " + err.Error()})
			continue
		}
		items, err := callProvider(ctx, p)
		if err != nil {
			out.Refs = append(out.Refs, Ref{Index: -1, Provider: n, Dropped: "render error: " + err.Error()})
			continue
		}
		for _, item := range items {
			ref := Ref{Index: -1, Provider: n, MIMEType: item.MIMEType, Caption: item.Caption, Bytes: len(item.Data)}
			switch {
			case len(out.Content) >= limits.MaxItems:
				ref.Dropped = fmt.Sprintf("item limit (%d) reached", limits.MaxItems)
			case totalBytes+len(item.Data) > limits.MaxTotalBytes:
				ref.Dropped = fmt.Sprintf("byte limit (%d) exceeded", limits.MaxTotalBytes)
			default:
				ref.Index = len(out.Content)
				totalBytes += len(item.Data)
				out.Content = append(out.Content, item)
			}
			out.Refs = append(out.Refs, ref)
		}
	}
	return out
}

func callProvider(ctx context.Context, p Provider) ([]Content, error) {
	var (
		items []Content
		err   error
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				items, err = nil, errors.Errorf("provider panicked: %v", r)
			}
		}()
		items, err = p.ResultContent(ctx)
	}()
	return items, err
}
