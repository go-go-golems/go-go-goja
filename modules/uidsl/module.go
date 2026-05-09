package uidsl

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/engine"
)

type Registrar struct{}

func NewRegistrar() *Registrar  { return &Registrar{} }
func (r *Registrar) ID() string { return "ui-dsl" }
func (r *Registrar) RegisterRuntimeModules(ctx *engine.RuntimeModuleContext, reg *require.Registry) error {
	reg.RegisterNativeModule("ui.dsl", Loader)
	reg.RegisterNativeModule("ui", Loader)
	return nil
}

var tags = []string{"html", "head", "body", "title", "meta", "link", "script", "style", "main", "img", "br", "hr", "time", "svg", "path", "rect", "line", "polyline", "circle", "div", "span", "h1", "h2", "h3", "h4", "p", "a", "form", "input", "button", "select", "option", "ul", "ol", "li", "table", "thead", "tbody", "tr", "th", "td", "section", "article", "header", "footer", "nav", "label", "textarea", "strong", "em", "small", "pre", "code"}
var headTags = map[string]bool{"meta": true, "link": true, "style": true, "title": true}

func Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	for _, tag := range tags {
		tag := tag
		_ = exports.Set(tag, func(call goja.FunctionCall) goja.Value { return vm.ToValue(elementFromCall(tag, call)) })
	}
	_ = exports.Set("fragment", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(&Fragment{Children: nodesFromArgs(call.Arguments)})
	})
	_ = exports.Set("text", func(v any) *Text { return &Text{Value: fmt.Sprint(v)} })
	_ = exports.Set("raw", func(s string) *RawHTML { return &RawHTML{Value: s} })
	_ = exports.Set("render", func(v goja.Value) (string, error) { return RenderAny(vm, v) })
	_ = exports.Set("page", func(call goja.FunctionCall) goja.Value { return vm.ToValue(pageFromCall(call)) })
	_ = exports.Set("codeBlock", func(language string, source goja.Value, options ...map[string]any) goja.Value {
		return vm.ToValue(codeBlockNode(language, source.Export(), firstOptions(options)))
	})
	_ = exports.Set("sql", func(source goja.Value, options ...map[string]any) goja.Value {
		return vm.ToValue(codeBlockNode("sql", source.Export(), firstOptions(options)))
	})
	_ = exports.Set("js", func(source goja.Value, options ...map[string]any) goja.Value {
		return vm.ToValue(codeBlockNode("javascript", source.Export(), firstOptions(options)))
	})
	_ = exports.Set("jsonBlock", func(value goja.Value, options ...map[string]any) goja.Value {
		return vm.ToValue(codeBlockNode("json", jsonBlockSource(value), firstOptions(options)))
	})
	_ = exports.Set("badge", func(value goja.Value, options ...map[string]any) goja.Value {
		return vm.ToValue(badgeNode(value.Export(), firstOptions(options)))
	})
	_ = exports.Set("tabs", func(id string, tabs goja.Value, options ...map[string]any) (goja.Value, error) {
		n, err := tabsNode(id, tabs, firstOptions(options))
		if err != nil {
			return nil, err
		}
		return vm.ToValue(n), nil
	})
	tableValue := vm.ToValue(func(id string) goja.Value { return tableBuilderObject(vm, newTableBuilder(id)) })
	if tableObj, ok := tableValue.(*goja.Object); ok {
		_ = tableObj.Set("fromRows", func(id string, rows goja.Value) (goja.Value, error) {
			builder, err := tableFromRows(id, rows)
			if err != nil {
				return nil, err
			}
			return tableBuilderObject(vm, builder), nil
		})
	}
	_ = exports.Set("table", tableValue)
}

func elementFromCall(tag string, call goja.FunctionCall) *Element {
	attrs := map[string]any{}
	args := call.Arguments
	if len(args) > 0 && isAttrs(args[0]) {
		if m, ok := args[0].Export().(map[string]any); ok {
			attrs = m
		}
		args = args[1:]
	}
	return &Element{Tag: tag, Attrs: attrs, Children: nodesFromArgs(args)}
}

func pageFromCall(call goja.FunctionCall) *Document {
	title := ""
	args := call.Arguments
	if len(args) > 0 && isAttrs(args[0]) {
		if m, ok := args[0].Export().(map[string]any); ok {
			if t, ok := m["title"]; ok {
				title = fmt.Sprint(t)
			}
		}
		args = args[1:]
	}
	children := nodesFromArgs(args)
	doc := &Document{Title: title}
	for _, child := range children {
		if e, ok := child.(*Element); ok && headTags[e.Tag] {
			doc.Head = append(doc.Head, child)
		} else {
			doc.Body = append(doc.Body, child)
		}
	}
	return doc
}

func nodesFromArgs(args []goja.Value) []Node {
	var out []Node
	for _, arg := range args {
		if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
			continue
		}
		n, err := NormalizeExport(arg.Export())
		if err != nil {
			out = append(out, &Text{Value: err.Error()})
			continue
		}
		if f, ok := n.(*Fragment); ok {
			out = append(out, f.Children...)
		} else {
			out = append(out, n)
		}
	}
	return out
}

func isAttrs(v goja.Value) bool {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return false
	}
	switch v.Export().(type) {
	case Node, string, []any, []Node, int, int64, float64, bool:
		return false
	case map[string]any:
		return true
	default:
		return false
	}
}
