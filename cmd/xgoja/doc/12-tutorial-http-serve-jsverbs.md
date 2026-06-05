---
Title: "Tutorial: HTTP serve commands for JavaScript verbs"
Slug: tutorial-http-serve-jsverbs
Short: "Build a generated xgoja binary whose provider-backed serve command runs JavaScript verbs as HTTP site setup functions."
Topics:
- xgoja
- tutorial
- jsverbs
- http
- express
Commands:
- xgoja build
- xgoja doctor
- xgoja list-modules
Flags:
- --http-listen
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial builds a generated xgoja binary that exposes HTTP site setup functions as JavaScript verbs.

The generated command shape is:

```bash
./dist/http-serve-jsverbs serve sites demo --http-listen 127.0.0.1:8787
```

The `serve` command is contributed by the `go-go-goja-http` provider. It scans the configured `jsverbs:` sources, creates commands from the discovered verb metadata, invokes the selected verb once, and keeps the runtime alive until Ctrl-C or SIGTERM. The selected verb should register routes through `require("express")`.

This is different from the built-in `verbs` command. `verbs sites demo` runs the JavaScript function as a short-lived CLI command and closes the runtime after the function returns. `serve sites demo` treats the function as HTTP setup code and keeps the runtime alive for request handling.

## 1. Create the JavaScript verb

Create `verbs/sites.js`:

```js
__package__({ name: "sites" })
__verb__("demo", { name: "demo", output: "text", short: "Serve demo site" })
function demo() {
  const express = require("express")
  const app = express.app()

  app.get("/", (_req, res) => res.send("hello from an xgoja jsverb site"))
  app.get("/healthz", (_req, res) => res.json({ ok: true, site: "demo" }))
}
```

The function registers routes and then returns. It does not need to block. The provider-backed `serve` command owns the blocking lifecycle.

## 2. Write xgoja.yaml

Enable the HTTP provider package, select the `express` module in the top-level module list, configure the `serve` command provider, and point `jsverbs:` at the verb directory:

```yaml
name: http-serve-jsverbs
target:
  kind: xgoja
  output: dist/http-serve-jsverbs
packages:
  - id: go-go-goja-http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
    register: Register
modules:
  - package: go-go-goja-http
    name: express
    as: express
commands:
  jsverbs:
    enabled: true
    name: verbs
command_providers:
  - id: http-serve
    package: go-go-goja-http
    name: serve
    mount: serve
jsverbs:
  - id: local
    path: verbs
```

The single top-level `modules:` list is the module set used by generated commands and provider commands. There is no separate runtime profile selection for this mode.

## 3. Validate, build, and run

Validate and inspect the module set:

```bash
xgoja doctor -f xgoja.yaml
xgoja list-modules -f xgoja.yaml
```

Build:

```bash
xgoja build -f xgoja.yaml --keep-work
```

Run the HTTP setup verb:

```bash
./dist/http-serve-jsverbs serve sites demo --http-listen 127.0.0.1:8787
```

Open:

```text
http://127.0.0.1:8787/
http://127.0.0.1:8787/healthz
```

Stop the server with Ctrl-C.

## 4. Compare the three HTTP execution modes

| Mode | Command shape | Runtime lifetime | Use when |
| --- | --- | --- | --- |
| Short-lived verb | `./dist/app verbs sites demo` | Closes after the verb returns | The verb is a normal CLI command. |
| Script server | `./dist/app run scripts/server.js --http-listen ... --keep-alive` | Kept alive by the built-in `run` command | The server setup is a script file. |
| Verb server | `./dist/app serve sites demo --http-listen ...` | Kept alive by the HTTP provider's `serve` command | The server setup should be discoverable as a generated jsverb command. |

Use `run --keep-alive` for standalone setup scripts. Use provider-backed `serve` when the route setup should be part of the generated verb command tree.

## 5. Runnable repository example

The repository contains a complete smoke-tested version at:

```text
examples/xgoja/13-http-serve-jsverbs
```

Run it with:

```bash
make -C examples/xgoja/13-http-serve-jsverbs smoke
```

That target builds the generated binary, starts `serve sites demo` on a test port, fetches the HTML and health endpoints with `curl`, and stops the process.

## Troubleshooting

| Problem | What to check |
| --- | --- |
| `Cannot find module 'express'` | Add `go-go-goja-http` to `packages:` and select `name: express` in `modules:`. |
| No `serve` command exists | Add a `command_providers:` entry with `package: go-go-goja-http`, `name: serve`, and `mount: serve`. |
| No `serve sites demo` command exists | Confirm the `jsverbs:` path points at the directory containing `sites.js` and that the file scans correctly. |
| Server exits immediately when using `verbs` | Use `serve sites demo`; the built-in `verbs` command is intentionally short-lived. |
| Address is already in use | Change `--http-listen` or stop the process currently bound to that address. The HTTP provider reports listen failures during startup. |
