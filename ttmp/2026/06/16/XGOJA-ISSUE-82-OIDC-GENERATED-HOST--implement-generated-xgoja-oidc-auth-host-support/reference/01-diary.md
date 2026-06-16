---
Title: Diary
Ticket: XGOJA-ISSUE-82-OIDC-GENERATED-HOST
Status: active
Topics:
    - xgoja
    - auth
    - oidc
    - http
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/06/16/XGOJA-AUTH-DEPLOY--deploy-an-xgoja-generated-keycloak-auth-host-to-yolo-scapegoat-dev/reference/01-investigation-diary.md
      Note: 'Prior production deployment evidence that motivated issue #82'
    - Path: ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/design-doc/01-generated-xgoja-oidc-auth-host-design-and-implementation-guide.md
      Note: Primary design guide created in Step 1
ExternalSources: []
Summary: ""
LastUpdated: 2026-06-16T18:01:38.554350979-04:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Capture the investigation and design work for issue #82: replacing the production auth-host demo's hand-written Go host shell with a self-contained generated `xgoja.yaml` + `serve` OIDC host.

## Step 1: Create the issue #82 design package

I created a new docmgr ticket and wrote the primary implementation guide for generated xgoja OIDC auth host support. The guide is intentionally written for a new intern: it explains the vocabulary, current architecture, exact gaps, proposed APIs, runtime flow, migration plan, tests, and production concerns with concrete file references.

The design is based on the production work that pushed example 19 to `goja-auth.yolo.scapegoat.dev`. That deployment proved Keycloak, Vault, Postgres, GHCR, GitOps, Argo, ingress, and smoke testing, but it also confirmed that issue #82 is the blocker preventing a pure generated `xgoja.yaml` host.

### Prompt Context

**User prompt (verbatim):** "Let's create a new ticket to address https://github.com/go-go-golems/go-go-goja/issues/82 in lieu of all the work we did to push our example into prod. We would like to modify the prod demo to be a self contained xgoja.yaml serve example instead of having a go host shell.

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new docmgr ticket for issue #82, analyze the generated-host/OIDC gap in detail, write an intern-ready implementation guide, and upload the resulting docs to reMarkable.

**Inferred user intent:** Convert lessons from the production example-host deployment into a concrete implementation plan for making the production demo a self-contained generated xgoja serve host.

**Commit (code):** N/A — documentation/ticket work only at this step.

### What I did
- Created ticket `XGOJA-ISSUE-82-OIDC-GENERATED-HOST`.
- Added primary design doc `design-doc/01-generated-xgoja-oidc-auth-host-design-and-implementation-guide.md`.
- Added this diary document.
- Read issue #82 through `gh issue view 82`.
- Inspected and referenced code in:
  - `pkg/xgoja/hostauth/*`
  - `pkg/xgoja/providers/http/*`
  - `pkg/gojahttp/auth/keycloakauth/keycloakauth.go`
  - `pkg/gojahttp/auth/appauth/appauth.go`
  - `cmd/xgoja/internal/specv2/types.go`
  - `pkg/xgoja/app/runtime_plan.go`
  - `cmd/xgoja/internal/generate/templates.go`
  - `examples/xgoja/19-express-keycloak-auth-host/*`
  - `examples/xgoja/21-generated-host-auth/*`

### Why
- The production deployment showed that the runtime auth behavior exists and works, but only because example 19 hand-wires `keycloakauth.New` and mounts OIDC endpoints.
- Issue #82 requires promoting that wiring into reusable generated-host infrastructure so the demo can be generated from `xgoja.yaml`.

### What worked
- Issue #82 already had a precise problem statement and acceptance criteria.
- The codebase has a clean host service seam: generated commands can discover `hostauth.ServiceFactoryKey` before command construction, and runtime services can carry `hostauth.Services` into the Express module.
- Example 19 provides a complete reference implementation for OIDC config, redirect URL validation, handler mounting, appauth normalization, stores, and smoke testing.

### What didn't work
- No code implementation was attempted in this step.
- The current architecture still requires a hand-written Go service injection point for hostauth defaults, as seen in example 21. The design identifies this as a schema/generator gap, not merely a hostauth builder gap.

### What I learned
- The desired self-contained `xgoja.yaml` example needs top-level auth config because provider command sets discover host services before runtime module config is evaluated.
- The issue is not just `ResolveConfig`; it spans config schema, generated runtime plan, generated binary templates, hostauth builder, service bundle shape, HTTP native handler mounting, and example/Docker/GitOps migration.

### What was tricky to build
- The main design challenge is the timing of host service discovery. `serve` needs `hostauth.ServiceFactoryKey` while building commands, so auth config cannot live only inside runtime module config.
- Another tricky point is keeping generic generated hostauth safe: the default OIDC normalizer can upsert a user, but it should not silently grant demo admin roles. Demo seed data must remain explicit.

### What warrants a second pair of eyes
- Whether native auth handlers should be mounted directly on `gojahttp.Host` or through an outer HTTP mux/handler service.
- Whether the top-level `auth:` schema should use `hostauth.Config` directly or a schema-local type that converts to hostauth config.
- How much of example 19's invite/capability demo should be preserved in the first generated version.

### What should be done in the future
- Implement the phases in the design doc.
- Upload the ticket docs to reMarkable after docmgr validation.
- When code implementation starts, update the issue #82 checklist as acceptance criteria are completed.

### Code review instructions
- Start with the design doc's Current-state architecture and Proposed architecture sections.
- Review the listed file references before touching code.
- Use the implementation phases as the PR breakdown.

### Technical details
- Ticket: `XGOJA-ISSUE-82-OIDC-GENERATED-HOST`.
- Issue: https://github.com/go-go-golems/go-go-goja/issues/82.
- Main design doc: `design-doc/01-generated-xgoja-oidc-auth-host-design-and-implementation-guide.md`.


## Step 2: Validate and upload the design bundle to reMarkable

I validated the ticket after normalizing its topics to the repository vocabulary and uploaded the design bundle to reMarkable. The uploaded bundle includes the primary implementation guide, this diary, task list, and changelog so the reader can see both the design and the investigation context.

The upload target is the standard ticket-scoped folder under `/ai/2026/06/16/`. The upload command reported success, so no additional cloud listing was needed.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Deliver the issue #82 design package to the ticket workspace and to reMarkable.

**Inferred user intent:** Make the implementation guide easy to review offline and share as a durable research/design deliverable.

**Commit (code):** Pending at time of diary update.

### What I did
- Replaced the non-vocabulary ticket topic `gojahttp` with existing topic `http`.
- Ran:
  ```bash
  docmgr doctor --ticket XGOJA-ISSUE-82-OIDC-GENERATED-HOST --stale-after 30
  ```
- Uploaded a reMarkable bundle containing:
  - `design-doc/01-generated-xgoja-oidc-auth-host-design-and-implementation-guide.md`
  - `reference/01-diary.md`
  - `tasks.md`
  - `changelog.md`

### Why
- The ticket should pass docmgr hygiene before delivery.
- The user explicitly asked to upload the guide to reMarkable.

### What worked
- `docmgr doctor` passed.
- `remarquee upload bundle` succeeded with:
  ```text
  OK: uploaded XGOJA Issue 82 Generated OIDC Host Design.pdf -> /ai/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST
  ```

### What didn't work
- First `docmgr doctor` pass warned about unknown topic `gojahttp`. I fixed it by using the existing `http` topic instead of adding new vocabulary.

### What I learned
- This repo already has `http`, `auth`, `oidc`, `keycloak`, and `xgoja` vocabulary entries, so `gojahttp` was unnecessary as a ticket topic.

### What was tricky to build
- The only delivery wrinkle was topic vocabulary normalization. The documentation itself was already organized correctly, but docmgr treats ticket topics as controlled vocabulary.

### What warrants a second pair of eyes
- Confirm the bundle content is the desired scope for reMarkable. It includes design, diary, tasks, and changelog rather than only the design doc.

### What should be done in the future
- When implementation begins, update issue #82 and this ticket with the selected native handler mounting approach.

### Code review instructions
- Review the uploaded design bundle or the local design doc path.
- Validate locally with `docmgr doctor --ticket XGOJA-ISSUE-82-OIDC-GENERATED-HOST --stale-after 30`.

### Technical details
- reMarkable destination: `/ai/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST`.
- Uploaded PDF name: `XGOJA Issue 82 Generated OIDC Host Design.pdf`.


## Step 3: Clarify hard-cutover HTTP server ownership

I updated the design after the architectural clarification that backwards compatibility is not required. The revised document now says to hard-cut the HTTP provider to a cleaner lifecycle model: the `serve` command owns `net.Listen`, `http.Server`, top-level mux composition, JavaScript route-registration readiness, and graceful shutdown. The Express module should only register routes and must no longer start a listener as a side effect of `require("express")`.

This changes the earlier native-handler mounting recommendation. Instead of adding native route support to `gojahttp.Host` to fit the old Express-starts-server model, the new design mounts native `/auth/*` handlers in a top-level mux owned by `serve`, then mounts the app host at `/`. Existing examples or users that relied on implicit Express startup should be migrated, not preserved with wrappers.

### Prompt Context

**User prompt (verbatim):** "update the document if needed, also we don't need any backwards compatibility and wrappers, we can do a hard cutover, we'll migrate whatever users of the express module we have if needed."

**Assistant interpretation:** Revise the design doc to prefer the clean command-owned HTTP server architecture and explicitly reject compatibility wrappers for old Express side-effect startup.

**Inferred user intent:** Keep issue #82 implementation guidance aligned with the desired long-term architecture rather than an incremental patch to reach OIDC quickly.

**Commit (code):** Pending at time of diary update.

### What I did
- Replaced the current-state discussion of Express-owned server startup with a statement that this is the wrong production boundary.
- Replaced the `gojahttp.Host` native route recommendation with a top-level `serve`-owned mux design.
- Added a decision record: `serve` owns HTTP server lifecycle.
- Rewrote Phase 3 as a hard-cutover refactor:
  - remove Express listener startup;
  - make `serveVerb` build auth/app host/top-level handler/runtime/server in order;
  - keep hot reload listener and native auth handlers stable;
  - migrate existing examples/tests.
- Added an addendum recording the no-backwards-compatibility clarification.

### Why
- OIDC native handlers, health/status routes, hot reload, and graceful shutdown are all server lifecycle concerns.
- The long-term architecture is clearer if the command that keeps the process alive owns the network listener.
- Compatibility wrappers would preserve the confusing old boundary and make future production behavior harder to reason about.

### What worked
- `docmgr doctor --ticket XGOJA-ISSUE-82-OIDC-GENERATED-HOST --stale-after 30` passed after the update.

### What didn't work
- N/A. This was a documentation/design revision only.

### What I learned
- The issue #82 implementation should be treated as an architectural cleanup, not only an OIDC feature. The correct server ownership boundary is a prerequisite for clean generated OIDC support.

### What was tricky to build
- The revised design needed to preserve the earlier evidence while changing the recommendation. The evidence remains valid: Express currently starts the server. The conclusion changed: that behavior should be removed rather than extended.

### What warrants a second pair of eyes
- Review whether all existing examples that rely on implicit Express startup are captured by the migration plan before implementation begins.
- Review hot reload after the server lifecycle refactor, because it currently assumes hosts/snapshots in ways that may need adjustment.

### What should be done in the future
- During implementation, start with the server lifecycle refactor before wiring OIDC handlers.
- Add a regression test that `require("express")` does not bind a port.

### Code review instructions
- In the design doc, review `The current HTTP server ownership is the wrong boundary`, `HTTP server ownership and top-level handler strategy`, the new decision record, and Phase 3.

### Technical details
- Revised reMarkable upload should use a v2 name so it is distinguishable from the first bundle.


### Step 3 delivery update

The revised v2 bundle was uploaded successfully:

```text
OK: uploaded XGOJA Issue 82 Generated OIDC Host Design v2.pdf -> /ai/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST
```


## Step 4: Add implementation task plan and inventory current HTTP ownership

I converted the issue #82 design into an implementation checklist before changing production code. The new tasks start with the HTTP server ownership cutover because generated OIDC support depends on a clear place to mount native `/auth/*` handlers. I also inspected the current provider and Express paths to identify the exact seam that must change.

The inventory confirmed the design problem: `serveVerb` currently creates a runtime and invokes the JS verb, but the actual non-hot-reload listener is still started as an Express module side effect. `newExpressLoader` wires `express.WithOnUse` to `capability.start`, and `capability.start` calls `net.Listen`/`http.Server.Serve`. Hot reload already owns a listener in `serveVerbHotReload`, so the normal serve path is the first target for a hard cutover.

### Prompt Context

**User prompt (verbatim):** "add detailed tasks to the ticket, then work on them step by step, committing at appropriate intervals, keeping a detailed diary as you work"

**Assistant interpretation:** Add a detailed task breakdown to the issue #82 ticket and begin implementation incrementally, committing coherent steps and recording diary/changelog updates.

**Inferred user intent:** Turn the design package into an executable implementation workflow with traceable progress and reviewable commits.

**Commit (code):** Pending at time of diary update.

### What I did
- Checked the previously completed reMarkable upload task.
- Added detailed implementation tasks for inventory, regression tests, server lifecycle refactor, hot reload adaptation, auth config/schema work, generated example work, docs, and validation.
- Inspected:
  - `pkg/xgoja/providers/http/http.go`
  - `pkg/xgoja/providers/http/serve.go`
  - `pkg/xgoja/providers/http/serve_test.go`
  - `modules/express/express.go`
- Confirmed current ownership path:
  - `newExpressLoader` installs `express.WithOnUse`.
  - `express.NewLoader` calls the registrar loader.
  - route-builder use eventually triggers the on-use callback.
  - `capability.start` creates the listener and `http.Server` unless the host is external and `OwnsListen=false`.

### Why
- The implementation should proceed from an explicit task list rather than an ad hoc refactor.
- The server lifecycle change is risky enough to merit a separate inventory checkpoint.

### What worked
- `docmgr task list` now shows concrete implementation tasks.
- The code inventory found a small number of concentrated files for the first cutover.

### What didn't work
- N/A for this planning/inventory step.

### What I learned
- Normal `serveVerb` still depends on Express side-effect listener startup.
- Hot reload already has command-owned listener behavior, but its mux currently mounts only status and manager routes; native auth handler mounting still needs to be designed into that path later.
- `ExternalHostService{OwnsListen:false}` is already used in tests for external host registration without listener ownership.

### What was tricky to build
- The task plan needed to separate two concerns that are currently intertwined: Express route registration and HTTP lifecycle. The inventory clarifies that the first commit can target normal serve ownership without yet completing generated OIDC schema work.

### What warrants a second pair of eyes
- The upcoming hard cutover may break examples that rely on `require("express")` outside the `serve` command. Those migrations should be found and tested before the final implementation bundle.

### What should be done in the future
- Implement a regression test proving `require("express")` no longer starts a listener when invoked through `serve`.
- Refactor normal `serveVerb` before changing hot reload or auth schema.

### Code review instructions
- Start with the task list in `tasks.md`, then review the current listener path in `pkg/xgoja/providers/http/http.go:newExpressLoader` and `capability.start`.

### Technical details
- Current normal serve ownership chain:

```text
serveVerb
  -> NewRuntimeFromSections(... registry.RequireLoader())
  -> InitRuntimeFromSections(... selected modules ...)
  -> registry.InvokeInRuntime(...)
  -> JS require("express")
  -> http capability newExpressLoader
  -> express.NewLoader(... express.WithOnUse(c.start))
  -> capability.start
  -> net.Listen + http.Server.Serve(host)
```


## Step 5: Move normal `serve` onto a command-owned HTTP server

I implemented the first code slice of the hard cutover: the non-hot-reload `serve` path now owns the listener and `http.Server` instead of waiting for Express route registration to start the process. The key behavioral regression test starts a site whose JS verb only calls `require("express")`; under the old behavior no listener existed, while the new command-owned server responds with a normal 404 from the app host.

This is not the full issue #82 implementation yet. It establishes the server lifecycle boundary needed for native OIDC handlers: `serve` creates the app host, injects it into the runtime as an external non-listening host, invokes JS route registration, then starts the server with a top-level mux. Express still has legacy listener-starting code for non-serve uses; the `serve` path no longer depends on it.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Continue executing the ticket tasks incrementally, committing the first coherent server lifecycle refactor slice with tests and diary evidence.

**Inferred user intent:** Make implementation progress while preserving traceability and avoiding one huge unreviewable change.

**Commit (code):** Pending at time of diary update.

### What I did
- Added `TestServeVerbStartsCommandOwnedServerWithoutExpressListen` in `pkg/xgoja/providers/http/serve_test.go`.
- Confirmed the test failed before implementation with:

```text
xgoja http serve: runtime is alive; press Ctrl-C to stop
--- FAIL: TestServeVerbStartsCommandOwnedServerWithoutExpressListen (5.05s)
    serve_test.go:273: timed out waiting for http://127.0.0.1:46233/__no_route_registered status 404
FAIL
FAIL	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http	5.066s
FAIL
```

- Refactored `pkg/xgoja/providers/http/serve.go` normal `serveVerb` to:
  - decode HTTP settings early;
  - require `http.enabled=true`;
  - build auth services when present;
  - create or reuse the app `gojahttp.Host`;
  - inject the app host into the runtime as `ExternalHostService{OwnsListen:false}`;
  - invoke JS route registration before listening;
  - start a command-owned `http.Server` with a top-level mux;
  - shut down gracefully on context cancellation/SIGINT/SIGTERM.
- Added `buildServeHandler` as the top-level mux seam for future native auth handlers.
- Added `serveHTTPServer` to share command-owned server shutdown mechanics.

### Why
- Generated OIDC needs a stable top-level mux where Go-owned `/auth/*` handlers can mount beside the app host.
- The command that keeps the process alive should own listener and shutdown behavior.
- The regression ensures `serve` does not rely on route registration or `app.listen()` to bind a port.

### What worked
- The new regression failed before the refactor and passed after it.
- Targeted tests passed:

```text
go test ./pkg/xgoja/providers/http -run 'TestServeVerb(StartsCommandOwnedServerWithoutExpressListen|LoadsIncludedHelperModulesWithoutHelperCommands|UsesHostAuthServiceFactory|AppliesHostAuthToExternalHost)$' -count=1
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http	0.237s
```

- Full HTTP provider package tests passed:

```text
go test ./pkg/xgoja/providers/http -count=1
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http	0.542s
```

### What didn't work
- The initial regression timed out exactly as expected because the old normal serve path never opened a listener when JS only required Express.
- The first commit attempt failed in the pre-commit lint hook because the refactor left behind the old helper:

```text
pkg/xgoja/providers/http/serve.go:581:6: func waitForServeShutdown is unused (unused)
func waitForServeShutdown(ctx context.Context) error {
     ^
1 issues:
* unused: 1
make: *** [Makefile:33: lint] Error 1
```

I removed `waitForServeShutdown` because `serveHTTPServer` now owns the normal serve wait/shutdown path.

### What I learned
- The current `serve` command can be cut over without removing all legacy Express startup code at once by injecting the host with `OwnsListen:false`.
- Existing hot reload already follows a command-owned listener pattern, so the normal path can reuse similar shutdown structure.

### What was tricky to build
- The runtime service layering can produce duplicate host-service values if a base host already has `HostServiceKey` and the runtime overlay adds another one. The implementation preserves the existing `includeGeneratedHost=false` behavior for externally supplied hosts to avoid `externalHostService` seeing multiple values.
- The listener must start after JS route registration so the command does not accept traffic before planned routes are installed.

### What warrants a second pair of eyes
- Review whether requiring `http.enabled=true` for normal `serve` is acceptable. Hot reload already required it; the old normal path effectively hung without a listener when HTTP was disabled.
- Review the `serveHTTPServer` helper against the existing hot reload shutdown flow; future cleanup could deduplicate hot reload onto the helper too.

### What should be done in the future
- Remove or demote legacy Express listener startup for non-serve use.
- Extend `buildServeHandler` to mount native auth handlers before the app host.
- Adapt hot reload's top-level mux to use the same native auth handler seam.

### Code review instructions
- Start with `pkg/xgoja/providers/http/serve.go:serveVerb` and confirm the order: settings/auth/host/runtime/invoke/listen/shutdown.
- Then read `TestServeVerbStartsCommandOwnedServerWithoutExpressListen` to understand the lifecycle regression.
- Validate with:

```bash
go test ./pkg/xgoja/providers/http -count=1
```

### Technical details
- New normal serve lifecycle:

```text
serveVerb
  -> decode HTTP/auth settings
  -> appHost := gojahttp.NewHost(...)
  -> topHandler := buildServeHandler(appHost, authServices)
  -> runtimeServices include ExternalHostService{Host: appHost, OwnsListen:false}
  -> NewRuntimeFromSectionsWithHostServices(...)
  -> InitRuntimeFromSections(...)
  -> InvokeInRuntime(...)
  -> net.Listen(http.listen)
  -> http.Server{Handler: topHandler}.Serve(listener)
  -> Shutdown on context/signal
```


## Step 6: Make Express route registration pure

I removed the remaining `serve`-time dependency on Express startup hooks. Express route APIs now register routes, static handlers, and mounts into their `gojahttp.Host` without attempting to bind a socket. The HTTP provider no longer passes `express.WithOnUse` into the module loader, and `app.listen()` now returns an explicit error that tells users to use the xgoja `serve` command for server ownership.

This is the hard-cutover behavior requested in the design: no compatibility wrapper keeps the old “route registration starts a listener” behavior alive. Existing users must run through `serve` or own a Go `http.Server` explicitly.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Continue the server lifecycle cleanup by removing legacy Express side-effect listener startup.

**Inferred user intent:** Ensure the implementation matches the clarified architecture: Express is only a route-registration DSL.

**Commit (code):** Pending at time of diary update.

### What I did
- Removed `StartFunc`, `WithOnUse`, and registrar `onUse` state from `modules/express/express.go`.
- Removed `r.start(...)` calls from:
  - `app.mount` / `app.mountHandler`;
  - `app.static`;
  - `app.staticFromAssetsModule`;
  - `app.spaFromAssetsModule`;
  - planned route `.handle(...)` in `modules/express/auth_builders.go`.
- Changed `app.listen()` to return a direct error telling users to use xgoja `serve`.
- Simplified `pkg/xgoja/providers/http/http.go`:
  - stopped passing `express.WithOnUse`;
  - removed `capability.start`;
  - removed per-runtime `http.Server` state from the module capability;
  - kept runtime-entry cleanup so the capability map does not retain closed runtimes.
- Updated `TestExpressRequireDoesNotBindHTTPPort` so route registration is expected not to bind an occupied port.
- Removed the obsolete port-conflict test for `capability.start`.

### Why
- `serve` now owns the process listener; keeping Express startup hooks would leave two competing lifecycle paths.
- OIDC native handlers need one top-level mux owned outside Express.
- A hard cutover is cleaner than a wrapper preserving old side effects.

### What worked
- Targeted validation passed:

```text
go test ./modules/express ./pkg/xgoja/providers/http -count=1
ok  	github.com/go-go-golems/go-go-goja/modules/express	0.040s
ok  	github.com/go-go-golems/go-goja/pkg/xgoja/providers/http	0.538s
```

### What didn't work
- N/A for this step; the code compiled and targeted tests passed after the direct edits.

### What I learned
- The legacy startup path was concentrated in a small API surface: `WithOnUse`, `r.start`, and `capability.start`.
- Normal `serve` can now be reasoned about independently of Express internals: runtime route registration mutates a host, while the command owns serving that host.

### What was tricky to build
- Removing `capability.start` required preserving cleanup for `capability.entries`; otherwise closed runtimes could remain in the capability map. The replacement closer now just deletes the entry.
- `app.listen()` needed an explicit behavior. I chose a clear error instead of a no-op so migrated users get actionable feedback.

### What warrants a second pair of eyes
- Review whether `app.listen()` should be removed from the JS API entirely or kept as this explicit migration error.
- Review external/custom uses of `express.NewLoader(..., opts...)`; `WithName` remains, but `WithOnUse` is gone.

### What should be done in the future
- Update public docs to state that Express never starts a server.
- Search downstream examples before final release for any explicit `app.listen()` usage.

### Code review instructions
- Review `modules/express/express.go` first, especially `appObject` and `app.listen`.
- Review `modules/express/auth_builders.go:needsHandlerObject` for planned-route registration.
- Review `pkg/xgoja/providers/http/http.go:newExpressLoader` and confirm it only creates/reuses a host and installs the loader.
- Validate with:

```bash
go test ./modules/express ./pkg/xgoja/providers/http -count=1
```

### Technical details
- Old path removed:

```text
route registration -> registrar.start -> onUse -> http capability.start -> net.Listen
```

- New path:

```text
route registration -> host.Register*/RegisterPlanned only
serve command -> net.Listen + http.Server
```


## Step 7: Align hot reload with the serve-owned handler seam

I aligned hot reload with the same top-level handler construction and shutdown helper used by normal `serve`. Hot reload already owned one listener and swapped app snapshots through the manager, so this step was intentionally small: it made hot reload use `buildServeHandler` and `serveHTTPServer` instead of building a separate mux/shutdown loop inline.

This keeps the future OIDC mounting point consistent. When `hostauth.Services` grows native auth handlers, both normal serve and hot reload will pass through the same handler-building seam before mounting the app host or hot reload manager at `/`.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Continue the step-by-step implementation by bringing hot reload onto the same top-level server/mux abstraction.

**Inferred user intent:** Avoid splitting normal serve and hot reload into divergent server architectures before adding OIDC handlers.

**Commit (code):** Pending at time of diary update.

### What I did
- Extended `buildServeHandler` to accept optional mux-mount callbacks before the app fallback.
- Changed hot reload status mounting to pass through `buildServeHandler`.
- Changed hot reload serving to reuse `serveHTTPServer` for `Serve`/shutdown behavior.
- Kept the hot reload manager as the `/` fallback handler, preserving the existing snapshot-swap model.

### Why
- Native auth handlers must eventually mount before the app/hot-reload fallback in both serve modes.
- Using one helper reduces the chance that normal serve and hot reload diverge during OIDC integration.

### What worked
- Targeted hot reload and lifecycle tests passed:

```text
go test ./pkg/xgoja/providers/http -run 'TestServeVerbHotReload|TestServeVerbStartsCommandOwnedServerWithoutExpressListen' -count=1
ok  	github.com/go-go-golems/go-goja/pkg/xgoja/providers/http	0.390s
```

- Full HTTP provider package tests passed:

```text
go test ./pkg/xgoja/providers/http -count=1
ok  	github.com/go-go-golems/go-goja/pkg/xgoja/providers/http	0.542s
```

### What didn't work
- N/A. This refactor was narrow and test-covered.

### What I learned
- Hot reload was already much closer to the desired architecture than normal serve. The main gap was that its mux/shutdown logic was separate from the new normal serve seam.

### What was tricky to build
- The status endpoint must be mounted before the `/` fallback manager. The `serveMuxMount` callback keeps that ordering explicit and leaves room for native auth handlers to be inserted before the fallback too.

### What warrants a second pair of eyes
- Review whether `serveHTTPServer(serveCtx, ...)` returning `context canceled` on shutdown remains the desired observable behavior for hot reload tests and callers.

### What should be done in the future
- Add native auth route mounting inside `buildServeHandler` once the hostauth service exposes concrete handlers.
- Consider moving hot reload initial reload/listen ordering into a shared lifecycle helper only if it remains readable.

### Code review instructions
- Review `pkg/xgoja/providers/http/serve.go:buildServeHandler` and the hot reload block around status-path mounting.
- Validate with:

```bash
go test ./pkg/xgoja/providers/http -count=1
```

### Technical details
- Hot reload topology after this step:

```text
one listener/server owned by serveVerbHotReload
  status path -> Go status handler, if configured
  /           -> hotreload.Manager -> current app snapshot
```


## Step 8: Add top-level `auth:` to generated runtime planning

I added the first generated-configuration slice for issue #82: `xgoja/v2` specs can now carry a top-level `auth:` block, the generated runtime plan preserves it, and host construction installs a lazy `hostauth.ServiceFactory` before command providers are built. This is the required plumbing for `xgoja serve` commands to discover auth configuration at command-construction time.

This step deliberately does not build OIDC native handlers yet. It only establishes the schema/runtime-plan/service-factory path so later work can add OIDC-specific fields and handler mounting without putting auth under an Express module config.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Continue the issue #82 checklist by implementing the top-level auth schema/planning path.

**Inferred user intent:** Move from server lifecycle cleanup into generated OIDC configuration support while keeping changes reviewable.

**Commit (code):** Pending at time of diary update.

### What I did
- Added `Auth *hostauth.Config` to `cmd/xgoja/internal/specv2.Config` with `yaml:"auth"` / `json:"auth"` tags.
- Added `Auth *hostauth.Config` to `pkg/xgoja/app.RuntimePlan`.
- Updated `RenderRuntimePlanJSONFromPlan` to copy top-level auth config into embedded runtime plan JSON.
- Updated `app.NewHostWithOptions` to install `hostauth.NewServiceFactory(...)` into host services when `runtimePlan.Auth` is present.
- Added `TestRenderRuntimePlanJSONFromPlanCopiesTopLevelAuth`.
- Added `TestNewHostInstallsRuntimePlanAuthServiceFactory`.

### Why
- Provider command sets discover host services before any JavaScript runtime exists, so auth cannot live only under `runtime.modules[].config`.
- A lazy service factory lets `serve` add auth flags/env-backed defaults during command construction without opening stores until command execution.

### What worked
- Targeted tests passed:

```text
go test ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./cmd/xgoja/internal/specv2 -count=1
ok  	github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate	0.074s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/app	0.303s
ok  	github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2	0.030s
```

### What didn't work
- I initially wrote the new app test with a typo in the import path (`go-goja` instead of `go-go-goja`) and fixed it before running tests.
- The first full pre-commit attempt failed because importing `hostauth` from `app` introduced a test-only import cycle: `hostauth` internal tests imported `app`, and `app` now imported `hostauth`.

```text
package github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth
	imports github.com/go-go-golems/go-go-goja/pkg/xgoja/app from lookup_test.go
	imports github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth from host.go: import cycle not allowed in test
FAIL	github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth [setup failed]
```

I fixed that by moving `pkg/xgoja/hostauth/lookup_test.go` to external `hostauth_test` style and qualifying hostauth symbols.

### What I learned
- `app.NewHostWithOptions` is the right installation point because it is used by generated roots before `AttachDefaultCommands`, and command providers receive `ctx.Host` from that host.
- Keeping `Auth` as a pointer preserves the difference between omitted auth config and an explicit auth block such as `auth: { mode: none }`.

### What was tricky to build
- The runtime plan lives in `pkg/xgoja/app`, while the source spec lives in `cmd/xgoja/internal/specv2`. Both need the same hostauth config shape so generated JSON can round-trip without ad hoc maps.
- Host service installation must happen before optional user `ConfigureServices` so custom hosts can still override the generated factory if needed.

### What warrants a second pair of eyes
- Review whether importing `hostauth` into `specv2` and `app` is acceptable for package boundaries. It keeps one config type, but it does couple the generic runtime plan to hostauth.
- Review whether `ConfigureServices` should override generated auth services as currently ordered.

### What should be done in the future
- Extend `hostauth.Config` with OIDC provider/client/public-base-url/redirect-url fields.
- Add generated example YAML using top-level `auth:`.
- Ensure docs show auth is top-level, not under the Express provider module.

### Code review instructions
- Review `cmd/xgoja/internal/specv2/types.go` and `pkg/xgoja/app/runtime_plan.go` for the new auth field.
- Review `pkg/xgoja/app/host.go:configureRuntimePlanAuthServices` for service-factory installation.
- Review `cmd/xgoja/internal/generate/templates.go:RenderRuntimePlanJSONFromPlan` for JSON propagation.
- Validate with:

```bash
go test ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./cmd/xgoja/internal/specv2 -count=1
```

### Technical details
- New generated auth plumbing:

```text
xgoja.yaml auth:
  -> specv2.Config.Auth
  -> app.RuntimePlan.Auth in embedded xgoja.runtime.json
  -> app.NewHostWithOptions
  -> HostServices[hostauth.ServiceFactoryKey]
  -> http serve command discovers factory and adds auth flags/builds services
```


## Step 9: Build OIDC-capable hostauth services from generated serve config

I added the hostauth configuration and service-building pieces needed for generated `xgoja serve` OIDC mode. The generated top-level `auth:` block can now carry OIDC issuer/client/public-base-url settings, Glazed exposes matching `--auth-oidc-*` fields, resolution derives the callback URL from `public-base-url`, and the service factory builds Keycloak/OIDC handlers into a `NativeHandlers` list.

This step intentionally stops just before mounting those handlers into the HTTP `serve` mux. The service layer now produces the routes; the next step wires them into the command-owned top-level mux before the JavaScript app fallback.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Continue task execution by implementing generated OIDC auth service construction and keeping diary/changelog evidence.

**Inferred user intent:** Turn the design into runnable infrastructure incrementally, with each commit reviewable and tested.

**Commit (code):** Pending at time of diary update.

### What I did
- Added `hostauth.OIDCConfig` under `hostauth.Config.OIDC`.
- Added `ResolvedOIDCConfig` under `ResolvedConfig.OIDC`.
- Added Glazed fields:
  - `auth-oidc-issuer-url`
  - `auth-oidc-client-id`
  - `auth-oidc-client-secret`
  - `auth-oidc-public-base-url`
  - `auth-oidc-redirect-url`
  - `auth-oidc-scopes`
  - `auth-oidc-after-login-url`
  - `auth-oidc-after-logout-url`
- Implemented OIDC config resolution:
  - `public-base-url` derives `<public-base-url>/auth/callback`;
  - explicit `redirect-url` overrides;
  - HTTPS is required except local HTTP when `allow-insecure-http=true`;
  - issuer URL and client ID are required in OIDC mode.
- Added `hostauth.NativeHandler` and `Services.NativeHandlers`.
- Added `BuildNativeHandlers` for `auth.mode=oidc` using `keycloakauth.New`.
- Added `DefaultOIDCUserNormalizer`, which upserts users by OIDC subject and projects existing memberships without granting roles.
- Added tests for OIDC config resolution, Glazed mapping, service native handler construction, and default user normalization.

### Why
- `serve` needs concrete Go-owned auth handlers before it can mount `/auth/login`, `/auth/callback`, and `/auth/logout`.
- OIDC settings must come from generated command/config/env surfaces, not from a hand-written demo host.
- Generic hostauth must not auto-seed tenants or grant demo roles.

### What worked
- Hostauth tests passed:

```text
go test ./pkg/xgoja/hostauth -count=1
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth	0.031s
```

- Cross-package targeted tests passed:

```text
go test ./pkg/xgoja/providers/http ./pkg/xgoja/app ./cmd/xgoja/internal/generate -count=1
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http	0.554s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/app	0.266s
ok  	github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate	0.042s
```

### What didn't work
- N/A in this step after edits; the targeted tests passed.

### What I learned
- `keycloakauth.New` only needs OIDC discovery during construction, so an in-test discovery server is enough to validate service construction without a full login flow.
- The existing appauth stores already provide the right generic primitive: `UpsertFromOIDC` plus `MembershipsForUser`.

### What was tricky to build
- The URL policy had to respect the existing production rule: never derive callback URLs from `--listen`; derive from `public-base-url` or accept explicit `redirect-url`.
- The generic normalizer needed to be useful without encoding the demo's admin seed behavior. It upserts the user and includes existing memberships only.

### What warrants a second pair of eyes
- Review whether `client-secret` should be required for OIDC mode or remain optional for public-client/local experiments.
- Review whether exposing both GET and POST logout handlers is acceptable; POST is the primary route, GET preserves redirect-friendly logout semantics from `keycloakauth`.

### What should be done in the future
- Mount `Services.NativeHandlers` in `buildServeHandler` before `/`.
- Add `/auth/session` if the generated host should expose a first-class session introspection endpoint.
- Add a generated OIDC example and docs after the route wiring lands.

### Code review instructions
- Start with `pkg/xgoja/hostauth/config.go` and `resolve.go` to review the public config and URL validation.
- Then review `pkg/xgoja/hostauth/builder.go:BuildNativeHandlers` and `DefaultOIDCUserNormalizer`.
- Validate with:

```bash
go test ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http ./pkg/xgoja/app ./cmd/xgoja/internal/generate -count=1
```

### Technical details
- Native handlers produced for `auth.mode=oidc`:

```text
GET  /auth/login
GET  /auth/callback
POST /auth/logout
GET  /auth/logout
```


## Step 10: Mount native auth handlers in the serve-owned mux

I wired the OIDC-native handler payload from hostauth into the shared `serve` handler builder. Native handlers are mounted before any extra Go-owned routes and before the JavaScript app or hot-reload fallback at `/`, so `/auth/login`, `/auth/callback`, and `/auth/logout` are handled by Go regardless of what JavaScript routes exist.

This completes the server-side seam required by the hard-cutover architecture: `serve` owns one top-level mux, auth routes are stable Go handlers, and the app host remains the fallback.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Continue the task sequence by wiring native OIDC handlers into the command-owned serve mux.

**Inferred user intent:** Make generated OIDC mode reachable over HTTP instead of only building handler objects.

**Commit (code):** Pending at time of diary update.

### What I did
- Updated `pkg/xgoja/providers/http/serve.go:buildServeHandler` to mount `authServices.NativeHandlers` before the app fallback.
- Added validation for method/path/handler and leading `/` paths.
- Added `muxHandle` to convert `http.ServeMux` duplicate-pattern panics into regular errors.
- Added `TestBuildServeHandlerMountsNativeAuthBeforeAppHost`.

### Why
- OIDC handlers must be mounted outside JavaScript so they remain stable across app reloads and are not affected by route registration order.
- The same handler builder is used by normal serve and hot reload, so this one change covers both server modes.

### What worked
- Targeted tests passed:

```text
go test ./pkg/xgoja/providers/http ./pkg/xgoja/hostauth -count=1
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http	0.557s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth	0.040s
```

### What didn't work
- N/A. The mount slice was direct and test-covered.

### What I learned
- The earlier `buildServeHandler` seam paid off: mounting native auth handlers did not require touching normal vs hot reload lifecycle code separately.

### What was tricky to build
- `http.ServeMux.Handle` panics on duplicate patterns. Since generated config errors should be returned, `muxHandle` recovers and turns those panics into errors with the conflicting pattern.

### What warrants a second pair of eyes
- Review whether native auth handlers should mount before or after hot reload status routes. The current order is auth first, then extra mounts, then `/` fallback.

### What should be done in the future
- Add an end-to-end generated example using `auth.mode=oidc`.
- Add session introspection if needed (`/auth/session`) after deciding the stable response contract.

### Code review instructions
- Review `pkg/xgoja/providers/http/serve.go:buildServeHandler`.
- Review `TestBuildServeHandlerMountsNativeAuthBeforeAppHost` for expected ordering.
- Validate with:

```bash
go test ./pkg/xgoja/providers/http ./pkg/xgoja/hostauth -count=1
```

### Technical details
- Mount order now is:

```text
hostauth native handlers
serve extra Go mounts, e.g. hot reload status
/ fallback -> app host or hotreload.Manager
```
