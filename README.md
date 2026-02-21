# go-go-goja – Native-Module Playground for goja

**go-go-goja** is a tiny sandbox project that shows how to wire Go-implemented "native" modules into a [dop251/goja] JavaScript runtime using the [goja_nodejs/require] subsystem.

The goal is to have a place where you can:

* add your own Go files under `modules/` and immediately use them from JS via `require("your-module")`
* experiment with goja + Node-style `require()` without having to set up a whole application
* run JS files non-interactively or open an interactive prompt (`go run ./cmd/repl`)

---

## Folder layout

```text
 go-go-goja/
 ├── cmd/
 │   └── repl/            # standalone runner / interactive prompt
 ├── engine/              # one helper: engine.New() → (*goja.Runtime, *require.RequireModule)
 ├── modules/             # ← add your Go-backed modules here
 │   ├── common.go        # registry plumbing (NativeModule, Register, …)
 │   ├── fs/              # example module 1: basic file-system helpers
 │   └── exec/            # example module 2: thin wrapper around os/exec
 ├── testdata/            # demo JS scripts used by Go tests
 ├── repl_test.go         # Go test that runs a JS script through the runner
 └── go.mod
```

`engine.New()` does the heavy lifting:

1. creates a fresh `goja.Runtime`
2. enables Node-style `require()`
3. calls `modules.EnableAll(reg)` so every module that called `modules.Register()` during `init()` becomes available to JS
4. exposes a global `console` object so that `console.log()` works out-of-the-box

---

## Quick start

```bash
# from the project root
cd go-go-goja

# run a script once
❯ go run ./cmd/repl testdata/hello.js
OK

# or open the prompt (type JS, :quit to exit)
❯ go run ./cmd/repl -debug
js> const fs = require("fs");
js> fs.writeFileSync("/tmp/demo.txt", "hi");
js> console.log(fs.readFileSync("/tmp/demo.txt"));
hi
```

The `-debug` flag prints extra logs such as which modules were registered.

---

## Adding **your** native module

Say we want to expose a simplistic `uuid` module that exports a single `v4()` function.

1. **Create a sub-folder** under `modules/`:
   ```bash
   mkdir modules/uuid && touch modules/uuid/uuid.go
   ```
2. **Implement the module** – only ~40 lines:
   ```go
   // modules/uuid/uuid.go
   package uuidmod

   import (
       "github.com/dop251/goja"
       "github.com/go-go-golems/go-go-goja/modules" // registry helpers
       "github.com/google/uuid"
   )

   type m struct{}

   // compile-time check – keeps the linter happy and guarantees the interface is satisfied
   var _ modules.NativeModule = (*m)(nil)

   func (m) Name() string { return "uuid" }

   func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
       exports := moduleObj.Get("exports").(*goja.Object)
       exports.Set("v4", func() string { return uuid.NewString() })
   }

   func init() { modules.Register(&m{}) }
   ```
3. **Make sure the package is imported somewhere** so that its `init()` runs. The simplest is to add a blank-import in `engine/runtime.go` (or in `cmd/repl/main.go` if you prefer):
   ```go
   import (
       _ "github.com/go-go-golems/go-go-goja/modules/uuid" // ← new module here
   )
   ```
   The blank import is only required once – after that every call to `engine.New()` automatically enables the module.
4. **Profit**
   ```js
   const { v4 } = require("uuid");
   console.log(v4());
   ```

### Tips & conventions

* Always keep your module self-contained inside `modules/<name>/<name>.go` – easier to copy around.
* Return only **plain Go types** (string, number, bool, maps, slices) or `goja.Value`s. The runtime converts between Go & JS automatically.
* If your module allocates goroutines, honour `context.Context` and consider `errgroup` for clean cancellation.
* Use `var _ modules.NativeModule = (*yourType)(nil)` for compile-time checks (see [Go guidelines in the repo rules]).

---

## Testing

`repl_test.go` shows how to execute a JS file from a Go test:

```go
cmd := exec.Command("go", "run", "./cmd/repl", "testdata/hello.js")
cmd.Dir = "./go-go-goja" // run from module root
```

The JS script lives in `testdata/hello.js` and prints `OK` on success. Add new test scripts the same way and extend the test function.

---

## License

MIT (see LICENSE file).

[dop251/goja]:   https://github.com/dop251/goja
[goja_nodejs/require]: https://pkg.go.dev/github.com/dop251/goja_nodejs/require

---

## Asynchronous APIs (Promises & Callbacks)

`goja` lets Go code create real JavaScript `Promise`s or invoke JS callbacks later, but all VM access must happen on the runtime owner thread.

The recommended reusable pattern is `pkg/runtimeowner`:

```go
runner := runtimeowner.NewRunner(vm, loop, runtimeowner.Options{
    Name:          "my-module",
    RecoverPanics: true,
})

promise, resolve, reject := vm.NewPromise()
go func() {
    out, err := slowWork()
    _ = runner.Post(context.Background(), "myModule.settle", func(context.Context, *goja.Runtime) {
        if err != nil {
            _ = reject(vm.ToValue(err.Error()))
            return
        }
        _ = resolve(vm.ToValue(out))
    })
}()
return vm.ToValue(promise)
```

Low-level `loop.RunOnLoop(...)` is still valid, but `runtimeowner.Runner` is preferred for:

- cancellation-aware `Call`/`Post`,
- standardized errors (`ErrClosed`, `ErrScheduleRejected`, etc.),
- owner-context fast-path for nested calls.

Important caveat:

- do not run blocking synchronous flows on the owner thread if those flows wait on background work that itself schedules back onto owner, or you can deadlock.

### Demo: `timer` module

Included in `modules/timer/timer.go` is a minimal example:

```js
const { sleep } = require("timer");
await sleep(1000);   // returns a Promise that resolves after 1 s
console.log("done");
```

The Go side (simplified):

```go
exports.Set("sleep", func(ms int64) goja.Value {
    p, resolve, _ := vm.NewPromise()
    go func() {
        time.Sleep(time.Duration(ms) * time.Millisecond)
        loop.RunOnLoop(func(*goja.Runtime) { resolve(goja.Undefined()) })
    }()
    return vm.ToValue(p)
})
```

Use it as a template for any async binding you need (HTTP fetchers, database calls, …).
