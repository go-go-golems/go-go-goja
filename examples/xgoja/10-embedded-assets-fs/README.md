# xgoja embedded assets fs example

This example builds a generated xgoja binary that embeds local files from `./assets` into the executable and exposes them to JavaScript through a read-only fs alias. It also includes a small Express-style HTTP script that serves bundled static files directly from the embedded asset filesystem with the `express` module.

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
3. builds `dist/embedded-assets-fs` using v2 workspace module resolution,
4. runs `scripts/read-assets.js`,
5. verifies that embedded JSON was copied to `out.json` through `fs:host`.

## Serve bundled static assets

```bash
make serve-smoke
```

The `serve-smoke` target builds the binary, starts:

```bash
./dist/embedded-assets-fs run scripts/serve-static-assets.js --http-listen 127.0.0.1:18787 --keep-alive
```

and checks:

- `http://127.0.0.1:18787/static/` serves `assets/public/index.html`,
- `http://127.0.0.1:18787/static/app.js` serves bundled JavaScript,
- `http://127.0.0.1:18787/api/config` can still read `assets/config/default.json` through `fs:assets`.

The script passes the actual embedded fs module object to `app.staticFromAssetsModule("/static", assets, "/app/public")`. The Go side recognizes that the module is backed by read-only embedded assets and serves it directly through `http.FileServer(http.FS(...))`; no host staging directory is needed. The generated `run` command uses `--keep-alive` so the runtime and HTTP server stay alive after route registration.

For manual exploration:

```bash
make build
./dist/embedded-assets-fs run scripts/serve-static-assets.js --http-listen 127.0.0.1:8787 --keep-alive
```

Then open <http://127.0.0.1:8787/static/> and stop the server with Ctrl-C.

## Prove it is self-contained

```bash
make prove-self-contained
```

That target copies the generated binary and script to a temporary directory and runs it away from the original `assets/` directory. The script still reads `/app/config/default.json` because the file is embedded into the binary.

## Important xgoja/v2 fragment

```yaml
sources:
  - id: app-assets
    kind: assets
    from:
      dir: ./assets

runtime:
  modules:
    - provider: go-go-goja-host
      name: fs
      as: fs:assets
      config:
        embedded:
          allow: true
          mounts:
            - asset: app-assets
              mount: /app
    - provider: go-go-goja-host
      name: fs
      as: fs:host
      config:
        allow: true
    - provider: go-go-goja-http
      name: express

artifacts:
  - id: embedded-assets
    type: embedded-assets
    sources: [app-assets]
```
