package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-go-golems/bobatea/pkg/commandpalette"
	"github.com/go-go-golems/bobatea/pkg/overlay"
)

func commandPaletteCmd(name string) tea.Cmd {
	return func() tea.Msg {
		return MsgCommandPaletteExec{Command: name}
	}
}

func (m *Model) setupCommandPaletteCommands() {
	cmds := []commandpalette.Command{
		{Name: "load-file", Description: "load <file.js> via command line", Action: func() tea.Cmd { return commandPaletteCmd("load-file") }},
		{Name: "reload-file", Description: "reload currently loaded file", Action: func() tea.Cmd { return commandPaletteCmd("reload-file") }},
		{Name: "help", Description: "toggle full help", Action: func() tea.Cmd { return commandPaletteCmd("help") }},
		{Name: "focus-globals", Description: "focus globals pane", Action: func() tea.Cmd { return commandPaletteCmd("focus-globals") }},
		{Name: "focus-members", Description: "focus members pane", Action: func() tea.Cmd { return commandPaletteCmd("focus-members") }},
		{Name: "focus-source", Description: "focus source pane", Action: func() tea.Cmd { return commandPaletteCmd("focus-source") }},
		{Name: "focus-repl", Description: "focus repl pane", Action: func() tea.Cmd { return commandPaletteCmd("focus-repl") }},
		{Name: "repl-singleline", Description: "use single-line repl input", Action: func() tea.Cmd { return commandPaletteCmd("repl-singleline") }},
		{Name: "repl-multiline", Description: "use multiline repl input (textarea)", Action: func() tea.Cmd { return commandPaletteCmd("repl-multiline") }},
		{Name: "repl-toggle-multiline", Description: "toggle repl multiline mode", Action: func() tea.Cmd { return commandPaletteCmd("repl-toggle-multiline") }},
		{Name: "clear-status", Description: "clear status line", Action: func() tea.Cmd { return commandPaletteCmd("clear-status") }},
		{Name: "quit", Description: "quit inspector", Action: func() tea.Cmd { return commandPaletteCmd("quit") }},
	}
	m.commandPalette.SetCommands(cmds)
}

func (m Model) applyCommandPaletteOverlay(base string) string {
	if !m.commandPalette.IsVisible() || m.width <= 0 || m.height <= 0 {
		return base
	}

	palette := m.commandPalette.View()
	if palette == "" {
		return base
	}

	x := (m.width - 80) / 2
	if x < 0 {
		x = 0
	}
	y := (m.height - 14) / 3
	if y < 0 {
		y = 0
	}

	return overlay.PlaceOverlay(x, y, palette, base, false)
}

func (m Model) executePaletteCommand(name string) (Model, tea.Cmd) {
	switch name {
	case "quit":
		return m, tea.Quit
	case "help":
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case "focus-globals":
		m.focus = FocusGlobals
		m.blurReplBuffers()
		m.updateMode()
		return m, nil
	case "focus-members":
		m.focus = FocusMembers
		m.blurReplBuffers()
		m.updateMode()
		return m, nil
	case "focus-source":
		m.focus = FocusSource
		m.blurReplBuffers()
		m.updateMode()
		return m, nil
	case "focus-repl":
		m.focus = FocusRepl
		_ = m.focusReplBuffer()
		m.updateMode()
		return m, nil
	case "repl-singleline":
		m.setReplMultiline(false)
		m.statusMsg = "REPL input mode: single-line"
		return m, nil
	case "repl-multiline":
		m.setReplMultiline(true)
		m.statusMsg = "REPL input mode: multiline (submit with ctrl+s)"
		return m, nil
	case "repl-toggle-multiline":
		m.setReplMultiline(!m.replMultiline)
		if m.replMultiline {
			m.statusMsg = "REPL input mode: multiline (submit with ctrl+s)"
		} else {
			m.statusMsg = "REPL input mode: single-line"
		}
		return m, nil
	case "clear-status":
		m.statusMsg = ""
		return m, nil
	case "reload-file":
		if m.filename == "" {
			m.statusMsg = "No file loaded"
			return m, nil
		}
		m.statusMsg = "Loading " + m.filename + "..."
		filename := m.filename
		return m, func() tea.Msg { return loadFile(filename) }
	case "load-file":
		m.cmdActive = true
		m.command.SetValue("load ")
		m.command.Focus()
		return m, nil
	default:
		m.statusMsg = "Unknown palette command: " + name
		return m, nil
	}
}
