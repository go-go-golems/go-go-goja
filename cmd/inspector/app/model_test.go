package app

import (
	"strings"
	"testing"

	"github.com/dop251/goja/parser"
)

func assertNoPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()
	fn()
}

func newTestModel(t *testing.T, src string) Model {
	t.Helper()

	program, err := parser.ParseFile(nil, "test.js", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	idx := BuildIndex(program, src)
	idx.Resolution = Resolve(program, idx)

	m := NewModel("test.js", src, program, nil, idx)
	m.width = 120
	m.height = 30
	return m
}

func nthIndex(s, needle string, n int) int {
	if n < 1 {
		return -1
	}
	start := 0
	for i := 0; i < n; i++ {
		idx := strings.Index(s[start:], needle)
		if idx < 0 {
			return -1
		}
		start += idx
		if i == n-1 {
			return start
		}
		start += len(needle)
	}
	return -1
}

func nodeAtOccurrence(t *testing.T, idx *Index, src, needle string, occurrence int) NodeID {
	t.Helper()

	bytePos := nthIndex(src, needle, occurrence)
	if bytePos < 0 {
		t.Fatalf("needle %q occurrence %d not found", needle, occurrence)
	}
	// Use a position inside the token to avoid boundary effects.
	offset := bytePos + 2
	node := idx.NodeAtOffset(offset)
	if node == nil {
		t.Fatalf("no node at offset %d for %q occurrence %d", offset, needle, occurrence)
	}
	return node.ID
}

func visibleIndex(ids []NodeID, target NodeID) int {
	for i, id := range ids {
		if id == target {
			return i
		}
	}
	return -1
}

func TestModelSyncSourceAndTree(t *testing.T) {
	src := `const greeting = "hello";
console.log(greeting);
`
	m := newTestModel(t, src)

	// Place source cursor on the usage site, then sync to tree.
	m.sourceCursorLine = 1
	m.sourceCursorCol = strings.Index(m.sourceLines[1], "greeting") + 2
	m.syncSourceToTree()

	if m.syncOrigin != SyncFromSource {
		t.Fatalf("expected source sync origin, got %v", m.syncOrigin)
	}
	selected := m.index.Nodes[m.selectedNodeID]
	if selected == nil || selected.Kind != "Identifier" {
		t.Fatalf("expected selected identifier node, got %+v", selected)
	}

	// Pick declaration node in tree and sync back to source.
	declID := nodeAtOccurrence(t, m.index, src, "greeting", 1)
	m.index.ExpandTo(declID)
	m.refreshTreeVisible()
	m.treeSelectedIdx = visibleIndex(m.treeVisibleNodes, declID)
	if m.treeSelectedIdx < 0 {
		t.Fatalf("declaration node %d not visible in tree", declID)
	}
	m.syncTreeToSource()

	if m.syncOrigin != SyncFromTree {
		t.Fatalf("expected tree sync origin, got %v", m.syncOrigin)
	}
	decl := m.index.Nodes[declID]
	if m.sourceCursorLine != decl.StartLine-1 || m.sourceCursorCol != decl.StartCol-1 {
		t.Fatalf("expected source cursor at decl (%d,%d), got (%d,%d)", decl.StartLine-1, decl.StartCol-1, m.sourceCursorLine, m.sourceCursorCol)
	}
}

func TestModelGoToDefinition(t *testing.T) {
	src := `const greeting = "hello";
console.log(greeting);
`
	m := newTestModel(t, src)

	usageID := nodeAtOccurrence(t, m.index, src, "greeting", 2)
	m.selectedNodeID = usageID
	m.goToDefinition()

	b := m.index.Resolution.BindingForNode(usageID)
	if b == nil {
		t.Fatal("expected binding for usage node")
	}
	if m.selectedNodeID != b.DeclNodeID {
		t.Fatalf("expected selected node to jump to decl %d, got %d", b.DeclNodeID, m.selectedNodeID)
	}

	decl := m.index.Nodes[b.DeclNodeID]
	if decl == nil {
		t.Fatal("declaration node missing")
	}
	if m.sourceCursorLine != decl.StartLine-1 || m.sourceCursorCol != decl.StartCol-1 {
		t.Fatalf("expected source cursor at declaration (%d,%d), got (%d,%d)", decl.StartLine-1, decl.StartCol-1, m.sourceCursorLine, m.sourceCursorCol)
	}
	if m.highlightStart != decl.Start || m.highlightEnd != decl.End {
		t.Fatalf("expected highlight [%d,%d), got [%d,%d)", decl.Start, decl.End, m.highlightStart, m.highlightEnd)
	}
}

func TestModelToggleHighlightUsages(t *testing.T) {
	src := `const value = 1;
const doubled = value + value;
`
	m := newTestModel(t, src)

	usageID := nodeAtOccurrence(t, m.index, src, "value", 2)
	m.selectedNodeID = usageID
	m.toggleHighlightUsages()

	b := m.index.Resolution.BindingForNode(usageID)
	if b == nil {
		t.Fatal("expected binding for value usage")
	}
	if m.highlightedBinding != b {
		t.Fatal("expected highlighted binding to be set")
	}
	if len(m.usageHighlights) != len(b.AllUsages()) {
		t.Fatalf("expected %d usage highlights, got %d", len(b.AllUsages()), len(m.usageHighlights))
	}

	// Toggle same binding again: should clear.
	m.toggleHighlightUsages()
	if m.highlightedBinding != nil || len(m.usageHighlights) != 0 {
		t.Fatalf("expected highlight toggle off, got binding=%v count=%d", m.highlightedBinding, len(m.usageHighlights))
	}
}

func TestModelDrawerCompletion(t *testing.T) {
	src := `const greeting = "hello";
console.log(greeting);
`
	m := newTestModel(t, src)

	for _, ch := range "console." {
		m.drawer.InsertChar(ch)
	}
	m.drawer.Reparse()
	m.triggerDrawerCompletion()

	if !m.drawer.completionActive {
		t.Fatal("expected drawer completion to be active")
	}
	if len(m.drawer.completionItems) == 0 {
		t.Fatal("expected completion items")
	}
	foundLog := false
	for _, c := range m.drawer.completionItems {
		if c.Label == "log" {
			foundLog = true
			break
		}
	}
	if !foundLog {
		t.Fatal("expected 'log' in console.* completion items")
	}
}

func TestModelDrawerGoToDefinitionUnresolvedDoesNotPanic(t *testing.T) {
	src := `const known = 1;
console.log(known);
`
	m := newTestModel(t, src)

	for _, ch := range "unknownName" {
		m.drawer.InsertChar(ch)
	}
	m.drawer.Reparse()

	originalSelected := m.selectedNodeID
	assertNoPanic(t, func() {
		m.drawerGoToDefinition()
	})

	if m.selectedNodeID != originalSelected {
		t.Fatalf("expected selection to stay unchanged for unresolved symbol; got %d want %d", m.selectedNodeID, originalSelected)
	}
}

func TestModelDrawerHighlightUsagesUnresolvedClearsWithoutPanic(t *testing.T) {
	src := `const known = 1;
console.log(known);
`
	m := newTestModel(t, src)

	root := m.index.Resolution.Scopes[m.index.Resolution.RootScopeID]
	if root == nil {
		t.Fatal("expected root scope")
	}
	b := root.Bindings["known"]
	if b == nil {
		t.Fatal("expected binding for known")
	}
	m.highlightedBinding = b
	m.usageHighlights = b.AllUsages()
	if len(m.usageHighlights) == 0 {
		t.Fatal("expected pre-existing usage highlights")
	}

	for _, ch := range "unknownName" {
		m.drawer.InsertChar(ch)
	}
	m.drawer.Reparse()

	assertNoPanic(t, func() {
		m.drawerHighlightUsages()
	})

	if m.highlightedBinding != nil {
		t.Fatal("expected unresolved lookup to clear highlighted binding")
	}
	if len(m.usageHighlights) != 0 {
		t.Fatalf("expected unresolved lookup to clear usage highlights, got %d", len(m.usageHighlights))
	}
}
