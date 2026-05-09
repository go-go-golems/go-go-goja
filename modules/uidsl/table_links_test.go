package uidsl

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestTableColumnLinks(t *testing.T) {
	vm := goja.New()
	obj := vm.NewObject()
	exports := vm.NewObject()
	_ = obj.Set("exports", exports)
	Loader(vm, obj)
	_ = vm.Set("ui", exports)
	value, err := vm.RunString(`
		ui.table.fromRows("customers", [{ id: 7, name: "Alice & Co" }])
		  .columns(c => c.text("name").label("Customer").link(row => "/customers/" + row.id))
		  .render({ query: {} })
	`)
	if err != nil {
		t.Fatal(err)
	}
	html, err := RenderAny(vm, value)
	if err != nil {
		t.Fatal(err)
	}
	want := `<td data-column="name"><a href="/customers/7">Alice &amp; Co</a></td>`
	if !strings.Contains(html, want) {
		t.Fatalf("missing %q in %s", want, html)
	}
}

func TestTableColumnTemplateLinks(t *testing.T) {
	vm := goja.New()
	obj := vm.NewObject()
	exports := vm.NewObject()
	_ = obj.Set("exports", exports)
	Loader(vm, obj)
	_ = vm.Set("ui", exports)
	value, err := vm.RunString(`
		ui.table.fromRows("tables", [{ name: "order items" }])
		  .columns(c => c.text("name").link("/tables/{name}"))
		  .render({ query: {} })
	`)
	if err != nil {
		t.Fatal(err)
	}
	html, err := RenderAny(vm, value)
	if err != nil {
		t.Fatal(err)
	}
	want := `<a href="/tables/order+items">order items</a>`
	if !strings.Contains(html, want) {
		t.Fatalf("missing %q in %s", want, html)
	}
}
