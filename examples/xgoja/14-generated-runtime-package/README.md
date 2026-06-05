# xgoja generated runtime package example

This example demonstrates `target.kind: package` and `xgoja generate`.
Instead of building a complete generated `main.go` binary, xgoja writes a
reusable Go package under `internal/xgojaruntime`. A normal host application
imports that package, creates a runtime bundle, creates an xgoja runtime, and
runs JavaScript inside the host application's own lifecycle.

Generate the package and run the host smoke test:

```bash
make -C examples/xgoja/14-generated-runtime-package smoke
```

The generated package exposes:

- `EmbeddedSpecJSON`
- `DecodeSpec()`
- `RegisterProviders(registry)`
- `NewBundle(options)`
- `Bundle.NewRuntime(ctx, ...)`
- `Bundle.NewRuntimeFromSections(ctx, vals, ...)`
- `Bundle.AttachDefaultCommands(root)`

The host program in `cmd/host/main.go` imports the generated package and runs:

```js
require("hello").greet("package host")
```

Expected output:

```text
hello package host
```
