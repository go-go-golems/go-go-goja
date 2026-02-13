package jsparse

import "testing"

func TestAnalyzeBuildsResult(t *testing.T) {
	src := `const answer = 42;`
	res := Analyze("test.js", src, nil)
	if res == nil {
		t.Fatal("expected non-nil analysis result")
	}
	if res.Program == nil {
		t.Fatal("expected parsed program")
	}
	if res.Index == nil {
		t.Fatal("expected index")
	}
	if res.Resolution == nil {
		t.Fatal("expected resolution")
	}
	node := res.NodeAtOffset(1)
	if node == nil {
		t.Fatal("expected node at offset 1")
	}
}

func TestAnalyzeDiagnosticsOnParseError(t *testing.T) {
	src := `function add(a, b) { return a + ; }`
	res := Analyze("invalid.js", src, nil)
	if res == nil {
		t.Fatal("expected non-nil analysis result")
	}
	diags := res.Diagnostics()
	if len(diags) == 0 {
		t.Fatal("expected at least one diagnostic for parse error")
	}
	if diags[0].Severity != "error" {
		t.Fatalf("expected error severity, got %q", diags[0].Severity)
	}
}

func TestAnalyzeCompleteAt(t *testing.T) {
	src := `const obj = { foo: 1, bar: 2 };
obj.`

	res := Analyze("test.js", src, nil)
	if res == nil {
		t.Fatal("expected non-nil analysis result")
	}

	tsParser, err := NewTSParser()
	if err != nil {
		t.Fatalf("new ts parser: %v", err)
	}
	defer tsParser.Close()

	root := tsParser.Parse([]byte(src))
	if root == nil {
		t.Fatal("expected CST root")
	}

	row := 1
	col := len("obj.")
	ctx := res.CompletionContextAt(root, row, col)
	if ctx.Kind != CompletionProperty {
		t.Fatalf("expected CompletionProperty, got %v", ctx.Kind)
	}

	candidates := res.CompleteAt(root, row, col)
	if len(candidates) == 0 {
		t.Fatal("expected completion candidates")
	}

	foundFoo := false
	for _, c := range candidates {
		if c.Label == "foo" {
			foundFoo = true
			break
		}
	}
	if !foundFoo {
		t.Fatalf("expected to find property candidate 'foo' (got %d candidates)", len(candidates))
	}
}
