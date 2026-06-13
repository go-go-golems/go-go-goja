package providerapi

import "github.com/go-go-golems/go-go-goja/pkg/jsverbs"

// RuntimeSourceKind identifies a source kind in the generated xgoja runtime plan.
type RuntimeSourceKind string

const (
	RuntimeSourceKindJSVerbs RuntimeSourceKind = "jsverbs"
	RuntimeSourceKindScript  RuntimeSourceKind = "script"
	RuntimeSourceKindAssets  RuntimeSourceKind = "assets"
	RuntimeSourceKindHelp    RuntimeSourceKind = "help"
)

// RuntimeSourceDescriptor describes one configured runtime source in a provider-neutral shape.
type RuntimeSourceDescriptor struct {
	ID         string
	Kind       RuntimeSourceKind
	Path       string
	Embed      bool
	Provider   string
	Source     string
	Include    []string
	Exclude    []string
	Extensions []string
	TypeScript *TypeScriptDescriptor
}

// SourceRegistry is the v2 runtime source lookup API passed to provider command sets.
type SourceRegistry interface {
	ListSources() []RuntimeSourceDescriptor
	ListSourcesByKind(kind RuntimeSourceKind) []RuntimeSourceDescriptor
	SourceByID(id string) (RuntimeSourceDescriptor, bool)
	JSVerbs() JSVerbSourceSet
}

// JSVerbScanner is implemented by JS verb source adapters that can scan source descriptors.
type JSVerbScanner interface {
	ScanJSVerbSource(id string) (*jsverbs.Registry, error)
	ScanAllJSVerbSources() ([]*jsverbs.Registry, error)
}
