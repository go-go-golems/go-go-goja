package jsparse

import "strings"

// CompletionKind classifies the type of completion context.
type CompletionKind int

const (
	CompletionNone       CompletionKind = iota
	CompletionProperty                  // obj.  or obj.fo
	CompletionIdentifier                // bare identifier
	CompletionArgument                  // inside function call args
)

// CompletionContext describes what kind of completion is appropriate at the cursor.
type CompletionContext struct {
	Kind         CompletionKind
	BaseExpr     string // for Property: the expression before the dot
	BaseNodeKind string // tree-sitter kind of the base expression
	PartialText  string // text typed after the dot (or partial identifier)
	CursorRow    int
	CursorCol    int
}

// CandidateKind classifies a completion candidate.
type CandidateKind int

const (
	CandidateProperty CandidateKind = iota
	CandidateMethod
	CandidateVariable
	CandidateFunction
	CandidateKeyword
)

func (k CandidateKind) Icon() string {
	switch k {
	case CandidateProperty:
		return "●"
	case CandidateMethod:
		return "ƒ"
	case CandidateVariable:
		return "◆"
	case CandidateFunction:
		return "λ"
	case CandidateKeyword:
		return "⊞"
	}
	return " "
}

// CompletionCandidate is a single suggestion in the completion popup.
type CompletionCandidate struct {
	Label  string
	Kind   CandidateKind
	Detail string // type hint
}

// ExtractCompletionContext analyzes the tree-sitter CST at the cursor position
// to determine what kind of completion is appropriate.
func ExtractCompletionContext(root *TSNode, source []byte, cursorRow, cursorCol int) CompletionContext {
	if root == nil {
		return CompletionContext{Kind: CompletionNone}
	}

	// Try at cursor position and one position back (cursor is often one past the last typed char)
	for _, col := range []int{cursorCol, cursorCol - 1} {
		if col < 0 {
			continue
		}

		node := root.NodeAtPosition(cursorRow, col)
		if node == nil {
			continue
		}

		// Case 1: cursor is on a property_identifier inside a member_expression
		if node.Kind == "property_identifier" {
			parent := findParentOfKind(root, node, "member_expression")
			if parent != nil {
				ctx := completionFromMemberExpression(parent, source, node.Text)
				ctx.CursorRow = cursorRow
				ctx.CursorCol = cursorCol
				return ctx
			}
		}

		// Case 2: cursor is inside an ERROR node — look for trailing dot pattern
		errNode := findContainingError(root, cursorRow, col)
		if errNode != nil {
			ctx := completionFromErrorNode(errNode, source)
			if ctx.Kind != CompletionNone {
				ctx.CursorRow = cursorRow
				ctx.CursorCol = cursorCol
				return ctx
			}
		}

		// Case 3: cursor is on a bare identifier
		if node.Kind == "identifier" {
			return CompletionContext{
				Kind:        CompletionIdentifier,
				PartialText: node.Text,
				CursorRow:   cursorRow,
				CursorCol:   cursorCol,
			}
		}
	}

	return CompletionContext{Kind: CompletionNone}
}

// completionFromMemberExpression extracts base expression from member_expression.
func completionFromMemberExpression(memExpr *TSNode, source []byte, partial string) CompletionContext {
	// member_expression has children: object, ".", property_identifier
	// We want the object (first non-dot, non-property child)
	for _, child := range memExpr.Children {
		if child.Kind != "." && child.Kind != "property_identifier" {
			baseText := extractNodeText(child, source)
			return CompletionContext{
				Kind:         CompletionProperty,
				BaseExpr:     baseText,
				BaseNodeKind: child.Kind,
				PartialText:  partial,
			}
		}
	}
	return CompletionContext{Kind: CompletionNone}
}

// completionFromErrorNode looks for identifier + "." pattern inside an ERROR node.
func completionFromErrorNode(errNode *TSNode, source []byte) CompletionContext {
	children := errNode.Children
	if len(children) < 2 {
		return CompletionContext{Kind: CompletionNone}
	}

	// Find the last "." child
	for i := len(children) - 1; i >= 1; i-- {
		if children[i].Kind == "." || children[i].Text == "." {
			// The child before the dot is the base expression
			base := children[i-1]
			baseText := extractNodeText(base, source)
			return CompletionContext{
				Kind:         CompletionProperty,
				BaseExpr:     baseText,
				BaseNodeKind: base.Kind,
				PartialText:  "",
			}
		}
	}

	return CompletionContext{Kind: CompletionNone}
}

// findParentOfKind walks up from a target node to find a parent of given kind.
// Since TSNode doesn't store parent pointers, we search from root.
func findParentOfKind(root *TSNode, target *TSNode, kind string) *TSNode {
	return findParentImpl(root, target, kind)
}

func findParentImpl(current *TSNode, target *TSNode, kind string) *TSNode {
	for _, child := range current.Children {
		if child == target {
			if current.Kind == kind {
				return current
			}
			return nil
		}
		if found := findParentImpl(child, target, kind); found != nil {
			return found
		}
	}
	return nil
}

// findContainingError finds the ERROR node containing the given position.
func findContainingError(root *TSNode, row, col int) *TSNode {
	if root.IsError && nodeContains(root, row, col) {
		return root
	}
	for _, child := range root.Children {
		if found := findContainingError(child, row, col); found != nil {
			return found
		}
	}
	return nil
}

// extractNodeText extracts the source text for a node using its byte range.
func extractNodeText(n *TSNode, source []byte) string {
	if n.Text != "" {
		return n.Text
	}
	// For non-leaf nodes, reconstruct from source using position
	// We approximate by using row/col → byte offset
	// This is simplified — for accurate results we'd need byte offsets on TSNode
	// For now, collect leaf texts
	var parts []string
	collectLeafTexts(n, &parts)
	return strings.Join(parts, "")
}

func collectLeafTexts(n *TSNode, parts *[]string) {
	if n == nil {
		return
	}
	if len(n.Children) == 0 && n.Text != "" {
		*parts = append(*parts, n.Text)
		return
	}
	for _, child := range n.Children {
		collectLeafTexts(child, parts)
	}
}

// --- Candidate resolution ---

// builtinPrototypes maps known global objects/types to their method names.
var builtinPrototypes = map[string][]CompletionCandidate{
	"Object": {
		{Label: "hasOwnProperty", Kind: CandidateMethod, Detail: "method"},
		{Label: "toString", Kind: CandidateMethod, Detail: "method"},
		{Label: "valueOf", Kind: CandidateMethod, Detail: "method"},
		{Label: "keys", Kind: CandidateMethod, Detail: "static method"},
		{Label: "values", Kind: CandidateMethod, Detail: "static method"},
		{Label: "entries", Kind: CandidateMethod, Detail: "static method"},
		{Label: "assign", Kind: CandidateMethod, Detail: "static method"},
		{Label: "freeze", Kind: CandidateMethod, Detail: "static method"},
		{Label: "fromEntries", Kind: CandidateMethod, Detail: "static method"},
	},
	"Array": {
		{Label: "push", Kind: CandidateMethod, Detail: "method"},
		{Label: "pop", Kind: CandidateMethod, Detail: "method"},
		{Label: "shift", Kind: CandidateMethod, Detail: "method"},
		{Label: "unshift", Kind: CandidateMethod, Detail: "method"},
		{Label: "map", Kind: CandidateMethod, Detail: "method"},
		{Label: "filter", Kind: CandidateMethod, Detail: "method"},
		{Label: "reduce", Kind: CandidateMethod, Detail: "method"},
		{Label: "find", Kind: CandidateMethod, Detail: "method"},
		{Label: "forEach", Kind: CandidateMethod, Detail: "method"},
		{Label: "indexOf", Kind: CandidateMethod, Detail: "method"},
		{Label: "includes", Kind: CandidateMethod, Detail: "method"},
		{Label: "slice", Kind: CandidateMethod, Detail: "method"},
		{Label: "splice", Kind: CandidateMethod, Detail: "method"},
		{Label: "concat", Kind: CandidateMethod, Detail: "method"},
		{Label: "join", Kind: CandidateMethod, Detail: "method"},
		{Label: "sort", Kind: CandidateMethod, Detail: "method"},
		{Label: "reverse", Kind: CandidateMethod, Detail: "method"},
		{Label: "length", Kind: CandidateProperty, Detail: "number"},
	},
	"String": {
		{Label: "charAt", Kind: CandidateMethod, Detail: "method"},
		{Label: "concat", Kind: CandidateMethod, Detail: "method"},
		{Label: "includes", Kind: CandidateMethod, Detail: "method"},
		{Label: "indexOf", Kind: CandidateMethod, Detail: "method"},
		{Label: "match", Kind: CandidateMethod, Detail: "method"},
		{Label: "replace", Kind: CandidateMethod, Detail: "method"},
		{Label: "slice", Kind: CandidateMethod, Detail: "method"},
		{Label: "split", Kind: CandidateMethod, Detail: "method"},
		{Label: "startsWith", Kind: CandidateMethod, Detail: "method"},
		{Label: "endsWith", Kind: CandidateMethod, Detail: "method"},
		{Label: "toLowerCase", Kind: CandidateMethod, Detail: "method"},
		{Label: "toUpperCase", Kind: CandidateMethod, Detail: "method"},
		{Label: "trim", Kind: CandidateMethod, Detail: "method"},
		{Label: "length", Kind: CandidateProperty, Detail: "number"},
	},
	"console": {
		{Label: "log", Kind: CandidateMethod, Detail: "method"},
		{Label: "error", Kind: CandidateMethod, Detail: "method"},
		{Label: "warn", Kind: CandidateMethod, Detail: "method"},
		{Label: "info", Kind: CandidateMethod, Detail: "method"},
		{Label: "debug", Kind: CandidateMethod, Detail: "method"},
		{Label: "table", Kind: CandidateMethod, Detail: "method"},
	},
	"Math": {
		{Label: "abs", Kind: CandidateMethod, Detail: "method"},
		{Label: "ceil", Kind: CandidateMethod, Detail: "method"},
		{Label: "floor", Kind: CandidateMethod, Detail: "method"},
		{Label: "round", Kind: CandidateMethod, Detail: "method"},
		{Label: "max", Kind: CandidateMethod, Detail: "method"},
		{Label: "min", Kind: CandidateMethod, Detail: "method"},
		{Label: "random", Kind: CandidateMethod, Detail: "method"},
		{Label: "sqrt", Kind: CandidateMethod, Detail: "method"},
		{Label: "PI", Kind: CandidateProperty, Detail: "number"},
		{Label: "E", Kind: CandidateProperty, Detail: "number"},
	},
	"JSON": {
		{Label: "parse", Kind: CandidateMethod, Detail: "method"},
		{Label: "stringify", Kind: CandidateMethod, Detail: "method"},
	},
}

// ResolveCandidates generates completion candidates for a given context.
// drawerRoot is optional — if provided, drawer-local bindings are included.
func ResolveCandidates(ctx CompletionContext, gojaIndex *Index, drawerRoot ...*TSNode) []CompletionCandidate {
	if ctx.Kind == CompletionNone {
		return nil
	}

	var candidates []CompletionCandidate

	switch ctx.Kind {
	case CompletionProperty:
		candidates = resolvePropertyCandidates(ctx, gojaIndex)
	case CompletionIdentifier:
		candidates = resolveIdentifierCandidates(ctx, gojaIndex)
	case CompletionArgument, CompletionNone:
		return nil
	}

	// Add drawer-local bindings if available
	if len(drawerRoot) > 0 && drawerRoot[0] != nil {
		drawerBindings := ExtractDrawerBindings(drawerRoot[0])
		candidates = append(candidates, drawerBindings...)
	}

	// Filter by partial text
	if ctx.PartialText != "" {
		prefix := strings.ToLower(ctx.PartialText)
		var filtered []CompletionCandidate
		for _, c := range candidates {
			if strings.HasPrefix(strings.ToLower(c.Label), prefix) {
				filtered = append(filtered, c)
			}
		}
		candidates = filtered
	}

	return candidates
}

func resolvePropertyCandidates(ctx CompletionContext, gojaIndex *Index) []CompletionCandidate {
	base := ctx.BaseExpr

	// Check built-in globals first
	if candidates, ok := builtinPrototypes[base]; ok {
		return candidates
	}

	var candidates []CompletionCandidate

	// Look up the binding in goja's scope resolver
	if gojaIndex != nil && gojaIndex.Resolution != nil {
		globalScope := gojaIndex.Resolution.Scopes[gojaIndex.Resolution.RootScopeID]
		if globalScope != nil {
			if binding, ok := globalScope.Bindings[base]; ok {
				// Find properties from the declaration's initializer
				props := extractObjectPropertiesFromBinding(gojaIndex, binding)
				for _, p := range props {
					candidates = append(candidates, CompletionCandidate{
						Label: p, Kind: CandidateProperty, Detail: "property",
					})
				}
			}
		}
	}

	// Always add basic Object.prototype methods
	candidates = append(candidates,
		CompletionCandidate{Label: "hasOwnProperty", Kind: CandidateMethod, Detail: "method"},
		CompletionCandidate{Label: "toString", Kind: CandidateMethod, Detail: "method"},
		CompletionCandidate{Label: "valueOf", Kind: CandidateMethod, Detail: "method"},
	)

	return candidates
}

func resolveIdentifierCandidates(ctx CompletionContext, gojaIndex *Index) []CompletionCandidate {
	var candidates []CompletionCandidate

	// Add all bindings from file's global scope
	if gojaIndex != nil && gojaIndex.Resolution != nil {
		globalScope := gojaIndex.Resolution.Scopes[gojaIndex.Resolution.RootScopeID]
		if globalScope != nil {
			for name, binding := range globalScope.Bindings {
				kind := CandidateVariable
				if binding.Kind == BindingFunction {
					kind = CandidateFunction
				}
				candidates = append(candidates, CompletionCandidate{
					Label: name, Kind: kind, Detail: binding.Kind.String(),
				})
			}
		}
	}

	// Add well-known globals
	for name := range builtinPrototypes {
		candidates = append(candidates, CompletionCandidate{
			Label: name, Kind: CandidateVariable, Detail: "global",
		})
	}

	return candidates
}

// ExtractDrawerBindings scans a tree-sitter CST for declarations and returns
// candidate names. This provides completion for variables defined in the drawer itself.
func ExtractDrawerBindings(root *TSNode) []CompletionCandidate {
	if root == nil {
		return nil
	}
	var candidates []CompletionCandidate
	seen := make(map[string]bool)
	extractBindingsRecursive(root, &candidates, seen)
	return candidates
}

func extractBindingsRecursive(n *TSNode, out *[]CompletionCandidate, seen map[string]bool) {
	if n == nil {
		return
	}

	// Look for variable_declarator children (inside lexical_declaration or variable_declaration)
	if n.Kind == "variable_declarator" {
		// First child is usually the identifier name
		for _, child := range n.Children {
			if child.Kind == "identifier" && child.Text != "" && !seen[child.Text] {
				seen[child.Text] = true
				*out = append(*out, CompletionCandidate{
					Label:  child.Text,
					Kind:   CandidateVariable,
					Detail: "drawer local",
				})
			}
		}
	}

	// Look for function_declaration
	if n.Kind == "function_declaration" {
		for _, child := range n.Children {
			if child.Kind == "identifier" && child.Text != "" && !seen[child.Text] {
				seen[child.Text] = true
				*out = append(*out, CompletionCandidate{
					Label:  child.Text,
					Kind:   CandidateFunction,
					Detail: "drawer local",
				})
			}
		}
	}

	for _, child := range n.Children {
		extractBindingsRecursive(child, out, seen)
	}
}

// extractObjectPropertiesFromBinding extracts property names from the initializer
// of a binding's declaration.
func extractObjectPropertiesFromBinding(gojaIndex *Index, binding *BindingRecord) []string {
	if binding == nil || binding.DeclNodeID < 0 {
		return nil
	}

	declNode := gojaIndex.Nodes[binding.DeclNodeID]
	if declNode == nil {
		return nil
	}

	// Walk up to find the Binding parent
	parent := gojaIndex.Nodes[declNode.ParentID]
	if parent == nil {
		return nil
	}

	// Look for ObjectLiteral or ArrayLiteral among siblings
	var props []string
	for _, childID := range parent.ChildIDs {
		child := gojaIndex.Nodes[childID]
		if child == nil {
			continue
		}
		if child.Kind == "ObjectLiteral" {
			props = append(props, extractPropertyNamesFromObjLit(gojaIndex, childID)...)
		}
		if child.Kind == "ArrayLiteral" {
			// Return array methods
			for _, c := range builtinPrototypes["Array"] {
				props = append(props, c.Label)
			}
			return props
		}
	}
	return props
}

func extractPropertyNamesFromObjLit(gojaIndex *Index, objID NodeID) []string {
	objNode := gojaIndex.Nodes[objID]
	if objNode == nil {
		return nil
	}
	var names []string
	for _, childID := range objNode.ChildIDs {
		child := gojaIndex.Nodes[childID]
		if child == nil {
			continue
		}
		// PropertyKeyed and PropertyShort have identifier/string children as keys
		for _, propChildID := range child.ChildIDs {
			propChild := gojaIndex.Nodes[propChildID]
			if propChild == nil {
				continue
			}
			if propChild.Kind == "Identifier" || propChild.Kind == "StringLiteral" {
				name := strings.Trim(propChild.Label, "\"")
				if name != "" {
					names = append(names, name)
				}
			}
		}
	}
	return names
}
