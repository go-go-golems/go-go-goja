package runtime

import (
	"testing"

	"github.com/dop251/goja"
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

func TestMapFunctionToSourceTopLevelFunction(t *testing.T) {
	source := `
function greet(name) {
  return "hi " + name;
}
class A {
  greet(name) {
    return "A " + name;
  }
}
`

	analysis := jsparse.Analyze("test.js", source, nil)
	if analysis == nil || analysis.Program == nil || analysis.Index == nil {
		t.Fatal("expected analysis with program and index")
	}

	s := NewSession()
	if err := s.Load(source); err != nil {
		t.Fatalf("runtime load failed: %v", err)
	}

	greetVal := s.VM.Get("greet")
	if greetVal == nil {
		t.Fatal("expected runtime value for greet")
	}

	mapping := MapFunctionToSource(greetVal, s.VM, analysis)
	if mapping == nil {
		t.Fatal("expected non-nil function source mapping")
	}
	if mapping.Name != "greet" {
		t.Fatalf("expected mapping for greet, got %q", mapping.Name)
	}
	if mapping.ClassName != "" {
		t.Fatalf("expected top-level function mapping with empty class name, got %q", mapping.ClassName)
	}
}

func TestMapFunctionToSourceReturnsNilForNonFunction(t *testing.T) {
	source := `const answer = 42;`
	analysis := jsparse.Analyze("test.js", source, nil)
	if analysis == nil || analysis.Program == nil || analysis.Index == nil {
		t.Fatal("expected analysis with program and index")
	}

	vm := goja.New()
	val := vm.ToValue(42)
	if mapping := MapFunctionToSource(val, vm, analysis); mapping != nil {
		t.Fatalf("expected nil mapping for non-function, got %+v", mapping)
	}
}
