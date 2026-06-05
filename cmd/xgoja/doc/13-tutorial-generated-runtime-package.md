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

## 4. Runnable repository example

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
