---
Title: Mountable HTTP handler ABI design and implementation guide
Ticket: ""
Status: active
Topics:
    - goja
    - http
    - express
    - xgoja
    - websocket
DocType: ""
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/modules/express/express.go
      Note: Express app API to extend with mount
    - Path: go-go-goja/modules/express/express.go:Current Express-like JS app API
    - Path: go-go-goja/modules/express/typescript.go:Current Express TypeScript declarations
    - Path: go-go-goja/pkg/gojahttp/host.go
      Note: HTTP host dispatch and mount behavior
    - Path: go-go-goja/pkg/gojahttp/host.go:Current HTTP host, route dispatch, and static prefix mount behavior
    - Path: go-go-goja/pkg/gojahttp/route_registry.go
      Note: Route parameter and wildcard matching
    - Path: go-go-goja/pkg/gojahttp/route_registry.go:Current JavaScript route matching with :params and wildcard segments
    - Path: go-go-goja/pkg/xgoja/providers/http/http.go:xgoja HTTP provider and express module registration
    - Path: go-go-goja/pkg/xgoja/providers/http/serve.go
      Note: xgoja serve command set behavior
    - Path: go-go-goja/pkg/xgoja/providers/http/serve.go:xgoja HTTP serve command-set implementation
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: Use when implementing or reviewing gojahttp/express handler mounting, WebSocket composition, or xgoja HTTP provider integration.
---


# Mountable HTTP handler ABI design and implementation guide

## Executive summary

`gojahttp` and the `express` module currently let JavaScript register JavaScript route handlers and static asset handlers. They do not let JavaScript compose two Go-backed modules when one module exposes an `http.Handler` and another owns the HTTP server. This blocks ergonomic use cases such as mounting a `sessionstream` WebSocket server from JavaScript:

```js
const express = require("express")
const ss = require("sessionstream")

const app = express.app()
const hub = ss.hub({ schemas })

app.mount("/ws", ss.webSocket.server(hub))
```

The proposed solution is a small shared ABI in `pkg/gojahttp`: JavaScript-visible objects can carry a hidden, non-enumerable Go `http.Handler` reference. The Express module can unwrap that reference and register it on the underlying `gojahttp.Host`. Sessionstream and future modules can expose mountable handler objects without depending on Express internals.

This keeps JavaScript as the composition layer while Go continues to own low-level HTTP/WebSocket semantics.

## Problem statement

The immediate downstream need comes from `sessionstream`. Its WebSocket transport is a Go `http.Handler` (`transport/ws.Server`) that performs WebSocket upgrades, hydration, subscription management, and UI event fanout. That logic should not be reimplemented as a JavaScript Express handler.

Today, however, the Express app API only exposes:

```js
app.get(pattern, handler)
app.post(pattern, handler)
app.put(pattern, handler)
app.patch(pattern, handler)
app.delete(pattern, handler)
app.all(pattern, handler)
app.static(prefix, dir)
app.staticFromAssetsModule(prefix, assets, root)
app.spaFromAssetsModule(prefix, assets, root, options)
app.listen()
```

There is no API for:

```js
app.mount("/ws", goBackedHandler)
```

Provider-side mounting is possible through xgoja host services, but that makes transport composition declarative-only. The user wants JavaScript to be able to connect two Go-backed pieces: the server (`express`/`gojahttp.Host`) and the handler (`sessionstream` WebSocket server).

## Current-state evidence

### gojahttp host dispatch

`pkg/gojahttp/host.go` has two dispatch layers:

1. static mounts in `Host.static`, checked first;
2. JavaScript routes in `Host.registry`, checked after static mounts.

Static mounts are prefix-based:

```go
func staticMountMatches(prefix, requestPath string) bool {
    prefix = cleanPath(prefix)
    requestPath = cleanPath(requestPath)
    if prefix == "/" {
        return true
    }
    return requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/")
}
```

`RegisterStaticHandlerWithOptions` currently wraps every non-root handler with `http.StripPrefix(prefix, handler)`.

### JavaScript route matching already supports params and wildcard segments

`pkg/gojahttp/route_registry.go` supports route parameters and a simple wildcard segment:

```go
func matchPattern(pattern, path string) (map[string]string, bool) {
    pp := splitPath(pattern)
    sp := splitPath(path)
    params := map[string]string{}
    for i := 0; i < len(pp); i++ {
        if pp[i] == "*" {
            return params, true
        }
        if i >= len(sp) {
            return nil, false
        }
        if strings.HasPrefix(pp[i], ":") {
            name := strings.TrimPrefix(pp[i], ":")
            if name == "" {
                return nil, false
            }
            params[name] = sp[i]
            continue
        }
        if pp[i] != sp[i] {
            return nil, false
        }
    }
    return params, len(pp) == len(sp)
}
```

So current route semantics are:

- `/users/:id` matches `/users/42` and exposes `params.id = "42"`.
- `/assets/*` matches `/assets/a/b/c`.
- `*` is a segment, not a glob inside a segment.
- wildcard captures no value today; it only terminates matching successfully.
- static/handler mounts are prefix-based, not pattern-based.

### xgoja HTTP serve exists as a provider command set

`pkg/xgoja/providers/http/http.go` registers:

- runtime module `express`;
- command-set provider `serve`.

In xgoja v2 config, the serve command can be mounted like this:

```yaml
commands:
  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [sites]
```

This command executes JS verbs that register Express routes and then keeps the runtime alive. Hot reload uses an external `gojahttp.Host` service. This means provider-side mounting and JS-side mounting should both target the same underlying `gojahttp.Host` abstraction.

## Proposed solution

### Add a shared mountable handler ABI

Add `pkg/gojahttp/mountable.go`:

```go
const hiddenHTTPHandlerKey = "__go_go_goja_http_handler"

type HandlerRef struct {
    Handler http.Handler
}

func AttachHTTPHandler(vm *goja.Runtime, obj *goja.Object, handler http.Handler) error
func HTTPHandlerFromValue(value goja.Value) (http.Handler, bool)
```

`AttachHTTPHandler` should store the reference as a hidden, non-enumerable, non-writable, non-configurable property. This mirrors `protogoja`'s hidden protobuf refs and avoids exposing Go pointers through normal JavaScript enumeration.

### Add explicit host handler mount APIs

Add a mount API separate from static asset naming:

```go
type MountOptions struct {
    StripPrefix bool
    ExcludePrefixes []string
}

func (h *Host) RegisterHandler(prefix string, handler http.Handler)
func (h *Host) RegisterHandlerWithOptions(prefix string, handler http.Handler, opts MountOptions)
```

`RegisterStaticHandlerWithOptions` can delegate to `RegisterHandlerWithOptions` with `StripPrefix: true` to preserve existing behavior.

For generic handler mounting, the default should be `StripPrefix: false`. WebSocket handlers commonly expect to see the original URL path, and preserving the path is less surprising for generic handlers. If a handler wants stripped paths, JavaScript can opt in:

```js
app.mount("/assets", handler, { stripPrefix: true })
```

### Add Express mount APIs

Expose both a short and explicit API:

```js
app.mount(prefix, handler, options?)
app.mountHandler(prefix, handler, options?)
```

Both unwrap the hidden `http.Handler` ref and register it with the host:

```go
handler, ok := gojahttp.HTTPHandlerFromValue(handlerValue)
if !ok {
    return fmt.Errorf("app.mount(%q) requires a Go http.Handler-backed object", prefix)
}
r.host.RegisterHandlerWithOptions(prefix, handler, gojahttp.MountOptions{StripPrefix: options.stripPrefix})
```

### TypeScript shape

The declaration should include a marker interface:

```ts
export interface MountableHandler {}

export interface App {
  mount(prefix: string, handler: MountableHandler, options?: MountOptions): void;
  mountHandler(prefix: string, handler: MountableHandler, options?: MountOptions): void;
}

export interface MountOptions {
  stripPrefix?: boolean;
  excludePrefixes?: string[];
}
```

The marker interface does not provide runtime safety by itself. Runtime safety comes from the hidden Go ref.

## Routing and wildcard answer

The current system has two kinds of matching:

1. **JavaScript routes** (`app.get`, `app.post`, etc.) use route patterns:
   - `:param` captures one path segment;
   - `*` matches the remainder, but does not currently expose a captured wildcard value.
2. **Mounted handlers** (`app.mount`) should use prefix matching:
   - `/ws` matches `/ws` and `/ws/...`;
   - no `:param` extraction for mounted Go handlers in the first implementation;
   - no wildcard syntax needed because prefix mount already covers the primary handler use case.

This is the right division. If a Go handler needs route params, it can parse the path itself, or a future API can attach mount metadata to request context. For WebSockets, prefix matching is sufficient.

## Design decisions

### Decision: shared ABI lives in `pkg/gojahttp`

Options considered:

1. Put hidden handler refs in `modules/express`.
2. Put hidden handler refs in each producer module such as sessionstream.
3. Put hidden handler refs in `pkg/gojahttp`.

Decision: use `pkg/gojahttp`.

Rationale: `gojahttp` is the shared HTTP substrate. Express is one consumer; sessionstream is one producer. A shared ABI lets future modules expose SSE, Prometheus, health check, file server, MCP, or other Go-backed handlers without depending on Express.

### Decision: hidden non-enumerable refs, not public `.handler`

The handler pointer should not appear in `Object.keys`, JSON, or ordinary JS inspection. A visible property would be easy to overwrite and would encourage treating the handler as a normal JS value. The runtime contract is Go-owned.

### Decision: default generic mounts do not strip prefix

Existing static asset mounts strip prefix. Generic handler mounts should preserve path by default because the handler is a full `http.Handler`. WebSocket and API handlers often need the original path for logging, origin checks, or subprotocol routing.

Keep static behavior unchanged by making static mounts delegate with `StripPrefix: true`.

### Decision: mounted handlers are checked before JS routes for now

This preserves existing static mount precedence. A mounted `/ws` handler should intercept WebSocket upgrade requests before JavaScript routes. The behavior should be documented and tested.

## Implementation plan

### Task 1: current-state analysis

- Confirm `:param` and `*` behavior in `route_registry.go`.
- Confirm static prefix matching and strip-prefix behavior in `host.go`.
- Confirm `express` does not expose arbitrary Go handler mounting.
- Confirm xgoja HTTP provider has a `serve` command-set and host-service path.

### Task 2: mountable handler ABI

Add `pkg/gojahttp/mountable.go` with:

- `HandlerRef`;
- `AttachHTTPHandler`;
- `HTTPHandlerFromValue`.

Tests:

- plain object does not extract;
- attached handler extracts;
- hidden property is non-enumerable;
- extracted handler can serve through `httptest`.

### Task 3: Host mount API

Refactor `StaticMount` into a more generic `Mount`, or extend it without breaking exported compatibility.

Preferred conservative implementation:

```go
type MountOptions struct {
    StripPrefix bool
    ExcludePrefixes []string
}

func (h *Host) RegisterHandler(prefix string, handler http.Handler)
func (h *Host) RegisterHandlerWithOptions(prefix string, handler http.Handler, opts MountOptions)
```

Keep `RegisterStaticHandlerWithOptions` as a wrapper.

Tests:

- `RegisterHandler("/api", handler)` preserves path `/api/ping`.
- `RegisterHandlerWithOptions(... StripPrefix:true)` strips path to `/ping`.
- exclude prefixes still work.
- handler mounts retain existing priority before JS routes.

### Task 4: Express API

Add to `appObject`:

```js
app.mount(prefix, handler, options?)
app.mountHandler(prefix, handler, options?)
```

Tests:

- JS can call `app.mount("/go", goHandlerObject)`.
- `host.ServeHTTP` dispatches to the Go handler.
- invalid plain object produces a useful error.
- options pass through `stripPrefix` and `excludePrefixes`.

### Task 5: route pattern tests

Add/extend tests for current route semantics:

- `/users/:id` captures params.
- `/assets/*` matches nested paths.
- wildcard does not capture a value yet.
- mount prefix semantics are separate from route pattern semantics.

### Task 6: docs and declarations

Update:

- `modules/express/typescript.go`;
- `modules/express` docs or xgoja HTTP docs if present;
- ticket diary/changelog.

### Task 7: validation

Run:

```bash
go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http -count=1
go test ./pkg/xgoja/... ./modules/express ./pkg/gojahttp -count=1
go test ./... -count=1
```

## Future sessionstream usage

Once this lands, sessionstream should attach its WebSocket server as a mountable handler:

```go
server, err := ws.NewServer(hub.hub)
obj := m.vm.NewObject()
gojahttp.AttachHTTPHandler(m.vm, obj, server)
```

Then JavaScript can compose:

```js
const express = require("express")
const ss = require("sessionstream")

const app = express.app()
const hub = ss.hub({ schemas })
app.mount("/ws", ss.webSocket.server(hub))
```

## Open questions

1. Should wildcard route matches expose a splat capture later (`params["*"]` or `params.splat`)? Not necessary for handler mounting.
2. Should mounted handlers be ordered among JS routes instead of before all JS routes? Existing host behavior checks static mounts first, so first implementation should preserve that.
3. Should `app.mount` default to `stripPrefix: false` or `true`? This design chooses false for generic handlers and preserves true for static assets.
4. Should xgoja provider config also offer declarative mounts? That remains useful, but JS composition via `app.mount` should be implemented first.
