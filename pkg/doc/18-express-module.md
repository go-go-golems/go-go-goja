---
Title: Express-style HTTP Module
Slug: express-module
Short: Host Goja HTTP routes with an Express-style JavaScript API
Topics:
- http
- modules
- goja
- javascript
Commands:
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `express` module exposes a small Express-style route registration API for Goja-hosted applications. It is intentionally **Express-style**, not full Express-compatible: it supports Go-owned planned auth routes, Go-backed handler mounts, static mounts, and response helpers, but not middleware stacks, routers, `next()`, template engines, or npm Express plugins.

## Go setup

`express` is runtime-scoped because it needs a configured `gojahttp.Host` and runtime owner. Install it with `modules/express.NewRegistrar(host)` rather than through the default native module registry.

```go
host := gojahttp.NewHost(gojahttp.HostOptions{
    Dev:      true,
    Renderer: uidsl.RenderAny,
})

factory, err := engine.NewRuntimeFactoryBuilder().
    WithModules(
        express.NewRegistrar(host),
        uidsl.NewRegistrar(),
    ).
    Build()
```

`pkg/gojahttp` owns the reusable HTTP host, route matching, request/response DTOs, body parsing, sessions, static mounts, and dispatch into the Goja runtime. The host is renderer-neutral; `HostOptions.Renderer` decides how `res.html(value)` renders non-string values.

## JavaScript usage

```javascript
const express = require("express");
const ui = require("ui.dsl");

const app = express.app();

app.get("/hello/:name")
  .public()
  .handle((ctx, res) => {
    return ui.page(
      { title: "Hello" },
      ui.h1("Hello " + ctx.params.name)
    );
  });

app.post("/api/echo")
  .public()
  .handle((ctx, res) => {
    res.status(201).json({ body: ctx.body });
  });
```

Handlers may call `res.*` explicitly. If a handler returns a string, the host sends it as text/HTML depending on content. If a handler returns any other non-null value, the host calls `res.html(value)` and uses the configured renderer. Planned handlers receive `(ctx, res)`; the original request DTO is available as `ctx.request`.

## App API

```javascript
app.get(pattern)
  .public()
  .handle(handler)

app.post(pattern)
  .auth(express.user().required())
  .csrf()
  .allow(action)
  .audit(event)
  .handle(handler)

app.patch(pattern)
  .auth(express.user().required())
  .resource(express.resource(type).idFromParam(paramName))
  .allow(action)
  .audit(event)
  .handle(handler)

app.route(method, pattern)
  .public()
  .handle(handler)

app.mount(prefix, mountableHandler, options?)
app.mountHandler(prefix, mountableHandler, options?)
app.static(prefix, directory)
app.staticFromAssetsModule(prefix, assetsModule, root)
```

`app.get`, `app.post`, `app.put`, `app.patch`, `app.delete`, and `app.all` are planned-route builders. They intentionally do **not** support the old `app.get(pattern, handler)` overload. Existing scripts must migrate public endpoints to `app.get(pattern).public().handle(handler)` and protected endpoints to an auth-aware chain.

`app.mount(prefix, mountableHandler, options?)` mounts a Go `http.Handler` that another native module exposed as a mountable JavaScript object. This is useful for Go-owned transports such as WebSocket servers. The mounted handler uses prefix matching and preserves the original request path by default.

```javascript
const express = require("express");
const app = express.app();

// wsServer is a JavaScript object carrying a hidden Go http.Handler ref.
app.mount("/ws", wsServer);

// Use stripPrefix when the mounted handler expects paths relative to the mount.
app.mountHandler("/api", apiHandler, { stripPrefix: true });
```

Mount options:

```ts
type MountOptions = {
  stripPrefix?: boolean;
  excludePrefixes?: string[];
};
```

`app.static(prefix, directory)` serves a real host filesystem directory. Static helpers preserve their historical behavior and strip the mount prefix before the file server sees the request.

`app.staticFromAssetsModule(prefix, assetsModule, root)` serves files directly from a read-only embedded fs module, for example:

```javascript
const express = require("express");
const assets = require("fs:assets");
const app = express.app();

app.staticFromAssetsModule("/static", assets, "/app/public");
```

Route patterns support exact paths, `:params`, and `*` wildcards. Parameter segments capture one path segment into `req.params`; for example `/hello/:name` exposes `req.params.name`. A `*` segment matches the remainder of the path, but currently does not expose a captured splat value. Handler mounts use prefix matching rather than route-pattern matching.

### Go-side routing for mounted handlers

`app.mount()` is intentionally a prefix handoff to a Go `http.Handler`, not a second JavaScript route-pattern system for Go-owned transports. Use a stable mount prefix in JavaScript and let the mounted Go handler interpret the remaining path.

```javascript
const express = require("express");
const app = express.app();

// Matches /ws and /ws/... . The Go handler receives the original path by default.
app.mount("/ws", sessionstream.webSocket.server(hub));
```

The mounted handler can route by inspecting `r.URL.Path`:

```go
func websocketHandler(w http.ResponseWriter, r *http.Request) {
    switch {
    case r.URL.Path == "/ws":
        serveDefaultSocket(w, r)
    case strings.HasPrefix(r.URL.Path, "/ws/rooms/"):
        roomID := strings.TrimPrefix(r.URL.Path, "/ws/rooms/")
        serveRoomSocket(w, r, roomID)
    default:
        http.NotFound(w, r)
    }
}
```

Or it can use its own Go router, including the Go standard library `http.ServeMux` path-value patterns:

```go
mux := http.NewServeMux()

mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
    serveDefaultSocket(w, r)
})

mux.HandleFunc("GET /ws/rooms/{roomID}", func(w http.ResponseWriter, r *http.Request) {
    roomID := r.PathValue("roomID")
    serveRoomSocket(w, r, roomID)
})

mux.HandleFunc("GET /ws/assets/{path...}", func(w http.ResponseWriter, r *http.Request) {
    assetPath := r.PathValue("path")
    serveAssetSocket(w, r, assetPath)
})
```

Use `{ stripPrefix: true }` only when the mounted Go handler expects paths relative to the mount point. Leave it unset for handlers that route on their full public path.

## Planned auth routes

All verb helpers now use planned routes. Planned routes use staged Go-backed builder objects: JavaScript gets a fluent API, but the security-critical route plan is compiled and validated by Go at registration time. `app.route(method, pattern)` remains available for dynamic or uncommon HTTP methods; prefer `app.get(pattern)`, `app.post(pattern)`, and the other verb helpers for normal routes.

A public planned route must explicitly call `.public()` before `.handle(...)`:

```javascript
const express = require("express");
const app = express.app();

app.get("/healthz")
  .public()
  .handle((_ctx, res) => res.json({ ok: true }));
```

An authenticated route declares its auth mode and permission action before the handler is registered:

```javascript
app.get("/me")
  .auth(express.user().required())
  .allow("user.self.read")
  .handle((ctx, res) => {
    res.json({ id: ctx.actor.id });
  });
```

A resource-bound route declares where the resource identity is extracted from the HTTP adapter layer. `idFromParam("projectId")` means “read the resource id from `:projectId`”, not “perform authorization in JavaScript”. The host's Go `ResourceResolver` and `Authorizer` still own resource loading and access control.

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
    const project = ctx.resource("project");
    res.json({ project: project.id, tenant: project.tenantId });
  });
```

The builder is intentionally strict:

- `.handle(...)` is not available until a route calls `.public()` or `.auth(...).allow(...)`.
- `.auth(...)` only accepts a Go-backed value returned by `express.user()`.
- `.resource(...)` only accepts a Go-backed value returned by `express.resource(type)`.
- `.csrf()` requires a host `CSRFProtector` on unsafe methods and rejects bad tokens before the handler runs.
- `.audit(event)` emits host-owned audit events for allowed, denied, completed, and failed planned requests.
- Referencing a missing path parameter, such as `.idFromParam("id")` on `/projects/:projectId`, fails at route registration time.

Planned handlers receive `(ctx, res)`:

```ts
type PlannedContext = {
  request: Request;
  actor: { id: string; kind: string; tenantIds?: string[]; claims?: Record<string, unknown> } | null;
  body: unknown;
  params: Record<string, string>;
  resources: Record<string, ResourceRef>;
  resource(name: string): ResourceRef | null;
  action: string;
  routeName: string;
};
```

Host applications configure planned auth through `gojahttp.HostOptions.Auth`:

```go
host := gojahttp.NewHost(gojahttp.HostOptions{
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

Missing auth services fail closed for authenticated planned routes. Missing CSRF services fail closed for unsafe routes that declare `.csrf()`. Missing credentials return 401, denied authorization and CSRF failures return 403, and resource lookup failures can return 404 via `gojahttp.ErrNotFound`. `RejectRawRoutes` rejects any matched low-level route that lacks a `RoutePlan`; enable it for production hosts that should only serve planned routes.

## Request object

Planned handlers can inspect the original request DTO through `ctx.request`:

```ts
type Request = {
  method: string;
  url: string;
  path: string;
  query: Record<string, string | string[]>;
  params: Record<string, string>;
  headers: Record<string, string>;
  cookies: Record<string, string>;
  session: { id: string; isNew: boolean; cookieName: string } | null;
  ip: string;
  body: unknown;
  rawBody: string;
};
```

JSON and form bodies are parsed automatically. Other request bodies are exposed as strings.

## Response object

```javascript
res.status(code)
res.set(name, value)
res.type(contentType)
res.json(value)
res.send(value)
res.html(value)
res.redirect(url)
res.redirect(status, url)
res.end()
```

`res.html(value)` requires a renderer in `gojahttp.HostOptions`. With `modules/uidsl.RenderAny`, route handlers can return or send `ui.dsl` nodes directly.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `app.get(pattern, handler) was removed` | The route uses the old raw handler overload. | Use `app.get(pattern).public().handle(handler)` or an auth-aware chain. |
| `.handle is not a function` | The route has not declared `.public()` or completed `.auth(...).allow(...)`. | Add the missing route-plan stage before `.handle(...)`. |
| Authenticated route returns 500 | The Go host is missing auth services. | Configure `gojahttp.HostOptions.Auth` with an authenticator and authorizer. |
| `.csrf()` route returns 500 | The route declares CSRF but the host has no `CSRFProtector`. | Configure `Auth.CSRF` or remove `.csrf()` from the route. |
| Handler cannot read `req.query` or `req.session` | Planned handlers receive `ctx`, not raw `req`. | Use `ctx.request.query` or `ctx.request.session`. |
| Raw route returns `raw routes disabled` | `HostOptions.RejectRawRoutes` is enabled and a low-level route without a plan matched. | Register the route through the planned Express API or `Host.RegisterPlanned`. |

## See Also

- [Express Auth User Guide](express-auth-user-guide) — Detailed guide to planned auth routes, host services, context shape, and error behavior.
- [Migrate Express Apps to Planned Auth Routes](migrate-express-apps-to-planned-auth) — Step-by-step tutorial for converting old `app.get(path, handler)` scripts.
- `examples/xgoja/17-express-planned-auth/scripts/server.js` — Compact planned auth route example.
- `examples/xgoja/18-express-auth-host` — Runnable Go host example for auth, resources, CSRF, audit, and strict mode.
