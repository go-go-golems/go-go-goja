__package__({ name: "sites" });
__verb__("demo", { name: "demo", short: "Serve generated-host auth demo", output: "text" });

function demo() {
  const express = require("express");
  const app = express.app();

  app.get("/")
    .public()
    .handle((_ctx, res) => res.type("text/plain").send("generated host auth demo"));

  app.get("/healthz")
    .public()
    .handle((_ctx, res) => res.json({ ok: true, auth: "generated-host" }));

  app.get("/me")
    .auth(express.user().required())
    .allow("user.self.read")
    .handle((ctx, res) => res.json({ actor: ctx.actor.id, action: ctx.action }));
}
