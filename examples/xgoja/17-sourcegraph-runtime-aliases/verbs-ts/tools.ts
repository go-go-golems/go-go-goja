import { format } from "./format"

const assets = require("fs:assets")
const express = require("express")

__package__({ name: "typed", short: "TypeScript sourcegraph coverage" })
__verb__("echo", { name: "echo", output: "text", short: "Read embedded config through a colon runtime alias" })

function echo(): string {
  const config = JSON.parse(assets.readFileSync("/app/config/default.json", "utf8"))
  return format(config.name + ":" + typeof express.app)
}
