---
Title: Node.js Primitives in go-go-goja
Slug: nodejs-primitives
Short: Reference for the built-in Node.js-style primitives exposed to JavaScript runtimes.
Topics:
- goja
- javascript
- nodejs
- modules
- primitives
Commands:
- goja-repl
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

go-go-goja provides a practical subset of Node.js-style primitives so JavaScript scripts can do useful work without writing a custom Go adapter for every task. These primitives cover file I/O, path handling, operating-system information, hashing/randomness, timing, and selected `goja_nodejs` built-ins such as `Buffer`, `URL`, and `util`.

The goal is not to clone Node.js completely. The goal is to provide the primitives most scripts need while keeping host exposure explicit. For example, `Buffer` and `URL` are safe globals, but both the `process` module and the global `process` object are opt-in because `process.env` exposes host environment data.

## Runtime Composition Model

This section explains how primitives become available in a runtime, how the pieces fit together, and why caller-controlled composition matters.

A go-go-goja runtime starts with goja, then adds the `goja_nodejs/require` registry, the event loop, and selected native modules. Modules under `modules/` implement `modules.NativeModule` and register themselves into `modules.DefaultRegistry` through `init()` functions.

The factory always installs data-only primitives and safe globals:

- `console`, from `goja_nodejs/console`
- `Buffer`, from `goja_nodejs/buffer`
- `URL` and `URLSearchParams`, from `goja_nodejs/url`
- `performance.now()`, implemented by go-go-goja
- `require("crypto")`
- `require("path")`
- `require("time")`
- `require("timer")`

Host-access modules are not enabled by default. Enable them one at a time with `engine.DefaultRegistryModule(name)` or in a selected group with `engine.DefaultRegistryModulesNamed(...)`:

```go
factory, err := engine.NewBuilder().
    WithModules(
        engine.DefaultRegistryModule("fs"),
        engine.DefaultRegistryModule("os"),
    ).
    Build()
```

You can still enable every module in `modules.DefaultRegistry` with `engine.DefaultRegistryModules()`, but that includes host-access modules such as `fs`, `os`, `exec`, and `database`. Use it only when the JavaScript code is trusted enough for that level of access.

The `process` module and global are not installed by default. Enable them only when exposing host environment variables is acceptable:

```go
factory, err := engine.NewBuilder().
    WithModules(engine.ProcessModule()).
    WithRuntimeInitializers(engine.ProcessEnv()).
    Build()
```

Use only `ProcessEnv()` if scripts need the global `process` object but not `require("process")`. Use only `ProcessModule()` if scripts should be able to call `require("process")` but should not receive a global `process` binding.

## Embedding from Third-Party Packages

This section shows how an application or library outside this repository should construct a runtime with the primitive modules enabled. The key idea is that module registration and runtime construction are explicit: import the `engine` package, build a factory, and opt in to the module set you want.

A minimal runtime already includes data-only primitives such as `crypto`, `path`, `time`, and `timer`:

```go
package myruntime

import (
    "context"

    "github.com/go-go-golems/go-go-goja/engine"
)

func NewRuntime(ctx context.Context) (*engine.Runtime, error) {
    factory, err := engine.NewBuilder().Build()
    if err != nil {
        return nil, err
    }

    return factory.NewRuntime(ctx)
}
```

If your package wants file and OS access, opt in explicitly:

```go
factory, err := engine.NewBuilder().
    WithModules(engine.DefaultRegistryModulesNamed("fs", "os")).
    Build()
```

For a single module, use:

```go
factory, err := engine.NewBuilder().
    WithModules(engine.DefaultRegistryModule("fs")).
    Build()
```

The engine package already contains the blank imports that make built-in modules register themselves, so third-party packages do not need to blank-import each module manually.

If the application wants process environment access, opt in explicitly. For both `require("process")` and the global `process` object, use:

```go
factory, err := engine.NewBuilder().
    WithModules(
        engine.DefaultRegistryModule("fs"),
        engine.ProcessModule(),
    ).
    WithRuntimeInitializers(engine.ProcessEnv()).
    Build()
```

Use `ProcessEnv()` only when JavaScript should see host environment variables through global `process.env`. Use `ProcessModule()` when JavaScript should be able to import the same host environment capability with `require("process")`. Without these opt-ins, both global `process` and `require("process")` are unavailable.

A runtime created this way can execute scripts such as:

```javascript
const fs = require("fs");
const path = require("path");
const crypto = require("crypto");

const file = path.join("/tmp", crypto.randomUUID() + ".txt");
fs.writeFileSync(file, Buffer.from("hello"));
console.log(fs.readFileSync(file, "utf8"));
```

For a tighter sandbox, do not use `DefaultRegistryModules()`. Instead, register only the modules your application wants through `DefaultRegistryModule`, `DefaultRegistryModulesNamed`, or explicit `engine.NativeModuleSpec` values.

## Available Primitives

This section lists the current built-in primitives, what each one is for, and how scripts should use them in practice.

| Primitive | How to access it | Purpose | Notes |
|-----------|------------------|---------|-------|
| `Buffer` | global, `require("buffer")` | Binary data | Installed globally by default. |
| `URL`, `URLSearchParams` | global, `require("url")` | URL parsing | Installed globally by default. |
| `util` | `require("util")` | Formatting helpers | Provided by goja_nodejs. |
| `process` | opt-in `require("process")` with `engine.ProcessModule()`; opt-in global with `engine.ProcessEnv()` | Environment variables | Both module and global are opt-in. |
| `fs` | opt-in `require("fs")` | Promise-based and sync file I/O | Host filesystem access; enable explicitly. |
| `path` | default `require("path")` | Host-platform path helpers | Data-only; uses Go `filepath`; no `posix`/`win32` split yet. |
| `os` | opt-in `require("os")` | Host OS information | Host info access; enable explicitly. |
| `crypto` | default `require("crypto")` | UUIDs, random bytes, basic hashes | Data-only default primitive. |
| `time` | default `require("time")` | Explicit timing helper | Data-only; pairs with global `performance.now()`. |
| `performance` | global | Monotonic elapsed timing | Provides `performance.now()`. |
| `console.time*` | global `console` | Quick timing logs | Adds `time`, `timeLog`, and `timeEnd`. |

## File System APIs

The `fs` module is async-first but also exposes sync helpers for scripts that intentionally block the runtime. The async helpers return Promises and should be used with `await` in interactive or long-running scripts.

```javascript
const fs = require("fs");

await fs.mkdir("/tmp/goja-demo", { recursive: true });
await fs.writeFile("/tmp/goja-demo/message.txt", "hello", { encoding: "utf8" });

const asBuffer = await fs.readFile("/tmp/goja-demo/message.txt");
console.log(asBuffer.toString());

const asString = await fs.readFile("/tmp/goja-demo/message.txt", "utf8");
console.log(asString);
```

Reads return `Buffer` by default and strings when an encoding is supplied. Writes and appends accept strings, Buffers, TypedArrays, and DataViews. This matches the most common Node.js workflows and avoids accidental string conversion of binary data.

Useful operations include:

```javascript
await fs.exists(path);
await fs.readdir(path);
await fs.stat(path);
await fs.appendFile(path, Buffer.from("more"));
await fs.copyFile(src, dst);
await fs.rename(oldPath, newPath);
await fs.unlink(path);
await fs.rm(path, { recursive: true, force: true });
```

Sync variants use the same names with `Sync` suffix:

```javascript
const fs = require("fs");

fs.writeFileSync("/tmp/goja-demo/sync.txt", Buffer.from("sync"));
const buf = fs.readFileSync("/tmp/goja-demo/sync.txt");
const text = fs.readFileSync("/tmp/goja-demo/sync.txt", { encoding: "utf8" });
```

When a filesystem operation fails, go-go-goja throws or rejects with an Error object that includes common Node-style fields:

```javascript
try {
  fs.readFileSync("/tmp/does-not-exist");
} catch (err) {
  console.log(err.code);    // ENOENT
  console.log(err.path);    // /tmp/does-not-exist
  console.log(err.syscall); // open
}
```

## Path APIs

The `path` module helps scripts build and inspect filesystem paths without hand-writing separators. It uses Go's `path/filepath`, so it follows the host platform instead of forcing POSIX behavior.

```javascript
const path = require("path");

const file = path.join("/tmp", "goja-demo", "message.txt");
console.log(path.dirname(file));
console.log(path.basename(file));
console.log(path.extname(file));
console.log(path.isAbsolute(file));
console.log(path.relative("/tmp", file));
console.log(path.separator);
console.log(path.delimiter);
```

Use `path` whenever a script constructs filenames that will be passed to `fs`. This keeps scripts portable across Unix-like and Windows hosts. If a script needs deterministic POSIX or Windows semantics independent of the host, that is not implemented yet; add `path.posix`/`path.win32` support before relying on that behavior.

## OS APIs

The `os` module exposes a small subset of host operating-system information. It is useful for locating home directories, temporary directories, and host metadata without exposing the full `process` global.

```javascript
const os = require("os");

console.log(os.homedir());
console.log(os.tmpdir());
console.log(os.platform());
console.log(os.arch());
console.log(os.hostname());
console.log(os.cpus().length);
console.log(os.EOL);
```

The current `release()` and `type()` implementations are pragmatic values based on Go runtime information. They are good enough for broad branching logic but are not exact Node.js clones.

## Crypto APIs

The `crypto` module provides the small set of cryptographic primitives that scripts commonly need for IDs, random data, and checksums.

```javascript
const crypto = require("crypto");

const id = crypto.randomUUID();
const bytes = crypto.randomBytes(16);
const sha = crypto.createHash("sha256")
  .update("hello")
  .digest("hex");
```

Supported hash algorithms are:

- `md5`
- `sha1`
- `sha256`
- `sha512`

`digest()` returns a Buffer by default. `digest("hex")` and `digest("base64")` return strings.

This subset intentionally avoids streaming classes, keys, ciphers, signatures, and other advanced Node crypto APIs. Add those only when a concrete script or package requires them.

## Timing APIs

Timing primitives let scripts measure their own performance from JavaScript. Use `performance.now()` for high-resolution elapsed milliseconds, `console.time*` for quick logs, or `require("time")` when you prefer explicit imports.

```javascript
const t0 = performance.now();
await fs.readFile("/tmp/goja-demo/message.txt");
console.log(`read took ${performance.now() - t0}ms`);
```

```javascript
console.time("work");
for (let i = 0; i < 100000; i++) {}
console.timeLog("work");
console.timeEnd("work");
```

```javascript
const time = require("time");
const start = time.now();
// ... work ...
console.log(time.since(start));
```

The runtime uses monotonic elapsed time from Go's `time` package, so measurements are appropriate for durations. Do not use these values as wall-clock timestamps.

## Security and Sandboxing Notes

These primitives expose useful host capabilities. That is powerful, but it means embedders need a clear sandbox policy.

- `fs` and `os` expose host filesystem and OS details and must be enabled explicitly.
- `path`, `time`, `timer`, and `crypto` are enabled by default as data-only primitives.
- `crypto.randomBytes()` uses host randomness.
- `require("process").env` requires explicit `engine.ProcessModule()` opt-in, and global `process` requires explicit `engine.ProcessEnv()` opt-in.
- `exec` and `database` remain selectable modules and should be treated as more sensitive than the data-only primitives documented here.

If your application runs untrusted JavaScript, do not blindly expose `DefaultRegistryModules()`. Compose a smaller registry with `DefaultRegistryModule(...)` or `DefaultRegistryModulesNamed(...)` before evaluating untrusted scripts.

## Implementation Map

This section maps the JavaScript APIs back to Go files so maintainers can review behavior quickly.

| API | Main files |
|-----|------------|
| Node core registration | `engine/nodejs_init.go` |
| Global Buffer/URL/performance install | `engine/factory.go`, `engine/performance.go` |
| Optional process module/global | `engine/module_specs.go`, `ProcessModule()`, `ProcessEnv()` |
| fs | `modules/fs/fs.go`, `fs_async.go`, `fs_sync.go`, `fs_errors.go` |
| path | `modules/path/path.go` |
| os | `modules/os/os.go` |
| crypto | `modules/crypto/crypto.go` |
| time | `modules/time/time.go` |

Smoke tests live next to each module and execute real JavaScript through a real runtime. Prefer adding tests there before changing module behavior.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `require("fs")` fails | `fs` is host filesystem access and is not enabled by default | Add `.WithModules(engine.DefaultRegistryModule("fs"))` or `.WithModules(engine.DefaultRegistryModulesNamed("fs", ...))`. |
| `process` is undefined | Global `process` is opt-in | Add `.WithRuntimeInitializers(engine.ProcessEnv())` if exposing global `process.env` is acceptable. |
| `require("process")` fails | The process module is opt-in because it exposes host environment variables | Add `.WithModules(engine.ProcessModule())` only if scripts should be able to import `process.env`. |
| `fs.readFile(path)` returns a Buffer, not a string | Node-style default read behavior | Pass an encoding: `await fs.readFile(path, "utf8")`. |
| `Buffer.isBuffer` is missing | goja_nodejs Buffer does not implement every Node helper | Check Buffer-like behavior with `length`, `toString()`, or add a helper if a package requires it. |
| `path` behavior differs from POSIX examples | `path` uses host `filepath` | Add/use future `path.posix` for host-independent POSIX behavior. |
| `crypto.createHash("algorithm")` fails | Only a small algorithm set is implemented | Use `md5`, `sha1`, `sha256`, or `sha512`, or extend `modules/crypto`. |

## See Also

- `glaze help introduction` for the runtime overview.
- `glaze help creating-modules` for the native module implementation pattern.
- `glaze help async-patterns` for Promise settlement and owner-thread rules.
- `glaze help repl-usage` for interactive JavaScript evaluation.
