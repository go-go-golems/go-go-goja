package uidsl

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestRenderOptionPreservesEmptyValue(t *testing.T) {
	vm := goja.New()
	moduleObj := vm.NewObject()
	exports := vm.NewObject()
	if err := moduleObj.Set("exports", exports); err != nil {
		t.Fatal(err)
	}
	Loader(vm, moduleObj)
	optionFn, ok := goja.AssertFunction(exports.Get("option"))
	if !ok {
		t.Fatalf("option export is not callable")
	}
	attrs := vm.NewObject()
	if err := attrs.Set("value", ""); err != nil {
		t.Fatal(err)
	}
	if err := attrs.Set("selected", true); err != nil {
		t.Fatal(err)
	}
	node, err := optionFn(goja.Undefined(), attrs, vm.ToValue("All columns"))
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := RenderAny(vm, node)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, `value=""`) {
		t.Fatalf("expected empty value attribute in %q", rendered)
	}
	if !strings.Contains(rendered, `selected`) {
		t.Fatalf("expected selected attribute in %q", rendered)
	}
}
