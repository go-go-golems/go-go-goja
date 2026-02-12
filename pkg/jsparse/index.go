package jsparse

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/file"
)

// Index holds all NodeRecords and provides lookup operations.
type Index struct {
	Nodes          map[NodeID]*NodeRecord
	RootID         NodeID
	OrderedByStart []NodeID // sorted by (Start asc, End desc) for containment lookup
	Resolution     *Resolution
	nextID         NodeID
	src            string
	fileObj        *file.File
}

// NewIndex creates a new empty index.
func NewIndex() *Index {
	return &Index{
		Nodes: make(map[NodeID]*NodeRecord),
	}
}

// BuildIndex parses an AST program and builds a complete node index.
func BuildIndex(program *ast.Program, src string) *Index {
	idx := NewIndex()
	idx.src = src

	// Build a file.File for position lookups
	fs := &file.FileSet{}
	fs.AddFile("source.js", src)
	idx.fileObj = file.NewFile("source.js", src, 1)

	idx.RootID = idx.walkNode(program, -1, 0)
	idx.buildOrderedByStart()
	return idx
}

// walkNode recursively walks an AST node and registers it in the index.
func (idx *Index) walkNode(n ast.Node, parentID NodeID, depth int) NodeID {
	if isNilNode(n) {
		return -1
	}

	// Guard against panics from malformed AST nodes
	var start, end int
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		start = int(n.Idx0())
		end = int(n.Idx1())
	}()
	if panicked || (start == 0 && end == 0) {
		return -1
	}
	// Some parser nodes (e.g. IfStatement) may have Idx0()==0 as a parser artifact;
	// clamp start to 1 (the minimum valid 1-based offset)
	if start < 1 {
		start = 1
	}
	if end < start {
		end = start
	}

	id := idx.nextID
	idx.nextID++

	kind := nodeKind(n)
	label := nodeLabel(n)
	snippet := idx.excerpt(start, end)

	// Compute line/col from offset
	startLine, startCol := idx.offsetToLineCol(start)
	endLine, endCol := idx.offsetToLineCol(end)

	rec := &NodeRecord{
		ID:        id,
		Kind:      kind,
		Start:     start,
		End:       end,
		StartLine: startLine,
		StartCol:  startCol,
		EndLine:   endLine,
		EndCol:    endCol,
		Label:     label,
		Snippet:   snippet,
		ParentID:  parentID,
		Depth:     depth,
		Expanded:  depth < 2, // auto-expand first two levels
	}

	idx.Nodes[id] = rec

	// Walk children
	children := childNodes(n)
	for _, child := range children {
		childID := idx.walkNode(child, id, depth+1)
		if childID >= 0 {
			rec.ChildIDs = append(rec.ChildIDs, childID)
		}
	}

	return id
}

// buildOrderedByStart creates the sorted node list for containment lookup.
func (idx *Index) buildOrderedByStart() {
	idx.OrderedByStart = make([]NodeID, 0, len(idx.Nodes))
	for id := range idx.Nodes {
		idx.OrderedByStart = append(idx.OrderedByStart, id)
	}
	sort.Slice(idx.OrderedByStart, func(i, j int) bool {
		ni := idx.Nodes[idx.OrderedByStart[i]]
		nj := idx.Nodes[idx.OrderedByStart[j]]
		if ni.Start == nj.Start {
			return ni.End > nj.End // wider first
		}
		return ni.Start < nj.Start
	})
}

// NodeAtOffset returns the smallest (most specific) node containing the given offset.
func (idx *Index) NodeAtOffset(offset int) *NodeRecord {
	var best *NodeRecord
	for _, id := range idx.OrderedByStart {
		n := idx.Nodes[id]
		if n.Start <= offset && offset < n.End {
			if best == nil || n.Span() < best.Span() ||
				(n.Span() == best.Span() && n.Depth > best.Depth) {
				best = n
			}
		}
	}
	return best
}

// AncestorPath returns the path from root to the given node (inclusive).
func (idx *Index) AncestorPath(id NodeID) []NodeID {
	var path []NodeID
	for id >= 0 {
		path = append(path, id)
		n := idx.Nodes[id]
		if n == nil {
			break
		}
		id = n.ParentID
	}
	// Reverse to get root-first
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// VisibleNodes returns the flat list of nodes visible in the tree (respecting expand/collapse).
func (idx *Index) VisibleNodes() []NodeID {
	var result []NodeID
	idx.collectVisible(idx.RootID, &result)
	return result
}

func (idx *Index) collectVisible(id NodeID, result *[]NodeID) {
	if id < 0 {
		return
	}
	n := idx.Nodes[id]
	if n == nil {
		return
	}
	*result = append(*result, id)
	if n.Expanded {
		for _, childID := range n.ChildIDs {
			idx.collectVisible(childID, result)
		}
	}
}

// ToggleExpand toggles the expanded state of a node.
func (idx *Index) ToggleExpand(id NodeID) {
	n := idx.Nodes[id]
	if n != nil && n.HasChildren() {
		n.Expanded = !n.Expanded
	}
}

// ExpandTo ensures all ancestors of a node are expanded so it becomes visible.
func (idx *Index) ExpandTo(id NodeID) {
	path := idx.AncestorPath(id)
	for _, pid := range path {
		n := idx.Nodes[pid]
		if n != nil && n.HasChildren() {
			n.Expanded = true
		}
	}
}

// offsetToLineCol converts a 1-based byte offset to line/col (both 1-based).
func (idx *Index) offsetToLineCol(offset int) (int, int) {
	if offset <= 0 {
		return 1, 1
	}
	// offset is 1-based (file.Idx convention)
	pos := offset - 1 // convert to 0-based index into src
	if pos > len(idx.src) {
		pos = len(idx.src)
	}

	line := 1
	lineStart := 0
	for i := 0; i < pos; i++ {
		if idx.src[i] == '\n' {
			line++
			lineStart = i + 1
		}
	}
	col := pos - lineStart + 1
	return line, col
}

// LineColToOffset converts 1-based line/col to 1-based byte offset.
func (idx *Index) LineColToOffset(line, col int) int {
	if line < 1 {
		line = 1
	}
	currentLine := 1
	for i := 0; i < len(idx.src); i++ {
		if currentLine == line {
			offset := i + col // 1-based offset
			if offset > len(idx.src)+1 {
				offset = len(idx.src) + 1
			}
			return offset
		}
		if idx.src[i] == '\n' {
			currentLine++
		}
	}
	return len(idx.src) + 1
}

// excerpt returns a short source snippet for display.
func (idx *Index) excerpt(start, end int) string {
	if start < 1 {
		start = 1
	}
	if end < start {
		end = start
	}
	s := start - 1
	e := end - 1
	if s > len(idx.src) {
		s = len(idx.src)
	}
	if e > len(idx.src) {
		e = len(idx.src)
	}
	chunk := idx.src[s:e]
	chunk = strings.ReplaceAll(chunk, "\n", "\\n")
	chunk = strings.TrimSpace(chunk)
	if len(chunk) > 40 {
		chunk = chunk[:40] + "..."
	}
	return chunk
}

// nodeKind returns a clean type name for an AST node.
func nodeKind(n ast.Node) string {
	t := reflect.TypeOf(n)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// nodeLabel returns a short additional label for identifiable nodes.
func nodeLabel(n ast.Node) string {
	switch v := n.(type) {
	case *ast.Identifier:
		return fmt.Sprintf("%q", string(v.Name))
	case *ast.StringLiteral:
		return fmt.Sprintf("%q", v.Literal)
	case *ast.NumberLiteral:
		return v.Literal
	case *ast.BooleanLiteral:
		return v.Literal
	case *ast.NullLiteral:
		return "null"
	case *ast.RegExpLiteral:
		return v.Literal
	case *ast.FunctionLiteral:
		if v.Name != nil {
			return fmt.Sprintf("%q", string(v.Name.Name))
		}
	case *ast.FunctionDeclaration:
		if v.Function != nil && v.Function.Name != nil {
			return fmt.Sprintf("%q", string(v.Function.Name.Name))
		}
	case *ast.ClassLiteral:
		if v.Name != nil {
			return fmt.Sprintf("%q", string(v.Name.Name))
		}
	case *ast.ClassDeclaration:
		if v.Class != nil && v.Class.Name != nil {
			return fmt.Sprintf("%q", string(v.Class.Name.Name))
		}
	case *ast.LexicalDeclaration:
		return v.Token.String()
	case *ast.BinaryExpression:
		return v.Operator.String()
	case *ast.AssignExpression:
		return v.Operator.String()
	case *ast.UnaryExpression:
		return v.Operator.String()
	}
	return ""
}

// childNodes extracts all child ast.Node values from a node using reflection.
func childNodes(n ast.Node) []ast.Node {
	v := reflect.ValueOf(n)
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	var out []ast.Node
	var visitValue func(fv reflect.Value)
	visitValue = func(fv reflect.Value) {
		if !fv.IsValid() {
			return
		}
		if fv.Kind() == reflect.Interface {
			if fv.IsNil() {
				return
			}
			fv = fv.Elem()
		}
		if fv.Kind() == reflect.Ptr && fv.IsNil() {
			return
		}

		if fv.CanInterface() {
			if node, ok := fv.Interface().(ast.Node); ok && !isNilNode(node) {
				out = append(out, node)
				return
			}
		}

		// Struct value whose pointer receiver implements ast.Node
		// (e.g. DotExpression.Identifier is ast.Identifier by value, not *ast.Identifier)
		if fv.Kind() == reflect.Struct && fv.CanAddr() {
			pv := fv.Addr()
			if pv.CanInterface() {
				if node, ok := pv.Interface().(ast.Node); ok && !isNilNode(node) {
					out = append(out, node)
					return
				}
			}
		}

		//exhaustive:ignore
		switch fv.Kind() {
		case reflect.Ptr:
			if fv.IsNil() {
				return
			}
			if fv.CanInterface() {
				if node, ok := fv.Interface().(ast.Node); ok && !isNilNode(node) {
					out = append(out, node)
					return
				}
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < fv.Len(); i++ {
				visitValue(fv.Index(i))
			}
		default:
			// Non-node scalar/container kinds are intentionally ignored.
		}
	}

	for i := 0; i < v.NumField(); i++ {
		visitValue(v.Field(i))
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Idx0() == out[j].Idx0() {
			return out[i].Idx1() < out[j].Idx1()
		}
		return out[i].Idx0() < out[j].Idx0()
	})

	return dedup(out)
}

// isNilNode checks if an ast.Node interface value holds a nil pointer.
func isNilNode(n ast.Node) bool {
	if n == nil {
		return true
	}
	v := reflect.ValueOf(n)
	return v.Kind() == reflect.Ptr && v.IsNil()
}

func dedup(nodes []ast.Node) []ast.Node {
	seen := map[uintptr]bool{}
	out := make([]ast.Node, 0, len(nodes))
	for _, n := range nodes {
		ptr := pointerKey(n)
		if ptr != 0 {
			if seen[ptr] {
				continue
			}
			seen[ptr] = true
		}
		out = append(out, n)
	}
	return out
}

func pointerKey(n ast.Node) uintptr {
	v := reflect.ValueOf(n)
	if !v.IsValid() {
		return 0
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return 0
		}
		return v.Pointer()
	}
	return 0
}
