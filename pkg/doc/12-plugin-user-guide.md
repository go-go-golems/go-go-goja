---
Title: "Using HashiCorp Plugins with go-go-goja"
Slug: goja-plugin-user-guide
Short: "User guide and surface reference for loading external HashiCorp go-plugin modules into go-go-goja runtimes."
Topics:
- goja
- plugins
- hashicorp
- repl
- javascript
- modules
Commands:
- repl
- js-repl
Flags:
- --plugin-dir
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This page explains how plugin-backed JavaScript modules work from a user point of view.

The short version is simple: a plugin is a separate Go binary that speaks the HashiCorp `go-plugin` protocol, publishes a manifest that describes one JavaScript module, and responds to function calls from the host runtime. The host `goja` runtime stays inside `go-go-goja`. The plugin does not own a `goja.Runtime`; it only exposes a process boundary and an RPC surface.

That distinction matters because it tells you what to expect operationally:

- You still use `require("plugin:...")` inside JavaScript.
- The host process still owns module loading, JavaScript execution, and runtime shutdown.
- Plugin modules are loaded from directories you explicitly trust.
- Closing the runtime shuts down the plugin subprocesses that were started for that runtime.

## What a plugin looks like to a JavaScript user

From JavaScript, a plugin-backed module looks like a normal native module:

```javascript
const echo = require("plugin:echo")

echo.ping("hello")
echo.math.add(2, 3)
echo.pid()
```

The module name is the manifest module name published by the plugin. In the current design, plugin modules are expected to live in the `plugin:` namespace, so `plugin:echo` is valid and `echo` is rejected.

The exports currently supported by the host are:

- Top-level functions such as `echo.ping(...)`
- Top-level objects containing methods such as `echo.math.add(...)`

The first implementation intentionally keeps the export surface small. That keeps plugin manifests easy to validate and makes it obvious how a plugin export maps onto a JavaScript call site.

## Quick start

This section shows the smallest working flow for testing a plugin manually.

### 1. Build a plugin binary

From the repository root:

```bash
mkdir -p ~/.go-go-goja/plugins/examples
go build -o ~/.go-go-goja/plugins/examples/goja-plugin-greeter ./plugins/examples/greeter
```

The binary name matters. Discovery uses the pattern `goja-plugin-*` by default, so `goja-plugin-greeter` is picked up automatically.

### 2. Start the line REPL with plugin discovery enabled

```bash
go run ./cmd/repl
```

At runtime, `repl` scans `~/.go-go-goja/plugins/...` by default, starts matching plugin binaries through `go-plugin`, validates their manifests, and registers them as runtime-scoped CommonJS modules.

If you want to use a different location for one run, pass one or more explicit flags:

```bash
go run ./cmd/repl --plugin-dir /tmp/goja-plugins
```

### 3. Require the module in JavaScript

```javascript
let greeter = require("plugin:greeter")
greeter.greet("hello")
greeter.strings.upper("hello")
greeter.meta.pid()
```

Expected results:

- `greeter.greet("hello")` returns `"hello, hello"`
- `greeter.strings.upper("hello")` returns `"HELLO"`
- `greeter.meta.pid()` returns the plugin subprocess PID

### 4. Exit the runtime

When you leave the REPL or the runtime is closed, the plugin subprocesses created for that runtime are shut down by runtime cleanup hooks.

## Running a script instead of using the interactive REPL

You can also execute a JavaScript file once, which is often easier for repeatable testing.

Create a script:

```javascript
const greeter = require("plugin:greeter")

console.log(greeter.greet("hello"))
console.log(greeter.strings.upper("hello"))
console.log(greeter.meta.pid())
```

Then run it:

```bash
go run ./cmd/repl /tmp/test-plugin.js
```

This is the best path when you want:

- deterministic repro steps,
- a fixture for a bug report,
- a smoke test during plugin development,
- a quick CI-style sanity check outside the Go test suite.

## Current command-line entrypoints

There are currently two REPL binaries in the repo, and both now expose the same plugin discovery model.

### `repl`

`repl` is the current user-facing line REPL and script runner for plugin testing.

It supports:

- `--plugin-dir`
- `--plugin-status`
- direct script execution
- interactive line-based evaluation

Example:

```bash
go run ./cmd/repl
```

### `js-repl`

`js-repl` is the Bobatea TUI REPL with completion and help widgets.

It follows the same plugin discovery rules as `repl`:

- default scan under `~/.go-go-goja/plugins/...`
- optional `--plugin-dir` flags for explicit directories
- `--plugin-status` for one-shot discovery/load reporting without entering the UI

Example:

```bash
go run ./cmd/js-repl
```

## Plugin discovery rules

This section explains how the host decides which binaries count as plugins.

By default, the host-side plugin config uses:

- discovery pattern: `goja-plugin-*`
- namespace prefix: `plugin:`
- gRPC transport only

The host scans the configured directories and filters for regular executable files that match the discovery pattern. For each candidate, it:

1. starts a `go-plugin` client,
2. dispenses the shared module service,
3. asks for the module manifest,
4. validates the manifest,
5. reifies the described exports into the runtime's `require.Registry`.

If any plugin has an invalid manifest, runtime creation fails early instead of partially registering a broken module set.

## Discovery visibility in the REPLs

The current REPL surfaces expose plugin visibility in two ways.

In `repl`:

- startup prints a short plugin summary when plugin directories are configured,
- `:plugins` prints the full discovery and load report,
- `--plugin-status` prints the report and exits.

In `js-repl`:

- the placeholder includes a short plugin summary when directories are configured,
- `--plugin-status` prints the same report and exits without starting the TUI.

## Surface API reference

This section is the user-facing contract for what a plugin may expose to JavaScript.

### Module name

- Must be non-empty
- Must begin with `plugin:`
- Must be unique within the runtime

Examples:

- valid: `plugin:echo`
- valid: `plugin:greeter`
- valid: `plugin:dbtools`
- invalid: `echo`
- invalid: `fs`

### Export kinds

Current export kinds:

- `FUNCTION`
- `OBJECT`

Rules:

- A `FUNCTION` export must not declare methods.
- An `OBJECT` export must declare one or more method names.
- Export names must be unique within the module.
- Object method names must be unique within the object export.

### Invocation model

Every JavaScript call is forwarded over RPC as:

- module export name,
- optional object method name,
- argument list converted into protobuf `structpb.Value` values.

The result comes back as one protobuf value and is converted back into a JavaScript value by the host runtime.

That means the most reliable data shapes are the ones that already round-trip cleanly through JSON-like values:

- strings,
- numbers,
- booleans,
- arrays,
- objects,
- `null`.

## Authoring rules for plugin users

If you are building your own plugin binary, follow these rules first before you optimize anything else.

- Use a binary name that matches `goja-plugin-*`.
- Publish a module name in the `plugin:` namespace.
- Keep arguments and return values JSON-like.
- Prefer small, explicit function signatures.
- Use object exports only when you want a real namespace such as `plugin:foo.math.add`.

These rules are not arbitrary. They align with how the first host implementation validates manifests and how it currently marshals values across the process boundary.

## Security and trust model

This feature is process isolation, not sandboxing.

Loading a plugin means executing a local Go binary that you selected via `--plugin-dir`. The plugin runs outside the host process, which is useful for isolation and lifecycle control, but it is still native code you are choosing to execute. Treat plugin directories as trusted execution inputs.

What the current system does provide:

- explicit directory opt-in,
- manifest validation,
- namespace validation,
- runtime-scoped lifecycle,
- gRPC-only transport.

What it does not currently provide by default:

- checksums,
- signatures,
- automatic provenance verification,
- an opinionated allowlist policy.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `Cannot find module 'plugin:greeter'` | The plugin binary was not discovered or the manifest was rejected | Confirm the binary is named `goja-plugin-greeter`, placed in the configured directory, and that you passed `--plugin-dir /path/to/dir` |
| Runtime creation fails with `must use namespace` | The plugin manifest published a module name outside `plugin:` | Update the plugin manifest to use a name like `plugin:echo` |
| The plugin binary exists but still is not loaded | The file is not executable or does not match the discovery pattern | Run `chmod +x` if needed and keep the binary name under `goja-plugin-*` |
| Calls fail on argument conversion | The JS values do not cleanly round-trip through protobuf `structpb.Value` | Use JSON-like values and avoid host-specific Goja objects/functions as arguments |
| `js-repl` does not see plugins | The plugin was not built under the default tree and no explicit directory was passed | Build into `~/.go-go-goja/plugins/...` or pass one or more `--plugin-dir` flags |

## See Also

- `repl help plugin-tutorial-build-install` — Step-by-step tutorial for building and installing a plugin
- `repl help goja-plugin-developer-guide` — Internal architecture and integration guide
- `repl help repl-usage` — General REPL usage
- `repl help creating-modules` — Native in-process module authoring guide
