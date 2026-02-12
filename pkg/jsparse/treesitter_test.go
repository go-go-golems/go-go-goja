package jsparse

import (
	"testing"
)

func TestTSParserParse(t *testing.T) {
	p, err := NewTSParser()
	if err != nil {
		t.Fatalf("NewTSParser: %v", err)
	}
	defer p.Close()

	root := p.Parse([]byte("const x = 1;"))
	if root == nil {
		t.Fatal("Parse returned nil")
	}
	if root.Kind != "program" {
		t.Errorf("expected root kind 'program', got %q", root.Kind)
	}
	if root.HasError() {
		t.Error("expected no error for valid source")
	}
	if root.ChildCount() != 1 {
		t.Errorf("expected 1 child, got %d", root.ChildCount())
	}

	child := root.Children[0]
	if child.Kind != "lexical_declaration" {
		t.Errorf("expected 'lexical_declaration', got %q", child.Kind)
	}
}

func TestTSParserIncremental(t *testing.T) {
	p, err := NewTSParser()
	if err != nil {
		t.Fatalf("NewTSParser: %v", err)
	}
	defer p.Close()

	// First parse
	root1 := p.Parse([]byte("const a = 1;\n"))
	if root1 == nil {
		t.Fatal("first parse returned nil")
	}
	if root1.HasError() {
		t.Error("first parse should not have error")
	}

	// Incremental: add a second line
	root2 := p.Parse([]byte("const a = 1;\na.f"))
	if root2 == nil {
		t.Fatal("second parse returned nil")
	}
	if root2.HasError() {
		t.Error("'a.f' should parse without error")
	}
	if root2.ChildCount() != 2 {
		t.Errorf("expected 2 children, got %d", root2.ChildCount())
	}

	// The second statement should be an expression_statement with member_expression
	stmt2 := root2.Children[1]
	if stmt2.Kind != "expression_statement" {
		t.Errorf("expected expression_statement, got %q", stmt2.Kind)
	}

	t.Logf("Incremental parse: %d children, second=%s", root2.ChildCount(), stmt2.Kind)
}

func TestTSParserErrorRecovery(t *testing.T) {
	p, err := NewTSParser()
	if err != nil {
		t.Fatalf("NewTSParser: %v", err)
	}
	defer p.Close()

	root := p.Parse([]byte("const a = 1;\na."))
	if root == nil {
		t.Fatal("parse returned nil")
	}
	if !root.HasError() {
		t.Error("expected error for 'a.'")
	}

	// Should have 2 children: valid lexical_declaration + ERROR
	if root.ChildCount() < 2 {
		t.Fatalf("expected at least 2 children, got %d", root.ChildCount())
	}

	// First child should be valid
	if root.Children[0].Kind != "lexical_declaration" {
		t.Errorf("first child should be lexical_declaration, got %q", root.Children[0].Kind)
	}

	// Second child should be ERROR containing identifier and dot
	errNode := root.Children[1]
	if !errNode.IsError {
		t.Errorf("second child should be ERROR, got %q (isError=%v)", errNode.Kind, errNode.IsError)
	}

	// ERROR should have children: identifier "a" and "."
	foundIdent := false
	foundDot := false
	for _, child := range errNode.Children {
		if child.Kind == "identifier" && child.Text == "a" {
			foundIdent = true
		}
		if child.Kind == "." && child.Text == "." {
			foundDot = true
		}
	}
	if !foundIdent {
		t.Error("ERROR node should contain identifier 'a'")
	}
	if !foundDot {
		t.Error("ERROR node should contain '.'")
	}

	t.Logf("Error recovery: ERROR has %d children, foundIdent=%v foundDot=%v",
		errNode.ChildCount(), foundIdent, foundDot)
}

func TestTSNodeSnapshot(t *testing.T) {
	p, err := NewTSParser()
	if err != nil {
		t.Fatalf("NewTSParser: %v", err)
	}
	defer p.Close()

	root := p.Parse([]byte("const x = 1;"))
	if root == nil {
		t.Fatal("parse returned nil")
	}

	// Force a reparse — the old snapshot should still be valid
	root2 := p.Parse([]byte("let y = 2;"))
	if root2 == nil {
		t.Fatal("second parse returned nil")
	}

	// Old snapshot should still have its data intact
	if root.Kind != "program" {
		t.Errorf("old snapshot corrupted: kind=%q", root.Kind)
	}
	if root.ChildCount() != 1 {
		t.Errorf("old snapshot corrupted: children=%d", root.ChildCount())
	}
	if root.Children[0].Kind != "lexical_declaration" {
		t.Errorf("old snapshot child corrupted: kind=%q", root.Children[0].Kind)
	}

	// New snapshot should reflect new code
	if root2.Children[0].Kind != "lexical_declaration" {
		t.Errorf("new snapshot first child: %q", root2.Children[0].Kind)
	}
}

func TestTSNodeAtPosition(t *testing.T) {
	p, err := NewTSParser()
	if err != nil {
		t.Fatalf("NewTSParser: %v", err)
	}
	defer p.Close()

	root := p.Parse([]byte("const obj = {foo: 1};\nobj.foo"))
	if root == nil {
		t.Fatal("parse returned nil")
	}

	// Position 1:4 should be inside "foo" in "obj.foo" (property_identifier)
	node := root.NodeAtPosition(1, 5)
	if node == nil {
		t.Fatal("NodeAtPosition returned nil")
	}
	if node.Kind != "property_identifier" {
		t.Errorf("expected property_identifier at 1:5, got %q", node.Kind)
	}
	if node.Text != "foo" {
		t.Errorf("expected text 'foo', got %q", node.Text)
	}

	// Position 1:1 should be inside "obj" (identifier)
	node2 := root.NodeAtPosition(1, 1)
	if node2 == nil {
		t.Fatal("NodeAtPosition at 1:1 returned nil")
	}
	if node2.Kind != "identifier" {
		t.Errorf("expected identifier at 1:1, got %q", node2.Kind)
	}

	t.Logf("NodeAtPosition(1,5)=%s %q, NodeAtPosition(1,1)=%s %q",
		node.Kind, node.Text, node2.Kind, node2.Text)
}
