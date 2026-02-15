package analysis

import "testing"

func TestGlobalsSortedAndExtends(t *testing.T) {
	s := NewSession("test.js", testSource)
	globals := s.Globals()
	if len(globals) != 4 {
		t.Fatalf("expected 4 globals, got %d", len(globals))
	}

	got := []string{globals[0].Name, globals[1].Name, globals[2].Name, globals[3].Name}
	want := []string{"Animal", "Dog", "greet", "version"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("globals[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	if globals[1].Extends != "Animal" {
		t.Fatalf("Dog extends = %q, want %q", globals[1].Extends, "Animal")
	}
}

func TestBindingDeclLine(t *testing.T) {
	s := NewSession("test.js", testSource)

	line, ok := s.BindingDeclLine("greet")
	if !ok || line <= 0 {
		t.Fatalf("expected declaration line for greet, got (%d,%v)", line, ok)
	}

	if _, ok := s.BindingDeclLine("missing"); ok {
		t.Fatal("expected missing binding lookup to fail")
	}
}

func TestMemberDeclLine(t *testing.T) {
	s := NewSession("test.js", testSource)

	line, ok := s.MemberDeclLine("Dog", "", "bark")
	if !ok || line <= 0 {
		t.Fatalf("expected declaration line for Dog.bark, got (%d,%v)", line, ok)
	}

	inhLine, ok := s.MemberDeclLine("Dog", "Animal", "eat")
	if !ok || inhLine <= 0 {
		t.Fatalf("expected declaration line for inherited Animal.eat, got (%d,%v)", inhLine, ok)
	}

	if _, ok := s.MemberDeclLine("Dog", "", "missing"); ok {
		t.Fatal("expected missing member lookup to fail")
	}
}

func TestSessionMembersAccessors(t *testing.T) {
	s := NewSession("test.js", testSource)

	classMembers := s.ClassMembers("Dog")
	if len(classMembers) == 0 {
		t.Fatal("expected class members for Dog")
	}

	var sawBark bool
	for _, m := range classMembers {
		if m.Name == "bark" {
			sawBark = true
			break
		}
	}
	if !sawBark {
		t.Fatalf("expected bark in class members, got %+v", classMembers)
	}

	funcMembers := s.FunctionMembers("greet")
	if len(funcMembers) != 1 || funcMembers[0].Name != "name" {
		t.Fatalf("unexpected function members: %+v", funcMembers)
	}
}

func TestParseErrorAccessor(t *testing.T) {
	s := NewSession("broken.js", "function () {")
	if s.ParseError() == nil {
		t.Fatal("expected parse error")
	}
}
