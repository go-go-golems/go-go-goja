package runtime

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

func TestRuntimeGlobalKinds(t *testing.T) {
	vm := goja.New()
	_, err := vm.RunString(`
var x = 1;
function fn() {}
`)
	if err != nil {
		t.Fatalf("RunString failed: %v", err)
	}

	kinds := RuntimeGlobalKinds(vm)
	if kinds["fn"] != jsparse.BindingFunction {
		t.Fatalf("expected fn to be function, got=%v", kinds["fn"])
	}
	if kinds["x"] != jsparse.BindingVar {
		t.Fatalf("expected x to be var, got=%v", kinds["x"])
	}
	if _, ok := kinds["Object"]; ok {
		t.Fatalf("builtin Object should be filtered out")
	}
}
