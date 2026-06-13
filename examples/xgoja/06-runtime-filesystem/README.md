# xgoja runtime filesystem jsverbs example

This example builds a generated xgoja binary that scans JavaScript verb files from disk at runtime.

Use this mode while developing verbs locally. The generated binary expects the `verbs/` directory to still exist when you run the verb command.

## Run

```bash
make smoke
```

The smoke target:

1. validates `xgoja.yaml`,
2. builds `dist/runtime-filesystem` using v2 workspace module resolution,
3. runs `eval` against the fixture provider module,
4. runs `run scripts/run.js` through the generated runtime,
5. runs the filesystem verb from `verbs/tools.js`.

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
sources:
  - id: local-dev
    kind: jsverbs
    from:
      dir: ./verbs
commands:
  - id: verbs
    type: builtin.jsverbs
    sources: [local-dev]
```

If your runtime source root contains generated files or browser bundles, keep the root narrow or add filters:

```yaml
sources:
  - id: local-dev
    kind: jsverbs
    from:
      dir: .
    include:
      - verbs/**/*.js
      - site.js
    exclude:
      - assets/**
      - dist/**
```

`include` and `exclude` match slash-separated paths relative to the source root.
