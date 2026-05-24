# xgoja examples

These examples provide runnable smoke tests for generated xgoja binaries.

Each example directory has its own `README.md`, `Makefile`, and `xgoja.yaml`.

## Examples

- `runtime-filesystem/` — JS verbs stay on disk and are scanned by the generated binary at runtime.
- `embedded-jsverbs/` — local JS verbs are copied into the generated workspace and embedded into the binary.
- `provider-shipped-jsverbs/` — JS verbs live inside a Go provider package and are selected by `package`/`source`.
- `core-provider/` — generated binary using the safe first-party core provider (`path`, `yaml`, `crypto`, etc.).
- `host-provider/` — generated binary using guarded host-capability modules (`fs`, `exec`, `database`) with explicit config.
- `multiple-runtimes/` — one generated binary with separate safe and host runtime profiles mapped to different commands.

## Run all examples

```bash
for dir in runtime-filesystem embedded-jsverbs provider-shipped-jsverbs; do
  make -C examples/xgoja/$dir smoke
done
```

The Makefiles use:

```bash
GOWORK=off go run ./cmd/xgoja ... --xgoja-replace <repo-root>
```

`GOWORK=off` avoids the local workspace `goja` override while this repository's workspace dependency mismatch is unresolved. `--xgoja-replace` makes generated binaries use this checkout of `go-go-goja` instead of a released module version.
