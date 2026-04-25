---
Title: 'FS Module and goja_nodejs Integration: Complete Analysis, Design, and Implementation Guide'
Ticket: GOJA-053
Status: active
Topics:
    - goja
    - modules
    - fs
    - nodejs-compat
    - goja-nodejs
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Complete analysis and implementation guide for enhancing the fs primitive module in go-go-goja and ensuring all goja_nodejs core modules (buffer, console, process, url, util) are properly require()-able from JavaScript."
LastUpdated: 2026-04-25T08:00:00-04:00
WhatFor: "Onboarding guide for implementing native Go modules in go-go-goja, with deep system context for all the moving parts."
WhenToUse: "When implementing or reviewing the fs module enhancement, when wiring goja_nodejs modules, or when onboarding to the go-go-goja module system."
---

# FS Module and goja_nodejs Integration: Complete Analysis, Design, and Implementation Guide

## Executive Summary

This document is a complete, detailed analysis, design, and implementation guide for two related tasks in the go-go-goja project:

1. **Rebuild the `fs` primitive module as promise-based** — The existing `fs` module in `go-go-goja/modules/fs/` currently provides only two synchronous functions: `readFileSync` and `writeFileSync`. This guide proposes rebuilding it as a **promise-based async module** following the proven pattern established by the `timer` module. The go-go-goja runtime already has full async support: the `runtimebridge` provides per-VM bindings to the event loop and runtime owner, the REPL evaluator automatically unwraps top-level `await` expressions and polls pending promises until resolution, and the `runtimeowner.Runner` serializes all VM mutations safely. The new fs module will expose async functions like `readFile(path)`, `writeFile(path, data)`, `mkdir(path)`, `readdir(path)`, `stat(path)`, `unlink(path)`, `appendFile(path, data)`, `rename(oldPath, newPath)`, `copyFile(src, dst)`, and `exists(path)` — all returning Promises. Synchronous counterparts (`readFileSync`, etc.) will also be provided as convenience wrappers for simple scripts.

2. **Ensure all `goja_nodejs` core modules are `require()`-able** — The `goja_nodejs` library (`github.com/dop251/goja_nodejs`) ships several built-in modules (`buffer`, `console`, `process`, `url`, `util`) that register themselves via `init()` functions using `require.RegisterCoreModule(...)`. However, these registrations only take effect when the corresponding Go packages are imported (blank-imported) into the binary. Currently, go-go-goja only directly imports `goja_nodejs/require`, `goja_nodejs/eventloop`, and `goja_nodejs/console`. The `buffer`, `process`, `url`, and `util` packages may not be registered and therefore may not be `require()`-able from JavaScript. This guide provides a definitive inventory and wiring plan.

The document is written for a new intern who needs to understand every layer of the system — from the goja JavaScript engine itself, through the goja_nodejs compatibility layer, through the go-go-goja module registry and engine factory, down to the specific files that need to change.

## Problem Statement

### Problem 1: The `fs` Module Is Too Minimal

The go-go-goja project ships an `fs` primitive module (at `modules/fs/fs.go`) that exposes only two functions to JavaScript:

- `readFileSync(path)` — reads a file and returns its content as a string
- `writeFileSync(path, data)` — writes a string to a file

While these two operations cover the most basic use case, real JavaScript programs typically need more file system operations. For example:

- Checking if a file or directory exists before operating on it
- Creating directories recursively
- Listing directory contents
- Getting file metadata (size, modification time, permissions)
- Deleting files
- Appending to existing files
- Renaming or moving files
- Copying files

Node.js provides all of these via its `fs` built-in module. When users write JavaScript that runs inside go-go-goja, they currently cannot do any of these things without dropping back to Go and writing a custom module. This is a significant gap for scripts that need to interact with the file system.

### Problem 2: goja_nodejs Modules May Not Be Accessible

The `goja_nodejs` library provides several Node.js-compatible modules implemented in Go. Each module registers itself in a Go `init()` function using `require.RegisterCoreModule(...)`. The modules that exist in goja_nodejs are:

| Package | Module Name | Registers As | Has `Enable()` |
|---------|-------------|---------------|----------------|
| `goja_nodejs/console` | `console` | Core module | Yes — sets global `console` |
| `goja_nodejs/buffer` | `buffer` | Core module | Yes — sets global `Buffer` |
| `goja_nodejs/process` | `process` | Core module | Yes — sets global `process` |
| `goja_nodejs/url` | `url` | Core module | Yes — sets global `URL`, `URLSearchParams` |
| `goja_nodejs/util` | `util` | Core module | No |

**The key insight**: Go only runs `init()` functions for packages that are transitively imported. If no code in the go-go-goja binary imports (even blank-imports) `goja_nodejs/buffer`, then that package's `init()` never runs, and `require("buffer")` will fail at runtime with "No such built-in module".

Currently, the go-go-goja `engine/runtime.go` file imports:

- `goja_nodejs/require` (directly, for the registry type)
- `goja_nodejs/eventloop` (directly, for the event loop)
- `goja_nodejs/console` (directly, called in `factory.go`)

The packages `buffer`, `process`, `url`, and `util` are **not** imported anywhere in go-go-goja. This means:

- `require("buffer")` will likely fail
- `require("process")` will likely fail
- `require("url")` will likely fail
- `require("util")` will likely fail
- `require("node:buffer")` will likely fail
- `require("node:process")` will likely fail
- `require("node:url")` will likely fail
- `require("node:util")` will likely fail

Additionally, some of these modules have `Enable()` functions that set up globals (like `Buffer`, `process`, `URL`). These `Enable()` calls need to happen during runtime initialization for the full Node.js compatibility experience.

### Why This Matters

Users writing JavaScript for go-go-goja expect a Node.js-like environment. When `require("fs")` only has two functions, or when `require("buffer")` throws an error, the experience is broken and confusing. The goal is to make the JavaScript runtime feel as Node.js-compatible as possible for the modules that goja_nodejs already implements.

## Current-State Architecture

This section explains every major component in the system, from the bottom up. If you understand these layers, you will understand exactly where and why changes are needed.

### Layer 1: The Goja JavaScript Engine

**What it is**: Goja (`github.com/dop251/goja`) is a JavaScript engine implemented entirely in Go. It is not a binding to V8 or SpiderMonkey — it is a from-scratch implementation of ECMAScript (ES5.1+ with many ES6+ features). Goja compiles JavaScript source code to bytecode and executes it on a custom virtual machine.

**Key characteristics**:

- **Single-threaded**: Like all JavaScript engines, goja executes code on a single thread. There is no parallel JavaScript execution within a single `goja.Runtime` instance.
- **No built-in I/O**: Goja provides the JavaScript language itself (objects, arrays, functions, promises, etc.) but does not include any I/O APIs. There is no `console.log`, no `require()`, no `fs`, no `http` — these must all be provided by the embedding application.
- **Go-interop**: Goja allows you to create JavaScript values from Go (`vm.ToValue(...)`) and to expose Go functions to JavaScript (`exports.Set("name", goFunc)`). When JavaScript calls a Go function, goja marshals the arguments from JS values to Go types and marshals the return value back.

**Key files in goja** (at `goja/` in the workspace):

| File | Purpose |
|------|----------|
| `runtime.go` | The `Runtime` struct — the main entry point for creating and using a JS VM |
| `vm.go` | The virtual machine that executes compiled bytecode |
| `value.go` | The `Value` interface — represents any JavaScript value in Go |
| `object.go` | The `Object` struct — represents a JavaScript object |
| `func.go` | Function handling — how Go functions are called from JS |
| `builtin_*.go` | Built-in constructors and prototypes (Object, Array, Function, Promise, etc.) |
| `compiler.go` | Compiles JavaScript AST to bytecode |

**How you use goja directly** (without any module system):

```go
vm := goja.New()
vm.Set("greet", func(name string) string {
    return "Hello, " + name
})
result, err := vm.RunString(`greet("world")`)
// result = "Hello, world"
```

Notice there is no `require()` function. Goja only has what you explicitly give it.

### Layer 2: The goja_nodejs Compatibility Library

**What it is**: `goja_nodejs` (`github.com/dop251/goja_nodejs`) is a companion library that provides Node.js-compatible APIs on top of goja. It does NOT provide full Node.js — it provides a subset of the most commonly needed modules.

**The require system** (`goja_nodejs/require/`):

The require package is the module loading system. It adds a `require()` function to the JavaScript runtime so that code can import modules. The require system handles three types of modules:

1. **Native modules** — Go-implemented modules registered via `Registry.RegisterNativeModule(name, loader)`. These take priority over everything else. The loader is a Go function `func(*goja.Runtime, *goja.Object)` that populates `module.exports`.

2. **Core/built-in modules** — Similar to native modules but registered globally via `require.RegisterCoreModule(name, loader)`. Core modules are special because they can also be loaded with the `node:` prefix (e.g., `require("node:buffer")` loads the same module as `require("buffer")`).

3. **File-based modules** — JavaScript files loaded from the host filesystem. The require system follows Node.js module resolution: it looks for `./relative/path.js`, then `node_modules/`, etc.

**Module resolution algorithm** (simplified, from `require/resolve.go`):

```
require(name):
  if name looks like a file path (starts with ./, ../, /):
    try to load as file or directory
  else:
    1. Check registry-specific native modules
    2. Check global native modules
    3. Check core/built-in modules (also try with/without node: prefix)
    4. Search node_modules directories
    5. Search global folders
    6. Return InvalidModuleError
```

The critical detail for us: **step 3** (core modules) only works if the corresponding Go package was imported, triggering its `init()` function which calls `require.RegisterCoreModule(...)`.

**The event loop** (`goja_nodejs/eventloop/`):

The event loop provides `setTimeout`, `setInterval`, `setImmediate`, and their `clear*` counterparts. It runs a background goroutine that accepts jobs and executes them serially on the VM. The event loop is essential for any asynchronous JavaScript patterns.

**Current goja_nodejs modules inventory** (from our investigation of the workspace at `goja_nodejs/`):

| Directory | Module Name | Register Call | Enable() Function | Status |
|-----------|-------------|---------------|--------------------|---------|
| `console/` | `"console"` | `require.RegisterCoreModule("console", Require)` + sets global | `Enable(vm)` — sets `console` global | **Already used by go-go-goja** |
| `buffer/` | `"buffer"` | `require.RegisterCoreModule("buffer", Require)` | `Enable(vm)` — sets `Buffer` global | **NOT imported by go-go-goja** |
| `process/` | `"process"` | `require.RegisterCoreModule("process", Require)` | `Enable(vm)` — sets `process` global with `env` | **NOT imported by go-go-goja** |
| `url/` | `"url"` | `require.RegisterCoreModule("url", Require)` | `Enable(vm)` — sets `URL`, `URLSearchParams` globals | **NOT imported by go-go-goja** |
| `util/` | `"util"` | `require.RegisterCoreModule("util", Require)` | No Enable function | **NOT imported by go-go-goja** |
| `errors/` | N/A (helper) | Not a module — provides error construction helpers used by other modules | N/A | Utility only |
| `eventloop/` | N/A (runtime) | Not a module — provides the event loop machinery | N/A | **Already used by go-go-goja** |
| `require/` | N/A (runtime) | Not a module — provides the require() system itself | N/A | **Already used by go-go-goja** |

**What is NOT in goja_nodejs**: There is NO `fs` module. The goja_nodejs library does not implement any file system operations. This is why go-go-goja has its own `fs` module at `modules/fs/fs.go`.

### Layer 3: The go-go-goja Module System

**What it is**: Go-go-goja builds its own module management layer on top of goja_nodejs. This layer provides:

1. A standardized `NativeModule` interface that all go-go-goja modules implement
2. A global `DefaultRegistry` that collects modules via `init()` registration
3. A `Factory` pattern for creating configured runtime instances
4. TypeScript declaration support via the `TypeScriptDeclarer` interface

**The NativeModule interface** (from `modules/common.go`):

```go
type NativeModule interface {
    Name() string   // e.g., "fs", "exec", "timer", "database"
    Doc() string    // Human-readable documentation
    Loader(*goja.Runtime, *goja.Object)  // Populates module.exports
}
```

Every module is a Go struct that implements this interface. The `Loader` method is the bridge between Go and JavaScript — it receives the goja runtime and the module object, and it sets properties on `module.exports` that become the JavaScript-facing API.

**Module registration flow**:

```
┌─────────────────────┐
│  Module init()      │  Package-level init() runs when Go package is imported
│  modules.Register() │  Adds module to DefaultRegistry
└─────────┬───────────┘
          │
          ▼
┌─────────────────────────────┐
│  DefaultRegistry            │  Global registry collects all modules
│  modules = [fs, exec,       │
│    timer, database, ...]    │
└─────────┬───────────────────┘
          │
          ▼ (at runtime creation time)
┌─────────────────────────────────────┐
│  engine.DefaultRegistryModules()    │  ModuleSpec that calls
│  -> modules.EnableAll(gojaReg)      │  modules.EnableAll which iterates
│     -> for each module:             │  over DefaultRegistry and calls
│        gojaReg.RegisterNativeModule │  goja_nodejs/require.Registry.RegisterNativeModule
└─────────┬───────────────────────────┘
          │
          ▼
┌─────────────────────────────┐
│  goja_nodejs require.Registry│  Native modules now registered
│  native = {"fs": loader,    │  and accessible via require("fs")
│    "exec": loader, ...}     │
└─────────────────────────────┘
```

**The Factory pattern** (from `engine/factory.go` and `engine/module_specs.go`):

The `Factory` is an immutable, pre-validated plan for creating runtime instances. You build it with `NewBuilder()`, add modules and configuration, then call `Build()` to get an immutable `Factory`. Each call to `factory.NewRuntime(ctx)` creates a fresh runtime with all the configured modules.

```go
factory, err := engine.NewBuilder().
    WithModules(engine.DefaultRegistryModules()).  // Register all modules from DefaultRegistry
    Build()

rt, err := factory.NewRuntime(ctx)
// rt.VM is a *goja.Runtime
// rt.Require is a *require.RequireModule
// rt.Loop is a *eventloop.EventLoop
// rt.Owner is a runtimeowner.Runner
```

**Existing go-go-goja modules** (from `modules/`):

| Package | Name | Functions | Has TS Decl |
|---------|------|-----------|-------------|
| `modules/fs/` | `"fs"` | `readFileSync`, `writeFileSync` | Yes |
| `modules/exec/` | `"exec"` | `run` | Yes |
| `modules/timer/` | `"timer"` | `sleep` (Promise-based) | No |
| `modules/database/` | `"database"` | `configure`, `query`, `exec`, `close` | Yes |

**How the existing fs module works** (current implementation at `modules/fs/fs.go`):

```go
type m struct{}  // Empty struct — no state needed

var _ modules.NativeModule = (*m)(nil)
var _ modules.TypeScriptDeclarer = (*m)(nil)

func (m) Name() string { return "fs" }

func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    
    modules.SetExport(exports, "fs", "readFileSync", func(path string) (string, error) {
        b, err := os.ReadFile(path)
        return string(b), err
    })
    
    modules.SetExport(exports, "fs", "writeFileSync", func(path, data string) error {
        return os.WriteFile(path, []byte(data), 0o644)
    })
}

func init() {
    modules.Register(&m{})
}
```

The pattern is:
1. Define an empty struct `m`
2. Implement `Name()`, `Doc()`, `Loader()`
3. Optionally implement `TypeScriptModule()` for TypeScript declaration generation
4. Call `modules.Register(&m{})` in `init()`
5. In `Loader`, get the `exports` object and set Go functions on it using `modules.SetExport(...)`
6. Go functions receive Go types (string, int, etc.) — goja automatically converts JS values to Go types
7. Return values from Go functions are automatically converted back to JS values
8. If a Go function returns an `error`, it becomes a JavaScript exception

### Layer 4: Runtime Infrastructure

Go-go-goja provides additional infrastructure beyond the basic module system:

**runtimebridge** (`pkg/runtimebridge/runtimebridge.go`): Stores per-VM bindings (context, event loop, owner) in a `sync.Map` keyed by `*goja.Runtime`. Modules like `timer` use this to access the event loop and owner from within their Go functions.

**runtimeowner** (`pkg/runtimeowner/`): Provides a `Runner` interface that serializes VM work. It has `Call()` (request/response) and `Post()` (fire-and-forget) methods. This ensures all JavaScript execution happens on the same thread, which is required by goja.

**The full runtime creation flow** (in `factory.NewRuntime()`):

```
1. goja.New()                    → Create bare JavaScript VM
2. eventloop.NewEventLoop()      → Create event loop
3. loop.Start()                  → Start loop goroutine
4. runtimeowner.NewRunner()      → Create runner for serialized execution
5. runtimebridge.Store()         → Store bindings for this VM
6. require.NewRegistry()         → Create module registry
7. For each ModuleSpec:          → Register modules into the registry
     spec.Register(reg)
8. For each RuntimeModuleRegistrar: → Register runtime-scoped modules
     registrar.RegisterRuntimeModules(ctx, reg)
9. reg.Enable(vm)                → Add require() function to VM
10. console.Enable(vm)           → Add console global
11. For each RuntimeInitializer: → Run per-runtime setup
      initializer.InitRuntime(ctx)
```

Notice step 10: only `console.Enable(vm)` is called. The other goja_nodejs modules' `Enable()` functions (buffer, process, url) are NOT called, meaning their globals are not set up.

---

## Gap Analysis

### Gap 1: fs Module — Missing Functions

The current `fs` module exposes only `readFileSync` and `writeFileSync`. Here is a comparison with the most commonly used Node.js `fs` synchronous functions:

| Function | Node.js has it | go-go-goja has it | Priority |
|----------|:-:|:-:|:-:|
| `readFileSync(path)` | ✅ | ✅ | — |
| `writeFileSync(path, data)` | ✅ | ✅ | — |
| `existsSync(path)` | ✅ | ❌ | **High** — basic guard before operations |
| `mkdirSync(path, opts?)` | ✅ | ❌ | **High** — needed for directory creation |
| `readdirSync(path)` | ✅ | ❌ | **High** — listing directory contents |
| `statSync(path)` | ✅ | ❌ | **Medium** — file metadata |
| `unlinkSync(path)` | ✅ | ❌ | **High** — deleting files |
| `appendFileSync(path, data)` | ✅ | ❌ | **Medium** — appending to files |
| `renameSync(oldPath, newPath)` | ✅ | ❌ | **Medium** — renaming/moving files |
| `copyFileSync(src, dst)` | ✅ | ❌ | **Medium** — copying files |
| `rmSync(path, opts?)` | ✅ | ❌ | **Low** — recursive removal (risky) |
| `readFileSync(path, "utf-8")` | ✅ | Partial | **Medium** — encoding parameter not supported |
| `writeFileSync(path, Buffer)` | ✅ | ❌ | **Low** — Buffer support |

The current implementation also has a limitation: `readFileSync` always returns a string, and `writeFileSync` only accepts string data. Node.js also supports Buffer input/output.

### Gap 2: goja_nodejs Modules Not Wired

**Missing blank imports** — The following goja_nodejs packages are never imported in go-go-goja, so their `init()` functions never run:

- `_ "github.com/dop251/goja_nodejs/buffer"` — Not imported anywhere
- `_ "github.com/dop251/goja_nodejs/process"` — Not imported anywhere  
- `_ "github.com/dop251/goja_nodejs/url"` — Not imported anywhere
- `_ "github.com/dop251/goja_nodejs/util"` — Not imported anywhere

**Missing Enable() calls** — Even if the `init()` functions ran (registering the modules as core modules so `require("buffer")` works), the globals would not be set up. These `Enable()` calls are missing from the factory:

- `buffer.Enable(vm)` — Should set the global `Buffer` constructor
- `process.Enable(vm)` — Should set the global `process` object with `process.env`
- `url.Enable(vm)` — Should set the global `URL` and `URLSearchParams` constructors

Note: `util` has no `Enable()` function — it is only accessible via `require("util")`.

### Evidence Summary

| Claim | Evidence |
|-------|----------|
| `buffer` not imported | `grep -rn 'goja_nodejs/buffer' go-go-goja/` returns only `go.mod` dependency |
| `process` not imported | `grep -rn 'goja_nodejs/process' go-go-goja/` returns nothing |
| `url` not imported | `grep -rn 'goja_nodejs/url' go-go-goja/` returns nothing |
| `util` not imported | `grep -rn 'goja_nodejs/util' go-go-goja/` returns nothing |
| Only `console.Enable` called | `factory.go` line: `console.Enable(vm)` — only console enabled |
| goja_nodejs has no fs | `find goja_nodejs/ -name '*fs*'` returns empty |
| fs module only has 2 functions | `modules/fs/fs.go` — only `readFileSync` and `writeFileSync` |

---

## Proposed Architecture

### Part A: Promise-Based fs Module

The enhanced `fs` module will be **primarily promise-based**, following the proven pattern from the `timer` module. The go-go-goja runtime already has all the infrastructure needed:

**Why promises work in go-go-goja**:

1. **`runtimebridge.Lookup(vm)`** returns per-VM bindings (`Bindings{Context, Loop, Owner}`) that give any module access to the event loop and runtime owner — already used by the `timer` module.

2. **`vm.NewPromise()`** creates a native goja Promise with `resolve` and `reject` callbacks. These callbacks must be called from the owner thread (event loop goroutine) — never from arbitrary goroutines.

3. **`bindings.Owner.Post(ctx, op, fn)`** schedules a function to run on the owner thread. This is the bridge: a background goroutine does the I/O work, then calls `Post()` to resolve/reject the promise on the owner thread.

4. **The REPL evaluator** (`pkg/repl/evaluators/javascript/evaluator.go`) already:
   - Detects when a `goja.Promise` is returned from evaluation
   - Polls it via `waitForPromise()` until resolved
   - Wraps top-level `await` expressions: `await readFile("x.txt")` becomes `(async () => { return await readFile("x.txt"); })()`

**The promise-based fs pattern** (illustrated with `readFile`):

```
JavaScript:  const content = await fs.readFile("/tmp/hello.txt")
                                     │
                                     ▼
Go Loader:   modules.SetExport(exports, "readFile", func(path string) goja.Value {
                 promise, resolve, reject := vm.NewPromise()
                 bindings := runtimebridge.Lookup(vm)
                 
                 go func() {                    // ← background goroutine
                     b, err := os.ReadFile(path)  // ← blocking I/O
                     if err != nil {
                         bindings.Owner.Post(ctx, "fs.readFile.reject", func(_ context.Context, _ *goja.Runtime) {
                             reject(vm.ToValue(err.Error()))  // ← back on owner thread
                         })
                         return
                     }
                     bindings.Owner.Post(ctx, "fs.readFile.resolve", func(_ context.Context, _ *goja.Runtime) {
                         resolve(vm.ToValue(string(b)))  // ← back on owner thread
                     })
                 }()
                 
                 return vm.ToValue(promise)  // ← return Promise immediately
             })
                                     │
                                     ▼
REPL:        waitForPromise() polls until fulfilled
             → displays resolved value
```

**Key design rules for promise-based modules**:

1. **Never call `resolve`/`reject` from a background goroutine** — always go through `bindings.Owner.Post()`.
2. **Never touch `vm` from a background goroutine** — `vm.ToValue()`, `vm.NewObject()`, etc. must only happen on the owner thread.
3. **Return the promise immediately** from the exported function — the caller gets a Promise and can `.then()`/`.catch()` or `await` it.
4. **Respect context cancellation** — check `bindings.Context.Done()` in the background goroutine.

**API design — async-first, sync as convenience**:

The primary API is async (returns Promises). The sync variants are thin wrappers that block until the promise resolves. This matches modern Node.js conventions where async is preferred.

```typescript
// Promise-based API (primary)
export interface FsModule {
  // Async functions — return Promises
  readFile(path: string): Promise<string>;
  writeFile(path: string, data: string): Promise<void>;
  exists(path: string): Promise<boolean>;
  mkdir(path: string, options?: { recursive?: boolean; mode?: number }): Promise<void>;
  readdir(path: string): Promise<string[]>;
  stat(path: string): Promise<FileStats>;
  unlink(path: string): Promise<void>;
  appendFile(path: string, data: string): Promise<void>;
  rename(oldPath: string, newPath: string): Promise<void>;
  copyFile(src: string, dst: string): Promise<void>;

  // Sync convenience wrappers (blocking, simpler for scripts)
  readFileSync(path: string): string;
  writeFileSync(path: string, data: string): void;
  existsSync(path: string): boolean;
  mkdirSync(path: string, options?: { recursive?: boolean; mode?: number }): void;
  readdirSync(path: string): string[];
  statSync(path: string): FileStats;
  unlinkSync(path: string): void;
  appendFileSync(path: string, data: string): void;
  renameSync(oldPath: string, newPath: string): void;
  copyFileSync(src: string, dst: string): void;
}

export interface FileStats {
  name: string;
  size: number;
  mode: number;
  modTime: string;     // ISO 8601
  isDir: boolean;
  isFile: boolean;
}
```

**Implementation structure** — to keep the module clean, we split the async and sync implementations:

- **`fs.go`** — Module definition, `Loader()` that wires both async and sync exports
- **`fs_async.go`** — Promise-based async functions (each spawns a goroutine)
- **`fs_sync.go`** — Synchronous blocking wrappers

**Pseudocode for async functions** (`fs_async.go`):

```go
func asyncReadFile(vm *goja.Runtime, bindings runtimebridge.Bindings, path string) goja.Value {
    promise, resolve, reject := vm.NewPromise()
    
    go func() {
        b, err := os.ReadFile(path)
        if err != nil {
            _ = bindings.Owner.Post(bindings.Context, "fs.readFile.reject", func(_ context.Context, _ *goja.Runtime) {
                _ = reject(vm.ToValue(err.Error()))
            })
            return
        }
        _ = bindings.Owner.Post(bindings.Context, "fs.readFile.resolve", func(_ context.Context, _ *goja.Runtime) {
            _ = resolve(vm.ToValue(string(b)))
        })
    }()
    
    return vm.ToValue(promise)
}

func asyncWriteFile(vm *goja.Runtime, bindings runtimebridge.Bindings, path, data string) goja.Value {
    promise, resolve, reject := vm.NewPromise()
    
    go func() {
        err := os.WriteFile(path, []byte(data), 0o644)
        if err != nil {
            _ = bindings.Owner.Post(bindings.Context, "fs.writeFile.reject", func(_ context.Context, _ *goja.Runtime) {
                _ = reject(vm.ToValue(err.Error()))
            })
            return
        }
        _ = bindings.Owner.Post(bindings.Context, "fs.writeFile.resolve", func(_ context.Context, _ *goja.Runtime) {
            _ = resolve(goja.Undefined())
        })
    }()
    
    return vm.ToValue(promise)
}

func asyncExists(vm *goja.Runtime, bindings runtimebridge.Bindings, path string) goja.Value {
    promise, resolve, _ := vm.NewPromise()
    
    go func() {
        _, err := os.Stat(path)
        _ = bindings.Owner.Post(bindings.Context, "fs.exists.resolve", func(_ context.Context, _ *goja.Runtime) {
            _ = resolve(vm.ToValue(err == nil))
        })
    }()
    
    return vm.ToValue(promise)
}

func asyncMkdir(vm *goja.Runtime, bindings runtimebridge.Bindings, path string, recursive bool, mode os.FileMode) goja.Value {
    promise, resolve, reject := vm.NewPromise()
    
    go func() {
        var err error
        if recursive {
            err = os.MkdirAll(path, mode)
        } else {
            err = os.Mkdir(path, mode)
        }
        if err != nil {
            _ = bindings.Owner.Post(bindings.Context, "fs.mkdir.reject", func(_ context.Context, _ *goja.Runtime) {
                _ = reject(vm.ToValue(err.Error()))
            })
            return
        }
        _ = bindings.Owner.Post(bindings.Context, "fs.mkdir.resolve", func(_ context.Context, _ *goja.Runtime) {
            _ = resolve(goja.Undefined())
        })
    }()
    
    return vm.ToValue(promise)
}

func asyncReaddir(vm *goja.Runtime, bindings runtimebridge.Bindings, path string) goja.Value {
    promise, resolve, reject := vm.NewPromise()
    
    go func() {
        entries, err := os.ReadDir(path)
        if err != nil {
            _ = bindings.Owner.Post(bindings.Context, "fs.readdir.reject", func(_ context.Context, _ *goja.Runtime) {
                _ = reject(vm.ToValue(err.Error()))
            })
            return
        }
        names := make([]string, len(entries))
        for i, entry := range entries {
            names[i] = entry.Name()
        }
        _ = bindings.Owner.Post(bindings.Context, "fs.readdir.resolve", func(_ context.Context, _ *goja.Runtime) {
            _ = resolve(vm.ToValue(names))
        })
    }()
    
    return vm.ToValue(promise)
}

func asyncStat(vm *goja.Runtime, bindings runtimebridge.Bindings, path string) goja.Value {
    promise, resolve, reject := vm.NewPromise()
    
    go func() {
        info, err := os.Stat(path)
        if err != nil {
            _ = bindings.Owner.Post(bindings.Context, "fs.stat.reject", func(_ context.Context, _ *goja.Runtime) {
                _ = reject(vm.ToValue(err.Error()))
            })
            return
        }
        _ = bindings.Owner.Post(bindings.Context, "fs.stat.resolve", func(_ context.Context, _ *goja.Runtime) {
            _ = resolve(vm.ToValue(map[string]interface{}{
                "name":    info.Name(),
                "size":    info.Size(),
                "mode":    int64(info.Mode()),
                "modTime": info.ModTime().Format(time.RFC3339),
                "isDir":   info.IsDir(),
                "isFile":  info.Mode().IsRegular(),
            }))
        })
    }()
    
    return vm.ToValue(promise)
}

func asyncUnlink(vm *goja.Runtime, bindings runtimebridge.Bindings, path string) goja.Value { /* same pattern with os.Remove */ }
func asyncAppendFile(vm *goja.Runtime, bindings runtimebridge.Bindings, path, data string) goja.Value { /* same pattern with os.OpenFile+WriteString */ }
func asyncRename(vm *goja.Runtime, bindings runtimebridge.Bindings, oldPath, newPath string) goja.Value { /* same pattern with os.Rename */ }
func asyncCopyFile(vm *goja.Runtime, bindings runtimebridge.Bindings, src, dst string) goja.Value { /* same pattern with os.ReadFile+os.WriteFile */ }
```

**Pseudocode for sync wrappers** (`fs_sync.go`):

```go
// Sync wrappers simply do the I/O inline. Since the JS thread is already
// blocked during a sync call, there's no need for promises/goroutines.
// These are for simple scripts that don't need async.

func syncReadFileSync(path string) (string, error) {
    b, err := os.ReadFile(path)
    return string(b), err
}

func syncWriteFileSync(path, data string) error {
    return os.WriteFile(path, []byte(data), 0o644)
}

func syncExistsSync(path string) bool {
    _, err := os.Stat(path)
    return err == nil
}

// ... etc for each function
```

**The Loader wires both** (`fs.go`):

```go
func (mod m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    bindings, ok := runtimebridge.Lookup(vm)
    if !ok || bindings.Owner == nil {
        panic(vm.NewGoError(fmt.Errorf("fs module requires runtime owner bindings")))
    }
    
    // Async functions (return Promises)
    modules.SetExport(exports, mod.Name(), "readFile", func(path string) goja.Value {
        return asyncReadFile(vm, bindings, path)
    })
    modules.SetExport(exports, mod.Name(), "writeFile", func(path, data string) goja.Value {
        return asyncWriteFile(vm, bindings, path, data)
    })
    // ... mkdir, readdir, stat, unlink, appendFile, rename, copyFile, exists
    
    // Sync functions (blocking, simple)
    modules.SetExport(exports, mod.Name(), "readFileSync", func(path string) (string, error) {
        b, err := os.ReadFile(path)
        return string(b), err
    })
    modules.SetExport(exports, mod.Name(), "writeFileSync", func(path, data string) error {
        return os.WriteFile(path, []byte(data), 0o644)
    })
    modules.SetExport(exports, mod.Name(), "existsSync", func(path string) bool {
        _, err := os.Stat(path)
        return err == nil
    })
    // ... etc
}
```

**Usage examples**:

```javascript
// Async with await (primary API)
const fs = require("fs");
const content = await fs.readFile("/tmp/hello.txt");
console.log(content);

await fs.mkdir("/tmp/mydir", { recursive: true });
await fs.writeFile("/tmp/mydir/test.txt", "hello world");
const entries = await fs.readdir("/tmp/mydir");
const stat = await fs.stat("/tmp/mydir/test.txt");
console.log(stat.size, stat.isFile);
await fs.rename("/tmp/mydir/test.txt", "/tmp/mydir/renamed.txt");
await fs.unlink("/tmp/mydir/renamed.txt");

// Async with .then/.catch
fs.readFile("/tmp/hello.txt")
  .then(content => console.log(content))
  .catch(err => console.error(err));

// Sync (for simple scripts)
const fs = require("fs");
const content = fs.readFileSync("/tmp/hello.txt");
fs.writeFileSync("/tmp/output.txt", "data");
if (fs.existsSync("/tmp/output.txt")) {
  console.log("file created!");
}
```

### Part B: goja_nodejs Module Integration

The goja_nodejs modules have two layers:

1. **Core module registration** — via `init()` + `require.RegisterCoreModule(...)`. This makes `require("buffer")` etc. work.
2. **Global setup** — via `Enable(vm)` functions. This makes `Buffer`, `process.env`, `URL` available as globals without `require()`.

**Security principle**: `Buffer` and `URL` are harmless type constructors — they don't expose host information. `process` exposes `process.env` which leaks the full host environment. `util` has no `Enable()` function at all.

**The design**:

| Module | `require()` via blank import | `Enable()` globals | Strategy |
|--------|:-:|:-:|---|
| `buffer` | ✅ always | ✅ always (hardcoded) | Harmless — just a constructor |
| `url` | ✅ always | ✅ always (hardcoded) | Harmless — just constructors |
| `console` | ✅ always | ✅ already hardcoded | Already done |
| `util` | ✅ always | N/A (no Enable) | `require("util")` only |
| `process` | ✅ always | ⚠️ **opt-in** via `ProcessEnv()` initializer | Exposes host env |

**Blank imports** go in a dedicated file so they're clearly separated from the `Runtime` struct:

```go
// engine/nodejs_init.go
package engine

import (
    _ "github.com/dop251/goja_nodejs/buffer"
    _ "github.com/dop251/goja_nodejs/process"
    _ "github.com/dop251/goja_nodejs/url"
    _ "github.com/dop251/goja_nodejs/util"
)
```

This ensures all four modules are always `require()`-able. The `init()` functions register them as core modules, so `require("buffer")`, `require("process")`, `require("url")`, `require("util")` always work.

**Always-on globals** in `engine/factory.go` (hardcoded):

```go
reqMod := reg.Enable(vm)
console.Enable(vm)   // already exists
buffer.Enable(vm)    // NEW — always on
url.Enable(vm)       // NEW — always on
rt.Require = reqMod
```

**Opt-in `process.env`** via `RuntimeInitializer`:

```go
// engine/module_specs.go

type processEnvSpec struct{}

func (s processEnvSpec) ID() string { return "process-env" }

func (s processEnvSpec) InitRuntime(ctx *RuntimeContext) error {
    process.Enable(ctx.VM)
    return nil
}

// ProcessEnv returns a RuntimeInitializer that enables the global `process`
// object with `process.env`. This is opt-in because it exposes the host
// environment to JavaScript.
func ProcessEnv() RuntimeInitializer {
    return processEnvSpec{}
}
```

**Callers opt in to `process.env`**:

```go
// Full Node.js compat (including process.env):
factory, _ := engine.NewBuilder().
    WithModules(engine.DefaultRegistryModules()).
    WithRuntimeInitializers(engine.ProcessEnv()).
    Build()

// Safe default (no process.env leak):
factory, _ := engine.NewBuilder().
    WithModules(engine.DefaultRegistryModules()).
    Build()
// Buffer, URL still available as globals
// process.env NOT available as global (but require("process") still works)
```

**What the user experience looks like**:

```javascript
// Always available (no opt-in needed):
const buf = Buffer.from("hello");     // global Buffer
const u = new URL("https://x.com");   // global URL, URLSearchParams
const util = require("util");          // always require-able
const fs = require("fs");              // go-go-goja native module

// Available via require() but NOT as global (unless opted in):
const proc = require("process");      // works, gives { env: {...} }
console.log(process);                  // undefined — unless ProcessEnv() initializer used

// With ProcessEnv() initializer opted in:
console.log(process.env.HOME);         // "/home/user"
```

---

## Detailed Implementation Guide

This section provides file-by-file instructions for implementing both changes. Each step includes the exact file to modify, what to add, and why.

### Phase 1: Rebuild the fs Module as Promise-Based

#### File: `modules/fs/fs.go` (REWRITE)

The main module file. Reduced to just the `NativeModule` interface impl and the `Loader()` that wires everything.

```go
package fs

import (
    "fmt"
    "os"
    "time"

    "github.com/dop251/goja"
    "github.com/go-go-golems/go-go-goja/modules"
    "github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
    "github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

type m struct{}

var _ modules.NativeModule = (*m)(nil)
var _ modules.TypeScriptDeclarer = (*m)(nil)

func (m) Name() string { return "fs" }

func (m) Doc() string { return `...` }

func (m) TypeScriptModule() *spec.Module {
    return &spec.Module{
        Name: "fs",
        Functions: []spec.Function{
            // Async (Promise-based)
            {Name: "readFile", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Named("Promise<string>")},
            {Name: "writeFile", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "data", Type: spec.String()}}, Returns: spec.Named("Promise<void>")},
            {Name: "exists", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Named("Promise<boolean>")},
            {Name: "mkdir", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "options", Type: spec.Optional(spec.Object())}}, Returns: spec.Named("Promise<void>")},
            {Name: "readdir", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Named("Promise<string[]>")},
            {Name: "stat", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Named("Promise<FileStats>")},
            {Name: "unlink", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Named("Promise<void>")},
            {Name: "appendFile", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "data", Type: spec.String()}}, Returns: spec.Named("Promise<void>")},
            {Name: "rename", Params: []spec.Param{{Name: "oldPath", Type: spec.String()}, {Name: "newPath", Type: spec.String()}}, Returns: spec.Named("Promise<void>")},
            {Name: "copyFile", Params: []spec.Param{{Name: "src", Type: spec.String()}, {Name: "dst", Type: spec.String()}}, Returns: spec.Named("Promise<void>")},
            // Sync
            {Name: "readFileSync", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.String()},
            {Name: "writeFileSync", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "data", Type: spec.String()}}, Returns: spec.Void()},
            {Name: "existsSync", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Boolean()},
            {Name: "mkdirSync", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "options", Type: spec.Optional(spec.Object())}}, Returns: spec.Void()},
            {Name: "readdirSync", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Array(spec.String())},
            {Name: "statSync", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Named("FileStats")},
            {Name: "unlinkSync", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Void()},
            {Name: "appendFileSync", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "data", Type: spec.String()}}, Returns: spec.Void()},
            {Name: "renameSync", Params: []spec.Param{{Name: "oldPath", Type: spec.String()}, {Name: "newPath", Type: spec.String()}}, Returns: spec.Void()},
            {Name: "copyFileSync", Params: []spec.Param{{Name: "src", Type: spec.String()}, {Name: "dst", Type: spec.String()}}, Returns: spec.Void()},
        },
    }
}

func (mod m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    bindings, ok := runtimebridge.Lookup(vm)
    if !ok || bindings.Owner == nil {
        panic(vm.NewGoError(fmt.Errorf("fs module requires runtime owner bindings")))
    }

    // Wire async functions
    wireAsyncFunctions(exports, vm, bindings)
    // Wire sync functions
    wireSyncFunctions(exports)
}

func init() {
    modules.Register(&m{})
}
```

#### File: `modules/fs/fs_async.go` (NEW)

Contains all promise-based async function implementations. Each function follows the same pattern:
1. Create a promise with `vm.NewPromise()`
2. Spawn a goroutine that does the blocking I/O
3. Use `bindings.Owner.Post()` to resolve/reject on the owner thread
4. Return the promise immediately

This file will be approximately 250-300 lines.

#### File: `modules/fs/fs_sync.go` (NEW)

Contains all synchronous blocking implementations. These are simple Go functions that call `os.*` directly and return Go types. No promises, no goroutines. Approximately 100-120 lines.

#### File: `modules/fs/fs_test.go` (NEW)

Tests for both async and sync functions. The async tests follow the pattern from `timer_test.go`:

```go
func TestFsAsyncReadFile(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "test.txt")
    os.WriteFile(path, []byte("hello world"), 0644)

    rt := newTestRuntime(t)

    _, err := rt.Owner.Call(context.Background(), "setup", func(_ context.Context, vm *goja.Runtime) (any, error) {
        _, err := vm.RunString(`
            globalThis.__result = { done: false, error: "", value: "" };
            const fs = require("fs");
            fs.readFile("` + path + `")
                .then(v => { globalThis.__result = { done: true, error: "", value: v }; })
                .catch(e => { globalThis.__result = { done: true, error: String(e), value: "" }; });
        `)
        return nil, err
    })
    require.NoError(t, err)

    require.Eventually(t, func() bool {
        val, _ := rt.Owner.Call(context.Background(), "check", func(_ context.Context, vm *goja.Runtime) (any, error) {
            v, _ := vm.RunString(`JSON.stringify(globalThis.__result)`)
            return v.Export(), nil
        })
        s, _ := val.(string)
        return strings.Contains(s, `"done":true`) && strings.Contains(s, `"value":"hello world"`)
    }, 2*time.Second, 10*time.Millisecond)
}

func TestFsSyncReadFile(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "test.txt")
    os.WriteFile(path, []byte("hello"), 0644)

    rt := newTestRuntime(t)
    ret, err := rt.Owner.Call(context.Background(), "test", func(_ context.Context, vm *goja.Runtime) (any, error) {
        val, err := vm.RunString(`require("fs").readFileSync("` + path + `")`)
        if err != nil { return nil, err }
        return val.Export(), nil
    })
    require.NoError(t, err)
    require.Equal(t, "hello", ret)
}
```

### Phase 2: Wire goja_nodejs Modules

#### File: `engine/runtime.go`

Add blank imports for the missing goja_nodejs packages.

**Current** (lines ~13-17):

```go
import (
    _ "github.com/go-go-golems/go-go-goja/modules/database"
    _ "github.com/go-go-golems/go-go-goja/modules/exec"
    _ "github.com/go-go-golems/go-go-goja/modules/fs"
    _ "github.com/go-go-golems/go-go-goja/modules/timer"
)
```

**New** — add four lines:

```go
import (
    _ "github.com/dop251/goja_nodejs/buffer"
    _ "github.com/dop251/goja_nodejs/process"
    _ "github.com/dop251/goja_nodejs/url"
    _ "github.com/dop251/goja_nodejs/util"

    _ "github.com/go-go-golems/go-go-goja/modules/database"
    _ "github.com/go-go-golems/go-go-goja/modules/exec"
    _ "github.com/go-go-golems/go-go-goja/modules/fs"
    _ "github.com/go-go-golems/go-go-goja/modules/timer"
)
```

**What this does**: When the Go binary starts, each of these packages' `init()` functions runs, calling `require.RegisterCoreModule(...)`. This makes the modules available for `require()` resolution.

#### File: `engine/factory.go`

Add `Enable()` calls for the modules that set up globals.

**Current** (in `NewRuntime()`, around line ~120):

```go
reqMod := reg.Enable(vm)
console.Enable(vm)
rt.Require = reqMod
```

**New** — add three `Enable()` calls:

```go
reqMod := reg.Enable(vm)
console.Enable(vm)
buffer.Enable(vm)
process.Enable(vm)
url.Enable(vm)
rt.Require = reqMod
```

**What this does**: Each `Enable()` function not only requires the module (loading it via the now-registered core module) but also sets up the corresponding global variable(s) on the VM. For example, `buffer.Enable(vm)` sets `vm.Buffer` to the Buffer constructor so JavaScript code can use `Buffer.from(...)` without importing.

You will also need to add the import statements (non-blank this time):

```go
import (
    "github.com/dop251/goja_nodejs/buffer"
    "github.com/dop251/goja_nodejs/console"
    "github.com/dop251/goja_nodejs/eventloop"
    "github.com/dop251/goja_nodejs/process"
    "github.com/dop251/goja_nodejs/require"
    "github.com/dop251/goja_nodejs/url"
    // ... other imports
)
```

#### File: `engine/factory_test.go` (or a new test file)

Add integration tests that verify each goja_nodejs module is `require()`-able:

```go
func TestRequireBuffer(t *testing.T) {
    factory := newTestFactory(t)
    rt := newTestRuntime(t, factory)
    defer rt.Close(context.Background())

    _, err := rt.Owner.Call(context.Background(), "test", func(_ context.Context, vm *goja.Runtime) (any, error) {
        _, err := vm.RunString(`require("buffer")`)
        return nil, err
    })
    require.NoError(t, err)
}

func TestRequireProcess(t *testing.T) { /* require("process") works */ }
func TestRequireUrl(t *testing.T)    { /* require("url") works */ }
func TestRequireUtil(t *testing.T)   { /* require("util") works */ }

func TestBufferGlobal(t *testing.T) {
    /* Verify Buffer is a global, e.g., Buffer.from("hello") works */
}

func TestProcessEnvGlobal(t *testing.T) {
    /* Verify process.env is a global object */
}

func TestURLGlobal(t *testing.T) {
    /* Verify new URL("https://example.com") works as a global */ }
```

### Phase 3: Validate End-to-End

After implementing both phases, run this validation script (as a test or manually):

```go
// This should work end-to-end:
vm.RunString(`
    // fs module — promise-based (primary API)
    const fs = require("fs");
    
    // Async with await
    const content = await fs.readFile("/tmp/hello.txt");
    console.log("Read:", content);
    
    await fs.mkdir("/tmp/goja-test", { recursive: true });
    await fs.writeFile("/tmp/goja-test/hello.txt", "world");
    console.log("Exists:", await fs.exists("/tmp/goja-test/hello.txt"));
    
    const entries = await fs.readdir("/tmp/goja-test");
    console.log("Entries:", entries);
    
    const stat = await fs.stat("/tmp/goja-test/hello.txt");
    console.log("Size:", stat.size);
    
    await fs.appendFile("/tmp/goja-test/hello.txt", "!");
    await fs.rename("/tmp/goja-test/hello.txt", "/tmp/goja-test/renamed.txt");
    await fs.copyFile("/tmp/goja-test/renamed.txt", "/tmp/goja-test/copy.txt");
    await fs.unlink("/tmp/goja-test/copy.txt");

    // Sync variants still work too
    fs.writeFileSync("/tmp/goja-test/sync.txt", "sync data");
    const syncContent = fs.readFileSync("/tmp/goja-test/sync.txt");
    console.log("Sync:", syncContent);
    console.log("ExistsSync:", fs.existsSync("/tmp/goja-test/sync.txt"));

    // goja_nodejs modules
    const buf = Buffer.from("hello");    // Buffer global
    console.log("Buffer:", buf.toString());
    console.log("PATH:", process.env.PATH);  // process global
    const u = new URL("https://example.com"); // URL global
    console.log("URL:", u.href);
    const util = require("util");
    console.log("Format:", util.format("%s says %d", "Alice", 42));
`)
```

---

## Testing and Validation Strategy

### Unit Tests (per-module)

Every new function in the `fs` module gets its own test function. Tests should:

1. Use `t.TempDir()` for all file operations (automatic cleanup on test end)
2. Create a go-go-goja factory and runtime for each test (isolation)
3. Execute the JavaScript that calls the function under test
4. Verify the result both from JavaScript return values AND by checking actual file system state from Go
5. Test error cases: non-existent paths, permission errors (where feasible), invalid arguments

### Integration Tests (cross-module)

Test that goja_nodejs modules work together with go-go-goja modules:

1. A script that uses `fs` to write a file, then `Buffer` to read it back as binary
2. A script that uses `process.env` to get a temp directory, then `fs` to create a file there
3. A script that uses `url.URL` to parse a file:// URL and `fs` to stat the path

### Validation Commands

After implementation, run:

```bash
# Build everything
cd /home/manuel/workspaces/2026-04-25/add-primitive-modules/go-go-goja
go build ./...

# Run all tests (workspace-aware)
go test ./... -count=1

# Run only fs module tests
go test ./modules/fs/... -v -count=1

# Run engine tests (to verify goja_nodejs wiring)
go test ./engine/... -v -count=1

# Run without workspace (CI mode)
GOWORK=off go test ./... -count=1

# Lint
golangci-lint run -v
```

### Manual Validation via REPL

If go-go-goja has a REPL command, start it and manually test:

```javascript
// In the REPL:
require("fs").existsSync("/tmp")
// Should return: true

require("buffer")
// Should return: { Buffer: [Function: Buffer], ... }

Buffer.from("hello").toString()
// Should return: "hello"

process.env.HOME
// Should return: "/home/manuel" (or similar)

new URL("https://example.com/path?q=1").searchParams.get("q")
// Should return: "1"
```

---

## Risks, Alternatives, and Open Questions

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|:-:|:-:|------------|
| Blank imports cause init-order issues | Low | Medium | Go guarantees init order within a package is deterministic; cross-package order follows import dependency order |
| `buffer.Enable()` conflicts with existing console.Enable() | Low | Low | These are independent globals; no known conflict |
| `mkdirSync` options parsing is fragile | Low | Low | Use `goja.FunctionCall` and check for undefined/nil carefully; write tests for all combinations |
| `process.Enable()` leaks env vars to JS | Expected | Medium | This is intentional — Node.js exposes `process.env`. Document this behavior. If security is a concern, future work can add env filtering. |
| `statSync` mode field loses precision (uint32 vs int64) | Low | Low | Go file modes fit in uint32; `int64` from `ToInteger()` is fine |
| No `node:` prefix for go-go-goja native modules | N/A | Low | The `fs`, `exec`, `timer`, `database` modules are go-go-goja-specific, not Node.js builtins. They should NOT use the `node:` prefix. |

### Alternatives Considered

1. **Sync-only fs module** — Could keep all functions synchronous, which is simpler to implement. Rejected because:
   - The runtime already has full promise support (timer module proves it works)
   - The REPL already unwraps `await` and polls promises
   - Async I/O is the modern convention and matches what users expect from Node.js
   - Sync wrappers are still provided as a convenience layer

2. **Full `fs.Stats` class instead of plain object** — Could create a proper JavaScript class with methods like `isFile()`, `isDirectory()`, etc. Rejected for v1 because:
   - Requires significant prototype setup in Go
   - Plain objects are easier to inspect and log
   - Can be migrated later

3. **Separate `fs-sync` and `fs-async` modules** — Could split into two modules. Rejected because:
   - Confusing for users
   - Node.js puts them both in the `fs` module
   - Both sets are provided from the same `require("fs")` import

4. **Register goja_nodejs modules in a dedicated registrar** — Instead of blank imports in `runtime.go`, create a `RuntimeModuleRegistrar` implementation. Rejected because:
   - The core modules are global (registered in `init()`), not per-registry
   - Blank imports are the established goja_nodejs pattern
   - A registrar would add complexity for no benefit

### Open Questions

1. **Should `readdirSync` return objects with type info?** Node.js `readdirSync` with `{ withFileTypes: true }` returns `Dirent` objects. For v1, we return just names. Should we add `withFileTypes` support?

2. **Should `readFileSync` support encoding parameter?** Currently returns string only. Node.js defaults to Buffer but can return string with `"utf8"` encoding. Should we add `readFileSync(path, encoding?)` overload?

3. **Should `url.Enable()` set globals?** The `url` module's `Enable()` function sets `URL` and `URLSearchParams` as globals. This is technically not standard Node.js behavior (Node.js doesn't set these globals — they come from the web standard). Should we still enable them for convenience?

4. **Should `util.Enable()` be added?** The `util` module has no `Enable()` function in goja_nodejs. Should we add one (e.g., setting `util.format` as a global) or leave it as require-only?

---

## Key File References

| File | Role | Action Needed |
|------|------|---------------|
| `modules/fs/fs.go` | Current fs module implementation | **Rewrite** — module def + Loader only |
| `modules/fs/fs_async.go` | Async promise-based functions | **Create** — 10 async functions |
| `modules/fs/fs_sync.go` | Sync blocking functions | **Create** — 10 sync functions |
| `modules/fs/fs_test.go` | Tests for fs module | **Create** — new test file |
| `modules/common.go` | NativeModule interface, Registry | No change |
| `modules/exports.go` | SetExport helper | No change |
| `modules/typing.go` | TypeScriptDeclarer interface | No change |
| `engine/runtime.go` | Blank imports for module init() | **Edit** — add 4 blank imports |
| `engine/factory.go` | Runtime creation (Enable calls) | **Edit** — add 3 Enable() calls |
| `engine/module_specs.go` | ModuleSpec, DefaultRegistryModules() | No change |
| `engine/options.go` | Builder settings | No change |
| `goja_nodejs/require/module.go` | Core module registration system | Reference only (external dep) |
| `goja_nodejs/require/resolve.go` | Module resolution algorithm | Reference only |
| `goja_nodejs/buffer/buffer.go` | Buffer module + init() | Reference only |
| `goja_nodejs/console/module.go` | Console module + init() | Reference only |
| `goja_nodejs/process/module.go` | Process module + init() | Reference only |
| `goja_nodejs/url/module.go` | URL module + init() | Reference only |
| `goja_nodejs/util/module.go` | Util module + init() | Reference only |
| `goja_nodejs/eventloop/eventloop.go` | Event loop | Reference only |
| `goja_nodejs/errors/errors.go` | Error helpers | Reference only |
| `pkg/runtimebridge/runtimebridge.go` | Per-VM bindings storage | Reference only |
| `pkg/runtimeowner/types.go` | Runner interface | Reference only |
| `pkg/tsgen/spec/` | TypeScript declaration types | Reference only |
| `go-go-goja/go.mod` | Module dependencies | No change (goja_nodejs already a dep) |
