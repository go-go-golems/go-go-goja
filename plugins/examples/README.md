# Plugin examples

This directory contains user-facing example plugins for `go-go-goja`.

These examples are different from `plugins/testplugin/...`:

- `plugins/examples/...` is meant to be read, copied, and built manually by plugin authors.
- `plugins/testplugin/...` is meant to stay small and stable as an integration-test fixture.

## Build the greeter example

From the repository root:

```bash
mkdir -p ~/.go-go-goja/plugins/examples
go build -o ~/.go-go-goja/plugins/examples/goja-plugin-greeter ./plugins/examples/greeter
```

Then start either REPL:

```bash
go run ./cmd/repl
# or
go run ./cmd/js-repl
```

And call it from JavaScript:

```javascript
const greeter = require("plugin:greeter")
greeter.greet("Manuel")
greeter.strings.upper("hello")
greeter.meta.pid()
```

## What the example shows

The greeter example is intentionally small but covers the main authoring surface:

- a manifest with a plugin-scoped module name,
- top-level function exports,
- object exports with methods,
- simple JSON-like argument/result values,
- standard `plugin.Serve(...)` wiring with the shared handshake.

If you are building your own plugin, start by copying `plugins/examples/greeter` rather than the test fixtures.
