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
	"github.com/dop251/goja/ast"
	mode_keymap "github.com/go-go-golems/bobatea/pkg/mode-keymap"
	"github.com/go-go-golems/go-go-goja/internal/inspectorui"
	inspectorcore "github.com/go-go-golems/go-go-goja/pkg/inspector/core"
	"github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// GlobalItem represents a top-level binding in the globals list.
type GlobalItem struct {
	Name    string
	Kind    jsparse.BindingKind
	Extends string // for classes that extend another
}

// MemberItem represents a member of a selected class/object.
type MemberItem struct {
	Name           string
	Kind           string // "function", "value", "param"
	Preview        string // short value/signature preview
	Inherited      bool
	Source         string // from which prototype level
	RuntimeDerived bool   // true if populated from runtime introspection (not AST)
}

// Model is the top-level bubbletea model for the Smalltalk inspector.
type Model struct {
	// Data
	filename string
	source   string
	analysis *jsparse.AnalysisResult

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
	rtSession        *runtime.Session
	replInput        textinput.Model
	replHistory      []string
	replResult       string
	replError        string
	replDefinedNames []string // names defined via REPL (const/let/class not on globalObject)

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
		filename:     filename,
		focus:        FocusGlobals,
		mode:         modeEmpty,
		globalIdx:    0,
		memberIdx:    0,
		sourceTarget: -1,
		keyMap:       newKeyMap(),
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
	if m.analysis == nil || m.analysis.Resolution == nil {
		return
	}

	rootScope := m.analysis.Resolution.Scopes[m.analysis.Resolution.RootScopeID]
	if rootScope == nil {
		return
	}

	// Collect entries
	type entry struct {
		name    string
		binding *jsparse.BindingRecord
	}
	var entries []entry
	for name, b := range rootScope.Bindings {
		entries = append(entries, entry{name, b})
	}
	// Sort: classes first, then functions, then values; within each, alphabetical
	sortOrder := func(k jsparse.BindingKind) int {
		//exhaustive:ignore
		switch k {
		case jsparse.BindingClass:
			return 0
		case jsparse.BindingFunction:
			return 1
		default:
			return 2
		}
	}
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			oi, oj := sortOrder(entries[i].binding.Kind), sortOrder(entries[j].binding.Kind)
			if oi > oj || (oi == oj && entries[i].name > entries[j].name) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	for _, e := range entries {
		gi := GlobalItem{
			Name: e.name,
			Kind: e.binding.Kind,
		}

		// Try to find extends info for classes
		if e.binding.Kind == jsparse.BindingClass {
			gi.Extends = m.findClassExtends(e.name)
		}

		m.globals = append(m.globals, gi)
	}

	m.globalIdx = 0
	m.globalScroll = 0
}

// findClassExtends tries to find the "extends" name for a class declaration.
func (m *Model) findClassExtends(className string) string {
	if m.analysis == nil || m.analysis.Program == nil {
		return ""
	}
	return inspectorcore.ClassExtends(m.analysis.Program, className)
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

	if m.analysis == nil || m.analysis.Resolution == nil || m.analysis.Index == nil {
		return
	}

	//exhaustive:ignore
	switch selected.Kind {
	case jsparse.BindingClass:
		m.buildClassMembers(selected.Name)
	case jsparse.BindingFunction:
		m.buildFunctionMembers(selected.Name)
	default:
		m.buildValueMembers(selected.Name)
	}
}

// buildClassMembers extracts methods and properties from a class declaration.
func (m *Model) buildClassMembers(className string) {
	if m.analysis.Program == nil {
		return
	}
	for _, member := range inspectorcore.BuildClassMembers(m.analysis.Program, className) {
		m.members = append(m.members, MemberItem{
			Name:      member.Name,
			Kind:      member.Kind,
			Preview:   member.Preview,
			Inherited: member.Inherited,
			Source:    member.Source,
		})
	}
}

// buildFunctionMembers shows parameters for a selected function.
func (m *Model) buildFunctionMembers(funcName string) {
	if m.analysis.Program == nil {
		return
	}
	for _, member := range inspectorcore.BuildFunctionMembers(m.analysis.Program, funcName) {
		m.members = append(m.members, MemberItem{
			Name: member.Name,
			Kind: member.Kind,
		})
	}
}

// buildValueMembers uses the runtime session to introspect a value-type global.
func (m *Model) buildValueMembers(name string) {
	if m.rtSession == nil {
		return
	}

	val := m.rtSession.GlobalValue(name)
	if val == nil || goja.IsUndefined(val) {
		m.members = append(m.members, MemberItem{
			Name:           "(value)",
			Kind:           "value",
			Preview:        " : undefined",
			RuntimeDerived: true,
		})
		return
	}

	if goja.IsNull(val) {
		m.members = append(m.members, MemberItem{
			Name:           "(value)",
			Kind:           "value",
			Preview:        " : null",
			RuntimeDerived: true,
		})
		return
	}

	// Check if it's an object — show its properties
	if obj, ok := val.(*goja.Object); ok {
		props := runtime.InspectObject(obj, m.rtSession.VM)
		for _, p := range props {
			m.members = append(m.members, MemberItem{
				Name:           p.Name,
				Kind:           p.Kind,
				Preview:        " : " + p.Preview,
				RuntimeDerived: true,
			})
		}
		return
	}

	// Primitive value — show a single entry with the preview
	m.members = append(m.members, MemberItem{
		Name:           "(value)",
		Kind:           "value",
		Preview:        " : " + runtime.ValuePreview(val, m.rtSession.VM, 40),
		RuntimeDerived: true,
	})
}

// jumpToSource sets the source pane to show the declaration of the selected member.
func (m *Model) jumpToSource() {
	m.sourceTarget = -1
	m.showingReplSrc = false

	if m.analysis == nil || m.analysis.Index == nil {
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
	if m.analysis.Resolution == nil {
		return
	}
	rootScope := m.analysis.Resolution.Scopes[m.analysis.Resolution.RootScopeID]
	if rootScope == nil {
		return
	}
	b, ok := rootScope.Bindings[name]
	if !ok {
		return
	}
	declNode := m.analysis.Index.Nodes[b.DeclNodeID]
	if declNode == nil {
		return
	}
	m.sourceTarget = declNode.StartLine - 1
	m.ensureSourceVisible(m.sourceTarget)
}

func (m *Model) jumpToMember(className string, member MemberItem) {
	if m.analysis.Program == nil {
		return
	}

	searchClass := className
	if member.Inherited && member.Source != "" {
		searchClass = member.Source
	}

	for _, stmt := range m.analysis.Program.Body {
		cd, ok := stmt.(*ast.ClassDeclaration)
		if !ok || cd.Class == nil || cd.Class.Name == nil {
			continue
		}
		if string(cd.Class.Name.Name) != searchClass {
			continue
		}

		for _, elem := range cd.Class.Body {
			if md, ok := elem.(*ast.MethodDefinition); ok {
				if astMethodName(md) == member.Name {
					offset := int(md.Idx0())
					if m.analysis.Index != nil {
						l, _ := m.analysis.Index.OffsetToLineCol(offset)
						m.sourceTarget = l - 1
						m.ensureSourceVisible(m.sourceTarget)
					}
					return
				}
			}
		}
		break
	}
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

// AST helper functions

func astMethodName(md *ast.MethodDefinition) string {
	if md == nil {
		return "<unknown>"
	}
	switch k := md.Key.(type) {
	case *ast.Identifier:
		return string(k.Name)
	case *ast.StringLiteral:
		return k.Literal
	}
	return "<computed>"
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
	if m.rtSession == nil {
		return
	}

	// Build a set of already-known names
	known := make(map[string]bool)
	for _, g := range m.globals {
		known[g.Name] = true
	}

	added := false

	// 1. Scan GlobalObject for new var/function names
	global := m.rtSession.VM.GlobalObject()
	for _, key := range global.Keys() {
		if known[key] || isBuiltinGlobal(key) {
			continue
		}
		val := global.Get(key)
		kind := jsparse.BindingVar
		if _, ok := goja.AssertFunction(val); ok {
			kind = jsparse.BindingFunction
		}
		m.globals = append(m.globals, GlobalItem{
			Name: key,
			Kind: kind,
		})
		known[key] = true
		added = true
	}

	// 2. Check tracked REPL names (const/let/class from REPL input)
	for _, name := range m.replDefinedNames {
		if known[name] {
			continue
		}
		// Verify it actually exists in the runtime
		val := m.rtSession.GlobalValue(name)
		if val == nil || goja.IsUndefined(val) {
			continue
		}
		m.globals = append(m.globals, GlobalItem{
			Name: name,
			Kind: jsparse.BindingConst,
		})
		known[name] = true
		added = true
	}

	if added {
		m.sortGlobals()
	}
}

// sortGlobals re-sorts the globals list (classes, functions, values; alphabetical within).
func (m *Model) sortGlobals() {
	sortOrder := func(k jsparse.BindingKind) int {
		//exhaustive:ignore
		switch k {
		case jsparse.BindingClass:
			return 0
		case jsparse.BindingFunction:
			return 1
		default:
			return 2
		}
	}
	for i := 0; i < len(m.globals); i++ {
		for j := i + 1; j < len(m.globals); j++ {
			oi, oj := sortOrder(m.globals[i].Kind), sortOrder(m.globals[j].Kind)
			if oi > oj || (oi == oj && m.globals[i].Name > m.globals[j].Name) {
				m.globals[i], m.globals[j] = m.globals[j], m.globals[i]
			}
		}
	}
}

// extractDeclaredNames parses REPL input for const/let/var/class/function declarations.
func extractDeclaredNames(expr string) []string {
	var names []string
	// Simple pattern matching for common REPL declarations
	words := strings.Fields(expr)
	for i := 0; i < len(words)-1; i++ {
		switch words[i] {
		case "const", "let", "var":
			name := strings.TrimRight(words[i+1], "=;,")
			if name != "" && name != "{" && name != "[" {
				names = append(names, name)
			}
		case "class":
			name := strings.TrimRight(words[i+1], "{")
			if name != "" {
				names = append(names, name)
			}
		case "function":
			name := strings.TrimRight(words[i+1], "(")
			if name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}

// isBuiltinGlobal returns true if the name is a standard JavaScript/goja builtin.
func isBuiltinGlobal(name string) bool {
	builtins := map[string]bool{
		"Object": true, "Array": true, "Math": true, "JSON": true,
		"String": true, "Number": true, "Boolean": true, "RegExp": true,
		"Date": true, "Error": true, "TypeError": true, "RangeError": true,
		"ReferenceError": true, "SyntaxError": true, "URIError": true, "EvalError": true,
		"Symbol": true, "Map": true, "Set": true, "WeakMap": true, "WeakSet": true,
		"Promise": true, "Proxy": true, "Reflect": true, "ArrayBuffer": true,
		"DataView": true, "Float32Array": true, "Float64Array": true,
		"Int8Array": true, "Int16Array": true, "Int32Array": true,
		"Uint8Array": true, "Uint16Array": true, "Uint32Array": true, "Uint8ClampedArray": true,
		"parseInt": true, "parseFloat": true, "isNaN": true, "isFinite": true,
		"undefined": true, "NaN": true, "Infinity": true, "eval": true,
		"encodeURI": true, "encodeURIComponent": true, "decodeURI": true, "decodeURIComponent": true,
		"escape": true, "unescape": true,
		"console": true, "globalThis": true, "require": true,
		"Function": true, "Iterator": true, "AggregateError": true,
		"SharedArrayBuffer": true, "Atomics": true, "WeakRef": true, "FinalizationRegistry": true,
	}
	return builtins[name]
}
