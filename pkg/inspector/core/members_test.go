package core

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dop251/goja/ast"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

func TestBuildClassMembersSelfCycle(t *testing.T) {
	src := `
class A extends A {
  foo(x) { return x; }
}
`
	program := mustProgram(t, src)
	members := BuildClassMembers(program, "A")
	if len(members) == 0 {
		t.Fatal("expected members for class A")
	}
	fooCount := 0
	for _, m := range members {
		if m.Name == "foo" {
			fooCount++
		}
	}
	if fooCount != 1 {
		t.Fatalf("expected foo exactly once, got %d", fooCount)
	}
}

func TestBuildClassMembersIndirectCycle(t *testing.T) {
	src := `
class A extends B {
  a() {}
}
class B extends A {
  b() {}
}
`
	program := mustProgram(t, src)
	members := BuildClassMembers(program, "A")
	if len(members) == 0 {
		t.Fatal("expected members for class A")
	}
	var sawA, sawB bool
	for _, m := range members {
		if m.Name == "a" {
			sawA = true
		}
		if m.Name == "b" {
			sawB = true
		}
	}
	if !sawA || !sawB {
		t.Fatalf("expected both a and b members, got %+v", members)
	}
}

func TestBuildFunctionMembers(t *testing.T) {
	src := `
function greet(name, times) {
  return name + times;
}
`
	program := mustProgram(t, src)
	members := BuildFunctionMembers(program, "greet")
	if len(members) != 2 {
		t.Fatalf("expected 2 params, got %d", len(members))
	}
	if members[0].Name != "name" || members[1].Name != "times" {
		t.Fatalf("unexpected params: %+v", members)
	}
}

func TestClassExtends(t *testing.T) {
	src := `
class Base {}
class Child extends Base {}
`
	program := mustProgram(t, src)
	got := ClassExtends(program, "Child")
	if got != "Base" {
		t.Fatalf("expected Base, got %q", got)
	}
}

func TestBuildClassMembersDepthGuard(t *testing.T) {
	var b strings.Builder
	const classCount = 180
	for i := 0; i < classCount; i++ {
		name := fmt.Sprintf("C%d", i)
		if i == 0 {
			fmt.Fprintf(&b, "class %s { m%d() {} }\n", name, i)
			continue
		}
		parent := fmt.Sprintf("C%d", i-1)
		fmt.Fprintf(&b, "class %s extends %s { m%d() {} }\n", name, parent, i)
	}

	program := mustProgram(t, b.String())
	last := fmt.Sprintf("C%d", classCount-1)
	members := BuildClassMembers(program, last)
	if len(members) == 0 {
		t.Fatal("expected members for deep class chain")
	}
	if len(members) > maxInheritanceDepth+1 {
		t.Fatalf("expected depth-guarded member count <= %d, got %d", maxInheritanceDepth+1, len(members))
	}
}

func mustProgram(t *testing.T, source string) *ast.Program {
	t.Helper()
	a := jsparse.Analyze("test.js", source, nil)
	if a.ParseErr != nil {
		t.Fatalf("parse error: %v", a.ParseErr)
	}
	return a.Program
}
