package runtime

import (
	"testing"

	"github.com/dop251/goja"
)

func TestSessionLoadAndEval(t *testing.T) {
	s := NewSession()

	err := s.Load(`
		class Dog {
			constructor(name) {
				this.name = name;
			}
			bark() { return "woof"; }
		}
	`)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	val, err := s.Eval(`new Dog("Rex").bark()`)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}
	if val.String() != "woof" {
		t.Errorf("expected 'woof', got %q", val.String())
	}
}

func TestEvalWithCapture(t *testing.T) {
	s := NewSession()
	_ = s.Load(`function boom() { throw new TypeError("oops"); }`)

	result := s.EvalWithCapture(`boom()`)
	if result.Error == nil {
		t.Fatal("expected error from boom()")
	}
	if result.ErrorStack == "" {
		t.Error("expected non-empty error stack")
	}
}

func TestValuePreview(t *testing.T) {
	s := NewSession()
	vm := s.VM

	tests := []struct {
		expr     string
		contains string
	}{
		{`"hello"`, `"hello"`},
		{`42`, `42`},
		{`true`, `true`},
		{`null`, `null`},
		{`undefined`, `undefined`},
		{`({a:1, b:2})`, `{a, b}`},
	}

	for _, tt := range tests {
		val, err := vm.RunString(tt.expr)
		if err != nil {
			t.Fatalf("RunString(%q) error: %v", tt.expr, err)
		}
		preview := ValuePreview(val, vm, 50)
		if preview == "" {
			t.Errorf("ValuePreview(%q) returned empty string", tt.expr)
		}
	}
}

func TestInspectObject(t *testing.T) {
	s := NewSession()
	vm := s.VM

	_ = s.Load(`var obj = {x: 1, y: "hello", z: function() {}};`)
	val := vm.Get("obj")
	obj := val.ToObject(vm)

	props := InspectObject(obj, vm)
	if len(props) == 0 {
		t.Fatal("expected properties from object")
	}

	found := make(map[string]string)
	for _, p := range props {
		found[p.Name] = p.Kind
	}

	if found["x"] != "number" {
		t.Errorf("expected x to be 'number', got %q", found["x"])
	}
	if found["y"] != "string" {
		t.Errorf("expected y to be 'string', got %q", found["y"])
	}
	if found["z"] != "function" {
		t.Errorf("expected z to be 'function', got %q", found["z"])
	}
}

func TestWalkPrototypeChain(t *testing.T) {
	s := NewSession()
	vm := s.VM

	_ = s.Load(`
		class Animal { eat() {} }
		class Dog extends Animal { bark() {} }
		var d = new Dog();
	`)

	val := vm.Get("d")
	obj := val.ToObject(vm)

	chain := WalkPrototypeChain(obj, vm)
	if len(chain) < 2 {
		t.Fatalf("expected at least 2 prototype levels, got %d", len(chain))
	}

	// First prototype should be Dog.prototype
	if chain[0].Name != "Dog" {
		t.Errorf("expected first proto to be Dog, got %q", chain[0].Name)
	}
}

func TestGetDescriptor(t *testing.T) {
	s := NewSession()
	vm := s.VM

	_ = s.Load(`var obj = {x: 42};`)
	val := vm.Get("obj")
	obj := val.ToObject(vm)

	desc, err := GetDescriptor(obj, vm, "x")
	if err != nil {
		t.Fatalf("GetDescriptor failed: %v", err)
	}
	if desc == nil {
		t.Fatal("expected non-nil descriptor")
	}
	if !desc.Writable {
		t.Error("expected x to be writable")
	}
	if !desc.Enumerable {
		t.Error("expected x to be enumerable")
	}
}

func TestParseException(t *testing.T) {
	s := NewSession()

	_ = s.Load(`
		function inner() { throw new TypeError("bad value"); }
		function outer() { inner(); }
	`)

	result := s.EvalWithCapture(`outer()`)
	if result.Error == nil {
		t.Fatal("expected error")
	}

	ex, ok := result.Error.(*goja.Exception)
	if !ok {
		t.Fatalf("expected *goja.Exception, got %T", result.Error)
	}

	info := ParseException(ex)
	if info.Message == "" {
		t.Error("expected non-empty error message")
	}
	if len(info.Frames) == 0 {
		t.Error("expected at least one stack frame")
	}

	// First frame should be "inner"
	if len(info.Frames) > 0 && info.Frames[0].FunctionName != "inner" {
		t.Errorf("expected first frame to be 'inner', got %q", info.Frames[0].FunctionName)
	}
}
