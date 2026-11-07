// Package treesitter provides the tree-sitter SourceParser for jsverbs.
//
// jsverbs scans with goja's parser by default, which is pure Go. Use this
// parser when sources contain syntax goja's script parser rejects, mainly ES
// module syntax (import/export), or when partial results from a file with
// syntax errors are preferred to a diagnostic. It links tree-sitter and its
// JavaScript grammar, which need cgo.
//
//	registry, err := jsverbs.ScanDir(dir, jsverbs.ScanOptions{
//		Extensions:             []string{".js"},
//		FailOnErrorDiagnostics: true,
//		Parser:                 treesitter.Parser{},
//	})
package treesitter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
)

// Parser is a jsverbs.SourceParser built on tree-sitter-javascript.
type Parser struct{}

var _ jsverbs.SourceParser = Parser{}

// ParseSource implements jsverbs.SourceParser. tree-sitter recovers from
// syntax errors, so this returns an error only when no tree can be built.
func (Parser) ParseSource(input jsverbs.ParseInput) (*jsverbs.ParsedSource, error) {
	parser := tree_sitter.NewParser()
	defer parser.Close()
	lang := tree_sitter.NewLanguage(tree_sitter_javascript.Language())
	if err := parser.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("set javascript language: %w", err)
	}
	tree := parser.Parse(input.Source, nil)
	if tree == nil {
		return nil, fmt.Errorf("parse %s: nil tree", input.RelPath)
	}
	defer tree.Close()

	w := &walker{src: input.Source, out: &jsverbs.ParsedSource{}}
	root := tree.RootNode()
	for i := uint(0); i < root.ChildCount(); i++ {
		w.topLevel(root.Child(i))
	}
	return w.out, nil
}

type walker struct {
	src []byte
	out *jsverbs.ParsedSource
}

func (w *walker) add(item jsverbs.ParsedItem) {
	w.out.Items = append(w.out.Items, item)
}

func (w *walker) topLevel(node *tree_sitter.Node) {
	if node == nil {
		return
	}
	switch node.Kind() {
	case "expression_statement":
		w.topLevel(node.Child(0))
	case "call_expression":
		w.call(node)
	case "function_declaration":
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil || nameNode.Kind() != "identifier" {
			return
		}
		w.add(jsverbs.ParsedItem{Kind: jsverbs.ParsedItemFunction, Name: w.text(nameNode), Params: w.params(node)})
	case "lexical_declaration", "variable_declaration", "export_statement":
		for i := uint(0); i < node.ChildCount(); i++ {
			w.topLevel(node.Child(i))
		}
	case "variable_declarator":
		nameNode := node.ChildByFieldName("name")
		valueNode := node.ChildByFieldName("value")
		if nameNode == nil || valueNode == nil || nameNode.Kind() != "identifier" {
			return
		}
		switch valueNode.Kind() {
		// "function" is the node name in older grammars; tree-sitter-javascript
		// 0.21 and later call it "function_expression".
		case "arrow_function", "function", "function_expression":
			w.add(jsverbs.ParsedItem{Kind: jsverbs.ParsedItemFunction, Name: w.text(nameNode), Params: w.params(valueNode)})
		}
	}
}

func (w *walker) call(node *tree_sitter.Node) {
	fnNode := node.ChildByFieldName("function")
	if fnNode == nil {
		return
	}
	name := w.text(fnNode)
	if name == jsverbs.DocTag {
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child.Kind() == "template_string" {
				w.add(jsverbs.ParsedItem{Kind: jsverbs.ParsedItemDoc, Text: w.templateRaw(child)})
				return
			}
		}
		return
	}
	if !jsverbs.IsMetadataSentinel(name) {
		return
	}
	argsNode := node.ChildByFieldName("arguments")
	if argsNode == nil {
		return
	}
	args := []jsverbs.ParsedArg{}
	for i := uint(0); i < argsNode.ChildCount(); i++ {
		child := argsNode.Child(i)
		switch child.Kind() {
		case ",", "(", ")", "comment":
			continue
		}
		value, err := w.literal(child)
		args = append(args, jsverbs.ParsedArg{Object: child.Kind() == "object", Value: value, Err: err})
	}
	w.add(jsverbs.ParsedItem{Kind: jsverbs.ParsedItemCall, Name: name, Args: args})
}

func (w *walker) text(node *tree_sitter.Node) string {
	if node == nil {
		return ""
	}
	start, end := node.StartByte(), node.EndByte()
	n := uint(len(w.src))
	if end < start || start > n || end > n {
		return ""
	}
	return string(w.src[start:end])
}

func (w *walker) templateRaw(node *tree_sitter.Node) string {
	text := strings.TrimPrefix(w.text(node), "`")
	return strings.TrimSuffix(text, "`")
}

func (w *walker) literal(node *tree_sitter.Node) (interface{}, error) {
	if node == nil {
		return nil, fmt.Errorf("literal node is nil")
	}
	switch node.Kind() {
	case "object":
		return w.object(node)
	case "array":
		ret := []interface{}{}
		for i := uint(0); i < node.NamedChildCount(); i++ {
			child := node.NamedChild(i)
			if child.Kind() == "comment" {
				continue
			}
			value, err := w.literal(child)
			if err != nil {
				return nil, err
			}
			ret = append(ret, value)
		}
		return ret, nil
	case "string":
		return decodeStringLiteral(w.text(node))
	case "template_string":
		for i := uint(0); i < node.NamedChildCount(); i++ {
			// Escape sequences are fine: the value is the raw text.
			if node.NamedChild(i).Kind() == "template_substitution" {
				return nil, fmt.Errorf("template metadata strings cannot contain substitutions")
			}
		}
		return w.templateRaw(node), nil
	case "number":
		return strconv.ParseFloat(w.text(node), 64)
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported metadata literal %q", node.Kind())
	}
}

func (w *walker) object(node *tree_sitter.Node) (map[string]interface{}, error) {
	ret := map[string]interface{}{}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		switch child.Kind() {
		case "pair":
			keyNode := child.ChildByFieldName("key")
			valueNode := child.ChildByFieldName("value")
			if keyNode == nil || valueNode == nil {
				return nil, fmt.Errorf("object pair missing key or value")
			}
			key, err := w.key(keyNode)
			if err != nil {
				return nil, err
			}
			value, err := w.literal(valueNode)
			if err != nil {
				return nil, fmt.Errorf("field %q: %w", key, err)
			}
			ret[key] = value
		case "comment":
			continue
		default:
			return nil, fmt.Errorf("unsupported object element %q", child.Kind())
		}
	}
	return ret, nil
}

func (w *walker) key(node *tree_sitter.Node) (string, error) {
	switch node.Kind() {
	case "property_identifier", "identifier":
		return w.text(node), nil
	case "string", "template_string":
		value, err := w.literal(node)
		if err != nil {
			return "", err
		}
		s, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("object key must resolve to string")
		}
		return s, nil
	default:
		return "", fmt.Errorf("unsupported object key %q", node.Kind())
	}
}

func (w *walker) params(node *tree_sitter.Node) []jsverbs.ParameterSpec {
	if node == nil {
		return nil
	}
	if paramsNode := node.ChildByFieldName("parameters"); paramsNode != nil {
		params := []jsverbs.ParameterSpec{}
		for i := uint(0); i < paramsNode.ChildCount(); i++ {
			child := paramsNode.Child(i)
			switch child.Kind() {
			case ",", "(", ")", "comment":
				continue
			default:
				params = append(params, w.param(child))
			}
		}
		return params
	}
	if paramNode := node.ChildByFieldName("parameter"); paramNode != nil {
		return []jsverbs.ParameterSpec{w.param(paramNode)}
	}
	return nil
}

func (w *walker) param(node *tree_sitter.Node) jsverbs.ParameterSpec {
	if node == nil {
		return jsverbs.ParameterSpec{}
	}
	switch node.Kind() {
	case "identifier":
		return jsverbs.ParameterSpec{Name: w.text(node), Kind: jsverbs.ParameterIdentifier}
	case "rest_pattern":
		argument := node.ChildByFieldName("argument")
		if argument == nil && node.NamedChildCount() > 0 {
			argument = node.NamedChild(0)
		}
		p := w.param(argument)
		p.Rest = true
		return p
	case "assignment_pattern":
		left := node.ChildByFieldName("left")
		if left == nil && node.NamedChildCount() > 0 {
			left = node.NamedChild(0)
		}
		return w.param(left)
	case "object_pattern":
		return jsverbs.ParameterSpec{Name: w.text(node), Kind: jsverbs.ParameterObject}
	case "array_pattern":
		return jsverbs.ParameterSpec{Name: w.text(node), Kind: jsverbs.ParameterArray}
	default:
		return jsverbs.ParameterSpec{Name: w.text(node), Kind: jsverbs.ParameterUnknown}
	}
}

func decodeStringLiteral(raw string) (string, error) {
	if len(raw) < 2 {
		return "", fmt.Errorf("invalid string literal")
	}
	switch raw[0] {
	case '"':
		value, err := strconv.Unquote(raw)
		if err != nil {
			return "", fmt.Errorf("decode string literal: %w", err)
		}
		return value, nil
	case '\'':
		return decodeSingleQuotedLiteral(raw)
	default:
		return "", fmt.Errorf("unsupported string literal %q", raw)
	}
}

func decodeSingleQuotedLiteral(raw string) (string, error) {
	if len(raw) < 2 || raw[0] != '\'' || raw[len(raw)-1] != '\'' {
		return "", fmt.Errorf("invalid single-quoted literal")
	}
	var sb strings.Builder
	for i := 1; i < len(raw)-1; i++ {
		ch := raw[i]
		if ch != '\\' {
			sb.WriteByte(ch)
			continue
		}
		i++
		if i >= len(raw)-1 {
			return "", fmt.Errorf("unterminated escape sequence")
		}
		switch raw[i] {
		case '\\', '\'', '"', '`':
			sb.WriteByte(raw[i])
		case 'n':
			sb.WriteByte('\n')
		case 'r':
			sb.WriteByte('\r')
		case 't':
			sb.WriteByte('\t')
		default:
			return "", fmt.Errorf("unsupported escape sequence \\%c", raw[i])
		}
	}
	return sb.String(), nil
}
