# xgoja examples

These examples are both runnable smoke tests and a numbered learning path for generated xgoja binaries.

Each example directory has its own `README.md`, `Makefile`, and `xgoja.yaml`. Start with the first provider example and continue down the list; later examples assume the runtime-profile and provider vocabulary from earlier examples.

## Learning path

1. `01-core-provider/` — safe first-party modules such as `path`, `yaml`, and `crypto`.
2. `02-host-provider/` — guarded host-capability modules such as `fs`, `exec`, and `database` with explicit config.
3. `03-multiple-runtimes/` — one generated binary with separate safe and host runtime profiles mapped to different commands.
4. `04-module-sections/` — provider Glazed config sections and runtime initializers on built-in `eval`, `run`, and `jsverbs` commands.
5. `05-command-provider/` — provider-owned Glazed command sets mounted into the generated root command.
6. `06-runtime-filesystem/` — JS verbs stay on disk and are scanned by the generated binary at runtime.
7. `07-embedded-jsverbs/` — local JS verbs are copied into the generated workspace and embedded into the binary.
8. `08-provider-shipped-jsverbs/` — JS verbs live inside a Go provider package and are selected by `package`/`source`.
9. `09-provider-shipped-help-docs/` — Glazed help pages live inside a Go provider package and are selected by `help.sources[].package`/`source`.
10. `10-embedded-assets-fs/` — local files are embedded into the generated binary and read through `require("fs:assets")`, while host writes use `require("fs:host")`.

The Discord bot xgoja example lives in the sibling `discord-bot` repository because it demonstrates inserting xgoja into an existing host-owned runner rather than a standalone generated binary.

## Run all examples

```bash
for dir in \
  01-core-provider \
  02-host-provider \
  03-multiple-runtimes \
  04-module-sections \
  05-command-provider \
  06-runtime-filesystem \
  07-embedded-jsverbs \
  08-provider-shipped-jsverbs \
  09-provider-shipped-help-docs \
  10-embedded-assets-fs; do
  make -C examples/xgoja/$dir smoke
done
```

The Makefiles use:

```bash
GOWORK=off go run ./cmd/xgoja ... --xgoja-replace <repo-root>
```

`GOWORK=off` avoids the local workspace `goja` override while this repository's workspace dependency mismatch is unresolved. `--xgoja-replace` makes generated binaries use this checkout of `go-go-goja` instead of a released module version.
