package main

import (
	"fmt"
	"time"

	goja "github.com/dop251/goja"
)

func main() {
	vm := goja.New()
	done := make(chan error, 1)

	go func() {
		_, err := vm.RunString(`(async () => { while (true) {} })()`)
		done <- err
	}()

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
}
