# xgoja HTTP serve jsverbs example

This example builds a generated xgoja binary from a native `schema: xgoja/v2`
spec. The binary exposes a provider-backed `serve` command from
`go-go-goja-http`. The command mirrors configured JavaScript verbs, invokes the
selected verb once to register Express routes, and keeps the runtime alive until
the process is stopped.

The v2 spec selects the HTTP provider, selects the Go-backed `express` runtime
module, declares `./verbs` as a `jsverbs` source set, and mounts that source set
both under the builtin `verbs` command and the provider `serve` command:

```yaml
schema: xgoja/v2
providers:
  - id: go-go-goja-http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
runtime:
  modules:
    - provider: go-go-goja-http
      name: express
sources:
  - id: local-sites
    kind: jsverbs
    from:
      dir: ./verbs
    language: javascript
commands:
  - id: verbs
    type: builtin.jsverbs
    sources: [local-sites]
  - id: http-serve
    type: provider.command-set
    provider: go-go-goja-http
    name: serve
    mount: serve
    sources: [local-sites]
artifacts:
  - id: binary
    type: binary
    output: dist/http-serve-jsverbs
    sources: [local-sites]
```

The binary artifact lists `local-sites` under `sources`, so the generated host
copies that jsverb source set into its embedded filesystem. The built binary can
serve the jsverb site without reading `./verbs` from the original checkout.

The site setup verb lives in `verbs/sites.js`:

```js
__package__({ name: "sites" })
__verb__("demo", { name: "demo", output: "text" })
function demo() {
  const express = require("express")
  const app = express.app()
  app.get("/").public().handle((_ctx, res) => res.send("hello from an xgoja jsverb site"))
  app.get("/healthz").public().handle((_ctx, res) => res.json({ ok: true, site: "demo" }))
}
```

Run the smoke test:

```bash
make -C examples/xgoja/13-http-serve-jsverbs smoke
```

Manual run:

```bash
make -C examples/xgoja/13-http-serve-jsverbs build
./examples/xgoja/13-http-serve-jsverbs/dist/http-serve-jsverbs \
  serve sites demo --http-listen 127.0.0.1:8787
```

Then open:

- <http://127.0.0.1:8787/>
- <http://127.0.0.1:8787/healthz>

Stop the server with Ctrl-C.

Development hot reload is opt-in:

```bash
./examples/xgoja/13-http-serve-jsverbs/dist/http-serve-jsverbs \
  serve sites demo \
  --http-listen 127.0.0.1:8787 \
  --hot-reload \
  --hot-reload-watch-root examples/xgoja/13-http-serve-jsverbs/verbs \
  --hot-reload-smoke-path /healthz
```

While hot reload is enabled, xgoja keeps one Go HTTP listener alive, reloads the
JavaScript runtime from watched source files, swaps successful candidates live,
and keeps serving the last-known-good runtime after broken edits. Inspect
`http://127.0.0.1:8787/__xgoja/status` for the active version, route list, and
latest reload error.
