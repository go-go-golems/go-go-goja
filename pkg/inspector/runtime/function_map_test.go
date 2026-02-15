package runtime

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

func TestMapFunctionToSourceDisambiguatesSameMethodNameAcrossClasses(t *testing.T) {
	source := `
class A {
  foo() { return "A"; }
}
class B {
  foo() { return "B"; }
}
const a = new A();
const b = new B();
`

	analysis := jsparse.Analyze("test.js", source, nil)
	if analysis == nil || analysis.Program == nil || analysis.Index == nil {
		t.Fatal("expected analysis with program and index")
	}

	s := NewSession()
	if err := s.Load(source); err != nil {
		t.Fatalf("runtime load failed: %v", err)
	}

	bVal := s.VM.Get("b")
	if bVal == nil {
		t.Fatal("expected runtime value for b")
	}
	bObj := bVal.ToObject(s.VM)
	fooVal := bObj.Get("foo")
	if fooVal == nil {
		t.Fatal("expected b.foo runtime value")
	}

	mapping := MapFunctionToSource(fooVal, s.VM, analysis)
	if mapping == nil {
		t.Fatal("expected non-nil function source mapping")
	}
	if mapping.ClassName != "B" {
		t.Fatalf("expected class B mapping, got class %q (line %d)", mapping.ClassName, mapping.StartLine)
	}
}
