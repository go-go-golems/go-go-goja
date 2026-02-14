package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	if !m.loaded {
		return m.renderEmptyView()
	}

	return m.renderLoadedView()
}

func (m Model) renderEmptyView() string {
	header := m.renderHeader()
	status := m.renderStatusBar()
	helpView := m.renderHelp()
	repl := m.renderReplArea()

	// Empty body with hints
	bodyHeight := m.height - 4 // header + status + help + repl separator
	if m.cmdActive {
		bodyHeight--
	}
	bodyHeight -= 3 // REPL
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, "")
	lines = append(lines, styleEmptyHint.Render("          No program loaded."))
	lines = append(lines, "")
	lines = append(lines, styleEmptyHint.Render("  :load <file.js>    Load a JavaScript file"))
	lines = append(lines, styleEmptyHint.Render("  :help              Show all commands"))
	lines = append(lines, styleEmptyHint.Render("  :quit              Exit"))

	for len(lines) < bodyHeight {
		lines = append(lines, "")
	}

	body := strings.Join(lines[:minInt(len(lines), bodyHeight)], "\n")

	parts := []string{header, body, repl, status, helpView}
	if m.cmdActive {
		parts = append(parts, m.renderCommandLine())
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) renderLoadedView() string {
	header := m.renderHeader()
	status := m.renderStatusBar()
	helpView := m.renderHelp()
	repl := m.renderReplArea()

	contentHeight := m.contentHeight()

	// Three-pane layout: globals | members | source
	globalsWidth := m.width / 4
	membersWidth := m.width / 4
	sourceWidth := m.width - globalsWidth - membersWidth
	if globalsWidth < 10 {
		globalsWidth = 10
	}
	if membersWidth < 10 {
		membersWidth = 10
	}
	if sourceWidth < 10 {
		sourceWidth = 10
	}

	globalsPane := m.renderGlobalsPane(globalsWidth, contentHeight)
	membersPane := m.renderMembersPane(membersWidth, contentHeight)
	sourcePane := m.renderSourcePane(sourceWidth, contentHeight)

	content := lipgloss.JoinHorizontal(lipgloss.Top, globalsPane, membersPane, sourcePane)

	parts := []string{header, content, repl, status, helpView}
	if m.cmdActive {
		parts = append(parts, m.renderCommandLine())
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) renderHeader() string {
	title := "goja-inspector"
	if m.filename != "" {
		title += " ─── " + m.filename
	}

	focusLabel := strings.ToUpper(m.mode)
	gap := m.width - len(title) - len(focusLabel) - 4
	if gap < 1 {
		gap = 1
	}

	return styleHeader.Width(m.width).Render(
		title + strings.Repeat(" ", gap) + "[" + focusLabel + "]",
	)
}

func (m Model) renderGlobalsPane(width, height int) string {
	var lines []string

	// Header
	label := " Globals "
	hStyle := stylePaneHeaderInactive
	if m.focus == FocusGlobals {
		hStyle = stylePaneHeaderActive
	}
	headerLine := hStyle.Render(label) + styleSeparator.Render(strings.Repeat("─", maxInt(0, width-ansi.StringWidth(label))))
	lines = append(lines, padRight(headerLine, width))

	contentHeight := height - 2 // header + footer
	if contentHeight < 1 {
		contentHeight = 1
	}

	if len(m.globals) == 0 {
		lines = append(lines, padRight(styleEmptyHint.Render(" (no globals)"), width))
		for len(lines) < height-1 {
			lines = append(lines, strings.Repeat(" ", width))
		}
	} else {
		endIdx := minInt(m.globalScroll+contentHeight, len(m.globals))
		for i := m.globalScroll; i < endIdx; i++ {
			g := m.globals[i]
			marker := "  "
			if i == m.globalIdx {
				marker = styleSelectedMarker.Render("▸ ")
			}

			icon := kindIcon(g.Kind)
			var iconStyled string
			//exhaustive:ignore
			switch g.Kind {
			case 4: // BindingClass
				iconStyled = styleItemClass.Render(icon)
			case 3: // BindingFunction
				iconStyled = styleItemFunction.Render(icon)
			default:
				iconStyled = styleItemValue.Render(icon)
			}

			name := g.Name
			ext := ""
			if g.Extends != "" {
				ext = stylePaneHeaderInactive.Render(" ← " + g.Extends)
			}

			line := marker + iconStyled + "  " + name + ext
			lines = append(lines, padRight(line, width))
		}
		for len(lines) < height-1 {
			lines = append(lines, strings.Repeat(" ", width))
		}
	}

	footer := fmt.Sprintf(" %d bindings", len(m.globals))
	lines = append(lines, padRight(styleGutterNormal.Render(footer), width))

	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
}

func (m Model) renderMembersPane(width, height int) string {
	var lines []string

	// Header
	label := " Members "
	if len(m.globals) > 0 && m.globalIdx < len(m.globals) {
		label = fmt.Sprintf(" %s: Members ", m.globals[m.globalIdx].Name)
	}
	hStyle := stylePaneHeaderInactive
	if m.focus == FocusMembers {
		hStyle = stylePaneHeaderActive
	}
	headerLine := hStyle.Render(label) + styleSeparator.Render(strings.Repeat("─", maxInt(0, width-ansi.StringWidth(label))))
	lines = append(lines, padRight(headerLine, width))

	contentHeight := height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	if len(m.members) == 0 {
		hint := " (select a global)"
		if len(m.globals) > 0 {
			hint = " (no members)"
		}
		lines = append(lines, padRight(styleEmptyHint.Render(hint), width))
		for len(lines) < height-1 {
			lines = append(lines, strings.Repeat(" ", width))
		}
	} else {
		// Show members with own/inherited separation
		lastInherited := false
		rendered := 0
		endIdx := minInt(m.memberScroll+contentHeight, len(m.members))

		for i := m.memberScroll; i < endIdx; i++ {
			mem := m.members[i]

			// Insert inherited section header
			if mem.Inherited && !lastInherited && rendered < contentHeight {
				if rendered > 0 {
					inhHeader := styleInheritedHeader.Render(fmt.Sprintf(" ── inherited (%s) ", mem.Source))
					lines = append(lines, padRight(inhHeader, width))
					rendered++
					if rendered >= contentHeight {
						break
					}
				}
				lastInherited = true
			}

			marker := "  "
			if i == m.memberIdx {
				marker = styleSelectedMarker.Render("▸ ")
			}

			icon := memberKindIcon(mem.Kind)
			var iconStyled string
			switch mem.Kind {
			case "function":
				iconStyled = styleItemFunction.Render(icon)
			default:
				iconStyled = styleItemValue.Render(icon)
			}

			line := marker + iconStyled + "  " + mem.Name + mem.Preview
			lines = append(lines, padRight(line, width))
			rendered++
		}

		for len(lines) < height-1 {
			lines = append(lines, strings.Repeat(" ", width))
		}
	}

	// Proto chain footer
	protoInfo := ""
	if len(m.globals) > 0 && m.globalIdx < len(m.globals) {
		g := m.globals[m.globalIdx]
		if g.Extends != "" {
			protoInfo = fmt.Sprintf("proto: %s → Object", g.Extends)
		}
	}
	footer := styleProtoChain.Render(" " + protoInfo)
	lines = append(lines, padRight(footer, width))

	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
}

func (m Model) renderSourcePane(width, height int) string {
	var lines []string

	// Header
	label := " Source "
	if m.focus == FocusMembers && len(m.members) > 0 && m.memberIdx < len(m.members) {
		label = fmt.Sprintf(" Source: %s ", m.members[m.memberIdx].Name)
	}
	hStyle := stylePaneHeaderInactive
	if m.focus == FocusSource {
		hStyle = stylePaneHeaderActive
	}
	headerLine := hStyle.Render(label) + styleSeparator.Render(strings.Repeat("─", maxInt(0, width-ansi.StringWidth(label))))
	lines = append(lines, padRight(headerLine, width))

	contentHeight := height - 1
	if contentHeight < 1 {
		contentHeight = 1
	}

	if len(m.sourceLines) == 0 {
		lines = append(lines, padRight(styleEmptyHint.Render(" (no source)"), width))
		for len(lines) < height {
			lines = append(lines, strings.Repeat(" ", width))
		}
		return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
	}

	gutterWidth := len(fmt.Sprintf("%d", len(m.sourceLines))) + 1

	endIdx := minInt(m.sourceScroll+contentHeight, len(m.sourceLines))
	for lineIdx := m.sourceScroll; lineIdx < endIdx; lineIdx++ {
		lineNum := fmt.Sprintf("%*d ", gutterWidth, lineIdx+1)
		content := m.sourceLines[lineIdx]

		isTarget := lineIdx == m.sourceTarget
		gs := styleGutterNormal
		if isTarget {
			gs = styleGutterCursor
		}
		gutter := gs.Render(lineNum)

		if isTarget {
			content = styleSourceHL.Render(content)
		}

		line := gutter + content
		lines = append(lines, padRight(line, width))
	}

	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
}

func (m Model) renderReplArea() string {
	sepStyle := styleSeparator
	separator := sepStyle.Render("─── REPL " + strings.Repeat("─", maxInt(0, m.width-9)))

	prompt := styleReplPrompt.Render("» ") + styleEmptyHint.Render("(press : to enter commands)")
	result := ""

	return lipgloss.JoinVertical(lipgloss.Left,
		padRight(separator, m.width),
		padRight(prompt, m.width),
		padRight(result, m.width),
	)
}

func (m Model) renderStatusBar() string {
	var parts []string

	if m.loaded {
		if m.analysis != nil && m.analysis.ParseErr != nil {
			parts = append(parts, styleReplError.Render("⚠ parse error"))
		} else {
			parts = append(parts, "✓ parse ok")
		}
		parts = append(parts, fmt.Sprintf("globals: %d", len(m.globals)))
		if len(m.globals) > 0 {
			parts = append(parts, fmt.Sprintf("selected: %s", m.globals[m.globalIdx].Name))
		}
	}

	if m.statusMsg != "" {
		parts = append(parts, m.statusMsg)
	}

	return styleStatus.Width(m.width).Render(formatStatus(parts...))
}

func (m Model) renderHelp() string {
	return styleHelp.Width(m.width).Render(m.help.View(m.keyMap))
}

func (m Model) renderCommandLine() string {
	return styleCommandLine.Width(m.width).Render(m.command.View())
}

// padRight pads a (possibly ANSI-styled) string to exactly the given visible width.
func padRight(s string, width int) string {
	w := ansi.StringWidth(s)
	if w >= width {
		return ansi.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", width-w)
}
