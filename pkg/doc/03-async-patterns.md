---
Title: Asynchronous Patterns with Promises
Slug: async-patterns
Short: Implementing Promise-based and callback-style async operations
Topics:
- async
- promises
- callbacks
- concurrency
Commands:
- repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

Asynchronous operations in go-go-goja bridge Go's goroutine model with JavaScript's Promise and callback patterns. The key constraint is that all JavaScript VM interactions must occur on the same goroutine that owns the runtime, requiring careful coordination between background work and JavaScript execution.

## Core Async Principle

The goja runtime is single-threaded, meaning any operation that touches JavaScript values, calls JavaScript functions, or resolves Promises must happen on the VM's goroutine.

Preferred approach in this repository:

- create a `runtimeowner.Runner` once (`runtimeowner.NewRunner(vm, loop, ...)`);
- use `runner.Call(...)` for request/response owner-thread work;
- use `runner.Post(...)` for fire-and-forget owner-thread settlement (Promise resolve/reject, callback notification).

`eventloop.RunOnLoop()` is still a valid low-level primitive, but `runtimeowner.Runner` is the recommended reusable API.

### Recommended Runner Pattern

```go
runner := runtimeowner.NewRunner(vm, loop, runtimeowner.Options{
    Name:          "my-module",
    RecoverPanics: true,
})

exports.Set("sleep", func(ms int64) goja.Value {
    promise, resolve, reject := vm.NewPromise()
    go func() {
        time.Sleep(time.Duration(ms) * time.Millisecond)
        _ = runner.Post(context.Background(), "timer.sleep.settle", func(context.Context, *goja.Runtime) {
            if ms < 0 {
                _ = reject(vm.ToValue("invalid duration"))
                return
            }
            _ = resolve(goja.Undefined())
        })
    }()
    return vm.ToValue(promise)
})
```

### Deadlock Safety Rule

Do not execute blocking synchronous flows on the owner thread when those flows wait on background goroutines that themselves need to schedule owner-thread callbacks. That creates a circular wait.

In practice:

- keep blocking orchestration off owner when possible;
- route callback/value boundaries onto owner via `runner.Call`/`runner.Post`.

## Promise-Based APIs

Promises provide the most natural async experience for JavaScript developers. Create them using `vm.NewPromise()` and resolve/reject them from background goroutines.

Note: examples below use explicit `eventloop.RunOnLoop(...)` for illustration. In production modules, prefer the runner wrapper above.

### Basic Promise Pattern

```go
// modules/timer/timer.go
package timermod

import (
    "time"
    "github.com/dop251/goja"
    "github.com/dop251/goja_nodejs/eventloop"
    "github.com/go-go-golems/go-go-goja/modules"
)

type m struct{}
var _ modules.NativeModule = (*m)(nil)

func (m) Name() string { return "timer" }

func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    loop := eventloop.Get(vm)
    
    exports.Set("sleep", func(ms int64) goja.Value {
        // Create Promise on VM goroutine
        promise, resolve, reject := vm.NewPromise()
        
        go func() {
            // Background work in separate goroutine
            time.Sleep(time.Duration(ms) * time.Millisecond)
            
            // Schedule resolution back to VM goroutine
            loop.RunOnLoop(func(*goja.Runtime) {
                resolve(goja.Undefined())
            })
        }()
        
        // Return Promise immediately
        return vm.ToValue(promise)
    })
}

func init() { modules.Register(&m{}) }
```

JavaScript usage:
```javascript
const timer = require("timer");

async function example() {
    console.log("Starting...");
    await timer.sleep(1000);
    console.log("Done after 1 second!");
}

example();
```

### HTTP Fetch with Promise

```go
// modules/fetch/fetch.go
package fetchmod

import (
    "encoding/json"
    "io"
    "net/http"
    "github.com/dop251/goja"
    "github.com/dop251/goja_nodejs/eventloop"
    "github.com/go-go-golems/go-go-goja/modules"
)

type m struct{}
var _ modules.NativeModule = (*m)(nil)

func (m) Name() string { return "fetch" }

func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    loop := eventloop.Get(vm)
    
    exports.Set("get", func(url string) goja.Value {
        promise, resolve, reject := vm.NewPromise()
        
        go func() {
            // Perform HTTP request in background
            resp, err := http.Get(url)
            if err != nil {
                loop.RunOnLoop(func(*goja.Runtime) {
                    reject(vm.ToValue(err.Error()))
                })
                return
            }
            defer resp.Body.Close()
            
            body, err := io.ReadAll(resp.Body)
            if err != nil {
                loop.RunOnLoop(func(*goja.Runtime) {
                    reject(vm.ToValue(err.Error()))
                })
                return
            }
            
            // Resolve with response data
            result := map[string]interface{}{
                "status": resp.StatusCode,
                "body":   string(body),
                "ok":     resp.StatusCode >= 200 && resp.StatusCode < 300,
            }
            
            loop.RunOnLoop(func(*goja.Runtime) {
                resolve(vm.ToValue(result))
            })
        }()
        
        return vm.ToValue(promise)
    })
}

func init() { modules.Register(&m{}) }
```

JavaScript usage:
```javascript
const fetch = require("fetch");

async function getUserData() {
    try {
        const response = await fetch.get("https://api.github.com/users/octocat");
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        
        const user = JSON.parse(response.body);
        console.log(`User: ${user.login} (${user.public_repos} repos)`);
    } catch (error) {
        console.error("Fetch failed:", error);
    }
}

getUserData();
```

## Callback-Style APIs

For Node.js-style callback patterns, accept callback functions as parameters and invoke them from the event loop.

### File Operations with Callbacks

```go
// modules/fs/async.go (addition to fs module)
func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    loop := eventloop.Get(vm)
    
    exports.Set("readFile", func(call goja.FunctionCall) goja.Value {
        if len(call.Arguments) < 2 {
            panic(vm.NewTypeError("readFile requires path and callback"))
        }
        
        path := call.Arguments[0].String()
        callback, ok := goja.AssertFunction(call.Arguments[1])
        if !ok {
            panic(vm.NewTypeError("second argument must be a function"))
        }
        
        go func() {
            data, err := os.ReadFile(path)
            
            loop.RunOnLoop(func(*goja.Runtime) {
                if err != nil {
                    // Node.js convention: callback(error, result)
                    callback(goja.Undefined(), vm.ToValue(err.Error()), goja.Undefined())
                } else {
                    callback(goja.Undefined(), goja.Null(), vm.ToValue(string(data)))
                }
            })
        }()
        
        return goja.Undefined()
    })
}
```

JavaScript usage:
```javascript
const fs = require("fs");

fs.readFile("/etc/hosts", (error, data) => {
    if (error) {
        console.error("Read failed:", error);
        return;
    }
    
    console.log("File contents:", data);
});
```

## Advanced: Streaming Data

For operations that produce multiple values over time, combine channels with repeated Promise resolution:

```go
// modules/stream/stream.go
package streammod

import (
    "context"
    "time"
    "github.com/dop251/goja"
    "github.com/dop251/goja_nodejs/eventloop"
    "github.com/go-go-golems/go-go-goja/modules"
)

type m struct{}
var _ modules.NativeModule = (*m)(nil)

func (m) Name() string { return "stream" }

func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    loop := eventloop.Get(vm)
    
    exports.Set("counter", func(limit int, intervalMs int64) goja.Value {
        promise, resolve, reject := vm.NewPromise()
        
        go func() {
            ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
            defer cancel()
            
            ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
            defer ticker.Stop()
            
            count := 0
            results := make([]int, 0, limit)
            
            for {
                select {
                case <-ctx.Done():
                    loop.RunOnLoop(func(*goja.Runtime) {
                        reject(vm.ToValue("timeout"))
                    })
                    return
                    
                case <-ticker.C:
                    count++
                    results = append(results, count)
                    
                    if count >= limit {
                        loop.RunOnLoop(func(*goja.Runtime) {
                            resolve(vm.ToValue(results))
                        })
                        return
                    }
                }
            }
        }()
        
        return vm.ToValue(promise)
    })
}

func init() { modules.Register(&m{}) }
```

JavaScript usage:
```javascript
const stream = require("stream");

async function countExample() {
    console.log("Starting counter...");
    
    // Get 5 numbers, one every 500ms
    const numbers = await stream.counter(5, 500);
    
    console.log("Received numbers:", numbers);
    // Output: [1, 2, 3, 4, 5]
}

countExample();
```

## Error Handling Best Practices

Robust error handling in async operations prevents silent failures and provides meaningful feedback to JavaScript code.

- **Always handle errors:** Never leave Promise rejection or callback error parameters unhandled
- **Use meaningful error messages:** Include context about what operation failed
- **Respect timeouts:** Use `context.WithTimeout()` for long-running operations
- **Clean up resources:** Ensure goroutines terminate and connections close properly

## Integration Tips

When combining async modules with the REPL:

```bash
go run ./cmd/repl
js> const timer = require("timer"); timer.sleep(1000).then(() => console.log("Done!"));
js> // REPL continues immediately, "Done!" appears after 1 second
```

For file-based scripts with async operations, ensure the runtime doesn't exit before Promises resolve by using top-level await or maintaining event loop activity.
