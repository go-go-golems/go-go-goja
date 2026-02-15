package app

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
)

func TestInspectNavigationKeepsSelectedRowVisible(t *testing.T) {
	m := NewModel("")
	m.loaded = true
	m.width = 120
	m.height = 16
	m.inspectObj = goja.New().NewObject()
	m.inspectProps = make([]runtime.PropertyInfo, 40)
	for i := range m.inspectProps {
		m.inspectProps[i] = runtime.PropertyInfo{
			Name:    fmt.Sprintf("p%d", i),
			Kind:    "value",
			Preview: "x",
		}
	}
	m.focus = FocusGlobals
	m.updateMode()

	for i := 0; i < 20; i++ {
		m = updateWithKey(t, m, runeKey('j'))
	}

	if m.inspectIdx != 20 {
		t.Fatalf("inspectIdx = %d, want 20", m.inspectIdx)
	}
	if m.inspectViewport.YOffset <= 0 {
		t.Fatalf("inspect viewport should have scrolled, got offset %d", m.inspectViewport.YOffset)
	}
	vpHeight := m.inspectPaneViewportHeight()
	if m.inspectIdx < m.inspectViewport.YOffset || m.inspectIdx >= m.inspectViewport.YOffset+vpHeight {
		t.Fatalf("selected row %d not visible in [%d,%d)", m.inspectIdx, m.inspectViewport.YOffset, m.inspectViewport.YOffset+vpHeight)
	}
}

func TestStackNavigationUsesStackModeAndMaintainsVisibility(t *testing.T) {
	m := NewModel("")
	m.loaded = true
	m.width = 120
	m.height = 16
	m.globalIdx = 3
	m.sourceLines = makeLines(200)
	frames := make([]runtime.StackFrame, 30)
	for i := range frames {
		frames[i] = runtime.StackFrame{
			FunctionName: fmt.Sprintf("f%d", i),
			FileName:     "test.js",
			Line:         i + 1,
			Column:       1,
		}
	}
	m.errorInfo = &runtime.ErrorInfo{
		Message: "boom",
		Frames:  frames,
	}
	m.showingError = true
	m.focus = FocusGlobals
	m.updateMode()

	for i := 0; i < 12; i++ {
		m = updateWithKey(t, m, runeKey('j'))
	}

	if m.mode != modeStack {
		t.Fatalf("mode = %s, want %s", m.mode, modeStack)
	}
	if m.globalIdx != 3 {
		t.Fatalf("globalIdx changed in stack mode: %d", m.globalIdx)
	}
	if m.stackIdx != 12 {
		t.Fatalf("stackIdx = %d, want 12", m.stackIdx)
	}
	if m.stackViewport.YOffset <= 0 {
		t.Fatalf("stack viewport should have scrolled, got offset %d", m.stackViewport.YOffset)
	}
	if m.sourceTarget != frames[12].Line-1 {
		t.Fatalf("sourceTarget = %d, want %d", m.sourceTarget, frames[12].Line-1)
	}
}

func TestSourceNavigationUsesViewportOffsetAndClamps(t *testing.T) {
	m := NewModel("")
	m.loaded = true
	m.width = 120
	m.height = 16
	m.focus = FocusSource
	m.sourceLines = makeLines(50)
	m.updateMode()

	m = updateWithKey(t, m, runeKey('G')) // bottom
	maxOffset := len(m.sourceLines) - m.sourceViewportHeight()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.sourceViewport.YOffset != maxOffset {
		t.Fatalf("source offset = %d, want %d", m.sourceViewport.YOffset, maxOffset)
	}

	m = updateWithKey(t, m, runeKey('j')) // down should stay clamped
	if m.sourceViewport.YOffset != maxOffset {
		t.Fatalf("source offset after extra down = %d, want %d", m.sourceViewport.YOffset, maxOffset)
	}

	m = updateWithKey(t, m, runeKey('g')) // top
	if m.sourceViewport.YOffset != 0 {
		t.Fatalf("source offset after top = %d, want 0", m.sourceViewport.YOffset)
	}

	m = updateWithKey(t, m, keyType(tea.KeyCtrlD)) // half down
	want := m.sourceViewportHeight() / 2
	if m.sourceViewport.YOffset != want {
		t.Fatalf("source offset after half-down = %d, want %d", m.sourceViewport.YOffset, want)
	}
}

func TestUpdateModePrecedenceAndReplOverride(t *testing.T) {
	m := NewModel("")
	m.loaded = true
	m.focus = FocusSource
	m.updateMode()
	if m.mode != modeSource {
		t.Fatalf("mode = %s, want %s", m.mode, modeSource)
	}

	m.inspectObj = goja.New().NewObject()
	m.updateMode()
	if m.mode != modeInspect {
		t.Fatalf("mode = %s, want %s", m.mode, modeInspect)
	}

	m.errorInfo = &runtime.ErrorInfo{Frames: []runtime.StackFrame{{Line: 1}}}
	m.showingError = true
	m.updateMode()
	if m.mode != modeStack {
		t.Fatalf("mode = %s, want %s", m.mode, modeStack)
	}

	m.focus = FocusRepl
	m.updateMode()
	if m.mode != modeRepl {
		t.Fatalf("mode = %s, want %s", m.mode, modeRepl)
	}
}

func updateWithKey(t *testing.T, m Model, msg tea.KeyMsg) Model {
	t.Helper()
	next, _ := m.Update(msg)
	typed, ok := next.(Model)
	if !ok {
		t.Fatalf("unexpected model type %T", next)
	}
	return typed
}

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{r},
	}
}

func keyType(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

func makeLines(n int) []string {
	lines := make([]string, n)
	for i := 0; i < n; i++ {
		lines[i] = fmt.Sprintf("line %d", i+1)
	}
	return lines
}
