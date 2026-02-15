package app

import (
	"testing"

	inspectoranalysis "github.com/go-go-golems/go-go-goja/pkg/inspector/analysis"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

func TestBuildMembersSelfExtendsNoPanic(t *testing.T) {
	src := `
class A extends A {
  foo(x) { return x; }
}
`
	m := modelFromSource(t, src)
	selectGlobalByName(t, &m, "A")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("buildMembers panicked: %v", r)
		}
	}()

	m.buildMembers()
	if len(m.members) == 0 {
		t.Fatal("expected class members for A")
	}
}

func TestBuildMembersIndirectCycleNoPanic(t *testing.T) {
	src := `
class A extends B { a() {} }
class B extends A { b() {} }
`
	m := modelFromSource(t, src)
	selectGlobalByName(t, &m, "A")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("buildMembers panicked: %v", r)
		}
	}()

	m.buildMembers()
	if len(m.members) == 0 {
		t.Fatal("expected class members for A")
	}
}

func modelFromSource(t *testing.T, source string) Model {
	t.Helper()
	m := NewModel("")
	a := jsparse.Analyze("test.js", source, nil)
	if a.ParseErr != nil {
		t.Fatalf("parse error: %v", a.ParseErr)
	}
	m.analysis = a
	m.session = inspectoranalysis.NewSessionFromResult(a)
	m.buildGlobals()
	return m
}

func selectGlobalByName(t *testing.T, m *Model, name string) {
	t.Helper()
	for i, g := range m.globals {
		if g.Name == name {
			m.globalIdx = i
			return
		}
	}
	t.Fatalf("global %q not found in %+v", name, m.globals)
}
