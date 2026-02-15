// Package navigation provides UI-agnostic source/tree synchronization helpers.
package navigation

import "github.com/go-go-golems/go-go-goja/pkg/jsparse"

// SourceSelection captures the node selected from a source cursor position.
type SourceSelection struct {
	NodeID         jsparse.NodeID
	HighlightStart int
	HighlightEnd   int
}

// TreeSelection captures source cursor/highlight for a selected tree row.
type TreeSelection struct {
	NodeID         jsparse.NodeID
	HighlightStart int
	HighlightEnd   int
	CursorLine     int // 0-based
	CursorCol      int // 0-based
}

// SourceOffset returns a 1-based byte offset from a 0-based (line,col) cursor.
func SourceOffset(sourceLines []string, cursorLine, cursorCol int) int {
	if cursorLine < 0 {
		cursorLine = 0
	}
	offset := 0
	for i := 0; i < cursorLine && i < len(sourceLines); i++ {
		offset += len(sourceLines[i]) + 1 // include '\n'
	}
	if cursorLine >= 0 && cursorLine < len(sourceLines) {
		if cursorCol < 0 {
			cursorCol = 0
		}
		if cursorCol > len(sourceLines[cursorLine]) {
			cursorCol = len(sourceLines[cursorLine])
		}
		offset += cursorCol
	}
	return offset + 1
}

// SelectionAtSourceCursor resolves source cursor -> best node + highlight span.
func SelectionAtSourceCursor(
	idx *jsparse.Index,
	sourceLines []string,
	cursorLine, cursorCol int,
) (SourceSelection, bool) {
	if idx == nil {
		return SourceSelection{}, false
	}
	off := SourceOffset(sourceLines, cursorLine, cursorCol)
	n := idx.NodeAtOffset(off)
	if n == nil {
		return SourceSelection{}, false
	}
	return SourceSelection{
		NodeID:         n.ID,
		HighlightStart: n.Start,
		HighlightEnd:   n.End,
	}, true
}

// FindVisibleNodeIndex returns the index of target in visibleIDs, or -1 if absent.
func FindVisibleNodeIndex(visibleIDs []jsparse.NodeID, target jsparse.NodeID) int {
	for i, id := range visibleIDs {
		if id == target {
			return i
		}
	}
	return -1
}

// SelectionFromVisibleTree resolves selected visible tree row -> source cursor/highlight.
func SelectionFromVisibleTree(
	idx *jsparse.Index,
	visibleIDs []jsparse.NodeID,
	selectedIdx int,
) (TreeSelection, bool) {
	if idx == nil || len(visibleIDs) == 0 {
		return TreeSelection{}, false
	}
	if selectedIdx < 0 || selectedIdx >= len(visibleIDs) {
		return TreeSelection{}, false
	}
	id := visibleIDs[selectedIdx]
	n := idx.Nodes[id]
	if n == nil {
		return TreeSelection{}, false
	}
	return TreeSelection{
		NodeID:         id,
		HighlightStart: n.Start,
		HighlightEnd:   n.End,
		CursorLine:     n.StartLine - 1,
		CursorCol:      n.StartCol - 1,
	}, true
}
