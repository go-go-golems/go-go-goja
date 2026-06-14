//go:build ignore

package main

import (
	"fmt"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
)

type edge struct {
	Kind      string
	Specifier string
	Dynamic   bool
}

func main() {
	cases := map[string]string{
		"cjs-require":                   `const x = require("fs:assets");`,
		"esm-import":                    `import assets from "fs:assets";`,
		"side-effect-import":            `import "./setup.js";`,
		"export-from":                   `export { x } from "./x.js";`,
		"dynamic-import":                `const x = await import("./x.js");`,
		"dynamic-require-expression":    `const x = require(["fs", "host"].join(":"));`,
		"string-comment-false-positive": `const s = "require('not-real')"; // import "also-not-real"`,
		"typescript-ish": `
import type { Thing } from "./types";
import { helper } from "./helper";
import assets from "fs:assets";
export { more } from "./more";
export interface Thing { name: string }
`,
	}

	parser := tree_sitter.NewParser()
	lang := tree_sitter.NewLanguage(tree_sitter_javascript.Language())
	if err := parser.SetLanguage(lang); err != nil {
		panic(err)
	}
	defer parser.Close()

	for name, src := range cases {
		tree := parser.Parse([]byte(src), nil)
		root := tree.RootNode()
		fmt.Printf("\n== %s ==\n", name)
		fmt.Printf("hasError=%v\n", root.HasError())
		fmt.Printf("sexp=%s\n", root.ToSexp())
		for _, e := range collect(root, []byte(src)) {
			fmt.Printf("edge kind=%s specifier=%q dynamic=%v\n", e.Kind, e.Specifier, e.Dynamic)
		}
		tree.Close()
	}
}

func collect(root *tree_sitter.Node, src []byte) []edge {
	var out []edge
	walk(root, src, func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "import_statement":
			for _, s := range stringDescendants(n, src) {
				out = append(out, edge{Kind: "import", Specifier: unquote(s)})
			}
		case "export_statement":
			for _, s := range stringDescendants(n, src) {
				out = append(out, edge{Kind: "export", Specifier: unquote(s)})
			}
		case "call_expression":
			fn := n.ChildByFieldName("function")
			if fn == nil {
				return
			}
			fnText := strings.TrimSpace(fn.Utf8Text(src))
			if fnText != "require" && fnText != "import" {
				return
			}
			args := n.ChildByFieldName("arguments")
			strings := stringDescendants(args, src)
			if len(strings) == 1 {
				out = append(out, edge{Kind: fnText, Specifier: unquote(strings[0])})
			} else {
				out = append(out, edge{Kind: fnText, Dynamic: true})
			}
		}
	})
	return out
}

func walk(n *tree_sitter.Node, src []byte, visit func(*tree_sitter.Node)) {
	if n == nil {
		return
	}
	visit(n)
	cursor := n.Walk()
	defer cursor.Close()
	for _, child := range n.NamedChildren(cursor) {
		childCopy := child
		walk(&childCopy, src, visit)
	}
}

func stringDescendants(n *tree_sitter.Node, src []byte) []string {
	if n == nil {
		return nil
	}
	var out []string
	walk(n, src, func(child *tree_sitter.Node) {
		if child.Kind() == "string" {
			out = append(out, child.Utf8Text(src))
		}
	})
	return out
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		q := s[0]
		if (q == '\'' || q == '"' || q == '`') && s[len(s)-1] == q {
			return s[1 : len(s)-1]
		}
	}
	return s
}
