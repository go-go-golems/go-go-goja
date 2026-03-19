# Plugin examples

This directory contains user-facing example plugins for `go-go-goja`.

These examples are different from `plugins/testplugin/...`:

- `plugins/examples/...` is meant to be read, copied, and built manually by plugin authors.
- `plugins/testplugin/...` is meant to stay small and stable as an integration-test fixture.

## Example catalog

| Example | Module name | What it demonstrates |
|---|---|---|
| `greeter` | `plugin:examples:greeter` | Baseline SDK shape: metadata, top-level functions, object methods |
| `clock` | `plugin:examples:clock` | Zero-argument calls, metadata, and structured object returns |
| `validator` | `plugin:examples:validator` | `sdk.Call` helpers, defaults, map/slice arguments, and validation errors |
| `kv` | `plugin:examples:kv` | Stateful object methods inside one plugin subprocess |
| `system-info` | `plugin:examples:system-info` | Mixed export shapes and nested JSON-like responses |
| `failing` | `plugin:examples:failing` | Explicit handler errors and host-visible failure behavior |

## Build one example

From the repository root:

```bash
mkdir -p ~/.go-go-goja/plugins/examples
go build -o ~/.go-go-goja/plugins/examples/goja-plugin-examples-greeter ./plugins/examples/greeter
```

Replace `greeter` with `clock`, `validator`, `kv`, `system-info`, or `failing` to build a different example.

You can also build several in one pass:

```bash
make install-modules
```

Then start either REPL:

```bash
go run ./cmd/repl
# or
go run ./cmd/js-repl
```

## Quick JavaScript probes

```javascript
const greeter = require("plugin:examples:greeter")
greeter.greet("Manuel")
greeter.strings.upper("hello")

const clock = require("plugin:examples:clock")
clock.now()

const validator = require("plugin:examples:validator")
validator.grade(0.9, true)

const kv = require("plugin:examples:kv")
kv.store.set("name", "Manuel")
kv.store.get("name")

const systemInfo = require("plugin:examples:system-info")
systemInfo.runtime.snapshot()

const failing = require("plugin:examples:failing")
failing.sometimes("fail")
```

## Which one to copy first

- Start with `greeter` if you want the smallest clean authoring template.
- Start with `validator` if your plugin mostly validates user input.
- Start with `kv` if your plugin needs process-local state.
- Start with `system-info` if your plugin returns richer nested objects.
- Read `failing` if you want to understand how handler errors surface back to JavaScript.

All of these examples use the richer SDK surface:

- `sdk.MustModule(...)`,
- `sdk.Function(...)`,
- `sdk.Object(...sdk.Method(...))`,
- `sdk.Call`,
- `sdk.Serve(...)`.
