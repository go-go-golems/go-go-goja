// Package resultcontent lets Go values returned from a JavaScript evaluation
// carry rich content (images, documents, plots) next to the textual result.
//
// A module returns a Go-backed object whose exported Go value implements
// Provider. After the evaluation settles, the host walks the returned value on
// the runtime owner goroutine with Extract, which finds the providers and
// renders the value as text with "[content N: ...]" placeholders. The host then
// calls Render off the owner goroutine to obtain the actual bytes, subject to
// item and byte limits.
package resultcontent

import (
	"context"
)

// Content is one rich item produced by a Provider.
type Content struct {
	// MIMEType is the media type of Data, for example "image/png".
	MIMEType string `json:"mimeType"`
	// Data is the raw (not base64-encoded) payload.
	Data []byte `json:"data,omitempty"`
	// Caption is a short human-readable label shown next to the content.
	Caption string `json:"caption,omitempty"`
}

// Provider is implemented by Go values that an evaluation may return to
// produce rich content. ResultContent is called off the runtime owner
// goroutine and must not touch the goja runtime.
type Provider interface {
	ResultContent(ctx context.Context) ([]Content, error)
}

// Describer may be implemented by a Provider to name itself cheaply in the
// text rendering without rendering its content. ResultPlaceholder is called on
// the runtime owner goroutine during Extract and must be fast.
type Describer interface {
	ResultPlaceholder() string
}

const (
	// DefaultMaxDepth is the default walk depth of Extract.
	DefaultMaxDepth = 8
	// DefaultMaxProperties is the default number of properties Extract visits.
	DefaultMaxProperties = 1000
	// DefaultMaxTextChars is the default size of the Extract text rendering.
	DefaultMaxTextChars = 16000
	// DefaultMaxItems is the default number of content items Render keeps.
	DefaultMaxItems = 8
	// DefaultMaxTotalBytes is the default total payload size Render keeps.
	DefaultMaxTotalBytes = 8 << 20
)

// Limits bounds both the walk (Extract) and the rendering (Render). Zero or
// negative fields select the defaults.
type Limits struct {
	MaxDepth      int
	MaxProperties int
	MaxTextChars  int
	MaxItems      int
	MaxTotalBytes int
}

// DefaultLimits returns the default limits.
func DefaultLimits() Limits {
	return Limits{
		MaxDepth:      DefaultMaxDepth,
		MaxProperties: DefaultMaxProperties,
		MaxTextChars:  DefaultMaxTextChars,
		MaxItems:      DefaultMaxItems,
		MaxTotalBytes: DefaultMaxTotalBytes,
	}
}

// Normalize replaces zero or negative fields with their defaults.
func (l Limits) Normalize() Limits {
	d := DefaultLimits()
	if l.MaxDepth <= 0 {
		l.MaxDepth = d.MaxDepth
	}
	if l.MaxProperties <= 0 {
		l.MaxProperties = d.MaxProperties
	}
	if l.MaxTextChars <= 0 {
		l.MaxTextChars = d.MaxTextChars
	}
	if l.MaxItems <= 0 {
		l.MaxItems = d.MaxItems
	}
	if l.MaxTotalBytes <= 0 {
		l.MaxTotalBytes = d.MaxTotalBytes
	}
	return l
}
