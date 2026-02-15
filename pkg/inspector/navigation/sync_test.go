package navigation

import (
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

func testIndexAndLines(t *testing.T, src string) (*jsparse.Index, []string) {
	t.Helper()
	res := jsparse.Analyze("test.js", src, nil)
	if res == nil || res.Index == nil {
		t.Fatalf("expected analysis with index")
	}
	return res.Index, strings.Split(src, "\n")
}

func TestSourceOffset(t *testing.T) {
	lines := []string{"ab", "cde"}

	if got := SourceOffset(lines, 0, 0); got != 1 {
		t.Fatalf("expected offset 1, got %d", got)
	}
	if got := SourceOffset(lines, 0, 2); got != 3 {
		t.Fatalf("expected offset 3, got %d", got)
	}
	if got := SourceOffset(lines, 1, 0); got != 4 {
		t.Fatalf("expected offset 4, got %d", got)
	}
	if got := SourceOffset(lines, 1, 99); got != 7 {
		t.Fatalf("expected clamped offset 7, got %d", got)
	}
	if got := SourceOffset(lines, -2, -5); got != 1 {
		t.Fatalf("expected clamped negative offset 1, got %d", got)
	}
}

func TestSelectionAtSourceCursor(t *testing.T) {
	src := "const x = 1;\nconsole.log(x)\n"
	idx, lines := testIndexAndLines(t, src)

	col := strings.Index(lines[1], "x")
	selection, ok := SelectionAtSourceCursor(idx, lines, 1, col)
	if !ok {
		t.Fatalf("expected selection at source cursor")
	}
	if selection.NodeID < 0 {
		t.Fatalf("expected selected node id, got %d", selection.NodeID)
	}
	if selection.HighlightStart <= 0 || selection.HighlightEnd <= selection.HighlightStart {
		t.Fatalf("expected valid highlight span, got [%d,%d)", selection.HighlightStart, selection.HighlightEnd)
	}
}

func TestFindVisibleNodeIndex(t *testing.T) {
	visible := []jsparse.NodeID{10, 20, 30}
	if got := FindVisibleNodeIndex(visible, 20); got != 1 {
		t.Fatalf("expected index 1, got %d", got)
	}
	if got := FindVisibleNodeIndex(visible, 99); got != -1 {
		t.Fatalf("expected -1 for missing node, got %d", got)
	}
}

func TestSelectionFromVisibleTree(t *testing.T) {
	src := "const x = 1;\nconsole.log(x)\n"
	idx, _ := testIndexAndLines(t, src)

	visible := idx.VisibleNodes()
	if len(visible) == 0 {
		t.Fatalf("expected visible nodes")
	}

	selection, ok := SelectionFromVisibleTree(idx, visible, 0)
	if !ok {
		t.Fatalf("expected tree selection")
	}
	if selection.NodeID != visible[0] {
		t.Fatalf("expected node %d, got %d", visible[0], selection.NodeID)
	}

	node := idx.Nodes[selection.NodeID]
	if node == nil {
		t.Fatalf("selected node is nil")
	}
	if selection.CursorLine != node.StartLine-1 || selection.CursorCol != node.StartCol-1 {
		t.Fatalf(
			"expected cursor (%d,%d), got (%d,%d)",
			node.StartLine-1,
			node.StartCol-1,
			selection.CursorLine,
			selection.CursorCol,
		)
	}
}
