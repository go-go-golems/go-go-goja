__package__({ name: "agentauth", short: "Programmatic agent auth server demo" });

__verb__("server", {
  name: "server",
  output: "text",
  short: "Expose agent-authenticated report routes and provision a local demo token",
  fields: {
    options: { bind: "all" },
    tokenFile: {
      help: "Path where the server writes the one-time bootstrap token metadata",
      default: "/tmp/xgoja-agent-auth-demo-token.json"
    }
  }
});

function server(options) {
  const express = require("express");
  const auth = require("auth");
  const fs = require("fs:host");
  const app = express.app();

  const tokenFile = options.tokenFile || "/tmp/xgoja-agent-auth-demo-token.json";
  const reports = {
    daily: {
      id: "daily",
      tenantId: "o1",
      title: "Daily revenue report",
      rows: [
        { metric: "bookings", value: 42 },
        { metric: "revenue", value: 12345 }
      ]
    }
  };

  const issued = auth.agents.create("daily-report-bot")
    .kind("ci")
    .tenantId("o1")
    .createdBy("server-bootstrap")
    .grants(auth.grants().allow("user.self.read").done())
    .issueApiToken("daily-report-token")
    .run();

  fs.writeFileSync(tokenFile, JSON.stringify({
    agent: issued.agent,
    token: issued.token,
    note: "Raw token value is returned only at issuance. Treat this file like a secret."
  }, null, 2), "utf8");

  app.get("/healthz")
    .public()
    .audit("health.checked")
    .handle((_ctx, res) => res.json({ ok: true, example: "programmatic-agent-auth" }));

  app.get("/agent/reports/:reportId")
    .auth(express.agent())
    .rateLimit(express.rateLimit("agent-report-read").perMinute(60).byActor().byRoute())
    .allow("user.self.read")
    .audit("agent.report.read")
    .handle((ctx, res) => {
      const report = reports[ctx.params.reportId];
      if (!report) {
        res.status(404).json({ error: "unknown report" });
        return;
      }
      res.json({
        ok: true,
        report,
        actor: ctx.actor,
        auth: ctx.auth
      });
    });

  app.get("/session-only")
    .auth(express.sessionUser())
    .allow("user.self.read")
    .audit("session.only.read")
    .handle((_ctx, res) => res.json({ ok: true, sessionOnly: true }));

  return `programmatic agent auth server ready; bootstrap token file: ${tokenFile}\n`;
}
