const assets = require("fs:assets")
const host = require("fs:host")

let plain = ""
try {
  require("fs")
} catch (e) {
  plain = "missing"
}
if (plain !== "missing") {
  throw new Error('expected require("fs") to be unavailable when only fs:assets and fs:host are registered')
}

const outPath = "out.json"

const text = assets.readFileSync("/app/config/default.json", "utf8")
const parsed = JSON.parse(text)
if (parsed.name !== "embedded-assets-fs" || parsed.ok !== true) {
  throw new Error("unexpected embedded asset payload: " + text)
}

host.writeFileSync(outPath, JSON.stringify(parsed), "utf8")
console.log("embedded assets ok")
