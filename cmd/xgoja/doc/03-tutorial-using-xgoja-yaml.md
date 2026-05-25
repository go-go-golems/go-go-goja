---
Title: "Tutorial: using xgoja.yaml"
Slug: tutorial-using-xgoja-yaml
Short: "Build and run a generated xgoja binary with provider modules and JavaScript verbs."
Topics:
- xgoja
- tutorial
- goja
- jsverbs
- providers
Commands:
- xgoja build
- xgoja doctor
- xgoja list-modules
Flags:
- --file
- --keep-work
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial shows the normal xgoja workflow: write a build spec, validate it, inspect selected modules, build a generated binary, and run JavaScript through the generated runtime.

The examples assume a provider package already exists and exposes a module named `hello` under provider package ID `fixture`. Replace the import path and module names with your own provider package.

## 1. Write xgoja.yaml

Create `xgoja.yaml`:

```yaml
name: fixture
go:
  version: "1.26"
  module: example.com/generated/fixture
target:
  kind: xgoja
  output: dist/fixture
packages:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
    register: Register
runtimes:
  repl:
    modules:
      - package: fixture
        name: hello
        as: hello
commands:
  eval:
    enabled: true
    runtime: repl
    name: eval
  run:
    enabled: true
    runtime: repl
    name: run
  repl:
    enabled: true
    runtime: repl
    name: repl
  jsverbs:
    enabled: true
    runtime: repl
    name: verbs
```

The `packages` section controls which Go provider package is compiled into the generated binary. The `runtimes.repl.modules` section controls which modules are available through `require()` for this runtime profile.

## 2. Validate the spec

Run:

```bash
xgoja doctor -f xgoja.yaml
```

Fix all reported errors before building. The most common errors are unknown package IDs, missing target imports for target modes, missing embedded verb paths, and duplicate runtime aliases.

## 3. Inspect selected modules

Run:

```bash
xgoja list-modules -f xgoja.yaml
```

This reports modules selected by the spec. Use it to confirm that the runtime profile exposes the expected `require()` names.

## 4. Build the binary

Run from an installed xgoja binary:

```bash
xgoja build -f xgoja.yaml --keep-work
```

When testing from a local checkout before a release, point generated builds at that checkout:

```bash
xgoja build -f xgoja.yaml --keep-work --xgoja-replace /path/to/go-go-goja
```

`--keep-work` leaves the generated workspace on disk. Use it while learning or debugging because it lets you inspect generated `go.mod`, generated `main.go`, copied embedded jsverb files, and the embedded spec JSON.

The output binary is written to `target.output`, for example:

```bash
./dist/fixture
```

## 5. Evaluate JavaScript

Run a simple expression against the generated runtime:

```bash
./dist/fixture eval 'require("hello").greet("intern")'
```

Execute a JavaScript file with the generated `run` command:

```bash
cat > script.js <<'EOF'
const hello = require("hello")
console.log(hello.greet("file"))
EOF
./dist/fixture run script.js
```

The generated binary creates a fresh goja runtime, registers the modules selected by the command's runtime profile, evaluates the source, and prints a non-null result. The `run` command also adds script-local module roots so `require("./helper")` resolves relative to the script file.

For an interactive terminal session, run:

```bash
./dist/fixture repl
```

The REPL command starts a Bubble Tea terminal UI backed by the same runtime-profile module policy.

## 6. Add a runtime filesystem jsverb source

Create `verbs/tools.js`:

```js
__package__({ name: "tools" })
__verb__("greet", {
  name: "greet",
  output: "text",
  fields: {
    name: { type: "string", required: true }
  }
})
function greet(name) {
  const hello = require("hello")
  return hello.greet(name)
}
```

Add this to `xgoja.yaml`:

```yaml
jsverbs:
  - id: local-dev
    path: ./verbs
    embed: false
```

Build again and run:

```bash
./dist/fixture verbs tools greet --name intern
```

This mode scans `./verbs` from disk when the generated binary starts.

## 7. Embed local jsverbs into the generated binary

Change the source to:

```yaml
jsverbs:
  - id: local
    path: ./verbs
    embed: true
```

Build again:

```bash
xgoja build -f xgoja.yaml --keep-work
```

xgoja copies `./verbs` into the generated workspace and generated `main.go` embeds it with `go:embed`. The final binary no longer needs the original `./verbs` directory at runtime.

## 8. Use provider-shipped jsverbs

Provider packages can ship JS verbs next to their native Go modules.

Provider side:

```go
//go:embed verbs/*.js
var verbsFS embed.FS

func Register(registry *providerapi.Registry) error {
    return registry.Package("fixture",
        providerapi.Module{Name: "hello", New: newHelloModule},
        providerapi.VerbSource{Name: "verbs", FS: verbsFS, Root: "verbs"},
    )
}
```

Spec side:

```yaml
jsverbs:
  - id: provider-defaults
    package: fixture
    source: verbs
```

This scans the provider's embedded filesystem, not the local project filesystem.

## 9. Debug generated builds

Use:

```bash
xgoja build -f xgoja.yaml --keep-work
```

Then inspect the generated workspace:

```text
go.mod
main.go
xgoja.gen.json
xgoja_embed/jsverbs/...
```

If the generated build fails, the generated source usually shows whether the problem is an import path, module version, replace path, target function name, or embedded source path.

## Troubleshooting

| Problem | What to check |
| --- | --- |
| `doctor` reports unknown package | The `package` field must match a `packages[].id`. |
| `require("name")` fails | The runtime profile must include a module with `as: name` or `name: name`. |
| jsverb command is missing | Confirm `commands.jsverbs.enabled: true` and that the source scans without diagnostics. |
| embedded jsverb missing | Build with `--keep-work` and inspect `xgoja_embed/jsverbs/<id>/`. |
| provider source missing | Confirm the provider registers `VerbSource{Name, FS, Root}` and the spec uses the same package ID and source name. |

## See also

- `overview` for the architecture.
- `buildspec` for the complete spec reference.
