package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Drawer is a bottom-drawer code editor with tree-sitter incremental parsing.
type Drawer struct {
	// Editor buffer
	lines     []string
	cursorRow int // 0-based
	cursorCol int // 0-based
	scrollY   int // first visible line

	// Dimensions
	height    int
	minHeight int
	maxHeight int

	// tree-sitter
	tsParser *TSParser
	tsRoot   *TSNode // current CST snapshot

	// Reference to file-level goja index (for scope lookups)
	gojaIndex *Index

	// Completion state
	completionActive bool
	completionItems  []CompletionCandidate
	completionIdx    int
	completionCtx    CompletionContext
}

// NewDrawer creates a bottom drawer with a tree-sitter parser.
func NewDrawer(gojaIndex *Index) *Drawer {
	tsParser, _ := NewTSParser() // error is non-fatal; completion just won't work
	return &Drawer{
		lines:     []string{""},
		minHeight: 3,
		maxHeight: 20,
		height:    6,
		tsParser:  tsParser,
		gojaIndex: gojaIndex,
	}
}

// Close releases tree-sitter resources.
func (d *Drawer) Close() {
	if d.tsParser != nil {
		d.tsParser.Close()
	}
}

// Source returns the drawer content as a byte slice.
func (d *Drawer) Source() []byte {
	return []byte(strings.Join(d.lines, "\n"))
}

// Reparse triggers a tree-sitter incremental parse of the drawer content.
func (d *Drawer) Reparse() {
	if d.tsParser == nil {
		return
	}
	d.tsRoot = d.tsParser.Parse(d.Source())
}

// --- Editor operations ---

// InsertChar inserts a character at the cursor position.
func (d *Drawer) InsertChar(ch rune) {
	line := d.lines[d.cursorRow]
	runes := []rune(line)
	col := d.cursorCol
	if col > len(runes) {
		col = len(runes)
	}
	runes = append(runes[:col], append([]rune{ch}, runes[col:]...)...)
	d.lines[d.cursorRow] = string(runes)
	d.cursorCol = col + 1
}

// InsertNewline splits the current line at the cursor.
func (d *Drawer) InsertNewline() {
	line := d.lines[d.cursorRow]
	runes := []rune(line)
	col := d.cursorCol
	if col > len(runes) {
		col = len(runes)
	}

	before := string(runes[:col])
	after := string(runes[col:])
	d.lines[d.cursorRow] = before

	// Insert new line after current
	newLines := make([]string, 0, len(d.lines)+1)
	newLines = append(newLines, d.lines[:d.cursorRow+1]...)
	newLines = append(newLines, after)
	newLines = append(newLines, d.lines[d.cursorRow+1:]...)
	d.lines = newLines

	d.cursorRow++
	d.cursorCol = 0
}

// DeleteBack deletes the character before the cursor (backspace).
func (d *Drawer) DeleteBack() {
	if d.cursorCol > 0 {
		line := d.lines[d.cursorRow]
		runes := []rune(line)
		col := d.cursorCol
		if col > len(runes) {
			col = len(runes)
		}
		runes = append(runes[:col-1], runes[col:]...)
		d.lines[d.cursorRow] = string(runes)
		d.cursorCol = col - 1
	} else if d.cursorRow > 0 {
		// Join with previous line
		prevLine := d.lines[d.cursorRow-1]
		curLine := d.lines[d.cursorRow]
		d.lines[d.cursorRow-1] = prevLine + curLine
		d.lines = append(d.lines[:d.cursorRow], d.lines[d.cursorRow+1:]...)
		d.cursorRow--
		d.cursorCol = len([]rune(prevLine))
	}
}

// MoveCursor moves the cursor by (dy, dx).
func (d *Drawer) MoveCursor(dy, dx int) {
	d.cursorRow += dy
	d.cursorCol += dx

	// Clamp row
	if d.cursorRow < 0 {
		d.cursorRow = 0
	}
	if d.cursorRow >= len(d.lines) {
		d.cursorRow = len(d.lines) - 1
	}

	// Clamp col
	lineLen := len([]rune(d.lines[d.cursorRow]))
	if d.cursorCol < 0 {
		d.cursorCol = 0
	}
	if d.cursorCol > lineLen {
		d.cursorCol = lineLen
	}

	d.ensureCursorVisible()
}

// MoveHome moves cursor to start of line.
func (d *Drawer) MoveHome() { d.cursorCol = 0 }

// MoveEnd moves cursor to end of line.
func (d *Drawer) MoveEnd() { d.cursorCol = len([]rune(d.lines[d.cursorRow])) }

func (d *Drawer) ensureCursorVisible() {
	editorHeight := d.editorHeight()
	if editorHeight <= 0 {
		return
	}
	if d.cursorRow < d.scrollY {
		d.scrollY = d.cursorRow
	}
	if d.cursorRow >= d.scrollY+editorHeight {
		d.scrollY = d.cursorRow - editorHeight + 1
	}
}

func (d *Drawer) editorHeight() int {
	return d.height - 1 // minus header
}

// Grow increases drawer height by delta (clamped).
func (d *Drawer) Grow(delta int) {
	d.height += delta
	if d.height < d.minHeight {
		d.height = d.minHeight
	}
	if d.height > d.maxHeight {
		d.height = d.maxHeight
	}
}

// --- Rendering ---

// Render renders the drawer as a string with given width and height.
func (d *Drawer) Render(width, height int, focused bool) string {
	d.height = height
	if d.height < d.minHeight {
		d.height = d.minHeight
	}

	leftWidth := width / 2
	rightWidth := width - leftWidth

	left := d.renderEditor(leftWidth, d.height, focused)
	right := d.renderTree(rightWidth, d.height)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (d *Drawer) renderEditor(width, height int, focused bool) string {
	var lines []string
	gutterWidth := len(fmt.Sprintf("%d", len(d.lines))) + 1

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("240"))
	paneLabel := " DRAWER "
	if focused {
		headerStyle = headerStyle.Background(lipgloss.Color("33"))
	}
	headerLine := headerStyle.Render(paneLabel) + strings.Repeat("─", max(0, width-len(paneLabel)))
	lines = append(lines, padRight(headerLine, width))

	editorHeight := height - 1
	if editorHeight < 1 {
		editorHeight = 1
	}

	cursorStyle := lipgloss.NewStyle().Reverse(true).Bold(true)
	gutterNormal := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	gutterCursor := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)

	for i := 0; i < editorHeight; i++ {
		lineIdx := d.scrollY + i
		if lineIdx >= len(d.lines) {
			lines = append(lines, strings.Repeat(" ", width))
			continue
		}

		lineNum := fmt.Sprintf("%*d ", gutterWidth, lineIdx+1)
		content := d.lines[lineIdx]
		isCursorLine := (lineIdx == d.cursorRow)

		gs := gutterNormal
		if isCursorLine {
			gs = gutterCursor
		}
		gutter := gs.Render(lineNum)

		var rendered strings.Builder
		runes := []rune(content)
		for col := 0; col < len(runes); col++ {
			ch := string(runes[col])
			if focused && isCursorLine && col == d.cursorCol {
				rendered.WriteString(cursorStyle.Render(ch))
			} else {
				rendered.WriteString(ch)
			}
		}
		// Cursor at end of line
		if focused && isCursorLine && d.cursorCol >= len(runes) {
			rendered.WriteString(cursorStyle.Render(" "))
		}

		lines = append(lines, padRight(gutter+rendered.String(), width))
	}

	// Overlay completion popup if active
	if d.completionActive && len(d.completionItems) > 0 {
		popup := d.renderCompletionPopup()
		if popup != "" {
			// Find which rendered line the cursor is on
			cursorVisRow := d.cursorRow - d.scrollY + 1 // +1 for header
			if cursorVisRow >= 0 && cursorVisRow < len(lines)-1 {
				// Insert popup lines after the cursor line
				popupLines := strings.Split(popup, "\n")
				insertAt := cursorVisRow + 1
				for pi, pl := range popupLines {
					targetRow := insertAt + pi
					if targetRow < len(lines) {
						// Overlay: replace part of the line with popup content
						lines[targetRow] = padRight(pl, width)
					}
				}
			}
		}
	}

	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
}

func (d *Drawer) renderCompletionPopup() string {
	if !d.completionActive || len(d.completionItems) == 0 {
		return ""
	}

	maxVisible := 8
	if len(d.completionItems) < maxVisible {
		maxVisible = len(d.completionItems)
	}

	// Compute scroll window around selected item
	startIdx := 0
	if d.completionIdx >= maxVisible {
		startIdx = d.completionIdx - maxVisible + 1
	}

	popupStyle := lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("252"))
	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("15")).Bold(true)

	var lines []string
	for i := startIdx; i < startIdx+maxVisible && i < len(d.completionItems); i++ {
		item := d.completionItems[i]
		icon := item.Kind.Icon()
		label := fmt.Sprintf(" %s %s  %s ", icon, item.Label, item.Detail)
		if i == d.completionIdx {
			lines = append(lines, selectedStyle.Render(label))
		} else {
			lines = append(lines, popupStyle.Render(label))
		}
	}

	return strings.Join(lines, "\n")
}

func (d *Drawer) renderTree(width, height int) string {
	var lines []string

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("240"))
	paneLabel := " CST "
	headerLine := headerStyle.Render(paneLabel) + strings.Repeat("─", max(0, width-len(paneLabel)))
	lines = append(lines, padRight(headerLine, width))

	contentHeight := height - 1
	if contentHeight < 1 {
		contentHeight = 1
	}

	if d.tsRoot == nil {
		lines = append(lines, padRight(" (no parse)", width))
		for len(lines) < height {
			lines = append(lines, strings.Repeat(" ", width))
		}
		return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
	}

	// Flatten tree-sitter CST into visible lines
	var flat []string
	d.flattenTSNode(d.tsRoot, "", &flat, 0, 50)

	for i := 0; i < contentHeight; i++ {
		if i < len(flat) {
			lines = append(lines, padRight(flat[i], width))
		} else {
			lines = append(lines, strings.Repeat(" ", width))
		}
	}

	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
}

func (d *Drawer) flattenTSNode(n *TSNode, indent string, out *[]string, depth, maxDepth int) {
	if n == nil || depth > maxDepth {
		return
	}

	label := n.Kind
	if n.IsError {
		label = "ERROR"
	}
	if n.IsMissing {
		label += " MISSING"
	}
	if n.Text != "" && len(n.Text) < 30 {
		label += fmt.Sprintf(" %q", n.Text)
	}
	span := fmt.Sprintf(" [%d:%d..%d:%d]", n.StartRow, n.StartCol, n.EndRow, n.EndCol)

	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	if n.IsError {
		*out = append(*out, indent+errStyle.Render(label+span))
	} else {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		*out = append(*out, indent+label+dimStyle.Render(span))
	}

	for _, child := range n.Children {
		d.flattenTSNode(child, indent+"  ", out, depth+1, maxDepth)
	}
}

// CursorInfo returns a status string about the drawer cursor.
func (d *Drawer) CursorInfo() string {
	parts := []string{fmt.Sprintf("drawer: %d:%d", d.cursorRow+1, d.cursorCol+1)}
	if d.tsRoot != nil {
		if d.tsRoot.HasError() {
			parts = append(parts, "⚠ ts-error")
		} else {
			parts = append(parts, "✓ ts-ok")
		}
		// Find node at cursor
		node := d.tsRoot.NodeAtPosition(d.cursorRow, d.cursorCol)
		if node != nil {
			label := node.Kind
			if node.Text != "" {
				label += fmt.Sprintf(" %q", node.Text)
			}
			parts = append(parts, label)
		}
	}
	if d.completionActive {
		parts = append(parts, fmt.Sprintf("completion: %d items", len(d.completionItems)))
	}
	return strings.Join(parts, " │ ")
}

// suppress unused import warning
var _ = ansi.StringWidth
