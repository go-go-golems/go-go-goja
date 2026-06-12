---
Title: Express Auth User Guide
Slug: express-auth-user-guide
Short: Configure and use Go-owned planned authentication for Express-style Goja HTTP routes.
Topics:
- http
- express
- auth
- security
- goja
- javascript
Commands:
- xgoja
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `express` auth framework makes every normal route explicit about its security mode. JavaScript authors keep the familiar `app.get`, `app.post`, and `app.patch` names, but those helpers now return staged planned-route builders instead of accepting raw `(req, res)` handlers. Go owns the route plan, validates it at registration time, and enforces authentication, resource resolution, and authorization before JavaScript handler code runs.

## Why planned auth routes exist

A route handler that manually reads cookies, headers, or session fields can forget a check or apply it inconsistently. Planned auth routes move that security decision into a Go-owned route plan. The JavaScript file declares intent, and the host decides whether the request has an actor, which resource is being touched, and whether the actor may perform the declared action.

The core rule is simple: a route cannot register a handler until it has declared whether it is public or protected.

```javascript
const express = require("express")
const app = express.app()

app.get("/healthz")
  .public()
  .handle((_ctx, res) => res.json({ ok: true }))

app.get("/me")
  .auth(express.user().required())
  .allow("user.self.read")
  .handle((ctx, res) => res.json({ id: ctx.actor.id }))
```

If you skip `.public()` or `.auth(...).allow(...)`, `.handle(...)` is not part of the current builder stage. If you pass a raw handler directly to `app.get(pattern, handler)`, the module throws a migration error instead of silently registering an unplanned route.

## Route builder stages

The builder stages encode the order required for a valid route. This makes invalid route declarations fail while the script loads, not on the first production request.

| Stage | Available methods | What it means |
| --- | --- | --- |
| `RouteNeedsSecurity` | `.name()`, `.public()`, `.auth()` | The route has a method and path but no security mode yet. |
| `RouteNeedsPolicy` | `.resource()`, `.allow()` | The route requires an actor and needs policy information before the handler can register. |
| `RouteNeedsHandler` | `.handle()` | The route has enough plan metadata to register. |

Use the verb helpers for normal HTTP methods:

```javascript
app.post("/projects")
  .auth(express.user().required())
  .allow("project.create")
  .handle((ctx, res) => res.status(201).json({ actor: ctx.actor.id }))
```

Use `app.route(method, pattern)` only when the method is dynamic or uncommon:

```javascript
app.route("REPORT", "/reports/:id")
  .auth(express.user().required())
  .allow("report.read")
  .handle((ctx, res) => res.json({ id: ctx.params.id }))
```

## Public routes

A public route is reachable without an actor, but it is still planned. Calling `.public()` records an explicit public security mode, which makes route lists and code reviews show that the exposure is intentional.

```javascript
app.get("/assets-manifest")
  .public()
  .handle((_ctx, res) => {
    res.json({ version: "2026-06-12" })
  })
```

The handler receives `(ctx, res)`. Public routes usually use `ctx.params`, `ctx.body`, or `ctx.request` for transport data. `ctx.actor` is `null` because no authenticated principal is required.

## User-authenticated routes

A user-authenticated route declares that the host must produce an actor before the handler runs. The current MVP supports the `express.user()` builder and `.required()` user mode.

```javascript
app.get("/me")
  .auth(express.user().required())
  .allow("user.self.read")
  .handle((ctx, res) => {
    res.json({
      id: ctx.actor.id,
      kind: ctx.actor.kind,
      tenantIds: ctx.actor.tenantIds || [],
    })
  })
```

The action string is application-owned. `go-go-goja` does not interpret `user.self.read` by itself. It passes the string to the host-provided `Authorizer`, together with the actor and any resolved resources.

Routes that need a recent MFA challenge can add `.mfaFresh(duration)` to the user spec. The duration is passed to the host authenticator as `SecuritySpec.MFAFreshWithin`; `sessionauth.Manager` enforces it against `Session.MFAAt` and rejects sessions with no MFA timestamp or a stale MFA timestamp.

```javascript
app.post("/billing/payment-methods")
  .auth(express.user().required().mfaFresh("10m"))
  .csrf()
  .allow("billing.payment_method.update")
  .handle((ctx, res) => res.json({ actor: ctx.actor.id }))
```

## Resource-bound routes

A resource-bound route declares how HTTP adapter values become typed inputs to the host resource resolver. The JavaScript route does not load the database row and does not decide whether the actor owns it. It only declares where the identifier and tenant boundary are located in the request.

```javascript
app.patch("/orgs/:orgId/projects/:projectId")
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
```

`idFromParam("projectId")` means the host reads `ctx.params.projectId` and passes the resolved string as `ResourceRequest.ID`. `tenantFromParam("orgId")` passes `ResourceRequest.TenantID`. If either parameter is missing from the route pattern, registration fails before the server starts serving that route.

## CSRF protection

CSRF protection is declared on the route plan and enforced by the Go host. Use `.csrf()` on unsafe browser/session routes that accept state-changing requests. The host `CSRFProtector` receives the HTTP request, planned request DTO, session, actor when present, and route plan before the JavaScript handler runs.

```javascript
app.post("/contact")
  .public()
  .csrf()
  .audit("contact.submitted")
  .handle((ctx, res) => {
    res.json({ received: true })
  })
```

For unsafe methods such as `POST`, `PUT`, `PATCH`, and `DELETE`, a route with `.csrf()` fails closed when the host has no `Auth.CSRF` service or when the verifier returns an error. CSRF failures return 403 and the JavaScript handler is not invoked. Safe methods such as `GET` do not call the verifier even if the route plan carries a CSRF flag.

## Audit events

Audit is also declared on the route plan. Use `.audit(event)` to ask the host to record structured security events around a planned request. The event name is application-owned and should describe the domain operation, such as `project.updated` or `contact.submitted`.

```javascript
app.patch("/orgs/:orgId/projects/:projectId")
  .auth(express.user().required())
  .resource(express.resource("project").idFromParam("projectId").tenantFromParam("orgId"))
  .allow("project.update")
  .csrf()
  .audit("project.updated")
  .handle((ctx, res) => {
    const project = ctx.resource("project")
    res.json({ updated: project.id })
  })
```

The host `AuditSink` receives events for `allowed`, `denied`, `completed`, and `failed` outcomes. Audit events include the route event name, route name, method, pattern, action, actor, first resource, all resolved resources, status code when known, and failure reason when present. Audit sink errors are best-effort in the MVP: they are ignored so an unavailable audit backend does not turn an otherwise successful request into a 500.

## Planned context

Planned handlers receive a context object instead of the old raw `req` object. The context exposes security state directly and keeps the original request DTO under `ctx.request`.

| Field | Meaning |
| --- | --- |
| `ctx.actor` | Authenticated actor or `null` for public routes. |
| `ctx.params` | Route parameters such as `{ projectId: "p1" }`. |
| `ctx.body` | Parsed request body. |
| `ctx.request` | Full request DTO: method, URL, query, headers, cookies, session, body, and raw body. |
| `ctx.resources` | Map of resolved resource references by name. |
| `ctx.resource(name)` | Convenience lookup that returns one resolved resource or `null`. |
| `ctx.action` | Action string declared by `.allow(action)`. |
| `ctx.routeName` | Optional name declared by `.name(name)`. |

Common migrations are mechanical:

| Old raw handler data | Planned handler data |
| --- | --- |
| `req.params.id` | `ctx.params.id` |
| `req.body` | `ctx.body` |
| `req.query.name` | `ctx.request.query.name` |
| `req.session.id` | `ctx.request.session.id` |
| `req.headers.authorization` | `ctx.request.headers.authorization` |

Use `ctx.actor` and `ctx.resource(...)` for security-sensitive logic. Use `ctx.request` for transport details that are not themselves an authorization decision.

## Host configuration

The JavaScript route only declares the plan. The embedding Go application provides auth services through `gojahttp.HostOptions.Auth`.

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

`RejectRawRoutes` is a production hardening option. It rejects any matched route that was registered without a `RoutePlan`, which keeps lower-level `Host.Register` callers from bypassing the planned auth framework.

The host interfaces are deliberately small:

```go
type Authenticator interface {
    Authenticate(ctx context.Context, req *http.Request, session *SessionDTO, spec SecuritySpec) (*Actor, error)
}

type ResourceResolver interface {
    ResolveResource(ctx context.Context, req ResourceRequest) (*ResourceRef, error)
}

type Authorizer interface {
    Authorize(ctx context.Context, req AuthorizationRequest) (AuthorizationDecision, error)
}
```

`Authenticator` converts a request, session, bearer token, cookie, or upstream identity into an `Actor`. `ResourceResolver` converts declared resource IDs into `ResourceRef` values. `Authorizer` decides whether the actor may perform the declared action. `CSRFProtector` verifies route-declared CSRF protection on unsafe methods. `AuditSink` records route-declared security events.

## Host auth package choices

The optional packages under `pkg/gojahttp/auth/...` provide reusable Go-side implementations for common host concerns. They are helpers, not a JavaScript auth framework: Express still only records route intent, and the host remains responsible for identity, sessions, resources, authorization, CSRF, and audit.

| Package | Use it for | Production note |
| --- | --- | --- |
| `devauth` | Local/demo users, login/logout/session endpoints, in-memory resources, authorization, CSRF, and audit. | Demo-only; use it for examples and tests, not production auth. |
| `sessionauth` | Opaque server-side app session cookies and CSRF token verification. | Replace `MemoryStore` with durable storage for multi-instance deployments. |
| `keycloakauth` | Keycloak/OIDC Authorization Code + PKCE login, callback, and logout handlers. | Keep IdP tokens server-side; planned routes use the app session cookie. |
| `appauth` | Small app-owned users, tenants, memberships, resources, and deny-by-default authorization helpers. | A starter shape; replace with domain services or a policy engine when needed. |
| `audit` | Normalize route audit events into redacted records for logs or stores. | Persist records if they matter for compliance or incident review. |
| `capability` | Hashed, expiring bearer-token flows such as org invites, email verification, password reset, or one-time downloads. | Store only hashes and make single-use redemption atomic. |

For development, start with `devauth` or the `examples/xgoja/16-express-auth-host` smoke. For production-shaped browser login, combine `keycloakauth`, `sessionauth`, `appauth`, and `audit` as shown in `examples/xgoja/17-express-keycloak-auth-host`.

## Error behavior

Planned routes fail closed. Missing services are host configuration errors; missing or invalid credentials are request errors.

| Condition | HTTP status | Reason |
| --- | ---: | --- |
| Public route | 200 if handler succeeds | No auth services required. |
| Auth route with no authenticator | 500 | Host is misconfigured. |
| Authenticator returns `ErrUnauthenticated` or no actor | 401 | Request has no usable actor. |
| Authorizer denies | 403 | Actor exists but lacks permission. |
| Resource resolver returns `ErrNotFound` | 404 | Resource was not found or should not be disclosed. |
| JavaScript handler throws after auth succeeds | 500 | Handler failed after the security envelope passed. |
| Raw route matched while `RejectRawRoutes` is true | 500 | Host rejected an unplanned route before handler execution. |
| `.csrf()` verifier fails | 403 | CSRF failure blocks handler execution. |
| `.audit(...)` sink fails | request status unchanged | Audit is best-effort in the MVP. |

In development mode, 500-class errors include more detail. In production mode, responses stay generic.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `app.get(pattern, handler) was removed` | The script still uses the old raw handler overload. | Change public routes to `app.get(pattern).public().handle(handler)` and protected routes to an auth-aware chain. |
| `.handle is not a function` | The route has not reached the handler stage. | Add `.public()` or `.auth(...).allow(...)` before `.handle(...)`. |
| `.auth(...) expects value returned by express.user()` | A plain JavaScript object was passed to `.auth(...)`. | Use `express.user().required()` so Go can validate the auth spec identity. |
| `.resource(...) expects value returned by express.resource(type)` | A plain JavaScript object was passed to `.resource(...)`. | Use `express.resource("project").idFromParam("projectId")`. |
| `references missing route parameter` | A resource builder references a param that is not in the path. | Match the parameter name exactly, for example `/projects/:projectId` with `.idFromParam("projectId")`. |
| Authenticated route returns 500 | The host is missing `Authenticator`, `Authorizer`, or required resource services. | Configure `gojahttp.HostOptions.Auth` in the embedding Go application. |
| Raw route returns 500 with `raw routes disabled` | The host enabled `RejectRawRoutes` and matched a route registered through low-level `Host.Register`. | Register the route through planned Express builders or `Host.RegisterPlanned`. |
| `.csrf()` route returns 500 | The route uses `.csrf()` on an unsafe method but the host has no `Auth.CSRF` service. | Configure `CSRFProtector` in `gojahttp.HostOptions.Auth`. |
| Audit events are missing | The route does not call `.audit(event)` or the host has no `Auth.Audit` sink. | Add `.audit("domain.event")` and configure an `AuditSink`. |
| Handler cannot find query or session fields | Planned handlers receive `ctx`, not raw `req`. | Use `ctx.request.query` or `ctx.request.session`. |

## See Also

- [Express-style HTTP Module](express-module) — The full module reference, including static mounts and response helpers.
- [Migrate Express Apps to Planned Auth Routes](migrate-express-apps-to-planned-auth) — Step-by-step migration tutorial for old `app.get(path, handler)` scripts.
- [Express Auth Examples](express-auth-examples) — Dev-auth and Keycloak smoke-test guide for full host wiring.
- `examples/xgoja/15-express-planned-auth/scripts/server.js` — Compact route-authoring example for public, current-user, and resource-bound routes.
