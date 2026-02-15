// Package tree provides UI-agnostic tree row shaping for inspector views.
package tree

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// Row is a render-ready, UI-framework-agnostic view model for one tree node row.
type Row struct {
	NodeID      jsparse.NodeID
	Title       string
	Description string
}

// BuildRowsFromIndex builds rows for currently visible nodes in index order.
func BuildRowsFromIndex(idx *jsparse.Index, usageHighlights []jsparse.NodeID) []Row {
	if idx == nil {
		return nil
	}
	visible := idx.VisibleNodes()
	rows := make([]Row, 0, len(visible))
	for _, id := range visible {
		n := idx.Nodes[id]
		if n == nil {
			continue
		}
		rows = append(rows, BuildRow(n, usageHighlights, idx.Resolution))
	}
	return rows
}

// BuildRow converts a node into a display row.
func BuildRow(node *jsparse.NodeRecord, usageHighlights []jsparse.NodeID, res *jsparse.Resolution) Row {
	if node == nil {
		return Row{}
	}

	indent := strings.Repeat("  ", node.Depth)
	expandMarker := " "
	if node.HasChildren() {
		if node.Expanded {
			expandMarker = "▼"
		} else {
			expandMarker = "▶"
		}
	}

	scopeHint := ""
	if res != nil && node.Kind == "Identifier" {
		if res.IsDeclaration(node.ID) {
			if b := res.BindingForNode(node.ID); b != nil {
				scopeHint = fmt.Sprintf(" [%s decl]", b.Kind)
			}
		} else if res.IsReference(node.ID) {
			scopeHint = " [ref]"
		} else if res.IsUnresolved(node.ID) {
			scopeHint = " [global]"
		}
	}

	isUsage := false
	for _, id := range usageHighlights {
		if id == node.ID {
			isUsage = true
			break
		}
	}
	usageHint := ""
	if isUsage {
		usageHint = " ★usage"
	}

	return Row{
		NodeID:      node.ID,
		Title:       indent + expandMarker + " " + node.DisplayLabel(),
		Description: fmt.Sprintf("[%d..%d]%s%s", node.Start, node.End, scopeHint, usageHint),
	}
}
