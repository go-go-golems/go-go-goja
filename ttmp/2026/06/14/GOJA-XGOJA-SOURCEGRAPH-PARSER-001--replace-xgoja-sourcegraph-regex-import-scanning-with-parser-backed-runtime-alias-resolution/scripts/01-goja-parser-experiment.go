//go:build ignore

package main

import (
	"fmt"

	"github.com/dop251/goja/parser"
)

func main() {
	cases := []string{
		`const x = require("fs:assets");`,
		`import assets from "fs:assets";`,
		`import "./setup.js";`,
		`export { x } from "./x.js";`,
		`const x = await import("./x.js");`,
	}
	for _, src := range cases {
		_, err := parser.ParseFile(nil, "test.js", src, 0)
		fmt.Printf("%q -> %v\n", src, err)
	}
}
