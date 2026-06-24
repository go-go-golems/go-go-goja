// Command lazyproxy demonstrates "lazy" Go data structures exposed to
// JavaScript through goja Proxies. Each subcommand is one experiment.
//
// Run from the module dir (workspace go.work forces the local goja fork):
//
//	GOWORK=off go run . 01      # lazy map / "device" object
//	GOWORK=off go run . 02      # infinite lazy array
//	GOWORK=off go run . 03      # lazy nested property object
package main

import (
	"fmt"
	"os"
)

func main() {
	sub := "01"
	if len(os.Args) > 1 {
		sub = os.Args[1]
	}
	switch sub {
	case "01":
		runLazyMap()
	case "02":
		runInfiniteArray()
	case "03":
		runLazyPropertyObject()
	case "04":
		runCheapEnum()
	default:
		fmt.Println("usage: go run . [01|02|03]")
		os.Exit(2)
	}
}
