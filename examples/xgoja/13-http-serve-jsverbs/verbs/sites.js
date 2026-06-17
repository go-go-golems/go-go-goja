__package__({ name: "sites", short: "HTTP site setup verbs" })

__verb__("demo", {
  name: "demo",
  output: "text",
  short: "Serve a tiny Express-backed demo site",
  tags: ["http", "site"]
})
function demo() {
  const express = require("express")
  const app = express.app()

  app.get("/").public().handle((_ctx, res) => res.send("hello from an xgoja jsverb site"))
  app.get("/healthz").public().handle((_ctx, res) => res.json({ ok: true, site: "demo" }))
}
