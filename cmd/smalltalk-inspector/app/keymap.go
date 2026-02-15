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

	CompletionTrigger  key.Binding `keymap-mode:"repl"`
	CompletionAccept   key.Binding `keymap-mode:"repl"`
	CompletionPrev     key.Binding `keymap-mode:"repl"`
	CompletionNext     key.Binding `keymap-mode:"repl"`
	CompletionPageUp   key.Binding `keymap-mode:"repl"`
	CompletionPageDown key.Binding `keymap-mode:"repl"`
	CompletionCancel   key.Binding `keymap-mode:"repl"`

	HelpDrawerToggle  key.Binding `keymap-mode:"repl"`
	HelpDrawerRefresh key.Binding `keymap-mode:"repl"`
	HelpDrawerPin     key.Binding `keymap-mode:"repl"`
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
		CompletionTrigger: key.NewBinding(
			key.WithKeys("ctrl+space"),
			key.WithHelp("ctrl+space", "complete"),
		),
		CompletionAccept: key.NewBinding(
			key.WithKeys("ctrl+y"),
			key.WithHelp("ctrl+y", "accept completion"),
		),
		CompletionPrev: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "prev completion"),
		),
		CompletionNext: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "next completion"),
		),
		CompletionPageUp: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("ctrl+b", "completion page up"),
		),
		CompletionPageDown: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "completion page down"),
		),
		CompletionCancel: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "cancel completion"),
		),
		HelpDrawerToggle: key.NewBinding(
			key.WithKeys("alt+h"),
			key.WithHelp("alt+h", "toggle help drawer"),
		),
		HelpDrawerRefresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh help drawer"),
		),
		HelpDrawerPin: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "pin help drawer"),
		),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NextPane, k.Command, k.Select, k.HelpDrawerToggle, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextPane, k.PrevPane, k.Command, k.Back, k.Quit},
		{k.Up, k.Down, k.Top, k.Bottom, k.HalfUp, k.HalfDown},
		{k.Select},
		{k.CompletionTrigger, k.CompletionAccept, k.CompletionPrev, k.CompletionNext, k.CompletionCancel},
		{k.HelpDrawerToggle, k.HelpDrawerRefresh, k.HelpDrawerPin},
	}
}
