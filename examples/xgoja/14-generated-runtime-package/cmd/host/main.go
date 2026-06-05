package main

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/examples/xgoja/14-generated-runtime-package/internal/xgojaruntime"
)

func main() {
	bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{})
	if err != nil {
		panic(err)
	}
	rt, err := bundle.NewRuntime(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	ret, err := rt.Owner.Call(context.Background(), "example-host", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`require("hello").greet("package host")`)
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(ret)
}
