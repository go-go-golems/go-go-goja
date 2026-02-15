package app

import (
	"testing"

	inspectoranalysis "github.com/go-go-golems/go-go-goja/pkg/inspector/analysis"
	inspectorruntime "github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
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

func TestBuildMembersValueRuntimeDerived(t *testing.T) {
	src := `
const cfg = {
  answer: 42,
  ping() { return "pong"; }
}
`
	m := modelFromSource(t, src)
	m.rtSession = inspectorruntime.NewSession()
	if err := m.rtSession.Load(src); err != nil {
		t.Fatalf("runtime load error: %v", err)
	}
	selectGlobalByName(t, &m, "cfg")

	m.buildMembers()
	if len(m.members) == 0 {
		t.Fatal("expected runtime-derived members for cfg")
	}

	var sawAnswer bool
	for _, member := range m.members {
		if !member.RuntimeDerived {
			t.Fatalf("expected runtime-derived member, got %+v", member)
		}
		if member.Name == "answer" {
			sawAnswer = true
		}
	}
	if !sawAnswer {
		t.Fatalf("expected answer member in %+v", m.members)
	}
}

func TestJumpToBindingAndMemberWithSession(t *testing.T) {
	src := `
class Foo {
  bar(x) { return x; }
}
`
	m := modelFromSource(t, src)
	selectGlobalByName(t, &m, "Foo")

	m.jumpToBinding("Foo")
	if m.sourceTarget < 0 {
		t.Fatalf("expected sourceTarget for binding jump, got %d", m.sourceTarget)
	}

	m.buildMembers()
	if len(m.members) == 0 {
		t.Fatal("expected class members for Foo")
	}

	var bar MemberItem
	var found bool
	for _, member := range m.members {
		if member.Name == "bar" {
			bar = member
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected bar member in %+v", m.members)
	}

	m.jumpToMember("Foo", bar)
	if m.sourceTarget < 0 {
		t.Fatalf("expected sourceTarget for member jump, got %d", m.sourceTarget)
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
