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

  app.get("/", (_req, res) => res.send("hello from an xgoja jsverb site"))
  app.get("/healthz", (_req, res) => res.json({ ok: true, site: "demo" }))
}
