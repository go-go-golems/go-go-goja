---
Title: Migrate Express Apps to Planned Auth Routes
Slug: migrate-express-apps-to-planned-auth
Short: Update old Express-style Goja route handlers to the explicit planned auth route API.
Topics:
- http
- express
- auth
- migration
- javascript
Commands:
- xgoja
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial migrates an existing `go-go-goja` Express-style HTTP script from raw two-argument handlers to planned auth routes. The end state keeps `app.get`, `app.post`, and the other verb names, but every route explicitly declares `.public()` or `.auth(...).allow(...)` before `.handle(...)`.

## What you will build

You will take a script that uses the removed `app.get(pattern, handler)` and `app.post(pattern, handler)` overloads and convert it to the planned route API. The migrated script has three routes:

- `GET /healthz` — public health check.
- `GET /me` — authenticated current-user endpoint.
- `PATCH /orgs/:orgId/projects/:projectId` — authenticated resource-bound update endpoint.

The tutorial focuses on JavaScript route authoring. Authenticated routes also require the embedding Go host to provide `Authenticator`, `ResourceResolver`, and `Authorizer` services.

## Prerequisites

You need a runtime that provides the `express` module through `modules/express.NewRegistrar(host)` or the `go-go-goja-http` xgoja provider. You also need to know whether each existing endpoint is intentionally public or protected. Do not migrate everything to `.public()` just to make the script load; public declarations should reflect real exposure decisions.

## Step 1 — Identify old raw handlers

Old route code passed a handler directly as the second argument. That shape is removed because it does not encode security intent.

```javascript
const express = require("express")
const app = express.app()

app.get("/healthz", (_req, res) => {
  res.json({ ok: true })
})

app.post("/api/echo", (req, res) => {
  res.status(201).json({ body: req.body })
})
```

Search for old handlers with a text search before editing:

```bash
rg 'app\.(get|post|put|patch|delete|all)\([^\n)]*,\s*(async\s*)?\('
```

That pattern finds most direct inline handlers. It may miss handlers stored in variables, so also search for `app.get(` and review each route manually.

## Step 2 — Convert public routes

Public routes now call the verb helper with only the pattern, then call `.public()`, then register the handler with `.handle(...)`.

```javascript
app.get("/healthz")
  .public()
  .handle((_ctx, res) => {
    res.json({ ok: true })
  })
```

This is not just syntax. `.public()` records that the route is intentionally reachable without an actor. Code review can now distinguish public exposure from accidental omission of authentication.

## Step 3 — Replace `req` with `ctx`

Planned handlers receive `(ctx, res)`, not `(req, res)`. The most common fields move to top-level context properties, while the full request DTO remains under `ctx.request`.

| Old raw handler | Planned handler |
| --- | --- |
| `req.params.id` | `ctx.params.id` |
| `req.body` | `ctx.body` |
| `req.query.name` | `ctx.request.query.name` |
| `req.session.id` | `ctx.request.session.id` |
| `req.headers.authorization` | `ctx.request.headers.authorization` |

For example, a raw echo route becomes a public planned route:

```javascript
app.post("/api/echo")
  .public()
  .handle((ctx, res) => {
    res.status(201).json({ body: ctx.body })
  })
```

If the route uses request data for access control, do not keep that logic in JavaScript. Declare the resource and action, then let the Go host resolve and authorize it.

## Step 4 — Convert current-user routes

A route that requires a signed-in user calls `.auth(express.user().required())`, declares an action with `.allow(action)`, and then registers the handler.

```javascript
app.get("/me")
  .auth(express.user().required())
  .allow("user.self.read")
  .handle((ctx, res) => {
    res.json({ id: ctx.actor.id, kind: ctx.actor.kind })
  })
```

`ctx.actor` is populated by the host `Authenticator`. The JavaScript route does not decide how users are stored or how sessions are validated. The action string is passed to the host `Authorizer`.

## Step 5 — Convert resource-bound routes

Routes that touch a domain object should declare a resource. The route specifies where the resource ID and tenant boundary come from, and the Go host performs the lookup and authorization.

```javascript
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
```

The parameter names must match the path exactly. `/projects/:projectId` pairs with `.idFromParam("projectId")`, not `.idFromParam("id")`. Mismatches fail at registration time.

## Step 6 — Add CSRF and audit declarations where needed

Routes that accept unsafe browser requests should declare CSRF protection before `.handle(...)`. Routes that represent security-relevant domain operations should declare an audit event.

```javascript
app.post("/projects")
  .auth(express.user().required())
  .csrf()
  .allow("project.create")
  .audit("project.created")
  .handle((ctx, res) => {
    res.status(201).json({ actor: ctx.actor.id })
  })
```

`.csrf()` is enforced by the host `CSRFProtector` on unsafe methods. `.audit(event)` emits structured audit events through the host `AuditSink` for allowed, denied, completed, and failed outcomes. Do not use JavaScript-side logging as the only audit trail for protected routes; host-owned audit events include actor, action, route, resource, status, and denial reason.

## Step 7 — Wire host auth services for protected routes

Public planned routes work without auth services. Protected planned routes fail closed unless the Go host provides the required services.

```go
host := gojahttp.NewHost(gojahttp.HostOptions{
    Dev:             true,
    RejectRawRoutes: true,
    Auth: gojahttp.AuthOptions{
        Authenticator: myAuthenticator,
        Resources:     myResourceResolver,
        Authorizer:    myAuthorizer,
        CSRF:          myCSRFProtector,
        Audit:         myAuditSink,
    },
})
```

For an initial test, use small in-memory implementations. The important contract is that `Authenticator` returns an `Actor`, `ResourceResolver` returns a `ResourceRef`, `Authorizer` returns an `AuthorizationDecision`, `CSRFProtector` verifies declared CSRF protection, and `AuditSink` records declared audit events. `RejectRawRoutes` is optional during migration but useful once the route inventory is clean because it rejects any matched low-level route that lacks a `RoutePlan`.

## Step 8 — Run migration checks

Run the focused package tests after updating code and docs:

```bash
go test ./modules/express ./pkg/gojahttp ./pkg/xgoja/providers/http -count=1
```

Then run the broader suite with VCS stamping disabled if generated build tests run from temporary directories:

```bash
GOFLAGS=-buildvcs=false go test ./... -count=1
```

Finally, search for old direct-handler routes again:

```bash
rg 'app\.(get|post|put|patch|delete|all)\([^\n)]*,\s*(async\s*)?\('
```

Only intentional migration notes or rejection tests should remain.

## Complete migrated example

This example shows the final JavaScript shape after migration.

```javascript
const express = require("express")
const app = express.app()

app.get("/healthz")
  .public()
  .handle((_ctx, res) => res.json({ ok: true }))

app.get("/me")
  .auth(express.user().required())
  .allow("user.self.read")
  .handle((ctx, res) => {
    res.json({ id: ctx.actor.id, kind: ctx.actor.kind })
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
```

Use `app.route(method, pattern)` only when the method is dynamic or not covered by the verb helpers:

```javascript
app.route("REPORT", "/reports/:id")
  .auth(express.user().required())
  .allow("report.read")
  .handle((ctx, res) => res.json({ id: ctx.params.id }))
```

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `app.get(pattern, handler) was removed` | The route still uses the old two-argument overload. | Change it to `app.get(pattern).public().handle(handler)` or an auth-aware chain. |
| `.handle is not a function` | The builder is still in the security or policy stage. | Add `.public()` or complete `.auth(...).allow(...)` before `.handle(...)`. |
| Route returns 500 after migration | The route is protected but the Go host lacks auth services. | Configure `gojahttp.HostOptions.Auth`. |
| `ctx.actor` is null | The route is public or authentication did not run. | Use `.auth(express.user().required())` for routes that need an actor. |
| `ctx.resource("project")` returns null | The route did not declare that resource name. | Add `.resource(express.resource("project")...)` before `.allow(...)`. |
| `.csrf()` route returns 500 | The host has no `Auth.CSRF` service. | Configure a `CSRFProtector` or remove `.csrf()` from routes that do not need it. |
| Audit events do not appear | The route does not call `.audit(event)` or the host has no `Auth.Audit` sink. | Add `.audit("domain.event")` and configure an `AuditSink`. |
| `raw routes disabled` | `RejectRawRoutes` is enabled and the host matched a route registered outside the planned route path. | Convert the route to Express planned builders or `Host.RegisterPlanned`. |
| Parameter validation fails | The resource builder references a missing path parameter. | Make `.idFromParam(...)` and `.tenantFromParam(...)` match the route pattern exactly. |

## See Also

- [Express Auth User Guide](express-auth-user-guide) — Conceptual guide to planned auth routes and host services.
- [Express-style HTTP Module](express-module) — General module reference for static mounts, route patterns, response helpers, and planned routes.
- `examples/xgoja/17-express-planned-auth/scripts/server.js` — Compact planned auth route example.
- `examples/xgoja/18-express-auth-host` — Runnable host smoke test for authenticated planned routes, CSRF, audit, and strict raw-route rejection.
