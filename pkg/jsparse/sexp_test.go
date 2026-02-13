package jsparse

import (
	"strings"
	"testing"

	"github.com/dop251/goja/parser"
)

func TestCSTToSExprEscapesLeafText(t *testing.T) {
	root := &TSNode{
		Kind: "program",
		Children: []*TSNode{
			{
				Kind: "identifier",
				Text: "a\"b\nc",
			},
		},
	}

	out := CSTToSExpr(root, nil)
	if !strings.Contains(out, "\"a\\\"b\\nc\"") {
		t.Fatalf("expected escaped leaf text in output, got: %s", out)
	}
}

func TestCSTToSExprIncludesFlags(t *testing.T) {
	root := &TSNode{
		Kind:    "ERROR",
		IsError: true,
		Children: []*TSNode{
			{
				Kind:      "identifier",
				Text:      "x",
				IsMissing: true,
			},
		},
	}

	out := CSTToSExpr(root, nil)
	if !strings.Contains(out, ":error true") {
		t.Fatalf("expected :error flag in output, got: %s", out)
	}
	if !strings.Contains(out, ":missing true") {
		t.Fatalf("expected :missing flag in output, got: %s", out)
	}
}

func TestASTIndexToSExprRendersProgram(t *testing.T) {
	src := `const x = 1;`
	program, err := parser.ParseFile(nil, "test.js", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	idx := BuildIndex(program, src)
	out := ASTIndexToSExpr(idx, &SExprOptions{IncludeSpan: true, IncludeText: true})

	if !strings.Contains(out, "(Program") {
		t.Fatalf("expected Program root in output, got: %s", out)
	}
	if !strings.Contains(out, "(Identifier") {
		t.Fatalf("expected Identifier node in output, got: %s", out)
	}
	if !strings.Contains(out, ":span (") {
		t.Fatalf("expected span metadata in output, got: %s", out)
	}
}

func TestCSTToSExprTruncatesByDepth(t *testing.T) {
	root := &TSNode{
		Kind: "program",
		Children: []*TSNode{
			{
				Kind: "stmt",
				Children: []*TSNode{
					{Kind: "identifier", Text: "x"},
				},
			},
		},
	}

	out := CSTToSExpr(root, &SExprOptions{MaxDepth: 1})
	if !strings.Contains(out, "(...)") {
		t.Fatalf("expected truncation marker in output, got: %s", out)
	}
}

func TestASTIndexToSExprTruncatesByNodeCount(t *testing.T) {
	src := `const a = 1; const b = 2;`
	program, err := parser.ParseFile(nil, "test.js", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	idx := BuildIndex(program, src)
	out := ASTIndexToSExpr(idx, &SExprOptions{MaxNodes: 1})

	if !strings.Contains(out, "(...)") {
		t.Fatalf("expected node-count truncation marker in output, got: %s", out)
	}
}

func TestASTToSExprHandlesNilProgram(t *testing.T) {
	out := ASTToSExpr(nil, "", nil)
	if out != "" {
		t.Fatalf("expected empty output for nil program, got: %q", out)
	}
}
