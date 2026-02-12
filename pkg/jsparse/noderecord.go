// Package inspector provides the AST index and UI models for the static program inspector.
package jsparse

import "fmt"

// NodeID is a unique identifier for a node in the index.
type NodeID int

// NodeRecord is the canonical tree node consumed by UI and sync logic.
type NodeRecord struct {
	ID        NodeID
	Kind      string // e.g. "FunctionDeclaration"
	Start     int    // byte offset (1-based, from file.Idx)
	End       int    // byte offset exclusive (1-based)
	StartLine int    // 1-based
	StartCol  int    // 1-based
	EndLine   int
	EndCol    int
	Label     string   // compact display label
	Snippet   string   // short source excerpt
	ParentID  NodeID   // -1 for root
	ChildIDs  []NodeID // ordered children
	Depth     int
	Expanded  bool // UI state: whether children are visible in tree
}

// Span returns the byte length of this node.
func (n *NodeRecord) Span() int {
	return n.End - n.Start
}

// String returns a compact description.
func (n *NodeRecord) String() string {
	return fmt.Sprintf("%s [%d..%d] %q", n.Kind, n.Start, n.End, n.Label)
}

// DisplayLabel returns the label for tree rendering.
func (n *NodeRecord) DisplayLabel() string {
	if n.Label != "" {
		return fmt.Sprintf("%s %s", n.Kind, n.Label)
	}
	return n.Kind
}

// HasChildren returns true if the node has children.
func (n *NodeRecord) HasChildren() bool {
	return len(n.ChildIDs) > 0
}
