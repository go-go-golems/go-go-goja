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
    - Path: cmd/xgoja/doc/18-go-planned-auth-api.md
      Note: |-
        User-facing embedded help guide for the Go planned auth API
        Updated public documentation for Enforcer usage
    - Path: examples/gojahttp/01-planned-auth/Makefile
      Note: Smoke and serve entrypoints for the runnable example
    - Path: examples/gojahttp/01-planned-auth/README.md
      Note: Example usage notes and curl commands
    - Path: examples/gojahttp/01-planned-auth/main.go
      Note: Runnable Go-only planned auth example covering fluent host routes and middleware
    - Path: pkg/gojahttp/enforcer.go
      Note: Reusable planned-auth enforcement pipeline extracted for hosts
    - Path: pkg/gojahttp/middleware.go
      Note: Middleware now uses Enforcer instead of a hidden Host
    - Path: ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/design/01-go-native-planned-auth-api-intern-implementation-guide.md
      Note: Main intern-oriented implementation guide.
    - Path: ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/sources/01-current-gojahttp-auth-surface.md
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

## Step 4: Add planned auth middleware for standard net/http routers

This step adds a bridge for Go programs that do not want to route requests through `gojahttp.Host`. The new middleware lets a standard `http.ServeMux` route use the same `RoutePlan` enforcement path and the same `PlannedHTTPHandler` signature.

The middleware is deliberately small. It validates the route plan, constructs request/session DTOs, extracts params either from the gojahttp `:param` matcher or from a caller-provided `ParamFunc`, and then delegates to the same planned HTTP dispatch path used by `RegisterPlannedHTTP`.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue the implementation in focused slices and commit after each validated increment.

**Inferred user intent:** Make the framework useful to multiple Go host styles, not only hosts that use `gojahttp.Host` as the primary router.

**Commit (code):** pending at time of diary update — planned message `Add planned auth middleware`.

### What I did

- Added `pkg/gojahttp/middleware.go`.
- Added `MiddlewareOptions` with `Auth`, `Sessions`, `Dev`, and `ParamFunc`.
- Added `PlannedMiddleware(opts, plan, next)`.
- Added fallback parameter extraction using the existing `:param` matcher.
- Added `ParamFunc` support for Go 1.22 `http.ServeMux` path values.
- Added tests in `pkg/gojahttp/middleware_test.go` for standard mux path values and method rejection.

### Why

- Some Go applications already have a router and should not have to replace it with `gojahttp.Host` just to use planned auth.
- The middleware keeps `RoutePlan` as the shared contract while letting callers integrate with standard `net/http` routing.

### What worked

- The middleware test uses `http.NewServeMux()` and `r.PathValue(name)` to map `{orgID}` / `{projectID}` into a `RoutePlan` that uses `:orgID` / `:projectID` resource sources.
- Focused validation passed:

```bash
GOFLAGS=-buildvcs=false go test ./pkg/gojahttp ./modules/express -count=1
```

### What didn't work

- The first commit attempt failed in the pre-commit lint hook because staticcheck preferred De Morgan form for method validation:

```text
pkg/gojahttp/middleware.go:27:57: QF1001: could apply De Morgan's law (staticcheck)
```

- I rewrote the condition from `!(r.Method == http.MethodHead && plan.Method == http.MethodGet)` to `(r.Method != http.MethodHead || plan.Method != http.MethodGet)` and reran focused validation before retrying the commit.

### What I learned

- `ParamFunc` is enough for the first middleware version. It avoids forcing `gojahttp` route-pattern syntax onto users of standard mux while still preserving `RoutePlan` validation.
- Reusing `servePlannedHTTP` keeps error and audit behavior aligned with host-native planned routes.

### What was tricky to build

- The middleware cannot rely on the host's normal `ServeHTTP` wrapping, so it creates an access-log response writer only to track whether the planned handler already wrote a response. It does not emit a second access log.
- Method validation must handle `ALL` and GET/HEAD compatibility without interfering with the caller's router.

### What warrants a second pair of eyes

- Whether `PlannedMiddleware` should be implemented on top of an exported `Enforcer` type before more integrations are added.
- Whether `ParamFunc` should return `(string, bool)` instead of an empty string for missing params to support empty literal-like values. Route params should probably stay non-empty.

### What should be done in the future

- Extract `Enforcer` if host, middleware, or tests start duplicating more request setup.
- Add documentation showing standard mux usage beside `gojahttp.NewApp(host)` usage.

### Code review instructions

- Review `pkg/gojahttp/middleware.go` to confirm it delegates to `servePlannedHTTP` rather than reimplementing enforcement.
- Review `pkg/gojahttp/middleware_test.go` for the expected Go 1.22 mux integration pattern.
- Validate with `GOFLAGS=-buildvcs=false go test ./pkg/gojahttp ./modules/express -count=1`.

### Technical details

Files changed in this step:

```text
pkg/gojahttp/middleware.go
pkg/gojahttp/middleware_test.go
ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/reference/01-implementation-diary.md
```

## Step 5: Document the Go planned auth API

This step turns the newly added Go route APIs into a user-facing help page. The implementation now has three entry points, so the documentation focuses on choosing the right one: fluent `NewApp(host)` for hand-written Go routes, `RegisterPlannedHTTP` for generated or low-level routes, and `PlannedMiddleware` for existing `net/http` routers.

The docs also restate the most important security boundary: route declarations describe intent, but Go-owned services still perform authentication, authorization, resource resolution, CSRF, and audit.

### Prompt Context

**User prompt (verbatim):** "add docs. 

then explain concisely what a reusable Enforcer is ."

**Assistant interpretation:** Add user-facing documentation for the Go-native planned auth API, then briefly explain the future reusable Enforcer idea.

**Inferred user intent:** Make the new API understandable to future callers and clarify the design follow-up before deciding whether to implement it.

**Commit (code):** pending at time of diary update — planned message `Docs: add Go planned auth API guide`.

### What I did

- Added `cmd/xgoja/doc/18-go-planned-auth-api.md`.
- Documented the core types: `RoutePlan`, `SecureContext`, `PlannedHTTPHandler`, `RegisterPlannedHTTP`, `NewApp`, and `PlannedMiddleware`.
- Added examples for:
  - host-owned auth service configuration,
  - fluent Go routes,
  - direct `RegisterPlannedHTTP`,
  - standard Go 1.22 `http.ServeMux` middleware integration.
- Ran focused validation:

```bash
GOFLAGS=-buildvcs=false go test ./cmd/xgoja ./cmd/xgoja/doc ./pkg/gojahttp ./modules/express -count=1
```

### Why

- The Go API is no longer just an internal primitive; it needs discoverable documentation in the embedded xgoja help docs.
- The docs reduce ambiguity about which API shape to use and where security decisions belong.

### What worked

- The new doc lives under `cmd/xgoja/doc`, which is already embedded by `cmd/xgoja/doc/doc.go` using `//go:embed *.md`.
- Focused validation passed.

### What didn't work

- N/A.

### What I learned

- The best documentation structure is “choose the API by host shape” rather than “list every type in implementation order”. That matches how adopters will approach the feature.

### What was tricky to build

- The examples need to be concrete enough to teach route planning, but not imply that docs should configure real secrets, stores, or policy inline. I kept auth services as host-provided variables and emphasized the boundary in prose.

### What warrants a second pair of eyes

- Whether this belongs only in xgoja help docs or should also be copied into package-level Go docs once the API settles.
- Whether examples should use a concrete appauth/sqlstore setup later, after auth service construction APIs stabilize.

### What should be done in the future

- Add a runnable example under `examples/xgoja` or `examples/gojahttp` if this API becomes a primary public integration path.
- Consider extracting a reusable Enforcer if more adapters need to share the same request setup and enforcement flow.

### Code review instructions

- Review `cmd/xgoja/doc/18-go-planned-auth-api.md` for API accuracy against `pkg/gojahttp/app.go`, `pkg/gojahttp/host.go`, and `pkg/gojahttp/middleware.go`.
- Validate with `GOFLAGS=-buildvcs=false go test ./cmd/xgoja ./cmd/xgoja/doc ./pkg/gojahttp ./modules/express -count=1`.

### Technical details

Files changed in this step:

```text
cmd/xgoja/doc/18-go-planned-auth-api.md
ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/reference/01-implementation-diary.md
```

## Step 6: Extract a reusable planned-auth Enforcer

This step converts the “future Enforcer” idea into an exported API. `Host` and `PlannedMiddleware` now share a router-independent `gojahttp.Enforcer` that owns the planned auth pipeline instead of making middleware construct a hidden host just to reuse host internals.

The new type keeps the higher-level route APIs intact. Application authors can still use `NewApp(host)` or `PlannedMiddleware`; adapter authors now have a smaller primitive for custom routers and generated dispatchers.

### Prompt Context

**User prompt (verbatim):** "add the reusable enforcer, and update the docs."

**Assistant interpretation:** Implement the reusable Enforcer abstraction discussed after the docs step, refactor the existing host/middleware paths to use it, and update the user-facing guide.

**Inferred user intent:** Make planned auth reusable by more Go router integrations without duplicating enforcement setup or routing everything through `Host`.

**Commit (code):** pending at time of diary update — planned message `Add reusable planned auth enforcer`.

### What I did

- Added `pkg/gojahttp/enforcer.go`.
- Added exported `EnforcerOptions`, `Enforcer`, `NewEnforcer`, `Enforcer.Session`, `Enforcer.Request`, and `Enforcer.Enforce`.
- Moved the shared secure-context construction logic from `Host.buildSecureEnvelope` into `Enforcer.Enforce`.
- Moved planned HTTP audit/error/handler execution into the enforcer-backed path.
- Refactored `Host` to create and hold an `Enforcer`.
- Refactored `PlannedMiddleware` to create an `Enforcer` instead of a hidden `Host`.
- Added `pkg/gojahttp/enforcer_test.go` for direct enforcer use and unauthenticated status behavior.
- Updated `cmd/xgoja/doc/18-go-planned-auth-api.md` with an Enforcer section and API-selection guidance.
- Ran focused validation:

```bash
GOFLAGS=-buildvcs=false go test ./pkg/gojahttp ./modules/express ./cmd/xgoja ./cmd/xgoja/doc -count=1
```

### Why

- Middleware and future router adapters need the same auth pipeline but not the whole `Host` router/runtime surface.
- A reusable enforcer gives generated code and custom routers a stable integration seam: build params/request DTO, call `Enforce`, then pass `SecureContext` to the domain handler.

### What worked

- Existing host and middleware tests continued to pass after the refactor.
- Direct `Enforcer` tests show the type can be used without `Host` routing.
- The JavaScript planned route path still receives a `secureEnvelope` adapter, but that adapter now wraps the `SecureContext` returned by the enforcer.

### What didn't work

- The first refactor had a Go compile error in `enforcer.go` because I tried to redeclare `status` in a scope where it already existed:

```text
pkg/gojahttp/enforcer.go:164:9: no new variables on left side of :=
```

- I changed the completed-audit status assignment from `status := 0` to `status = 0` and reran validation.

### What I learned

- `Enforcer.Enforce` is the right public primitive; the handler-execution helper still uses an unexported method because it depends on the package's access-log response writer.
- Keeping the JavaScript `secureEnvelope` as an adapter around exported `SecureContext` lets JS behavior remain stable while Go gets a cleaner API.

### What was tricky to build

- The old host method returned a `secureEnvelope`; the reusable API should return `SecureContext`. The solution was to make `Host.buildSecureEnvelope` a thin wrapper around `Enforcer.Enforce`.
- Middleware had to preserve its previous session-error versus request-DTO-error HTTP statuses, so it uses `Enforcer.Session` plus `NewRequestDTO` directly instead of only using `Enforcer.Request`.

### What warrants a second pair of eyes

- Whether `Enforcer.Request` should remain as a convenience if middleware prefers the more explicit `Session` + `NewRequestDTO` split.
- Whether a future exported helper should run a full `PlannedHTTPHandler` with audit/error behavior, or whether public callers should keep using `Enforce` and own their response writing.

### What should be done in the future

- Add package-level Go examples once the public API stabilizes.
- Consider exposing an audited handler runner only if a real external adapter needs it.

### Code review instructions

- Start with `pkg/gojahttp/enforcer.go` and compare `Enforcer.Enforce` to the previous `Host.buildSecureEnvelope` behavior.
- Review `pkg/gojahttp/planned_dispatch.go` to confirm JavaScript planned routes still wrap `SecureContext` in `secureEnvelope`.
- Review `pkg/gojahttp/middleware.go` to confirm middleware no longer constructs a hidden `Host`.
- Review `cmd/xgoja/doc/18-go-planned-auth-api.md` for updated public guidance.
- Validate with `GOFLAGS=-buildvcs=false go test ./pkg/gojahttp ./modules/express ./cmd/xgoja ./cmd/xgoja/doc -count=1`.

### Technical details

Files changed in this step:

```text
pkg/gojahttp/enforcer.go
pkg/gojahttp/enforcer_test.go
pkg/gojahttp/host.go
pkg/gojahttp/middleware.go
pkg/gojahttp/planned_dispatch.go
pkg/gojahttp/planned_http.go
cmd/xgoja/doc/18-go-planned-auth-api.md
ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/reference/01-implementation-diary.md
```

## Step 7: Add a runnable Go-only planned auth example

This step adds a concrete example for developers who want to see the Go-native planned auth API outside of tests. The example is intentionally small: a public health route, an authenticated project route registered through the fluent Go builder, and an equivalent protected route mounted through standard `http.ServeMux` and `PlannedMiddleware`.

The example does not attempt to be production auth. Its demo auth services read `X-Demo-User`, resolve synthetic project resources, and deny a single `forbidden` resource ID so smoke tests can exercise public, unauthenticated, allowed, middleware, and forbidden paths.

### Prompt Context

**User prompt (verbatim):** "go ahead."

**Assistant interpretation:** Proceed with the previously suggested next step: add a runnable example for the new Go planned auth API.

**Inferred user intent:** Make the new public API easier to evaluate and copy from by adding a smoke-testable example.

**Commit (code):** pending at time of diary update — planned message `Add Go planned auth example`.

### What I did

- Added `examples/gojahttp/01-planned-auth/main.go`.
- Added `examples/gojahttp/01-planned-auth/README.md`.
- Added `examples/gojahttp/01-planned-auth/Makefile` with `smoke` and `serve` targets.
- Updated `examples/README.md` to list the new `gojahttp/` examples folder and smoke command.
- Updated `cmd/xgoja/doc/18-go-planned-auth-api.md` to point readers at the runnable example.
- Ran validation:

```bash
make -C examples/gojahttp/01-planned-auth smoke
GOFLAGS=-buildvcs=false go test ./examples/gojahttp/01-planned-auth ./pkg/gojahttp ./modules/express ./cmd/xgoja ./cmd/xgoja/doc -count=1
```

### Why

- Tests prove behavior, but examples teach intended use. The API now has several entry points, so a runnable example helps developers see how they fit together.
- The example also gives maintainers a cheap smoke target for the host route and middleware integration paths.

### What worked

- The smoke target runs through `go run` and uses an internal `httptest.Server`, so it does not leave a long-running server behind.
- The fluent `NewApp(host)` and `PlannedMiddleware` paths can share the same tiny demo auth implementation.
- Focused validation passed.

### What didn't work

- N/A.

### What I learned

- The middleware route does not emit host access logs because it is mounted directly on the outer `http.ServeMux`, while the fluent app routes do go through `Host` logging. That difference is expected and is useful to remember when comparing smoke output.

### What was tricky to build

- The example mux has both `/` mounted to `gojahttp.Host` and a more specific Go 1.22 mux pattern for `/middleware/projects/{projectID}`. The standard mux selects the more specific pattern, allowing the middleware route to demonstrate non-Host dispatch.
- The demo had to stay small enough to be readable while still covering resource resolution and authorization, so the policy is intentionally simple: `project.read` is allowed unless the resource ID is `forbidden`.

### What warrants a second pair of eyes

- Whether the example should live under `examples/gojahttp` or as a numbered `examples/xgoja` example. It is Go-only, so `examples/gojahttp` seems more accurate.
- Whether the docs should include a shorter inline excerpt from the example once the API stabilizes further.

### What should be done in the future

- Add a second example only if a real generated-host adapter starts using `Enforcer` directly.
- Consider package-level `Example...` tests if Go doc discoverability becomes important.

### Code review instructions

- Start with `examples/gojahttp/01-planned-auth/main.go` and verify the demo auth services are clearly non-production.
- Run `make -C examples/gojahttp/01-planned-auth smoke`.
- Run `GOFLAGS=-buildvcs=false go test ./examples/gojahttp/01-planned-auth ./pkg/gojahttp ./modules/express ./cmd/xgoja ./cmd/xgoja/doc -count=1`.

### Technical details

Files changed in this step:

```text
examples/gojahttp/01-planned-auth/main.go
examples/gojahttp/01-planned-auth/README.md
examples/gojahttp/01-planned-auth/Makefile
examples/README.md
cmd/xgoja/doc/18-go-planned-auth-api.md
ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/reference/01-implementation-diary.md
```
