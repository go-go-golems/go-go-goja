# xgoja embedded jsverbs example

This example builds a generated xgoja binary that copies local JavaScript verb files into the generated workspace and embeds them into the final binary with `go:embed`.

Use this mode when local verb files should become part of the generated executable.

## Run

```bash
make smoke
```

The smoke target:

1. validates `xgoja.yaml`,
2. builds `dist/embedded-jsverbs` with a local `go-go-goja` replace,
3. runs `repl` against the fixture provider module,
4. runs `run scripts/run.js` through the generated runtime,
5. runs the embedded verb from the generated binary.

Expected final lines:

```text
hello embedded
hello embedded
```

## Prove it is self-contained

```bash
make prove-self-contained
```

That target copies the generated binary to a temporary directory and runs the verb away from this example's `verbs/` directory.

## Source mode

The important part of `xgoja.yaml` is:

```yaml
jsverbs:
  - id: local
    path: ./verbs
    embed: true
```
