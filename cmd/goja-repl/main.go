package main

import (
	"fmt"
	"os"
)

func main() {
	root, err := newRootCommand(os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
