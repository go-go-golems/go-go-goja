const path = require("path")
const fs = require("fs")

const file = path.join(".", "multiple-runtimes-host.txt")
fs.writeFileSync(file, "host runtime ok", "utf8")
const value = fs.readFileSync(file, "utf8")
fs.unlinkSync(file)

if (value !== "host runtime ok") {
  throw new Error("host runtime fs smoke failed")
}

console.log("multiple runtimes host run ok")
