package uidsl

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestRenderEscapesTextAndAttributes(t *testing.T) {
	vm := goja.New()
	node := vm.ToValue(&Element{Tag: "div", Attrs: attrsFromMap(map[string]any{"class": "a&b", "hidden": true}), Children: []Node{&Text{Value: "<hello>"}}})
	got, err := RenderAny(vm, node)
	if err != nil {
		t.Fatal(err)
	}
	if got != `<div class="a&amp;b" hidden>&lt;hello&gt;</div>` {
		t.Fatalf("unexpected render: %s", got)
	}
}

func TestUIDSLModulePage(t *testing.T) {
	vm := goja.New()
	mod := vm.NewObject()
	exports := vm.NewObject()
	_ = mod.Set("exports", exports)
	Loader(vm, mod)
	_ = vm.Set("ui", exports)
	v, err := vm.RunString(`ui.page({title:"Demo"}, ui.link({rel:"stylesheet", href:"/x.css"}), ui.main(ui.h1("Hi")))`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RenderAny(vm, v)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<!doctype html>", "<title>Demo</title>", `<link href="/x.css" rel="stylesheet">`, "<main><h1>Hi</h1></main>"} {
		if !strings.Contains(got, want) {
			t.Fatalf("render missing %q in %s", want, got)
		}
	}
}
