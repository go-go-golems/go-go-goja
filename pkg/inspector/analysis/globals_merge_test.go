package analysis

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

func TestMergeGlobals(t *testing.T) {
	existing := []GlobalBinding{
		{Name: "A", Kind: jsparse.BindingClass},
		{Name: "f", Kind: jsparse.BindingFunction},
	}
	runtimeKinds := map[string]jsparse.BindingKind{
		"x": jsparse.BindingVar,
		"f": jsparse.BindingFunction, // duplicate
	}
	declared := []DeclaredBinding{
		{Name: "K", Kind: jsparse.BindingClass},
		{Name: "x", Kind: jsparse.BindingLet}, // duplicate (already runtime)
		{Name: "missing", Kind: jsparse.BindingConst},
	}
	hasValue := func(name string) bool {
		return name != "missing"
	}

	got := MergeGlobals(existing, runtimeKinds, declared, hasValue)
	if len(got) != 4 {
		t.Fatalf("expected 4 globals, got=%d (%+v)", len(got), got)
	}
	seen := map[string]jsparse.BindingKind{}
	for _, g := range got {
		seen[g.Name] = g.Kind
	}
	if seen["A"] != jsparse.BindingClass {
		t.Fatalf("missing class A")
	}
	if seen["f"] != jsparse.BindingFunction {
		t.Fatalf("missing function f")
	}
	if seen["x"] != jsparse.BindingVar {
		t.Fatalf("runtime kind for x should be var")
	}
	if seen["K"] != jsparse.BindingClass {
		t.Fatalf("declared class K should be present")
	}
	if _, ok := seen["missing"]; ok {
		t.Fatalf("missing should not be added when runtime value is absent")
	}
}
