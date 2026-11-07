package jsverbs

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/file"
	"github.com/dop251/goja/parser"
	"github.com/dop251/goja/token"
)

// GojaParser is the default SourceParser. It reads sources with goja's own
// parser (pure Go), so it accepts exactly the JavaScript that goja can run as
// a script: no ES module syntax (import/export) and no TypeScript.
type GojaParser struct{}

var _ SourceParser = GojaParser{}

// DefaultSourceParser returns the parser the scanner uses when
// ScanOptions.Parser is nil.
func DefaultSourceParser() SourceParser {
	return GojaParser{}
}

// require() runs a module as the body of this function (goja_nodejs wraps it
// so), and the parser reads it the same way: a top-level return is legal, and
// the wrapper's parameters are names the module cannot redeclare with let or
// const. The prefix has no newline, so only columns on the first line shift.
const (
	commonJSPrefix = "(function(exports,require,module,__filename,__dirname){"
	commonJSSuffix = "\n})"
)

// ParseSource implements SourceParser.
func (GojaParser) ParseSource(input ParseInput) (*ParsedSource, error) {
	src := commonJSPrefix + blankHashbang(string(input.Source)) + commonJSSuffix
	program, err := parser.ParseFile(nil, input.RelPath, src, parser.IgnoreRegExpErrors, parser.WithDisableSourceMaps)
	if err != nil {
		return nil, unwrapPositions(err)
	}
	body, ok := moduleBody(program)
	if !ok {
		return nil, fmt.Errorf("%s: the source closes the CommonJS function it runs in", input.RelPath)
	}
	p := &gojaWalker{src: src}
	out := &ParsedSource{}
	for _, stmt := range body {
		out.Items = append(out.Items, p.topLevel(stmt)...)
	}
	return out, nil
}

// moduleBody returns the statements of the wrapper function: the module's own
// top level. A source that closes the wrapper early (and opens another
// function after it) has no single body.
func moduleBody(program *ast.Program) ([]ast.Statement, bool) {
	if program == nil || len(program.Body) != 1 {
		return nil, false
	}
	stmt, ok := program.Body[0].(*ast.ExpressionStatement)
	if !ok {
		return nil, false
	}
	fn, ok := stmt.Expression.(*ast.FunctionLiteral)
	if !ok || fn.Body == nil {
		return nil, false
	}
	return fn.Body.List, true
}

// unwrapPositions moves error columns on the first line back by the wrapper's
// prefix, so that they point into the module's own source.
func unwrapPositions(err error) error {
	var list parser.ErrorList
	if !errors.As(err, &list) {
		return err
	}
	for _, e := range list {
		if e != nil && e.Position.Line == 1 && e.Position.Column > len(commonJSPrefix) {
			e.Position.Column -= len(commonJSPrefix)
		}
	}
	return list
}

// blankHashbang replaces a leading "#!" line with spaces, keeping offsets,
// because goja's parser does not accept hashbang lines.
func blankHashbang(src string) string {
	if !strings.HasPrefix(src, "#!") {
		return src
	}
	end := strings.IndexByte(src, '\n')
	if end < 0 {
		end = len(src)
	}
	return strings.Repeat(" ", end) + src[end:]
}

type gojaWalker struct {
	src string
}

// text returns the source between two goja positions. Positions are 1-based
// byte offsets because the source is parsed without a FileSet.
func (w *gojaWalker) text(from, to file.Idx) string {
	start, end := int(from)-1, int(to)-1
	if start < 0 || end > len(w.src) || start > end {
		return ""
	}
	return w.src[start:end]
}

func (w *gojaWalker) nodeText(node ast.Node) string {
	if node == nil || isNilNode(node) {
		return ""
	}
	return w.text(node.Idx0(), node.Idx1())
}

func (w *gojaWalker) topLevel(stmt ast.Statement) []ParsedItem {
	switch s := stmt.(type) {
	case *ast.FunctionDeclaration:
		if item, ok := w.function(s.Function); ok {
			return []ParsedItem{item}
		}
	case *ast.VariableStatement:
		return w.bindings(s.List)
	case *ast.LexicalDeclaration:
		return w.bindings(s.List)
	case *ast.ExpressionStatement:
		return w.expressionStatement(s.Expression)
	}
	return nil
}

func (w *gojaWalker) function(fn *ast.FunctionLiteral) (ParsedItem, bool) {
	if fn == nil || fn.Name == nil || fn.Generator {
		return ParsedItem{}, false
	}
	return ParsedItem{Kind: ParsedItemFunction, Name: fn.Name.Name.String(), Params: w.params(fn.ParameterList)}, true
}

func (w *gojaWalker) bindings(list []*ast.Binding) []ParsedItem {
	items := []ParsedItem{}
	for _, b := range list {
		ident, ok := b.Target.(*ast.Identifier)
		if !ok || b.Initializer == nil {
			continue
		}
		switch init := b.Initializer.(type) {
		case *ast.ArrowFunctionLiteral:
			items = append(items, ParsedItem{Kind: ParsedItemFunction, Name: ident.Name.String(), Params: w.params(init.ParameterList)})
		case *ast.FunctionLiteral:
			if !init.Generator {
				items = append(items, ParsedItem{Kind: ParsedItemFunction, Name: ident.Name.String(), Params: w.params(init.ParameterList)})
			}
		}
	}
	return items
}

func (w *gojaWalker) expressionStatement(expr ast.Expression) []ParsedItem {
	switch e := expr.(type) {
	case *ast.CallExpression:
		callee, ok := e.Callee.(*ast.Identifier)
		if !ok || !IsMetadataSentinel(callee.Name.String()) {
			return nil
		}
		args := make([]ParsedArg, 0, len(e.ArgumentList))
		for _, arg := range e.ArgumentList {
			value, err := w.literal(arg)
			_, isObject := arg.(*ast.ObjectLiteral)
			args = append(args, ParsedArg{Object: isObject, Value: value, Err: err})
		}
		return []ParsedItem{{Kind: ParsedItemCall, Name: callee.Name.String(), Args: args}}
	case *ast.TemplateLiteral:
		tag, ok := e.Tag.(*ast.Identifier)
		if !ok || tag.Name.String() != DocTag {
			return nil
		}
		return []ParsedItem{{Kind: ParsedItemDoc, Text: w.templateRaw(e)}}
	}
	return nil
}

// templateRaw is the text between the backticks, substitutions included.
func (w *gojaWalker) templateRaw(t *ast.TemplateLiteral) string {
	return w.text(t.OpenQuote+1, t.CloseQuote)
}

func (w *gojaWalker) params(list *ast.ParameterList) []ParameterSpec {
	if list == nil {
		return nil
	}
	params := make([]ParameterSpec, 0, len(list.List)+1)
	for _, b := range list.List {
		params = append(params, w.param(b.Target))
	}
	if list.Rest != nil {
		p := w.param(list.Rest)
		p.Rest = true
		params = append(params, p)
	}
	return params
}

func (w *gojaWalker) param(target ast.Node) ParameterSpec {
	switch t := target.(type) {
	case *ast.Identifier:
		return ParameterSpec{Name: t.Name.String(), Kind: ParameterIdentifier}
	case *ast.ObjectPattern:
		return ParameterSpec{Name: w.nodeText(t), Kind: ParameterObject}
	case *ast.ArrayPattern:
		return ParameterSpec{Name: w.nodeText(t), Kind: ParameterArray}
	default:
		return ParameterSpec{Name: w.nodeText(target), Kind: ParameterUnknown}
	}
}

// literal decodes a metadata literal. Messages name the unsupported syntax
// with the same node names the tree-sitter backend uses, so diagnostics read
// the same with either backend.
func (w *gojaWalker) literal(expr ast.Expression) (interface{}, error) {
	switch e := expr.(type) {
	case *ast.ObjectLiteral:
		return w.object(e)
	case *ast.ArrayLiteral:
		ret := []interface{}{}
		for _, el := range e.Value {
			if el == nil { // a hole: [1, , 2]
				continue
			}
			value, err := w.literal(el)
			if err != nil {
				return nil, err
			}
			ret = append(ret, value)
		}
		return ret, nil
	case *ast.StringLiteral:
		return e.Value.String(), nil
	case *ast.TemplateLiteral:
		if e.Tag != nil {
			return nil, fmt.Errorf("unsupported metadata literal %q", "call_expression")
		}
		if len(e.Expressions) > 0 {
			return nil, fmt.Errorf("template metadata strings cannot contain substitutions")
		}
		return w.templateRaw(e), nil
	case *ast.NumberLiteral:
		switch v := e.Value.(type) {
		case int64:
			return float64(v), nil
		case float64:
			return v, nil
		default:
			return nil, fmt.Errorf("unsupported metadata number %q", e.Literal)
		}
	case *ast.BooleanLiteral:
		return e.Value, nil
	case *ast.NullLiteral:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported metadata literal %q", nodeKind(expr))
	}
}

func (w *gojaWalker) object(o *ast.ObjectLiteral) (map[string]interface{}, error) {
	ret := map[string]interface{}{}
	for _, prop := range o.Value {
		switch p := prop.(type) {
		case *ast.PropertyKeyed:
			if p.Kind != ast.PropertyKindValue {
				return nil, fmt.Errorf("unsupported object element %q", "method_definition")
			}
			if p.Computed {
				return nil, fmt.Errorf("unsupported object key %q", "computed_property_name")
			}
			key, ok := p.Key.(*ast.StringLiteral)
			if !ok {
				return nil, fmt.Errorf("unsupported object key %q", nodeKind(p.Key))
			}
			value, err := w.literal(p.Value)
			if err != nil {
				return nil, fmt.Errorf("field %q: %w", key.Value.String(), err)
			}
			ret[key.Value.String()] = value
		case *ast.PropertyShort:
			return nil, fmt.Errorf("unsupported object element %q", "shorthand_property_identifier")
		case *ast.SpreadElement:
			return nil, fmt.Errorf("unsupported object element %q", "spread_element")
		default:
			return nil, fmt.Errorf("unsupported object element %q", nodeKind(prop))
		}
	}
	return ret, nil
}

// nodeKind names a goja node the way tree-sitter-javascript names the same
// syntax, for messages.
func nodeKind(node ast.Node) string {
	switch n := node.(type) {
	case *ast.Identifier:
		if n.Name.String() == "undefined" {
			return "undefined"
		}
		return "identifier"
	case *ast.CallExpression:
		return "call_expression"
	case *ast.NewExpression:
		return "new_expression"
	case *ast.BinaryExpression:
		return "binary_expression"
	case *ast.UnaryExpression:
		if n.Operator == token.INCREMENT || n.Operator == token.DECREMENT {
			return "update_expression"
		}
		return "unary_expression"
	case *ast.ConditionalExpression:
		return "ternary_expression"
	case *ast.AssignExpression:
		return "assignment_expression"
	case *ast.SequenceExpression:
		return "sequence_expression"
	case *ast.DotExpression, *ast.PrivateDotExpression:
		return "member_expression"
	case *ast.BracketExpression:
		return "subscript_expression"
	case *ast.ArrowFunctionLiteral:
		return "arrow_function"
	case *ast.FunctionLiteral:
		return "function_expression"
	case *ast.ClassLiteral:
		return "class"
	case *ast.RegExpLiteral:
		return "regex"
	case *ast.ThisExpression:
		return "this"
	case *ast.AwaitExpression:
		return "await_expression"
	case *ast.YieldExpression:
		return "yield_expression"
	case *ast.SpreadElement:
		return "spread_element"
	case *ast.NumberLiteral:
		return "number"
	case *ast.TemplateLiteral:
		return "template_string"
	case nil:
		return "nil"
	default:
		name := reflect.TypeOf(node).String()
		return strings.ToLower(strings.TrimPrefix(name, "*ast."))
	}
}

func isNilNode(node ast.Node) bool {
	v := reflect.ValueOf(node)
	return v.Kind() == reflect.Ptr && v.IsNil()
}
