---
Title: "Tutorial: build and install a go-go-goja plugin"
Slug: plugin-tutorial-build-install
Short: "Build a minimal HashiCorp plugin binary, install it into a plugin directory, and call it from the REPL."
Topics:
- goja
- plugins
- tutorial
- repl
- hashicorp
- javascript
Commands:
- repl
Flags:
- --plugin-dir
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial walks through the full happy path for building and installing a plugin for `go-go-goja`.

By the end, you will have:

- a plugin binary built from Go source,
- a plugin directory that `repl` can scan,
- a working `require("plugin:...")` call from JavaScript,
- a mental model for how your Go code becomes a JavaScript module.

This tutorial uses the user-facing example plugin under `plugins/examples/greeter` as the fastest route to success first, and then shows the richer SDK-based code shape you should use for your own plugin.

The repo now also ships additional examples under `plugins/examples/...`:

- `clock` for structured return values,
- `validator` for `sdk.Call` helper usage and validation errors,
- `kv` for stateful object methods,
- `system-info` for nested mixed-shape responses,
- `failing` for explicit handler errors.

Work through `greeter` first, then read the others as focused follow-up examples.

## Prerequisites

You need:

- a working Go toolchain,
- this repository checked out,
- the ability to run `go run ./cmd/repl`,
- a writable local directory such as `/tmp/goja-plugins`.

This tutorial assumes you are running commands from the repository root.

If you want to constrain the runtime to one expected plugin module while testing, you can add `--allow-plugin-module plugin:greeter` to the REPL commands shown below.

## Step 1: Build the example plugin

Start by building the existing example. This proves the host side is working before you start editing your own code.

```bash
mkdir -p ~/.go-go-goja/plugins/examples
go build -o ~/.go-go-goja/plugins/examples/goja-plugin-greeter ./plugins/examples/greeter
```

Why this step matters:

- it gives you a known-good binary,
- it verifies your local Go environment,
- it gives you a reference binary name and location for discovery.

The output file name should match the host discovery pattern. The default pattern is `goja-plugin-*`, so `goja-plugin-greeter` is a safe default.

## Step 2: Start the REPL with plugin discovery enabled

Run:

```bash
go run ./cmd/repl
```

This tells the runtime builder to:

- scan `~/.go-go-goja/plugins/...`,
- launch matching plugin binaries,
- validate their manifests,
- register valid plugin modules into the runtime.

If startup succeeds, you now have a runtime that can resolve plugin modules through `require()`.

## Step 3: Require the plugin module

Inside the REPL:

```javascript
let greeter = require("plugin:greeter")
```

If this line works, the main integration path is already correct:

- the binary was discovered,
- the plugin handshake succeeded,
- the manifest was accepted,
- the module was registered into the runtime.

## Step 4: Call exported functions

Continue in the REPL:

```javascript
greeter.greet("hello")
greeter.strings.upper("hello")
greeter.meta.pid()
```

Expected results:

- `"hello, hello"`
- `"HELLO"`
- a numeric process ID

At this point you have verified both supported export styles:

- direct function export,
- object method export.

## Step 5: Run the same test as a script

Interactive testing is useful, but a script is better when you want repeatability.

Create a file:

```javascript
const greeter = require("plugin:greeter")

console.log(greeter.greet("hello"))
console.log(greeter.strings.upper("hello"))
console.log(greeter.meta.pid())
```

Run it:

```bash
go run ./cmd/repl /tmp/test-plugin.js
```

This is the easiest way to create a small regression harness while you iterate on plugin code.

## Step 6: Understand the recommended SDK-based plugin shape

The current recommended path is to author plugins with `pkg/hashiplugin/sdk`.

That gives you four useful building blocks:

1. `sdk.MustModule(...)` to define one plugin module,
2. `sdk.Function(...)` for top-level exports,
3. `sdk.Object(...sdk.Method(...))` for object-method exports,
4. `sdk.Serve(...)` to boot the shared transport.

The minimal useful shape now looks like this:

```go
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/sdk"
)

func main() {
	mod := sdk.MustModule(
		"plugin:hello",
		sdk.Version("v1"),
		sdk.Doc("Simple hello plugin"),
		sdk.Function("greet", func(_ context.Context, call *sdk.Call) (any, error) {
			name := call.StringDefault(0, "world")
			return fmt.Sprintf("hello, %s", name), nil
		}),
		sdk.Object("strings",
			sdk.Method("upper", func(_ context.Context, call *sdk.Call) (any, error) {
				return strings.ToUpper(call.StringDefault(0, "")), nil
			}),
		),
	)

	sdk.Serve(mod)
}
```

This is the new mental model:

- `sdk.MustModule(...)` defines the plugin module and its manifest shape.
- `sdk.Function(...)` and `sdk.Object(...sdk.Method(...))` declare exports.
- `sdk.Call` gives handlers easy access to decoded arguments.
- `sdk.Serve(...)` publishes the service over the shared transport.

## Step 7: Build your own plugin

Put your plugin in a package of your choice and build it into the plugin directory.

Example:

```bash
go build -o ~/.go-go-goja/plugins/examples/goja-plugin-hello ./path/to/your/plugin
```

Make sure:

- the output filename still matches `goja-plugin-*`,
- the manifest module name begins with `plugin:`,
- the SDK declarations match the exports you expect JavaScript to see.

## Step 8: Load your plugin in JavaScript

If your manifest name is `plugin:hello`, test it like this:

```javascript
const hello = require("plugin:hello")
hello.greet("Manuel")
```

This should now behave exactly like any other runtime-registered module from the JavaScript side.

## Step 9: Add an object export

If you want namespaced methods, add an object export with methods:

```go
sdk.Object("math",
	sdk.Method("add", func(_ context.Context, call *sdk.Call) (any, error) {
		a, err := call.Float64(0)
		if err != nil {
			return nil, err
		}
		b, err := call.Float64(1)
		if err != nil {
			return nil, err
		}
		return a + b, nil
	}),
)
```

From JavaScript:

```javascript
const mod = require("plugin:hello")
mod.math.add(2, 3)
```

This is the right approach when you want one module to expose several closely related operations without flattening every function into the top-level export object.

## Step 10: Know the current limits

This first implementation is intentionally conservative.

Plan around these constraints:

- values should be JSON-like,
- manifests only support function and object exports,
- `repl` is the current CLI path for plugin testing,
- both `repl` and `js-repl` default to scanning `~/.go-go-goja/plugins/...`.

These are not permanent limits, but they are the limits you should design against today.

## Complete reference workflow

Here is the full shell sequence in one place:

```bash
mkdir -p ~/.go-go-goja/plugins/examples
go build -o ~/.go-go-goja/plugins/examples/goja-plugin-greeter ./plugins/examples/greeter
cat >/tmp/test-plugin.js <<'EOF'
const greeter = require("plugin:greeter")
console.log(greeter.greet("hello"))
console.log(greeter.strings.upper("hello"))
console.log(greeter.meta.pid())
EOF
go run ./cmd/repl /tmp/test-plugin.js
```

That sequence is a good smoke test after host-side changes or when onboarding a new plugin author.

After you finish the baseline flow, the next useful follow-up commands are:

```bash
go build -o ~/.go-go-goja/plugins/examples/goja-plugin-validator ./plugins/examples/validator
go build -o ~/.go-go-goja/plugins/examples/goja-plugin-kv ./plugins/examples/kv
```

Then in JavaScript:

```javascript
const validator = require("plugin:validator")
validator.grade(0.9, true)

const kv = require("plugin:kv")
kv.store.set("name", "Manuel")
kv.store.get("name")
```

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `require("plugin:hello")` fails | The binary was not discovered or the manifest did not match the requested module name | Check the output filename, the plugin directory path, and the `ModuleName` field in the manifest |
| Runtime startup fails before the REPL prompt appears | Plugin validation failed during runtime creation | Build with a known-good manifest first, then compare your plugin to the example fixture |
| A handler returns the wrong value shape | The SDK could not encode the returned Go value into `structpb.Value` | Stick to strings, numbers, booleans, arrays, objects, and null |
| The plugin works in tests but not manually | The test built the binary into a temp dir, but your manual run expects the default per-user tree | Rebuild into `~/.go-go-goja/plugins/...` or pass the exact path with `--plugin-dir` |
| `js-repl` does not see plugins | The binary was not built under the default tree and no explicit directory was passed | Build into `~/.go-go-goja/plugins/...` or start `js-repl` with `--plugin-dir /path/to/plugins` |

## See Also

- `repl help goja-plugin-user-guide` — User-facing plugin reference
- `repl help goja-plugin-developer-guide` — Internal architecture and integration guide
- `repl help repl-usage` — General REPL usage
