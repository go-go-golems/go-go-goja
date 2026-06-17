__package__({ name: "sites" });
__verb__("demo", { name: "demo", short: "Serve generated OIDC host auth demo", output: "text" });

function demo() {
  const express = require("express");
  const timer = require("timer");
  const app = express.app();

  app.get("/")
    .public()
    .handle((_ctx, res) => res.type("text/html").send(landingPage()));

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
}

function landingPage() {
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>go-go-goja generated OIDC auth host</title>
  <style>
    :root { color-scheme: dark; --bg:#0f172a; --panel:#111827; --muted:#94a3b8; --text:#e5e7eb; --accent:#38bdf8; --ok:#34d399; --bad:#fb7185; }
    body { margin:0; font:15px/1.5 system-ui,-apple-system,Segoe UI,sans-serif; background:radial-gradient(circle at top left,#1e3a8a 0,#0f172a 36rem); color:var(--text); }
    main { max-width:1100px; margin:0 auto; padding:40px 20px; }
    h1 { font-size:clamp(32px,5vw,58px); line-height:1; margin:0 0 12px; letter-spacing:-0.04em; }
    h2 { margin:0 0 12px; }
    p { color:var(--muted); }
    a { color:var(--accent); }
    .hero { display:grid; gap:18px; margin-bottom:24px; }
    .badge { display:inline-flex; gap:8px; align-items:center; width:max-content; padding:6px 10px; border:1px solid #334155; border-radius:999px; background:#020617aa; color:#bae6fd; }
    .grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(260px,1fr)); gap:16px; }
    .card { background:linear-gradient(180deg,#111827ee,#020617dd); border:1px solid #334155; border-radius:18px; padding:18px; box-shadow:0 18px 50px #0006; }
    .actions { display:flex; flex-wrap:wrap; gap:10px; margin-top:14px; }
    button,.button { border:1px solid #475569; background:#0f172a; color:var(--text); padding:9px 12px; border-radius:10px; cursor:pointer; text-decoration:none; }
    button.primary,.button.primary { background:#0284c7; border-color:#38bdf8; color:white; }
    button:hover,.button:hover { border-color:#7dd3fc; }
    pre { max-height:360px; overflow:auto; white-space:pre-wrap; word-break:break-word; background:#020617; border:1px solid #1f2937; border-radius:14px; padding:12px; color:#d1d5db; }
    .status { font-weight:700; }
    .ok { color:var(--ok); } .bad { color:var(--bad); }
    ul { padding-left:20px; color:var(--muted); }
  </style>
</head>
<body>
<main>
  <section class="hero">
    <span class="badge">generated xgoja serve · native OIDC · server-side sessions</span>
    <h1>go-go-goja auth host</h1>
    <p>This public demo is served by a binary generated from <code>xgoja.yaml</code>. Login is handled by native Go OIDC routes, while the app routes are declared in JavaScript.</p>
    <div class="actions">
      <a class="button primary" href="/auth/login">Login with Keycloak</a>
      <button onclick="postLogout()">Logout</button>
      <a class="button" href="/healthz">Health JSON</a>
      <a class="button" href="/me">/me</a>
      <a class="button" href="/auth/session">/auth/session</a>
      <a class="button" href="/auth/audit">/auth/audit</a>
    </div>
  </section>

  <section class="grid">
    <article class="card">
      <h2>Session</h2>
      <p id="session-status" class="status">Checking session…</p>
      <div class="actions">
        <button onclick="loadSession()">Refresh session</button>
        <button onclick="loadMe()">Load /me</button>
      </div>
      <pre id="session-output"></pre>
    </article>

    <article class="card">
      <h2>Protected project update</h2>
      <p>Uses the app session, CSRF token, resource lookup, authorization, and audit pipeline.</p>
      <div class="actions">
        <button onclick="updateProject()">PATCH project p1</button>
        <button onclick="updateMissingProject()">PATCH missing project</button>
      </div>
      <pre id="project-output"></pre>
    </article>

    <article class="card">
      <h2>Invite capability</h2>
      <p>Issues a single-use org invite capability and redeems it once.</p>
      <div class="actions">
        <button onclick="issueInvite()">Issue invite</button>
        <button onclick="acceptInvite()">Accept invite</button>
        <button onclick="acceptInvite()">Accept again</button>
      </div>
      <pre id="invite-output"></pre>
    </article>

    <article class="card">
      <h2>Recent audit records</h2>
      <p>Shows the last records from the configured audit store. You must be logged in.</p>
      <div class="actions"><button onclick="loadAudit()">Load audit log</button></div>
      <pre id="audit-output"></pre>
    </article>
  </section>
</main>
<script>
let csrfToken = "";
let lastInviteToken = "";
async function fetchText(url, options) {
  const res = await fetch(url, options);
  const text = await res.text();
  let body = text;
  try { body = JSON.parse(text); } catch (_) {}
  return { status: res.status, ok: res.ok, body };
}
function show(id, value) { document.getElementById(id).textContent = typeof value === "string" ? value : JSON.stringify(value, null, 2); }
async function loadSession() {
  const out = await fetchText("/auth/session");
  if (out.ok) {
    csrfToken = out.body.csrfToken || "";
    document.getElementById("session-status").innerHTML = '<span class="ok">Logged in</span>';
  } else {
    document.getElementById("session-status").innerHTML = '<span class="bad">Not logged in</span>';
  }
  show("session-output", out);
}
async function loadMe() { show("session-output", await fetchText("/me")); }
async function postLogout() { show("session-output", await fetchText("/auth/logout", { method:"POST" })); csrfToken = ""; await loadSession(); }
async function updateProject() {
  if (!csrfToken) await loadSession();
  show("project-output", await fetchText("/orgs/o1/projects/p1", { method:"PATCH", headers:{ "X-CSRF-Token": csrfToken } }));
}
async function updateMissingProject() {
  if (!csrfToken) await loadSession();
  show("project-output", await fetchText("/orgs/o1/projects/missing", { method:"PATCH", headers:{ "X-CSRF-Token": csrfToken } }));
}
async function issueInvite() {
  if (!csrfToken) await loadSession();
  const out = await fetchText("/orgs/o1/invites", { method:"POST", headers:{ "Content-Type":"application/json", "X-CSRF-Token": csrfToken }, body: JSON.stringify({ email:"invitee@example.test", role:"viewer" }) });
  if (out.ok) lastInviteToken = out.body.token || "";
  show("invite-output", out);
}
async function acceptInvite() {
  show("invite-output", await fetchText("/org-invites/accept", { method:"POST", headers:{ "Content-Type":"application/json" }, body: JSON.stringify({ token:lastInviteToken }) }));
}
async function loadAudit() { show("audit-output", await fetchText("/auth/audit")); }
loadSession();
</script>
</body>
</html>`;
}
