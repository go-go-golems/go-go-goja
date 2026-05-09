package uidsl

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestTableFromRowsFiltersSortsAndPaginates(t *testing.T) {
	vm := goja.New()
	obj := vm.NewObject()
	exports := vm.NewObject()
	_ = obj.Set("exports", exports)
	Loader(vm, obj)
	_ = vm.Set("ui", exports)
	value, err := vm.RunString(`
		ui.table.fromRows("customers", [
		  { id: 3, name: "Carla Canvas", segment: "prospect", total_cents: 0 },
		  { id: 1, name: "Alice Example", segment: "vip", total_cents: 17998 },
		  { id: 2, name: "Bob Browser", segment: "active", total_cents: 2500 }
		])
		.columns(c => c
		  .text("id").label("ID").sortable()
		  .text("name").label("Customer").sortable().filterable()
		  .badge("segment").label("Segment").filterable()
		  .money("total_cents").label("Total").align("right")
		)
		.features(f => f.filters().pagination({ size: 1 }).sorting())
		.render({ query: { q: "example", sort: "id", dir: "desc", page: "1" } })
	`)
	if err != nil {
		t.Fatal(err)
	}
	html, err := RenderAny(vm, value)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<form class="ui-table-filters" method="get">`,
		`name="q" placeholder="all columns" type="search" value="example"`,
		`<td data-column="name">Alice Example</td>`,
		`<span class="ui-badge ui-badge--vip">vip</span>`,
		`<td class="align-right" data-column="total_cents">$179.98</td>`,
		`Page 1 of 1 (1 rows)`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
	for _, notWant := range []string{"Bob Browser", "Carla Canvas"} {
		if strings.Contains(html, notWant) {
			t.Fatalf("unexpected %q in %s", notWant, html)
		}
	}
}

func TestTableFilterEmptyState(t *testing.T) {
	vm := goja.New()
	obj := vm.NewObject()
	exports := vm.NewObject()
	_ = obj.Set("exports", exports)
	Loader(vm, obj)
	_ = vm.Set("ui", exports)
	value, err := vm.RunString(`
		ui.table.fromRows("customers", [{ name: "Alice" }])
		  .features(f => f.filters())
		  .render({ query: { q: "zzz" } })
	`)
	if err != nil {
		t.Fatal(err)
	}
	html, err := RenderAny(vm, value)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `No rows match the current filters.`) {
		t.Fatalf("missing empty state in %s", html)
	}
}
