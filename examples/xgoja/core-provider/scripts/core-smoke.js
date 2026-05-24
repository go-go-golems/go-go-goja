const path = require("path")
const yaml = require("yaml")
const crypto = require("crypto")

if (path.basename("/tmp/core-provider.txt") !== "core-provider.txt") {
  throw new Error("path.basename failed")
}

const parsed = yaml.parse("name: core\ncount: 2\n")
if (parsed.name !== "core" || parsed.count !== 2) {
  throw new Error("yaml.parse failed")
}

const digest = crypto.createHash("sha256").update("xgoja").digest("hex")
if (digest.length !== 64) {
  throw new Error("crypto sha256 digest length failed")
}

console.log("core provider ok")
