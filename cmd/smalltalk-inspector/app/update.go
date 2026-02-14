package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
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

		// Initialize runtime session BEFORE building globals/members
		// so buildValueMembers() can use it for runtime introspection.
		m.rtSession = runtime.NewSession()
		rtErr := m.rtSession.Load(msg.Source)

		m.buildGlobals()
		m.buildMembers()

		if rtErr != nil {
			m.statusMsg = fmt.Sprintf("✓ Loaded %s (%d lines, %d globals) ⚠ runtime: %v",
				msg.Filename, len(m.sourceLines), len(m.globals), rtErr)
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
			m.replResult = ""
			m.inspectObj = nil
			m.inspectProps = nil
			m.navStack = nil

			// Parse exception for stack trace display
			if ex, ok := msg.Result.Error.(*goja.Exception); ok {
				info := runtime.ParseException(ex)
				m.errorInfo = &info
				m.stackIdx = 0
				m.showingError = true
				m.replError = info.Message
				// Jump source to first frame
				if len(info.Frames) > 0 {
					m.sourceTarget = info.Frames[0].Line - 1
					m.ensureSourceVisible(m.sourceTarget)
				}
			} else {
				m.replError = msg.Result.Error.Error()
				m.errorInfo = nil
				m.showingError = false
			}
		} else {
			m.replError = ""
			m.errorInfo = nil
			m.showingError = false
			m.navStack = nil
			val := msg.Result.Value
			m.replResult = runtime.ValuePreview(val, m.rtSession.VM, 80)

			// If result is an object, set up object inspection
			if val != nil && !goja.IsUndefined(val) && !goja.IsNull(val) {
				if obj, ok := val.(*goja.Object); ok {
					m.inspectObj = obj
					m.inspectProps = buildInspectProps(obj, m.rtSession.VM)
					m.inspectIdx = 0
					m.inspectLabel = msg.Result.Expression
				} else {
					m.inspectObj = nil
					m.inspectProps = nil
				}
			}
		}
		m.replHistory = append(m.replHistory, msg.Result.Expression)

		// Refresh globals list to pick up new REPL-defined names
		m.refreshRuntimeGlobals()
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
		// Esc: navigate back in inspect stack, or clear inspect, or clear status
		if m.inspectObj != nil && len(m.navStack) > 0 {
			// Pop one level
			frame := m.navStack[len(m.navStack)-1]
			m.navStack = m.navStack[:len(m.navStack)-1]
			if gojaObj, ok := frame.Obj.(*goja.Object); ok {
				m.inspectObj = gojaObj
				m.inspectProps = frame.Props
				m.inspectIdx = frame.Idx
				m.inspectLabel = frame.Label
			}
			return m, nil
		}
		if m.inspectObj != nil {
			// Clear inspect view
			m.inspectObj = nil
			m.inspectProps = nil
			m.inspectLabel = ""
			return m, nil
		}
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

	// If we are showing an error stack trace, handle stack navigation
	if m.showingError && m.errorInfo != nil && m.focus != FocusRepl {
		return m.handleStackKey(msg)
	}

	// If we are in inspect mode and not in REPL, handle inspect-specific navigation
	if m.inspectObj != nil && m.focus != FocusRepl {
		return m.handleInspectKey(msg)
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
		if len(m.globals) == 0 || m.globalIdx >= len(m.globals) {
			return m, nil
		}
		selected := m.globals[m.globalIdx]

		// For value-type globals, trigger runtime inspection
		if selected.Kind != jsparse.BindingClass && selected.Kind != jsparse.BindingFunction {
			if m.rtSession != nil {
				result := m.rtSession.EvalWithCapture(selected.Name)
				return m, func() tea.Msg {
					return MsgEvalResult{Result: result}
				}
			}
			return m, nil
		}

		// For class/function, move focus to members pane
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

		// Extract declared names for const/let tracking
		declNames := extractDeclaredNames(expr)
		m.replDefinedNames = append(m.replDefinedNames, declNames...)

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

func (m Model) handleInspectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keyMap.Down) {
		if m.inspectIdx < len(m.inspectProps)-1 {
			m.inspectIdx++
		}
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Up) {
		if m.inspectIdx > 0 {
			m.inspectIdx--
		}
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Top) {
		m.inspectIdx = 0
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Bottom) {
		if len(m.inspectProps) > 0 {
			m.inspectIdx = len(m.inspectProps) - 1
		}
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Select) {
		// Enter: drill into selected property if it's an object
		if m.inspectIdx < len(m.inspectProps) && m.rtSession != nil {
			prop := m.inspectProps[m.inspectIdx]
			if prop.Value != nil && prop.Kind == "object" {
				if obj, ok := prop.Value.(*goja.Object); ok {
					// Push current state onto nav stack
					m.navStack = append(m.navStack, NavFrame{
						Label: m.inspectLabel,
						Props: m.inspectProps,
						Obj:   m.inspectObj,
						Idx:   m.inspectIdx,
					})
					// Navigate into the property
					m.inspectObj = obj
					m.inspectProps = buildInspectProps(obj, m.rtSession.VM)
					m.inspectIdx = 0
					m.inspectLabel = m.inspectLabel + " → " + prop.Name
				}
			}
			// If it's a function, try to jump to source
			if prop.Kind == "function" && prop.Value != nil {
				mapping := runtime.MapFunctionToSource(prop.Value, m.rtSession.VM, m.analysis)
				if mapping != nil {
					m.sourceTarget = mapping.StartLine - 1
					m.ensureSourceVisible(m.sourceTarget)
				}
			}
		}
		return m, nil
	}

	return m, nil
}

// buildInspectProps creates property info list including [[Proto]] entry for prototype navigation.
func buildInspectProps(obj *goja.Object, vm *goja.Runtime) []runtime.PropertyInfo {
	props := runtime.InspectObject(obj, vm)

	// Add [[Proto]] entry if prototype exists
	proto := obj.Prototype()
	if proto != nil {
		protoName := "<anonymous>"
		if ctor := proto.Get("constructor"); ctor != nil && !goja.IsUndefined(ctor) {
			if ctorObj, ok := ctor.(*goja.Object); ok {
				if n := ctorObj.Get("name"); n != nil && !goja.IsUndefined(n) && n.String() != "" {
					protoName = n.String() + ".prototype"
				}
			}
		}
		props = append(props, runtime.PropertyInfo{
			Name:    "[[Proto]]",
			Value:   proto,
			Kind:    "object",
			Preview: protoName,
		})
	}

	return props
}

func (m Model) handleStackKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keyMap.Down) {
		if m.errorInfo != nil && m.stackIdx < len(m.errorInfo.Frames)-1 {
			m.stackIdx++
			frame := m.errorInfo.Frames[m.stackIdx]
			m.sourceTarget = frame.Line - 1
			m.ensureSourceVisible(m.sourceTarget)
		}
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Up) {
		if m.stackIdx > 0 {
			m.stackIdx--
			if m.errorInfo != nil && m.stackIdx < len(m.errorInfo.Frames) {
				frame := m.errorInfo.Frames[m.stackIdx]
				m.sourceTarget = frame.Line - 1
				m.ensureSourceVisible(m.sourceTarget)
			}
		}
		return m, nil
	}
	if key.Matches(msg, m.keyMap.Back) {
		// Clear error view
		m.showingError = false
		m.errorInfo = nil
		m.sourceTarget = -1
		return m, nil
	}

	return m, nil
}
