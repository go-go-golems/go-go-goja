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

This tutorial uses the user-facing example plugin under `plugins/examples/greeter` as the fastest route to success first, and then shows the minimal code shape you need for your own plugin.

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

## Step 6: Understand the minimal plugin shape

A plugin binary needs three main pieces:

1. a type that implements the shared module interface,
2. a manifest that describes the JavaScript module,
3. a `plugin.Serve(...)` main function wired with the shared handshake and plugin set.

The minimal shape looks like this:

```go
package main

import (
	"context"
	"fmt"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/shared"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/protobuf/types/known/structpb"
)

type helloModule struct{}

func (helloModule) Manifest(context.Context) (*contract.ModuleManifest, error) {
	return &contract.ModuleManifest{
		ModuleName: "plugin:hello",
		Version:    "v1",
		Exports: []*contract.ExportSpec{
			{
				Name: "greet",
				Kind: contract.ExportKind_EXPORT_KIND_FUNCTION,
			},
		},
	}, nil
}

func (helloModule) Invoke(_ context.Context, req *contract.InvokeRequest) (*contract.InvokeResponse, error) {
	switch req.GetExportName() {
	case "greet":
		name := "world"
		if len(req.GetArgs()) > 0 {
			name = req.GetArgs()[0].GetStringValue()
		}
		value, err := structpb.NewValue("hello, " + name)
		if err != nil {
			return nil, err
		}
		return &contract.InvokeResponse{Result: value}, nil
	default:
		return nil, fmt.Errorf("unsupported export %q", req.GetExportName())
	}
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig:  shared.Handshake,
		VersionedPlugins: shared.VersionedServerPluginSets(helloModule{}),
		GRPCServer:       plugin.DefaultGRPCServer,
	})
}
```

This is the smallest useful mental model:

- `Manifest(...)` tells the host what module exists.
- `Invoke(...)` handles calls from JavaScript.
- `plugin.Serve(...)` publishes the service.

## Step 7: Build your own plugin

Put your plugin in a package of your choice and build it into the plugin directory.

Example:

```bash
go build -o ~/.go-go-goja/plugins/examples/goja-plugin-hello ./path/to/your/plugin
```

Make sure:

- the output filename still matches `goja-plugin-*`,
- the manifest module name begins with `plugin:`,
- the manifest export list matches what `Invoke(...)` actually supports.

## Step 8: Load your plugin in JavaScript

If your manifest name is `plugin:hello`, test it like this:

```javascript
const hello = require("plugin:hello")
hello.greet("Manuel")
```

This should now behave exactly like any other runtime-registered module from the JavaScript side.

## Step 9: Add an object export

If you want namespaced methods, add an object export to the manifest:

```go
{
	Name:    "math",
	Kind:    contract.ExportKind_EXPORT_KIND_OBJECT,
	Methods: []string{"add"},
}
```

Then handle it in `Invoke(...)`:

```go
case "math":
	if req.GetMethodName() != "add" {
		return nil, fmt.Errorf("unsupported method %q", req.GetMethodName())
	}
	sum := req.GetArgs()[0].GetNumberValue() + req.GetArgs()[1].GetNumberValue()
	value, err := structpb.NewValue(sum)
	if err != nil {
		return nil, err
	}
	return &contract.InvokeResponse{Result: value}, nil
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

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `require("plugin:hello")` fails | The binary was not discovered or the manifest did not match the requested module name | Check the output filename, the plugin directory path, and the `ModuleName` field in the manifest |
| Runtime startup fails before the REPL prompt appears | Plugin validation failed during runtime creation | Build with a known-good manifest first, then compare your plugin to the example fixture |
| `Invoke(...)` returns the wrong value shape | The plugin returned data that does not map cleanly through `structpb.Value` | Stick to strings, numbers, booleans, arrays, objects, and null |
| The plugin works in tests but not manually | The test built the binary into a temp dir, but your manual run expects the default per-user tree | Rebuild into `~/.go-go-goja/plugins/...` or pass the exact path with `--plugin-dir` |
| `js-repl` does not see plugins | The binary was not built under the default tree and no explicit directory was passed | Build into `~/.go-go-goja/plugins/...` or start `js-repl` with `--plugin-dir /path/to/plugins` |

## See Also

- `repl help goja-plugin-user-guide` — User-facing plugin reference
- `repl help goja-plugin-developer-guide` — Internal architecture and integration guide
- `repl help repl-usage` — General REPL usage
