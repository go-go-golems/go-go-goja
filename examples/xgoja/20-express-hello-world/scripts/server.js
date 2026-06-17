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

app.get("/healthz")
  .public()
  .handle((_ctx, res) => res.json({ ok: true }))
