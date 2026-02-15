package app

import (
	"github.com/dop251/goja/ast"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

type NodeID = jsparse.NodeID
type NodeRecord = jsparse.NodeRecord
type Index = jsparse.Index
type Resolution = jsparse.Resolution
type BindingRecord = jsparse.BindingRecord
type ScopeID = jsparse.ScopeID

type CompletionContext = jsparse.CompletionContext
type CompletionKind = jsparse.CompletionKind
type CompletionCandidate = jsparse.CompletionCandidate
type CandidateKind = jsparse.CandidateKind

type TSNode = jsparse.TSNode
type TSParser = jsparse.TSParser

const (
	CompletionNone       = jsparse.CompletionNone
	CompletionProperty   = jsparse.CompletionProperty
	CompletionIdentifier = jsparse.CompletionIdentifier
	CompletionArgument   = jsparse.CompletionArgument
)

func NewTSParser() (*TSParser, error) {
	return jsparse.NewTSParser()
}

func BuildIndex(program *ast.Program, src string) *Index {
	return jsparse.BuildIndex(program, src)
}

func Resolve(program *ast.Program, idx *Index) *Resolution {
	return jsparse.Resolve(program, idx)
}

func ExtractCompletionContext(root *TSNode, source []byte, cursorRow, cursorCol int) CompletionContext {
	return jsparse.ExtractCompletionContext(root, source, cursorRow, cursorCol)
}

func ResolveCandidates(ctx CompletionContext, gojaIndex *Index, drawerRoot ...*TSNode) []CompletionCandidate {
	return jsparse.ResolveCandidates(ctx, gojaIndex, drawerRoot...)
}

func ExtractDrawerBindings(root *TSNode) []CompletionCandidate {
	return jsparse.ExtractDrawerBindings(root)
}
