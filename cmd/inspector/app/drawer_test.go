package app

import (
	"testing"

	"github.com/dop251/goja/parser"
)

func TestDrawerCompletionFlow(t *testing.T) {
	// Build a goja index from sample JS
	src := `function add(a, b) {
  const sum = a + b;
  return sum;
}

const result = add(2, 3);
console.log(result);
`
	program, err := parser.ParseFile(nil, "test.js", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	idx := BuildIndex(program, src)
	idx.Resolution = Resolve(program, idx)

	d := NewDrawer(idx)
	defer d.Close()

	// Type "console."
	for _, ch := range "console." {
		d.InsertChar(ch)
	}
	d.Reparse()

	t.Logf("After typing: lines=%v row=%d col=%d", d.lines, d.cursorRow, d.cursorCol)
	t.Logf("tsRoot=%v hasError=%v", d.tsRoot != nil, d.tsRoot != nil && d.tsRoot.HasError())

	ctx := ExtractCompletionContext(d.tsRoot, d.Source(), d.cursorRow, d.cursorCol)
	t.Logf("Context: kind=%d base=%q partial=%q", ctx.Kind, ctx.BaseExpr, ctx.PartialText)

	if ctx.Kind != CompletionProperty {
		t.Fatalf("expected CompletionProperty, got %d", ctx.Kind)
	}

	candidates := ResolveCandidates(ctx, idx)
	t.Logf("Candidates: %d", len(candidates))
	for _, c := range candidates {
		t.Logf("  %s %s (%s)", c.Kind.Icon(), c.Label, c.Detail)
	}

	if len(candidates) < 3 {
		t.Errorf("expected at least 3 candidates, got %d", len(candidates))
	}
}

func TestDrawerGoToDefinition(t *testing.T) {
	src := `const greeting = "hello";
console.log(greeting);
`
	program, err := parser.ParseFile(nil, "test.js", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	idx := BuildIndex(program, src)
	idx.Resolution = Resolve(program, idx)

	d := NewDrawer(idx)
	defer d.Close()

	// Type "greeting" in drawer
	for _, ch := range "greeting" {
		d.InsertChar(ch)
	}
	d.Reparse()

	// Find the identifier at cursor
	node := d.tsRoot.NodeAtPosition(0, 4) // middle of "greeting"
	if node == nil {
		t.Fatal("no node at cursor")
	}
	if node.Kind != "identifier" {
		t.Fatalf("expected identifier, got %q", node.Kind)
	}
	if node.Text != "greeting" {
		t.Fatalf("expected 'greeting', got %q", node.Text)
	}

	// Look up in global scope
	globalScope := idx.Resolution.Scopes[idx.Resolution.RootScopeID]
	binding, ok := globalScope.Bindings["greeting"]
	if !ok {
		t.Fatal("'greeting' not found in global scope")
	}

	declNode := idx.Nodes[binding.DeclNodeID]
	t.Logf("go-to-def: 'greeting' → decl at %d:%d (%s)", declNode.StartLine, declNode.StartCol, declNode.Kind)
}
