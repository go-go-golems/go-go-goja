package main

import (
	"fmt"

	goja "github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

func main() {
	vm := goja.New()
	loop := eventloop.NewEventLoop()
	ch := make(chan bool, 1)

	go loop.Start()

	loop.RunOnLoop(func(loopVM *goja.Runtime) {
		ch <- (vm == loopVM)
	})

	fmt.Println("sameVM", <-ch)
	loop.Stop()
}
