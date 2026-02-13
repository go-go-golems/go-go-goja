package jsparse

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
)

const maxInt = int(^uint(0) >> 1)

// TSNode is a lightweight, owning snapshot of a tree-sitter CST node.
// Unlike *tree_sitter.Node, TSNode values survive tree reparse/close.
type TSNode struct {
	Kind      string
	Text      string // non-empty for leaf (terminal) nodes only
	StartRow  int
	StartCol  int
	EndRow    int
	EndCol    int
	IsError   bool
	IsMissing bool
	Children  []*TSNode
}

// TSParser wraps a tree-sitter parser with incremental state for JavaScript.
type TSParser struct {
	parser *tree_sitter.Parser
	lang   *tree_sitter.Language
	tree   *tree_sitter.Tree // last parse result, kept for incremental reparse
	source []byte
}

// NewTSParser creates a tree-sitter parser configured for JavaScript.
func NewTSParser() (*TSParser, error) {
	p := tree_sitter.NewParser()
	lang := tree_sitter.NewLanguage(tree_sitter_javascript.Language())
	if err := p.SetLanguage(lang); err != nil {
		p.Close()
		return nil, err
	}
	return &TSParser{parser: p, lang: lang}, nil
}

// Parse parses source from scratch. Returns a snapshot root TSNode.
// Note: we don't use incremental parsing here because the drawer buffer is small
// and we'd need to track precise byte-level edit positions for tree.Edit().
func (tp *TSParser) Parse(source []byte) *TSNode {
	tp.source = source
	// Close old tree before parsing fresh (don't pass old tree without Edit info)
	if tp.tree != nil {
		tp.tree.Close()
		tp.tree = nil
	}
	newTree := tp.parser.Parse(source, nil)
	tp.tree = newTree
	if tp.tree == nil {
		return nil
	}
	root := tp.tree.RootNode()
	return snapshotNode(root, source, 64) // max depth 64
}

// Close releases all tree-sitter resources.
func (tp *TSParser) Close() {
	if tp.tree != nil {
		tp.tree.Close()
		tp.tree = nil
	}
	if tp.parser != nil {
		tp.parser.Close()
		tp.parser = nil
	}
}

// snapshotNode recursively converts a tree-sitter node into an owning TSNode.
func snapshotNode(n *tree_sitter.Node, src []byte, maxDepth int) *TSNode {
	if n == nil || maxDepth <= 0 {
		return nil
	}
	startPos := n.StartPosition()
	endPos := n.EndPosition()
	sn := &TSNode{
		Kind:      n.Kind(),
		StartRow:  uintToClampedInt(startPos.Row),
		StartCol:  uintToClampedInt(startPos.Column),
		EndRow:    uintToClampedInt(endPos.Row),
		EndCol:    uintToClampedInt(endPos.Column),
		IsError:   n.IsError(),
		IsMissing: n.IsMissing(),
	}
	childCount := n.ChildCount()
	if childCount == 0 {
		start := n.StartByte()
		end := n.EndByte()
		if start <= end && end <= uint(len(src)) {
			startInt, okStart := uintToInt(start)
			endInt, okEnd := uintToInt(end)
			if okStart && okEnd {
				sn.Text = string(src[startInt:endInt])
			}
		}
	} else {
		cc, ok := uintToInt(childCount)
		if !ok {
			cc = maxInt
		}
		sn.Children = make([]*TSNode, 0, cc)
		for i := uint(0); i < childCount; i++ {
			child := snapshotNode(n.Child(i), src, maxDepth-1)
			if child != nil {
				sn.Children = append(sn.Children, child)
			}
		}
	}
	return sn
}

// NodeAtPosition returns the deepest TSNode containing (row, col) (both 0-based).
func (sn *TSNode) NodeAtPosition(row, col int) *TSNode {
	if sn == nil {
		return nil
	}
	if !nodeContains(sn, row, col) {
		return nil
	}
	// Try to find a more specific child
	for _, child := range sn.Children {
		if found := child.NodeAtPosition(row, col); found != nil {
			return found
		}
	}
	return sn
}

// nodeContains checks whether (row, col) is within [start, end) of node.
func nodeContains(n *TSNode, row, col int) bool {
	if row < n.StartRow || row > n.EndRow {
		return false
	}
	if row == n.StartRow && col < n.StartCol {
		return false
	}
	if row == n.EndRow && col >= n.EndCol {
		return false
	}
	return true
}

// HasError returns true if this node or any descendant is an error node.
func (sn *TSNode) HasError() bool {
	if sn == nil {
		return false
	}
	if sn.IsError {
		return true
	}
	for _, child := range sn.Children {
		if child.HasError() {
			return true
		}
	}
	return false
}

// ChildCount returns the number of children.
func (sn *TSNode) ChildCount() int {
	if sn == nil {
		return 0
	}
	return len(sn.Children)
}

func uintToInt(v uint) (int, bool) {
	if v > uint(maxInt) {
		return 0, false
	}
	return int(v), true
}

func uintToClampedInt(v uint) int {
	i, ok := uintToInt(v)
	if !ok {
		return maxInt
	}
	return i
}
