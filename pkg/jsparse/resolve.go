package jsparse

import (
	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/token"
	"github.com/dop251/goja/unistring"
)

// ScopeKind classifies a scope.
type ScopeKind int

const (
	ScopeGlobal ScopeKind = iota
	ScopeFunction
	ScopeBlock
	ScopeCatch
	ScopeFor
)

func (k ScopeKind) String() string {
	switch k {
	case ScopeGlobal:
		return "global"
	case ScopeFunction:
		return "function"
	case ScopeBlock:
		return "block"
	case ScopeCatch:
		return "catch"
	case ScopeFor:
		return "for"
	}
	return "unknown"
}

// BindingKind classifies how a name was introduced.
type BindingKind int

const (
	BindingVar BindingKind = iota
	BindingLet
	BindingConst
	BindingFunction
	BindingClass
	BindingParameter
	BindingCatchParam
)

func (k BindingKind) String() string {
	switch k {
	case BindingVar:
		return "var"
	case BindingLet:
		return "let"
	case BindingConst:
		return "const"
	case BindingFunction:
		return "function"
	case BindingClass:
		return "class"
	case BindingParameter:
		return "param"
	case BindingCatchParam:
		return "catch"
	}
	return "unknown"
}

// ScopeID uniquely identifies a scope.
type ScopeID int

// BindingRecord represents a single named binding (declaration).
type BindingRecord struct {
	Name       string
	Kind       BindingKind
	DeclNodeID NodeID   // the Identifier node at the declaration site
	ScopeID    ScopeID  // scope that owns this binding
	References []NodeID // Identifier nodes that reference this binding (excluding declaration)
}

// ScopeRecord represents a lexical scope.
type ScopeRecord struct {
	ID       ScopeID
	ParentID ScopeID // -1 for root
	Kind     ScopeKind
	Start    int // byte offset (1-based)
	End      int
	Bindings map[string]*BindingRecord
	Children []ScopeID
}

// Resolution holds the complete scope analysis result.
type Resolution struct {
	Scopes      map[ScopeID]*ScopeRecord
	RootScopeID ScopeID
	NodeBinding map[NodeID]*BindingRecord // for each Identifier, which binding it resolves to
	Unresolved  []NodeID                  // identifiers that couldn't be resolved
	nextScopeID ScopeID
}

// resolver is the internal state for scope resolution.
type resolver struct {
	index      *Index
	resolution *Resolution
	current    ScopeID
	program    *ast.Program
}

// Resolve performs scope analysis on a parsed AST, using the existing Index for node lookups.
func Resolve(program *ast.Program, idx *Index) *Resolution {
	res := &Resolution{
		Scopes:      make(map[ScopeID]*ScopeRecord),
		NodeBinding: make(map[NodeID]*BindingRecord),
	}

	r := &resolver{
		index:      idx,
		resolution: res,
		program:    program,
	}

	// Create global scope
	start := int(program.Idx0())
	end := int(program.Idx1())
	if start < 1 {
		start = 1
	}
	globalScope := r.pushScope(ScopeGlobal, -1, start, end)
	res.RootScopeID = globalScope

	// Pass 1: collect declarations (with hoisting)
	r.collectDeclarations(program.Body)

	// Pass 2: resolve references
	r.resolveStatements(program.Body)

	return res
}

// --- Scope management ---

func (r *resolver) pushScope(kind ScopeKind, parentID ScopeID, start, end int) ScopeID {
	id := r.resolution.nextScopeID
	r.resolution.nextScopeID++
	scope := &ScopeRecord{
		ID:       id,
		ParentID: parentID,
		Kind:     kind,
		Start:    start,
		End:      end,
		Bindings: make(map[string]*BindingRecord),
	}
	r.resolution.Scopes[id] = scope
	if parentID >= 0 {
		parent := r.resolution.Scopes[parentID]
		if parent != nil {
			parent.Children = append(parent.Children, id)
		}
	}
	r.current = id
	return id
}

func (r *resolver) popScope(previousID ScopeID) {
	r.current = previousID
}

// addBinding registers a declaration in the appropriate scope.
func (r *resolver) addBinding(name string, kind BindingKind, declNodeID NodeID) *BindingRecord {
	targetScope := r.current

	// var and function declarations hoist to the nearest function/global scope
	if kind == BindingVar || kind == BindingFunction {
		targetScope = r.nearestFunctionOrGlobal(r.current)
	}

	scope := r.resolution.Scopes[targetScope]
	if scope == nil {
		return nil
	}

	// If already bound in this scope, reuse the existing binding (var re-declaration is allowed)
	if existing, ok := scope.Bindings[name]; ok {
		// For var, just return the existing binding
		// For function, update the declaration node
		if kind == BindingFunction {
			existing.DeclNodeID = declNodeID
			existing.Kind = kind
		}
		r.resolution.NodeBinding[declNodeID] = existing
		return existing
	}

	b := &BindingRecord{
		Name:       name,
		Kind:       kind,
		DeclNodeID: declNodeID,
		ScopeID:    targetScope,
	}
	scope.Bindings[name] = b
	r.resolution.NodeBinding[declNodeID] = b
	return b
}

// nearestFunctionOrGlobal finds the enclosing function or global scope.
func (r *resolver) nearestFunctionOrGlobal(scopeID ScopeID) ScopeID {
	for id := scopeID; id >= 0; {
		s := r.resolution.Scopes[id]
		if s == nil {
			break
		}
		if s.Kind == ScopeFunction || s.Kind == ScopeGlobal {
			return id
		}
		id = s.ParentID
	}
	return r.resolution.RootScopeID
}

// lookupBinding resolves a name by walking the scope chain.
func (r *resolver) lookupBinding(name string) *BindingRecord {
	for id := r.current; id >= 0; {
		s := r.resolution.Scopes[id]
		if s == nil {
			break
		}
		if b, ok := s.Bindings[name]; ok {
			return b
		}
		id = s.ParentID
	}
	return nil
}

// findNodeID finds the NodeID for an ast.Identifier by matching offset.
// Returns the lowest NodeID on ties for determinism.
func (r *resolver) findNodeID(ident *ast.Identifier) NodeID {
	if ident == nil {
		return -1
	}
	start := int(ident.Idx0())
	end := int(ident.Idx1())
	bestID := NodeID(-1)
	for id, node := range r.index.Nodes {
		if node.Kind == "Identifier" && node.Start == start && node.End == end {
			if bestID < 0 || id < bestID {
				bestID = id
			}
		}
	}
	return bestID
}

// --- Pass 1: Collect declarations ---

func (r *resolver) collectDeclarations(stmts []ast.Statement) {
	// First pass: hoist function declarations and var statements
	for _, stmt := range stmts {
		r.collectDeclStatement(stmt)
	}
}

func (r *resolver) collectDeclStatement(stmt ast.Statement) {
	if stmt == nil {
		return
	}
	switch s := stmt.(type) {
	case *ast.FunctionDeclaration:
		if s.Function != nil && s.Function.Name != nil {
			nodeID := r.findNodeID(s.Function.Name)
			r.addBinding(string(s.Function.Name.Name), BindingFunction, nodeID)
		}
	case *ast.VariableStatement:
		for _, b := range s.List {
			r.collectVarBinding(b)
		}
	case *ast.LexicalDeclaration:
		kind := BindingLet
		if s.Token == token.CONST {
			kind = BindingConst
		}
		for _, b := range s.List {
			r.collectLexicalBinding(b, kind)
		}
	case *ast.ClassDeclaration:
		if s.Class != nil && s.Class.Name != nil {
			nodeID := r.findNodeID(s.Class.Name)
			r.addBinding(string(s.Class.Name.Name), BindingClass, nodeID)
		}
	case *ast.BlockStatement:
		// Check if block contains lexical declarations
		if r.hasLexicalDecls(s.List) {
			saved := r.current
			start := int(s.Idx0())
			end := int(s.Idx1())
			if start < 1 {
				start = 1
			}
			r.pushScope(ScopeBlock, saved, start, end)
			r.collectDeclarations(s.List)
			r.popScope(saved)
		} else {
			r.collectDeclarations(s.List)
		}
	case *ast.IfStatement:
		r.collectDeclStatement(s.Consequent)
		if s.Alternate != nil {
			r.collectDeclStatement(s.Alternate)
		}
	case *ast.ForStatement:
		if s.Initializer != nil {
			if lexInit, ok := s.Initializer.(*ast.ForLoopInitializerLexicalDecl); ok {
				saved := r.current
				start := int(s.For)
				end := int(s.Body.Idx1())
				if start < 1 {
					start = 1
				}
				r.pushScope(ScopeFor, saved, start, end)
				kind := BindingLet
				if lexInit.LexicalDeclaration.Token == token.CONST {
					kind = BindingConst
				}
				for _, b := range lexInit.LexicalDeclaration.List {
					r.collectLexicalBinding(b, kind)
				}
				r.collectDeclStatement(s.Body)
				r.popScope(saved)
				return
			}
		}
		r.collectDeclStatement(s.Body)
	case *ast.ForInStatement:
		if decl, ok := s.Into.(*ast.ForDeclaration); ok {
			saved := r.current
			start := int(s.For)
			end := int(s.Body.Idx1())
			if start < 1 {
				start = 1
			}
			r.pushScope(ScopeFor, saved, start, end)
			kind := BindingLet
			if decl.IsConst {
				kind = BindingConst
			}
			r.collectBindingTarget(decl.Target, kind)
			r.collectDeclStatement(s.Body)
			r.popScope(saved)
			return
		}
		r.collectDeclStatement(s.Body)
	case *ast.ForOfStatement:
		if decl, ok := s.Into.(*ast.ForDeclaration); ok {
			saved := r.current
			start := int(s.For)
			end := int(s.Body.Idx1())
			if start < 1 {
				start = 1
			}
			r.pushScope(ScopeFor, saved, start, end)
			kind := BindingLet
			if decl.IsConst {
				kind = BindingConst
			}
			r.collectBindingTarget(decl.Target, kind)
			r.collectDeclStatement(s.Body)
			r.popScope(saved)
			return
		}
		r.collectDeclStatement(s.Body)
	case *ast.WhileStatement:
		r.collectDeclStatement(s.Body)
	case *ast.DoWhileStatement:
		r.collectDeclStatement(s.Body)
	case *ast.WithStatement:
		r.collectDeclStatement(s.Body)
	case *ast.SwitchStatement:
		// switch body can have lexical declarations
		hasLex := false
		for _, c := range s.Body {
			if r.hasLexicalDecls(c.Consequent) {
				hasLex = true
				break
			}
		}
		if hasLex {
			saved := r.current
			start := int(s.Switch)
			end := int(s.Idx1())
			if start < 1 {
				start = 1
			}
			r.pushScope(ScopeBlock, saved, start, end)
			for _, c := range s.Body {
				r.collectDeclarations(c.Consequent)
			}
			r.popScope(saved)
		} else {
			for _, c := range s.Body {
				r.collectDeclarations(c.Consequent)
			}
		}
	case *ast.TryStatement:
		r.collectDeclStatement(s.Body)
		if s.Catch != nil {
			saved := r.current
			start := int(s.Catch.Catch)
			end := int(s.Catch.Body.Idx1())
			if start < 1 {
				start = 1
			}
			r.pushScope(ScopeCatch, saved, start, end)
			if s.Catch.Parameter != nil {
				r.collectBindingTarget(s.Catch.Parameter, BindingCatchParam)
			}
			r.collectDeclarations(s.Catch.Body.List)
			r.popScope(saved)
		}
		if s.Finally != nil {
			r.collectDeclStatement(s.Finally)
		}
	case *ast.LabelledStatement:
		r.collectDeclStatement(s.Statement)
	}
}

func (r *resolver) collectVarBinding(b *ast.Binding) {
	if b == nil {
		return
	}
	r.collectBindingTarget(b.Target, BindingVar)
}

func (r *resolver) collectLexicalBinding(b *ast.Binding, kind BindingKind) {
	if b == nil {
		return
	}
	r.collectBindingTarget(b.Target, kind)
}

func (r *resolver) collectBindingTarget(target ast.BindingTarget, kind BindingKind) {
	if target == nil {
		return
	}
	switch t := target.(type) {
	case *ast.Identifier:
		nodeID := r.findNodeID(t)
		r.addBinding(string(t.Name), kind, nodeID)
	case *ast.ObjectPattern:
		for _, prop := range t.Properties {
			switch p := prop.(type) {
			case *ast.PropertyShort:
				nodeID := r.findNodeID(&p.Name)
				r.addBinding(string(p.Name.Name), kind, nodeID)
			case *ast.PropertyKeyed:
				if bt, ok := p.Value.(ast.BindingTarget); ok {
					r.collectBindingTarget(bt, kind)
				}
			}
		}
		if t.Rest != nil {
			if bt, ok := t.Rest.(ast.BindingTarget); ok {
				r.collectBindingTarget(bt, kind)
			}
		}
	case *ast.ArrayPattern:
		for _, elt := range t.Elements {
			if elt != nil {
				if bt, ok := elt.(ast.BindingTarget); ok {
					r.collectBindingTarget(bt, kind)
				}
			}
		}
		if t.Rest != nil {
			if bt, ok := t.Rest.(ast.BindingTarget); ok {
				r.collectBindingTarget(bt, kind)
			}
		}
	}
}

func (r *resolver) hasLexicalDecls(stmts []ast.Statement) bool {
	for _, s := range stmts {
		switch s.(type) {
		case *ast.LexicalDeclaration, *ast.ClassDeclaration:
			return true
		}
	}
	return false
}

// --- Pass 2: Resolve references ---

func (r *resolver) resolveStatements(stmts []ast.Statement) {
	for _, stmt := range stmts {
		r.resolveStatement(stmt)
	}
}

func (r *resolver) resolveStatement(stmt ast.Statement) {
	if stmt == nil {
		return
	}
	switch s := stmt.(type) {
	case *ast.ExpressionStatement:
		r.resolveExpression(s.Expression)
	case *ast.VariableStatement:
		for _, b := range s.List {
			// Only resolve the initializer — the target identifier is a declaration
			if b.Initializer != nil {
				r.resolveExpression(b.Initializer)
			}
		}
	case *ast.LexicalDeclaration:
		for _, b := range s.List {
			if b.Initializer != nil {
				r.resolveExpression(b.Initializer)
			}
		}
	case *ast.ReturnStatement:
		if s.Argument != nil {
			r.resolveExpression(s.Argument)
		}
	case *ast.ThrowStatement:
		r.resolveExpression(s.Argument)
	case *ast.IfStatement:
		r.resolveExpression(s.Test)
		r.resolveStatement(s.Consequent)
		if s.Alternate != nil {
			r.resolveStatement(s.Alternate)
		}
	case *ast.BlockStatement:
		// Enter the block scope if one was created during declaration collection
		scopeID := r.findScopeForBlock(s)
		if scopeID >= 0 {
			saved := r.current
			r.current = scopeID
			r.resolveStatements(s.List)
			r.popScope(saved)
		} else {
			r.resolveStatements(s.List)
		}
	case *ast.ForStatement:
		// Check if a for-scope was created
		scopeID := r.findScopeForOffset(int(s.For), int(s.Body.Idx1()), ScopeFor)
		saved := r.current
		if scopeID >= 0 {
			r.current = scopeID
		}
		if s.Initializer != nil {
			r.resolveForInit(s.Initializer)
		}
		if s.Test != nil {
			r.resolveExpression(s.Test)
		}
		if s.Update != nil {
			r.resolveExpression(s.Update)
		}
		r.resolveStatement(s.Body)
		if scopeID >= 0 {
			r.popScope(saved)
		}
	case *ast.ForInStatement:
		scopeID := r.findScopeForOffset(int(s.For), int(s.Body.Idx1()), ScopeFor)
		saved := r.current
		if scopeID >= 0 {
			r.current = scopeID
		}
		r.resolveExpression(s.Source)
		r.resolveStatement(s.Body)
		if scopeID >= 0 {
			r.popScope(saved)
		}
	case *ast.ForOfStatement:
		scopeID := r.findScopeForOffset(int(s.For), int(s.Body.Idx1()), ScopeFor)
		saved := r.current
		if scopeID >= 0 {
			r.current = scopeID
		}
		r.resolveExpression(s.Source)
		r.resolveStatement(s.Body)
		if scopeID >= 0 {
			r.popScope(saved)
		}
	case *ast.WhileStatement:
		r.resolveExpression(s.Test)
		r.resolveStatement(s.Body)
	case *ast.DoWhileStatement:
		r.resolveStatement(s.Body)
		r.resolveExpression(s.Test)
	case *ast.SwitchStatement:
		r.resolveExpression(s.Discriminant)
		// Enter switch scope if created
		scopeID := r.findScopeForOffset(int(s.Switch), int(s.Idx1()), ScopeBlock)
		saved := r.current
		if scopeID >= 0 {
			r.current = scopeID
		}
		for _, c := range s.Body {
			if c.Test != nil {
				r.resolveExpression(c.Test)
			}
			r.resolveStatements(c.Consequent)
		}
		if scopeID >= 0 {
			r.popScope(saved)
		}
	case *ast.TryStatement:
		r.resolveStatement(s.Body)
		if s.Catch != nil {
			scopeID := r.findScopeForOffset(int(s.Catch.Catch), int(s.Catch.Body.Idx1()), ScopeCatch)
			saved := r.current
			if scopeID >= 0 {
				r.current = scopeID
			}
			r.resolveStatements(s.Catch.Body.List)
			if scopeID >= 0 {
				r.popScope(saved)
			}
		}
		if s.Finally != nil {
			r.resolveStatement(s.Finally)
		}
	case *ast.WithStatement:
		r.resolveExpression(s.Object)
		r.resolveStatement(s.Body)
	case *ast.LabelledStatement:
		r.resolveStatement(s.Statement)
	case *ast.FunctionDeclaration:
		// Resolve the function body in a new function scope
		if s.Function != nil {
			r.resolveFunctionLiteral(s.Function)
		}
	case *ast.ClassDeclaration:
		if s.Class != nil {
			r.resolveClassLiteral(s.Class)
		}
	}
}

func (r *resolver) resolveForInit(init ast.ForLoopInitializer) {
	switch i := init.(type) {
	case *ast.ForLoopInitializerExpression:
		r.resolveExpression(i.Expression)
	case *ast.ForLoopInitializerVarDeclList:
		for _, b := range i.List {
			if b.Initializer != nil {
				r.resolveExpression(b.Initializer)
			}
		}
	case *ast.ForLoopInitializerLexicalDecl:
		for _, b := range i.LexicalDeclaration.List {
			if b.Initializer != nil {
				r.resolveExpression(b.Initializer)
			}
		}
	}
}

func (r *resolver) resolveExpression(expr ast.Expression) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *ast.Identifier:
		r.resolveIdentifier(e)
	case *ast.CallExpression:
		r.resolveExpression(e.Callee)
		for _, arg := range e.ArgumentList {
			r.resolveExpression(arg)
		}
	case *ast.DotExpression:
		r.resolveExpression(e.Left)
		// e.Identifier is a property access — NOT a variable reference; skip it
	case *ast.PrivateDotExpression:
		r.resolveExpression(e.Left)
	case *ast.BracketExpression:
		r.resolveExpression(e.Left)
		r.resolveExpression(e.Member)
	case *ast.AssignExpression:
		r.resolveExpression(e.Left)
		r.resolveExpression(e.Right)
	case *ast.BinaryExpression:
		r.resolveExpression(e.Left)
		r.resolveExpression(e.Right)
	case *ast.UnaryExpression:
		r.resolveExpression(e.Operand)
	case *ast.ConditionalExpression:
		r.resolveExpression(e.Test)
		r.resolveExpression(e.Consequent)
		r.resolveExpression(e.Alternate)
	case *ast.NewExpression:
		r.resolveExpression(e.Callee)
		for _, arg := range e.ArgumentList {
			r.resolveExpression(arg)
		}
	case *ast.SequenceExpression:
		for _, expr := range e.Sequence {
			r.resolveExpression(expr)
		}
	case *ast.ObjectLiteral:
		for _, prop := range e.Value {
			r.resolveProperty(prop)
		}
	case *ast.ArrayLiteral:
		for _, elt := range e.Value {
			if elt != nil {
				r.resolveExpression(elt)
			}
		}
	case *ast.FunctionLiteral:
		r.resolveFunctionLiteral(e)
	case *ast.ArrowFunctionLiteral:
		r.resolveArrowFunction(e)
	case *ast.ClassLiteral:
		r.resolveClassLiteral(e)
	case *ast.TemplateLiteral:
		for _, expr := range e.Expressions {
			r.resolveExpression(expr)
		}
	case *ast.YieldExpression:
		if e.Argument != nil {
			r.resolveExpression(e.Argument)
		}
	case *ast.AwaitExpression:
		if e.Argument != nil {
			r.resolveExpression(e.Argument)
		}
	case *ast.MetaProperty:
		// new.target, import.meta — not variable references
	case *ast.SpreadElement:
		r.resolveExpression(e.Expression)
	case *ast.Binding:
		// In expression context (e.g. default params)
		if e.Initializer != nil {
			r.resolveExpression(e.Initializer)
		}
	}
}

func (r *resolver) resolveProperty(prop ast.Property) {
	switch p := prop.(type) {
	case *ast.PropertyKeyed:
		if p.Computed {
			r.resolveExpression(p.Key)
		}
		r.resolveExpression(p.Value)
	case *ast.PropertyShort:
		// Shorthand property { x } is both a reference to x and a property name
		r.resolveIdentifier(&p.Name)
		if p.Initializer != nil {
			r.resolveExpression(p.Initializer)
		}
	case *ast.SpreadElement:
		r.resolveExpression(p.Expression)
	}
}

func (r *resolver) resolveIdentifier(ident *ast.Identifier) {
	if ident == nil {
		return
	}
	name := string(ident.Name)
	nodeID := r.findNodeID(ident)
	if nodeID < 0 {
		return
	}

	// Skip if this is already registered as a declaration
	if _, alreadyBound := r.resolution.NodeBinding[nodeID]; alreadyBound {
		return
	}

	b := r.lookupBinding(name)
	if b != nil {
		b.References = append(b.References, nodeID)
		r.resolution.NodeBinding[nodeID] = b
	} else {
		r.resolution.Unresolved = append(r.resolution.Unresolved, nodeID)
	}
}

func (r *resolver) resolveFunctionLiteral(fn *ast.FunctionLiteral) {
	if fn == nil || fn.Body == nil {
		return
	}
	saved := r.current
	start := int(fn.Idx0())
	end := int(fn.Idx1())
	if start < 1 {
		start = 1
	}
	r.pushScope(ScopeFunction, saved, start, end)

	// Bind the function name inside the function scope (for named function expressions)
	if fn.Name != nil {
		nodeID := r.findNodeID(fn.Name)
		// Only add as internal binding if not already declared in outer scope
		if _, exists := r.resolution.NodeBinding[nodeID]; !exists {
			r.addBinding(string(fn.Name.Name), BindingFunction, nodeID)
		}
	}

	// Bind parameters
	if fn.ParameterList != nil {
		for _, param := range fn.ParameterList.List {
			if param != nil {
				r.collectBindingTarget(param.Target, BindingParameter)
			}
		}
		if fn.ParameterList.Rest != nil {
			if bt, ok := fn.ParameterList.Rest.(ast.BindingTarget); ok {
				r.collectBindingTarget(bt, BindingParameter)
			}
		}
	}

	// Collect declarations in body
	r.collectDeclarations(fn.Body.List)

	// Resolve parameter default values
	if fn.ParameterList != nil {
		for _, param := range fn.ParameterList.List {
			if param != nil && param.Initializer != nil {
				r.resolveExpression(param.Initializer)
			}
		}
	}

	// Resolve body
	r.resolveStatements(fn.Body.List)

	r.popScope(saved)
}

func (r *resolver) resolveArrowFunction(fn *ast.ArrowFunctionLiteral) {
	if fn == nil {
		return
	}
	saved := r.current
	start := int(fn.Idx0())
	end := int(fn.Idx1())
	if start < 1 {
		start = 1
	}
	r.pushScope(ScopeFunction, saved, start, end)

	// Bind parameters
	if fn.ParameterList != nil {
		for _, param := range fn.ParameterList.List {
			if param != nil {
				r.collectBindingTarget(param.Target, BindingParameter)
			}
		}
		if fn.ParameterList.Rest != nil {
			if bt, ok := fn.ParameterList.Rest.(ast.BindingTarget); ok {
				r.collectBindingTarget(bt, BindingParameter)
			}
		}
	}

	// Resolve body
	switch body := fn.Body.(type) {
	case *ast.BlockStatement:
		r.collectDeclarations(body.List)
		r.resolveStatements(body.List)
	case *ast.ExpressionBody:
		r.resolveExpression(body.Expression)
	}

	r.popScope(saved)
}

func (r *resolver) resolveClassLiteral(cls *ast.ClassLiteral) {
	if cls == nil {
		return
	}
	if cls.SuperClass != nil {
		r.resolveExpression(cls.SuperClass)
	}
	for _, elem := range cls.Body {
		switch e := elem.(type) {
		case *ast.MethodDefinition:
			if e.Computed {
				r.resolveExpression(e.Key)
			}
			if e.Body != nil {
				r.resolveFunctionLiteral(e.Body)
			}
		case *ast.FieldDefinition:
			if e.Computed {
				r.resolveExpression(e.Key)
			}
			if e.Initializer != nil {
				r.resolveExpression(e.Initializer)
			}
		case *ast.ClassStaticBlock:
			if e.Block != nil {
				r.resolveStatements(e.Block.List)
			}
		}
	}
}

// --- Scope lookup helpers ---

// findScopeForBlock finds a child scope that was created for this block statement.
func (r *resolver) findScopeForBlock(block *ast.BlockStatement) ScopeID {
	start := int(block.Idx0())
	end := int(block.Idx1())
	return r.findChildScope(start, end, ScopeBlock)
}

// findScopeForOffset finds a child scope matching the given offset range and kind.
func (r *resolver) findScopeForOffset(start, end int, kind ScopeKind) ScopeID {
	if start < 1 {
		start = 1
	}
	return r.findChildScope(start, end, kind)
}

func (r *resolver) findChildScope(start, end int, kind ScopeKind) ScopeID {
	current := r.resolution.Scopes[r.current]
	if current == nil {
		return -1
	}
	for _, childID := range current.Children {
		child := r.resolution.Scopes[childID]
		if child != nil && child.Kind == kind && child.Start == start && child.End == end {
			return childID
		}
	}
	return -1
}

// BindingForNode returns the binding associated with a given node ID, if any.
func (res *Resolution) BindingForNode(nodeID NodeID) *BindingRecord {
	return res.NodeBinding[nodeID]
}

// IsDeclaration returns true if this node is the declaration site of its binding.
func (res *Resolution) IsDeclaration(nodeID NodeID) bool {
	b := res.NodeBinding[nodeID]
	return b != nil && b.DeclNodeID == nodeID
}

// IsReference returns true if this node references (but doesn't declare) a binding.
func (res *Resolution) IsReference(nodeID NodeID) bool {
	b := res.NodeBinding[nodeID]
	return b != nil && b.DeclNodeID != nodeID
}

// IsUnresolved returns true if this identifier couldn't be resolved.
func (res *Resolution) IsUnresolved(nodeID NodeID) bool {
	for _, id := range res.Unresolved {
		if id == nodeID {
			return true
		}
	}
	return false
}

// AllUsages returns declaration + all references for a binding.
func (b *BindingRecord) AllUsages() []NodeID {
	result := make([]NodeID, 0, 1+len(b.References))
	if b.DeclNodeID >= 0 {
		result = append(result, b.DeclNodeID)
	}
	result = append(result, b.References...)
	return result
}

// suppress unused import warnings
var _ unistring.String
