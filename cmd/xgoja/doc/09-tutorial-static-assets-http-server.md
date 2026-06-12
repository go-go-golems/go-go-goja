---
Title: "Tutorial: embedded static asset HTTP server"
Slug: tutorial-static-assets-http-server
Short: "Build a generated xgoja binary that serves bundled HTML, CSS, and JavaScript with express."
Topics:
- xgoja
- tutorial
- assets
- http
- express
Commands:
- xgoja build
- xgoja doctor
- xgoja list-modules
Flags:
- --http-listen
- --keep-alive
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial builds a generated xgoja binary that embeds a small static web site and serves it through the HTTP provider's Express-style module.

The key idea is that embedded files are exposed to JavaScript through a read-only fs module instance, usually `require("fs:assets")`. The Express module can serve that module directly:

```js
const express = require("express")
const assets = require("fs:assets")

const app = express.app()
app.staticFromAssetsModule("/static", assets, "/app/public")
```

No temporary host staging directory is required.

## 1. Create static assets

Create a small site under `assets/public`:

```bash
mkdir -p assets/public assets/config
cat > assets/public/index.html <<'EOF'
<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>xgoja embedded static assets</title>
    <link rel="stylesheet" href="/static/styles.css" />
  </head>
  <body>
    <h1>Hello from embedded assets</h1>
    <p id="message">loading...</p>
    <script src="/static/app.js"></script>
  </body>
</html>
EOF
cat > assets/public/styles.css <<'EOF'
body { font-family: system-ui, sans-serif; margin: 3rem; }
h1 { color: #2563eb; }
EOF
cat > assets/public/app.js <<'EOF'
document.getElementById("message").textContent = "served from the generated binary"
EOF
printf '{"name":"static-assets","ok":true}\n' > assets/config/default.json
```

Asset trees may contain dot directories such as `.well-known`; xgoja uses Go's `all:` embed pattern for assets. `assets[].include` and `assets[].exclude` are not supported yet and are rejected by validation.

## 2. Write xgoja.yaml

Use both the host provider and the HTTP provider:

```yaml
name: embedded-static-server
target:
  kind: xgoja
  output: dist/embedded-static-server
packages:
  - id: go-go-goja-host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host
  - id: go-go-goja-http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
assets:
  - id: app-assets
    path: ./assets
    embed: true
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
  - package: go-go-goja-http
    name: express
commands:
  run:
    enabled: true
  eval:
    enabled: true
  repl:
    enabled: false
  jsverbs:
    enabled: false
```

The `assets:` entry copies `./assets` into the generated Go workspace and embeds it into the binary. The `fs:assets` module mounts that tree at `/app`. The `express` module registers HTTP routes and static handlers.

## 3. Create the server setup script

Create `scripts/server.js`:

```js
const express = require("express")
const assets = require("fs:assets")

const app = express.app()
app.staticFromAssetsModule("/static", assets, "/app/public")
app.get("/").public().handle((_ctx, res) => res.redirect("/static/"))
app.get("/api/config").public().handle((_ctx, res) => {
  const config = JSON.parse(assets.readFileSync("/app/config/default.json", "utf8"))
  res.json({ ok: true, config })
})

console.log("open http://127.0.0.1:8787/static/")
console.log("try  http://127.0.0.1:8787/api/config")
```

`staticFromAssetsModule(prefix, assetsModule, root)` accepts the actual module object returned by `require("fs:assets")`. The Go side verifies that it is backed by embedded read-only assets and serves `root` with `http.FileServer(http.FS(...))`.

## 4. Validate, build, and run

Validate and inspect the module set:

```bash
xgoja doctor -f xgoja.yaml
xgoja list-modules -f xgoja.yaml
```

Build:

```bash
xgoja build -f xgoja.yaml --keep-work
```

Run the server setup script:

```bash
./dist/embedded-static-server run scripts/server.js --http-listen 127.0.0.1:8787 --keep-alive
```

`--keep-alive` matters. The script registers routes and then returns; without `--keep-alive`, the generated `run` command would close the runtime and shut down the HTTP server immediately.

Open:

```text
http://127.0.0.1:8787/static/
http://127.0.0.1:8787/api/config
```

Stop the server with Ctrl-C.

## 5. Runnable repository example

The repository contains a complete smoke-tested version at:

```text
examples/xgoja/10-embedded-assets-fs
```

Run it with:

```bash
make -C examples/xgoja/10-embedded-assets-fs serve-smoke
```

That target builds the generated binary, starts the server on a test port, fetches the embedded static site and JSON endpoint with `curl`, and stops the process.

## Troubleshooting

| Problem | What to check |
| --- | --- |
| `Cannot find module 'express'` | Add `go-go-goja-http` to `packages:` and select `name: express` in the module set. |
| `Cannot find module 'fs:assets'` | Register host provider `name: fs` with `as: fs:assets` and `embedded.allow: true`. |
| Static site works during build but not after moving the binary | Confirm files live under `assets:` with `embed: true`, not a host filesystem `app.static` path. |
| Server exits immediately | Use `run ... --keep-alive`; route registration scripts otherwise finish and close the runtime. |
| `include` or `exclude` is rejected | Asset filters are not implemented yet. Pre-build a filtered asset directory before running `xgoja build`. |
