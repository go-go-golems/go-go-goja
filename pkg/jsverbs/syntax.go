package jsverbs

// This file defines the boundary between the scanner's syntax backends and the
// scanner itself. A backend (SourceParser) reads one JavaScript source and
// reports, in source order, only what jsverbs needs from the top level of the
// file: function declarations, calls to the metadata sentinels with their
// literal arguments, and doc`...` blocks. The scanner turns those items into
// packages, sections, verbs and diagnostics the same way for every backend.
//
// The default backend uses goja's own parser and needs no cgo. The package
// github.com/go-go-golems/go-go-goja/pkg/jsverbs/treesitter provides the
// tree-sitter backend for callers that opt into it (for example to read ES
// module syntax, which goja's parser does not accept).

// Metadata sentinels recognised at the top level of a scanned file.
const (
	PackageSentinel = "__package__"
	SectionSentinel = "__section__"
	VerbSentinel    = "__verb__"
	// DocTag is the tag of doc`...` template blocks that carry long help.
	DocTag = "doc"
)

// SourceParser reads the top-level declarations jsverbs needs from one source.
//
// ParseSource returns an error only when the source cannot be read as
// JavaScript at all. The scanner then records a diagnostic and keeps the file
// without declarations (see ScanOptions.Parser).
type SourceParser interface {
	ParseSource(input ParseInput) (*ParsedSource, error)
}

// ParseInput is one source file handed to a SourceParser.
type ParseInput struct {
	// RelPath is the slash-separated path of the file, for messages.
	RelPath string
	// Source is the JavaScript text, after any ScanOptions.SourceTransform.
	Source []byte
}

// ParsedSource lists the top-level items of one file in source order.
type ParsedSource struct {
	Items []ParsedItem
}

// ParsedItemKind says what a ParsedItem describes.
type ParsedItemKind string

const (
	// ParsedItemFunction is a top-level function declaration, or a top-level
	// variable initialised with a non-generator function or arrow function.
	ParsedItemFunction ParsedItemKind = "function"
	// ParsedItemCall is a top-level call statement to one of the metadata
	// sentinels (PackageSentinel, SectionSentinel, VerbSentinel).
	ParsedItemCall ParsedItemKind = "call"
	// ParsedItemDoc is a top-level doc`...` tagged template statement.
	ParsedItemDoc ParsedItemKind = "doc"
)

// ParsedItem is one top-level item. Which fields are set depends on Kind.
type ParsedItem struct {
	Kind ParsedItemKind
	// Name is the function name (ParsedItemFunction) or the sentinel called
	// (ParsedItemCall).
	Name string
	// Params are the function's parameters (ParsedItemFunction). An object or
	// array pattern's Name is its source text, for example "{ a, b }".
	Params []ParameterSpec
	// Args are the call's arguments (ParsedItemCall), comments excluded.
	Args []ParsedArg
	// Text is the raw text between the backticks (ParsedItemDoc).
	Text string
}

// ParsedArg is one argument of a sentinel call, decoded as a metadata literal.
//
// Supported literals are objects with plain keys, arrays, strings, template
// strings without substitutions (their raw text), numbers (float64),
// booleans and null. Objects decode to map[string]interface{} and arrays to
// []interface{}. Anything else sets Err.
type ParsedArg struct {
	// Object reports whether the argument is an object literal.
	Object bool
	// Value is the decoded literal when Err is nil.
	Value interface{}
	// Err says why the argument is not a supported metadata literal.
	Err error
}

// IsMetadataSentinel reports whether name is one of the metadata sentinels a
// SourceParser reports calls to.
func IsMetadataSentinel(name string) bool {
	switch name {
	case PackageSentinel, SectionSentinel, VerbSentinel:
		return true
	default:
		return false
	}
}
