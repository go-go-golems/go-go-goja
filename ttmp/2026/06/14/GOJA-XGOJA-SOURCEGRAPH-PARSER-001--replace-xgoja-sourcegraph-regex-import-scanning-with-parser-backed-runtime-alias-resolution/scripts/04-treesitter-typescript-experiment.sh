#!/usr/bin/env bash
set -euo pipefail

# Self-contained experiment so the ticket can evaluate tree-sitter-typescript
# without modifying the repository go.mod. If adopted, add
# github.com/tree-sitter/tree-sitter-typescript to go-go-goja/go.mod.

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
cd "$tmpdir"

go mod init xgoja-ts-tree-exp >/dev/null
go get github.com/tree-sitter/go-tree-sitter@v0.25.0 >/dev/null
go get github.com/tree-sitter/tree-sitter-typescript@v0.23.2 >/dev/null

cat > main.go <<'GO'
package main

import (
	"fmt"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

type edge struct {
	Kind      string
	Specifier string
	Dynamic   bool
}

func main() {
	cases := map[string]struct {
		Source string
		TSX    bool
	}{
		"typescript-imports": {Source: `
import type { Thing } from "./types";
import { helper } from "./helper";
import assets from "fs:assets";
export { more } from "./more";
export interface Thing { name: string }
const loaded = await import("./dynamic");
const host = require("fs:host");
const hidden = require(["fs", "assets"].join(":"));
`},
		"tsx-imports": {TSX: true, Source: `
import React from "react";
import assets from "fs:assets";
import Widget from "./Widget";
type Props = { name: string };
export const View = (props: Props) => <section>{props.name}</section>;
`},
		"comment-string-false-positive": {Source: `
const s: string = "import nope from 'not-real'";
// require("also-not-real")
`},
	}

	for name, tc := range cases {
		parser := tree_sitter.NewParser()
		var lang *tree_sitter.Language
		if tc.TSX {
			lang = tree_sitter.NewLanguage(tree_sitter_typescript.LanguageTSX())
		} else {
			lang = tree_sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript())
		}
		if err := parser.SetLanguage(lang); err != nil {
			panic(err)
		}
		tree := parser.Parse([]byte(tc.Source), nil)
		root := tree.RootNode()
		fmt.Printf("\n== %s ==\n", name)
		fmt.Printf("hasError=%v\n", root.HasError())
		fmt.Printf("sexp=%s\n", root.ToSexp())
		for _, e := range collect(root, []byte(tc.Source)) {
			fmt.Printf("edge kind=%s specifier=%q dynamic=%v\n", e.Kind, e.Specifier, e.Dynamic)
		}
		tree.Close()
		parser.Close()
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
GO

go run main.go
