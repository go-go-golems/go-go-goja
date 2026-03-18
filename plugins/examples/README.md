# Plugin examples

This directory contains user-facing example plugins for `go-go-goja`.

These examples are different from `plugins/testplugin/...`:

- `plugins/examples/...` is meant to be read, copied, and built manually by plugin authors.
- `plugins/testplugin/...` is meant to stay small and stable as an integration-test fixture.

## Example catalog

| Example | Module name | What it demonstrates |
|---|---|---|
| `greeter` | `plugin:greeter` | Baseline SDK shape: metadata, top-level functions, object methods |
| `clock` | `plugin:clock` | Zero-argument calls, metadata, and structured object returns |
| `validator` | `plugin:validator` | `sdk.Call` helpers, defaults, map/slice arguments, and validation errors |
| `kv` | `plugin:kv` | Stateful object methods inside one plugin subprocess |
| `system-info` | `plugin:system-info` | Mixed export shapes and nested JSON-like responses |
| `failing` | `plugin:failing` | Explicit handler errors and host-visible failure behavior |

## Build one example

From the repository root:

```bash
mkdir -p ~/.go-go-goja/plugins/examples
go build -o ~/.go-go-goja/plugins/examples/goja-plugin-greeter ./plugins/examples/greeter
```

Replace `greeter` with `clock`, `validator`, `kv`, `system-info`, or `failing` to build a different example.

You can also build several in one pass:

```bash
mkdir -p ~/.go-go-goja/plugins/examples
for name in greeter clock validator kv system-info failing; do
  go build -o ~/.go-go-goja/plugins/examples/goja-plugin-${name} ./plugins/examples/${name}
done
```

Then start either REPL:

```bash
go run ./cmd/repl
# or
go run ./cmd/js-repl
```

## Quick JavaScript probes

```javascript
const greeter = require("plugin:greeter")
greeter.greet("Manuel")
greeter.strings.upper("hello")

const clock = require("plugin:clock")
clock.now()

const validator = require("plugin:validator")
validator.grade(0.9, true)

const kv = require("plugin:kv")
kv.store.set("name", "Manuel")
kv.store.get("name")

const systemInfo = require("plugin:system-info")
systemInfo.runtime.snapshot()

const failing = require("plugin:failing")
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
