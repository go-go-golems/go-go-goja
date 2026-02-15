package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dop251/goja"
	mode_keymap "github.com/go-go-golems/bobatea/pkg/mode-keymap"
	"github.com/go-go-golems/go-go-goja/internal/inspectorui"
	"github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
	"github.com/go-go-golems/go-go-goja/pkg/inspectorapi"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// GlobalItem is the app-facing alias for reusable inspector global binding data.
type GlobalItem = inspectorapi.Global

// MemberItem represents a member of a selected class/object.
type MemberItem = inspectorapi.Member

// Model is the top-level bubbletea model for the Smalltalk inspector.
type Model struct {
	// Data
	filename         string
	source           string
	analysis         *jsparse.AnalysisResult
	documentID       inspectorapi.DocumentID
	inspectorService *inspectorapi.Service

	// Global scope items
	globals      []GlobalItem
	globalIdx    int
	globalScroll int

	// Members for selected global
	members      []MemberItem
	memberIdx    int
	memberScroll int

	// Source pane
	sourceLines  []string
	sourceTarget int // target line to highlight (0-based), -1 = none

	// Syntax highlighting
	tsParser        *jsparse.TSParser
	fileSyntaxSpans []jsparse.SyntaxSpan // spans for file source
	replSyntaxSpans []jsparse.SyntaxSpan // spans for REPL source

	// REPL source tracking
	replSourceLines []string // accumulated REPL expressions as source lines
	replSourceLog   []replSourceEntry
	showingReplSrc  bool // true when source pane shows REPL source instead of file

	// Runtime
	rtSession    *runtime.Session
	replInput    textinput.Model
	replHistory  []string
	replResult   string
	replError    string
	replDeclared []inspectorapi.DeclaredBinding

	// Object browser (for REPL results and inspect targets)
	inspectObj   *goja.Object
	inspectProps []runtime.PropertyInfo
	inspectIdx   int
	inspectLabel string

	// Navigation stack for drill-in inspection
	navStack []NavFrame

	// Error/stack trace state
	errorInfo    *runtime.ErrorInfo
	stackIdx     int
	showingError bool

	// UI state
	focus     FocusPane
	mode      string
	width     int
	height    int
	loaded    bool
	statusMsg string

	// Components
	keyMap          KeyMap
	help            help.Model
	spinner         spinner.Model
	command         textinput.Model
	sourceViewport  viewport.Model
	inspectViewport viewport.Model
	stackViewport   viewport.Model
	cmdActive       bool
}

// NewModel creates a new Smalltalk inspector model.
func NewModel(filename string) Model {
	m := Model{
		filename:         filename,
		focus:            FocusGlobals,
		mode:             modeEmpty,
		globalIdx:        0,
		memberIdx:        0,
		sourceTarget:     -1,
		keyMap:           newKeyMap(),
		inspectorService: inspectorapi.NewService(),
	}
	m.help = help.New()
	m.help.ShowAll = false
	m.spinner = spinner.New(spinner.WithSpinner(spinner.Line))
	m.command = textinput.New()
	m.command.Prompt = ": "
	m.command.Placeholder = "load <file.js> | help | quit"
	m.command.CharLimit = 256
	m.command.Width = 60
	m.command.Blur()
	m.sourceViewport = viewport.New(0, 0)
	m.inspectViewport = viewport.New(0, 0)
	m.stackViewport = viewport.New(0, 0)

	m.replInput = textinput.New()
	m.replInput.Prompt = "» "
	m.replInput.Placeholder = "evaluate expression..."
	m.replInput.CharLimit = 512
	m.replInput.Width = 60
	m.replInput.Blur()

	// Initialize tree-sitter parser for syntax highlighting
	if parser, err := jsparse.NewTSParser(); err == nil {
		m.tsParser = parser
	}
	mode_keymap.EnableMode(&m.keyMap, m.mode)

	return m
}

// Close releases resources (tree-sitter parser).
func (m *Model) Close() {
	if m.tsParser != nil {
		m.tsParser.Close()
		m.tsParser = nil
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, m.spinner.Tick)

	// If filename was provided on CLI, load it immediately
	if m.filename != "" {
		cmds = append(cmds, func() tea.Msg {
			return loadFile(m.filename)
		})
	}

	return tea.Batch(cmds...)
}

// loadFile reads and analyzes a JavaScript file, returning the appropriate message.
func loadFile(filename string) tea.Msg {
	filename = filepath.Clean(filename)
	// #nosec G304 -- inspector intentionally reads user-selected source files.
	src, err := os.ReadFile(filename)
	if err != nil {
		return MsgFileLoadError{Filename: filename, Err: err}
	}

	sourceText := string(src)
	analysis := jsparse.Analyze(filename, sourceText, nil)

	return MsgFileLoaded{
		Filename: filename,
		Source:   sourceText,
		Analysis: analysis,
	}
}

// buildGlobals extracts global bindings from the analysis result.
func (m *Model) buildGlobals() {
	m.globals = nil
	if m.inspectorService == nil || m.documentID == "" {
		return
	}

	resp, err := m.inspectorService.ListGlobals(inspectorapi.ListGlobalsRequest{
		DocumentID: m.documentID,
	})
	if err != nil {
		return
	}
	m.globals = append(m.globals, resp.Globals...)

	m.globalIdx = 0
	m.globalScroll = 0
}

// buildMembers builds the member list for the currently selected global.
func (m *Model) buildMembers() {
	m.members = nil
	m.memberIdx = 0
	m.memberScroll = 0

	if len(m.globals) == 0 || m.globalIdx >= len(m.globals) {
		return
	}

	selected := m.globals[m.globalIdx]

	if m.inspectorService == nil || m.documentID == "" {
		return
	}

	resp, err := m.inspectorService.ListMembers(inspectorapi.ListMembersRequest{
		DocumentID: m.documentID,
		GlobalName: selected.Name,
	}, m.rtSession)
	if err != nil {
		return
	}
	m.members = append(m.members, resp.Members...)
}

// jumpToSource sets the source pane to show the declaration of the selected member.
func (m *Model) jumpToSource() {
	m.sourceTarget = -1
	m.showingReplSrc = false

	if m.inspectorService == nil || m.documentID == "" {
		return
	}

	if m.focus == FocusGlobals && len(m.globals) > 0 {
		m.jumpToBinding(m.globals[m.globalIdx].Name)
	} else if m.focus == FocusMembers && len(m.members) > 0 && len(m.globals) > 0 {
		member := m.members[m.memberIdx]
		if member.RuntimeDerived {
			return // no source location for runtime-derived members
		}
		global := m.globals[m.globalIdx]
		m.jumpToMember(global.Name, member)
	}
}

func (m *Model) jumpToBinding(name string) {
	resp, err := m.inspectorService.BindingDeclarationLine(inspectorapi.BindingDeclarationRequest{
		DocumentID: m.documentID,
		Name:       name,
	})
	if err != nil || !resp.Found {
		return
	}
	m.sourceTarget = resp.Line - 1
	m.ensureSourceVisible(m.sourceTarget)
}

func (m *Model) jumpToMember(className string, member MemberItem) {
	sourceClass := ""
	if member.Inherited && member.Source != "" {
		sourceClass = member.Source
	}
	resp, err := m.inspectorService.MemberDeclarationLine(inspectorapi.MemberDeclarationRequest{
		DocumentID:  m.documentID,
		ClassName:   className,
		SourceClass: sourceClass,
		MemberName:  member.Name,
	})
	if err != nil || !resp.Found {
		return
	}
	m.sourceTarget = resp.Line - 1
	m.ensureSourceVisible(m.sourceTarget)
}

func (m *Model) ensureSourceVisible(targetLine int) {
	m.sourceViewport.YOffset = targetLine - m.sourceViewportHeight()/2
	inspectorui.EnsureRowVisible(
		&m.sourceViewport,
		targetLine,
		len(m.activeSourceLines()),
		m.sourceViewportHeight(),
	)
}

func (m *Model) activeSourceLines() []string {
	if m.showingReplSrc && len(m.replSourceLines) > 0 {
		return m.replSourceLines
	}
	return m.sourceLines
}

func (m *Model) sourceViewportHeight() int {
	h := m.contentHeight() - 1
	if h < 1 {
		h = 1
	}
	return h
}

func (m *Model) inspectPaneViewportHeight() int {
	h := m.contentHeight()
	if len(m.navStack) > 0 {
		h-- // breadcrumb line
	}
	h-- // pane header
	if h < 1 {
		h = 1
	}
	return h
}

func (m *Model) stackPaneViewportHeight() int {
	h := m.contentHeight() - 1 // error banner line
	h--                        // pane header
	if h < 1 {
		h = 1
	}
	return h
}

func (m *Model) contentHeight() int {
	overhead := 3 // header + status + help
	if m.cmdActive {
		overhead++
	}
	overhead += 3 // REPL area
	h := m.height - overhead
	if h < 3 {
		h = 3
	}
	return h
}

func (m *Model) listViewportHeight() int {
	h := m.contentHeight() - 2 // pane header + footer
	if h < 1 {
		h = 1
	}
	return h
}

// rebuildFileSyntaxSpans parses file source with tree-sitter and builds syntax spans.
func (m *Model) rebuildFileSyntaxSpans(source string) {
	m.fileSyntaxSpans = nil
	if m.tsParser == nil {
		return
	}
	root := m.tsParser.Parse([]byte(source))
	m.fileSyntaxSpans = jsparse.BuildSyntaxSpans(root)
}

// rebuildReplSyntaxSpans parses accumulated REPL source with tree-sitter and builds syntax spans.
func (m *Model) rebuildReplSyntaxSpans() {
	m.replSyntaxSpans = nil
	if m.tsParser == nil || len(m.replSourceLines) == 0 {
		return
	}
	src := strings.Join(m.replSourceLines, "\n")
	root := m.tsParser.Parse([]byte(src))
	m.replSyntaxSpans = jsparse.BuildSyntaxSpans(root)
}

// replSourceEntry tracks where a REPL expression lives in the accumulated source buffer.
type replSourceEntry struct {
	Expression string
	StartLine  int // 0-based index into replSourceLines
	EndLine    int // exclusive
}

// appendReplSource adds a REPL expression to the source log and returns the start line.
func (m *Model) appendReplSource(expr string) int {
	startLine := len(m.replSourceLines)

	// Add a separator comment showing the REPL expression number
	m.replSourceLines = append(m.replSourceLines,
		fmt.Sprintf("// ─── REPL [%d] ───", len(m.replSourceLog)+1))

	// Add the expression lines
	exprLines := strings.Split(expr, "\n")
	m.replSourceLines = append(m.replSourceLines, exprLines...)
	m.replSourceLines = append(m.replSourceLines, "")

	endLine := len(m.replSourceLines)
	m.replSourceLog = append(m.replSourceLog, replSourceEntry{
		Expression: expr,
		StartLine:  startLine,
		EndLine:    endLine,
	})

	// Rebuild syntax spans for REPL source
	m.rebuildReplSyntaxSpans()

	return startLine + 1 // +1 to skip the separator, point at actual code
}

// showReplFunctionSource switches the source pane to show a REPL-defined function's source.
func (m *Model) showReplFunctionSource(name, fnSrc string) {
	// Check if this expression is already in the REPL source log
	for _, entry := range m.replSourceLog {
		if strings.Contains(entry.Expression, fnSrc) || strings.Contains(entry.Expression, name) {
			// Found in REPL log — show REPL source at this entry
			m.showingReplSrc = true
			m.sourceTarget = entry.StartLine + 1 // point at the code, skip separator
			m.ensureReplSourceVisible(m.sourceTarget)
			return
		}
	}

	// Not found in log — create a temporary display from toString()
	m.showingReplSrc = true
	startLine := len(m.replSourceLines)
	m.replSourceLines = append(m.replSourceLines,
		fmt.Sprintf("// ─── %s (runtime) ───", name))
	m.replSourceLines = append(m.replSourceLines, strings.Split(fnSrc, "\n")...)
	m.replSourceLines = append(m.replSourceLines, "")
	m.sourceTarget = startLine + 1
	m.rebuildReplSyntaxSpans()
	m.ensureReplSourceVisible(m.sourceTarget)
}

func (m *Model) ensureReplSourceVisible(targetLine int) {
	m.ensureSourceVisible(targetLine)
}

// getFunctionSource returns the source text of a runtime function.
// goja's Value.String() returns the source for function values.
func getFunctionSource(val goja.Value) string {
	if _, ok := goja.AssertFunction(val); !ok {
		return ""
	}
	return val.String()
}

// kindIcon returns a display icon for binding kinds.
func kindIcon(k jsparse.BindingKind) string {
	//exhaustive:ignore
	switch k {
	case jsparse.BindingClass:
		return "C"
	case jsparse.BindingFunction:
		return "ƒ"
	default:
		return "●"
	}
}

// memberKindIcon returns a display icon for member kinds.
func memberKindIcon(k string) string {
	switch k {
	case "function":
		return "ƒ"
	case "value":
		return "●"
	case "param":
		return "→"
	default:
		return "·"
	}
}

// refreshRuntimeGlobals merges newly-defined runtime globals into the globals list.
func (m *Model) refreshRuntimeGlobals() {
	if m.inspectorService == nil || m.documentID == "" || m.rtSession == nil {
		return
	}

	resp, err := m.inspectorService.MergeRuntimeGlobals(inspectorapi.MergeRuntimeGlobalsRequest{
		DocumentID: m.documentID,
		Existing:   m.globals,
		Declared:   m.replDeclared,
	}, m.rtSession)
	if err != nil {
		return
	}
	m.globals = resp.Globals
}
