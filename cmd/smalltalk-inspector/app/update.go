package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.command.Width = maxInt(16, msg.Width-4)
		return m, nil

	case MsgFileLoaded:
		m.filename = msg.Filename
		m.source = msg.Source
		m.analysis = msg.Analysis
		m.sourceLines = strings.Split(msg.Source, "\n")
		m.loaded = true
		m.mode = modeGlobals
		m.focus = FocusGlobals
		m.sourceTarget = -1
		m.sourceScroll = 0
		m.buildGlobals()
		m.buildMembers()

		// Initialize runtime session
		m.rtSession = runtime.NewSession()
		if err := m.rtSession.Load(msg.Source); err != nil {
			m.statusMsg = fmt.Sprintf("✓ Loaded %s (%d lines, %d globals) ⚠ runtime: %v",
				msg.Filename, len(m.sourceLines), len(m.globals), err)
		} else {
			m.statusMsg = fmt.Sprintf("✓ Loaded %s (%d lines, %d globals)",
				msg.Filename, len(m.sourceLines), len(m.globals))
		}

		// Clear previous REPL state
		m.replResult = ""
		m.replError = ""
		m.inspectObj = nil
		m.inspectProps = nil
		return m, nil

	case MsgFileLoadError:
		m.statusMsg = fmt.Sprintf("✗ Error loading %s: %v", msg.Filename, msg.Err)
		return m, nil

	case MsgStatusNotice:
		m.statusMsg = msg.Text
		return m, nil

	case MsgEvalResult:
		if msg.Result.Error != nil {
			m.replError = msg.Result.Error.Error()
			m.replResult = ""
			m.inspectObj = nil
			m.inspectProps = nil
			// Check if it's a stack trace we can display
			if msg.Result.ErrorStack != "" {
				m.replError = msg.Result.ErrorStack
			}
		} else {
			m.replError = ""
			val := msg.Result.Value
			m.replResult = runtime.ValuePreview(val, m.rtSession.VM, 80)

			// If result is an object, set up object inspection
			if val != nil && !goja.IsUndefined(val) && !goja.IsNull(val) {
				if obj, ok := val.(*goja.Object); ok {
					m.inspectObj = obj
					m.inspectProps = runtime.InspectObject(obj, m.rtSession.VM)
					m.inspectIdx = 0
					m.inspectLabel = msg.Result.Expression
				} else {
					m.inspectObj = nil
					m.inspectProps = nil
				}
			}
		}
		m.replHistory = append(m.replHistory, msg.Result.Expression)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Command input mode
	if m.cmdActive {
		return m.handleCommandInput(msg)
	}

	// Global bindings
	if key.Matches(msg, m.keyMap.Quit) {
		return m, tea.Quit
	}

	if key.Matches(msg, m.keyMap.Command) {
		m.cmdActive = true
		m.command.SetValue("")
		m.command.Focus()
		return m, nil
	}

	if key.Matches(msg, m.keyMap.Back) {
		// Esc: clear status or go back
		m.statusMsg = ""
		return m, nil
	}

	if key.Matches(msg, m.keyMap.NextPane) {
		m.cyclePane(1)
		return m, nil
	}

	if key.Matches(msg, m.keyMap.PrevPane) {
		m.cyclePane(-1)
		return m, nil
	}

	// Pane-specific keys
	if !m.loaded {
		return m, nil
	}

	//exhaustive:ignore
	switch m.focus {
	case FocusGlobals:
		return m.handleGlobalsKey(msg)
	case FocusMembers:
		return m.handleMembersKey(msg)
	case FocusSource:
		return m.handleSourceKey(msg)
	case FocusRepl:
		return m.handleReplKey(msg)
	}

	return m, nil
}

func (m *Model) cyclePane(dir int) {
	if !m.loaded {
		return
	}

	panes := []FocusPane{FocusGlobals, FocusMembers, FocusSource, FocusRepl}
	current := 0
	for i, p := range panes {
		if p == m.focus {
			current = i
			break
		}
	}
	next := (current + dir + len(panes)) % len(panes)
	m.focus = panes[next]

	// Focus/blur REPL input
	if m.focus == FocusRepl {
		m.replInput.Focus()
	} else {
		m.replInput.Blur()
	}

	m.updateMode()
}

func (m *Model) updateMode() {
	switch m.focus {
	case FocusGlobals:
		m.mode = modeGlobals
	case FocusMembers:
		m.mode = modeMembers
	case FocusSource:
		m.mode = modeSource
	case FocusRepl:
		m.mode = modeRepl
	default:
		m.mode = modeEmpty
	}
}

func (m Model) handleGlobalsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keyMap.Down) {
		if m.globalIdx < len(m.globals)-1 {
			m.globalIdx++
			m.ensureGlobalsVisible()
			m.buildMembers()
			m.jumpToSource()
		}
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Up) {
		if m.globalIdx > 0 {
			m.globalIdx--
			m.ensureGlobalsVisible()
			m.buildMembers()
			m.jumpToSource()
		}
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Top) {
		m.globalIdx = 0
		m.globalScroll = 0
		m.buildMembers()
		m.jumpToSource()
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Bottom) {
		if len(m.globals) > 0 {
			m.globalIdx = len(m.globals) - 1
			m.ensureGlobalsVisible()
			m.buildMembers()
			m.jumpToSource()
		}
		return m, nil
	}
	if key.Matches(msg, m.keyMap.HalfDown) {
		m.globalIdx = minInt(m.globalIdx+m.listViewportHeight()/2, len(m.globals)-1)
		m.ensureGlobalsVisible()
		m.buildMembers()
		m.jumpToSource()
		return m, nil
	}
	if key.Matches(msg, m.keyMap.HalfUp) {
		m.globalIdx = maxInt(m.globalIdx-m.listViewportHeight()/2, 0)
		m.ensureGlobalsVisible()
		m.buildMembers()
		m.jumpToSource()
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Select) {
		// Enter: move focus to members
		if len(m.members) > 0 {
			m.focus = FocusMembers
			m.updateMode()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleMembersKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keyMap.Down) {
		if m.memberIdx < len(m.members)-1 {
			m.memberIdx++
			m.ensureMembersVisible()
			m.jumpToSource()
		}
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Up) {
		if m.memberIdx > 0 {
			m.memberIdx--
			m.ensureMembersVisible()
			m.jumpToSource()
		}
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Top) {
		m.memberIdx = 0
		m.memberScroll = 0
		m.jumpToSource()
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Bottom) {
		if len(m.members) > 0 {
			m.memberIdx = len(m.members) - 1
			m.ensureMembersVisible()
			m.jumpToSource()
		}
		return m, nil
	}
	if key.Matches(msg, m.keyMap.HalfDown) {
		m.memberIdx = minInt(m.memberIdx+m.listViewportHeight()/2, len(m.members)-1)
		m.ensureMembersVisible()
		m.jumpToSource()
		return m, nil
	}
	if key.Matches(msg, m.keyMap.HalfUp) {
		m.memberIdx = maxInt(m.memberIdx-m.listViewportHeight()/2, 0)
		m.ensureMembersVisible()
		m.jumpToSource()
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Back) {
		// Esc: go back to globals
		m.focus = FocusGlobals
		m.updateMode()
		return m, nil
	}

	return m, nil
}

func (m Model) handleSourceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keyMap.Down) {
		m.sourceScroll++
		maxScroll := len(m.sourceLines) - m.sourceViewportHeight()
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.sourceScroll > maxScroll {
			m.sourceScroll = maxScroll
		}
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Up) {
		m.sourceScroll--
		if m.sourceScroll < 0 {
			m.sourceScroll = 0
		}
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Top) {
		m.sourceScroll = 0
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Bottom) {
		maxScroll := len(m.sourceLines) - m.sourceViewportHeight()
		if maxScroll < 0 {
			maxScroll = 0
		}
		m.sourceScroll = maxScroll
		return m, nil
	}
	if key.Matches(msg, m.keyMap.HalfDown) {
		m.sourceScroll += m.sourceViewportHeight() / 2
		maxScroll := len(m.sourceLines) - m.sourceViewportHeight()
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.sourceScroll > maxScroll {
			m.sourceScroll = maxScroll
		}
		return m, nil
	}
	if key.Matches(msg, m.keyMap.HalfUp) {
		m.sourceScroll -= m.sourceViewportHeight() / 2
		if m.sourceScroll < 0 {
			m.sourceScroll = 0
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleReplKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		expr := strings.TrimSpace(m.replInput.Value())
		if expr == "" {
			return m, nil
		}
		m.replInput.SetValue("")

		if m.rtSession == nil {
			m.replError = "no runtime session (load a file first)"
			return m, nil
		}

		// Evaluate synchronously (expressions should be fast)
		result := m.rtSession.EvalWithCapture(expr)
		return m, func() tea.Msg {
			return MsgEvalResult{Result: result}
		}
	case "esc", "escape":
		m.focus = FocusGlobals
		m.replInput.Blur()
		m.updateMode()
		return m, nil
	}

	// Forward all other keys to the textinput
	var cmd tea.Cmd
	m.replInput, cmd = m.replInput.Update(msg)
	return m, cmd
}

func (m Model) handleCommandInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "escape":
		m.cmdActive = false
		m.command.Blur()
		m.command.SetValue("")
		return m, nil
	case "enter":
		cmd := strings.TrimSpace(m.command.Value())
		m.cmdActive = false
		m.command.Blur()
		m.command.SetValue("")
		return m.executeCommand(cmd)
	}

	var cmd tea.Cmd
	m.command, cmd = m.command.Update(msg)
	return m, cmd
}

func (m Model) executeCommand(input string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return m, nil
	}

	switch parts[0] {
	case "q", "quit":
		return m, tea.Quit
	case "load":
		if len(parts) < 2 {
			m.statusMsg = "Usage: :load <file.js>"
			return m, nil
		}
		filename := parts[1]
		m.statusMsg = fmt.Sprintf("Loading %s...", filename)
		return m, func() tea.Msg {
			return loadFile(filename)
		}
	case "help":
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	default:
		m.statusMsg = fmt.Sprintf("Unknown command: %s", parts[0])
		return m, nil
	}
}

func (m *Model) ensureGlobalsVisible() {
	vpHeight := m.listViewportHeight()
	if m.globalIdx < m.globalScroll {
		m.globalScroll = m.globalIdx
	}
	if m.globalIdx >= m.globalScroll+vpHeight {
		m.globalScroll = m.globalIdx - vpHeight + 1
	}
}

func (m *Model) ensureMembersVisible() {
	vpHeight := m.listViewportHeight()
	if m.memberIdx < m.memberScroll {
		m.memberScroll = m.memberIdx
	}
	if m.memberIdx >= m.memberScroll+vpHeight {
		m.memberScroll = m.memberIdx - vpHeight + 1
	}
}
