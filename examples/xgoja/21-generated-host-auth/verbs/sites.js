__package__({ name: "sites" });
__verb__("demo", { name: "demo", short: "Serve generated OIDC host auth demo", output: "text" });

function demo() {
  const express = require("express");
  const timer = require("timer");
  const assets = require("fs:assets");
  const auth = require("auth");
  const app = express.app();

  app.staticFromAssetsModule("/static", assets, "/app/public");

  app.get("/")
    .public()
    .handle((_ctx, res) => res.type("text/html").send(assets.readFileSync("/app/public/index.html", "utf8")));

  app.get("/healthz")
    .public()
    .audit("health.checked")
    .handle((_ctx, res) => res.json({ ok: true, auth: "generated-oidc", example: "generated-oidc-host-auth" }));

  app.get("/async-return")
    .public()
    .audit("async.returned")
    .handle(async (ctx, _res) => {
      await timer.sleep(5);
      return `async return ${ctx.request.query.name || "anonymous"}`;
    });

  app.get("/async-send")
    .public()
    .audit("async.sent")
    .handle(async (ctx, res) => {
      await timer.sleep(5);
      res.json({ ok: true, mode: "send", name: ctx.request.query.name || "anonymous" });
    });

  app.get("/me")
    .auth(express.user().required())
    .allow("user.self.read")
    .audit("user.self.read")
    .handle((ctx, res) => {
      res.json({ id: ctx.actor.id, kind: ctx.actor.kind, claims: ctx.actor.claims || {} });
    });

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
      const project = ctx.resource("project");
      res.json({ updated: project.id, tenant: project.tenantId });
    });

  app.post("/orgs/:orgId/invites")
    .auth(express.user().required())
    .resource(
      express.resource("org")
        .idFromParam("orgId")
        .mustExist()
    )
    .csrf()
    .allow("org.member.invite")
    .audit("org.invite.issued")
    .handle((ctx, res) => {
      const org = ctx.resource("org");
      const body = ctx.body || {};
      const issued = auth.capabilities.issue("org-invite")
        .resource("org", org.id)
        .tenantId(org.id)
        .claimString("email", body.email || "")
        .claimString("role", body.role || "viewer")
        .ttlSeconds(900)
        .singleUse(true)
        .createdBy(ctx.actor.id)
        .run();
      res.json({ token: issued.token, expiresAt: issued.capability.expiresAt, capabilityId: issued.capability.id });
    });

  app.post("/org-invites/accept")
    .public()
    .audit("org.invite.accepted")
    .handle((ctx, res) => {
      const body = ctx.body || {};
      const accepted = auth.capabilities.consume(body.token || "")
        .expectedType("org-invite")
        .expectedResource("org", "o1")
        .run();
      res.json({
        capabilityId: accepted.id,
        orgId: accepted.resourceId,
        email: accepted.claims.email,
        role: accepted.claims.role
      });
    });

  app.get("/orgs/:orgId/audit")
    .auth(express.user().required())
    .resource(
      express.resource("org")
        .idFromParam("orgId")
        .mustExist()
    )
    .allow("audit.read")
    .audit("audit.records.read")
    .handle((ctx, res) => {
      const org = ctx.resource("org");
      const records = auth.audit.query()
        .tenantId(org.id)
        .outcome(ctx.request.query.outcome || "")
        .limit(Number(ctx.request.query.limit || 50))
        .run();
      res.json({ records, count: records.length });
    });
}
