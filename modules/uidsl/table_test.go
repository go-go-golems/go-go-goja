package uidsl

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestTableFromRowsRendersEscapedHTMLTable(t *testing.T) {
	vm := goja.New()
	obj := vm.NewObject()
	exports := vm.NewObject()
	_ = obj.Set("exports", exports)
	Loader(vm, obj)
	_ = vm.Set("ui", exports)
	value, err := vm.RunString(`
		ui.table.fromRows("people", [
		  { name: "Alice", note: "<admin>" },
		  { name: "Bob", note: "ok" }
		]).features(f => f.pagination().sorting().columnPicker()).render({ query: {} })
	`)
	if err != nil {
		t.Fatal(err)
	}
	html, err := RenderAny(vm, value)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<table class="ui-table ui-table--pagination ui-table--sorting ui-table--column-picker" id="people">`,
		`<th data-column="name">name</th>`,
		`<th data-column="note">note</th>`,
		`<td data-column="name">Alice</td>`,
		`<td data-column="note">&lt;admin&gt;</td>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered table missing %q in %s", want, html)
		}
	}
}

func TestTableFunctionBuilderRendersEmptyTable(t *testing.T) {
	vm := goja.New()
	obj := vm.NewObject()
	exports := vm.NewObject()
	_ = obj.Set("exports", exports)
	Loader(vm, obj)
	_ = vm.Set("ui", exports)
	value, err := vm.RunString(`ui.table("empty").render({})`)
	if err != nil {
		t.Fatal(err)
	}
	html, err := RenderAny(vm, value)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `<table class="ui-table" id="empty">`) {
		t.Fatalf("rendered empty table = %s", html)
	}
}
