# Express auth host example

This example is a runnable Go-owned host for the planned Express auth framework. It complements `examples/xgoja/15-express-planned-auth`, which is only a JavaScript route-authoring sketch.

The Go host wires the services that authenticated planned routes require:

- `Authenticator` accepts `Authorization: Bearer demo-user` and returns actor `u1`.
- `ResourceResolver` resolves project `p1` in tenant `o1`.
- `Authorizer` allows `user.self.read` and `project.update` for the demo actor/resource.
- `CSRFProtector` requires `X-CSRF-Token: demo-csrf` on unsafe `.csrf()` routes.
- `AuditSink` logs route audit events.
- `RejectRawRoutes` is enabled so only planned routes are served.

The JavaScript routes live in `scripts/server.js`:

```js
app.get("/healthz")
  .public()
  .audit("health.checked")
  .handle((_ctx, res) => res.json({ ok: true }))

app.get("/me")
  .auth(express.user().required())
  .allow("user.self.read")
  .audit("user.self.read")
  .handle((ctx, res) => res.json({ id: ctx.actor.id, kind: ctx.actor.kind }))

app.patch("/orgs/:orgId/projects/:projectId")
  .auth(express.user().required())
  .resource(express.resource("project").idFromParam("projectId").tenantFromParam("orgId").mustExist())
  .csrf()
  .allow("project.update")
  .audit("project.updated")
  .handle((ctx, res) => res.json({ updated: ctx.resource("project").id }))
```

Run the smoke test:

```bash
make -C examples/xgoja/16-express-auth-host smoke
```

The smoke test starts an in-process HTTP server and checks:

- public health route returns 200,
- unauthenticated `/me` returns 401,
- authenticated `/me` returns actor data,
- missing CSRF token returns 403,
- valid authenticated project update returns 200,
- missing project returns 404.

Manual run:

```bash
make -C examples/xgoja/16-express-auth-host serve
```

Then try:

```bash
curl -i http://127.0.0.1:18789/healthz
curl -i http://127.0.0.1:18789/me
curl -i -H 'Authorization: Bearer demo-user' http://127.0.0.1:18789/me
curl -i -X PATCH \
  -H 'Authorization: Bearer demo-user' \
  -H 'X-CSRF-Token: demo-csrf' \
  http://127.0.0.1:18789/orgs/o1/projects/p1
```
