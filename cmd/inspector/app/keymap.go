package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines key bindings for the inspector model.
// keymap-mode tags are consumed by bobatea's mode-keymap helper.
type KeyMap struct {
	Quit key.Binding `keymap-mode:"*"`

	NextPane   key.Binding `keymap-mode:"*"`
	OpenDrawer key.Binding `keymap-mode:"source,tree"`
	Close      key.Binding `keymap-mode:"*"`

	Yank key.Binding `keymap-mode:"source,tree"`

	Up        key.Binding `keymap-mode:"source,tree,drawer"`
	Down      key.Binding `keymap-mode:"source,tree,drawer"`
	Left      key.Binding `keymap-mode:"source,tree,drawer"`
	Right     key.Binding `keymap-mode:"source,tree,drawer"`
	Top       key.Binding `keymap-mode:"source,tree"`
	Bottom    key.Binding `keymap-mode:"source,tree"`
	HalfDown  key.Binding `keymap-mode:"source,tree"`
	HalfUp    key.Binding `keymap-mode:"source,tree"`
	Toggle    key.Binding `keymap-mode:"tree"`
	Apply     key.Binding `keymap-mode:"tree"`
	GoToDef   key.Binding `keymap-mode:"source,tree,drawer"`
	FindUsage key.Binding `keymap-mode:"source,tree,drawer"`

	Complete   key.Binding `keymap-mode:"drawer"`
	DrawGrow   key.Binding `keymap-mode:"drawer"`
	DrawShrink key.Binding `keymap-mode:"drawer"`
}

func newKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
		NextPane: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next pane"),
		),
		OpenDrawer: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "drawer"),
		),
		Close: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close/clear"),
		),
		Yank: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yank to drawer"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
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
			key.WithHelp("ctrl+d", "half-page down"),
		),
		HalfUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "half-page up"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle node"),
		),
		Apply: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "jump/apply"),
		),
		GoToDef: key.NewBinding(
			key.WithKeys("d", "ctrl+d"),
			key.WithHelp("d/ctrl+d", "go to def"),
		),
		FindUsage: key.NewBinding(
			key.WithKeys("*", "ctrl+g"),
			key.WithHelp("*/ctrl+g", "usages"),
		),
		Complete: key.NewBinding(
			key.WithKeys("ctrl+space", "ctrl+@", "ctrl+n"),
			key.WithHelp("ctrl+n", "complete"),
		),
		DrawGrow: key.NewBinding(
			key.WithKeys("ctrl+up"),
			key.WithHelp("ctrl+↑", "grow drawer"),
		),
		DrawShrink: key.NewBinding(
			key.WithKeys("ctrl+down"),
			key.WithHelp("ctrl+↓", "shrink drawer"),
		),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NextPane, k.OpenDrawer, k.GoToDef, k.FindUsage, k.Close, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextPane, k.OpenDrawer, k.Close, k.Quit},
		{k.Up, k.Down, k.Left, k.Right, k.Top, k.Bottom, k.HalfUp, k.HalfDown},
		{k.Toggle, k.Apply, k.Yank, k.GoToDef, k.FindUsage, k.Complete},
		{k.DrawGrow, k.DrawShrink},
	}
}
