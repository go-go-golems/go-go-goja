package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func applyCmd(t *testing.T, m *Model, cmd tea.Cmd) *Model {
	t.Helper()
	if cmd == nil {
		return m
	}
	msg := cmd()
	next, nextCmd := m.Update(msg)
	nm, ok := next.(*Model)
	if !ok {
		t.Fatalf("expected *Model, got %T", next)
	}
	_ = nextCmd
	return nm
}

func newTestModel(t *testing.T, src string) *Model {
	t.Helper()
	m := NewModel("test.js", src)
	m.parseDebounce = 0

	cmd := m.Init()
	m = applyCmd(t, m, cmd)
	return m
}

func TestInitialParseProducesASTSExpr(t *testing.T) {
	m := newTestModel(t, "const x = 1;")

	if m.astParseErr != nil {
		t.Fatalf("expected no parse error, got %v", m.astParseErr)
	}
	if m.astSExpr == "" {
		t.Fatal("expected non-empty AST S-expression")
	}
}

func TestEditToInvalidSourceClearsASTPane(t *testing.T) {
	m := newTestModel(t, "const obj = {};\nobj")
	m.cursorRow = 1
	m.cursorCol = len([]rune(m.lines[1]))

	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}}
	next, cmd := m.Update(key)
	var ok bool
	m, ok = next.(*Model)
	if !ok {
		t.Fatalf("expected *Model after key update, got %T", next)
	}
	m = applyCmd(t, m, cmd)

	if m.astParseErr == nil {
		t.Fatal("expected parse error after typing trailing dot")
	}
	if m.astSExpr != "" {
		t.Fatalf("expected AST pane to clear on invalid parse, got: %s", m.astSExpr)
	}
}

func TestCSTSExprChangesAfterEdit(t *testing.T) {
	m := newTestModel(t, "const x = 1;")
	old := m.tsSExpr
	m.cursorRow = 0
	m.cursorCol = len([]rune(m.lines[0]))

	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{';'}}
	next, cmd := m.Update(key)
	var ok bool
	m, ok = next.(*Model)
	if !ok {
		t.Fatalf("expected *Model after key update, got %T", next)
	}
	m = applyCmd(t, m, cmd)

	if m.tsSExpr == old {
		t.Fatal("expected CST S-expression to change after edit")
	}
}

func TestStaleASTParseMessageIsIgnored(t *testing.T) {
	m := newTestModel(t, "const x = 1;")
	old := m.astSExpr
	m.pendingSeq = 5

	next, _ := m.Update(astParsedMsg{
		Seq:      4,
		ParseErr: nil,
		ASTSExpr: "(Program stale)",
	})
	nm, ok := next.(*Model)
	if !ok {
		t.Fatalf("expected *Model, got %T", next)
	}

	if nm.astSExpr != old {
		t.Fatalf("expected stale AST message to be ignored, got: %s", nm.astSExpr)
	}
}

func TestASTParseTransitionsInvalidBackToValid(t *testing.T) {
	m := newTestModel(t, "const obj = {};\nobj")
	m.cursorRow = 1
	m.cursorCol = len([]rune(m.lines[1]))

	// Make it invalid: obj.
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}})
	var ok bool
	m, ok = next.(*Model)
	if !ok {
		t.Fatalf("expected *Model after invalidating edit, got %T", next)
	}
	m = applyCmd(t, m, cmd)
	if m.astParseErr == nil {
		t.Fatal("expected parse error for invalid source")
	}
	if m.astSExpr != "" {
		t.Fatalf("expected cleared AST S-expression on invalid source, got %s", m.astSExpr)
	}

	// Recover by adding identifier chars: obj.foo
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f', 'o', 'o'}})
	m, ok = next.(*Model)
	if !ok {
		t.Fatalf("expected *Model after recovery edit, got %T", next)
	}
	m = applyCmd(t, m, cmd)

	if m.astParseErr != nil {
		t.Fatalf("expected recovered valid parse, got %v", m.astParseErr)
	}
	if m.astSExpr == "" {
		t.Fatal("expected AST S-expression after recovery to valid source")
	}
}
