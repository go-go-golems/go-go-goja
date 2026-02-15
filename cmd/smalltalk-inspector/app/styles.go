package app

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorSecondary   = lipgloss.Color("62")  // purple
	colorMuted       = lipgloss.Color("240") // dim gray
	colorBright      = lipgloss.Color("15")  // white
	colorDim         = lipgloss.Color("244") // mid gray
	colorSuccess     = lipgloss.Color("78")  // green
	colorError       = lipgloss.Color("196") // red - used in REPL error display
	colorWarning     = lipgloss.Color("226") // yellow
	colorHighlight   = lipgloss.Color("117") // light blue
	colorStatusBg    = lipgloss.Color("236") // dark gray
	colorPaneFocusBg = lipgloss.Color("33")  // blue (active pane header)
	colorPaneDimBg   = lipgloss.Color("240") // gray (inactive pane header)

	// Styles
	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBright).
			Background(colorSecondary).
			Padding(0, 1)

	stylePaneHeaderActive = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorBright).
				Background(colorPaneFocusBg)

	stylePaneHeaderInactive = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorBright).
				Background(colorPaneDimBg)

	styleStatus = lipgloss.NewStyle().
			Foreground(colorBright).
			Background(colorStatusBg).
			Padding(0, 1)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorDim).
			Padding(0, 1)

	styleCommandLine = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Background(lipgloss.Color("235")).
				Padding(0, 1)

	styleSeparator = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Pane content styles
	styleEmptyHint = lipgloss.NewStyle().
			Foreground(colorDim).
			Italic(true)

	styleItemClass    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // orange
	styleItemFunction = lipgloss.NewStyle().Foreground(lipgloss.Color("117")) // blue
	styleItemValue    = lipgloss.NewStyle().Foreground(lipgloss.Color("150")) // green

	styleGutterNormal = lipgloss.NewStyle().Foreground(colorDim)
	styleGutterCursor = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
	styleSourceHL     = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(colorHighlight)

	styleSelectedMarker = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)

	styleInheritedHeader = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)

	styleProtoChain = lipgloss.NewStyle().Foreground(colorDim).Italic(true)

	styleReplPrompt = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	styleReplError  = lipgloss.NewStyle().Foreground(colorError)
)
