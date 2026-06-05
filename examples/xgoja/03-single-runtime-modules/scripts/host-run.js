const path = require("path")
const fs = require("fs")

const file = path.join(".", "single-runtime-modules-host.txt")
fs.writeFileSync(file, "single runtime fs ok", "utf8")
const value = fs.readFileSync(file, "utf8")
fs.unlinkSync(file)

if (value !== "single runtime fs ok") {
  throw new Error("single runtime fs smoke failed")
}

console.log("single runtime modules host run ok")
