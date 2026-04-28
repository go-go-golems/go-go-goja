package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/engine"
)

func main() {
	ctx := context.Background()

	factory, err := engine.NewBuilder().WithModules(engine.DefaultRegistryModules()).Build() //nolint:staticcheck
	if err != nil {
		panic(err)
	}

	rt, err := factory.NewRuntime(ctx)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = rt.Close(context.Background())
	}()

	type result struct {
		value any
		err   error
	}

	done := make(chan result, 1)
	interruptErr := errors.New("engine runtime interrupt")

	go func() {
		value, callErr := rt.Owner.Call(ctx, "script.sync-interrupt", func(_ context.Context, vm *goja.Runtime) (any, error) {
			fmt.Println("sameVM", vm == rt.VM)

			time.AfterFunc(100*time.Millisecond, func() {
				fmt.Println("interrupting")
				rt.VM.Interrupt(interruptErr)
			})

			return vm.RunString("while (true) {}")
		})
		done <- result{value: value, err: callErr}
	}()

	select {
	case res := <-done:
		fmt.Printf("interruptResult value=%#v err=%v\n", res.value, res.err)
		fmt.Println("interruptIs", errors.Is(res.err, interruptErr))
	case <-time.After(2 * time.Second):
		fmt.Println("interruptResult timed out waiting")
		return
	}

	rt.VM.ClearInterrupt()

	value, err := rt.Owner.Call(ctx, "script.post-interrupt", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString("1 + 1")
	})
	fmt.Printf("postInterrupt value=%v err=%v\n", value, err)
}
