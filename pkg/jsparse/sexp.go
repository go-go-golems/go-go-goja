package jsparse

import (
	"strconv"
	"strings"

	"github.com/dop251/goja/ast"
)

// SExprOptions controls how CST/AST trees are rendered as S-expressions.
type SExprOptions struct {
	IncludeSpan  bool
	IncludeText  bool
	IncludeFlags bool
	MaxDepth     int
	MaxNodes     int
	Compact      bool
}

const (
	defaultSExprMaxDepth = 64
	defaultSExprMaxNodes = 5000
)

type sexprRenderer struct {
	opts      SExprOptions
	nodeCount int
}

// CSTToSExpr renders a tree-sitter snapshot as a LISP S-expression string.
func CSTToSExpr(root *TSNode, opts *SExprOptions) string {
	if root == nil {
		return ""
	}
	r := &sexprRenderer{opts: normalizeSExprOptions(opts)}
	var b strings.Builder
	r.writeCSTNode(&b, root, 0)
	return b.String()
}

// ASTIndexToSExpr renders an indexed goja AST as a LISP S-expression string.
func ASTIndexToSExpr(idx *Index, opts *SExprOptions) string {
	if idx == nil || idx.RootID < 0 {
		return ""
	}
	r := &sexprRenderer{opts: normalizeSExprOptions(opts)}
	var b strings.Builder
	r.writeASTNode(&b, idx, idx.RootID, 0)
	return b.String()
}

// ASTToSExpr builds an AST index from the program/source and renders it as S-expression.
func ASTToSExpr(program *ast.Program, src string, opts *SExprOptions) string {
	if program == nil {
		return ""
	}
	idx := BuildIndex(program, src)
	return ASTIndexToSExpr(idx, opts)
}

func normalizeSExprOptions(opts *SExprOptions) SExprOptions {
	cfg := SExprOptions{
		IncludeText:  true,
		IncludeFlags: true,
		MaxDepth:     defaultSExprMaxDepth,
		MaxNodes:     defaultSExprMaxNodes,
	}
	if opts == nil {
		return cfg
	}
	cfg = *opts
	if cfg.MaxDepth <= 0 {
		cfg.MaxDepth = defaultSExprMaxDepth
	}
	if cfg.MaxNodes <= 0 {
		cfg.MaxNodes = defaultSExprMaxNodes
	}
	return cfg
}

func (r *sexprRenderer) shouldTruncate(depth int) bool {
	if depth > r.opts.MaxDepth {
		return true
	}
	if r.nodeCount >= r.opts.MaxNodes {
		return true
	}
	r.nodeCount++
	return false
}

func (r *sexprRenderer) writeTruncated(b *strings.Builder) {
	b.WriteString("(...)")
}

func (r *sexprRenderer) writeCSTNode(b *strings.Builder, n *TSNode, depth int) {
	if n == nil {
		b.WriteString("(nil)")
		return
	}
	if r.shouldTruncate(depth) {
		r.writeTruncated(b)
		return
	}

	b.WriteString("(")
	b.WriteString(sexprAtom(n.Kind))

	if r.opts.IncludeFlags {
		if n.IsError {
			b.WriteString(" :error true")
		}
		if n.IsMissing {
			b.WriteString(" :missing true")
		}
	}
	if r.opts.IncludeSpan {
		b.WriteString(" :range (")
		b.WriteString(strconv.Itoa(n.StartRow))
		b.WriteString(" ")
		b.WriteString(strconv.Itoa(n.StartCol))
		b.WriteString(" ")
		b.WriteString(strconv.Itoa(n.EndRow))
		b.WriteString(" ")
		b.WriteString(strconv.Itoa(n.EndCol))
		b.WriteString(")")
	}
	if r.opts.IncludeText && len(n.Children) == 0 && n.Text != "" {
		b.WriteString(" ")
		b.WriteString(strconv.Quote(n.Text))
	}

	if len(n.Children) > 0 {
		for _, child := range n.Children {
			if r.opts.Compact {
				b.WriteString(" ")
			} else {
				b.WriteString("\n")
				writeIndent(b, depth+1)
			}
			r.writeCSTNode(b, child, depth+1)
		}
		if !r.opts.Compact {
			b.WriteString("\n")
			writeIndent(b, depth)
		}
	}

	b.WriteString(")")
}

func (r *sexprRenderer) writeASTNode(b *strings.Builder, idx *Index, id NodeID, depth int) {
	n := idx.Nodes[id]
	if n == nil {
		b.WriteString("(missing-node)")
		return
	}
	if r.shouldTruncate(depth) {
		r.writeTruncated(b)
		return
	}

	b.WriteString("(")
	b.WriteString(sexprAtom(n.Kind))

	if r.opts.IncludeSpan {
		b.WriteString(" :span (")
		b.WriteString(strconv.Itoa(n.Start))
		b.WriteString(" ")
		b.WriteString(strconv.Itoa(n.End))
		b.WriteString(")")
	}
	if r.opts.IncludeText && n.Label != "" {
		b.WriteString(" :label ")
		b.WriteString(strconv.Quote(n.Label))
	}

	if len(n.ChildIDs) > 0 {
		for _, childID := range n.ChildIDs {
			if r.opts.Compact {
				b.WriteString(" ")
			} else {
				b.WriteString("\n")
				writeIndent(b, depth+1)
			}
			r.writeASTNode(b, idx, childID, depth+1)
		}
		if !r.opts.Compact {
			b.WriteString("\n")
			writeIndent(b, depth)
		}
	}

	b.WriteString(")")
}

func writeIndent(b *strings.Builder, depth int) {
	for i := 0; i < depth; i++ {
		b.WriteString("  ")
	}
}

func sexprAtom(s string) string {
	if s == "" {
		return "\"\""
	}
	if strings.ContainsAny(s, " \t\r\n()\"") {
		return strconv.Quote(s)
	}
	return s
}
