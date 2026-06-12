const express = require("express")

const app = express.app()

// Public planned routes still declare that they are public before registering a handler.
app.route("GET", "/healthz")
  .public()
  .handle((_ctx, res) => res.json({ ok: true }))

// Current-user routes authenticate an actor and declare an authorization action.
app.route("GET", "/me")
  .auth(express.user().required())
  .allow("user.self.read")
  .handle((ctx, res) => {
    res.json({ id: ctx.actor.id, kind: ctx.actor.kind })
  })

// Resource routes bind HTTP adapter values to a Go-owned resource request.
// The host ResourceResolver receives projectId and orgId as typed inputs; the
// JavaScript handler does not perform resource lookup or access control itself.
app.route("PATCH", "/orgs/:orgId/projects/:projectId")
  .auth(express.user().required())
  .resource(
    express.resource("project")
      .idFromParam("projectId")
      .tenantFromParam("orgId")
      .mustExist()
  )
  .allow("project.update")
  .handle((ctx, res) => {
    const project = ctx.resource("project")
    res.json({ updated: project.id, tenant: project.tenantId })
  })
