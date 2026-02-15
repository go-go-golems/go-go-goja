package analysis

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

func TestDeclaredBindingsFromSource(t *testing.T) {
	src := `
const c = 1;
let l = 2;
var v = 3;
function f(x) { return x; }
class K {}
`
	bindings := DeclaredBindingsFromSource(src)
	got := map[string]jsparse.BindingKind{}
	for _, b := range bindings {
		got[b.Name] = b.Kind
	}

	tests := map[string]jsparse.BindingKind{
		"c": jsparse.BindingConst,
		"l": jsparse.BindingLet,
		"v": jsparse.BindingVar,
		"f": jsparse.BindingFunction,
		"K": jsparse.BindingClass,
	}
	for name, want := range tests {
		kind, ok := got[name]
		if !ok {
			t.Fatalf("missing binding %q", name)
		}
		if kind != want {
			t.Fatalf("binding %q: got=%v want=%v", name, kind, want)
		}
	}
}

func TestDeclaredBindingsFromSourceInvalid(t *testing.T) {
	bindings := DeclaredBindingsFromSource("const")
	if len(bindings) != 0 {
		t.Fatalf("expected no bindings for invalid snippet, got=%d", len(bindings))
	}
}
