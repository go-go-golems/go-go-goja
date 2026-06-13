# Express auth host example

This example is a runnable Go-owned host for the planned Express auth framework. It complements `examples/xgoja/17-express-planned-auth`, which is only a JavaScript route-authoring sketch.

The host now uses `pkg/gojahttp/auth/devauth`, a no-external-service development auth kit that implements the same `gojahttp.AuthOptions` interfaces a production host would implement.

The dev auth kit provides:

- in-memory user `demo@example.test` / `demo-password`,
- in-memory session cookie authentication,
- session-bound CSRF token verification,
- in-memory project resource `p1` in tenant `o1`,
- explicit demo authorization for `user.self.read` and `project.update`,
- in-memory/logged audit events,
- `RejectRawRoutes` enabled so only planned routes are served.

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
make -C examples/xgoja/18-express-auth-host smoke
```

The smoke test starts an in-process HTTP server and checks:

- public health route returns 200,
- `/me` before login returns 401,
- bad login returns 401,
- good login returns a session cookie and CSRF token,
- `/me` after login returns actor data,
- project update without CSRF returns 403,
- project update with session cookie and CSRF token returns 200,
- missing project returns 404,
- logout clears the session,
- `/me` after logout returns 401.

Manual run:

```bash
make -C examples/xgoja/18-express-auth-host serve
```

Then try:

```bash
# Public route.
curl -i http://127.0.0.1:18789/healthz

# Login. Save the session cookie and copy csrfToken from the response.
curl -i -c /tmp/goja-dev-auth.cookie \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo@example.test","password":"demo-password"}' \
  http://127.0.0.1:18789/auth/dev/login

# Authenticated current-user route.
curl -i -b /tmp/goja-dev-auth.cookie \
  http://127.0.0.1:18789/me

# Unsafe authenticated route. Replace TOKEN with csrfToken from login.
curl -i -X PATCH \
  -b /tmp/goja-dev-auth.cookie \
  -H 'X-CSRF-Token: TOKEN' \
  http://127.0.0.1:18789/orgs/o1/projects/p1

# Logout. Replace TOKEN with csrfToken from login.
curl -i -X POST \
  -b /tmp/goja-dev-auth.cookie \
  -H 'X-CSRF-Token: TOKEN' \
  http://127.0.0.1:18789/auth/dev/logout
```

This is intentionally a development/demo auth system, not a production identity stack. The production track should use Keycloak/OIDC, server-side persistent sessions, app-owned users/tenants/memberships, persistent audit, and optional capability tokens.
