---
Title: Implementation Diary
Ticket: XGOJA-GO-AUTH-API-DESIGN
Status: active
Topics:
    - goja
    - xgoja
    - auth
    - security
    - architecture
    - rest-api
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/design/01-go-native-planned-auth-api-intern-implementation-guide.md
      Note: Main intern-oriented implementation guide.
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/sources/01-current-gojahttp-auth-surface.md
      Note: Current code evidence used by the guide.
ExternalSources: []
Summary: Diary for creating the Go-native planned auth API design ticket and uploading it to reMarkable.
LastUpdated: 2026-06-15T22:18:00-04:00
WhatFor: Preserve context for the Go-side planned auth API design work.
WhenToUse: Read before implementing SecureContext, RegisterPlannedHTTP, Go fluent route builders, middleware, or hostauth.AuthKit.
---

# Diary

## Goal

Capture the creation of a new ticket and intern-oriented guide for making the planned auth framework usable from traditional Go hosts, not only JavaScript Goja Express routes.

## Step 1: Create the Go-native planned auth API guide

This step turns the design discussion into a concrete implementation guide. The guide explains the current route-plan/auth system, identifies the coupling to Goja handlers, and proposes a Go-native API built around `SecureContext`, `RegisterPlannedHTTP`, a Go fluent builder, standard `net/http` middleware, and a reusable `hostauth.AuthKit`.

The intended reader is a new intern. The document therefore emphasizes concepts, current file references, sequence diagrams, pseudocode, staged implementation phases, and tests rather than only listing API signatures.

### Prompt Context

**User prompt (verbatim):** "Ok, create a new ticket and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.
 for building this go side API."

**Assistant interpretation:** Create a separate docmgr ticket, write a clear intern-oriented technical implementation guide for building a Go-side planned auth API, include diagrams/pseudocode/API and file references, store the guide in the ticket, and upload it to reMarkable.

**Inferred user intent:** Provide an implementation-ready onboarding document so an intern can understand the current system and safely add Go-native APIs that reuse the existing planned auth framework.

**Commit (code):** N/A — ticket documentation only.

### What I did

- Created ticket `XGOJA-GO-AUTH-API-DESIGN`.
- Added design doc `design/01-go-native-planned-auth-api-intern-implementation-guide.md`.
- Added diary doc `reference/01-implementation-diary.md`.
- Added evidence doc `sources/01-current-gojahttp-auth-surface.md` with line anchors from current product code.
- Wrote the implementation guide in an explanatory style with prose paragraphs, diagrams, tables, API sketches, pseudocode, phased tasks, and tests.

### Why

- Traditional Go programs can already assemble `gojahttp.HostOptions.Auth`, but they do not yet get the same ergonomic planned-route enforcement path as JavaScript routes.
- The central design is to make planned auth a `gojahttp` framework feature and make JavaScript Express just one frontend to `RoutePlan`.

### What worked

- The current code already has the important reusable pieces: `RoutePlan`, `AuthOptions`, `sessionauth`, `appauth`, `audit`, `hostauth`, and `gojahttp.Host`.
- The design can be implemented incrementally by extracting `SecureContext` before adding new public APIs.

### What didn't work

- N/A.

### What I learned

- The key missing abstraction is a reusable secure context/enforcer that is independent of JavaScript object construction.
- `hostauth` should be repositioned as a general Go auth kit while preserving generated-host service factory behavior.

### What was tricky to build

- The guide had to preserve the existing JavaScript API and explain that Go and JS need identical semantics, not identical syntax.
- The route registry change needs special care because it currently stores a single `goja.Callable` handler shape.

### What warrants a second pair of eyes

- Whether the route registry should use separate handler fields, an interface, or separate route tables.
- Whether `SecureContext` should be exported immediately or introduced as internal first.
- Whether `hostauth.AuthKit` belongs in `pkg/xgoja/hostauth` or a more neutral package later.

### What should be done in the future

- Implement the guide in phases:
  1. Extract `SecureContext`.
  2. Add `RegisterPlannedHTTP`.
  3. Add Go fluent builder.
  4. Add planned middleware.
  5. Add `hostauth.NewKit`.
  6. Add examples/docs.

### Code review instructions

- Start with `pkg/gojahttp/planned_dispatch.go` and verify the extraction preserves current JS planned-route behavior.
- Review route registry changes carefully so raw Goja routes, planned Goja routes, and planned Go routes dispatch predictably.
- Validate with tests for public, authenticated, CSRF-denied, resource-denied, authorization-denied, handler-failed, and mixed Go/JS hosts.

### Technical details

Primary files created:

```text
ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/design/01-go-native-planned-auth-api-intern-implementation-guide.md
ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/reference/01-implementation-diary.md
ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/sources/01-current-gojahttp-auth-surface.md
```

## Step 2: Extract SecureContext and add direct planned Go handlers

This step implements the first executable slice of the Go-native planned auth API. The goal was not to add the full fluent Go builder yet; it was to make `gojahttp.Host` capable of serving a protected Go handler through the same route-plan enforcement path used by planned Goja routes.

The change adds a public `SecureContext`, introduces route kinds, adds `RegisterPlannedHTTP`, and adds a planned HTTP dispatch path. Existing planned Goja routes still pass through the same tests, while new Go handler tests prove that a Go-only host does not need a Goja runtime for planned Go routes.

### Prompt Context

**User prompt (verbatim):** "Go ahead, commit at appropriate intervals, keep a detailed diary as you work."

**Assistant interpretation:** Start implementing the Go-side planned auth API from the design, commit focused slices, and keep this diary current with what changed and how it was validated.

**Inferred user intent:** Move from design to implementation while preserving reviewability and continuation context.

**Commit (code):** pending at time of diary update — planned message `Add Go planned HTTP routes`.

### What I did

- Added `SecureContext` as the exported Go-native result of planned-route enforcement.
- Kept `secureEnvelope` as the JavaScript adapter over `SecureContext` so planned Goja behavior remains stable.
- Added `PlannedHTTPHandler` for native Go planned handlers.
- Added `RouteKind` values for raw Goja, planned Goja, and planned HTTP routes.
- Extended the registry with `AddPlannedHTTP`.
- Added `Host.RegisterPlannedHTTP`.
- Changed `Host.ServeHTTP` so a Go-only planned HTTP route does not require `h.owner` / a Goja runtime.
- Added `servePlannedHTTP` with the same allowed/denied/completed/failed audit semantics as planned Goja routes.
- Added tests in `pkg/gojahttp/planned_http_test.go` for public Go handlers, authenticated Go handlers, CSRF denial, and handler failure audit.

### Why

- `RoutePlan` is the shared security contract. Go handlers need a first-class registration path that consumes the same contract instead of manually calling authenticator, CSRF, resolver, authorizer, and audit services.
- Splitting handler backends by route kind is the smallest route-registry change that lets existing Goja routes and new Go handlers coexist on one host.

### What worked

- Existing planned Goja tests still pass after the route-kind change.
- New planned HTTP tests pass and demonstrate that Go planned routes do not need a runtime owner.
- Focused validation passed:

```bash
GOFLAGS=-buildvcs=false go test ./pkg/gojahttp ./modules/express -count=1
```

### What didn't work

- The first test run failed because `RouteDescriptor` exact-struct tests expected zero values, and adding `Kind: "raw-goja"` / `Kind: "planned-goja"` changed those comparisons:

```text
--- FAIL: TestRegistryRoutesReturnsCopySafeDescriptors
    route_registry_test.go:38: first route = gojahttp.RouteDescriptor{Method:"GET", Pattern:"/cards/:id", Kind:"raw-goja", Planned:false, ...}
--- FAIL: TestHostRoutesDelegatesToRegistry
--- FAIL: TestRegistryRoutesIncludesPlannedMetadata
```

- I fixed this by only exposing `RouteDescriptor.Kind` for the new `planned-http` backend. That preserves existing descriptor compatibility while still making the new handler backend visible.

### What I learned

- `Host.ServeHTTP` previously checked `h.owner == nil` before route matching. That made sense when every dynamic route required Goja, but it prevented Go-only planned routes from working. The owner check now belongs only to Goja-backed route kinds.
- The registry can remain simple with separate handler fields for now. A polymorphic handler interface is not necessary for the first slice.

### What was tricky to build

- The main ordering constraint was preserving raw-route rejection and session DTO construction while moving the runtime-owner check later. Route matching must happen before deciding whether a runtime is required.
- Planned HTTP error handling needs to avoid overwriting responses already written by the handler. The first implementation uses the existing access-log response writer's `wroteHeader` state.

### What warrants a second pair of eyes

- Whether `RouteDescriptor.Kind` should eventually expose all route kinds, with tests updated intentionally, instead of only exposing `planned-http` for compatibility.
- Whether `PlannedHandlerFunc` is useful or should be removed before the API settles.
- Whether `SecureContext.Body` should be deep-copied before Go handlers receive it, or documented as request-owned data.

### What should be done in the future

- Add the fluent Go builder on top of `RegisterPlannedHTTP`.
- Consider moving shared enforcement out of `planned_dispatch.go` into an `Enforcer` type before adding standard `net/http` middleware.
- Add mixed Go + Goja route integration tests once the builder exists.

### Code review instructions

- Start with `pkg/gojahttp/route_registry.go` to review the route kind and handler storage model.
- Then review `pkg/gojahttp/host.go` to verify runtime-owner checks apply only to Goja-backed routes.
- Then review `pkg/gojahttp/planned_http.go` and compare audit/error behavior against `servePlannedRoute` in `planned_dispatch.go`.
- Validate with `GOFLAGS=-buildvcs=false go test ./pkg/gojahttp ./modules/express -count=1`.

### Technical details

Files changed in this step:

```text
pkg/gojahttp/route_registry.go
pkg/gojahttp/host.go
pkg/gojahttp/planned_dispatch.go
pkg/gojahttp/planned_http.go
pkg/gojahttp/planned_http_test.go
ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/reference/01-implementation-diary.md
```

## Step 3: Add the Go fluent planned-route builder

This step adds the ergonomic API that traditional Go callers should use most often. `RegisterPlannedHTTP` is the low-level primitive; `gojahttp.NewApp(host)` is the human-facing builder that mirrors the JavaScript planned-route chain while staying idiomatic enough for Go.

The builder produces the same `RoutePlan` as JavaScript routes and then delegates to `RegisterPlannedHTTP`. It does not contain a second enforcement path. That is the important design property: JavaScript routes and Go routes differ in declaration syntax and handler backend, not in authentication semantics.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue implementing the Go-side planned auth API in focused commits, updating the diary and tests at each step.

**Inferred user intent:** Build the planned API incrementally without losing reviewability.

**Commit (code):** pending at time of diary update — planned message `Add Go planned route builder`.

### What I did

- Added `pkg/gojahttp/app.go`.
- Added `gojahttp.NewApp(host)` and method helpers: `Get`, `Post`, `Put`, `Patch`, `Delete`, `All`, and `Route`.
- Added staged builder types:
  - `RouteNeedsSecurity`
  - `RouteNeedsPolicy`
  - `RouteNeedsHandler`
- Added Go auth/resource spec builders:
  - `gojahttp.User().Required()`
  - `gojahttp.User().MFAFresh(duration)`
  - `gojahttp.Resource(type).IDFromParam(...).TenantFromParam(...).MustExist()`
- Added `Handle` for native planned handlers.
- Added `HandleJSON` for common JSON route responses.
- Added tests in `pkg/gojahttp/app_test.go` for public JSON routes, authenticated resource/CSRF/action routes, and builder validation errors.

### Why

- Direct `RoutePlan` construction is useful for generated code, but it is verbose for application authors.
- The Go builder gives traditional Go hosts the same mental model as Express planned routes: declare security first, declare policy, then attach a handler.

### What worked

- The builder composes cleanly over `RegisterPlannedHTTP`.
- `HandleJSON` gives a concise happy path without hiding the lower-level `http.ResponseWriter` API.
- Focused validation passed:

```bash
GOFLAGS=-buildvcs=false go test ./pkg/gojahttp ./modules/express -count=1
```

### What didn't work

- My first validation-error test attempted to call `.Handle(...)` directly after `.Auth(...)`. That failed at compile time because the staged Go API correctly returns `*RouteNeedsPolicy`, which has no `Handle` method:

```text
pkg/gojahttp/app_test.go:85:73: app.Get("/me").Auth(...).Handle undefined (type *gojahttp.RouteNeedsPolicy has no field or method Handle)
```

- I changed the test to call `.Allow("").Handle(...)`, which reaches the handler stage but still fails `ValidateRoutePlan` because the action is empty.

### What I learned

- Even without generics, separate staged builder types catch some incorrect usage at compile time.
- The remaining validation still belongs in `ValidateRoutePlan`; the builder should not duplicate all plan validation rules.

### What was tricky to build

- The resource builder needs to support both fluent route declarations and final `ResourceSpec` values. I used value receivers so each builder method returns a modified copy, and `MustExist()` / `Spec()` produce the final `ResourceSpec`.
- `HandleJSON` must return encoder errors so handler failure audit still works through the planned HTTP dispatch path.

### What warrants a second pair of eyes

- Whether `ResourceBuilder.MustExist()` should return `ResourceBuilder` instead of `ResourceSpec` for a more uniform builder API. The current shape optimizes for concise route declarations.
- Whether `HandleJSON` should support status-code helpers now or wait for a future response abstraction.
- Whether `gojahttp.User()` and `gojahttp.Resource()` are the right package-level names, or whether they should live under a sub-builder namespace to avoid future name collisions.

### What should be done in the future

- Add standard `net/http` planned middleware on top of the same enforcement path.
- Add docs/examples for the Go builder once the API settles.
- Consider adding `Name(...)` to later builder stages as a convenience if route authors want to name routes after declaring auth.

### Code review instructions

- Start with `pkg/gojahttp/app.go` and verify the builder only accumulates `RoutePlan` data and delegates to `RegisterPlannedHTTP`.
- Review `pkg/gojahttp/app_test.go` to see intended usage.
- Validate with `GOFLAGS=-buildvcs=false go test ./pkg/gojahttp ./modules/express -count=1`.

### Technical details

Files changed in this step:

```text
pkg/gojahttp/app.go
pkg/gojahttp/app_test.go
ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/reference/01-implementation-diary.md
```
