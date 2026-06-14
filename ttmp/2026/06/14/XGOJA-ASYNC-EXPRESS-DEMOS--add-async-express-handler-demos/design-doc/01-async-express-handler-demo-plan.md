---
Title: Async Express handler demo plan
Ticket: XGOJA-ASYNC-EXPRESS-DEMOS
Status: active
Topics:
    - express
    - examples
    - auth
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: modules/express/express_integration_test.go
      Note: Existing async handler tests that shape demo behavior
    - Path: modules/timer/timer.go
      Note: Promise-based sleep helper used by demos
    - Path: pkg/gojahttp/planned_dispatch.go
      Note: Planned route promise-awaiting implementation
ExternalSources: []
Summary: Plan for adding runnable async handler examples to xgoja Express demos.
LastUpdated: 2026-06-14T13:53:14.39465507-04:00
WhatFor: Use when updating Express examples to demonstrate async handlers and Promise awaiting.
WhenToUse: When a user asks whether async Express route handlers are supported or wants runnable examples beyond tests.
---


# Async Express handler demo plan

## Executive Summary

`gojahttp` already supports async Express handlers: both raw and planned dispatch detect returned `*goja.Promise` values, wait for settlement, render fulfilled return values, and treat rejections as handler errors. The current coverage lives primarily in `modules/express/express_integration_test.go`, which is useful for correctness but not very discoverable to users trying examples.

This design adds small async routes to the runnable planned-auth examples. The demos should show both common patterns:

1. an async handler that returns a value after `await`, and
2. an async handler that sends the response itself after `await`.

The examples must stay focused on Express planned routes, not on a new async abstraction. They should reuse the existing `timer` module as the Promise source because it is deterministic, host-provided, and already covered by tests.

## Problem Statement

Users can ask whether `async`/`await` works in Express handlers, but the answer currently requires knowing where to look in integration tests. That leaves a documentation gap:

- standalone examples do not advertise async handler support,
- auth examples do not show that planned routes also await promises,
- smokes do not prove that runnable examples remain configured with any modules required by async snippets,
- README snippets do not explain what happens for returned values versus `res.json(...)` inside async functions.

## Proposed Solution

Add async routes to the dev-auth Express host example (`examples/xgoja/18-express-auth-host`) and the Keycloak host example (`examples/xgoja/19-express-keycloak-auth-host`). These are the best runnable targets because they already have smokes and host-owned auth wiring.

### Route shape

Use the `timer` module in JavaScript:

```js
const timer = require("timer")

app.get("/async-return")
  .public()
  .audit("async.returned")
  .handle(async (ctx, res) => {
    await timer.sleep(5)
    return `async return ${ctx.request.query.name || "anonymous"}`
  })

app.get("/async-send")
  .public()
  .audit("async.sent")
  .handle(async (ctx, res) => {
    await timer.sleep(5)
    res.json({ ok: true, mode: "send", name: ctx.request.query.name || "anonymous" })
  })
```

### Host wiring

The existing host examples build runtimes with explicit modules:

```go
engine.NewRuntimeFactoryBuilder().WithModules(express.NewRegistrar(host)).Build()
```

Because explicit `WithModules(...)` does not automatically append default registry modules, the host must opt into `timer` with module middleware:

```go
engine.NewRuntimeFactoryBuilder().
    UseModuleMiddleware(engine.MiddlewareOnly("timer")).
    WithModules(express.NewRegistrar(host)).
    Build()
```

This keeps the demo least-privilege: it exposes Express plus the one extra module required by the async demo.

### Smoke coverage

Extend existing smokes to request:

- `GET /async-return?name=demo`, expecting `async return demo`.
- `GET /async-send?name=demo`, expecting `"mode":"send"` and `"name":"demo"`.

For the Keycloak smoke, the routes can be public so the checks can run before or after login without adding browser automation complexity. The goal is Promise awaiting, not auth policy.

### Documentation

Update READMEs to call out that:

- handlers may be `async`,
- returned string promise fulfillment values are sent like synchronous string returns,
- handlers may call `res.json(...)` after `await` for structured payloads,
- rejected promises become handler errors.

## Design Decisions

### Decision 1: Use `timer.sleep` rather than external HTTP/fetch

Status: accepted.

The `timer` module gives the examples a Promise without depending on a network service or browser-like `fetch` API. This makes the smoke deterministic and keeps the demo focused on Express handler semantics.

### Decision 2: Add demos to runnable auth host examples

Status: accepted.

`examples/xgoja/18-express-auth-host` and `examples/xgoja/19-express-keycloak-auth-host` already exercise planned routes through real host HTTP dispatch. Adding async routes there proves the same machinery works in the auth-oriented examples users are likely to run.

### Decision 3: Keep async routes public

Status: accepted.

Public async routes isolate Promise handling from session/login state. Authenticated async route coverage already follows from the same planned dispatch path; a future example can add protected async routes if there is a documentation need.

## Alternatives Considered

### Add only README snippets

Rejected because snippets alone would not verify runtime module wiring or smoke behavior.

### Add a new standalone `20-express-async` example

Deferred. A new example would be clean, but it would also add another host, Makefile, README, and smoke. Adding routes to existing examples is smaller and keeps auth docs cohesive.

### Use `Promise.resolve(...)` only

Rejected as the primary demo because it does not show actual asynchronous settlement. `timer.sleep(...)` proves the request waits across an event-loop turn.

## Implementation Plan

1. Create this design doc and task list under `XGOJA-ASYNC-EXPRESS-DEMOS`.
2. Add `timer` to the runtime factory for the dev-auth and Keycloak examples using `UseModuleMiddleware(engine.MiddlewareOnly("timer"))`.
3. Add `/async-return` and `/async-send` routes to both example `scripts/server.js` files.
4. Extend dev-auth and Keycloak smokes to assert both async routes.
5. Update example READMEs with async handler snippets and caveats.
6. Run targeted tests and smokes.
7. Update the implementation diary, relate files, update changelog, and commit.

## Open Questions

None for this scope.

## References

- `modules/express/express_integration_test.go` — existing async handler tests.
- `pkg/gojahttp/host.go` — raw handler promise awaiting.
- `pkg/gojahttp/planned_dispatch.go` — planned route promise awaiting.
- `modules/timer/timer.go` — Promise-based `timer.sleep(ms)` helper.
- `examples/xgoja/18-express-auth-host` — dev-auth runnable planned-route host.
- `examples/xgoja/19-express-keycloak-auth-host` — Keycloak runnable planned-route host.
