# xgoja HTTP serve jsverbs example

This example builds a generated xgoja binary that exposes a provider-backed
`serve` command from `go-go-goja-http`. The command mirrors configured
JavaScript verbs, invokes the selected verb once to register Express routes, and
keeps the runtime alive until the process is stopped.

The site setup verb lives in `verbs/sites.js`:

```js
__package__({ name: "sites" })
__verb__("demo", { name: "demo", output: "text" })
function demo() {
  const express = require("express")
  const app = express.app()
  app.get("/", (_req, res) => res.send("hello from an xgoja jsverb site"))
  app.get("/healthz", (_req, res) => res.json({ ok: true, site: "demo" }))
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
