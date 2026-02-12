package jsparse

import (
	"testing"

	"github.com/dop251/goja/parser"
)

func buildTestResolution(t *testing.T, src string) (*Index, *Resolution) {
	t.Helper()
	program, err := parser.ParseFile(nil, "test.js", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	idx := BuildIndex(program, src)
	res := Resolve(program, idx)
	return idx, res
}

// findIdentifierNode finds a NodeID for an Identifier with the given name
// whose Start offset is closest to nearOffset.
func findIdentifierNode(idx *Index, name string, nearOffset int) NodeID {
	bestID := NodeID(-1)
	bestDist := 999999
	quotedName := "\"" + name + "\""
	for id, node := range idx.Nodes {
		if node.Kind == "Identifier" && node.Label == quotedName {
			dist := node.Start - nearOffset
			if dist < 0 {
				dist = -dist
			}
			if dist < bestDist || (dist == bestDist && id < bestID) {
				bestDist = dist
				bestID = id
			}
		}
	}
	return bestID
}

// findIdentifierNodeExact finds a NodeID for an Identifier with the given name
// whose Start offset is exactly the given value. Returns the lowest NodeID on ties.
func findIdentifierNodeExact(idx *Index, name string, startOffset int) NodeID {
	quotedName := "\"" + name + "\""
	bestID := NodeID(-1)
	for id, node := range idx.Nodes {
		if node.Kind == "Identifier" && node.Label == quotedName && node.Start == startOffset {
			if bestID < 0 || id < bestID {
				bestID = id
			}
		}
	}
	return bestID
}

func TestResolveSimpleVar(t *testing.T) {
	src := `var x = 10;
console.log(x);
`
	idx, res := buildTestResolution(t, src)

	// Find the declaration of x (first occurrence at offset 5)
	declNode := findIdentifierNodeExact(idx, "x", 5)
	if declNode < 0 {
		t.Fatal("could not find declaration node for 'x'")
	}

	// Find the usage of x (second occurrence at offset 25)
	refNode := findIdentifierNodeExact(idx, "x", 25)
	if refNode < 0 {
		t.Fatal("could not find reference node for 'x'")
	}

	// Check that declaration is bound
	if !res.IsDeclaration(declNode) {
		t.Errorf("expected declNode %d to be a declaration", declNode)
	}

	// Check that reference resolves to the same binding
	declBinding := res.BindingForNode(declNode)
	refBinding := res.BindingForNode(refNode)
	if declBinding == nil || refBinding == nil {
		t.Fatalf("expected both nodes to have bindings: decl=%v ref=%v", declBinding, refBinding)
	}
	if declBinding != refBinding {
		t.Errorf("declaration and reference should share the same binding")
	}
	if declBinding.Kind != BindingVar {
		t.Errorf("expected BindingVar, got %s", declBinding.Kind)
	}
	if len(declBinding.References) != 1 {
		t.Errorf("expected 1 reference, got %d", len(declBinding.References))
	}

	t.Logf("Binding '%s': decl=%d, refs=%v", declBinding.Name, declBinding.DeclNodeID, declBinding.References)
}

func TestResolveLetBlockScoping(t *testing.T) {
	src := `let x = 1;
{
  let x = 2;
  console.log(x);
}
console.log(x);
`
	idx, res := buildTestResolution(t, src)

	// Outer x declaration (offset ~5)
	outerDecl := findIdentifierNode(idx, "x", 5)
	// Inner x declaration (offset ~18)
	innerDecl := findIdentifierNode(idx, "x", 18)

	if outerDecl < 0 || innerDecl < 0 {
		t.Fatal("could not find x declarations")
	}
	if outerDecl == innerDecl {
		t.Fatal("outer and inner x should be different nodes")
	}

	outerBinding := res.BindingForNode(outerDecl)
	innerBinding := res.BindingForNode(innerDecl)
	if outerBinding == nil || innerBinding == nil {
		t.Fatal("both should have bindings")
	}
	if outerBinding == innerBinding {
		t.Error("outer and inner x should have different bindings (block scoping)")
	}
	if outerBinding.Kind != BindingLet || innerBinding.Kind != BindingLet {
		t.Error("both should be BindingLet")
	}

	t.Logf("Outer x: decl=%d refs=%v", outerBinding.DeclNodeID, outerBinding.References)
	t.Logf("Inner x: decl=%d refs=%v", innerBinding.DeclNodeID, innerBinding.References)
}

func TestResolveConstDeclaration(t *testing.T) {
	src := `const PI = 3.14;
const area = PI * 2;
`
	idx, res := buildTestResolution(t, src)

	piDecl := findIdentifierNode(idx, "PI", 7)
	piRef := findIdentifierNode(idx, "PI", 32)

	if piDecl < 0 || piRef < 0 {
		t.Fatal("could not find PI nodes")
	}

	b := res.BindingForNode(piDecl)
	if b == nil {
		t.Fatal("PI declaration should have a binding")
	}
	if b.Kind != BindingConst {
		t.Errorf("expected BindingConst, got %s", b.Kind)
	}
	if res.BindingForNode(piRef) != b {
		t.Error("PI reference should resolve to same binding as declaration")
	}
}

func TestResolveFunctionParams(t *testing.T) {
	src := `function add(a, b) {
  return a + b;
}
`
	idx, res := buildTestResolution(t, src)

	// Find parameter declarations
	aDecl := findIdentifierNode(idx, "a", 14)
	bDecl := findIdentifierNode(idx, "b", 17)

	if aDecl < 0 || bDecl < 0 {
		t.Fatal("could not find parameter declarations")
	}

	aBinding := res.BindingForNode(aDecl)
	bBinding := res.BindingForNode(bDecl)
	if aBinding == nil || bBinding == nil {
		t.Fatal("parameters should have bindings")
	}
	if aBinding.Kind != BindingParameter || bBinding.Kind != BindingParameter {
		t.Errorf("expected BindingParameter, got a=%s b=%s", aBinding.Kind, bBinding.Kind)
	}

	// The references to a and b in the function body
	if len(aBinding.References) < 1 {
		t.Errorf("expected at least 1 reference to 'a', got %d", len(aBinding.References))
	}
	if len(bBinding.References) < 1 {
		t.Errorf("expected at least 1 reference to 'b', got %d", len(bBinding.References))
	}

	t.Logf("a: decl=%d refs=%v", aBinding.DeclNodeID, aBinding.References)
	t.Logf("b: decl=%d refs=%v", bBinding.DeclNodeID, bBinding.References)
}

func TestResolveVarHoisting(t *testing.T) {
	src := `function f() {
  console.log(x);
  var x = 10;
}
`
	idx, res := buildTestResolution(t, src)

	// x is used before declaration but var hoists
	xDecl := findIdentifierNodeExact(idx, "x", 40)
	xRef := findIdentifierNodeExact(idx, "x", 30)

	if xDecl < 0 || xRef < 0 {
		t.Fatal("could not find x nodes")
	}

	declBinding := res.BindingForNode(xDecl)
	refBinding := res.BindingForNode(xRef)
	if declBinding == nil {
		t.Fatal("x declaration should have a binding")
	}
	if refBinding == nil {
		t.Fatal("x reference should resolve (var is hoisted)")
	}
	if declBinding != refBinding {
		t.Error("hoisted var reference should resolve to same binding")
	}
}

func TestResolveFunctionDeclaration(t *testing.T) {
	src := `function greet(name) {
  return "Hello " + name;
}
greet("world");
`
	idx, res := buildTestResolution(t, src)

	// greet declaration (function name)
	greetDecl := findIdentifierNode(idx, "greet", 10)
	// greet usage (call)
	greetRef := findIdentifierNode(idx, "greet", 55)

	if greetDecl < 0 || greetRef < 0 {
		t.Fatal("could not find greet nodes")
	}

	b := res.BindingForNode(greetDecl)
	if b == nil {
		t.Fatal("greet should have a binding")
	}
	if b.Kind != BindingFunction {
		t.Errorf("expected BindingFunction, got %s", b.Kind)
	}
	if res.BindingForNode(greetRef) != b {
		t.Error("greet call should resolve to same binding")
	}
}

func TestResolveCatchBinding(t *testing.T) {
	src := `try {
  throw new Error("oops");
} catch (e) {
  console.log(e);
}
`
	idx, res := buildTestResolution(t, src)

	eDecl := findIdentifierNode(idx, "e", 43)
	eRef := findIdentifierNode(idx, "e", 62)

	if eDecl < 0 || eRef < 0 {
		t.Fatal("could not find 'e' nodes")
	}

	b := res.BindingForNode(eDecl)
	if b == nil {
		t.Fatal("e should have a binding")
	}
	if b.Kind != BindingCatchParam {
		t.Errorf("expected BindingCatchParam, got %s", b.Kind)
	}
	if res.BindingForNode(eRef) != b {
		t.Error("e reference should resolve to catch binding")
	}
}

func TestResolveArrowFunction(t *testing.T) {
	src := `const numbers = [1, 2, 3];
const doubled = numbers.map(n => n * 2);
`
	idx, res := buildTestResolution(t, src)

	// n parameter in arrow function
	nDecl := findIdentifierNode(idx, "n", 46)
	nRef := findIdentifierNode(idx, "n", 51)

	if nDecl < 0 || nRef < 0 {
		t.Fatal("could not find 'n' nodes")
	}

	b := res.BindingForNode(nDecl)
	if b == nil {
		t.Fatal("n should have a binding")
	}
	if b.Kind != BindingParameter {
		t.Errorf("expected BindingParameter, got %s", b.Kind)
	}
	if res.BindingForNode(nRef) != b {
		t.Error("n reference should resolve to arrow param")
	}
}

func TestResolveDotExpressionExcluded(t *testing.T) {
	src := `console.log("hello");
`
	idx, res := buildTestResolution(t, src)

	// "log" is a property access, should NOT be resolved
	logNode := findIdentifierNode(idx, "log", 9)
	if logNode < 0 {
		t.Skip("log node not found in index")
	}

	b := res.BindingForNode(logNode)
	if b != nil {
		t.Errorf("'log' in console.log should NOT have a binding (property access), got binding: %s", b.Name)
	}

	// "console" should be unresolved (global)
	consoleNode := findIdentifierNode(idx, "console", 1)
	if consoleNode >= 0 {
		if res.IsUnresolved(consoleNode) {
			t.Log("'console' correctly marked as unresolved (global)")
		}
	}
}

func TestResolveAllUsages(t *testing.T) {
	src := `function add(a, b) {
  const sum = a + b;
  return sum;
}
const result = add(2, 3);
console.log(result);
`
	idx, res := buildTestResolution(t, src)

	// Find 'result' declaration
	resultDecl := findIdentifierNode(idx, "result", 66)
	if resultDecl < 0 {
		t.Fatal("could not find result declaration")
	}

	b := res.BindingForNode(resultDecl)
	if b == nil {
		t.Fatal("result should have a binding")
	}

	usages := b.AllUsages()
	if len(usages) < 2 {
		t.Errorf("expected at least 2 usages (1 decl + 1 ref), got %d", len(usages))
	}

	t.Logf("Binding 'result': %d total usages (decl=%d, refs=%v)", len(usages), b.DeclNodeID, b.References)
}
