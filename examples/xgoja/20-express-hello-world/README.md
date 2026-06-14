# Express hello world example

This is the smallest runnable Go-owned Express host in the xgoja examples. It has no auth/session/CSRF/resource/audit infrastructure. Routes are public-only, but they still use the planned route builder API, so each route explicitly calls `.public()` before `.handle(...)`.

The JavaScript routes live in `scripts/server.js`:

```js
const express = require("express")

const app = express.app()

app.get("/")
  .public()
  .handle(() => "Hello, world!")

app.get("/hello/:name")
  .public()
  .handle((ctx, res) => {
    res.json({ message: `Hello, ${ctx.request.params.name}!` })
  })
```

Run the smoke test:

```bash
make -C examples/xgoja/20-express-hello-world smoke
```

Run the server manually:

```bash
make -C examples/xgoja/20-express-hello-world serve
```

Then try:

```bash
curl -i http://127.0.0.1:18790/
curl -i http://127.0.0.1:18790/hello/goja
curl -i http://127.0.0.1:18790/healthz
```

Important: the old raw Express style is intentionally not used here:

```js
app.get("/", (_req, res) => res.send("Hello"))
```

Use the planned-route form instead:

```js
app.get("/").public().handle(() => "Hello")
```
