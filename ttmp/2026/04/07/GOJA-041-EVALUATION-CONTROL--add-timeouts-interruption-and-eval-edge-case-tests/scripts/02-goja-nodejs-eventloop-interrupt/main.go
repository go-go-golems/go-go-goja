package main

import (
	"fmt"
	"time"

	goja "github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

func main() {
	vm := goja.New()
	loop := eventloop.NewEventLoop()
	done := make(chan error, 1)

	go loop.Start()

	ok := loop.RunOnLoop(func(loopVM *goja.Runtime) {
		_, err := loopVM.RunString(`(async () => { while (true) {} })()`)
		done <- err
	})

	fmt.Println("scheduled", ok)

	time.AfterFunc(50*time.Millisecond, func() {
		fmt.Println("interrupting")
		vm.Interrupt(fmt.Errorf("timeout"))
	})

	select {
	case err := <-done:
		fmt.Printf("done %T %v\n", err, err)
	case <-time.After(2 * time.Second):
		fmt.Println("timed out waiting")
	}

	loop.Stop()
}
