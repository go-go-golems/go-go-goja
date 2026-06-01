# xgoja embedded assets fs example

This example builds a generated xgoja binary that embeds local files from `./assets` into the executable and exposes them to JavaScript through a read-only fs alias.

The runtime registers the same provider module twice:

- `require("fs:assets")` reads embedded files mounted at `/app`.
- `require("fs:host")` uses the explicitly allowed host filesystem.

It does **not** register plain `require("fs")`, which makes filesystem capability use visible at JavaScript call sites.

## Run

```bash
make smoke
```

The smoke target:

1. validates `xgoja.yaml`,
2. lists selected modules,
3. builds `dist/embedded-assets-fs` with a local `go-go-goja` replace,
4. runs `scripts/read-assets.js`,
5. verifies that embedded JSON was copied to `out.json` through `fs:host`.

## Prove it is self-contained

```bash
make prove-self-contained
```

That target copies the generated binary and script to a temporary directory and runs it away from the original `assets/` directory. The script still reads `/app/config/default.json` because the file is embedded into the binary.

## Important buildspec fragment

```yaml
assets:
  - id: app-assets
    path: ./assets
    embed: true

runtimes:
  main:
    modules:
      - package: go-go-goja-host
        name: fs
        as: fs:assets
        config:
          embedded:
            allow: true
            mounts:
              - asset: app-assets
                mount: /app
      - package: go-go-goja-host
        name: fs
        as: fs:host
        config:
          allow: true
```
