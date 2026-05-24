# xgoja runtime filesystem jsverbs example

This example builds a generated xgoja binary that scans JavaScript verb files from disk at runtime.

Use this mode while developing verbs locally. The generated binary expects the `verbs/` directory to still exist when you run the verb command.

## Run

```bash
make smoke
```

The smoke target:

1. validates `xgoja.yaml`,
2. builds `dist/runtime-filesystem` with a local `go-go-goja` replace,
3. runs `repl` against the fixture provider module,
4. runs the filesystem verb from `verbs/tools.js`.

Expected final lines:

```text
hello filesystem
hello filesystem
```

## Useful targets

```bash
make doctor
make build
make run
make clean
```

## Source mode

The important part of `xgoja.yaml` is:

```yaml
jsverbs:
  - id: local-dev
    path: ./verbs
    embed: false
```
