package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dop251/goja"
	"github.com/dop251/goja/ast"
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
	Name      string
	Kind      string // "function", "value", "param"
	Preview   string // short value/signature preview
	Inherited bool
	Source    string // from which prototype level
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
	sourceScroll int
	sourceTarget int // target line to highlight (0-based), -1 = none

	// Runtime
	rtSession   *runtime.Session
	replInput   textinput.Model
	replHistory []string
	replResult  string
	replError   string

	// Object browser (for REPL results and inspect targets)
	inspectObj   *goja.Object
	inspectProps []runtime.PropertyInfo
	inspectIdx   int
	inspectLabel string

	// Navigation stack for drill-in inspection
	navStack []NavFrame

	// UI state
	focus     FocusPane
	mode      string
	width     int
	height    int
	loaded    bool
	statusMsg string

	// Components
	keyMap    KeyMap
	help      help.Model
	spinner   spinner.Model
	command   textinput.Model
	cmdActive bool
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

	m.replInput = textinput.New()
	m.replInput.Prompt = "» "
	m.replInput.Placeholder = "evaluate expression..."
	m.replInput.CharLimit = 512
	m.replInput.Width = 60
	m.replInput.Blur()

	return m
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
	for _, stmt := range m.analysis.Program.Body {
		cd, ok := stmt.(*ast.ClassDeclaration)
		if !ok || cd.Class == nil || cd.Class.Name == nil {
			continue
		}
		if string(cd.Class.Name.Name) != className {
			continue
		}
		if cd.Class.SuperClass != nil {
			if ident, ok := cd.Class.SuperClass.(*ast.Identifier); ok {
				return string(ident.Name)
			}
		}
		break
	}
	return ""
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
	}
}

// buildClassMembers extracts methods and properties from a class declaration.
func (m *Model) buildClassMembers(className string) {
	if m.analysis.Program == nil {
		return
	}

	for _, stmt := range m.analysis.Program.Body {
		cd, ok := stmt.(*ast.ClassDeclaration)
		if !ok || cd.Class == nil || cd.Class.Name == nil {
			continue
		}
		if string(cd.Class.Name.Name) != className {
			continue
		}

		for _, elem := range cd.Class.Body {
			switch e := elem.(type) {
			case *ast.MethodDefinition:
				name := astMethodName(e)
				kind := "function"
				preview := "()"
				if e.Body != nil && e.Body.ParameterList != nil {
					params := astParamNames(e.Body.ParameterList)
					preview = "(" + strings.Join(params, ", ") + ")"
				}
				m.members = append(m.members, MemberItem{
					Name:    name,
					Kind:    kind,
					Preview: preview,
				})
			case *ast.FieldDefinition:
				name := astFieldName(e)
				m.members = append(m.members, MemberItem{
					Name: name,
					Kind: "value",
				})
			}
		}

		// If extends another class, add inherited members
		if cd.Class.SuperClass != nil {
			if ident, ok := cd.Class.SuperClass.(*ast.Identifier); ok {
				superName := string(ident.Name)
				m.addInheritedMembers(superName, superName)
			}
		}
		break
	}
}

// addInheritedMembers adds methods from a parent class.
func (m *Model) addInheritedMembers(className, source string) {
	if m.analysis.Program == nil {
		return
	}

	for _, stmt := range m.analysis.Program.Body {
		cd, ok := stmt.(*ast.ClassDeclaration)
		if !ok || cd.Class == nil || cd.Class.Name == nil {
			continue
		}
		if string(cd.Class.Name.Name) != className {
			continue
		}

		for _, elem := range cd.Class.Body {
			if md, ok := elem.(*ast.MethodDefinition); ok {
				name := astMethodName(md)
				if name == "constructor" {
					continue
				}
				if m.hasMember(name) {
					continue
				}
				kind := "function"
				preview := "()"
				if md.Body != nil && md.Body.ParameterList != nil {
					params := astParamNames(md.Body.ParameterList)
					preview = "(" + strings.Join(params, ", ") + ")"
				}
				m.members = append(m.members, MemberItem{
					Name:      name,
					Kind:      kind,
					Preview:   preview,
					Inherited: true,
					Source:    source,
				})
			}
		}

		// Recurse into grandparent
		if cd.Class.SuperClass != nil {
			if ident, ok := cd.Class.SuperClass.(*ast.Identifier); ok {
				m.addInheritedMembers(string(ident.Name), string(ident.Name))
			}
		}
		break
	}
}

// buildFunctionMembers shows parameters for a selected function.
func (m *Model) buildFunctionMembers(funcName string) {
	if m.analysis.Program == nil {
		return
	}

	for _, stmt := range m.analysis.Program.Body {
		fd, ok := stmt.(*ast.FunctionDeclaration)
		if !ok || fd.Function == nil || fd.Function.Name == nil {
			continue
		}
		if string(fd.Function.Name.Name) != funcName {
			continue
		}

		if fd.Function.ParameterList != nil {
			params := astParamNames(fd.Function.ParameterList)
			for _, p := range params {
				m.members = append(m.members, MemberItem{
					Name: p,
					Kind: "param",
				})
			}
		}
		break
	}
}

func (m *Model) hasMember(name string) bool {
	for _, mem := range m.members {
		if mem.Name == name {
			return true
		}
	}
	return false
}

// jumpToSource sets the source pane to show the declaration of the selected member.
func (m *Model) jumpToSource() {
	m.sourceTarget = -1

	if m.analysis == nil || m.analysis.Index == nil {
		return
	}

	if m.focus == FocusGlobals && len(m.globals) > 0 {
		m.jumpToBinding(m.globals[m.globalIdx].Name)
	} else if m.focus == FocusMembers && len(m.members) > 0 && len(m.globals) > 0 {
		member := m.members[m.memberIdx]
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
	vpHeight := m.sourceViewportHeight()
	if vpHeight <= 0 {
		return
	}
	m.sourceScroll = targetLine - vpHeight/2
	if m.sourceScroll < 0 {
		m.sourceScroll = 0
	}
	maxScroll := len(m.sourceLines) - vpHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.sourceScroll > maxScroll {
		m.sourceScroll = maxScroll
	}
}

func (m *Model) sourceViewportHeight() int {
	h := m.contentHeight() - 1
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

func astFieldName(fd *ast.FieldDefinition) string {
	if fd == nil {
		return "<unknown>"
	}
	switch k := fd.Key.(type) {
	case *ast.Identifier:
		return string(k.Name)
	case *ast.StringLiteral:
		return k.Literal
	}
	return "<computed>"
}

func astParamNames(pl *ast.ParameterList) []string {
	var names []string
	if pl == nil {
		return names
	}
	for _, p := range pl.List {
		if p == nil || p.Target == nil {
			continue
		}
		if ident, ok := p.Target.(*ast.Identifier); ok {
			names = append(names, string(ident.Name))
		}
	}
	return names
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatStatus(parts ...string) string {
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, " │ ")
}
