---
Title: Implementation diary
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
      Note: Express app.mount API
    - Path: go-go-goja/modules/express/express_integration_test.go
      Note: Express Go handler mount integration tests
    - Path: go-go-goja/modules/express/typescript.go
      Note: Mountable handler TypeScript declarations
    - Path: go-go-goja/pkg/doc/18-express-module.md
      Note: User-facing mount docs
    - Path: go-go-goja/pkg/gojahttp/host.go
      Note: Host handler mount API
    - Path: go-go-goja/pkg/gojahttp/host_mount_test.go
      Note: Host mount behavior tests
    - Path: go-go-goja/pkg/gojahttp/mountable.go
      Note: Shared hidden http.Handler ref ABI
    - Path: go-go-goja/pkg/gojahttp/mountable_test.go
      Note: Hidden handler ref tests
    - Path: go-go-goja/pkg/gojahttp/route_registry_pattern_test.go
      Note: Route pattern tests for :params and wildcard
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Capture the implementation journey for adding a shared Go `http.Handler` mountable-object ABI to `go-go-goja`, so JavaScript Express apps can compose Go-backed handlers such as sessionstream WebSocket servers.

## Step 1: Analyze current HTTP, Express, xgoja serve, and routing behavior

I started by mapping the existing HTTP stack after the xgoja v2 merge. The current `gojahttp.Host` already has two dispatch paths: prefix-based static handler mounts checked first, then JavaScript route handlers registered through the route registry. The Express module exposes route helpers and static asset helpers, but it does not expose a general way for JavaScript to mount a Go `http.Handler` object produced by another module.

The current route registry already supports argument-capturing routes and a simple wildcard segment. That means the new handler mount API does not need to solve route parameters for JavaScript routes; it should instead provide prefix-based Go handler mounting as a separate concept.

### Prompt Context

**User prompt (verbatim):**

```text
we should add an API to the express app that allow smounting something that implements the correct interface, through a goja object. then, sessionstream (and other who want to mount handlers) can then expose an object that implements that interface to JS, and the JS can connect the two even though both the server and the handler are go. Does that make sense? sketch it out.
```

**Assistant interpretation:** Design and then implement a shared hidden-reference ABI that lets one Go-backed Goja module expose an `http.Handler` object and lets the Express app mount it from JavaScript.

**Inferred user intent:** The user wants JavaScript to remain the composition layer for Go-backed HTTP modules, especially so sessionstream can expose a WebSocket handler that an Express app can mount.

**Commit (code):** N/A — analysis and ticket setup only.

### What I did

- Inspected `pkg/gojahttp/host.go` for host dispatch and prefix mount behavior.
- Inspected `pkg/gojahttp/route_registry.go` for `:param` and wildcard route matching.
- Inspected `modules/express/express.go` for the current JS app API.
- Inspected `pkg/xgoja/providers/http/http.go` and `serve.go` for provider command-set and host-service behavior.
- Created ticket `GOJA-HTTP-MOUNT-001` with a design doc, diary, and implementation tasks.
- Wrote the design/implementation guide.

### Why

- Sessionstream's WebSocket server is a real Go `http.Handler`, and WebSocket upgrades should stay in Go.
- Express currently owns the HTTP host, but JavaScript cannot connect a Go-backed handler object to that host.
- A shared ABI in `gojahttp` avoids coupling sessionstream directly to Express internals.

### What worked

- Current route semantics are straightforward:
  - `/users/:id` captures one segment as `params.id`.
  - `/assets/*` matches the rest of a path but does not expose a splat capture.
  - static mounts use prefix matching.
- xgoja HTTP provider already has a `serve` command set, so mounted handlers will naturally work under `serve` once Express can mount them.

### What didn't work

- There is no existing `app.mount` or `app.mountHandler` API.
- `RegisterStaticHandlerWithOptions` always strips prefixes, which is probably wrong as the default for generic WebSocket/API handlers.

### What I learned

- Handler mounts and JS route patterns should remain separate concepts. JS routes can use `:params` and `*`; Go handler mounts should use prefix semantics.
- The hidden-ref pattern used by `protogoja` is the right model for HTTP handlers too.

### What was tricky to build

- The main design edge is prefix stripping. Static assets generally want stripped prefixes; generic Go handlers often want the original path. The design therefore introduces explicit mount options and preserves static behavior through a wrapper.

### What warrants a second pair of eyes

- Whether `app.mount` should default to `stripPrefix: false` as proposed.
- Whether wildcard route matches should expose a captured splat in a later change.
- Whether mounted handlers should always precede JS routes or be interleaved by registration order in a future router refactor.

### What should be done in the future

- Implement the shared handler ABI and Express mounting API.
- Later, update sessionstream to attach its WebSocket server as a `gojahttp` mountable handler.

### Code review instructions

- Start with the design doc for the planned ABI and semantics.
- Review `pkg/gojahttp/host.go`, `pkg/gojahttp/route_registry.go`, and `modules/express/express.go` before implementation.

### Technical details

Current route examples:

```text
/users/:id  + /users/42     => params.id = "42"
/assets/*   + /assets/a/b/c => match, no wildcard value captured
/ws mount   + /ws/anything  => prefix match for mounted Go handlers
```

## Step 2: Add mountable handler refs, Host mounts, and Express app.mount

I implemented the core `gojahttp` and Express changes. The shared ABI is now a hidden `http.Handler` reference attached to a JavaScript object, and Express can unwrap that reference through `app.mount` / `app.mountHandler` to register the Go handler on the underlying `gojahttp.Host`.

The implementation keeps generic handler mounts separate from static asset mounts. Static mounts still strip their prefix by default, while generic Go handler mounts preserve the request path unless JavaScript passes `{ stripPrefix: true }`.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the sketched mountable handler ABI and Express composition API in go-go-goja.

**Inferred user intent:** Enable JavaScript apps to compose Go-backed HTTP server and handler modules, especially for sessionstream WebSocket transport mounting.

**Commit (code):** pending.

### What I did

- Added `pkg/gojahttp/mountable.go` with:
  - `HandlerRef`;
  - `AttachHTTPHandler`;
  - `HTTPHandlerFromValue`.
- Added hidden-ref tests in `pkg/gojahttp/mountable_test.go`.
- Added `MountOptions`, `RegisterHandler`, and `RegisterHandlerWithOptions` to `pkg/gojahttp/host.go`.
- Preserved static behavior by making `RegisterStaticHandlerWithOptions` delegate with `StripPrefix: true`.
- Added host mount tests for path preservation, prefix stripping, exclusions, and precedence over JS routes.
- Added explicit route-registry tests for `:param` capture and `*` wildcard behavior.
- Added `app.mount` and `app.mountHandler` to `modules/express`.
- Added Express integration tests for mounting Go handler objects from JavaScript, mount options, and invalid plain objects.
- Updated Express TypeScript declarations and `pkg/doc/18-express-module.md`.

### Why

- This gives all Goja HTTP modules a shared cross-module ABI for mountable Go handlers.
- It lets JavaScript remain the composition layer while avoiding reimplementation of WebSocket upgrade behavior in JavaScript.
- It prepares sessionstream to expose `ss.webSocket.server(hub)` as an object that Express can mount.

### What worked

Targeted validation passed:

```bash
go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http -count=1
```

The tests prove:

- hidden handler refs are extractable by Go but not enumerable from JavaScript;
- plain objects do not satisfy the mountable handler ABI;
- generic handler mounts preserve paths by default;
- `stripPrefix` and `excludePrefixes` work;
- mounted Go handlers are dispatched before JavaScript routes;
- `:params` and wildcard route semantics are explicitly covered.

### What didn't work

- No implementation blockers occurred in this step.
- One small test draft had unused temporary variables while checking hidden keys; I removed them before validation.

### What I learned

- The existing `StaticMount` storage can support generic handler mounts without a larger router rewrite.
- The current host dispatch order is mount-first, route-second. Preserving that behavior keeps the implementation small and predictable.

### What was tricky to build

- The main tricky detail was preserving backward compatibility for static file mounts while changing the default for generic handler mounts. The solution was to make static helpers explicitly opt into `StripPrefix: true` while `RegisterHandler` defaults to preserving the original path.

### What warrants a second pair of eyes

- Whether the public name should remain both `mount` and `mountHandler`, or whether only one should be documented long-term.
- Whether wildcard routes should eventually expose a captured splat value.
- Whether generic handler mounts should eventually be ordered with routes by registration order instead of living in the mount/static dispatch layer.

### What should be done in the future

- Update sessionstream so `ss.webSocket.server(hub)` calls `gojahttp.AttachHTTPHandler`.
- Add a runnable sessionstream Goja smoke app using `app.mount("/ws", ss.webSocket.server(hub))`.

### Code review instructions

- Start with `pkg/gojahttp/mountable.go` and `pkg/gojahttp/host.go`.
- Review `modules/express/express.go` for `app.mount` behavior and option parsing.
- Review `modules/express/express_integration_test.go` for the intended JS composition shape.

### Technical details

Representative JavaScript:

```javascript
const express = require("express");
const app = express.app();

app.mount("/ws", wsServer);
app.mountHandler("/api", apiHandler, { stripPrefix: true });
```
