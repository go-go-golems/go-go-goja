// Package model defines the data structures for the JS documentation system.
package model

// Param describes a function parameter.
type Param struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

// ReturnInfo describes a function's return value.
type ReturnInfo struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

// SymbolDoc holds the metadata extracted from a __doc__ sentinel call.
type SymbolDoc struct {
	Name     string     `json:"name"`
	Summary  string     `json:"summary,omitempty"`
	Concepts []string   `json:"concepts,omitempty"`
	DocPage  string     `json:"docpage,omitempty"`
	Params   []Param    `json:"params,omitempty"`
	Returns  ReturnInfo `json:"returns,omitempty"`
	Related  []string   `json:"related,omitempty"`
	Tags     []string   `json:"tags,omitempty"`
	// Prose is the long-form Markdown body from a doc`` tagged template.
	Prose string `json:"prose,omitempty"`
	// SourceFile is the file this symbol was extracted from.
	SourceFile string `json:"source_file,omitempty"`
	// Line is a 1-based line number in source.
	Line int `json:"line,omitempty"`
}

// Example holds the metadata extracted from a __example__ sentinel call.
type Example struct {
	ID       string   `json:"id"`
	Title    string   `json:"title,omitempty"`
	Symbols  []string `json:"symbols,omitempty"`
	Concepts []string `json:"concepts,omitempty"`
	DocPage  string   `json:"docpage,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	// Body is the source code of the function immediately following the sentinel.
	Body string `json:"body,omitempty"`

	SourceFile string `json:"source_file,omitempty"`
	Line       int    `json:"line,omitempty"`
}

// Package holds the metadata extracted from a __package__ sentinel call.
type Package struct {
	Name        string   `json:"name"`
	Title       string   `json:"title,omitempty"`
	Category    string   `json:"category,omitempty"`
	Guide       string   `json:"guide,omitempty"`
	Version     string   `json:"version,omitempty"`
	Description string   `json:"description,omitempty"`
	SeeAlso     []string `json:"see_also,omitempty"`
	// Prose is the long-form Markdown body from a doc`` tagged template at package level.
	Prose string `json:"prose,omitempty"`

	SourceFile string `json:"source_file,omitempty"`
}

// FileDoc is the complete documentation extracted from a single JS file.
type FileDoc struct {
	Package  *Package     `json:"package,omitempty"`
	Symbols  []*SymbolDoc `json:"symbols,omitempty"`
	Examples []*Example   `json:"examples,omitempty"`
	FilePath string       `json:"file_path"`
}
