const express = require("express")
const assets = require("fs:assets")
const host = require("fs:host")

const mount = "/app/public"
const stagedRoot = ".xgoja-static"
const stagedPublic = stagedRoot + "/public"

function join(left, right) {
  if (left === "" || left === "/") return "/" + right
  return left + "/" + right
}

function copyEmbeddedTree(src, dst) {
  host.mkdirSync(dst, { recursive: true })
  for (const name of assets.readdirSync(src)) {
    const srcPath = join(src, name)
    const dstPath = dst + "/" + name
    const stat = assets.statSync(srcPath)
    if (stat.isDir) {
      copyEmbeddedTree(srcPath, dstPath)
      continue
    }
    host.writeFileSync(dstPath, assets.readFileSync(srcPath))
  }
}

host.rmSync(stagedRoot, { recursive: true, force: true })
copyEmbeddedTree(mount, stagedPublic)

const app = express.app()
app.static("/static", stagedPublic)
app.get("/", (_req, res) => res.redirect("/static/"))
app.get("/api/config", (_req, res) => {
  const config = JSON.parse(assets.readFileSync("/app/config/default.json", "utf8"))
  res.json({ ok: true, config })
})

console.log("serving embedded static assets")
console.log("open http://127.0.0.1:8787/static/")
console.log("try  http://127.0.0.1:8787/api/config")
