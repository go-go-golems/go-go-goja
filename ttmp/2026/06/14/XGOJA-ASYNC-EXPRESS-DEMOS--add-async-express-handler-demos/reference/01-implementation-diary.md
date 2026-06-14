---
Title: Implementation diary
Ticket: XGOJA-ASYNC-EXPRESS-DEMOS
Status: active
Topics:
    - express
    - examples
    - auth
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - path: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/18-express-auth-host/cmd/host/main.go
      note: Dev-auth host timer module wiring and smoke assertions (commit 1c48940)
    - path: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/18-express-auth-host/scripts/server.js
      note: Dev-auth async planned route examples (commit 1c48940)
    - path: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
      note: Keycloak host timer module wiring (commit 1c48940)
    - path: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/19-express-keycloak-auth-host/scripts/server.js
      note: Keycloak async planned route examples (commit 1c48940)
    - path: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/19-express-keycloak-auth-host/scripts/keycloak_smoke.py
      note: Keycloak smoke async route assertions (commit 1c48940)
ExternalSources: []
Summary: "Implementation diary for async Express handler demos."
LastUpdated: 2026-06-14T13:53:14.747196536-04:00
WhatFor: "Chronological record of adding async handler demos to runnable Express examples."
WhenToUse: "Use when reviewing or extending async Express handler examples."
---

# Diary

## Goal

Capture the design, implementation, validation, and review notes for adding discoverable runnable async Express handler demos.

## Step 1: Design the async demo scope

I created a focused ticket for async Express demos because async handler support already existed in integration tests but was not visible in runnable examples. The design chooses the existing `timer` module as the deterministic Promise source and targets the dev-auth and Keycloak planned-route examples because both already have smoke coverage.

The design intentionally keeps the routes public. This isolates Promise awaiting from auth/session state while still exercising the same planned dispatch machinery used by protected routes.

### Prompt Context

**User prompt (verbatim):** "Create a new design document to add async demos, then tasks, then implement them."

**Assistant interpretation:** Create docmgr planning artifacts first, then implement runnable async Express handler examples with tasks, validation, diary, and commits.

**Inferred user intent:** Make async Express handler support discoverable and executable, not just test-internal.

**Commit (code):** `6cb5596` — "Docs: plan async Express demos"

### What I did
- Created ticket `XGOJA-ASYNC-EXPRESS-DEMOS`.
- Added design doc `design-doc/01-async-express-handler-demo-plan.md`.
- Added implementation diary `reference/01-implementation-diary.md`.
- Replaced the placeholder task list with six concrete tasks.
- Related existing evidence files: async tests, planned dispatch promise handling, and the `timer` module.
- Marked Task 1 complete and updated the changelog.

### Why
- Tests proved support, but examples did not show how to use `async`/`await` in route handlers.
- A small design doc prevents accidentally creating a broad new example or changing runtime semantics.

### What worked
- `docmgr ticket create-ticket`, `docmgr doc add`, `docmgr doc relate`, and `docmgr task check` created the expected workspace and metadata.
- The existing async tests gave a concrete route shape to copy.

### What didn't work
- N/A.

### What I learned
- Explicit host examples use `WithModules(express.NewRegistrar(host))`, so default modules are not automatically available. Any demo that requires `timer` must opt in explicitly.

### What was tricky to build
- The important constraint was not JavaScript syntax; it was runtime composition. The examples intentionally expose only explicit modules, so adding `require("timer")` to scripts without host wiring would fail at script load time.

### What warrants a second pair of eyes
- Confirm that public async routes are the right documentation tradeoff versus adding a protected async route.

### What should be done in the future
- If users ask for async data fetching examples, add a host-provided fetch/client module separately rather than overloading this Promise-awaiting demo.

### Code review instructions
- Start with `ttmp/2026/06/14/XGOJA-ASYNC-EXPRESS-DEMOS--add-async-express-handler-demos/design-doc/01-async-express-handler-demo-plan.md`.
- Validate that tasks describe implementation rather than broad async runtime changes.

### Technical details
- Existing async tests: `modules/express/express_integration_test.go`.
- Promise awaiting path: `pkg/gojahttp/planned_dispatch.go`.
- Promise source: `modules/timer/timer.go`.

## Step 2: Implement async routes in runnable examples

I added two routes to both runnable auth examples: `/async-return` and `/async-send`. `/async-return` awaits `timer.sleep(5)` and returns a string, while `/async-send` awaits and then calls `res.json(...)`; together they demonstrate the two supported handler completion styles without requiring an HTML renderer.

I initially tried returning a JavaScript object from `/async-return`, which failed in the dev-auth host because non-string returned values go through the host renderer and that example does not configure one. I changed the returned-value demo to return a string and kept structured JSON in the `res.json(...)` demo.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the planned async demos and validate them in the existing runnable example smokes.

**Inferred user intent:** Provide copy/pasteable and smoke-tested proof that Express handlers can be `async`.

**Commit (code):** `1c48940` — "Add async Express handler demos"

### What I did
- Added `UseModuleMiddleware(engine.MiddlewareOnly("timer"))` to:
  - `examples/xgoja/18-express-auth-host/cmd/host/main.go`
  - `examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go`
- Added `const timer = require("timer")` and public async routes to:
  - `examples/xgoja/18-express-auth-host/scripts/server.js`
  - `examples/xgoja/19-express-keycloak-auth-host/scripts/server.js`
- Extended the dev-auth in-process smoke with `/async-return?name=dev` and `/async-send?name=dev` checks.
- Extended the Keycloak Python smoke with `/async-return?name=keycloak` and `/async-send?name=keycloak` checks.
- Updated both READMEs with async handler snippets, manual `curl` commands, and caveats.
- Adjusted the design doc after the returned-object renderer failure: the returned-value demo is now a string-return demo.

### Why
- The demos need to show both promise settlement paths: returned fulfillment values and response writes after `await`.
- The examples need host runtime wiring because `timer` is not available when only Express is explicitly registered.

### What worked
- `timer.sleep(5)` gave deterministic asynchronous settlement.
- Both planned-route examples correctly awaited promises before completing the HTTP response.
- Dev-auth smoke passed with async routes included.
- Keycloak/Postgres smoke passed with async checks included before login.

### What didn't work
- First attempt returned an object from `/async-return`:
  ```text
  JavaScript handler error: no HTML renderer configured
  ```
  Command that exposed it:
  ```bash
  make -C examples/xgoja/18-express-auth-host smoke
  ```
  Cause: `finishHandlerResult` sends strings directly but routes non-string return values through `res.HTML(...)`; the dev-auth host has no renderer configured.

### What I learned
- Async returned values follow the same rendering rules as synchronous returned values. `async` changes timing, not response serialization.
- For examples without a renderer, return strings or call `res.json(...)` for structured payloads.

### What was tricky to build
- The sharp edge was distinguishing "Promise awaiting works" from "arbitrary returned objects become JSON". The latter is not true for this host: structured responses should use `res.json(...)`. The solution was to split the examples cleanly: string return for returned-value awaiting, JSON send for response-sending awaiting.
- Runtime module selection also mattered. Because the examples use explicit `WithModules(...)`, the host needed `UseModuleMiddleware(engine.MiddlewareOnly("timer"))` to make `require("timer")` available while keeping the demo's module surface narrow.

### What warrants a second pair of eyes
- Review whether the README wording is sufficiently clear that non-string returned values require a renderer and are not automatically JSON.
- Review whether `MiddlewareOnly("timer")` plus explicit Express registration is the preferred way to expose a single default-registry module in examples.

### What should be done in the future
- Consider adding a protected async route only if users need to see async handlers combined with actor/resource data.
- Consider a separate demo for host-provided async I/O once there is a standard fetch/client module.

### Code review instructions
- Start with the JavaScript scripts:
  - `examples/xgoja/18-express-auth-host/scripts/server.js`
  - `examples/xgoja/19-express-keycloak-auth-host/scripts/server.js`
- Then review host runtime composition:
  - `examples/xgoja/18-express-auth-host/cmd/host/main.go`
  - `examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go`
- Validate with:
  ```bash
  python3 -m py_compile examples/xgoja/19-express-keycloak-auth-host/scripts/keycloak_smoke.py
  go test ./examples/xgoja/18-express-auth-host/cmd/host ./examples/xgoja/19-express-keycloak-auth-host/cmd/host -count=1
  make -C examples/xgoja/18-express-auth-host smoke
  make -C examples/xgoja/19-express-keycloak-auth-host smoke
  ```

### Technical details
- Validation passed:
  ```bash
  python3 -m py_compile examples/xgoja/19-express-keycloak-auth-host/scripts/keycloak_smoke.py
  go test ./examples/xgoja/18-express-auth-host/cmd/host ./examples/xgoja/19-express-keycloak-auth-host/cmd/host -count=1
  make -C examples/xgoja/18-express-auth-host smoke
  make -C examples/xgoja/19-express-keycloak-auth-host smoke
  ```
- Pre-commit for `1c48940` also passed lint, Glazed vet, `go generate ./...`, and `go test ./...`.
