package jsparse

import (
	"testing"

	"github.com/dop251/goja/parser"
)

const sampleJS = `function add(a, b) {
  const sum = a + b;
  return sum;
}

const result = add(2, 3);
console.log(result);
`

func TestBuildIndex(t *testing.T) {
	program, err := parser.ParseFile(nil, "test.js", sampleJS, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	idx := BuildIndex(program, sampleJS)
	if idx == nil {
		t.Fatal("index is nil")
	}

	// Should have a root node
	root := idx.Nodes[idx.RootID]
	if root == nil {
		t.Fatal("root node is nil")
	}
	if root.Kind != "Program" {
		t.Errorf("expected root kind 'Program', got %q", root.Kind)
	}

	// Should have multiple nodes
	if len(idx.Nodes) < 5 {
		t.Errorf("expected at least 5 nodes, got %d", len(idx.Nodes))
	}

	// Root should have children
	if len(root.ChildIDs) == 0 {
		t.Error("root should have children")
	}

	t.Logf("Index has %d nodes, root has %d children", len(idx.Nodes), len(root.ChildIDs))
}

func TestNodeAtOffset(t *testing.T) {
	program, err := parser.ParseFile(nil, "test.js", sampleJS, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	idx := BuildIndex(program, sampleJS)

	tests := []struct {
		name     string
		offset   int
		wantKind string
		wantNil  bool
	}{
		{"beginning of function keyword", 1, "FunctionLiteral", false}, // FunctionDeclaration wraps FunctionLiteral which starts at same offset
		{"at identifier add", 10, "Identifier", false},
		{"inside block body", 25, "LexicalDeclaration", false},        // 'const' keyword starts the LexicalDeclaration
		{"inside const result line", 65, "LexicalDeclaration", false}, // outer const decl
		{"past end of source", 200, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := idx.NodeAtOffset(tt.offset)
			if tt.wantNil {
				if node != nil {
					t.Errorf("expected nil, got %s", node.Kind)
				}
				return
			}
			if node == nil {
				t.Fatal("expected non-nil node")
			}
			if tt.wantKind != "" && node.Kind != tt.wantKind {
				t.Errorf("expected kind %q, got %q (label=%q start=%d end=%d)", tt.wantKind, node.Kind, node.Label, node.Start, node.End)
			}
		})
	}
}

func TestOffsetToLineCol(t *testing.T) {
	program, err := parser.ParseFile(nil, "test.js", sampleJS, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	idx := BuildIndex(program, sampleJS)

	tests := []struct {
		offset   int
		wantLine int
		wantCol  int
	}{
		{1, 1, 1},   // first character
		{21, 1, 21}, // end of first line
		{22, 2, 1},  // beginning of second line
	}

	for _, tt := range tests {
		line, col := idx.offsetToLineCol(tt.offset)
		if line != tt.wantLine || col != tt.wantCol {
			t.Errorf("offsetToLineCol(%d) = (%d, %d), want (%d, %d)", tt.offset, line, col, tt.wantLine, tt.wantCol)
		}
	}
}

func TestAncestorPath(t *testing.T) {
	program, err := parser.ParseFile(nil, "test.js", sampleJS, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	idx := BuildIndex(program, sampleJS)

	// Find a deep node (Identifier "add" at offset 10)
	node := idx.NodeAtOffset(10)
	if node == nil {
		t.Fatal("expected node at offset 10")
	}

	path := idx.AncestorPath(node.ID)
	if len(path) < 2 {
		t.Errorf("expected ancestor path length >= 2, got %d", len(path))
	}

	// First in path should be root
	if path[0] != idx.RootID {
		t.Errorf("first ancestor should be root")
	}

	// Last should be the node itself
	if path[len(path)-1] != node.ID {
		t.Errorf("last ancestor should be the node itself")
	}
}

func TestVisibleNodes(t *testing.T) {
	program, err := parser.ParseFile(nil, "test.js", sampleJS, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	idx := BuildIndex(program, sampleJS)

	visible := idx.VisibleNodes()
	if len(visible) == 0 {
		t.Error("expected some visible nodes")
	}

	// All visible nodes should exist in the index
	for _, id := range visible {
		if _, ok := idx.Nodes[id]; !ok {
			t.Errorf("visible node %d not found in index", id)
		}
	}

	t.Logf("Visible nodes: %d out of %d total", len(visible), len(idx.Nodes))
}

func TestExpandCollapse(t *testing.T) {
	program, err := parser.ParseFile(nil, "test.js", sampleJS, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	idx := BuildIndex(program, sampleJS)

	// Get initial visible count
	initialVisible := len(idx.VisibleNodes())

	// Collapse root
	root := idx.Nodes[idx.RootID]
	if root.Expanded {
		idx.ToggleExpand(idx.RootID)
		collapsed := len(idx.VisibleNodes())
		if collapsed >= initialVisible {
			t.Errorf("collapsing root should reduce visible nodes: collapsed=%d, initial=%d", collapsed, initialVisible)
		}
		if collapsed != 1 {
			t.Errorf("collapsing root should leave 1 visible, got %d", collapsed)
		}

		// Expand again
		idx.ToggleExpand(idx.RootID)
		expanded := len(idx.VisibleNodes())
		if expanded != initialVisible {
			t.Errorf("re-expanding root should restore visible count: got %d, want %d", expanded, initialVisible)
		}
	}
}

func TestBuildIndexComplexFile(t *testing.T) {
	complexJS := `class Animal {
  constructor(name) {
    this.name = name;
  }
  speak() {
    return this.name + " makes a noise.";
  }
}

class Dog extends Animal {
  constructor(name) {
    super(name);
  }
  speak() {
    return this.name + " barks.";
  }
}

const d = new Dog("Rex");
console.log(d.speak());

const numbers = [1, 2, 3, 4, 5];
const doubled = numbers.map(n => n * 2);
const sum = doubled.reduce((acc, val) => acc + val, 0);

if (sum > 10) {
  for (let i = 0; i < numbers.length; i++) {
    console.log(numbers[i]);
  }
} else {
  console.log("Sum is small:", sum);
}

try {
  const result = JSON.parse('{"key": "value"}');
  console.log(result.key);
} catch (e) {
  console.log("Parse error:", e.message);
} finally {
  console.log("Done");
}
`

	program, err := parser.ParseFile(nil, "complex.js", complexJS, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	idx := BuildIndex(program, complexJS)
	if idx == nil {
		t.Fatal("index is nil")
	}

	root := idx.Nodes[idx.RootID]
	if root == nil {
		t.Fatal("root is nil")
	}
	if root.Kind != "Program" {
		t.Errorf("expected root kind 'Program', got %q", root.Kind)
	}

	// Should have many nodes (classes, methods, arrow functions, try/catch)
	if len(idx.Nodes) < 50 {
		t.Errorf("expected at least 50 nodes for complex file, got %d", len(idx.Nodes))
	}

	// All nodes should have valid spans
	for id, node := range idx.Nodes {
		if node.Start < 1 {
			t.Errorf("node %d (%s) has invalid start %d", id, node.Kind, node.Start)
		}
		if node.End < node.Start {
			t.Errorf("node %d (%s) has end %d < start %d", id, node.Kind, node.End, node.Start)
		}
	}

	// NodeAtOffset should work throughout the file
	for offset := 1; offset <= len(complexJS); offset++ {
		node := idx.NodeAtOffset(offset)
		if node == nil {
			// Some offsets (whitespace between top-level statements) may not match any node
			continue
		}
		if node.Start > offset || offset >= node.End {
			t.Errorf("NodeAtOffset(%d) returned node %s [%d..%d] that doesn't contain offset",
				offset, node.Kind, node.Start, node.End)
		}
	}

	t.Logf("Complex file: %d nodes, root has %d children", len(idx.Nodes), len(root.ChildIDs))
}

func TestBuildIndexWithParseError(t *testing.T) {
	invalidJS := `function add(a, b) {
  const sum = a + ;
  return sum;
}
`
	program, parseErr := parser.ParseFile(nil, "invalid.js", invalidJS, 0)
	if parseErr == nil {
		t.Fatal("expected parse error")
	}

	// Parser should still return a partial AST
	if program == nil {
		t.Skip("parser returned nil program on error")
	}

	idx := BuildIndex(program, invalidJS)
	if idx == nil {
		t.Fatal("index is nil")
	}

	if len(idx.Nodes) < 3 {
		t.Errorf("expected at least 3 nodes in partial AST, got %d", len(idx.Nodes))
	}

	// Should contain a BadExpression node
	foundBad := false
	for _, node := range idx.Nodes {
		if node.Kind == "BadExpression" {
			foundBad = true
			break
		}
	}
	if foundBad {
		t.Log("Found BadExpression in partial AST (expected for invalid source)")
	}

	t.Logf("Partial AST: %d nodes", len(idx.Nodes))
}

func TestSyncRoundTrip(t *testing.T) {
	program, err := parser.ParseFile(nil, "test.js", sampleJS, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	idx := BuildIndex(program, sampleJS)

	// For each node, verify that NodeAtOffset(node.Start) returns the node itself
	// or a more specific child (which is also correct)
	for id, node := range idx.Nodes {
		found := idx.NodeAtOffset(node.Start)
		if found == nil {
			t.Errorf("NodeAtOffset(%d) returned nil for node %d (%s)", node.Start, id, node.Kind)
			continue
		}
		// The found node should either be this node or a descendant
		if found.Start < node.Start || found.End > node.End {
			t.Errorf("NodeAtOffset(%d) returned %s [%d..%d] which is not contained in %s [%d..%d]",
				node.Start, found.Kind, found.Start, found.End, node.Kind, node.Start, node.End)
		}
	}
}

func TestLineColToOffset(t *testing.T) {
	program, err := parser.ParseFile(nil, "test.js", sampleJS, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	idx := BuildIndex(program, sampleJS)

	// Round-trip: offset -> line/col -> offset
	for _, testOffset := range []int{1, 10, 22, 50} {
		line, col := idx.offsetToLineCol(testOffset)
		got := idx.LineColToOffset(line, col)
		if got != testOffset {
			t.Errorf("round-trip failed: offset %d -> (%d,%d) -> %d", testOffset, line, col, got)
		}
	}
}
