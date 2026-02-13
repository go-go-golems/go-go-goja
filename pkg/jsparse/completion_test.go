package jsparse

import (
	"testing"

	"github.com/dop251/goja/parser"
)

func TestExtractContextPropertyDot(t *testing.T) {
	p, err := NewTSParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	src := []byte("console.")
	root := p.Parse(src)
	ctx := ExtractCompletionContext(root, src, 0, 8)
	if ctx.Kind != CompletionProperty {
		t.Fatalf("expected CompletionProperty, got %d", ctx.Kind)
	}
	if ctx.BaseExpr != "console" {
		t.Errorf("expected base 'console', got %q", ctx.BaseExpr)
	}

	candidates := ResolveCandidates(ctx, nil)
	t.Logf("Candidates for console.: %d", len(candidates))
	for _, c := range candidates {
		t.Logf("  %s %s (%s)", c.Kind.Icon(), c.Label, c.Detail)
	}

	// Should have console built-in methods
	foundLog := false
	for _, c := range candidates {
		if c.Label == "log" {
			foundLog = true
		}
	}
	if !foundLog {
		t.Error("expected 'log' in console. candidates")
	}
}

func TestExtractContextPropertyPartial(t *testing.T) {
	p, err := NewTSParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	src := []byte("console.lo")
	root := p.Parse(src)
	ctx := ExtractCompletionContext(root, src, 0, 10)
	if ctx.Kind != CompletionProperty {
		t.Fatalf("expected CompletionProperty, got %d", ctx.Kind)
	}
	if ctx.PartialText != "lo" {
		t.Errorf("expected partial 'lo', got %q", ctx.PartialText)
	}

	candidates := ResolveCandidates(ctx, nil)
	t.Logf("Filtered for 'lo': %d candidates", len(candidates))
	if len(candidates) != 1 || candidates[0].Label != "log" {
		for _, c := range candidates {
			t.Logf("  %s", c.Label)
		}
		t.Errorf("expected exactly [log]")
	}
}

func TestExtractContextIdentifier(t *testing.T) {
	p, err := NewTSParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	src := []byte("con")
	root := p.Parse(src)
	ctx := ExtractCompletionContext(root, src, 0, 3)
	if ctx.Kind != CompletionIdentifier {
		t.Fatalf("expected CompletionIdentifier, got %d", ctx.Kind)
	}
	if ctx.PartialText != "con" {
		t.Errorf("expected partial 'con', got %q", ctx.PartialText)
	}
}

func TestExtractContextChainDot(t *testing.T) {
	p, err := NewTSParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	src := []byte("[1,2,3].filter(x => x > 1).")
	root := p.Parse(src)
	ctx := ExtractCompletionContext(root, src, 0, 27)
	if ctx.Kind != CompletionProperty {
		t.Fatalf("expected CompletionProperty, got %d", ctx.Kind)
	}
	t.Logf("Chain context: base=%q kind=%q", ctx.BaseExpr, ctx.BaseNodeKind)
}

func TestResolveCandidatesWithIndex(t *testing.T) {
	src := `const obj = {foo: 1, bar: 2};`
	program, err := parser.ParseFile(nil, "test.js", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	idx := BuildIndex(program, src)
	idx.Resolution = Resolve(program, idx)

	ctx := CompletionContext{
		Kind:     CompletionProperty,
		BaseExpr: "obj",
	}

	candidates := ResolveCandidates(ctx, idx)
	t.Logf("Candidates for obj.: %d", len(candidates))
	for _, c := range candidates {
		t.Logf("  %s %s (%s)", c.Kind.Icon(), c.Label, c.Detail)
	}

	found := map[string]bool{}
	for _, c := range candidates {
		found[c.Label] = true
	}
	if !found["foo"] {
		t.Error("expected 'foo' in candidates")
	}
	if !found["bar"] {
		t.Error("expected 'bar' in candidates")
	}
}

func TestDrawerLocalBindings(t *testing.T) {
	p, err := NewTSParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	src := []byte("const x = 1;\nlet y = 2;\nfunction foo() {}\n")
	root := p.Parse(src)
	bindings := ExtractDrawerBindings(root)
	t.Logf("Drawer bindings: %d", len(bindings))
	for _, b := range bindings {
		t.Logf("  %s %s (%s)", b.Kind.Icon(), b.Label, b.Detail)
	}

	found := map[string]bool{}
	for _, b := range bindings {
		found[b.Label] = true
	}
	if !found["x"] {
		t.Error("expected 'x'")
	}
	if !found["y"] {
		t.Error("expected 'y'")
	}
	if !found["foo"] {
		t.Error("expected 'foo'")
	}
}

func TestIdentifierCompletionWithDrawerLocals(t *testing.T) {
	p, err := NewTSParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	src := []byte("const myVar = 42;\nmy")
	root := p.Parse(src)
	ctx := ExtractCompletionContext(root, src, 1, 2)
	if ctx.Kind != CompletionIdentifier {
		t.Fatalf("expected CompletionIdentifier, got %d", ctx.Kind)
	}

	candidates := ResolveCandidates(ctx, nil, root)
	t.Logf("Candidates for 'my': %d", len(candidates))
	foundMyVar := false
	for _, c := range candidates {
		t.Logf("  %s %s (%s)", c.Kind.Icon(), c.Label, c.Detail)
		if c.Label == "myVar" {
			foundMyVar = true
		}
	}
	if !foundMyVar {
		t.Error("expected 'myVar' from drawer-local bindings")
	}
}

func TestExtractContextNone(t *testing.T) {
	p, err := NewTSParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	src := []byte("42")
	root := p.Parse(src)
	ctx := ExtractCompletionContext(root, src, 0, 2)
	if ctx.Kind != CompletionNone {
		t.Errorf("expected CompletionNone for '42', got %d", ctx.Kind)
	}
}
