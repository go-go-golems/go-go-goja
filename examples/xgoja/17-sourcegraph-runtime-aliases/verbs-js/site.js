const express = require("express")
const assets = require("fs:assets")
const { decorate } = require("./helper.js")

__package__({ name: "imports", short: "Runtime alias import coverage" })

__verb__("serve", {
  name: "serve",
  output: "text",
  short: "Serve a site that uses local imports, runtime aliases, and embedded assets"
})
function serve() {
  const app = express.app()

  app.get("/", (_req, res) => {
    res.type("text/html").send(assets.readFileSync("/app/public/index.html", "utf8"))
  })
  app.staticFromAssetsModule("/static", assets, "/app/public")
  app.get("/api/config", (_req, res) => {
    const config = JSON.parse(assets.readFileSync("/app/config/default.json", "utf8"))
    res.json({ ok: true, config, decorated: decorate("config") })
  })
  app.get("/healthz", (_req, res) => {
    res.json({ ok: true, site: "sourcegraph-runtime-aliases", decorated: decorate("health") })
  })
}
