package sourcegraph

import (
	"fmt"
	"path"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

type importSpec struct {
	Specifier string
	Kind      string
	Dynamic   bool
}

func parseImports(filename string, source []byte) ([]importSpec, error) {
	parser := tree_sitter.NewParser()
	defer parser.Close()

	language := languageForPath(filename)
	if err := parser.SetLanguage(language); err != nil {
		return nil, fmt.Errorf("configure parser for %s: %w", filename, err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return nil, fmt.Errorf("parse %s: parser returned nil tree", filename)
	}
	defer tree.Close()
	root := tree.RootNode()
	if root == nil {
		return nil, fmt.Errorf("parse %s: parser returned nil root", filename)
	}
	if root.HasError() {
		return nil, fmt.Errorf("parse %s: syntax errors while collecting imports", filename)
	}

	seen := map[string]bool{}
	out := []importSpec{}
	add := func(spec importSpec) {
		key := fmt.Sprintf("%s\x00%s\x00%t", spec.Kind, spec.Specifier, spec.Dynamic)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, spec)
	}

	walkTree(root, source, func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "import_statement":
			if specifier, ok := staticSourceField(n, source); ok {
				add(importSpec{Kind: "import", Specifier: specifier})
			}
		case "export_statement":
			if specifier, ok := staticSourceField(n, source); ok {
				add(importSpec{Kind: "export", Specifier: specifier})
			}
		case "call_expression":
			fn := n.ChildByFieldName("function")
			if fn == nil {
				return
			}
			fnText := strings.TrimSpace(fn.Utf8Text(source))
			if fnText != "require" && fnText != "import" {
				return
			}
			args := n.ChildByFieldName("arguments")
			if specifier, ok := firstArgumentStringLiteral(args, source); ok {
				add(importSpec{Kind: fnText, Specifier: specifier})
				return
			}
			add(importSpec{Kind: fnText, Dynamic: true})
		}
	})
	return out, nil
}

func languageForPath(filename string) *tree_sitter.Language {
	switch strings.ToLower(path.Ext(filename)) {
	case ".ts", ".mts", ".cts":
		return tree_sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript())
	case ".tsx":
		return tree_sitter.NewLanguage(tree_sitter_typescript.LanguageTSX())
	default:
		return tree_sitter.NewLanguage(tree_sitter_javascript.Language())
	}
}

func staticSourceField(n *tree_sitter.Node, source []byte) (string, bool) {
	if n == nil {
		return "", false
	}
	sourceNode := n.ChildByFieldName("source")
	if sourceNode == nil || sourceNode.Kind() != "string" {
		return "", false
	}
	return unquoteTreeSitterString(sourceNode.Utf8Text(source)), true
}

func firstArgumentStringLiteral(args *tree_sitter.Node, source []byte) (string, bool) {
	if args == nil {
		return "", false
	}
	cursor := args.Walk()
	defer cursor.Close()
	children := args.NamedChildren(cursor)
	if len(children) == 0 {
		return "", false
	}
	first := children[0]
	if first.Kind() != "string" {
		return "", false
	}
	return unquoteTreeSitterString(first.Utf8Text(source)), true
}

func walkTree(n *tree_sitter.Node, source []byte, visit func(*tree_sitter.Node)) {
	if n == nil {
		return
	}
	visit(n)
	cursor := n.Walk()
	defer cursor.Close()
	for _, child := range n.NamedChildren(cursor) {
		childCopy := child
		walkTree(&childCopy, source, visit)
	}
}

func unquoteTreeSitterString(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		quote := s[0]
		if (quote == '\'' || quote == '"' || quote == '`') && s[len(s)-1] == quote {
			return s[1 : len(s)-1]
		}
	}
	return s
}
