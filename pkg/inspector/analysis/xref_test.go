package analysis

import (
	"testing"
)

func TestCrossReferences(t *testing.T) {
	src := `
function greet(name) {
  return "Hello, " + name;
}

function main() {
  greet("world");
  greet("test");
}
`
	s := NewSession("test.js", src)
	if s.Result == nil || s.Result.Resolution == nil {
		t.Fatal("expected analysis to succeed")
	}

	refs := CrossReferences(s.Result.Resolution, s.Result.Index, "greet")
	if len(refs) == 0 {
		t.Fatal("expected cross-references for 'greet'")
	}

	// Should have at least 3 usages: declaration + 2 calls
	if len(refs) < 3 {
		t.Errorf("expected at least 3 xrefs for 'greet', got %d", len(refs))
	}

	// Each entry should have a valid line number
	for _, ref := range refs {
		if ref.Line <= 0 {
			t.Errorf("expected positive line number, got %d", ref.Line)
		}
	}
}

func TestCrossReferencesNotFound(t *testing.T) {
	src := `const x = 1;`
	s := NewSession("test.js", src)
	refs := CrossReferences(s.Result.Resolution, s.Result.Index, "nonexistent")
	if len(refs) != 0 {
		t.Errorf("expected 0 xrefs for nonexistent binding, got %d", len(refs))
	}
}
