package app

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/dop251/goja/parser"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

type focusPane int

const (
	focusEditor focusPane = iota
	focusTSSExpr
	focusASTSExpr
)

type editorMode int

const (
	editorModeInsert editorMode = iota
	editorModeASTSelect
)

type syntaxClass int

const (
	syntaxClassNone syntaxClass = iota
	syntaxClassKeyword
	syntaxClassString
	syntaxClassNumber
	syntaxClassComment
	syntaxClassIdentifier
	syntaxClassOperator
)

type syntaxSpan struct {
	startLine int // 1-based
	startCol  int // 1-based
	endLine   int // 1-based
	endCol    int // 1-based
	class     syntaxClass
}

type astParsedMsg struct {
	Seq      uint64
	ParseErr error
	ASTSExpr string
	ASTIndex *jsparse.Index
}

// Model drives the live 3-pane AST parse editor.
type Model struct {
	filename string

	lines     []string
	cursorRow int
	cursorCol int

	editorScroll int
	tsScroll     int
	astScroll    int

	focus focusPane

	editorMode      editorMode
	syntaxHighlight bool

	width  int
	height int

	tsParser *jsparse.TSParser
	tsRoot   *jsparse.TSNode
	tsSExpr  string

	cursorNode         *jsparse.TSNode
	highlightStartLine int // 1-based
	highlightStartCol  int // 1-based
	highlightEndLine   int // 1-based
	highlightEndCol    int // 1-based

	astSExpr    string
	astParseErr error
	astIndex    *jsparse.Index

	selectedASTNodeID jsparse.NodeID

	syntaxSpans []syntaxSpan

	parseSeq      uint64
	pendingSeq    uint64
	parseDebounce time.Duration
}

// NewModel creates a new live editor model.
func NewModel(filename, source string) *Model {
	lines := strings.Split(source, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	tsParser, _ := jsparse.NewTSParser()
	m := &Model{
		filename:          filename,
		lines:             lines,
		focus:             focusEditor,
		editorMode:        editorModeInsert,
		syntaxHighlight:   true,
		tsParser:          tsParser,
		parseDebounce:     120 * time.Millisecond,
		selectedASTNodeID: -1,
	}
	m.reparseCST()
	return m
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return m.scheduleASTParse()
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case astParsedMsg:
		if msg.Seq != m.pendingSeq {
			return m, nil
		}
		m.astParseErr = msg.ParseErr
		if msg.ParseErr == nil {
			m.astSExpr = msg.ASTSExpr
			m.astIndex = msg.ASTIndex
		} else {
			m.astSExpr = ""
			m.astIndex = nil
			m.selectedASTNodeID = -1
		}
		if m.editorMode == editorModeASTSelect {
			if !m.syncASTSelectionFromCursor() {
				m.updateCursorNodeHighlight()
			}
		}
		m.astScroll = clamp(m.astScroll, 0, maxInt(0, len(strings.Split(m.astTextForView(), "\n"))-1))
		return m, nil
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.Close()
		return m, tea.Quit
	case "tab":
		m.focus = (m.focus + 1) % 3
		return m, nil
	case "m":
		m.toggleEditorMode()
		return m, nil
	case "s":
		m.syntaxHighlight = !m.syntaxHighlight
		return m, nil
	}

	switch m.focus {
	case focusEditor:
		return m.handleEditorKey(msg)
	case focusTSSExpr:
		return m.handleScrollKey(msg, &m.tsScroll, m.tsSExpr)
	case focusASTSExpr:
		return m.handleScrollKey(msg, &m.astScroll, m.astTextForView())
	default:
		return m, nil
	}
}

func (m *Model) handleEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editorMode == editorModeASTSelect {
		return m.handleASTSelectKey(msg)
	}

	switch msg.String() {
	case "up":
		m.moveCursor(-1, 0)
		return m, nil
	case "down":
		m.moveCursor(1, 0)
		return m, nil
	case "left":
		m.moveCursor(0, -1)
		return m, nil
	case "right":
		m.moveCursor(0, 1)
		return m, nil
	case "home":
		m.cursorCol = 0
		m.ensureCursorVisible()
		m.updateCursorNodeHighlight()
		return m, nil
	case "end":
		m.cursorCol = len([]rune(m.lines[m.cursorRow]))
		m.ensureCursorVisible()
		m.updateCursorNodeHighlight()
		return m, nil
	case "enter":
		m.insertNewline()
		return m, m.postEdit()
	case "backspace":
		m.deleteBack()
		return m, m.postEdit()
	}

	if runes := msg.Runes; len(runes) > 0 {
		for _, r := range runes {
			m.insertChar(r)
		}
		return m, m.postEdit()
	}

	return m, nil
}

func (m *Model) handleASTSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "h", "left":
		m.astSelectParent()
	case "l", "right":
		m.astSelectFirstChild()
	case "j", "down":
		m.astSelectNextSibling()
	case "k", "up":
		m.astSelectPrevSibling()
	case "g":
		m.astSelectRoot()
	}
	return m, nil
}

func (m *Model) handleScrollKey(msg tea.KeyMsg, scroll *int, content string) (tea.Model, tea.Cmd) {
	lines := strings.Split(content, "\n")
	maxScroll := maxInt(0, len(lines)-1)

	switch msg.String() {
	case "up", "k":
		*scroll = clamp(*scroll-1, 0, maxScroll)
	case "down", "j":
		*scroll = clamp(*scroll+1, 0, maxScroll)
	case "pgup":
		*scroll = clamp(*scroll-10, 0, maxScroll)
	case "pgdown":
		*scroll = clamp(*scroll+10, 0, maxScroll)
	case "g":
		*scroll = 0
	case "G":
		*scroll = maxScroll
	}
	return m, nil
}

func (m *Model) postEdit() tea.Cmd {
	m.reparseCST()
	return m.scheduleASTParse()
}

func (m *Model) source() string {
	return strings.Join(m.lines, "\n")
}

func (m *Model) scheduleASTParse() tea.Cmd {
	m.parseSeq++
	seq := m.parseSeq
	source := m.source()
	m.pendingSeq = seq
	return parseASTCmd(seq, m.filename, source, m.parseDebounce)
}

func parseASTCmd(seq uint64, filename, source string, delay time.Duration) tea.Cmd {
	run := func() tea.Msg {
		program, parseErr := parser.ParseFile(nil, filename, source, 0)

		astSExpr := ""
		var astIndex *jsparse.Index
		if parseErr == nil && program != nil {
			astIndex = jsparse.BuildIndex(program, source)
			astSExpr = jsparse.ASTIndexToSExpr(astIndex, &jsparse.SExprOptions{
				IncludeSpan: true,
				IncludeText: true,
				MaxDepth:    80,
				MaxNodes:    8000,
			})
			if strings.TrimSpace(astSExpr) == "" {
				astSExpr = "(Program)"
			}
		}

		return astParsedMsg{
			Seq:      seq,
			ParseErr: parseErr,
			ASTSExpr: astSExpr,
			ASTIndex: astIndex,
		}
	}

	if delay <= 0 {
		return run
	}
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return run()
	})
}

func (m *Model) reparseCST() {
	if m.tsParser == nil {
		m.tsRoot = nil
		m.tsSExpr = "(tree-sitter unavailable)"
		m.syntaxSpans = nil
		m.clearCursorNodeHighlight()
		return
	}

	m.tsRoot = m.tsParser.Parse([]byte(m.source()))
	if m.tsRoot == nil {
		m.tsSExpr = "(no parse)"
		m.syntaxSpans = nil
		m.clearCursorNodeHighlight()
		return
	}

	m.tsSExpr = jsparse.CSTToSExpr(m.tsRoot, &jsparse.SExprOptions{
		IncludeSpan:  true,
		IncludeText:  true,
		IncludeFlags: true,
		MaxDepth:     80,
		MaxNodes:     8000,
	})
	m.rebuildSyntaxSpans()
	if m.editorMode == editorModeInsert {
		m.updateCursorNodeHighlight()
	}
}

func (m *Model) astTextForView() string {
	if m.astParseErr != nil {
		return fmt.Sprintf("(parse-error %q)", m.astParseErr.Error())
	}
	if strings.TrimSpace(m.astSExpr) == "" {
		return "(waiting-for-valid-parse)"
	}
	return m.astSExpr
}

func (m *Model) toggleEditorMode() {
	if m.editorMode == editorModeInsert {
		m.editorMode = editorModeASTSelect
		if !m.syncASTSelectionFromCursor() {
			m.selectedASTNodeID = -1
			m.updateCursorNodeHighlight()
		}
		return
	}

	m.editorMode = editorModeInsert
	m.selectedASTNodeID = -1
	m.updateCursorNodeHighlight()
}

func (m *Model) editorModeLabel() string {
	if m.editorMode == editorModeASTSelect {
		return "AST-SELECT"
	}
	return "INSERT"
}

func (m *Model) cursorOffset() int {
	offset := 0
	for i := 0; i < m.cursorRow && i < len(m.lines); i++ {
		offset += len(m.lines[i]) + 1
	}
	if m.cursorRow < 0 || m.cursorRow >= len(m.lines) {
		return offset + 1
	}

	runes := []rune(m.lines[m.cursorRow])
	col := clamp(m.cursorCol, 0, len(runes))
	offset += len(string(runes[:col]))
	return offset + 1
}

func (m *Model) syncASTSelectionFromCursor() bool {
	if m.astIndex == nil {
		return false
	}
	n := m.astIndex.NodeAtOffset(m.cursorOffset())
	if n == nil {
		return false
	}
	m.selectASTNode(n.ID)
	return true
}

func (m *Model) selectASTNode(id jsparse.NodeID) {
	if m.astIndex == nil {
		return
	}
	n := m.astIndex.Nodes[id]
	if n == nil {
		return
	}
	m.selectedASTNodeID = id
	m.highlightStartLine = n.StartLine
	m.highlightStartCol = n.StartCol
	m.highlightEndLine = n.EndLine
	m.highlightEndCol = n.EndCol
	m.cursorRow = clamp(n.StartLine-1, 0, len(m.lines)-1)
	if m.cursorRow >= 0 && m.cursorRow < len(m.lines) {
		lineLen := len([]rune(m.lines[m.cursorRow]))
		m.cursorCol = clamp(n.StartCol-1, 0, lineLen)
	}
	m.ensureCursorVisible()
}

func (m *Model) astSelectRoot() {
	if m.astIndex == nil || m.astIndex.RootID < 0 {
		return
	}
	m.selectASTNode(m.astIndex.RootID)
}

func (m *Model) astSelectParent() {
	if m.astIndex == nil || m.selectedASTNodeID < 0 {
		return
	}
	n := m.astIndex.Nodes[m.selectedASTNodeID]
	if n == nil || n.ParentID < 0 {
		return
	}
	m.selectASTNode(n.ParentID)
}

func (m *Model) astSelectFirstChild() {
	if m.astIndex == nil || m.selectedASTNodeID < 0 {
		return
	}
	n := m.astIndex.Nodes[m.selectedASTNodeID]
	if n == nil || len(n.ChildIDs) == 0 {
		return
	}
	m.selectASTNode(n.ChildIDs[0])
}

func (m *Model) astSelectNextSibling() {
	m.astSelectSibling(1)
}

func (m *Model) astSelectPrevSibling() {
	m.astSelectSibling(-1)
}

func (m *Model) astSelectSibling(step int) {
	if m.astIndex == nil || m.selectedASTNodeID < 0 {
		return
	}
	n := m.astIndex.Nodes[m.selectedASTNodeID]
	if n == nil || n.ParentID < 0 {
		return
	}
	parent := m.astIndex.Nodes[n.ParentID]
	if parent == nil || len(parent.ChildIDs) == 0 {
		return
	}
	for i, childID := range parent.ChildIDs {
		if childID != m.selectedASTNodeID {
			continue
		}
		j := i + step
		if j < 0 || j >= len(parent.ChildIDs) {
			return
		}
		m.selectASTNode(parent.ChildIDs[j])
		return
	}
}

func (m *Model) moveCursor(dy, dx int) {
	m.cursorRow = clamp(m.cursorRow+dy, 0, len(m.lines)-1)
	lineLen := len([]rune(m.lines[m.cursorRow]))
	m.cursorCol = clamp(m.cursorCol+dx, 0, lineLen)
	m.ensureCursorVisible()
	m.updateCursorNodeHighlight()
}

func (m *Model) ensureCursorVisible() {
	vh := maxInt(1, m.contentHeight()-1)
	if m.cursorRow < m.editorScroll {
		m.editorScroll = m.cursorRow
	}
	if m.cursorRow >= m.editorScroll+vh {
		m.editorScroll = m.cursorRow - vh + 1
	}
}

func (m *Model) insertChar(ch rune) {
	line := m.lines[m.cursorRow]
	runes := []rune(line)
	col := clamp(m.cursorCol, 0, len(runes))

	runes = append(runes[:col], append([]rune{ch}, runes[col:]...)...)
	m.lines[m.cursorRow] = string(runes)
	m.cursorCol = col + 1
	m.ensureCursorVisible()
}

func (m *Model) insertNewline() {
	line := m.lines[m.cursorRow]
	runes := []rune(line)
	col := clamp(m.cursorCol, 0, len(runes))

	before := string(runes[:col])
	after := string(runes[col:])
	m.lines[m.cursorRow] = before

	newLines := make([]string, 0, len(m.lines)+1)
	newLines = append(newLines, m.lines[:m.cursorRow+1]...)
	newLines = append(newLines, after)
	newLines = append(newLines, m.lines[m.cursorRow+1:]...)
	m.lines = newLines

	m.cursorRow++
	m.cursorCol = 0
	m.ensureCursorVisible()
}

func (m *Model) deleteBack() {
	if m.cursorCol > 0 {
		line := m.lines[m.cursorRow]
		runes := []rune(line)
		col := clamp(m.cursorCol, 0, len(runes))
		runes = append(runes[:col-1], runes[col:]...)
		m.lines[m.cursorRow] = string(runes)
		m.cursorCol = col - 1
		m.ensureCursorVisible()
		return
	}

	if m.cursorRow > 0 {
		prev := m.lines[m.cursorRow-1]
		cur := m.lines[m.cursorRow]
		m.lines[m.cursorRow-1] = prev + cur
		m.lines = append(m.lines[:m.cursorRow], m.lines[m.cursorRow+1:]...)
		m.cursorRow--
		m.cursorCol = len([]rune(prev))
		m.ensureCursorVisible()
	}
}

// View implements tea.Model.
func (m *Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Initializing..."
	}

	header := m.renderHeader()
	status := m.renderStatus()
	help := m.renderHelp()

	contentHeight := m.contentHeight()
	leftW := m.width / 3
	midW := m.width / 3
	rightW := m.width - leftW - midW

	editor := m.renderEditorPane(leftW, contentHeight)
	ts := m.renderTextPane(" TS SEXP ", m.tsSExpr, m.tsScroll, midW, contentHeight, m.focus == focusTSSExpr)
	ast := m.renderTextPane(" AST SEXP ", m.astTextForView(), m.astScroll, rightW, contentHeight, m.focus == focusASTSExpr)

	content := lipgloss.JoinHorizontal(lipgloss.Top, editor, ts, ast)
	return lipgloss.JoinVertical(lipgloss.Left, header, content, status, help)
}

func (m *Model) contentHeight() int {
	h := m.height - 3 // header + status + help
	if h < 3 {
		h = 3
	}
	return h
}

func (m *Model) renderHeader() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Width(m.width).
		Padding(0, 1)

	focus := "EDITOR"
	switch m.focus {
	case focusEditor:
		// keep default
	case focusTSSExpr:
		focus = "TS SEXP"
	case focusASTSExpr:
		focus = "AST SEXP"
	}
	title := fmt.Sprintf("File: %s", m.filename)
	mode := fmt.Sprintf("AST PARSE EDITOR [%s | %s]", focus, m.editorModeLabel())
	gap := m.width - len(title) - len(mode) - 2
	if gap < 1 {
		gap = 1
	}

	return style.Render(title + strings.Repeat(" ", gap) + mode)
}

func (m *Model) renderStatus() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("236")).
		Width(m.width).
		Padding(0, 1)

	parts := []string{
		fmt.Sprintf("cursor: %d:%d", m.cursorRow+1, m.cursorCol+1),
		fmt.Sprintf("lines: %d", len(m.lines)),
		fmt.Sprintf("seq: %d", m.pendingSeq),
		fmt.Sprintf("mode: %s", m.editorModeLabel()),
	}
	if m.syntaxHighlight {
		parts = append(parts, "syntax: on")
	} else {
		parts = append(parts, "syntax: off")
	}

	if m.tsRoot != nil && m.tsRoot.HasError() {
		parts = append(parts, "ts: error-recovered")
	} else {
		parts = append(parts, "ts: ok")
	}

	if m.astParseErr != nil {
		parts = append(parts, "ast: invalid")
	} else if strings.TrimSpace(m.astSExpr) == "" {
		parts = append(parts, "ast: pending")
	} else {
		parts = append(parts, "ast: valid")
	}
	if m.cursorNode != nil {
		parts = append(parts, fmt.Sprintf("node: %s (%d:%d-%d:%d)",
			m.cursorNode.Kind,
			m.highlightStartLine, m.highlightStartCol,
			m.highlightEndLine, m.highlightEndCol))
	}
	if m.selectedASTNodeID >= 0 && m.astIndex != nil {
		if n := m.astIndex.Nodes[m.selectedASTNodeID]; n != nil {
			parts = append(parts, fmt.Sprintf("ast-node: %s [%d..%d]", n.Kind, n.Start, n.End))
		}
	}

	return style.Render(strings.Join(parts, " | "))
}

func (m *Model) renderHelp() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Width(m.width).
		Padding(0, 1)

	editorKeys := "Editor(INSERT): type, Enter, Backspace, arrows"
	if m.editorMode == editorModeASTSelect {
		editorKeys = "Editor(AST-SELECT): h/j/k/l navigate AST parent/siblings/child"
	}
	return style.Render("Tab:focus pane | m:toggle mode | s:toggle syntax | " + editorKeys + " | TS/AST panes: j/k or arrows to scroll | q:quit")
}

func (m *Model) renderEditorPane(width, height int) string {
	var lines []string
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("240"))
	if m.focus == focusEditor {
		headerStyle = headerStyle.Background(lipgloss.Color("33"))
	}

	title := " EDITOR "
	header := headerStyle.Render(title) + strings.Repeat("─", maxInt(0, width-len(title)))
	lines = append(lines, padRight(header, width))

	contentHeight := maxInt(1, height-1)
	gutterWidth := len(fmt.Sprintf("%d", len(m.lines))) + 1
	cursorStyle := lipgloss.NewStyle().Reverse(true).Bold(true)
	highlightStyle := lipgloss.NewStyle().Background(lipgloss.Color("238"))
	gutterNormal := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	gutterCursor := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)

	for i := 0; i < contentHeight; i++ {
		lineIdx := m.editorScroll + i
		if lineIdx >= len(m.lines) {
			lines = append(lines, strings.Repeat(" ", width))
			continue
		}

		raw := m.lines[lineIdx]
		runes := []rune(raw)
		lineNum := fmt.Sprintf("%*d ", gutterWidth, lineIdx+1)
		gs := gutterNormal
		if lineIdx == m.cursorRow {
			gs = gutterCursor
		}
		gutter := gs.Render(lineNum)

		var content strings.Builder
		for c := 0; c < len(runes); c++ {
			ch := string(runes[c])
			lineNo := lineIdx + 1
			colNo := c + 1
			isNodeHighlight := inRange(lineNo, colNo, m.highlightStartLine, m.highlightStartCol, m.highlightEndLine, m.highlightEndCol)
			syntaxClass := m.syntaxClassAt(lineNo, colNo)
			rendered := renderSyntaxChar(syntaxClass, ch)
			if lineIdx == m.cursorRow && c == m.cursorCol && m.focus == focusEditor {
				content.WriteString(cursorStyle.Render(ch))
			} else if isNodeHighlight {
				content.WriteString(highlightStyle.Render(rendered))
			} else {
				content.WriteString(rendered)
			}
		}
		if lineIdx == m.cursorRow && m.cursorCol >= len(runes) && m.focus == focusEditor {
			content.WriteString(cursorStyle.Render(" "))
		}

		lines = append(lines, padRight(gutter+content.String(), width))
	}

	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
}

func (m *Model) clearCursorNodeHighlight() {
	m.cursorNode = nil
	m.highlightStartLine = 0
	m.highlightStartCol = 0
	m.highlightEndLine = 0
	m.highlightEndCol = 0
}

func (m *Model) updateCursorNodeHighlight() {
	m.clearCursorNodeHighlight()
	if m.tsRoot == nil {
		return
	}

	row := m.cursorRow
	for _, col := range []int{m.cursorCol, m.cursorCol - 1} {
		if col < 0 {
			continue
		}
		n := m.tsRoot.NodeAtPosition(row, col)
		if n == nil {
			continue
		}
		m.cursorNode = n
		m.highlightStartLine = n.StartRow + 1
		m.highlightStartCol = n.StartCol + 1
		m.highlightEndLine = n.EndRow + 1
		m.highlightEndCol = n.EndCol + 1
		return
	}
}

func (m *Model) rebuildSyntaxSpans() {
	m.syntaxSpans = nil
	if m.tsRoot == nil {
		return
	}
	m.appendSyntaxSpans(m.tsRoot)
}

func (m *Model) appendSyntaxSpans(n *jsparse.TSNode) {
	if n == nil {
		return
	}
	if len(n.Children) == 0 {
		class := classifySyntaxKind(n.Kind)
		if class != syntaxClassNone {
			m.syntaxSpans = append(m.syntaxSpans, syntaxSpan{
				startLine: n.StartRow + 1,
				startCol:  n.StartCol + 1,
				endLine:   n.EndRow + 1,
				endCol:    n.EndCol + 1,
				class:     class,
			})
		}
		return
	}
	for _, child := range n.Children {
		m.appendSyntaxSpans(child)
	}
}

func classifySyntaxKind(kind string) syntaxClass {
	switch kind {
	case "comment":
		return syntaxClassComment
	case "string", "string_fragment", "template_string", "template_chars":
		return syntaxClassString
	case "number":
		return syntaxClassNumber
	case "identifier", "property_identifier":
		return syntaxClassIdentifier
	case "const", "let", "var", "function", "return", "if", "else", "for", "while", "do",
		"switch", "case", "default", "break", "continue", "new", "class", "import", "export",
		"from", "try", "catch", "finally", "throw", "await", "async", "extends", "this":
		return syntaxClassKeyword
	case ".", ",", ";", ":", "=", "==", "===", "!=", "!==", "+", "-", "*", "/", "%", "=>",
		"&&", "||", "!", ">", "<", ">=", "<=":
		return syntaxClassOperator
	default:
		return syntaxClassNone
	}
}

func (m *Model) syntaxClassAt(lineNo, colNo int) syntaxClass {
	if !m.syntaxHighlight {
		return syntaxClassNone
	}
	for _, span := range m.syntaxSpans {
		if inRange(lineNo, colNo, span.startLine, span.startCol, span.endLine, span.endCol) {
			return span.class
		}
	}
	return syntaxClassNone
}

func renderSyntaxChar(class syntaxClass, ch string) string {
	switch class {
	case syntaxClassNone:
		return ch
	case syntaxClassKeyword:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true).Render(ch)
	case syntaxClassString:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render(ch)
	case syntaxClassNumber:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(ch)
	case syntaxClassComment:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true).Render(ch)
	case syntaxClassIdentifier:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Render(ch)
	case syntaxClassOperator:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Render(ch)
	default:
		return ch
	}
}

func (m *Model) renderTextPane(title, body string, scroll, width, height int, focused bool) string {
	var lines []string
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("240"))
	if focused {
		headerStyle = headerStyle.Background(lipgloss.Color("33"))
	}

	header := headerStyle.Render(title) + strings.Repeat("─", maxInt(0, width-len(title)))
	lines = append(lines, padRight(header, width))

	contentHeight := maxInt(1, height-1)
	bodyLines := strings.Split(body, "\n")
	start := clamp(scroll, 0, maxInt(0, len(bodyLines)-1))

	for i := 0; i < contentHeight; i++ {
		idx := start + i
		if idx >= len(bodyLines) {
			lines = append(lines, strings.Repeat(" ", width))
			continue
		}
		lines = append(lines, padRight(bodyLines[idx], width))
	}

	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
}

// Close releases parser resources.
func (m *Model) Close() {
	if m.tsParser != nil {
		m.tsParser.Close()
		m.tsParser = nil
	}
}

func padRight(s string, width int) string {
	w := ansi.StringWidth(s)
	if w >= width {
		return ansi.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", width-w)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func inRange(lineNo, colNo, startLine, startCol, endLine, endCol int) bool {
	if startLine <= 0 || endLine <= 0 {
		return false
	}
	if lineNo > startLine && lineNo < endLine {
		return true
	}
	if lineNo == startLine && lineNo == endLine {
		return colNo >= startCol && colNo < endCol
	}
	if lineNo == startLine {
		return colNo >= startCol
	}
	if lineNo == endLine {
		return colNo < endCol
	}
	return false
}
