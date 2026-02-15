package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
)

func TestSetReplMultilineTransfersDraftBetweenWidgets(t *testing.T) {
	m := NewModel("")
	m.loaded = true
	m.focus = FocusRepl
	m.updateMode()
	m.replInput.SetValue("const x = 1")

	m.setReplMultiline(true)
	if !m.replMultiline {
		t.Fatal("expected replMultiline to be enabled")
	}
	if got := m.replTextarea.Value(); got != "const x = 1" {
		t.Fatalf("textarea value = %q, want %q", got, "const x = 1")
	}

	m.replTextarea.SetValue("const x = 1\nx + 1")
	m.setReplMultiline(false)
	if m.replMultiline {
		t.Fatal("expected replMultiline to be disabled")
	}
	if got := m.replInput.Value(); got != "const x = 1 x + 1" {
		t.Fatalf("text input value = %q, want %q", got, "const x = 1 x + 1")
	}
}

func TestReplMultilineSubmitWithCtrlS(t *testing.T) {
	m := NewModel("")
	m.loaded = true
	m.focus = FocusRepl
	m.updateMode()

	m.rtSession = runtime.NewSession()
	if err := m.rtSession.Load(""); err != nil {
		t.Fatalf("runtime load failed: %v", err)
	}

	m.setReplMultiline(true)
	m.replTextarea.SetValue("1 + 1")

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m2 := next.(Model)
	if cmd == nil {
		t.Fatal("expected submit command for ctrl+s in multiline mode")
	}

	msg := cmd()
	next2, _ := m2.Update(msg)
	m3 := next2.(Model)
	if m3.replError != "" {
		t.Fatalf("unexpected replError: %s", m3.replError)
	}
	if !strings.Contains(m3.replResult, "2") {
		t.Fatalf("replResult = %q, want to contain %q", m3.replResult, "2")
	}
	if got := m3.replTextarea.Value(); got != "" {
		t.Fatalf("textarea should be cleared after submit, got %q", got)
	}
}

func TestContentHeightAccountsForMultilineReplArea(t *testing.T) {
	m := NewModel("")
	m.height = 24
	single := m.contentHeight()

	m.setReplMultiline(true)
	m.replTextarea.SetHeight(3)
	multi := m.contentHeight()

	if multi >= single {
		t.Fatalf("expected multiline content height (%d) to be smaller than single-line (%d)", multi, single)
	}
}
