---
Title: Express-style middleware auth design and implementation guide
Ticket: XGOJA-EXPRESS-AUTH
Status: active
Topics:
    - goja
    - http
    - security
    - xgoja
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: modules/express/express.go
      Note: App object and route registration surface that would gain app.use
    - Path: modules/express/express_integration_test.go
      Note: Compatibility test suite that must remain green while middleware/router behavior is added
    - Path: modules/express/typescript.go
      Note: TypeScript declarations that would gain Middleware
    - Path: pkg/doc/18-express-module.md
      Note: Current docs explicitly identify middleware stacks and routers as unsupported; this design changes that boundary
    - Path: pkg/gojahttp/host.go
      Note: Current single-handler dispatch and promise handling to refactor into a middleware executor
    - Path: pkg/gojahttp/request_response.go
      Note: Request and response DTOs reused and extended by middleware execution
    - Path: pkg/gojahttp/route_registry.go
      Note: Route-only registry and matcher to evolve into router/layer stack support
    - Path: pkg/gojahttp/session.go
      Note: Opaque session cookie layer used by auth middleware as a lookup key
    - Path: pkg/xgoja/providers/http/http.go
      Note: Provider host setup and external host service injection relevant to auth service configuration
    - Path: pkg/xgoja/providers/http/serve.go
      Note: Serve and hot reload lifecycle where strict route validation should run after JS route bootstrap
    - Path: ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/design/01-mvp-authentication-api-design-and-implementation-guide.md
      Note: Companion staged RoutePlan design used for tradeoff comparison
    - Path: ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/sources/01-auth-preliminary-api-ideas.md
      Note: Original auth API idea source reconciled into middleware-compatible security helpers
ExternalSources:
    - ../sources/01-auth-preliminary-api-ideas.md
Summary: Alternative design that adds Express-style middleware stacks and routers while preserving Go-owned security middleware and route coverage validation.
LastUpdated: 2026-06-12T15:05:00-04:00
WhatFor: Use this when evaluating an Express-compatible direction for authentication in the go-go-goja HTTP module.
WhenToUse: Read before implementing app.use, Router, next, error middleware, or middleware-based auth/authorization for modules/express and pkg/gojahttp.
---


# Express-style middleware auth design and implementation guide

## Executive summary

This document describes an alternative to the staged route-plan API in `01-mvp-authentication-api-design-and-implementation-guide.md`. Instead of making authentication declarations part of a new fluent route builder, this approach moves the module closer to familiar Express by adding middleware stacks, `next()`, error middleware, and possibly `Router()` instances. Authentication then becomes a set of Go-owned middleware factories such as `express.auth.required()`, `express.loadResource(...)`, `express.allow(...)`, `express.validateBody(...)`, `express.csrf()`, and `express.audit(...)`.

The important compromise is that we should not blindly clone Node Express. The current module is embedded in Goja, runs through a Go-owned `gojahttp.Host`, and already has a deliberately small request/response model. A useful MVP should therefore be **Express-compatible in control flow**, but **Go-owned for security-critical middleware**. Custom JavaScript middleware should be allowed for logging, request shaping, and application glue. Authentication, authorization, CSRF, resource loading, body validation, and audit should be built-in middleware with machine-readable metadata so the Go host can validate route coverage and fail closed when production policy requires it.

The target developer experience looks like this:

```js
const express = require("express")
const app = express.app()
const api = express.Router()

app.use(express.requestId())
app.use(express.auditContext())
app.use("/api", api)

api.use(express.auth.required())
api.use(express.csrf())

api.patch(
  "/orgs/:orgId/projects/:projectId",
  express.loadResource("project", {
    type: "project",
    id: "param:projectId",
    tenant: "param:orgId",
    mustExist: true,
  }),
  express.allow("project.update", { resource: "project" }),
  express.validateBody("project.patch"),
  express.audit("project.updated"),
  function (req, res) {
    return req.services.projects
      .byResource(req.resource("project"))
      .patch(req.body)
      .commit()
  }
)
```

This API is easier for Express users to understand than a new staged builder. The tradeoff is that middleware order is now part of the security model. To avoid fail-open routes, Go-owned security middleware must carry metadata and the host must provide a strict coverage validator that can reject routes not covered by required auth/allow/resource middleware.

## Problem statement and scope

### Why revisit the design from an Express direction?

The first design document optimized for security declarations by making every secure route compile a `RoutePlan`. That gives excellent auditability, but it is not how Express users normally build applications. Express applications typically compose behavior through ordered middleware:

```js
app.use(logger)
app.use(session)
app.use(auth)
app.use("/api", apiRouter)
router.get("/me", handler)
router.patch("/projects/:id", authorize("project.update"), handler)
```

If the goal is to make go-go-goja's `express` module feel more familiar, the module needs at least a small middleware stack and router model. The existing user-facing documentation explicitly says the current module is **not full Express-compatible** and lacks middleware stacks, routers, and `next()` (`pkg/doc/18-express-module.md:18`). This design asks what it would take to add those features without giving up the main security lesson from the imported source document: user scripts should describe intent, while Go owns enforcement.

### MVP scope

The MVP should add:

- `app.use(...)` with optional path prefix.
- `express.Router()` with `.use(...)`, HTTP verb methods, and router mounting.
- Middleware functions with signature `(req, res, next)`.
- Error middleware functions with signature `(err, req, res, next)`.
- Route-local middleware arrays and multiple handlers per route.
- A shared pipeline executor in `pkg/gojahttp`.
- Go-owned security middleware factories exposed through `require("express")`.
- Metadata tags on Go-owned security middleware so route coverage can be validated.
- A strict host option that rejects unsafe or uncovered routes in production.

The MVP should not try to match every Express edge case:

- No npm Express plugin compatibility.
- No template engine system.
- No `app.param(...)` in the first patch unless resource loading needs it; route-local `loadResource(...)` is clearer.
- No `next("route")` in the first patch unless explicitly needed.
- No path-to-regexp compatibility beyond the current exact, `:param`, and `*` pattern support.
- No mutable prototype-compatible Node request/response objects.
- No streaming request body API.

## Current-state architecture recap

### Express module today

The current `express` module exports only `app()` from its loader (`modules/express/express.go:100-103`). The app object is created by `appObject`. It loops over `get`, `post`, `put`, `patch`, `delete`, and `all`; each method accepts exactly one handler, asserts the handler is callable, starts the host, and calls `r.host.Register(strings.ToUpper(method), pattern, fn)` (`modules/express/express.go:132-146`). Static helpers are separate methods on the same app object (`modules/express/express.go:148-187`).

The TypeScript declaration matches that minimal shape: `App` has direct HTTP verb methods and static helpers, and `Handler` is `(req: Request, res: Response) => unknown` (`modules/express/typescript.go:21-32`). There is no representation of `Middleware`, `NextFunction`, `Router`, or `ErrorMiddleware` today.

### gojahttp today

`gojahttp.Host.ServeHTTP` currently follows this order:

```text
static mounts
  -> runtime owner check
  -> registry match
  -> session creation or reuse
  -> request DTO construction
  -> response wrapper construction
  -> call one JavaScript handler
  -> await promise if returned
  -> finish returned value or send error
```

The exact route dispatch is in `pkg/gojahttp/host.go:94-152`. Route storage is deliberately small: `Route` has only method, pattern, and one `goja.Callable` (`pkg/gojahttp/route_registry.go:10-14`). `Registry.Add` appends a route to a slice (`pkg/gojahttp/route_registry.go:28-32`), and `Registry.Match` returns the first route whose method and path match (`pkg/gojahttp/route_registry.go:47-61`). Pattern matching supports cleaned paths, exact segments, `:params`, and `*` (`pkg/gojahttp/route_registry.go:64-112`).

The request object is a plain DTO. It exposes method, URL, path, query, params, headers, cookies, session, IP, parsed body, and raw body (`pkg/gojahttp/request_response.go:16-44`). The response object already supports the methods Express users expect for a small subset: `status`, `set`, `type`, `json`, `send`, `html`, `redirect`, and `end` (`pkg/gojahttp/request_response.go:89-104`).

The session layer is an opaque cookie ID, not authentication. The comment in `pkg/gojahttp/session.go` says application state remains in the application database (`pkg/gojahttp/session.go:16-18`). `SessionManager.Session` reuses a syntactically valid cookie or creates a new random ID and sets an HttpOnly cookie (`pkg/gojahttp/session.go:63-84`).

### xgoja provider and hot reload constraints

The xgoja HTTP provider creates or reuses a per-runtime `gojahttp.Host` during module setup. If an embedding application contributes an external host service, the provider uses that host; otherwise it creates a new one (`pkg/xgoja/providers/http/http.go:124-149`). Hot reload creates candidate hosts, loads JavaScript routes into them, smoke-tests them, and swaps them live (`pkg/xgoja/providers/http/serve.go:151-179`).

That means middleware stacks, routers, security metadata, and route coverage validation must be stored on the host/router for the current runtime. They must not be process-global.

## Design goals

### Developer goals

- Express users should recognize the control-flow model.
- Simple examples should stay simple.
- Existing `app.get("/path", handler)` code should still work.
- Middleware should support both app-wide and route-local use.
- Routers should make API grouping natural.

### Security goals

- Authentication should be easy to apply globally with `app.use` or `router.use`.
- Authorization should be route-local and action-specific.
- Resource loading should happen before authorization.
- Security-critical middleware should be Go-owned and host-configurable.
- Production hosts should be able to reject routes that are not covered by required security middleware.
- Middleware order should be visible in route diagnostics.

### Implementation goals

- Reuse the existing `RequestDTO`, `Response`, body parsing, session, static mount, route matching helpers, and runtime owner calls.
- Keep the first middleware engine small and testable.
- Avoid full Express path-to-regexp and Node stream semantics.
- Preserve promise handling behavior from existing route handlers.
- Make a future staged route-plan API and middleware API share the same auth service interfaces.

## Proposed JavaScript API

### App and router creation

```ts
export function app(): App
export function Router(options?: RouterOptions): Router

interface RouterOptions {
  mergeParams?: boolean
}
```

`express.app()` returns the root app object as it does today. `express.Router()` returns a router object with the same registration surface as an app except for `listen` and static mount helpers.

### Middleware registration

```ts
interface AppLike {
  use(...handlers: MiddlewareLike[]): void
  use(path: string, ...handlers: MiddlewareLike[]): void

  get(path: string, ...handlers: HandlerLike[]): void
  post(path: string, ...handlers: HandlerLike[]): void
  put(path: string, ...handlers: HandlerLike[]): void
  patch(path: string, ...handlers: HandlerLike[]): void
  delete(path: string, ...handlers: HandlerLike[]): void
  all(path: string, ...handlers: HandlerLike[]): void
}

type Middleware = (req: Request, res: Response, next: NextFunction) => unknown
type Handler = (req: Request, res: Response, next?: NextFunction) => unknown
type ErrorMiddleware = (err: unknown, req: Request, res: Response, next: NextFunction) => unknown
type MiddlewareLike = Middleware | ErrorMiddleware | Router | GoOwnedMiddleware
type HandlerLike = Handler | Middleware | GoOwnedMiddleware

type NextFunction = (err?: unknown) => void
```

Compatibility rule: the existing two-argument `(req, res)` handler remains valid. A single-handler route keeps the current auto-finish behavior: if it returns a string, send it; if it returns another non-null value, render/send it through the existing response logic. For middleware stacks, auto-finish should only apply to the last route handler. Non-terminal middleware should call `next()` or send a response explicitly.

### Built-in security middleware factories

```ts
export namespace auth {
  function optional(options?: AuthOptions): GoOwnedMiddleware
  function required(options?: AuthOptions): GoOwnedMiddleware
  function bearer(options?: AuthOptions): GoOwnedMiddleware
  function session(options?: AuthOptions): GoOwnedMiddleware
}

export function loadResource(name: string, spec: ResourceSpec): GoOwnedMiddleware
export function allow(action: string, options?: AllowOptions): GoOwnedMiddleware
export function validateBody(schemaName: string): GoOwnedMiddleware
export function csrf(options?: CSRFOptions): GoOwnedMiddleware
export function audit(eventName: string, options?: AuditOptions): GoOwnedMiddleware
export function secure(spec: SecureSpec): GoOwnedMiddleware[]
```

`secure(spec)` is the bridge back to the previous design. It lets authors keep Express-style handler registration while declaring a compact route envelope:

```js
api.patch(
  "/projects/:projectId",
  ...express.secure({
    auth: "required",
    resource: {
      name: "project",
      type: "project",
      id: "param:projectId",
      tenant: "param:orgId",
      mustExist: true,
    },
    allow: "project.update",
    body: "project.patch",
    csrf: true,
    audit: "project.updated",
  }),
  handler
)
```

Internally, `secure(spec)` returns the same Go-owned middleware objects as the explicit sequence. That gives route authors two equivalent styles:

```js
// Explicit middleware style
api.patch("/projects/:projectId",
  express.auth.required(),
  express.loadResource("project", { type: "project", id: "param:projectId" }),
  express.allow("project.update", { resource: "project" }),
  express.validateBody("project.patch"),
  handler)

// Compact policy-envelope style
api.patch("/projects/:projectId",
  ...express.secure({ auth: "required", resource: { name: "project", type: "project", id: "param:projectId" }, allow: "project.update", body: "project.patch" }),
  handler)
```

### Request additions

The current request DTO already exposes basic fields. Middleware auth adds fields and methods to the JavaScript request object:

```ts
interface Request {
  method: string
  url: string
  originalUrl: string
  baseUrl: string
  path: string
  routePath: string
  query: Record<string, string | string[]>
  params: Record<string, string>
  headers: Record<string, string>
  cookies: Record<string, string>
  session: Session | null
  ip: string
  body: unknown
  rawBody: string

  actor?: Actor
  auth?: AuthState
  resources?: Record<string, ResourceRef>
  resource(name: string): ResourceRef | null
  services?: Record<string, unknown>
}
```

`actor`, `auth`, `resources`, and `resource(name)` are owned by Go-backed middleware. `services` is optional and should only be present when the host application explicitly injects safe operation builders.

### Router example

```js
const express = require("express")
const app = express.app()
const api = express.Router()
const admin = express.Router()

app.use(express.requestId())
app.use(express.auditContext())
app.use("/api", api)

api.use(express.auth.required())
api.get("/me", express.allow("user.self.read"), (req) => ({ actor: req.actor }))

admin.use(express.auth.required({ mfaFresh: "10m" }))
admin.use(express.allow("admin.access"))
admin.get("/users", (req, res) => req.services.users.list())

api.use("/admin", admin)
```

The effective middleware chain for `GET /api/admin/users` is:

```text
app requestId
app auditContext
mount /api
  api auth.required
  mount /admin
    admin auth.required with MFA freshness
    admin allow admin.access
    admin route GET /users handler
```

## Runtime model

### Layer stack

Replace the current route-only registry with a router/layer stack. Existing `Host.Register` can remain as a compatibility wrapper that adds a route layer with one handler.

```go
type LayerKind string

const (
    LayerMiddleware LayerKind = "middleware"
    LayerRoute      LayerKind = "route"
    LayerMount      LayerKind = "mount"
)

type Router struct {
    mu     sync.RWMutex
    layers []Layer
}

type Layer struct {
    Kind        LayerKind
    Method      string        // only for route layers
    Pattern     string        // route path or middleware prefix
    Handlers    []HandlerSpec // middleware or route handlers
    Router      *Router       // for mount layers
    Source      string        // optional debug label
    Security    SecurityTags  // metadata summarized from Go-owned handlers
}

type HandlerSpec struct {
    Callable goja.Callable
    Kind     HandlerKind
    Arity    int
    Native   NativeMiddleware // non-nil for Go-owned middleware
    Tags     SecurityTags
    Source   string
}
```

`SecurityTags` is the key addition that keeps middleware-style auth auditable:

```go
type SecurityTags struct {
    LoadsActor      bool
    RequiresActor   bool
    LoadsResources  []string
    AllowsActions   []ActionBinding
    ValidatesBodies []string
    RequiresCSRF    bool
    AuditEvents     []string
    Public          bool
}
```

Custom JavaScript middleware has empty tags. Go-owned security middleware has explicit tags. The coverage validator can walk the layer stack and compute the effective tags for every route.

### Request dispatch pipeline

The host dispatch becomes:

```text
HTTP request
  -> static mounts, same as today
  -> session creation or reuse, same as today
  -> build mutable JS request object from RequestDTO
  -> build response object, same as today
  -> execute root router stack
       -> app middleware
       -> mounted router middleware
       -> route-local middleware
       -> terminal route handler
  -> if no route matched, 404
  -> if error is unhandled, error fallback
  -> if response is not sent and terminal handler returned a value, finish result
```

Plain-text control-flow diagram:

```text
+----------------+
| net/http req   |
+-------+--------+
        |
        v
+-----------------------+
| gojahttp.Host         |
| static mounts first   |
+-------+---------------+
        |
        v
+-----------------------+
| SessionManager        |
| NewRequestDTO         |
| NewResponse           |
+-------+---------------+
        |
        v
+-----------------------+
| Router executor       |
| layers in order       |
+---+---------------+---+
    |               |
    v               v
 middleware      route handler
 next continues  sends or returns
```

### Matching rules

MVP matching should extend current matching rather than replacing it with path-to-regexp:

- Route layers match method and full relative path.
- Middleware layers match path prefix.
- Mounted routers match path prefix and execute against the remaining path.
- `:params` work in routes and router mount paths.
- Wildcard `*` works like the current matcher.
- Registration order matters.
- Static mounts stay before dynamic router execution, preserving current behavior in `Host.ServeHTTP` (`pkg/gojahttp/host.go:94-103`).

Router mount example:

```js
app.use("/api/:orgId", api)
api.get("/projects/:projectId", handler)
```

Request:

```text
GET /api/o1/projects/p9
```

Computed request fields:

```js
req.originalUrl === "/api/o1/projects/p9"
req.baseUrl === "/api/o1"
req.path === "/projects/p9"        // while inside router
req.routePath === "/api/:orgId/projects/:projectId"
req.params.orgId === "o1"
req.params.projectId === "p9"
```

The implementation can keep a Go-side execution frame with current path/baseUrl/params and update the JS request object before each handler call.

## Middleware execution semantics

### Normal middleware

A middleware function can do one of three things:

1. Call `next()` to continue.
2. Call `next(err)` to enter error mode.
3. Send a response and not call `next()`.

```js
app.use(function logger(req, res, next) {
  console.log(req.method + " " + req.url)
  next()
})
```

### Route handler

A route handler may send explicitly or return a value. To preserve current module ergonomics, the terminal route handler keeps return-value auto-send:

```js
app.get("/hello", function () {
  return { message: "hello" }
})
```

In a multi-handler route, only the final handler should auto-send its return value. Earlier route-local middleware should call `next()`.

```js
app.get(
  "/me",
  express.auth.required(),
  express.allow("user.self.read"),
  function (req) {
    return { id: req.actor.id }
  }
)
```

### Error middleware

Error middleware has arity four and runs only when an error exists:

```js
app.use(function errorHandler(err, req, res, next) {
  console.error(err)
  res.status(500).json({ error: "internal" })
})
```

Implementation detail: when registering a JavaScript function, read its `length` property inside Goja and store it in `HandlerSpec.Arity`. Arity 4 is error middleware. Arity 0-3 is normal middleware/handler.

### Promise handling

The current host already detects returned promises from route handlers and waits for fulfillment or rejection (`pkg/gojahttp/host.go:135-143`, `pkg/gojahttp/host.go:167-190`). The middleware executor should reuse that logic for every middleware call.

Supported async style:

```js
app.use(function asyncAuth(req, res, next) {
  return req.services.auth.load(req.session.id)
    .then(actor => { req.actor = actor; next() })
    .catch(next)
})
```

Preferred Go-owned security middleware should be native Go code and avoid JS promise complexity where possible.

### `next()` correctness

`next()` should be guarded:

- Calling `next()` twice should produce a clear dev-mode error.
- Calling `next()` after sending a response should be allowed only if an error middleware needs to observe it; otherwise log or ignore in dev mode.
- Throwing inside middleware is equivalent to `next(err)`.
- A rejected promise is equivalent to `next(err)`.

MVP simplification: if a middleware neither sends a response, returns a terminal value, nor calls `next()`, the pipeline stops. In dev mode, the host should report a helpful error such as `middleware did not call next() or send a response` if the request would otherwise hang. Because Go cannot hang a net/http request indefinitely, the executor should return a 500 in dev mode and generic 500 in production for that case.

## Go-owned security middleware

### Host auth service interfaces

Reuse the interface direction from the first design. These live in `pkg/gojahttp` because the host executes middleware and owns request dispatch.

```go
type AuthOptions struct {
    Authenticator Authenticator
    Resources     ResourceResolver
    Authorizer    Authorizer
    BodySchemas   BodyValidator
    Audit         AuditSink
    CSRF          CSRFProtector
    Strict        StrictSecurityOptions
}

type Authenticator interface {
    Authenticate(ctx context.Context, req *http.Request, session *SessionDTO, spec AuthSpec) (*Actor, error)
}

type ResourceResolver interface {
    ResolveResource(ctx context.Context, req ResourceRequest) (*ResourceRef, error)
}

type Authorizer interface {
    Authorize(ctx context.Context, req AuthorizationRequest) (AuthorizationDecision, error)
}
```

The security middleware factories in `modules/express` should create `HandlerSpec` values that call these services through the host. They should not ask JavaScript to perform raw auth checks.

### `express.auth.required()`

Behavior:

- Calls `AuthOptions.Authenticator.Authenticate`.
- On success, sets `req.actor` and `req.auth`.
- On missing/invalid credentials, sends 401 and stops.
- On host misconfiguration, sends 500 and stops.
- Tags the layer with `RequiresActor=true` and `LoadsActor=true`.

Pseudocode:

```go
func requiredAuthMiddleware(spec AuthSpec) NativeMiddleware {
    return NativeMiddleware{
        Tags: SecurityTags{LoadsActor: true, RequiresActor: true},
        Call: func(ctx *MiddlewareContext) error {
            if ctx.Host.auth.Authenticator == nil {
                return ctx.Fail(http.StatusInternalServerError, "authenticator not configured")
            }
            actor, err := ctx.Host.auth.Authenticator.Authenticate(ctx.Context, ctx.HTTPRequest, ctx.Session, spec)
            if errors.Is(err, ErrUnauthenticated) {
                return ctx.Fail(http.StatusUnauthorized, "unauthenticated")
            }
            if err != nil { return err }
            ctx.Request.Actor = actor
            ctx.JSRequest.Set("actor", actor)
            return ctx.Next(nil)
        },
    }
}
```

### `express.loadResource(name, spec)`

Behavior:

- Resolves resource IDs from params, query, body, actor, or literals.
- Calls `AuthOptions.Resources.ResolveResource`.
- Stores the returned `ResourceRef` in `req.resources[name]`.
- Provides `req.resource(name)` convenience lookup.
- Tags the layer with `LoadsResources=[name]`.

Example:

```js
express.loadResource("project", {
  type: "project",
  id: "param:projectId",
  tenant: "param:orgId",
  mustExist: true,
})
```

### `express.allow(action, options)`

Behavior:

- Requires `req.actor` unless the action is explicitly public/system in options.
- Loads a named resource from `req.resources` when `options.resource` is set.
- Calls `AuthOptions.Authorizer.Authorize`.
- Sends 403 on denial.
- Tags the layer with `AllowsActions=[{ action, resourceName }]`.

Example:

```js
api.patch("/projects/:projectId",
  express.loadResource("project", { type: "project", id: "param:projectId" }),
  express.allow("project.update", { resource: "project" }),
  handler)
```

### `express.validateBody(schemaName)`

Behavior:

- Calls `AuthOptions.BodySchemas.ValidateBody` with the already parsed `req.body`.
- Replaces `req.body` with normalized/validated output when the validator returns one.
- Sends 400 or 422 on validation failure; pick one consistently in implementation.
- Tags the layer with `ValidatesBodies=[schemaName]`.

### `express.csrf()`

Behavior:

- Runs only for unsafe methods unless configured otherwise.
- Calls `AuthOptions.CSRF.Check`.
- Sends 403 on failure.
- Tags the layer with `RequiresCSRF=true`.

The current session cookie has SameSite=Lax by default (`pkg/gojahttp/session.go:57-58`, `pkg/gojahttp/session.go:74-82`), but SameSite is not a complete substitute for explicit CSRF checks on unsafe browser requests.

### `express.audit(eventName)`

Behavior:

- Adds audit metadata before handler execution.
- Emits success/failure outcome after downstream middleware completes.
- Should not let JavaScript write arbitrary security audit records; JavaScript declares the event name, Go emits the structured event.
- Tags the layer with `AuditEvents=[eventName]`.

## Route coverage validation

Middleware-based auth is easy to read but easy to forget. The previous RoutePlan design solved that by making security declarations mandatory before `.handle(...)`. The middleware design should recover most of that safety with route coverage validation.

### Coverage model

For every route layer, the host computes the effective middleware tags that can run before the route:

```text
root app middleware tags
+ matching mount middleware tags
+ router middleware tags
+ route-local middleware tags before terminal handler
= effective route security coverage
```

Example route:

```js
api.use(express.auth.required())
api.patch("/projects/:projectId",
  express.loadResource("project", { type: "project", id: "param:projectId" }),
  express.allow("project.update", { resource: "project" }),
  handler)
```

Effective tags:

```yaml
RequiresActor: true
LoadsActor: true
LoadsResources:
  - project
AllowsActions:
  - action: project.update
    resource: project
```

### Strict security options

```go
type StrictSecurityOptions struct {
    Enabled                    bool
    UnsafeMethodsRequireAuth    bool
    UnsafeMethodsRequireAction  bool
    ActionsRequireResource      bool
    RequireCSRFForSessionAuth   bool
    AllowRawRoutes              bool
    PublicRouteNames            []string
}
```

Suggested defaults:

- Development/demo hosts: strict disabled, but diagnostics available.
- Production hosts: strict enabled by embedding application.
- Generated xgoja default: strict disabled initially to avoid breaking examples, but expose a future `--http-strict-auth` flag.

Validation examples:

```text
POST /api/users has no auth.required and no public tag
  -> reject: unsafe route is not covered by required auth middleware

PATCH /api/projects/:projectId has auth.required but no allow action
  -> reject: unsafe authenticated route is not covered by allow(action)

PATCH /api/projects/:projectId has allow project.update but no loadResource project
  -> reject: action project.update references missing resource project
```

### When to validate

There are three possible validation points:

1. **At registration time.** Good error locality, but app-wide middleware can be registered before or after routes in Express style, and mounted router context may not be known yet.
2. **At `app.listen()` or after route bootstrap.** Best for complete-stack validation, but current provider starts the server on first route/static registration through `WithOnUse` (`modules/express/express.go:141-144`, `pkg/xgoja/providers/http/http.go:146-148`).
3. **Before first request.** Works with current lifecycle, but moves errors later.

MVP recommendation: implement `Host.ValidateRoutes()` and call it from two places:

- `app.validate()` or `app.listen()` for scripts that opt in explicitly.
- The xgoja `serve` command after it invokes the jsverb and before waiting for requests.

As a safe fallback, strict hosts should also validate once before serving the first dynamic request.

## Data structures and implementation plan

### Phase 1: Introduce router and layer types

Files:

- `pkg/gojahttp/route_registry.go`
- new `pkg/gojahttp/router.go`
- `pkg/gojahttp/route_registry_test.go`

Implementation sketch:

```go
type Registry struct {
    root *Router
}

func NewRegistry() *Registry {
    return &Registry{root: NewRouter()}
}

func (r *Registry) Add(method, pattern string, handler goja.Callable) {
    r.root.Handle(method, pattern, HandlerSpec{Callable: handler, Kind: HandlerRoute})
}

func (r *Registry) Use(prefix string, handlers ...HandlerSpec) {
    r.root.Use(prefix, handlers...)
}

func (r *Registry) Mount(prefix string, router *Router) {
    r.root.Mount(prefix, router)
}
```

Keep `Registry.Match` available for old tests initially, but have it delegate to a route-only lookup over the root router. Then add a new executor path for middleware. This reduces migration risk.

### Phase 2: Add middleware-aware app and router objects

Files:

- `modules/express/express.go`
- new `modules/express/router.go`
- `modules/express/typescript.go`
- `modules/express/express_integration_test.go`

Add exports:

```go
_ = exports.Set("app", func() goja.Value { return r.appObject(vm) })
_ = exports.Set("Router", func(call goja.FunctionCall) goja.Value { return r.routerObject(vm, routerOptions(call)) })
_ = exports.Set("auth", authNamespace(vm, r.host))
_ = exports.Set("loadResource", loadResourceFactory(vm, r.host))
_ = exports.Set("allow", allowFactory(vm, r.host))
_ = exports.Set("validateBody", validateBodyFactory(vm, r.host))
_ = exports.Set("csrf", csrfFactory(vm, r.host))
_ = exports.Set("audit", auditFactory(vm, r.host))
_ = exports.Set("secure", secureFactory(vm, r.host))
```

Change HTTP verb registration to accept variadic handlers:

```go
_ = obj.Set(method, func(call goja.FunctionCall) goja.Value {
    pattern := call.Argument(0).String()
    handlers, err := parseHandlers(vm, call.Arguments[1:])
    if err != nil { panic(vm.NewGoError(err)) }
    if err := r.start(vm); err != nil { panic(vm.NewGoError(err)) }
    router.Handle(strings.ToUpper(method), pattern, handlers...)
    return goja.Undefined()
})
```

`parseHandlers` should accept:

- Goja functions.
- Arrays of functions or Go-owned middleware.
- Router objects for `use` mounts.
- Go-owned middleware objects returned by factory functions.

### Phase 3: Build pipeline executor

Files:

- `pkg/gojahttp/host.go`
- new `pkg/gojahttp/middleware_executor.go`

Executor pseudocode:

```go
func (h *Host) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if h.serveStatic(w, r) { return }
    if h.owner == nil { http.Error(...); return }

    session, err := h.sessions.Session(w, r)
    if err != nil { ... }

    reqDTO, err := NewRequestDTO(r, nil, session)
    if err != nil { ... }

    res := NewResponse(w, h.renderer)
    exec := newExecutor(h, r, reqDTO, res)
    matched, err := exec.Run(h.registry.Root())
    if err != nil && !res.Sent() { h.writeError(w, err) }
    if !matched && !res.Sent() { http.NotFound(w, r) }
}
```

Inside the executor, every JavaScript call should happen through `h.owner.Call` to preserve runtime ownership:

```go
func (e *Executor) callJS(handler HandlerSpec, args ...goja.Value) (goja.Value, error) {
    ret, err := e.host.owner.Call(e.httpReq.Context(), "http-middleware", func(ctx context.Context, vm *goja.Runtime) (any, error) {
        values := buildArgs(vm, args...)
        result, err := handler.Callable(goja.Undefined(), values...)
        if err != nil { return nil, err }
        return result, nil
    })
    if err != nil { return nil, err }
    return ret.(goja.Value), nil
}
```

Be careful: `goja.Value` belongs to a runtime. If values are returned across owner boundaries as `any`, the existing host already returns `*goja.Promise` and later re-enters the runtime to inspect it. Follow that pattern rather than exporting arbitrary JS objects across goroutines.

### Phase 4: Add Go-owned middleware factories

Files:

- new `pkg/gojahttp/auth_middleware.go`
- new `modules/express/auth_middleware.go`

A Go-owned middleware object can be represented as an opaque Goja object with a hidden pointer to `HandlerSpec`:

```go
type nativeMiddlewareObject struct {
    spec gojahttp.HandlerSpec
}
```

`parseHandlers` can recognize these objects by private symbol/property or Go export type. Keep the JS surface clean; users should not rely on internals.

### Phase 5: Coverage validation and diagnostics

Files:

- new `pkg/gojahttp/security_validate.go`
- `pkg/gojahttp/host.go`
- `pkg/xgoja/providers/http/serve.go`

Add:

```go
func (h *Host) ValidateRoutes() (*RouteValidationReport, error)
func (h *Host) Routes() []RouteDescriptor
func (h *Host) MiddlewareGraph() []LayerDescriptor
```

Extend `RouteDescriptor` to include route security summary:

```go
type RouteDescriptor struct {
    Method       string          `json:"method"`
    Pattern      string          `json:"pattern"`
    MountPath    string          `json:"mountPath,omitempty"`
    Middlewares  []string        `json:"middlewares,omitempty"`
    Security     SecuritySummary `json:"security,omitempty"`
}
```

Do not remove the existing fields, because current tests expect method and pattern.

## Testing strategy

### Existing behavior must keep passing

Existing tests in `modules/express/express_integration_test.go` cover:

- HTML node return rendering.
- Static assets.
- SPA fallback and `/api` exclusion.
- POST JSON echo.
- Promise-returning route handlers.
- Promise route handlers that send responses.
- HEAD fallback to GET.

All of these should keep passing. The single-handler route path must remain compatible.

### New middleware tests

Add tests for:

1. `app.use` middleware runs before route handler.
2. Middleware can mutate `req` and route handler sees the mutation.
3. Middleware can stop the pipeline by sending a response.
4. Middleware can call `next(err)` and error middleware receives it.
5. Thrown JS error enters error middleware.
6. Rejected promise enters error middleware.
7. Terminal route handler still auto-sends returned object.
8. Non-terminal middleware must call `next()` or send.
9. Multiple route-local handlers run in order.
10. `app.use("/prefix", middleware)` applies only under the prefix.

### Router tests

Add tests for:

1. `express.Router()` can be mounted with `app.use("/api", router)`.
2. Router-level middleware runs before router routes.
3. Nested routers preserve mount order.
4. Mount params merge with route params.
5. `req.baseUrl`, `req.originalUrl`, `req.path`, and `req.params` are correct inside nested routers.

### Security middleware tests

Use fake host auth services:

1. `express.auth.required()` with no authenticator fails closed.
2. Missing credentials returns 401.
3. Authenticator success sets `req.actor`.
4. `loadResource` calls resolver and sets `req.resource(name)`.
5. `allow` denies with 403 when authorizer denies.
6. `validateBody` replaces body with normalized output.
7. `csrf` rejects unsafe request.
8. `audit` records success and denial events.
9. `secure(spec)` expands into equivalent security middleware.
10. Strict coverage rejects unsafe route without auth/allow.

### xgoja tests

Add provider tests only after local host tests pass:

- Generated HTTP provider serves middleware-based public route.
- External host service can provide auth services for middleware auth route.
- Hot reload candidate host includes middleware stack and route descriptors.
- Smoke path works when route is behind public middleware.

## Migration and compatibility

### Existing code

This should continue to work:

```js
const express = require("express")
const app = express.app()
app.get("/hello/:name", (req, res) => "hello " + req.params.name)
```

Internally, it becomes a route layer with one terminal handler.

### New code

New applications can opt into middleware gradually:

```js
app.use(express.auth.optional())
app.get("/public", handler)
app.get("/me", express.auth.required(), express.allow("user.self.read"), handler)
```

### Docs

Update `pkg/doc/18-express-module.md` after implementation. The first paragraph currently says middleware stacks and routers are not supported (`pkg/doc/18-express-module.md:18`); that will need to change to describe the supported subset and explicit non-goals.

## Tradeoffs versus the staged RoutePlan design

### Advantages

- Familiar to Express users.
- Easier to port small Express examples.
- Global concerns like logging, request IDs, optional auth, and audit context are natural.
- Routers make grouping APIs ergonomic.
- Route-local security middleware still reads clearly.

### Disadvantages

- Middleware order becomes security-critical.
- A route can accidentally omit `allow(...)` unless strict coverage catches it.
- Static analysis is harder because protection may be inherited from parent routers.
- Error and async semantics are more complex than one-handler dispatch.
- It is tempting for users to write JS auth middleware that bypasses Go-owned security services.

### Recommendation

If the project values maximum security by construction, the staged RoutePlan API from the first design should be the default recommendation. If the project values Express familiarity and migration, this middleware design is viable **only if** Go-owned security middleware and strict coverage validation are part of the MVP or very near follow-up.

The strongest combined path is:

1. Implement middleware stacks and routers.
2. Implement Go-owned `express.secure(spec)` as the recommended auth helper.
3. Let `secure(spec)` produce middleware so users stay in Express style.
4. Add strict coverage validation so production hosts can reject uncovered routes.
5. Optionally implement the staged route builder later as a thin wrapper around the same middleware/security metadata.

## Decision records

### Decision: Add Express-style middleware as a stack of host-owned layers

- **Context:** The current registry stores only direct routes with one handler, while an Express-like design needs app middleware, route-local middleware, routers, and error middleware.
- **Options considered:** Keep route-only dispatch; add only route-local arrays; add full layer stack; embed a third-party router.
- **Decision:** Add a host-owned router/layer stack in `pkg/gojahttp` and keep `Host.Register` as a compatibility wrapper.
- **Rationale:** The host already owns dispatch and runtime calls, and a layer stack preserves registration order while keeping hot reload state per host.
- **Consequences:** `Host.ServeHTTP` becomes more complex and needs careful tests for next/error/promise behavior.
- **Status:** proposed

### Decision: Make security middleware Go-owned but Express-composable

- **Context:** Middleware is familiar, but raw JavaScript auth checks are easy to forget or implement incorrectly.
- **Options considered:** Let users write all auth middleware in JS; expose only declarative RoutePlan builder; expose Go-owned middleware factories; use external Go `net/http` middleware only.
- **Decision:** Expose Go-owned middleware factories through `express.auth`, `express.loadResource`, `express.allow`, `express.validateBody`, `express.csrf`, `express.audit`, and `express.secure`.
- **Rationale:** This keeps Express composition while preserving Go-owned security enforcement.
- **Consequences:** Middleware objects need metadata and parser support, and host applications must configure auth services.
- **Status:** proposed

### Decision: Add strict route coverage validation

- **Context:** Middleware-based security can fail open if a route forgets required middleware.
- **Options considered:** Trust docs; validate only route-local middleware; validate full effective middleware chain; require staged RoutePlan for auth routes.
- **Decision:** Provide full effective-chain validation through `Host.ValidateRoutes()` and strict host options.
- **Rationale:** This recovers much of the safety of the staged design while preserving Express style.
- **Consequences:** Registration/mount metadata must be inspectable, and xgoja serve should call validation after route bootstrap when strict mode is enabled.
- **Status:** proposed

### Decision: Preserve return-value auto-send for terminal route handlers only

- **Context:** Existing go-go-goja handlers can return values instead of calling `res.send`, while Express usually ignores return values.
- **Options considered:** Drop auto-send; keep auto-send for every middleware; keep auto-send only for terminal route handlers.
- **Decision:** Keep auto-send only for terminal route handlers.
- **Rationale:** This preserves existing examples without making middleware return values accidentally terminate requests.
- **Consequences:** Documentation must clearly distinguish middleware from terminal handlers.
- **Status:** proposed

## Implementation checklist for a new intern

Start with tests and small steps:

1. Read `pkg/doc/18-express-module.md` to understand the documented current subset.
2. Read `modules/express/express.go:132-187` to see app object construction and route/static registration.
3. Read `pkg/gojahttp/host.go:94-152` to understand current dispatch and promise handling.
4. Read `pkg/gojahttp/route_registry.go:10-112` to understand current matching.
5. Add router/layer data structures without changing behavior.
6. Make existing `Host.Register` add a route layer and keep old tests green.
7. Add `app.use` with simple synchronous middleware.
8. Add route-local multiple handlers.
9. Add error middleware.
10. Add promise support in middleware.
11. Add `express.Router()` and mounting.
12. Add Go-owned security middleware factories.
13. Add route coverage validation.
14. Update TypeScript declarations.
15. Update `pkg/doc/18-express-module.md`.
16. Run targeted tests:

```bash
go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http -count=1
```

## Open questions

1. Should the MVP implement `next("route")`, or is normal `next()` plus error middleware enough?
2. Should `app.use(express.auth.required())` protect static mounts, or should static mounts always stay before dynamic middleware as they do today?
3. Should strict coverage be off by default for generated xgoja binaries, or should generated HTTP apps fail closed unless routes declare `.public()` or auth middleware?
4. Should `express.secure(spec)` return an array of middleware or one middleware that internally runs a mini pipeline?
5. Should `req.services` be part of the Express module, or should operation builders live in application-specific modules only?
6. Should middleware mutate the plain request map directly, or should `RequestDTO` become a Go-backed JS object to support methods and controlled fields?

## File reference map

- `pkg/doc/18-express-module.md`: Current user-facing statement that middleware stacks and routers are not supported; must be updated after this design is implemented.
- `modules/express/express.go`: Current loader and app object; add `Router`, `use`, variadic route handlers, and security middleware factories here or in sibling files.
- `modules/express/typescript.go`: Current TypeScript declarations; add `Middleware`, `NextFunction`, `Router`, and security middleware declarations.
- `modules/express/express_integration_test.go`: Existing compatibility tests that must continue passing.
- `pkg/gojahttp/host.go`: Current dispatch and promise handling; refactor into middleware executor while preserving static-first behavior.
- `pkg/gojahttp/route_registry.go`: Current route-only registry and path matching; evolve into router/layer stack or wrap with a new router implementation.
- `pkg/gojahttp/request_response.go`: Request/response DTOs reused by middleware; add request methods and auth/resource fields.
- `pkg/gojahttp/session.go`: Existing opaque cookie session; security middleware should treat it as a lookup key, not an authenticated identity by itself.
- `pkg/xgoja/providers/http/http.go`: Provider host creation and external host service injection; auth services can be supplied by embedding applications through host options.
- `pkg/xgoja/providers/http/serve.go`: jsverb-backed serve and hot reload lifecycle; strict route validation should run after route bootstrap when enabled.
- `ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/design/01-mvp-authentication-api-design-and-implementation-guide.md`: Companion staged RoutePlan design; compare tradeoffs before implementation.
- `ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/sources/01-auth-preliminary-api-ideas.md`: Original preliminary API ideas that motivated Go-owned security enforcement.
