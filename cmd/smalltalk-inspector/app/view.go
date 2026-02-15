package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/internal/inspectorui"
	"github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
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

	// If we have an error with stack trace, show error view
	if m.showingError && m.errorInfo != nil {
		return m.renderErrorView(header, status, helpView, repl, contentHeight)
	}

	// If we have an inspect object from REPL, show two-pane layout
	if m.inspectObj != nil && len(m.inspectProps) > 0 {
		return m.renderInspectView(header, status, helpView, repl, contentHeight)
	}

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

func (m Model) renderErrorView(header, status, helpView, repl string, contentHeight int) string {
	stackWidth := m.width / 2
	sourceWidth := m.width - stackWidth
	if stackWidth < 20 {
		stackWidth = 20
	}
	if sourceWidth < 20 {
		sourceWidth = 20
	}

	// Error banner
	errorBanner := styleReplError.Render(" ✗ " + m.errorInfo.Message + " ")
	bannerLine := errorBanner + styleSeparator.Render(strings.Repeat("─", maxInt(0, m.width-ansi.StringWidth(errorBanner))))
	contentHeight-- // banner takes one line

	stackPane := m.renderStackPane(stackWidth, contentHeight)
	sourcePane := m.renderSourcePane(sourceWidth, contentHeight)

	content := lipgloss.JoinHorizontal(lipgloss.Top, stackPane, sourcePane)

	parts := []string{header, padRight(bannerLine, m.width), content, repl, status, helpView}
	if m.cmdActive {
		parts = append(parts, m.renderCommandLine())
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) renderStackPane(width, height int) string {
	label := " Call Stack "
	headerLine := stylePaneHeaderActive.Render(label) +
		styleSeparator.Render(strings.Repeat("─", maxInt(0, width-ansi.StringWidth(label))))

	contentHeight := height - 1
	if contentHeight < 1 {
		contentHeight = 1
	}

	var rows []string
	if m.errorInfo == nil || len(m.errorInfo.Frames) == 0 {
		rows = append(rows, padRight(styleEmptyHint.Render(" (no stack frames)"), width))
	} else {
		for i, frame := range m.errorInfo.Frames {
			marker := "  "
			if i == m.stackIdx {
				marker = styleSelectedMarker.Render("▸ ")
			}

			line := fmt.Sprintf("%s#%d  %-16s  %s:%d:%d",
				marker, i, frame.FunctionName, frame.FileName, frame.Line, frame.Column)
			rows = append(rows, padRight(line, width))
		}
	}

	vp := m.stackViewport
	vp.Width = width
	vp.Height = contentHeight
	vp.SetContent(strings.Join(rows, "\n"))

	body := lipgloss.NewStyle().Width(width).Height(contentHeight).Render(vp.View())
	pane := lipgloss.JoinVertical(
		lipgloss.Left,
		padRight(headerLine, width),
		body,
	)
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(pane)
}

func (m Model) renderInspectView(header, status, helpView, repl string, contentHeight int) string {
	inspectWidth := m.width / 2
	sourceWidth := m.width - inspectWidth
	if inspectWidth < 20 {
		inspectWidth = 20
	}
	if sourceWidth < 20 {
		sourceWidth = 20
	}

	// Breadcrumb bar
	breadcrumb := ""
	if len(m.navStack) > 0 {
		var crumbs []string
		for _, frame := range m.navStack {
			crumbs = append(crumbs, frame.Label)
		}
		crumbs = append(crumbs, m.inspectLabel)
		breadcrumb = stylePaneHeaderActive.Render(" Breadcrumb: "+strings.Join(crumbs, " → ")+" ") +
			styleSeparator.Render(strings.Repeat("─", maxInt(0, m.width-20)))
		contentHeight-- // breadcrumb takes one line
	}

	inspectPane := m.renderInspectPane(inspectWidth, contentHeight)
	sourcePane := m.renderSourcePane(sourceWidth, contentHeight)

	content := lipgloss.JoinHorizontal(lipgloss.Top, inspectPane, sourcePane)

	var parts []string
	parts = append(parts, header)
	if breadcrumb != "" {
		parts = append(parts, padRight(breadcrumb, m.width))
	}
	parts = append(parts, content, repl, status, helpView)
	if m.cmdActive {
		parts = append(parts, m.renderCommandLine())
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) renderInspectPane(width, height int) string {
	// Header with expression label
	label := fmt.Sprintf(" REPL Result: %s ", m.inspectLabel)
	if len(label) > width-2 {
		label = label[:width-2]
	}
	headerLine := stylePaneHeaderActive.Render(label) +
		styleSeparator.Render(strings.Repeat("─", maxInt(0, width-ansi.StringWidth(label))))

	contentHeight := height - 1
	if contentHeight < 1 {
		contentHeight = 1
	}

	var rows []string
	// Show properties
	for i, prop := range m.inspectProps {
		marker := "  "
		if i == m.inspectIdx {
			marker = styleSelectedMarker.Render("▸ ")
		}

		icon := memberKindIcon(prop.Kind)
		var iconStyled string
		switch prop.Kind {
		case "function":
			iconStyled = styleItemFunction.Render(icon)
		default:
			iconStyled = styleItemValue.Render(icon)
		}

		kindLabel := styleGutterNormal.Render(fmt.Sprintf("  %-9s", prop.Kind))
		name := prop.Name
		if prop.IsSymbol {
			name = "[" + name + "]"
		}

		line := marker + iconStyled + "  " + name + " : " + prop.Preview + kindLabel
		rows = append(rows, padRight(line, width))
	}

	if len(rows) == 0 {
		rows = append(rows, padRight(styleEmptyHint.Render(" (no properties)"), width))
	}

	vp := m.inspectViewport
	vp.Width = width
	vp.Height = contentHeight
	vp.SetContent(strings.Join(rows, "\n"))

	body := lipgloss.NewStyle().Width(width).Height(contentHeight).Render(vp.View())
	pane := lipgloss.JoinVertical(
		lipgloss.Left,
		padRight(headerLine, width),
		body,
	)
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(pane)
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
		start, end := inspectorui.VisibleRange(m.globalScroll, len(m.globals), contentHeight)
		for i := start; i < end; i++ {
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
		start, end := inspectorui.VisibleRange(m.memberScroll, len(m.members), contentHeight)

		for i := start; i < end; i++ {
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
		} else if m.rtSession != nil {
			// Show runtime proto chain names for non-class globals
			val := m.rtSession.GlobalValue(g.Name)
			if val != nil && !goja.IsUndefined(val) && !goja.IsNull(val) {
				if obj, ok := val.(*goja.Object); ok {
					names := runtime.PrototypeChainNames(obj, m.rtSession.VM)
					if len(names) > 0 {
						protoInfo = "proto: " + strings.Join(names, " → ")
					}
				}
			}
		}
	}
	footer := styleProtoChain.Render(" " + protoInfo)
	lines = append(lines, padRight(footer, width))

	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
}

func (m Model) renderSourcePane(width, height int) string {
	var lines []string

	// Determine which source to display
	srcLines := m.sourceLines
	headerSuffix := ""
	if m.showingReplSrc && len(m.replSourceLines) > 0 {
		srcLines = m.replSourceLines
		headerSuffix = " (REPL)"
	}

	// Header
	label := " Source" + headerSuffix + " "
	if !m.showingReplSrc && m.focus == FocusMembers && len(m.members) > 0 && m.memberIdx < len(m.members) {
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

	if len(srcLines) == 0 {
		lines = append(lines, padRight(styleEmptyHint.Render(" (no source)"), width))
		for len(lines) < height {
			lines = append(lines, strings.Repeat(" ", width))
		}
		return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
	}

	// Select the right syntax spans
	var syntaxSpans []jsparse.SyntaxSpan
	if m.showingReplSrc {
		syntaxSpans = m.replSyntaxSpans
	} else {
		syntaxSpans = m.fileSyntaxSpans
	}

	gutterWidth := len(fmt.Sprintf("%d", len(srcLines))) + 1

	endIdx := minInt(m.sourceScroll+contentHeight, len(srcLines))
	for lineIdx := m.sourceScroll; lineIdx < endIdx; lineIdx++ {
		lineNum := fmt.Sprintf("%*d ", gutterWidth, lineIdx+1)
		content := srcLines[lineIdx]

		isTarget := lineIdx == m.sourceTarget
		gs := styleGutterNormal
		if isTarget {
			gs = styleGutterCursor
		}
		gutter := gs.Render(lineNum)

		if isTarget {
			content = styleSourceHL.Render(content)
		} else if len(syntaxSpans) > 0 {
			content = renderSyntaxLine(content, lineIdx+1, syntaxSpans)
		}

		line := gutter + content
		lines = append(lines, padRight(line, width))
	}

	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
}

// renderSyntaxLine applies syntax highlighting to a single source line.
// lineNo is 1-based to match SyntaxSpan coordinates.
func renderSyntaxLine(line string, lineNo int, spans []jsparse.SyntaxSpan) string {
	if len(spans) == 0 || len(line) == 0 {
		return line
	}
	var b strings.Builder
	for colIdx, ch := range line {
		colNo := colIdx + 1 // 1-based
		class := jsparse.SyntaxClassAt(spans, lineNo, colNo)
		b.WriteString(jsparse.RenderSyntaxChar(class, string(ch)))
	}
	return b.String()
}

func (m Model) renderReplArea() string {
	label := " REPL "
	hStyle := stylePaneHeaderInactive
	if m.focus == FocusRepl {
		hStyle = stylePaneHeaderActive
	}
	separator := hStyle.Render(label) + styleSeparator.Render(strings.Repeat("─", maxInt(0, m.width-len(label))))

	// Result/error line
	var resultLine string
	if m.replError != "" {
		// Show first line of error
		errLines := strings.Split(m.replError, "\n")
		resultLine = styleReplError.Render("✗ " + errLines[0])
	} else if m.replResult != "" {
		resultLine = "→ " + m.replResult
	} else if len(m.replHistory) > 0 {
		resultLine = styleEmptyHint.Render("  " + m.replHistory[len(m.replHistory)-1])
	}

	// Prompt line
	var promptLine string
	if m.focus == FocusRepl {
		promptLine = m.replInput.View()
	} else {
		promptLine = styleReplPrompt.Render("» ") + styleEmptyHint.Render("(tab to REPL)")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		padRight(separator, m.width),
		padRight(resultLine, m.width),
		padRight(promptLine, m.width),
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
