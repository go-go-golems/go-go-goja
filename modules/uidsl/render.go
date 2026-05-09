package uidsl

import (
	"bytes"
	"fmt"
	"html"
	"sort"
	"strings"

	"github.com/dop251/goja"
)

func RenderAny(vm *goja.Runtime, v goja.Value) (string, error) {
	n, err := Normalize(vm, v)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := renderNode(&b, n); err != nil {
		return "", err
	}
	return b.String(), nil
}

func Normalize(vm *goja.Runtime, v goja.Value) (Node, error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return &Fragment{}, nil
	}
	return NormalizeExport(v.Export())
}

func NormalizeExport(x any) (Node, error) {
	switch v := x.(type) {
	case nil:
		return &Fragment{}, nil
	case Node:
		return v, nil
	case string:
		return &Text{Value: v}, nil
	case int, int64, float64, bool:
		return &Text{Value: fmt.Sprint(v)}, nil
	case []Node:
		return &Fragment{Children: v}, nil
	case []any:
		children, err := normalizeChildren(v)
		if err != nil {
			return nil, err
		}
		return &Fragment{Children: children}, nil
	default:
		return &Text{Value: fmt.Sprint(v)}, nil
	}
}

func normalizeChildren(values []any) ([]Node, error) {
	var out []Node
	for _, v := range values {
		if v == nil {
			continue
		}
		n, err := NormalizeExport(v)
		if err != nil {
			return nil, err
		}
		if f, ok := n.(*Fragment); ok {
			out = append(out, f.Children...)
		} else {
			out = append(out, n)
		}
	}
	return out, nil
}

func renderNode(b *bytes.Buffer, n Node) error {
	switch v := n.(type) {
	case *Document:
		b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
		if v.Title != "" {
			b.WriteString("<title>")
			b.WriteString(html.EscapeString(v.Title))
			b.WriteString("</title>")
		}
		for _, c := range v.Head {
			if err := renderNode(b, c); err != nil {
				return err
			}
		}
		b.WriteString("</head><body>")
		for _, c := range v.Body {
			if err := renderNode(b, c); err != nil {
				return err
			}
		}
		b.WriteString("</body></html>")
	case *Element:
		b.WriteByte('<')
		b.WriteString(v.Tag)
		renderAttrs(b, v.Attrs)
		if voidTags[v.Tag] {
			b.WriteByte('>')
			return nil
		}
		b.WriteByte('>')
		for _, c := range v.Children {
			if err := renderNode(b, c); err != nil {
				return err
			}
		}
		b.WriteString("</")
		b.WriteString(v.Tag)
		b.WriteByte('>')
	case *Text:
		b.WriteString(html.EscapeString(v.Value))
	case *RawHTML:
		b.WriteString(v.Value)
	case *Fragment:
		for _, c := range v.Children {
			if err := renderNode(b, c); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unknown node type %T", n)
	}
	return nil
}

func renderAttrs(b *bytes.Buffer, attrs map[string]any) {
	if len(attrs) == 0 {
		return
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		if k != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := attrs[k]
		if v == nil {
			continue
		}
		if bv, ok := v.(bool); ok {
			if bv {
				b.WriteByte(' ')
				b.WriteString(k)
			}
			continue
		}
		value := attrValue(k, v)
		if value == "" {
			continue
		}
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteString("=\"")
		b.WriteString(html.EscapeString(value))
		b.WriteByte('"')
	}
}

func attrValue(k string, v any) string {
	if k == "class" {
		switch x := v.(type) {
		case []any:
			parts := []string{}
			for _, p := range x {
				if p != nil && fmt.Sprint(p) != "" && fmt.Sprint(p) != "false" {
					parts = append(parts, fmt.Sprint(p))
				}
			}
			return strings.Join(parts, " ")
		case map[string]any:
			parts := []string{}
			for name, enabled := range x {
				if truthy(enabled) {
					parts = append(parts, name)
				}
			}
			sort.Strings(parts)
			return strings.Join(parts, " ")
		}
	}
	if k == "style" {
		if m, ok := v.(map[string]any); ok {
			keys := make([]string, 0, len(m))
			for kk := range m {
				keys = append(keys, kk)
			}
			sort.Strings(keys)
			parts := []string{}
			for _, kk := range keys {
				parts = append(parts, kk+":"+fmt.Sprint(m[kk]))
			}
			return strings.Join(parts, ";")
		}
	}
	return fmt.Sprint(v)
}

func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case nil:
		return false
	case string:
		return x != "" && x != "false"
	default:
		return true
	}
}

var voidTags = map[string]bool{"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true, "img": true, "input": true, "link": true, "meta": true, "source": true, "track": true, "wbr": true}
