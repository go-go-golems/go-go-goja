package main

import (
    "fmt"
    "log"
    "os"

    "github.com/go-go-golems/go-go-goja/engine"
)

func main() {
    vm, req := engine.New()

    if len(os.Args) > 1 {
        // Run the provided JS file through require().
        if _, err := req.Require(os.Args[1]); err != nil {
            log.Fatal(err)
        }
        return
    }

    // Quick inline snippet to verify the environment.
    if _, err := vm.RunString(`console.log("goja runtime ready");`); err != nil {
        fmt.Println("error:", err)
    }
} 