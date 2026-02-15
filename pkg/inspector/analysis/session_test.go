package analysis

import (
	"testing"
)

const testSource = `
class Animal {
  constructor(name) {
    this.name = name;
  }
  eat(food) {
    return food;
  }
}

class Dog extends Animal {
  bark() {
    const sound = "woof";
    return sound;
  }
}

function greet(name) {
  return "Hello, " + name;
}

const version = 3;
`

func TestGlobalBindings(t *testing.T) {
	s := NewSession("test.js", testSource)
	if s.Result == nil {
		t.Fatal("expected non-nil result")
	}
	if s.Result.ParseErr != nil {
		t.Fatalf("unexpected parse error: %v", s.Result.ParseErr)
	}

	bindings := s.GlobalBindings()
	if bindings == nil {
		t.Fatal("expected non-nil bindings")
	}

	expectedNames := []string{"Animal", "Dog", "greet", "version"}
	for _, name := range expectedNames {
		if _, ok := bindings[name]; !ok {
			t.Errorf("expected binding %q not found", name)
		}
	}
}

func TestMethodSymbols(t *testing.T) {
	s := NewSession("test.js", testSource)
	if s.Result == nil || s.Result.Resolution == nil || s.Result.Index == nil {
		t.Fatal("expected analysis to succeed")
	}

	// Find the bark method scope range
	// bark is in the Dog class, it contains "const sound"
	symbols := MethodSymbols(s.Result.Resolution, s.Result.Index, 1, len(testSource)+1)

	// Should find at least "sound" and "name" and "food" as local bindings
	found := make(map[string]bool)
	for _, sym := range symbols {
		found[sym.Name] = true
	}

	if !found["sound"] {
		t.Error("expected to find 'sound' in method symbols")
	}
	if !found["name"] {
		t.Error("expected to find 'name' (parameter) in method symbols")
	}
}
