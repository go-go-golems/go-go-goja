import { message } from "./message"

__package__({ name: "sites", short: "TypeScript demo sites" })
__verb__("demo", { name: "demo", output: "text" })

function demo(): void {
  const express = require("express")
  const app = express.app()
  const version = 1

  app.get("/", (_req: unknown, res: any) => {
    res.send(message("xgoja", version))
  })

  app.get("/healthz", (_req: unknown, res: any) => {
    res.json({ ok: true, site: "typescript-demo", version })
  })
}
