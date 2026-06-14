const express = require("express")
const assets = require("fs:assets")

const app = express.app()
app.staticFromAssetsModule("/static", assets, "/app/public")
app.get("/").public().handle((_ctx, res) => res.redirect("/static/"))
app.get("/api/config").public().handle((_ctx, res) => {
  const config = JSON.parse(assets.readFileSync("/app/config/default.json", "utf8"))
  res.json({ ok: true, config })
})

console.log("serving embedded static assets")
console.log("open http://127.0.0.1:8787/static/")
console.log("try  http://127.0.0.1:8787/api/config")
