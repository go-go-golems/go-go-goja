---
Title: "Tutorial: generated xgoja runtime package"
Slug: tutorial-generated-runtime-package
Short: "Generate an importable Go package that creates xgoja runtimes inside an existing application."
Topics:
- xgoja
- tutorial
- code-generation
- application-integration
- javascript
Commands:
- xgoja generate
- xgoja doctor
Flags:
- --output
- --package
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial shows how to use `xgoja.yaml` to generate a reusable Go package instead of a complete generated `main.go` binary.

Use this mode when an existing Go application owns the command tree, service lifecycle, TUI lifecycle, request lifecycle, or tool execution policy, but still wants xgoja to generate provider registration, embedded runtime specification data, and runtime construction helpers.

## 1. Write xgoja.yaml

Set `target.kind` to `package`:

```yaml
name: generated-runtime-package
target:
  kind: package
  output: internal/xgojaruntime
  package: xgojaruntime
packages:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
    register: Register
modules:
  - package: fixture
    name: hello
    as: hello
commands:
  eval:
    enabled: true
    name: eval
```

`target.output` is the directory that receives generated source. `target.package` is the generated Go package name. If `target.package` is omitted, `xgoja generate` defaults to the output directory basename.

## 2. Generate source

Run:

```bash
xgoja generate -f xgoja.yaml
```

or override the output path and package name from the CLI:

```bash
xgoja generate -f xgoja.yaml \
  --output ./internal/xgojaruntime \
  --package xgojaruntime
```

The command writes `xgoja_runtime.gen.go` and any embedded `xgoja_embed/...` resource directories. It does not write `go.mod`, does not write `main.go`, does not run `go mod tidy`, and does not compile a binary.

## 3. Import the generated package

A host application can create a bundle and runtime:

```go
bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{})
if err != nil {
    return err
}
rt, err := bundle.NewRuntime(context.Background())
if err != nil {
    return err
}
defer rt.Close(context.Background())
```

Evaluate JavaScript through the runtime owner:

```go
ret, err := rt.Owner.Call(context.Background(), "host-eval", func(_ context.Context, vm *goja.Runtime) (any, error) {
    return vm.RunString(`require("hello").greet("host")`)
})
```

The generated package also exposes:

| API | Purpose |
| --- | --- |
| `EmbeddedSpecJSON` | Runtime spec JSON embedded in generated Go source. |
| `DecodeSpec()` | Decode `EmbeddedSpecJSON` into `*app.RuntimeSpec`. |
| `RegisterProviders(registry)` | Register generated provider imports into a registry. |
| `NewBundle(options)` | Build providers, runtime spec, host, and runtime factory. |
| `Bundle.NewRuntime(ctx, ...)` | Create a runtime with default module configuration. |
| `Bundle.NewRuntimeFromSections(ctx, vals, ...)` | Create a runtime using parsed Glazed section values. |
| `Bundle.AttachDefaultCommands(root)` | Attach generated xgoja commands to a host-owned Cobra root. |

## 4. Source-fragment mode

Use `target.kind: source` when you want the standard generated API split across multiple files instead of one `xgoja_runtime.gen.go` file:

```yaml
target:
  kind: source
  output: internal/xgojaruntime
  package: xgojaruntime
```

Run:

```bash
xgoja generate -f xgoja.yaml
```

The output files are:

```text
spec.gen.go       # EmbeddedSpecJSON and DecodeSpec()
providers.gen.go  # provider imports and RegisterProviders()
bundle.gen.go     # Options, Bundle, NewBundle(), NewRuntime(), AttachDefaultCommands()
embed.gen.go      # optional //go:embed variables when jsverbs/help/assets are embedded
```

This is useful when reviewers want smaller generated files or when a host application expects to replace one layer later with a custom implementation.

## 5. Custom-template mode

Use `target.kind: template` when the host application wants to control the generated Go source shape directly:

```yaml
target:
  kind: template
  output: internal/xgojaruntime/custom.gen.go
  package: xgojaruntime
  template: templates/runtime.go.tmpl
```

Run:

```bash
xgoja generate -f xgoja.yaml
```

A custom template receives the same data used by the standard runtime package template. Common fields include:

| Field | Meaning |
| --- | --- |
| `.PackageName` | Generated Go package name. |
| `.SpecJSON` | Runtime spec JSON after embedded path rewriting. |
| `.HasEmbeddedJSVerb` / `.HasEmbeddedHelp` / `.HasEmbeddedAssets` | Whether each embedded filesystem is needed. |
| `.ProviderImports` | Provider import aliases, import paths, and register function names. |

Available template helpers:

| Helper | Purpose |
| --- | --- |
| `quote` | Render a Go string literal. |
| `rawString` | Escape backticks for raw string usage. |
| `json` | JSON-encode a template value. |

Minimal example:

```gotemplate
// Code generated by custom xgoja template; DO NOT EDIT.
package {{ .PackageName }}

const ProviderCount = {{ len .ProviderImports }}
const RuntimeSpec = {{ quote .SpecJSON }}
```

xgoja still owns v2 plan loading, validation, embedded resource copying, runtime spec path rewriting, and `gofmt` formatting. The template controls the Go source body.

## 6. Inspect template data

Use `--template-data` to print the JSON data passed to package/source/custom templates without writing generated files:

```bash
xgoja generate -f xgoja.yaml --template-data
```

This is useful while authoring custom templates because it shows fields such as `PackageName`, `SpecJSON`, embedded-resource booleans, and `ProviderImports`.

## 7. Clean stale generated files

Use `--clean` when switching generation modes or when embedded resources have changed:

```bash
xgoja generate -f xgoja.yaml --clean
```

For package and source-fragment modes, `--clean` removes only known xgoja outputs in the output directory:

```text
xgoja_runtime.gen.go
spec.gen.go
providers.gen.go
bundle.gen.go
embed.gen.go
xgoja_embed/
```

For template mode, `--clean` removes the selected output file only if its name ends in `.gen.go`. It does not delete arbitrary host application files.

## 8. Runnable repository example

The repository contains a complete example at:

```text
examples/xgoja/14-generated-runtime-package
```

Run:

```bash
make -C examples/xgoja/14-generated-runtime-package smoke
```

The smoke target regenerates `internal/xgojaruntime`, runs a host application that imports the generated package, creates an xgoja runtime, and evaluates `require("hello").greet("package host")`.

Expected output contains:

```text
hello package host
```

## Troubleshooting

| Problem | What to check |
| --- | --- |
| `xgoja generate currently supports target.kind package` | Set `target.kind: package`; binary targets still use `xgoja build`. |
| Generated package cannot import a provider | Ensure the host module can resolve the provider import path and has any needed `replace` directives. |
| `Cannot find module` at runtime | Confirm the module is listed in top-level `modules:` and the provider package is listed in `packages:`. |
| Host code cannot import an `internal` package | Import generated `internal/xgojaruntime` only from code inside the parent tree, or generate into a non-internal package directory. |
