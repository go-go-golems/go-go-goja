---
Title: go-go-goja Express and ui.dsl module upstreaming design guide
Ticket: GO-GO-GOJA-EXPRESS-UIDSL-MODULES
Status: active
Topics:
  - goja
  - ui-dsl
  - web-ui
  - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
  - path: /home/manuel/code/wesen/2026-05-07--db-browser/internal/web
    note: Current db-browser copy/adaptation of the Express-style HTTP host.
  - path: /home/manuel/code/wesen/2026-05-07--db-browser/internal/uidsl
    note: Current db-browser ui.dsl implementation, including rich tables and inspection components.
  - path: /home/manuel/code/wesen/2026-05-03--goja-hosting-site/pkg/web
    note: Original Express-style HTTP host copied into db-browser.
  - path: /home/manuel/code/wesen/2026-05-03--goja-hosting-site/pkg/uidsl
    note: Original low-level ui.dsl renderer copied into db-browser.
  - path: /home/manuel/code/wesen/corporate-headquarters/go-go-goja/modules/common.go
    note: NativeModule registry interface and default module registration pattern.
  - path: /home/manuel/code/wesen/corporate-headquarters/go-go-goja/engine/runtime_modules.go
    note: RuntimeModuleRegistrar interface needed for runtime-scoped modules such as express.
  - path: /home/manuel/code/wesen/corporate-headquarters/go-go-goja/modules/database/database.go
    note: Existing module option/declaration pattern to follow for configurable modules.
ExternalSources: []
Summary: "Initial design for upstreaming db-browser/goja-hosting-site Express-style web hosting and ui.dsl into go-go-goja as reusable packages."
LastUpdated: 2026-05-08T18:35:00-04:00
WhatFor: "Use this before implementing go-go-goja packages for Express-style HTTP hosting and server-rendered ui.dsl."
WhenToUse: "Read when moving code from db-browser/goja-hosting-site into corporate-headquarters/go-go-goja or when updating downstream repos to use the upstream packages."
---

# go-go-goja Express and ui.dsl module upstreaming design guide

## Executive Summary

`db-browser` currently carries local copies of two generally useful Goja-hosting packages:

1. `internal/web`: an Express-style HTTP host that lets JavaScript register route handlers with `app.get(...)`, `app.post(...)`, and response helpers such as `res.html(...)` and `res.json(...)`.
2. `internal/uidsl`: a server-rendered HTML node DSL exposed as `require("ui.dsl")` / `require("ui")`, including tag helpers, document rendering, rich tables, code blocks, badges, and tabs.

Both originated from `../2026-05-03--goja-hosting-site`, then `db-browser` adapted and extended them. `internal/web` remains very close to the original package. `internal/uidsl` is now a superset: it includes the original low-level renderer plus rich inspection components built during the db-browser work.

The best upstream shape in `go-go-goja` is to split responsibilities into three packages:

```text
go-go-goja/
├── pkg/gojahttp/
│   ├── host.go
│   ├── request_response.go
│   ├── route_registry.go
│   ├── session.go
│   ├── body.go
│   └── *_test.go
├── modules/express/
│   ├── express.go
│   ├── express_test.go
│   └── typescript.go
└── modules/uidsl/
    ├── node.go
    ├── render.go
    ├── module.go
    ├── table.go
    ├── components.go
    └── *_test.go
```

`pkg/gojahttp` should contain the reusable HTTP host. `modules/express` should be a small runtime-scoped adapter that exposes that host as `require("express")`. `modules/uidsl` should contain the reusable UI DSL and renderer.

The most important design decision is that `express` must **not** be a default globally registered `modules.NativeModule`. It requires a configured `Host`, renderer, session manager, and runtime owner. It should be installed through `engine.RuntimeModuleRegistrar`, the same mechanism that can receive runtime-scoped context from the engine.

## Problem Statement

The current implementation is duplicated across three repositories:

```text
/home/manuel/code/wesen/2026-05-03--goja-hosting-site/pkg/web
/home/manuel/code/wesen/2026-05-03--goja-hosting-site/pkg/uidsl
/home/manuel/code/wesen/2026-05-07--db-browser/internal/web
/home/manuel/code/wesen/2026-05-07--db-browser/internal/uidsl
/home/manuel/code/wesen/corporate-headquarters/go-go-goja
```

`db-browser` copied `pkg/web` and `pkg/uidsl` as requested, then adapted imports and extended `ui.dsl`. That was a good implementation shortcut, but it creates drift:

- future fixes to `express` routing/session/body parsing need to be applied in multiple repos;
- future `ui.dsl` components would likely be useful beyond db-browser;
- consumers of `go-go-goja` cannot currently build server-rendered JavaScript apps without copying these packages again;
- TypeScript declarations and module documentation cannot live in one canonical place;
- the current package names are local (`internal/web`, `pkg/web`) rather than a reusable `go-go-goja` module boundary.

The upstreaming project should make the Express-style host and `ui.dsl` first-class reusable parts of `go-go-goja` while preserving db-browser's current JavaScript API.

## Current State Analysis

### `internal/web` versus `goja-hosting-site/pkg/web`

A recursive diff shows that db-browser's `internal/web` is nearly identical to the original `goja-hosting-site/pkg/web`. The main differences are tests:

- imports changed from `github.com/go-go-golems/goja-site/pkg/uidsl` to `github.com/go-go-golems/db-browser/internal/uidsl`;
- copied tests removed calls to `engine.NewBuilder().UseModuleMiddleware(engine.MiddlewareOnly("time"))` because the db-browser dependency version initially did not expose the same API shape.

The production package files are effectively reusable as the starting point for `pkg/gojahttp` and `modules/express`.

Current files:

```text
internal/web/body.go
internal/web/express_module.go
internal/web/host.go
internal/web/request_response.go
internal/web/route_registry.go
internal/web/session.go
```

### `internal/uidsl` versus `goja-hosting-site/pkg/uidsl`

`goja-hosting-site/pkg/uidsl` contains the base renderer:

```text
module.go
node.go
render.go
render_test.go
```

`db-browser/internal/uidsl` now contains the base renderer plus:

```text
components.go
components_test.go
table.go
table_test.go
table_rich_test.go
table_filters_test.go
table_links_test.go
```

The db-browser version is therefore the canonical candidate for upstreaming. It includes:

- safe text and attribute escaping;
- `ui.page`, tag helpers, `ui.text`, `ui.raw`, and `ui.render`;
- `ui.table.fromRows(...)` and `ui.table(id).data(...)`;
- table columns, filters, sorting, pagination, money/date/tags/badge kinds, links, empty states;
- `ui.codeBlock`, `ui.sql`, `ui.js`, `ui.jsonBlock`;
- `ui.badge`;
- `ui.tabs` with CSS-only radio switching and per-instance CSS.

### `go-go-goja` module architecture

`go-go-goja` already has two relevant extension mechanisms.

The first is `modules.NativeModule`:

```go
type NativeModule interface {
    Name() string
    Doc() string
    Loader(*goja.Runtime, *goja.Object)
}
```

This is correct for modules such as `yaml`, `path`, `time`, and `database` when no per-runtime host object is needed.

The second is `engine.RuntimeModuleRegistrar`:

```go
type RuntimeModuleRegistrar interface {
    ID() string
    RegisterRuntimeModules(ctx *RuntimeModuleContext, reg *require.Registry) error
}
```

This is the correct fit for `express`, because `express` needs access to:

- the configured `Host`;
- the current runtime owner (`ctx.Owner`);
- optionally future runtime-scoped values such as sessions, event bridges, or per-runtime app IDs.

## Goals

1. Provide a reusable Express-style HTTP host in `go-go-goja`.
2. Provide a reusable `ui.dsl` module in `go-go-goja`.
3. Preserve the current JavaScript API used by db-browser examples:

   ```js
   const express = require("express");
   const ui = require("ui.dsl");
   const app = express.app();
   ```

4. Keep `express` runtime-scoped rather than default-global.
5. Keep the HTTP host renderer-neutral; it should accept a renderer function rather than import `ui.dsl` directly.
6. Add TypeScript declarations for both modules.
7. Add runtime integration tests proving `require("express")` and `require("ui.dsl")` work in `go-go-goja` runtimes.
8. Migrate `goja-hosting-site` and `db-browser` to use the upstream packages and delete local copies.

## Non-goals

- Do not implement full Express compatibility.
- Do not implement Express middleware/`next` semantics in the first upstreaming pass.
- Do not make `express` available in every default runtime.
- Do not couple `express` to SQLite or db-browser.
- Do not implement server-interactive UI events in this ticket.
- Do not require client-side JavaScript for `ui.dsl` components in the first pass.

## Proposed Package Layout

### `pkg/gojahttp`

Move host infrastructure here:

```text
pkg/gojahttp/body.go
pkg/gojahttp/host.go
pkg/gojahttp/request_response.go
pkg/gojahttp/route_registry.go
pkg/gojahttp/session.go
```

Suggested public API:

```go
package gojahttp

type Renderer func(*goja.Runtime, goja.Value) (string, error)

type HostOptions struct {
    Dev      bool
    Renderer Renderer
    Sessions SessionOptions
}

type Host struct { ... }

func NewHost(opts HostOptions) *Host
func (h *Host) SetRuntime(owner runtimeowner.Runner)
func (h *Host) Register(method, pattern string, handler goja.Callable)
func (h *Host) RegisterStatic(prefix, dir string)
func (h *Host) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

This package should not import `modules/uidsl`. It accepts a renderer so different applications can provide different HTML rendering strategies.

### `modules/express`

Move the current `express_module.go` into a module adapter package:

```text
modules/express/express.go
modules/express/express_test.go
modules/express/typescript.go
```

Suggested API:

```go
package expressmod

type Option func(*Registrar)

type Registrar struct {
    host *gojahttp.Host
    name string
}

func NewRegistrar(host *gojahttp.Host, opts ...Option) *Registrar
func WithName(name string) Option

func (r *Registrar) ID() string
func (r *Registrar) RegisterRuntimeModules(ctx *engine.RuntimeModuleContext, reg *require.Registry) error
```

`RegisterRuntimeModules` should set the host runtime owner and register the native module:

```go
func (r *Registrar) RegisterRuntimeModules(ctx *engine.RuntimeModuleContext, reg *require.Registry) error {
    if r.host == nil {
        return fmt.Errorf("express registrar requires host")
    }
    r.host.SetRuntime(ctx.Owner)
    reg.RegisterNativeModule(r.name, r.loader)
    return nil
}
```

The loader should export:

```js
express.app(): App
```

`App` should support:

```js
app.get(pattern, handler)
app.post(pattern, handler)
app.put(pattern, handler)
app.patch(pattern, handler)
app.delete(pattern, handler)
app.all(pattern, handler)
app.static(prefix, dir)
```

### `modules/uidsl`

Move db-browser's richer UI DSL into:

```text
modules/uidsl/node.go
modules/uidsl/render.go
modules/uidsl/module.go
modules/uidsl/table.go
modules/uidsl/components.go
```

Suggested public API:

```go
package uidsl

func RenderAny(vm *goja.Runtime, v goja.Value) (string, error)
func Loader(vm *goja.Runtime, moduleObj *goja.Object)
func NewRegistrar() *Registrar
```

`NewRegistrar()` should register both `ui.dsl` and `ui`:

```go
reg.RegisterNativeModule("ui.dsl", Loader)
reg.RegisterNativeModule("ui", Loader)
```

A future optional `modules.NativeModule` wrapper can be added later if we want `ui.dsl` to participate in the default module registry. Start with the explicit registrar because it preserves the current db-browser pattern and avoids surprising default availability.

## JavaScript API Contract

### Express-style host

The first stable API should be:

```js
const express = require("express");
const app = express.app();

app.get("/", (req, res) => {
  res.html("<h1>Hello</h1>");
});

app.post("/api/echo", (req, res) => {
  res.json({ body: req.body });
});
```

Request object:

```ts
type Request = {
  method: string;
  url: string;
  path: string;
  query: Record<string, string | string[]>;
  params: Record<string, string>;
  headers: Record<string, string>;
  cookies: Record<string, string>;
  session: Session | null;
  ip: string;
  body: unknown;
  rawBody: string;
};
```

Response object:

```ts
type Response = {
  status(code: number): Response;
  set(name: string, value: string): Response;
  type(value: string): Response;
  json(value: unknown): void;
  send(value?: unknown): void;
  html(value: unknown): void;
  redirect(url: string): void;
  redirect(status: number, url: string): void;
  end(): void;
};
```

### UI DSL

The first stable API should include:

```js
const ui = require("ui.dsl");

ui.page({ title }, ...children)
ui.fragment(...children)
ui.text(value)
ui.raw(html)
ui.render(value)

ui.div(attrs?, ...children)
ui.span(attrs?, ...children)
ui.a(attrs, ...children)
ui.form(attrs?, ...children)
ui.input(attrs)
// plus the current tag list
```

Inspection components:

```js
ui.table.fromRows(id, rows)
ui.table(id).data(fn)
ui.codeBlock(language, source, options?)
ui.sql(source, options?)
ui.js(source, options?)
ui.jsonBlock(value, options?)
ui.badge(value, options?)
ui.tabs(id, tabs, options?)
```

The safety contract should be explicit in docs:

- normal text and attribute values are escaped;
- `ui.raw` bypasses escaping and is only for trusted HTML/CSS;
- code blocks and JSON blocks must escape source text by default;
- tab IDs, badge values, and language classes are normalized into CSS/DOM-safe tokens.

## Recommended Consumer Usage

A consumer should configure the host explicitly:

```go
host := gojahttp.NewHost(gojahttp.HostOptions{
    Dev:      true,
    Renderer: uidsl.RenderAny,
})

factory, err := engine.NewBuilder().
    UseModuleMiddleware(engine.MiddlewareOnly("fs", "path", "yaml", "time", "timer")).
    WithRuntimeModuleRegistrars(
        expressmod.NewRegistrar(host),
        uidsl.NewRegistrar(),
    ).
    Build()
```

Then load app code:

```go
rt, err := factory.NewRuntime(ctx)
if err != nil {
    return err
}

_, err = rt.Owner.Call(ctx, "load-script", func(_ context.Context, vm *goja.Runtime) (any, error) {
    _, err := vm.RunScript("app.js", source)
    return nil, err
})
```

Then serve:

```go
return http.ListenAndServe(":8080", host)
```

App JavaScript:

```js
const express = require("express");
const ui = require("ui.dsl");

const app = express.app();

app.get("/", (req, res) => {
  res.html(ui.page(
    { title: "Hello" },
    ui.h1("Hello from Goja"),
    ui.p("Path: " + req.path)
  ));
});
```

## TypeScript Declaration Plan

`modules/express` should implement a TypeScript declaration provider if the current `go-go-goja` declaration generator can represent the module shape. If the generator cannot describe nested interfaces well enough, store a hand-authored declaration document next to the module and wire it into the docs first.

Initial `express` declaration:

```ts
declare module "express" {
  export function app(): App;

  export interface App {
    get(pattern: string, handler: Handler): void;
    post(pattern: string, handler: Handler): void;
    put(pattern: string, handler: Handler): void;
    patch(pattern: string, handler: Handler): void;
    delete(pattern: string, handler: Handler): void;
    all(pattern: string, handler: Handler): void;
    static(prefix: string, dir: string): void;
  }

  export type Handler = (req: Request, res: Response) => unknown;

  export interface Request {
    method: string;
    url: string;
    path: string;
    query: Record<string, string | string[]>;
    params: Record<string, string>;
    headers: Record<string, string>;
    cookies: Record<string, string>;
    session: Session | null;
    ip: string;
    body: unknown;
    rawBody: string;
  }

  export interface Session {
    id: string;
    isNew: boolean;
    cookieName: string;
  }

  export interface Response {
    status(code: number): Response;
    set(name: string, value: string): Response;
    type(value: string): Response;
    json(value: unknown): void;
    send(value?: unknown): void;
    html(value: unknown): void;
    redirect(url: string): void;
    redirect(status: number, url: string): void;
    end(): void;
  }
}
```

Initial `ui.dsl` declaration should cover common tag helpers and rich components. It can start conservative with `unknown`/`Node` where the generator cannot express fluent builders.

## Implementation Plan

### T01 — Create upstream package skeletons

In `/home/manuel/code/wesen/corporate-headquarters/go-go-goja`:

```text
pkg/gojahttp
modules/express
modules/uidsl
```

Copy from db-browser first because db-browser is the current adapted/superset version.

### T02 — Move HTTP host into `pkg/gojahttp`

Copy and rename package from `web` to `gojahttp`:

```text
internal/web/body.go              -> pkg/gojahttp/body.go
internal/web/host.go              -> pkg/gojahttp/host.go
internal/web/request_response.go  -> pkg/gojahttp/request_response.go
internal/web/route_registry.go    -> pkg/gojahttp/route_registry.go
internal/web/session.go           -> pkg/gojahttp/session.go
```

Cleanups:

- rename default cookie from `goja_site_session` to a neutral `goja_http_session` or `go_go_goja_session`;
- update comments that mention `goja-site`;
- keep `HostOptions.Renderer` as an injected function;
- keep session configuration unchanged except naming.

### T03 — Add `modules/express`

Move/adapt `express_module.go` into `modules/express`.

Do not call `modules.Register(...)` in `init()` for the first version.

Expose:

```go
func NewRegistrar(host *gojahttp.Host, opts ...Option) *Registrar
func WithName(name string) Option
```

Default module name should be `express`.

### T04 — Move `ui.dsl` into `modules/uidsl`

Copy db-browser's `internal/uidsl` into `modules/uidsl`.

Keep package name `uidsl`.

Expose:

```go
func RenderAny(vm *goja.Runtime, v goja.Value) (string, error)
func NewRegistrar() *Registrar
func Loader(vm *goja.Runtime, moduleObj *goja.Object)
```

Register aliases `ui.dsl` and `ui` through the registrar.

### T05 — Add tests in go-go-goja

Move/adapt tests:

```text
pkg/gojahttp/route_registry_test.go
pkg/gojahttp/session_test.go
modules/express/express_test.go
modules/uidsl/render_test.go
modules/uidsl/table_test.go
modules/uidsl/table_rich_test.go
modules/uidsl/table_filters_test.go
modules/uidsl/table_links_test.go
modules/uidsl/components_test.go
```

Add an integration test that builds an engine runtime with both registrars and serves a route via `httptest`.

### T06 — Add docs and declarations

Add docs under `go-go-goja/pkg/doc`, for example:

```text
pkg/doc/18-express-module.md
pkg/doc/19-uidsl-module.md
```

Document:

- host setup from Go;
- JavaScript route API;
- request/response shapes;
- renderer injection;
- `ui.dsl` escaping model;
- table/component APIs;
- limitations compared with Express.

Add TypeScript declarations or declaration descriptors where practical.

### T07 — Validate upstream package

Run in `go-go-goja`:

```bash
go test ./pkg/gojahttp ./modules/express ./modules/uidsl -count=1
go test ./... -count=1
GOWORK=off go test ./... -count=1
```

If lint is configured and quick enough:

```bash
GOWORK=off make lint
```

### T08 — Update goja-hosting-site

Replace local imports with upstream:

```go
github.com/go-go-golems/go-go-goja/pkg/gojahttp
github.com/go-go-golems/go-go-goja/modules/express
github.com/go-go-golems/go-go-goja/modules/uidsl
```

Delete or deprecate local `pkg/web` and `pkg/uidsl` after tests pass.

### T09 — Update db-browser

Replace local imports:

```go
github.com/go-go-golems/db-browser/internal/web
github.com/go-go-golems/db-browser/internal/uidsl
```

with upstream imports:

```go
gojahttp "github.com/go-go-golems/go-go-goja/pkg/gojahttp"
expressmod "github.com/go-go-golems/go-go-goja/modules/express"
uidsl "github.com/go-go-golems/go-go-goja/modules/uidsl"
```

Then delete local copies if no db-browser-specific behavior remains.

## Test Plan

### `pkg/gojahttp`

Tests should cover:

- exact route matching;
- `:param` extraction;
- wildcard routes;
- method matching and `all` behavior;
- HEAD fallback to GET without body;
- JSON body parsing;
- form body parsing;
- cookie/session issue and reuse;
- static mount serving;
- dev error response versus production error response.

### `modules/express`

Tests should cover:

- `require("express")` loads when registrar is installed;
- `express.app()` returns an app object;
- route registration calls `Host.Register`;
- handler receives `req` and `res`;
- `res.html(ui.h1("ok"))` renders through injected renderer;
- `res.json(...)` returns JSON;
- invalid handler value throws/returns a useful error.

### `modules/uidsl`

Tests should cover the current db-browser behavior:

- text and attribute escaping;
- `ui.page` document layout;
- `ui.raw` bypass behavior;
- tag helper argument parsing;
- `ui.table.fromRows`;
- dynamic `ui.table(id).data(ctx => ...)`;
- filters, sorting, pagination;
- cell links;
- code block escaping and token classes;
- JSON block safe formatting;
- badge tone/value classes;
- tabs selected fallback, duplicate IDs, disabled tabs, and generated CSS.

### Downstream validation

After upstreaming, run in db-browser:

```bash
go test ./...
ttmp/2026/05/07/DB-BROWSER-JSVERBS-DESIGN--goja-jsverbs-database-browser-web-app-design/scripts/011-final-validation.sh
ttmp/2026/05/07/DB-BROWSER-UIDSL-COMPONENTS--ui-dsl-component-spec-for-code-blocks-badges-and-tabs/scripts/001-uidsl-components-smoke.sh
```

## Design Decisions

### Decision: `express` is runtime-scoped

`express` requires a host and runtime owner, so it belongs behind `engine.RuntimeModuleRegistrar`. It should not be default-registered through `modules.Register` in the first version.

### Decision: the HTTP host is renderer-neutral

`pkg/gojahttp` accepts a renderer callback. It should not import `modules/uidsl`. This lets consumers render with `ui.dsl`, plain strings, or other renderers.

### Decision: `ui.dsl` moves as the rich db-browser version

The richer db-browser implementation is the useful reusable surface. Upstreaming only the older low-level renderer would immediately require another downstream extension package.

### Decision: preserve current JavaScript names

Keep:

```js
require("express")
require("ui.dsl")
require("ui")
```

Changing names would break existing examples and produce no meaningful architectural gain.

### Decision: call it Express-style, not Express-compatible

The module implements a compact route API inspired by Express. It does not implement middleware stacks, `next`, routers, template engines, or npm package compatibility.

## Alternatives Considered

### Alternative A: keep packages copied in each app

Rejected because duplicated host and UI fixes will drift across repos.

### Alternative B: put all code under `modules/express`

Rejected because the HTTP host is reusable Go infrastructure, while the JavaScript module adapter is only one way to expose it.

### Alternative C: make `express` a default `NativeModule`

Rejected because default modules cannot carry a configured host and runtime owner. A default `require("express")` without a host would fail at request registration time.

### Alternative D: make `express` depend directly on `ui.dsl`

Rejected because the host should be renderer-neutral. `res.html(...)` should call a configured renderer.

### Alternative E: upstream only low-level `ui.dsl`

Rejected for now because db-browser already proved rich tables and inspection components are generally useful for Goja-hosted internal tools.

## Migration Notes for db-browser

`internal/app/server.go` should change from:

```go
host := web.NewHost(web.HostOptions{Dev: cfg.Dev, Renderer: uidsl.RenderAny})
...
WithRuntimeModuleRegistrars(web.NewExpressRegistrar(host), uidsl.NewRegistrar())
```

to:

```go
host := gojahttp.NewHost(gojahttp.HostOptions{Dev: cfg.Dev, Renderer: uidsl.RenderAny})
...
WithRuntimeModuleRegistrars(expressmod.NewRegistrar(host), uidsl.NewRegistrar())
```

After migration, delete:

```text
internal/web
internal/uidsl
```

unless db-browser keeps a small local extension layer. The preferred result is no local copy.

## Open Questions

1. Should `ui.dsl` also be default-registered as a `modules.NativeModule`, or only runtime-registered at first?
2. What should the neutral session cookie name be: `goja_http_session`, `go_go_goja_session`, or configurable-only with no public default mention?
3. Should `modules/express` offer `WithName(...)` for aliases such as `web`, or only `express`?
4. Should TypeScript declarations be generated through `modules.TypeScriptDeclarer`, hand-authored, or both?
5. Should rich `ui.table` move upstream before scoped table query state, or should scoped state be implemented first in db-browser and then moved?
6. Should the upstream `ui.dsl` include retro theme helpers, or should theme assets remain downstream until static asset support is clearer?

## Risks

### Runtime-scoped module confusion

Consumers may expect `require("express")` to work with `DefaultRegistryModules()`. Documentation must show that `express` is installed with `expressmod.NewRegistrar(host)`.

### API stabilization too early

Moving rich `ui.dsl` upstream makes table/component APIs more durable. This is acceptable, but docs should mark some features as current behavior rather than final API if they remain experimental.

### Package naming bikeshed

`pkg/gojahttp` is explicit and avoids collision with `net/http`, but names are easy to debate. Avoid blocking implementation on naming after a reasonable choice is made.

### Downstream module version timing

db-browser depends on a tagged `go-go-goja` version. After upstreaming, either tag a new release or use a temporary replace directive during migration.

## Recommended First Pull Request

The first upstream PR should be intentionally small:

1. Add `pkg/gojahttp` from db-browser `internal/web`, excluding `express_module.go`.
2. Rename package and comments.
3. Add/move tests for route registry, sessions, request/response, and host basics.
4. Do not add `modules/express` yet.
5. Run `go test ./pkg/gojahttp -count=1` and `go test ./... -count=1`.

The second PR can add `modules/express`. The third can add `modules/uidsl`.

## References

Primary source packages:

```text
/home/manuel/code/wesen/2026-05-07--db-browser/internal/web
/home/manuel/code/wesen/2026-05-07--db-browser/internal/uidsl
/home/manuel/code/wesen/2026-05-03--goja-hosting-site/pkg/web
/home/manuel/code/wesen/2026-05-03--goja-hosting-site/pkg/uidsl
```

Upstream target:

```text
/home/manuel/code/wesen/corporate-headquarters/go-go-goja
```

Relevant go-go-goja files:

```text
modules/common.go
modules/exports.go
modules/typing.go
modules/database/database.go
modules/yaml/yaml.go
engine/runtime_modules.go
engine/module_specs.go
pkg/doc/02-creating-modules.md
```

Downstream db-browser files to update after upstreaming:

```text
cmd/db-browser/main.go
internal/app/server.go
internal/verbcli/runtime.go
internal/doc/topics/js-api-reference.md
examples/generic-browser/scripts/app.js
examples/playwright-smoke/scripts/app.js
examples/yaml-dashboard/scripts/app.js
```
