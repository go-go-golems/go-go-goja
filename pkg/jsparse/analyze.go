package jsparse

import (
	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
)

// AnalyzeOptions controls parser behavior for Analyze.
type AnalyzeOptions struct {
	Mode          parser.Mode
	ParserOptions []parser.Option
}

// Diagnostic is a lightweight issue record that tools can render.
type Diagnostic struct {
	Severity string
	Message  string
}

// AnalysisResult is a reusable bundle returned by Analyze.
type AnalysisResult struct {
	Filename   string
	Source     string
	Program    *ast.Program
	ParseErr   error
	Index      *Index
	Resolution *Resolution
}

// Analyze parses source and builds index/resolution structures suitable for tooling.
func Analyze(filename, source string, opts *AnalyzeOptions) *AnalysisResult {
	mode := parser.Mode(0)
	var parserOptions []parser.Option
	if opts != nil {
		mode = opts.Mode
		parserOptions = opts.ParserOptions
	}

	program, parseErr := parser.ParseFile(nil, filename, source, mode, parserOptions...)

	var idx *Index
	var resolution *Resolution
	if program != nil {
		idx = BuildIndex(program, source)
		resolution = Resolve(program, idx)
		idx.Resolution = resolution
	}

	return &AnalysisResult{
		Filename:   filename,
		Source:     source,
		Program:    program,
		ParseErr:   parseErr,
		Index:      idx,
		Resolution: resolution,
	}
}

// Diagnostics returns analysis diagnostics that can be shown to end users.
func (r *AnalysisResult) Diagnostics() []Diagnostic {
	if r == nil || r.ParseErr == nil {
		return nil
	}
	return []Diagnostic{{
		Severity: "error",
		Message:  r.ParseErr.Error(),
	}}
}

// NodeAtOffset returns the most specific AST node containing a source offset.
func (r *AnalysisResult) NodeAtOffset(offset int) *NodeRecord {
	if r == nil || r.Index == nil {
		return nil
	}
	return r.Index.NodeAtOffset(offset)
}

// CompletionContextAt returns completion context derived from CST at (row, col).
func (r *AnalysisResult) CompletionContextAt(root *TSNode, row, col int) CompletionContext {
	if r == nil || root == nil {
		return CompletionContext{Kind: CompletionNone}
	}
	return ExtractCompletionContext(root, []byte(r.Source), row, col)
}

// CompleteAt returns completion candidates for source position (row, col).
func (r *AnalysisResult) CompleteAt(root *TSNode, row, col int) []CompletionCandidate {
	ctx := r.CompletionContextAt(root, row, col)
	if ctx.Kind == CompletionNone {
		return nil
	}
	return ResolveCandidates(ctx, r.Index, root)
}
