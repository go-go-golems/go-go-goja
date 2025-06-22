package main

import (
    "bufio"
    "flag"
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/go-go-golems/go-go-goja/engine"
    "github.com/dop251/goja"
)

// A tiny interactive loop for quickly evaluating JavaScript snippets against the
// fully-featured goja runtime (Node-style require(), native modules, console, …).
//
// Usage:
//   go run ./cmd/repl                 # interactive prompt
//   go run ./cmd/repl path/to/file.js # run a JS file via require()
//
// Special commands inside the prompt:
//   :quit     – exit
//   :help     – show help
// Anything else is passed verbatim to vm.RunString().
//
// Debugging: pass -debug to enable verbose engine logs.

func main() {
    var debug bool
    flag.BoolVar(&debug, "debug", false, "enable verbose debug logs")
    flag.Parse()

    vm, req := engine.New()

    if debug {
        log.Printf("engine initialised, args=%v", os.Args[1:])
    }

    // If a script path is provided, run it once and exit.
    if flag.NArg() > 0 {
        if _, err := req.Require(flag.Arg(0)); err != nil {
            log.Fatalf("failed to run script: %v", err)
        }
        return
    }

    // Interactive loop.
    reader := bufio.NewReader(os.Stdin)
    fmt.Println("goja> type JS code (:help for help)")

    for {
        fmt.Print("js> ")
        line, err := reader.ReadString('\n')
        if err != nil {
            if err.Error() == "EOF" {
                fmt.Println()
                return
            }
            log.Fatalf("reading stdin: %v", err)
        }

        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }

        switch line {
        case ":quit", ":exit":
            return
        case ":help":
            fmt.Println("Commands:\n  :help    show this help\n  :quit    exit\nOtherwise any line is evaluated as JavaScript.")
            continue
        }

        val, err := vm.RunString(line)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            if debug {
                log.Printf("eval error: %v", err)
            }
            continue
        }

        // Print non-undefined results.
        if val != nil && !goja.IsUndefined(val) {
            fmt.Println(val)
        }
    }
} 