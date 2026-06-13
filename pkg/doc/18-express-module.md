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

The `express` module exposes a small Express-style route registration API for Goja-hosted applications. It is intentionally **Express-style**, not full Express-compatible: it supports route handlers, Go-backed handler mounts, and static mounts, but not middleware stacks, routers, `next()`, template engines, or npm Express plugins.

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

app.get("/hello/:name", (req, res) => {
  return ui.page(
    { title: "Hello" },
    ui.h1("Hello " + req.params.name)
  );
});

app.post("/api/echo", (req, res) => {
  res.status(201).json({ body: req.body });
});
```

Handlers may call `res.*` explicitly. If a handler returns a string, the host sends it as text/HTML depending on content. If a handler returns any other non-null value, the host calls `res.html(value)` and uses the configured renderer.

## App API

```javascript
app.get(pattern, handler)
app.post(pattern, handler)
app.put(pattern, handler)
app.patch(pattern, handler)
app.delete(pattern, handler)
app.all(pattern, handler)
app.mount(prefix, mountableHandler, options?)
app.mountHandler(prefix, mountableHandler, options?)
app.static(prefix, directory)
app.staticFromAssetsModule(prefix, assetsModule, root)
```

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

## Request object

Handlers receive a plain JavaScript request object:

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
