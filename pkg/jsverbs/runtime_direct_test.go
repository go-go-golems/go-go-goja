package jsverbs

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

func TestInvokeInGojaRuntime(t *testing.T) {
	registry, err := ScanSource("verbs/tools.js", `
__package__({ name: "tools" })
__verb__("greet", {
  name: "greet",
  output: "text"
})
function greet() {
  return "hello intern"
}
`)
	if err != nil {
		t.Fatalf("scan source: %v", err)
	}
	verb, ok := registry.Verb("verbs tools greet")
	if !ok {
		t.Fatalf("expected verbs tools greet verb")
	}
	vm := goja.New()
	reqRegistry := require.NewRegistry(require.WithLoader(registry.RequireLoader()))
	req := reqRegistry.Enable(vm)
	ret, err := registry.InvokeInGojaRuntime(context.Background(), vm, req, verb, nil)
	if err != nil {
		t.Fatalf("invoke direct runtime: %v", err)
	}
	if ret != "hello intern" {
		t.Fatalf("ret = %#v", ret)
	}
}
