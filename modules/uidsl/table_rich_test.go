package uidsl

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestRichTableDataColumnsSortingAndPagination(t *testing.T) {
	vm := goja.New()
	obj := vm.NewObject()
	exports := vm.NewObject()
	_ = obj.Set("exports", exports)
	Loader(vm, obj)
	_ = vm.Set("ui", exports)
	value, err := vm.RunString(`
		ui.table("orders")
		  .data(ctx => ({
		    rows: [
		      { id: "A-1", status: "paid", total: 1200 },
		      { id: "A-2", status: "pending", total: 900 }
		    ],
		    total: 30
		  }))
		  .columns(c => c
		    .text("id").label("Order").sortable()
		    .badge("status").label("Status")
		    .money("total").label("Total").align("right")
		  )
		  .features(f => f.pagination({ size: 10 }).sorting())
		  .render({ query: { page: "2", sort: "id", dir: "asc" } })
	`)
	if err != nil {
		t.Fatal(err)
	}
	html, err := RenderAny(vm, value)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<a href="?dir=desc&amp;page=1&amp;sort=id">Order</a>`,
		`<td class="align-right" data-column="total">$12.00</td>`,
		`<nav class="ui-table-pagination">Page 2 of 3 (30 rows)`,
		`<a href="?dir=asc&amp;page=1&amp;sort=id">Previous</a>`,
		`<a href="?dir=asc&amp;page=3&amp;sort=id">Next</a>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
}
