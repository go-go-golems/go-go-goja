package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	mode_keymap "github.com/go-go-golems/bobatea/pkg/mode-keymap"
	inspectornav "github.com/go-go-golems/go-go-goja/pkg/inspector/navigation"
)

// FocusPane indicates which pane has focus.
type FocusPane int

const (
	FocusSource FocusPane = iota
	FocusTree
	FocusDrawer
)

const (
	modeSource = "source"
	modeTree   = "tree"
	modeDrawer = "drawer"
)

// SyncOrigin indicates the source of the last sync event.
type SyncOrigin int

const (
	SyncNone SyncOrigin = iota
	SyncFromSource
	SyncFromTree
)

// Model is the top-level bubbletea model for the inspector.
type Model struct {
	// Data
	filename   string
	sourceText string
	parseErr   error
	index      *Index

	// UI state
	focus      FocusPane
	syncOrigin SyncOrigin
	width      int
	height     int
	uiMode     string
	keyMap     KeyMap
	help       help.Model
	spinner    spinner.Model
	command    textinput.Model
	commandOn  bool
	commandMsg string

	// Source pane state
	sourceCursorLine int // 0-based
	sourceCursorCol  int // 0-based
	sourceScrollY    int // first visible line (0-based)
	sourceLines      []string
	sourceViewport   viewport.Model
	highlightStart   int // 1-based offset or 0 for none
	highlightEnd     int // 1-based offset exclusive or 0

	// Tree pane state
	treeSelectedIdx  int      // index into visibleNodes
	treeScrollY      int      // first visible row (0-based)
	treeVisibleNodes []NodeID // cached visible node list
	treeList         list.Model
	metaTable        table.Model

	// Selected node
	selectedNodeID NodeID // -1 for none

	// Highlight-usages state
	highlightedBinding *BindingRecord // non-nil when usages are highlighted
	usageHighlights    []NodeID       // all node IDs to highlight for the current binding

	// Bottom drawer
	drawer     *Drawer
	drawerOpen bool
}

// NewModel creates a new inspector model.
func NewModel(filename, sourceText string, program interface{}, parseErr error, index *Index) Model {
	lines := strings.Split(sourceText, "\n")

	m := Model{
		filename:       filename,
		sourceText:     sourceText,
		parseErr:       parseErr,
		index:          index,
		sourceLines:    lines,
		selectedNodeID: -1,
		focus:          FocusSource,
		uiMode:         modeSource,
		keyMap:         newKeyMap(),
	}
	m.help = help.New()
	m.help.ShowAll = false
	m.spinner = spinner.New(spinner.WithSpinner(spinner.Line))
	m.sourceViewport = viewport.New(0, 0)
	m.treeList = newTreeListModel()
	m.command = textinput.New()
	m.command.Prompt = ":"
	m.command.Placeholder = "drawer | clear | help | quit"
	m.command.CharLimit = 120
	m.command.Width = 60
	m.command.Blur()
	m.metaTable = table.New(
		table.WithColumns([]table.Column{
			{Title: "Field", Width: 16},
			{Title: "Value", Width: 40},
		}),
		table.WithRows(nil),
		table.WithFocused(false),
		table.WithHeight(5),
	)
	mode_keymap.EnableMode(&m.keyMap, m.uiMode)

	if index != nil {
		m.selectedNodeID = index.RootID
		m.refreshTreeVisible()
	}

	m.drawer = NewDrawer(index)

	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.command.Width = maxInt(16, msg.Width-4)
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.commandOn {
		return m.handleCommandInput(msg)
	}

	// Drawer-focused keys
	if m.focus == FocusDrawer {
		return m.handleDrawerKey(msg)
	}

	if key.Matches(msg, m.keyMap.Quit) {
		return m, tea.Quit
	}

	if key.Matches(msg, m.keyMap.NextPane) {
		if m.drawerOpen {
			switch m.focus {
			case FocusSource:
				m.focus = FocusTree
			case FocusTree:
				m.focus = FocusDrawer
			case FocusDrawer:
				m.focus = FocusSource
			}
		} else {
			if m.focus == FocusSource {
				m.focus = FocusTree
			} else {
				m.focus = FocusSource
			}
		}
		m.updateInteractionMode()
		return m, nil
	}

	if key.Matches(msg, m.keyMap.OpenDrawer) {
		if m.focus != FocusDrawer {
			m.drawerOpen = true
			m.focus = FocusDrawer
			m.updateInteractionMode()
		}
		return m, nil
	}

	if key.Matches(msg, m.keyMap.Command) {
		m.commandOn = true
		m.command.SetValue("")
		m.command.Focus()
		return m, nil
	}

	if key.Matches(msg, m.keyMap.Yank) {
		// Yank selected node's source text to drawer
		if m.index != nil && m.focus != FocusDrawer {
			m.yankToDrawer()
		}
		return m, nil
	}

	switch msg.String() {
	case "j", "down":
		if m.focus == FocusSource {
			m.sourceMoveCursor(1, 0)
			m.syncSourceToTree()
		} else {
			m.treeMoveSelection(1)
			m.syncTreeToSource()
		}
		return m, nil

	case "k", "up":
		if m.focus == FocusSource {
			m.sourceMoveCursor(-1, 0)
			m.syncSourceToTree()
		} else {
			m.treeMoveSelection(-1)
			m.syncTreeToSource()
		}
		return m, nil

	case "h", "left":
		if m.focus == FocusSource {
			m.sourceMoveCursor(0, -1)
			m.syncSourceToTree()
		} else {
			// Collapse node
			m.treeCollapseSelected()
		}
		return m, nil

	case "l", "right":
		if m.focus == FocusSource {
			m.sourceMoveCursor(0, 1)
			m.syncSourceToTree()
		} else {
			// Expand node
			m.treeExpandSelected()
		}
		return m, nil

	case "enter":
		if m.focus == FocusTree {
			m.syncTreeToSource()
		}
		return m, nil

	case " ":
		if m.focus == FocusTree {
			m.treeToggleSelected()
		}
		return m, nil

	case "g":
		if m.focus == FocusSource {
			m.sourceCursorLine = 0
			m.sourceCursorCol = 0
			m.sourceScrollY = 0
			m.sourceViewport.SetYOffset(0)
			m.syncSourceToTree()
		} else {
			m.treeSelectedIdx = 0
			m.treeScrollY = 0
			m.treeList.Select(0)
			m.syncTreeToSource()
		}
		return m, nil

	case "G":
		if m.focus == FocusSource {
			m.sourceCursorLine = len(m.sourceLines) - 1
			m.sourceCursorCol = 0
			m.ensureSourceCursorVisible()
			m.syncSourceToTree()
		} else {
			if len(m.treeVisibleNodes) > 0 {
				m.treeSelectedIdx = len(m.treeVisibleNodes) - 1
				m.treeList.Select(m.treeSelectedIdx)
				m.ensureTreeSelectionVisible()
				m.syncTreeToSource()
			}
		}
		return m, nil

	case "ctrl+d":
		if m.focus == FocusSource {
			m.sourceMoveCursor(m.sourceViewportHeight()/2, 0)
			m.syncSourceToTree()
		} else {
			m.treeMoveSelection(m.treeViewportHeight() / 2)
			m.syncTreeToSource()
		}
		return m, nil

	case "ctrl+u":
		if m.focus == FocusSource {
			m.sourceMoveCursor(-m.sourceViewportHeight()/2, 0)
			m.syncSourceToTree()
		} else {
			m.treeMoveSelection(-m.treeViewportHeight() / 2)
			m.syncTreeToSource()
		}
		return m, nil

	case "d":
		// Go-to-definition: jump to declaration of identifier under cursor
		m.goToDefinition()
		return m, nil

	case "*":
		// Highlight usages: toggle highlight of all usages of binding under cursor
		m.toggleHighlightUsages()
		return m, nil

	case "esc", "escape":
		// Clear usage highlights
		if m.highlightedBinding != nil {
			m.clearHighlightUsages()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleCommandInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "escape":
		m.commandOn = false
		m.command.Blur()
		m.command.SetValue("")
		return m, nil
	case "enter":
		cmd := strings.TrimSpace(m.command.Value())
		m.commandOn = false
		m.command.Blur()
		m.command.SetValue("")
		m.executeCommand(cmd)
		if cmd == "quit" || cmd == "q" {
			return m, tea.Quit
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.command, cmd = m.command.Update(msg)
	return m, cmd
}

func (m *Model) executeCommand(cmd string) {
	switch cmd {
	case "":
		m.commandMsg = ""
	case "drawer":
		m.drawerOpen = true
		m.focus = FocusDrawer
		m.updateInteractionMode()
		m.commandMsg = "opened drawer"
	case "clear":
		m.clearHighlightUsages()
		m.commandMsg = "cleared usage highlights"
	case "help":
		m.help.ShowAll = !m.help.ShowAll
		if m.help.ShowAll {
			m.commandMsg = "help: full"
		} else {
			m.commandMsg = "help: short"
		}
	case "q", "quit":
		m.commandMsg = "quitting"
	default:
		m.commandMsg = fmt.Sprintf("unknown command: %s", cmd)
	}
}

// --- Drawer key handling ---

func (m Model) handleDrawerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	if key.Matches(msg, m.keyMap.Quit) {
		return m, tea.Quit
	}

	switch keyStr {
	case "q":
		// In drawer, q types a character (not quit)
	case "tab":
		// Tab cycles focus
		m.focus = FocusSource
		m.updateInteractionMode()
		return m, nil
	case "esc", "escape":
		if m.drawer.completionActive {
			m.drawer.completionActive = false
		} else {
			m.drawerOpen = false
			m.focus = FocusSource
			m.updateInteractionMode()
		}
		return m, nil
	case "enter":
		if m.drawer.completionActive && len(m.drawer.completionItems) > 0 {
			m.acceptCompletion()
		} else {
			m.drawer.InsertNewline()
			m.drawer.Reparse()
		}
		return m, nil
	case "backspace":
		m.drawer.DeleteBack()
		m.drawer.Reparse()
		m.updateDrawerCompletion()
		return m, nil
	case "up":
		if m.drawer.completionActive {
			m.drawer.completionIdx--
			if m.drawer.completionIdx < 0 {
				m.drawer.completionIdx = len(m.drawer.completionItems) - 1
			}
		} else {
			m.drawer.MoveCursor(-1, 0)
		}
		return m, nil
	case "down":
		if m.drawer.completionActive {
			m.drawer.completionIdx++
			if m.drawer.completionIdx >= len(m.drawer.completionItems) {
				m.drawer.completionIdx = 0
			}
		} else {
			m.drawer.MoveCursor(1, 0)
		}
		return m, nil
	case "left":
		m.drawer.MoveCursor(0, -1)
		m.drawer.completionActive = false
		return m, nil
	case "right":
		m.drawer.MoveCursor(0, 1)
		m.drawer.completionActive = false
		return m, nil
	case "home":
		m.drawer.MoveHome()
		return m, nil
	case "end":
		m.drawer.MoveEnd()
		return m, nil
	case "ctrl+space", "ctrl+@", "ctrl+n":
		m.triggerDrawerCompletion()
		return m, nil
	case "ctrl+up":
		m.drawer.Grow(2)
		return m, nil
	case "ctrl+down":
		m.drawer.Grow(-2)
		return m, nil
	case "ctrl+d":
		m.drawerGoToDefinition()
		return m, nil
	case "ctrl+g":
		m.drawerHighlightUsages()
		return m, nil
	}

	// Default: insert character
	runes := msg.Runes
	if len(runes) > 0 {
		for _, r := range runes {
			m.drawer.InsertChar(r)
		}
		m.drawer.Reparse()
		// Auto-trigger completion on "."
		if len(runes) == 1 && runes[0] == '.' {
			m.triggerDrawerCompletion()
		} else {
			m.updateDrawerCompletion()
		}
	}

	return m, nil
}

func (m *Model) updateInteractionMode() {
	switch m.focus {
	case FocusSource:
		m.uiMode = modeSource
	case FocusTree:
		m.uiMode = modeTree
	case FocusDrawer:
		m.uiMode = modeDrawer
	default:
		m.uiMode = modeSource
	}
	mode_keymap.EnableMode(&m.keyMap, m.uiMode)
}

func (m *Model) triggerDrawerCompletion() {
	if m.drawer == nil || m.drawer.tsRoot == nil {
		return
	}
	ctx := ExtractCompletionContext(m.drawer.tsRoot, m.drawer.Source(), m.drawer.cursorRow, m.drawer.cursorCol)
	if ctx.Kind == CompletionNone {
		m.drawer.completionActive = false
		return
	}
	candidates := ResolveCandidates(ctx, m.index, m.drawer.tsRoot)
	m.drawer.completionCtx = ctx
	m.drawer.completionItems = candidates
	m.drawer.completionIdx = 0
	m.drawer.completionActive = len(candidates) > 0
}

func (m *Model) updateDrawerCompletion() {
	if !m.drawer.completionActive {
		return
	}
	// Re-extract context and re-filter
	m.triggerDrawerCompletion()
}

func (m *Model) acceptCompletion() {
	if !m.drawer.completionActive || len(m.drawer.completionItems) == 0 {
		return
	}
	item := m.drawer.completionItems[m.drawer.completionIdx]

	// Delete the partial text already typed
	partial := m.drawer.completionCtx.PartialText
	for i := 0; i < len([]rune(partial)); i++ {
		m.drawer.DeleteBack()
	}

	// Insert the full label
	for _, r := range item.Label {
		m.drawer.InsertChar(r)
	}

	m.drawer.completionActive = false
	m.drawer.Reparse()
}

func (m *Model) yankToDrawer() {
	if m.index == nil || m.drawer == nil {
		return
	}
	node := m.index.Nodes[m.selectedNodeID]
	if node == nil {
		return
	}
	// Extract source text for the selected node
	startIdx := int(node.Start) - 1 // 1-based to 0-based
	endIdx := int(node.End) - 1
	if startIdx < 0 {
		startIdx = 0
	}
	src := m.index.Source()
	if endIdx > len(src) {
		endIdx = len(src)
	}
	text := src[startIdx:endIdx]

	// Insert into drawer at cursor
	for _, ch := range string(text) {
		if ch == '\n' {
			m.drawer.InsertNewline()
		} else {
			m.drawer.InsertChar(ch)
		}
	}
	m.drawer.Reparse()
	m.drawerOpen = true
}

func (m *Model) drawerGoToDefinition() {
	if m.drawer == nil || m.drawer.tsRoot == nil || m.index == nil || m.index.Resolution == nil {
		return
	}

	// Find identifier at drawer cursor (try cursor pos and cursor-1)
	var node *TSNode
	for _, col := range []int{m.drawer.cursorCol, m.drawer.cursorCol - 1} {
		if col < 0 {
			continue
		}
		n := m.drawer.tsRoot.NodeAtPosition(m.drawer.cursorRow, col)
		if n != nil && (n.Kind == "identifier" || n.Kind == "property_identifier") {
			node = n
			break
		}
	}
	if node == nil {
		return
	}

	name := node.Text
	// Look up in ALL scopes (not just global)
	var binding *BindingRecord
	for _, scope := range m.index.Resolution.Scopes {
		if b, ok := scope.Bindings[name]; ok {
			binding = b
			break
		}
	}
	if binding == nil {
		return
	}

	// Jump to declaration in source pane
	declNode := m.index.Nodes[binding.DeclNodeID]
	if declNode == nil {
		return
	}
	m.sourceCursorLine = declNode.StartLine - 1
	m.sourceCursorCol = declNode.StartCol - 1
	m.ensureSourceCursorVisible()
	m.highlightStart = declNode.Start
	m.highlightEnd = declNode.End
	m.selectedNodeID = binding.DeclNodeID
	m.index.ExpandTo(binding.DeclNodeID)
	m.refreshTreeVisible()
	for i, vid := range m.treeVisibleNodes {
		if vid == binding.DeclNodeID {
			m.treeSelectedIdx = i
			break
		}
	}
	m.ensureTreeSelectionVisible()
}

func (m *Model) drawerHighlightUsages() {
	if m.drawer == nil || m.drawer.tsRoot == nil || m.index == nil || m.index.Resolution == nil {
		return
	}

	var node *TSNode
	for _, col := range []int{m.drawer.cursorCol, m.drawer.cursorCol - 1} {
		if col < 0 {
			continue
		}
		n := m.drawer.tsRoot.NodeAtPosition(m.drawer.cursorRow, col)
		if n != nil && n.Kind == "identifier" {
			node = n
			break
		}
	}
	if node == nil {
		return
	}

	name := node.Text
	var binding *BindingRecord
	for _, scope := range m.index.Resolution.Scopes {
		if b, ok := scope.Bindings[name]; ok {
			binding = b
			break
		}
	}
	if binding == nil {
		m.clearHighlightUsages()
		return
	}

	if m.highlightedBinding == binding {
		m.clearHighlightUsages()
	} else {
		m.highlightedBinding = binding
		m.usageHighlights = binding.AllUsages()
	}
}

// --- Source pane methods ---

func (m *Model) sourceMoveCursor(dy, dx int) {
	m.sourceCursorLine += dy
	m.sourceCursorCol += dx

	// Clamp line
	if m.sourceCursorLine < 0 {
		m.sourceCursorLine = 0
	}
	if m.sourceCursorLine >= len(m.sourceLines) {
		m.sourceCursorLine = len(m.sourceLines) - 1
	}

	// Clamp column
	lineLen := len(m.sourceLines[m.sourceCursorLine])
	if m.sourceCursorCol < 0 {
		m.sourceCursorCol = 0
	}
	if m.sourceCursorCol > lineLen {
		m.sourceCursorCol = lineLen
	}

	m.ensureSourceCursorVisible()
}

func (m *Model) ensureSourceCursorVisible() {
	vpHeight := m.sourceViewportHeight()
	if vpHeight <= 0 {
		return
	}
	if m.sourceCursorLine < m.sourceScrollY {
		m.sourceScrollY = m.sourceCursorLine
	}
	if m.sourceCursorLine >= m.sourceScrollY+vpHeight {
		m.sourceScrollY = m.sourceCursorLine - vpHeight + 1
	}
	m.sourceViewport.SetYOffset(m.sourceScrollY)
}

func (m *Model) sourceViewportHeight() int {
	if m.sourceViewport.Height > 0 {
		return m.sourceViewport.Height
	}
	h := m.height - 4 // header + status + help + border
	if h < 1 {
		h = 1
	}
	return h
}

// --- Tree pane methods ---

func (m *Model) refreshTreeVisible() {
	if m.index != nil {
		m.treeVisibleNodes = m.index.VisibleNodes()
		m.rebuildTreeListItems()
	}
}

func (m *Model) rebuildTreeListItems() {
	items := make([]list.Item, 0, len(m.treeVisibleNodes))
	for _, nodeID := range m.treeVisibleNodes {
		node := m.index.Nodes[nodeID]
		if node == nil {
			continue
		}
		items = append(items, buildTreeListItem(node, m.usageHighlights, m.index.Resolution))
	}
	_ = m.treeList.SetItems(items)
	if len(items) == 0 {
		m.treeSelectedIdx = 0
		return
	}
	if m.treeSelectedIdx < 0 {
		m.treeSelectedIdx = 0
	}
	if m.treeSelectedIdx >= len(items) {
		m.treeSelectedIdx = len(items) - 1
	}
	m.treeList.Select(m.treeSelectedIdx)
}

func (m *Model) treeMoveSelection(delta int) {
	if len(m.treeVisibleNodes) == 0 {
		return
	}
	m.treeSelectedIdx = m.treeList.Index() + delta
	if m.treeSelectedIdx < 0 {
		m.treeSelectedIdx = 0
	}
	if m.treeSelectedIdx >= len(m.treeVisibleNodes) {
		m.treeSelectedIdx = len(m.treeVisibleNodes) - 1
	}
	m.treeList.Select(m.treeSelectedIdx)
	m.selectedNodeID = m.treeVisibleNodes[m.treeSelectedIdx]
	m.ensureTreeSelectionVisible()
}

func (m *Model) treeToggleSelected() {
	if m.index == nil || len(m.treeVisibleNodes) == 0 {
		return
	}
	id := m.treeVisibleNodes[m.treeSelectedIdx]
	m.index.ToggleExpand(id)
	m.refreshTreeVisible()
	// Keep selected index valid
	if m.treeSelectedIdx >= len(m.treeVisibleNodes) {
		m.treeSelectedIdx = len(m.treeVisibleNodes) - 1
	}
	m.treeList.Select(m.treeSelectedIdx)
}

func (m *Model) treeExpandSelected() {
	if m.index == nil || len(m.treeVisibleNodes) == 0 {
		return
	}
	id := m.treeVisibleNodes[m.treeSelectedIdx]
	n := m.index.Nodes[id]
	if n != nil && n.HasChildren() && !n.Expanded {
		n.Expanded = true
		m.refreshTreeVisible()
	}
}

func (m *Model) treeCollapseSelected() {
	if m.index == nil || len(m.treeVisibleNodes) == 0 {
		return
	}
	id := m.treeVisibleNodes[m.treeSelectedIdx]
	n := m.index.Nodes[id]
	if n != nil && n.HasChildren() && n.Expanded {
		n.Expanded = false
		m.refreshTreeVisible()
	} else if n != nil && n.ParentID >= 0 {
		// Move to parent if leaf or already collapsed
		m.selectedNodeID = n.ParentID
		for i, vid := range m.treeVisibleNodes {
			if vid == m.selectedNodeID {
				m.treeSelectedIdx = i
				break
			}
		}
		m.ensureTreeSelectionVisible()
	}
}

func (m *Model) ensureTreeSelectionVisible() {
	if len(m.treeVisibleNodes) == 0 {
		return
	}
	m.treeSelectedIdx = m.treeList.Index()
	m.treeScrollY = m.treeSelectedIdx
}

func (m *Model) treeViewportHeight() int {
	h := m.height - 4 // header + status + help + border
	if h < 1 {
		h = 1
	}
	return h
}

// --- Sync methods ---

func (m *Model) syncSourceToTree() {
	if m.index == nil {
		return
	}
	selection, ok := inspectornav.SelectionAtSourceCursor(m.index, m.sourceLines, m.sourceCursorLine, m.sourceCursorCol)
	if !ok {
		return
	}

	m.selectedNodeID = selection.NodeID
	m.syncOrigin = SyncFromSource

	// Ensure ancestors are expanded so node is visible.
	m.index.ExpandTo(selection.NodeID)
	m.refreshTreeVisible()

	visibleIdx := inspectornav.FindVisibleNodeIndex(m.treeVisibleNodes, selection.NodeID)
	if visibleIdx >= 0 {
		m.treeSelectedIdx = visibleIdx
		m.treeList.Select(m.treeSelectedIdx)
		m.ensureTreeSelectionVisible()
	}

	m.highlightStart = selection.HighlightStart
	m.highlightEnd = selection.HighlightEnd
}

func (m *Model) syncTreeToSource() {
	selection, ok := inspectornav.SelectionFromVisibleTree(m.index, m.treeVisibleNodes, m.treeSelectedIdx)
	if !ok {
		return
	}

	m.selectedNodeID = selection.NodeID
	m.syncOrigin = SyncFromTree

	// Move source cursor to start of selected node
	m.highlightStart = selection.HighlightStart
	m.highlightEnd = selection.HighlightEnd
	m.sourceCursorLine = selection.CursorLine
	m.sourceCursorCol = selection.CursorCol
	m.ensureSourceCursorVisible()
}

// --- Go-to-definition and highlight-usages ---

func (m *Model) goToDefinition() {
	if m.index == nil || m.index.Resolution == nil {
		return
	}

	// Find the identifier under cursor
	nodeID := m.selectedNodeID
	if nodeID < 0 {
		return
	}

	b := m.index.Resolution.BindingForNode(nodeID)
	if b == nil {
		return
	}

	// Jump to declaration
	declNode := m.index.Nodes[b.DeclNodeID]
	if declNode == nil {
		return
	}

	// Move cursor to declaration
	m.sourceCursorLine = declNode.StartLine - 1
	m.sourceCursorCol = declNode.StartCol - 1
	m.ensureSourceCursorVisible()
	m.highlightStart = declNode.Start
	m.highlightEnd = declNode.End

	// Select declaration node in tree
	m.selectedNodeID = b.DeclNodeID
	m.index.ExpandTo(b.DeclNodeID)
	m.refreshTreeVisible()
	for i, vid := range m.treeVisibleNodes {
		if vid == b.DeclNodeID {
			m.treeSelectedIdx = i
			break
		}
	}
	m.ensureTreeSelectionVisible()
}

func (m *Model) toggleHighlightUsages() {
	if m.index == nil || m.index.Resolution == nil {
		return
	}

	nodeID := m.selectedNodeID
	if nodeID < 0 {
		return
	}

	b := m.index.Resolution.BindingForNode(nodeID)
	if b == nil {
		// Try the current cursor position — might be a different node
		m.clearHighlightUsages()
		return
	}

	// If already highlighting this binding, toggle off
	if m.highlightedBinding == b {
		m.clearHighlightUsages()
		return
	}

	m.highlightedBinding = b
	m.usageHighlights = b.AllUsages()
}

func (m *Model) clearHighlightUsages() {
	m.highlightedBinding = nil
	m.usageHighlights = nil
}

// --- View ---

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	// Layout: header, two panes side by side, [drawer], status, help
	headerHeight := 1
	statusHeight := 1
	helpHeight := 1
	commandHeight := 0
	if m.commandOn {
		commandHeight = 1
	}
	drawerHeight := 0
	separatorHeight := 0
	if m.drawerOpen && m.drawer != nil {
		drawerHeight = m.drawer.height
		separatorHeight = 1
	}
	contentHeight := m.height - headerHeight - statusHeight - helpHeight - commandHeight - drawerHeight - separatorHeight
	if contentHeight < 3 {
		contentHeight = 3
	}

	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth

	header := m.renderHeader()
	leftPane := m.renderSourcePane(leftWidth, contentHeight)
	rightPane := m.renderTreePane(rightWidth, contentHeight)
	status := m.renderStatusBar()
	help := m.renderHelp()

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	if m.drawerOpen && m.drawer != nil {
		sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		separator := sepStyle.Render(strings.Repeat("─", m.width))
		drawerView := m.drawer.Render(m.width, drawerHeight, m.focus == FocusDrawer)
		if m.commandOn {
			return lipgloss.JoinVertical(lipgloss.Left, header, content, separator, drawerView, status, help, m.renderCommandLine())
		}
		return lipgloss.JoinVertical(lipgloss.Left, header, content, separator, drawerView, status, help)
	}

	if m.commandOn {
		return lipgloss.JoinVertical(lipgloss.Left, header, content, status, help, m.renderCommandLine())
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, content, status, help)
}

func (m Model) renderHeader() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Width(m.width).
		Padding(0, 1)

	focusStr := "SOURCE"
	switch m.focus {
	case FocusSource:
		// keep default focusStr
	case FocusTree:
		focusStr = "TREE"
	case FocusDrawer:
		focusStr = "DRAWER"
	}

	title := fmt.Sprintf("File: %s", m.filename)
	mode := fmt.Sprintf("STATIC INSPECTOR [%s]", focusStr)
	gap := m.width - len(title) - len(mode) - 2
	if gap < 1 {
		gap = 1
	}

	return style.Render(title + strings.Repeat(" ", gap) + mode)
}

func (m Model) renderSourcePane(width, height int) string {
	var lines []string
	gutterWidth := len(fmt.Sprintf("%d", len(m.sourceLines))) + 1

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("240"))

	paneLabel := " SOURCE "
	if m.focus == FocusSource {
		headerStyle = headerStyle.Background(lipgloss.Color("33"))
	}
	headerLine := headerStyle.Render(paneLabel) + strings.Repeat("─", maxInt(0, width-len(paneLabel)))
	lines = append(lines, padRight(headerLine, width))

	contentHeight := height - 1
	if contentHeight < 1 {
		contentHeight = 1
	}

	var hlStartLine, hlStartCol, hlEndLine, hlEndCol int
	if m.highlightStart > 0 && m.highlightEnd > 0 && m.index != nil {
		hlStartLine, hlStartCol = m.index.OffsetToLineCol(m.highlightStart)
		hlEndLine, hlEndCol = m.index.OffsetToLineCol(m.highlightEnd)
	}

	hlStyle := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("117"))
	usageStyle := lipgloss.NewStyle().Background(lipgloss.Color("58")).Foreground(lipgloss.Color("229")).Bold(true)
	cursorStyle := lipgloss.NewStyle().Reverse(true).Bold(true)
	gutterNormal := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	gutterCursor := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)

	type hlRange struct{ startLine, startCol, endLine, endCol int }
	var usageRanges []hlRange
	if m.highlightedBinding != nil && m.index != nil {
		for _, nid := range m.usageHighlights {
			node := m.index.Nodes[nid]
			if node == nil {
				continue
			}
			sl, sc := m.index.OffsetToLineCol(node.Start)
			el, ec := m.index.OffsetToLineCol(node.End)
			usageRanges = append(usageRanges, hlRange{sl, sc, el, ec})
		}
	}

	renderedLines := make([]string, 0, len(m.sourceLines))
	for lineIdx := 0; lineIdx < len(m.sourceLines); lineIdx++ {
		lineNum := fmt.Sprintf("%*d ", gutterWidth, lineIdx+1)
		content := m.sourceLines[lineIdx]
		lineNo := lineIdx + 1

		isCursorLine := (lineIdx == m.sourceCursorLine)
		showCursor := isCursorLine && m.focus == FocusSource

		gs := gutterNormal
		if isCursorLine {
			gs = gutterCursor
		}
		gutter := gs.Render(lineNum)

		var rendered strings.Builder
		runes := []rune(content)
		for col := 0; col < len(runes); col++ {
			ch := string(runes[col])
			col1 := col + 1

			isHL := false
			if hlStartLine > 0 {
				if lineNo > hlStartLine && lineNo < hlEndLine {
					isHL = true
				} else if lineNo == hlStartLine && lineNo == hlEndLine {
					isHL = col1 >= hlStartCol && col1 < hlEndCol
				} else if lineNo == hlStartLine {
					isHL = col1 >= hlStartCol
				} else if lineNo == hlEndLine {
					isHL = col1 < hlEndCol
				}
			}

			isUsageHL := false
			for _, ur := range usageRanges {
				if lineNo > ur.startLine && lineNo < ur.endLine {
					isUsageHL = true
					break
				} else if lineNo == ur.startLine && lineNo == ur.endLine {
					if col1 >= ur.startCol && col1 < ur.endCol {
						isUsageHL = true
						break
					}
				} else if lineNo == ur.startLine && col1 >= ur.startCol {
					isUsageHL = true
					break
				} else if lineNo == ur.endLine && col1 < ur.endCol {
					isUsageHL = true
					break
				}
			}

			isCursorChar := showCursor && col == m.sourceCursorCol
			if isCursorChar {
				rendered.WriteString(cursorStyle.Render(ch))
			} else if isUsageHL {
				rendered.WriteString(usageStyle.Render(ch))
			} else if isHL {
				rendered.WriteString(hlStyle.Render(ch))
			} else {
				rendered.WriteString(ch)
			}
		}
		if showCursor && m.sourceCursorCol >= len(runes) {
			rendered.WriteString(cursorStyle.Render(" "))
		}
		renderedLines = append(renderedLines, padRight(gutter+rendered.String(), width))
	}

	m.sourceViewport.Width = width
	m.sourceViewport.Height = contentHeight
	m.sourceViewport.SetContent(strings.Join(renderedLines, "\n"))
	m.sourceViewport.SetYOffset(m.sourceScrollY)

	lines = append(lines, m.sourceViewport.View())
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
}

func (m Model) renderTreePane(width, height int) string {
	var lines []string

	// Pane header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("240"))

	paneLabel := " AST TREE "
	if m.focus == FocusTree {
		headerStyle = headerStyle.Background(lipgloss.Color("33"))
	}
	headerLine := headerStyle.Render(paneLabel) + strings.Repeat("─", maxInt(0, width-len(paneLabel)))
	lines = append(lines, padRight(headerLine, width))

	contentHeight := height - 1
	if contentHeight < 1 {
		contentHeight = 1
	}

	if m.parseErr != nil {
		// Show parse error banner
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		errMsg := fmt.Sprintf(" ⚠ %v", m.parseErr)
		lines = append(lines, padRight(errStyle.Render(errMsg), width))
		contentHeight-- // eat one line for error banner
	}

	if m.index == nil || len(m.treeVisibleNodes) == 0 {
		lines = append(lines, padRight(" (no AST)", width))
		for len(lines) < height {
			lines = append(lines, strings.Repeat(" ", width))
		}
		return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
	}

	metaHeight := 5
	if contentHeight <= metaHeight+2 {
		metaHeight = 3
	}
	treeHeight := contentHeight - metaHeight - 1
	if treeHeight < 1 {
		treeHeight = 1
		metaHeight = maxInt(2, contentHeight-treeHeight)
	}

	m.treeList.SetSize(width, treeHeight)
	lines = append(lines, m.treeList.View())
	lines = append(lines, padRight(strings.Repeat("─", width), width))

	meta := m.metaTable
	meta.SetWidth(width)
	meta.SetHeight(metaHeight)
	meta.SetRows(m.selectedMetaRows())
	lines = append(lines, meta.View())
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(strings.Join(lines, "\n"))
}

func (m Model) selectedMetaRows() []table.Row {
	rows := []table.Row{{"Focus", m.uiMode}}
	if m.selectedNodeID < 0 || m.index == nil {
		return rows
	}
	n := m.index.Nodes[m.selectedNodeID]
	if n == nil {
		return rows
	}

	rows = append(rows,
		table.Row{"Node", n.Kind},
		table.Row{"Span", fmt.Sprintf("[%d..%d]", n.Start, n.End)},
		table.Row{"Lines", fmt.Sprintf("%d:%d → %d:%d", n.StartLine, n.StartCol, n.EndLine, n.EndCol)},
	)
	if m.index.Resolution != nil {
		if b := m.index.Resolution.BindingForNode(m.selectedNodeID); b != nil {
			rows = append(rows, table.Row{"Binding", fmt.Sprintf("%s (%s)", b.Name, b.Kind)})
			rows = append(rows, table.Row{"Usages", fmt.Sprintf("%d", len(b.AllUsages()))})
		}
	}
	if m.highlightedBinding != nil {
		rows = append(rows, table.Row{"Highlight", m.highlightedBinding.Name})
	}
	return rows
}

func (m Model) renderStatusBar() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("236")).
		Width(m.width).
		Padding(0, 1)

	var parts []string

	if m.parseErr != nil {
		parts = append(parts, "⚠ parse error")
	} else {
		parts = append(parts, "✓ parse ok")
		if m.index != nil {
			parts = append(parts, fmt.Sprintf("nodes: %d", len(m.index.Nodes)))
		}
	}

	if m.selectedNodeID >= 0 && m.index != nil {
		n := m.index.Nodes[m.selectedNodeID]
		if n != nil {
			parts = append(parts, fmt.Sprintf("selected: %s [%d..%d] (%d:%d → %d:%d)",
				n.Kind, n.Start, n.End, n.StartLine, n.StartCol, n.EndLine, n.EndCol))
		}
	}

	// Add binding info if on an identifier
	if m.selectedNodeID >= 0 && m.index != nil && m.index.Resolution != nil {
		b := m.index.Resolution.BindingForNode(m.selectedNodeID)
		if b != nil {
			usages := len(b.References) + 1 // +1 for decl
			parts = append(parts, fmt.Sprintf("binding: %s (%s, %d usages)", b.Name, b.Kind, usages))
		} else if m.index.Resolution.IsUnresolved(m.selectedNodeID) {
			node := m.index.Nodes[m.selectedNodeID]
			if node != nil {
				parts = append(parts, fmt.Sprintf("'%s' (global/unresolved)", node.Label))
			}
		}
	}

	if m.highlightedBinding != nil {
		parts = append(parts, fmt.Sprintf("★ %d usages of '%s'", len(m.usageHighlights), m.highlightedBinding.Name))
	}
	if m.drawer != nil && m.drawer.completionActive {
		parts = append(parts, m.spinner.View()+" completing")
	}

	if m.focus == FocusDrawer && m.drawer != nil {
		parts = append(parts, m.drawer.CursorInfo())
	} else {
		parts = append(parts, fmt.Sprintf("cursor: %d:%d", m.sourceCursorLine+1, m.sourceCursorCol+1))
	}

	return style.Render(strings.Join(parts, " │ "))
}

func (m Model) renderHelp() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Width(m.width).
		Padding(0, 1)
	return style.Render(m.help.View(m.keyMap))
}

func (m Model) renderCommandLine() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("235")).
		Width(m.width).
		Padding(0, 1)
	input := m.command.View()
	if m.commandMsg != "" {
		input = input + " │ " + m.commandMsg
	}
	return style.Render(input)
}

// truncateAnsi truncates a string that may contain ANSI escape sequences
// to the given visible width, preserving escape codes.
func truncateAnsi(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	return ansi.Truncate(s, maxWidth, "")
}

// padRight pads a (possibly ANSI-styled) string to exactly the given visible width.
func padRight(s string, width int) string {
	w := ansi.StringWidth(s)
	if w >= width {
		return truncateAnsi(s, width)
	}
	return s + strings.Repeat(" ", width-w)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
