---
Title: Investigation Diary
Ticket: XGOJA-EXTERNAL-HTTP-HOST
Status: active
Topics:
    - xgoja
    - goja
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/changelog.md
      Note: Ticket changelog for guide creation and upload
    - Path: go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/design-doc/01-external-go-http-host-integration-implementation-guide.md
      Note: Primary guide produced by this diary step
    - Path: go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md
      Note: Ticket implementation and delivery checklist
ExternalSources:
    - https://github.com/go-go-golems/go-go-goja/issues/65
Summary: Chronological diary for creating the external Go HTTP host integration ticket and implementation guide.
LastUpdated: 2026-06-09T01:25:00-04:00
WhatFor: Use this diary to understand why the first implementation should be non-invasive and how the design guide was assembled.
WhenToUse: Read before resuming XGOJA-EXTERNAL-HTTP-HOST implementation or updating the guide after code changes.
---


# Diary

## Goal

Capture the ticket setup, evidence collection, and delivery work for the non-invasive xgoja external Go HTTP host integration design. This diary records why the guide keeps current `HostServices` names for now, what code paths shaped the proposal, and how the document was validated and uploaded.

## Step 1: Create external Go HTTP host integration ticket and guide

I created a new `go-go-goja` docmgr ticket for the planned non-invasive approach: make generated xgoja packages configurable enough for a Go application to inject an external `gojahttp.Host`, then teach the HTTP provider to register Express routes into that host without owning the listener. This intentionally does not perform the larger `HostService*` to `RuntimeService*` rename because that breaking cleanup is now tracked separately in GitHub issue #65.

The resulting design document is written for a new intern. It explains the current xgoja provider/runtime construction path, the `HostServices` and provider contribution APIs, the generated package templates, `gojahttp.Host`, the Express module, the HTTP provider, the proposed service-injection hook, external-host behavior, route introspection, and runtime-manager validation strategy.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket, Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a focused docmgr ticket for the non-invasive xgoja external HTTP host plan, write a thorough implementation guide for an intern, validate the ticket, and upload the document bundle to reMarkable.

**Inferred user intent:** Preserve the future HostServices rename as a later issue, but give implementers a concrete, evidence-backed plan for external Go HTTP host integration now.

**Commit (code):** N/A — documentation and ticket setup only.

### What I did

- Created docmgr ticket `XGOJA-EXTERNAL-HTTP-HOST` under `go-go-goja/ttmp`.
- Added primary design doc:
  - `design-doc/01-external-go-http-host-integration-implementation-guide.md`
- Added this diary:
  - `reference/01-investigation-diary.md`
- Wrote `tasks.md` with completed documentation tasks and future implementation phases.
- Read the reMarkable upload skill and used the minimized upload workflow.
- Inspected current xgoja/provider/http/express/gojahttp code paths with line-numbered evidence.
- Cross-referenced existing broader design material from the ClubMedMeetup ticket and go-go-goja generated-package / HTTP serve tickets.
- Created GitHub issue #65 immediately before this ticket to track the later breaking HostServices-to-RuntimeServices rename.

### Why

- The current implementation need is not a broad rename. It is a small embedding hook plus HTTP provider external-host support.
- Generated package users need a way to supply live Go services without editing generated code or custom templates.
- The HTTP provider already has most required machinery after Express lazy binding: route registration now triggers the start hook, so external no-listen mode can be implemented as an ownership-aware start no-op.
- A detailed guide reduces the chance that an intern accidentally changes the provider API names, starts with a generic runtime manager too early, or makes the HTTP provider bind a listener in external-host mode.

### What worked

- `docmgr --root go-go-goja/ttmp ticket create-ticket` created the ticket workspace successfully.
- `docmgr doc add` created the design doc and diary skeletons.
- Existing code already provided strong evidence for the guide:
  - `app.HostServices` already has a keyed service map.
  - `RuntimeFactory` already composes provider contributions before module setup.
  - `modules/express` already accepts an external `*gojahttp.Host`.
  - `gojahttp.Host` is already an `http.Handler` and dispatches through `runtimeowner`.
  - Generated package templates expose a clean `Options`/`NewBundle` seam.

### What didn't work

- There was no existing focused go-go-goja ticket for this exact non-invasive external-host approach. The relevant material existed in broader ClubMedMeetup research/design docs and adjacent go-go-goja tickets (`GOJA-064`, `GOJA-065`, `XGOJA-RUNTIME-POLISH`).
- The deliverable checklist mentions routine reMarkable status/account/listing checks, but the current reMarkable upload skill explicitly says not to run those expensive checks unless upload fails. I followed the upload skill and planned dry-run plus real upload only.

### What I learned

- The generated package API is close to sufficient; the main missing piece is `ConfigureServices` on `Options` and `HostOptions`.
- The non-invasive implementation can reuse the current `HostServiceLookup`/`HostServiceValues` model rather than inventing a parallel API.
- The bigger naming cleanup is worthwhile but should be isolated because `ModuleSetupContext.Host` and `HostService*` are already used across several provider packages.

### What was tricky to build

- The main tricky design point was error handling for `ConfigureServices`. `app.NewHostWithOptions` currently returns `*Host`, not `(*Host, error)`, so a non-invasive callback is easier than an error-returning callback. The guide recommends keeping provider payload validation in module setup, where errors can already abort runtime construction.
- Another tricky point was separating reusable xgoja work from app-specific RuntimeManager work. The guide recommends implementing external host support in `go-go-goja`, then proving the runtime manager app-locally before extracting a generic package.

### What warrants a second pair of eyes

- Whether `ConfigureServices func(*app.HostServices)` is sufficient, or whether we should accept a slightly more invasive error-returning constructor.
- Whether `ExternalHostService.OwnsListen` should exist in the first patch or whether the first patch should only support no-listen external mode.
- Whether route introspection should include static mounts in the first implementation or only method/pattern route descriptors.

### What should be done in the future

- Implement the phases in `tasks.md` in order.
- Keep the breaking rename issue #65 separate until downstream providers can be updated together.
- Add a follow-up implementation diary entry with actual commit hashes after code changes begin.

### Code review instructions

- Start with the design doc executive summary and sections 5-8 for the proposed API and validation plan.
- Check that file references match the current post-rebase branch state.
- Validate ticket hygiene with:
  - `docmgr --root go-go-goja/ttmp doctor --ticket XGOJA-EXTERNAL-HTTP-HOST --stale-after 30`
- Validate upload delivery with the `remarquee upload bundle` output.

### Technical details

Ticket creation commands:

```bash
docmgr --root go-go-goja/ttmp ticket create-ticket \
  --ticket XGOJA-EXTERNAL-HTTP-HOST \
  --title "Support external Go HTTP hosts for generated xgoja runtimes" \
  --topics xgoja,goja,http,architecture,runtime

docmgr --root go-go-goja/ttmp doc add \
  --ticket XGOJA-EXTERNAL-HTTP-HOST \
  --doc-type design-doc \
  --title "External Go HTTP Host Integration Implementation Guide"

docmgr --root go-go-goja/ttmp doc add \
  --ticket XGOJA-EXTERNAL-HTTP-HOST \
  --doc-type reference \
  --title "Investigation Diary"
```

Key evidence commands:

```bash
cd go-go-goja
nl -ba pkg/xgoja/providerapi/module.go | sed -n '1,130p'
nl -ba pkg/xgoja/providerapi/capabilities.go | sed -n '1,130p'
nl -ba pkg/xgoja/app/host.go | sed -n '1,160p'
nl -ba pkg/xgoja/app/factory.go | sed -n '1,220p'
nl -ba pkg/xgoja/providers/http/http.go | sed -n '1,220p'
nl -ba modules/express/express.go | sed -n '1,220p'
nl -ba pkg/gojahttp/host.go | sed -n '1,220p'
nl -ba pkg/gojahttp/route_registry.go | sed -n '1,140p'
nl -ba cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl | sed -n '1,140p'
```

### Delivery evidence

Docmgr validation passed after keeping the ticket topics within the shared vocabulary:

```text
## Doctor Report (1 findings)

### XGOJA-EXTERNAL-HTTP-HOST

- ✅ All checks passed
```

reMarkable dry-run succeeded and listed the four bundled files: index, implementation guide, diary, and tasks. The real upload then succeeded:

```text
OK: uploaded XGOJA External HTTP Host Guide.pdf -> /ai/2026/06/09/XGOJA-EXTERNAL-HTTP-HOST
```

## Step 2: Add host service configuration to app HostOptions

I implemented the first, intentionally non-invasive code slice: `app.HostServices` now has small helper methods for setting and appending keyed host services, and `app.HostOptions` can configure the service bag before `NewRuntimeFactory` captures it. This gives handwritten Go hosts a clean way to inject live services without manually constructing `map[string][]any`.

This step does not touch generated package templates or the HTTP provider yet. It only proves that the existing provider-facing `ModuleSetupContext.Host` path can carry host-supplied services from `app.NewHostWithOptions` into provider module setup.

### Prompt Context

**User prompt (verbatim):** "go on, implement task by task, commiting at appropriate intervals, keeping a detailed diary."

**Assistant interpretation:** Start implementing the new external HTTP host ticket one phase at a time, validating and committing at sensible boundaries while keeping the ticket diary current.

**Inferred user intent:** Move from design into implementation without batching all phases into one hard-to-review commit.

**Commit (code):** ed5c24a — "Add xgoja host service configuration hook"

### What I did

- Added `(*HostServices).SetHostService(key, value)` in `pkg/xgoja/app/assets.go`.
- Added `(*HostServices).AddHostService(key, value)` in `pkg/xgoja/app/assets.go`.
- Added `ConfigureServices func(*HostServices)` to `pkg/xgoja/app.HostOptions`.
- Updated `NewHostWithOptions` so it creates the asset-backed service bag, calls `ConfigureServices`, and only then builds `NewRuntimeFactory`.
- Added `TestHostServicesSetAndAddHelpers` in `pkg/xgoja/app/host_services_test.go`.
- Added `TestHostOptionsConfigureServicesVisibleToModuleSetup` in `pkg/xgoja/app/host_services_test.go`.
- Checked off the first two implementation tasks in `tasks.md`.

### Why

- External HTTP host injection needs a service value to reach provider module setup through the existing `ctx.Host` mechanism.
- The helper methods keep host application code from depending on the internal `map[string][]any` representation.
- Calling `ConfigureServices` before `NewRuntimeFactory` preserves the existing construction flow: runtime factory receives one base service bag and later merges provider contributions into runtime-specific service bags.

### What worked

- The existing `HostServices` type already implemented `providerapi.HostServiceLookup`, so adding mutation helpers was small.
- The module setup test confirmed host-supplied services are visible through `ctx.Host` during `NewModuleFactory`.
- Focused validation passed:
  - `go test ./pkg/xgoja/app -run 'TestHostServices(SetAndAddHelpers|ConfigureServicesVisibleToModuleSetup)|TestRuntimeFactoryPassesHostServicesToModules' -count=1`

### What didn't work

- N/A. This phase matched the design-guide sketch closely.

### What I learned

- `HostServices` is currently value-passed into `NewRuntimeFactory`, but the mutation callback runs before that value is copied, so the service map is captured correctly.
- The existing contribution collector already handles merging base services with provider-contributed services; no new collector path was needed.

### What was tricky to build

- The helper methods need pointer receivers because they may initialize the `Services` map. Read methods remain value receivers because they should not mutate the service bag.
- The test must trigger module setup by actually creating a runtime; simply constructing `app.Host` is not enough because provider module factories run during runtime module registration.

### What warrants a second pair of eyes

- Whether `SetHostService` should replace all existing values for a key, as implemented, or return an error if the key already exists. The current behavior is useful for host-owned singleton services.
- Whether `ConfigureServices` should eventually return an error. This phase keeps the non-invasive `func(*HostServices)` shape from the guide.

### What should be done in the future

- Wire the same callback into generated package and source-fragment templates.
- Use the callback to inject the HTTP provider's external host service.

### Code review instructions

- Start with `pkg/xgoja/app/host.go` to confirm callback ordering.
- Review `pkg/xgoja/app/assets.go` for helper validation and map initialization.
- Review `pkg/xgoja/app/host_services_test.go` for service visibility through `ModuleSetupContext.Host`.
- Validate with:
  - `go test ./pkg/xgoja/app -run 'TestHostServices(SetAndAddHelpers|ConfigureServicesVisibleToModuleSetup)|TestRuntimeFactoryPassesHostServicesToModules' -count=1`

### Technical details

Focused validation command:

```bash
cd go-go-goja && gofmt -w pkg/xgoja/app/assets.go pkg/xgoja/app/host.go pkg/xgoja/app/host_services_test.go && go test ./pkg/xgoja/app -run 'TestHostServices(SetAndAddHelpers|ConfigureServicesVisibleToModuleSetup)|TestRuntimeFactoryPassesHostServicesToModules' -count=1
```

Result:

```text
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/app	0.022s
```

## Step 3: Wire ConfigureServices into generated package outputs

I implemented the generated-code half of the service injection hook. Generated runtime packages and source-fragment bundles now expose `ConfigureServices func(*app.HostServices)` on their `Options` type and pass it through to `app.NewHostWithOptions`.

This makes the Phase 2 implementation usable by actual generated-package callers instead of only handwritten `app.NewHostWithOptions` tests. A host application can now import a generated package and configure the same service bag that provider modules receive through `ModuleSetupContext.Host`.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue the ticket implementation by wiring the previously added app-level service hook through generated runtime package code.

**Inferred user intent:** Keep implementation increments reviewable while making sure generated-package integrations, not just internal app tests, are covered.

**Commit (code):** bdb28e4 — "Expose service configuration in generated xgoja packages"

### What I did

- Added `ConfigureServices func(*app.HostServices)` to `Options` in `runtime_package.go.tmpl`.
- Passed `opts.ConfigureServices` into `app.HostOptions` in `runtime_package.go.tmpl`.
- Made the same `Options` and `NewBundle` updates in `bundle_fragment.go.tmpl` so source-fragment generation stays consistent.
- Updated package rendering tests to assert the generated API includes `ConfigureServices` and passes it through.
- Updated source-fragment rendering tests to assert `bundle.gen.go` includes the service hook.
- Updated generated package host smoke code to compile a caller using `xgojaruntime.Options{ConfigureServices: func(*app.HostServices) { ... }}`.
- Ran `go generate ./...` to check whether committed generated fixtures needed updates; no generated files changed.
- Checked off the generated template and generated package smoke tasks in `tasks.md`.

### Why

- The external HTTP host integration target is generated-package embedding. If the hook only exists on `app.HostOptions`, generated-package users still cannot use it without custom templates.
- Package and source-fragment outputs should expose the same host embedding API because both are meant to be imported by Go applications.

### What worked

- The template renderer formats generated Go with `go/format`, so adding the extra field in templates produced normal formatted output in tests.
- The existing generated package host smoke helper was a good validation point because it builds a temporary host module, imports the generated package, and runs it with `go run`.
- Focused validation passed:
  - `go test ./cmd/xgoja/internal/generate -run 'TestRenderPackageExposesRuntimeBundleAPI|TestRenderSourceFragmentsSplitsRuntimePackageAPI|TestGeneratedPackageTargetBuildsAndCreatesRuntime|TestWriteSourceFragmentsBuildsAndCreatesRuntime' -count=1`

### What didn't work

- N/A. Running `go generate ./...` was noisy because the bun-demo Dagger generator prints engine/session logs, but it completed and did not leave unrelated generated output changes.

### What I learned

- `runtime_package.go.tmpl` and `bundle_fragment.go.tmpl` must be kept in sync for embedding APIs; otherwise package mode and source-fragment mode drift.
- The existing smoke harness can validate compile-time API availability without inventing a new generated fixture provider.

### What was tricky to build

- The smoke test needed to import `pkg/xgoja/app` only to type the callback argument. That is intentional because generated package users will need that type to configure services.
- The test only validates generated API plumbing, not that a specific provider consumes the service. Provider consumption belongs to the HTTP external-host phase.

### What warrants a second pair of eyes

- Whether generated package docs should be updated in the same implementation PR or after HTTP external-host mode lands.
- Whether committed example generated code should be regenerated manually if future `go generate` behavior changes.

### What should be done in the future

- Add HTTP provider `ExternalHostService` and tests that use this generated hook for real route registration.
- Update generated package tutorial docs once the full external-host story is implemented.

### Code review instructions

- Review `cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl` and `bundle_fragment.go.tmpl` together.
- Review `cmd/xgoja/internal/generate/generate_test.go` for string assertions and the generated package smoke update.
- Validate with:
  - `go test ./cmd/xgoja/internal/generate -run 'TestRenderPackageExposesRuntimeBundleAPI|TestRenderSourceFragmentsSplitsRuntimePackageAPI|TestGeneratedPackageTargetBuildsAndCreatesRuntime|TestWriteSourceFragmentsBuildsAndCreatesRuntime' -count=1`

### Technical details

Focused validation command:

```bash
cd go-go-goja && gofmt -w cmd/xgoja/internal/generate/generate_test.go && go test ./cmd/xgoja/internal/generate -run 'TestRenderPackageExposesRuntimeBundleAPI|TestRenderSourceFragmentsSplitsRuntimePackageAPI|TestGeneratedPackageTargetBuildsAndCreatesRuntime|TestWriteSourceFragmentsBuildsAndCreatesRuntime' -count=1
```

Result:

```text
ok  	github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate	2.963s
```

Generator check:

```bash
cd go-go-goja && go generate ./...
```

The command completed and left only the intended template/test/doc changes in `git status`.

## Step 4: Add HTTP provider external-host mode

I implemented the core HTTP provider integration: selected xgoja runtimes can now carry an externally supplied `*gojahttp.Host` through the existing `ModuleSetupContext.Host` service bag, and the HTTP provider's Express module will register JavaScript routes into that host. In external no-listen mode, route registration does not bind the provider's TCP listener.

This is the main behavior needed by a Go-owned HTTP server. The Go application can own the outer `net/http.Server` and `ServeMux`, while JavaScript continues to use `require("express").app().get(...)` to register flexible route handlers.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue implementation after generated service hooks by making the HTTP provider consume the injected service and prove it does not bind a listener in external mode.

**Inferred user intent:** Deliver the key non-invasive bridge between generated xgoja packages and Go-owned HTTP hosts.

**Commit (code):** cf521dd — "Support external hosts in xgoja HTTP provider"

### What I did

- Added `HostServiceKey = "go-go-goja-http.host"` in `pkg/xgoja/providers/http/http.go`.
- Added `ExternalHostService{Host *gojahttp.Host, OwnsListen bool}`.
- Changed the provider module factory so it passes `ModuleSetupContext.Host` into HTTP loader construction.
- Added `capability.newExpressLoader(hostServices)` to resolve and validate external host services during module setup.
- Kept `capability.NewExpressLoader()` as the existing no-argument helper for tests and direct callers.
- Extended `runtimeEntry` with `external` and `ownsListen` flags.
- Changed `start` so `external && !ownsListen` returns before any `net.Listen` call.
- Added tests for:
  - wrong/nil external host service validation,
  - registering an Express route into an external `gojahttp.Host` through the full xgoja app/provider path,
  - route registration with an occupied configured listen address in external no-listen mode.
- Checked off the HTTP provider service, consumption, and no-listen tasks in `tasks.md`.

### Why

- The raw Express module already accepts a `gojahttp.Host`; the provider path was the missing layer.
- Service validation belongs in module setup because `NewModuleFactory` can return errors before JavaScript executes.
- Listener ownership must be explicit to prevent hybrid Go hosts from accidentally racing with the xgoja provider for the same TCP address.

### What worked

- The `ConfigureServices` hook from Step 2 made the full provider-path test straightforward: `app.NewHostWithOptions(... ConfigureServices ...)` injects the external host, and the HTTP provider sees it in `ctx.Host`.
- The recent Express lazy-start behavior made no-listen mode easy: route registration still calls `start`, but `start` can no-op when the host is external and the listener is owned by the embedding app.
- Focused validation passed:
  - `go test ./pkg/xgoja/providers/http -count=1`

### What didn't work

- N/A in the final implementation. The existing `NewExpressLoader()` API was preserved by adding an internal `newExpressLoader(hostServices)` helper rather than changing every direct test caller.

### What I learned

- Full app/provider tests are more useful than only testing `engine.NativeModuleRegistrar` because they prove `ModuleSetupContext.Host` carries the service through xgoja's runtime factory.
- A second lower-level occupied-port test is still valuable because it explicitly initializes HTTP settings to the occupied address and proves no bind happens in external no-listen mode.

### What was tricky to build

- The provider needs to validate service type early without breaking the existing no-arg `NewExpressLoader()` helper. The split between exported `NewExpressLoader()` and internal `newExpressLoader(hostServices)` keeps compatibility while allowing provider setup to return validation errors.
- `HostService(key)` may return a `[]any` when multiple values exist for a key. The external HTTP service is intentionally singular, so a multi-valued key will fail the type assertion with a clear error.

### What warrants a second pair of eyes

- Whether `OwnsListen: true` with an externally supplied host should be supported immediately or treated as reserved behavior. The start path supports it, but tests focus on `OwnsListen: false`.
- Whether the service key string should be versioned (`go-go-goja-http.host.v1`) before this becomes a compatibility contract.

### What should be done in the future

- Add route introspection so external-host tests and future runtime-manager status endpoints can list registered routes.
- Add generated-package end-to-end coverage once route introspection and/or a small generated HTTP fixture is available.

### Code review instructions

- Start with `pkg/xgoja/providers/http/http.go`, especially `ExternalHostService`, `externalHostService`, `newExpressLoader`, and the ownership checks in `start`.
- Review `pkg/xgoja/providers/http/http_test.go` for the full xgoja app/provider external-host test and the occupied-port no-listen regression test.
- Validate with:
  - `go test ./pkg/xgoja/providers/http -count=1`

### Technical details

Focused validation command:

```bash
cd go-go-goja && gofmt -w pkg/xgoja/providers/http/http.go pkg/xgoja/providers/http/http_test.go && go test ./pkg/xgoja/providers/http -count=1
```

Result:

```text
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http	0.021s
```

## Step 5: Add gojahttp route introspection

I added copy-safe route introspection to `gojahttp.Registry` and `gojahttp.Host`. This gives tests and future RuntimeManager status endpoints a way to list JavaScript-registered routes without exposing Goja callables or mutable registry internals.

I also connected the new introspection surface to the external-host provider test so it proves not only that requests are served, but also that Express route registration populated the externally supplied host's route registry.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue the implementation phases by adding route metadata needed for validation and future runtime-manager status/debug surfaces.

**Inferred user intent:** Keep implementation progressing through the checklist while preserving small, reviewable commits.

**Commit (code):** 7bdc617 — "Add gojahttp route introspection"

### What I did

- Added `RouteDescriptor{Method, Pattern}` to `pkg/gojahttp/route_registry.go`.
- Added `(*Registry).Routes() []RouteDescriptor`, returning normalized method/pattern descriptors and a copy-safe slice.
- Added `(*Host).Routes() []RouteDescriptor`, delegating to the underlying registry.
- Added route introspection tests in `pkg/gojahttp/route_registry_test.go`.
- Updated the HTTP provider external-host test to assert the externally supplied host lists the registered `/hello/:name` route.
- Checked off the route introspection and focused test tasks in `tasks.md`.

### Why

- RuntimeManager hot reload needs status/debug evidence such as active version, last error, and route list.
- Tests that inspect route descriptors can distinguish "route registered into the external host" from only "some request happened to return OK".
- The descriptor type intentionally omits handlers because Goja callables should not be exposed outside the registry.

### What worked

- `Registry` already had a mutex, so adding a read-locked copy method was straightforward.
- `Host.Routes()` was a tiny delegating method and works for both provider-created and externally supplied hosts.
- Focused validation passed:
  - `go test ./pkg/gojahttp ./pkg/xgoja/providers/http -count=1`

### What didn't work

- N/A.

### What I learned

- Route pattern normalization happens at registration time through `cleanPath`, so route descriptors naturally report canonical patterns such as `/cards/:id`.
- Route introspection does not need to know about request matching or params; it only needs method and pattern for the current use case.

### What was tricky to build

- The main correctness concern was not leaking mutable backing storage. The test mutates the returned slice and then reads routes again to prove the registry was not modified.
- Static mounts are intentionally not included in `Routes()` yet. They are a different kind of mount and can get a separate descriptor later if runtime status needs them.

### What warrants a second pair of eyes

- Whether route descriptors should include registration order explicitly. The current slice order is registration order, but it is not named in the type.
- Whether static mount descriptors should be added in the same API family before RuntimeManager status work begins.

### What should be done in the future

- Use `Host.Routes()` in RuntimeManager candidate snapshots after successful bootstrap/smoke tests.
- Consider adding static mount introspection if SPA/static reload status needs it.

### Code review instructions

- Review `pkg/gojahttp/route_registry.go` for `RouteDescriptor` and copy-safe `Routes()`.
- Review `pkg/gojahttp/host.go` for the host-level delegation.
- Review tests in `pkg/gojahttp/route_registry_test.go` and `pkg/xgoja/providers/http/http_test.go`.
- Validate with:
  - `go test ./pkg/gojahttp ./pkg/xgoja/providers/http -count=1`

### Technical details

Focused validation command:

```bash
cd go-go-goja && gofmt -w pkg/gojahttp/route_registry.go pkg/gojahttp/host.go pkg/gojahttp/route_registry_test.go pkg/xgoja/providers/http/http_test.go && go test ./pkg/gojahttp ./pkg/xgoja/providers/http -count=1
```

Result:

```text
ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp	0.005s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http	0.030s
```

## Step 6: Update public docs and implementation status

I updated the public xgoja documentation now that the external-host service injection path exists in code. The docs now explain that generated runtime package bundles can receive host-owned services through `Options.ConfigureServices`, and they call out the HTTP provider's `ExternalHostService` pattern for Go-owned servers.

I also refreshed this ticket's design guide with an implementation status section so future readers can distinguish the completed non-invasive `go-go-goja` work from the intentionally deferred app-local RuntimeManager proof.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue the XGOJA-EXTERNAL-HTTP-HOST implementation workflow after the HTTP provider and route-introspection commits by closing documentation gaps and keeping the diary accurate.

**Inferred user intent:** Finish the current ticket's reviewable implementation/documentation slices without losing the commit history and validation trail.

**Commit (code):** 91d88f8 — "Document external xgoja host services"

### What I did

- Replaced pending commit placeholders in Steps 2-5 with actual commit hashes.
- Updated `cmd/xgoja/doc/02-user-guide.md` to mention generated package `ConfigureServices` and the HTTP external-host use case.
- Updated `cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md` with a new host-supplied service section and HTTP provider example.
- Updated `examples/xgoja/14-generated-runtime-package/README.md` to list `Options.ConfigureServices` and show host-owned service injection.
- Added an implementation status section to the external host design guide.

### Why

- The generated package API changed, so user-facing docs should describe how embedding applications use the new hook.
- The HTTP external-host API is a cross-package integration point; docs need to make the listener ownership invariant explicit.
- The ticket design document was written before implementation and needed a status note to avoid stale "planned" language being misread as unimplemented.

### What worked

- The documentation changes were limited to existing xgoja user/provider docs and the generated-package example README.
- The implementation status section provides a compact checklist of what landed and what remains deferred.

### What didn't work

- N/A.

### What I learned

- The provider host-service documentation already had the right conceptual home for both provider-contributed and host-supplied services; adding the generated-package callback there keeps the API story in one place.

### What was tricky to build

- The terminology is still overloaded because `HostServices` is both the current provider API name and the host-supplied service bag name. The docs avoid introducing the future `RuntimeService` rename until the breaking cleanup issue is implemented.
- The design guide still contains pre-implementation evidence sections. Rather than rewriting the whole guide, I added an explicit implementation status section near the top.

### What warrants a second pair of eyes

- Whether `cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md` should include full imports for the HTTP example or stay focused on the API shape.
- Whether the generated runtime package example should grow a real external HTTP host sample in a later PR.

### What should be done in the future

- Add a complete generated-package HTTP example once an app-local RuntimeManager proof exists.
- Revisit the docs after the deferred `HostService*` to `RuntimeService*` rename is implemented.

### Code review instructions

- Start with `cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md` for the new host-supplied services section.
- Review `cmd/xgoja/doc/02-user-guide.md` and `examples/xgoja/14-generated-runtime-package/README.md` for user-facing generated-package wording.
- Review the implementation status block at the top of the ticket design guide.
- Validate with:
  - `docmgr --root go-go-goja/ttmp doctor --ticket XGOJA-EXTERNAL-HTTP-HOST --stale-after 30`

### Technical details

Doc validation command:

```bash
docmgr --root go-go-goja/ttmp doctor --ticket XGOJA-EXTERNAL-HTTP-HOST --stale-after 30
```

Result:

```text
## Doctor Report (1 findings)

### XGOJA-EXTERNAL-HTTP-HOST

- ✅ All checks passed
```
