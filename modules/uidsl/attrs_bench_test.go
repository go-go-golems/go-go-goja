package uidsl

import (
	"testing"

	"github.com/dop251/goja"
)

func benchmarkUIRuntime(b *testing.B) (*goja.Runtime, *goja.Object) {
	b.Helper()
	vm := goja.New()
	obj := vm.NewObject()
	exports := vm.NewObject()
	if err := obj.Set("exports", exports); err != nil {
		b.Fatal(err)
	}
	Loader(vm, obj)
	return vm, exports
}

func BenchmarkElementFromCallAttrs9(b *testing.B) {
	vm, exports := benchmarkUIRuntime(b)
	divFn, ok := goja.AssertFunction(exports.Get("div"))
	if !ok {
		b.Fatal("div export is not callable")
	}
	attrs := vm.NewObject()
	_ = attrs.Set("class", "attr-node state-1")
	_ = attrs.Set("id", "attr-node-1")
	_ = attrs.Set("data-index", "1")
	_ = attrs.Set("data-kind", "benchmark")
	_ = attrs.Set("data-group", "1")
	_ = attrs.Set("data-label", "attribute heavy node 1")
	_ = attrs.Set("aria-label", "Attribute heavy node 1")
	_ = attrs.Set("role", "listitem")
	_ = attrs.Set("tabindex", "0")
	child := vm.ToValue("node-1")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := divFn(goja.Undefined(), attrs, child); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderAttrsAttrs9(b *testing.B) {
	attrs := map[string]any{
		"class":      "attr-node state-1",
		"id":         "attr-node-1",
		"data-index": "1",
		"data-kind":  "benchmark",
		"data-group": "1",
		"data-label": "attribute heavy node 1",
		"aria-label": "Attribute heavy node 1",
		"role":       "listitem",
		"tabindex":   "0",
	}
	el := &Element{Tag: "div", Attrs: attrsFromMap(attrs), Children: []Node{&Text{Value: "node-1"}}}
	vm := goja.New()
	value := vm.ToValue(el)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := RenderAny(vm, value); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderPageAttrs1000(b *testing.B) {
	vm, exports := benchmarkUIRuntime(b)
	_ = vm.Set("ui", exports)
	_, err := vm.RunString(`function attrsPage() {
	const children = [];
	for (let i = 0; i < 1000; i++) {
		children.push(ui.div({
			class: "attr-node state-" + (i % 5),
			id: "attr-node-" + i,
			"data-index": String(i),
			"data-kind": "benchmark",
			"data-group": String(i % 17),
			"data-label": "attribute heavy node " + i,
			"aria-label": "Attribute heavy node " + i,
			role: "listitem",
			tabindex: "0"
		}, ui.span({ class: "label" }, "node-" + i)));
	}
	return ui.html(ui.body(ui.main(ui.section({ class: "attr-root", role: "list" }, children))));
}`)
	if err != nil {
		b.Fatal(err)
	}
	fn, ok := goja.AssertFunction(vm.Get("attrsPage"))
	if !ok {
		b.Fatal("attrsPage is not callable")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value, err := fn(goja.Undefined())
		if err != nil {
			b.Fatal(err)
		}
		if _, err := RenderAny(vm, value); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderPageFlat1000(b *testing.B) {
	vm, exports := benchmarkUIRuntime(b)
	_ = vm.Set("ui", exports)
	_, err := vm.RunString(`function flatPage() {
	const children = [];
	for (let i = 0; i < 1000; i++) {
		children.push(ui.div({ class: "flat-node", "data-index": String(i) }, "node-" + i));
	}
	return ui.html(ui.body(ui.main(ui.section({ class: "flat-root" }, children))));
}`)
	if err != nil {
		b.Fatal(err)
	}
	fn, ok := goja.AssertFunction(vm.Get("flatPage"))
	if !ok {
		b.Fatal("flatPage is not callable")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value, err := fn(goja.Undefined())
		if err != nil {
			b.Fatal(err)
		}
		if _, err := RenderAny(vm, value); err != nil {
			b.Fatal(err)
		}
	}
}
