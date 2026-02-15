package app

import (
	"github.com/charmbracelet/bubbles/key"
)

// FocusPane indicates which pane has focus.
type FocusPane int

const (
	FocusGlobals FocusPane = iota
	FocusMembers
	FocusSource
	FocusRepl
)

// Mode names for mode-keymap integration.
const (
	modeEmpty   = "empty"
	modeGlobals = "globals"
	modeMembers = "members"
	modeSource  = "source"
	modeRepl    = "repl"
	modeInspect = "inspect"
	modeStack   = "stack"
)

// KeyMap defines key bindings for the smalltalk inspector.
type KeyMap struct {
	Quit    key.Binding `keymap-mode:"*"`
	Command key.Binding `keymap-mode:"*"`

	NextPane key.Binding `keymap-mode:"*"`
	PrevPane key.Binding `keymap-mode:"*"`

	Up       key.Binding `keymap-mode:"globals,members,source,repl,inspect,stack"`
	Down     key.Binding `keymap-mode:"globals,members,source,repl,inspect,stack"`
	Top      key.Binding `keymap-mode:"globals,members,source,repl,inspect,stack"`
	Bottom   key.Binding `keymap-mode:"globals,members,source,repl,inspect,stack"`
	HalfDown key.Binding `keymap-mode:"globals,members,source,repl,inspect,stack"`
	HalfUp   key.Binding `keymap-mode:"globals,members,source,repl,inspect,stack"`

	Select key.Binding `keymap-mode:"globals,members,repl,inspect"`
	Back   key.Binding `keymap-mode:"*"`
}

func newKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		Command: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "command"),
		),
		NextPane: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next pane"),
		),
		PrevPane: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev pane"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		HalfDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "½pg down"),
		),
		HalfUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "½pg up"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select/inspect"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/close"),
		),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NextPane, k.Command, k.Select, k.Back, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextPane, k.PrevPane, k.Command, k.Back, k.Quit},
		{k.Up, k.Down, k.Top, k.Bottom, k.HalfUp, k.HalfDown},
		{k.Select},
	}
}
