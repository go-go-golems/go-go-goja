const express = require("express")
const timer = require("timer")
const app = express.app()

app.get("/healthz")
  .public()
  .audit("health.checked")
  .handle((_ctx, res) => res.json({ ok: true, example: "keycloak-auth-host" }))

app.get("/async-return")
  .public()
  .audit("async.returned")
  .handle(async (ctx, _res) => {
    await timer.sleep(5)
    return `async return ${ctx.request.query.name || "anonymous"}`
  })

app.get("/async-send")
  .public()
  .audit("async.sent")
  .handle(async (ctx, res) => {
    await timer.sleep(5)
    res.json({ ok: true, mode: "send", name: ctx.request.query.name || "anonymous" })
  })

app.get("/me")
  .auth(express.user().required())
  .allow("user.self.read")
  .audit("user.self.read")
  .handle((ctx, res) => {
    res.json({ id: ctx.actor.id, kind: ctx.actor.kind, claims: ctx.actor.claims || {} })
  })

app.patch("/orgs/:orgId/projects/:projectId")
  .auth(express.user().required())
  .resource(
    express.resource("project")
      .idFromParam("projectId")
      .tenantFromParam("orgId")
      .mustExist()
  )
  .csrf()
  .allow("project.update")
  .audit("project.updated")
  .handle((ctx, res) => {
    const project = ctx.resource("project")
    res.json({ updated: project.id, tenant: project.tenantId })
  })
