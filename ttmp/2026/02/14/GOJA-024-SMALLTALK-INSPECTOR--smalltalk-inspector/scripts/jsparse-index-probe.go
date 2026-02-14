//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"sort"

	"github.com/dop251/goja/parser"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

func main() {
	source := `
function greet(name) { return "hi " + name; }
class Animal {
  constructor(name) { this.name = name; this.alive = true; }
  eat(food) { if (!this.alive) throw new Error("dead"); return food; }
  sleep() { return "zzz"; }
}
class Dog extends Animal {
  constructor(name) { super(name); this.breed = "lab"; }
  bark() { const sound = this.breed === "husky" ? "awoo" : "woof"; return sound; }
  fetch(item) { return this.eat(item); }
}
function main() {
  const config = { apiUrl: "https://api.example.com/v3", retries: 5 };
  return new Dog("Rex");
}
`

	program, err := parser.ParseFile(nil, "sample.js", source, 0)
	if err != nil {
		fmt.Printf("parseErr=%v\n", err)
	}
	if program == nil {
		panic("no program")
	}

	idx := jsparse.BuildIndex(program, source)
	res := jsparse.Resolve(program, idx)
	idx.Resolution = res

	global := res.Scopes[res.RootScopeID]
	fmt.Println("== Global Bindings ==")
	names := make([]string, 0, len(global.Bindings))
	for name := range global.Bindings {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		b := global.Bindings[name]
		n := idx.Nodes[b.DeclNodeID]
		if n == nil {
			fmt.Printf("- %s kind=%s decl=<missing> refs=%d\n", name, b.Kind, len(b.References))
			continue
		}
		fmt.Printf("- %s kind=%s declNode=%s span=%d..%d refs=%d\n", name, b.Kind, n.Kind, n.Start, n.End, len(b.References))
	}

	fmt.Println("== Unresolved Identifiers ==")
	for _, nodeID := range res.Unresolved {
		n := idx.Nodes[nodeID]
		if n == nil {
			continue
		}
		fmt.Printf("- %s span=%d..%d\n", n.Label, n.Start, n.End)
	}

	fmt.Println("== Selected Node Labels Around ClassLiteral/MethodDefinition ==")
	for _, nodeID := range idx.OrderedByStart {
		n := idx.Nodes[nodeID]
		if n == nil {
			continue
		}
		if n.Kind == "ClassLiteral" || n.Kind == "MethodDefinition" || n.Kind == "ClassDeclaration" {
			fmt.Printf("- %s label=%s span=%d..%d depth=%d\n", n.Kind, n.Label, n.Start, n.End, n.Depth)
		}
	}
}
