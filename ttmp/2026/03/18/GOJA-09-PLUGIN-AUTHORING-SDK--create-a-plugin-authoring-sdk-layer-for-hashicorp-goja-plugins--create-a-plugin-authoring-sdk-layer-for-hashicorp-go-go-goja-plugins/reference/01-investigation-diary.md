---
Title: Investigation diary
Ticket: GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins
Status: active
Topics:
    - goja
    - go
    - js-bindings
    - architecture
    - tooling
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/doc/02-creating-modules.md
      Note: |-
        Existing native-module authoring guide used as the comparison point for desired SDK ergonomics
        Built-in module authoring guide used as the comparison point
    - Path: pkg/doc/13-plugin-developer-guide.md
      Note: |-
        Current plugin architecture guide used as context for host/runtime constraints the SDK must preserve
        Subsystem architecture guide used as background
    - Path: pkg/hashiplugin/contract/contract.go
      Note: |-
        Low-level plugin interface inspected to identify what an SDK should implement
        Low-level contract inspected during diary investigation
    - Path: pkg/hashiplugin/contract/jsmodule.proto
      Note: |-
        Transport schema inspected to understand the author-facing capability boundary
        Transport schema inspected during diary investigation
    - Path: pkg/hashiplugin/host/reify.go
      Note: |-
        Host reification logic inspected to confirm the only currently supported export shapes
        Supported export shapes captured in diary rationale
    - Path: pkg/hashiplugin/host/validate.go
      Note: |-
        Host validation rules inspected to determine which invalid states the SDK should make hard to express
        Validation rules captured in diary rationale
    - Path: pkg/hashiplugin/shared/plugin.go
      Note: |-
        Shared transport wiring inspected to identify bootstrapping boilerplate the SDK should hide
        Serve/handshake wiring inspected during diary investigation
    - Path: plugins/examples/greeter/main.go
      Note: |-
        Current example plugin reviewed as the main source of author-facing duplication
        Example plugin inspected as the main source of boilerplate
    - Path: plugins/testplugin/echo/main.go
      Note: |-
        Test plugin reviewed to confirm the same duplication appears outside the example path
        Test plugin inspected to confirm repeated boilerplate
    - Path: ttmp/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins--create-a-plugin-authoring-sdk-layer-for-hashicorp-go-go-goja-plugins/design-doc/01-plugin-authoring-sdk-layer-for-hashicorp-go-go-goja-plugins-architecture-and-implementation-guide.md
      Note: Primary design deliverable produced from this investigation
ExternalSources: []
Summary: Chronological record of how the GOJA-09 plugin authoring SDK ticket was created, investigated, designed, and then implemented in reviewable slices.
LastUpdated: 2026-03-18T16:45:00-04:00
WhatFor: Capture the evidence, decisions, and delivery steps behind the plugin authoring SDK design ticket.
WhenToUse: Use when reviewing the GOJA-09 analysis or continuing the SDK implementation later.
---


# Diary

## Goal

Document how the GOJA-09 plugin authoring SDK ticket was created, what repository evidence shaped the recommendation, and what a later implementer should review first.

## Step 1: Create GOJA-09 and turn the current plugin runtime into an authoring-SDK plan

This step created a new ticket focused on the author-facing side of the plugin system rather than the already-landed host/runtime transport work. I started from the current plugin implementation and documentation in the repository itself, not from an external note, because the next design question is now local: how to make plugin binaries easier to author without disturbing the host-owned runtime architecture that GOJA-08 established.

The core finding is simple: the runtime path is in good shape, but authoring is too low-level. Both current plugin binaries hand-assemble manifests, hand-dispatch invoke requests, hand-convert `structpb` values, and hand-wire `plugin.Serve(...)`. The design guide therefore recommends an SDK package above `contract` and `shared`, but below `host`, so plugin authors get a compact builder/dispatch API while the host/runtime layering stays unchanged.

### Prompt Context

**User prompt (verbatim):**

```text
Create a full ticket for 1. Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable.
```

**Assistant interpretation:** Create a new docmgr ticket for the plugin authoring SDK idea, inspect the existing plugin runtime and examples in detail, write a repo-grounded intern-oriented design guide, keep a diary, validate the ticket, and upload the bundle to reMarkable.

**Inferred user intent:** Turn the "authoring SDK" idea into an implementation-ready architecture artifact, not just a short suggestion in chat, so a new engineer can build it without rediscovering the plugin subsystem.

**Commit (code):** N/A

### What I did

- Created ticket `GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins`.
- Added a design doc and diary document to the ticket workspace.
- Inspected the current engine seam, plugin contract, shared transport, host validation/reification flow, example plugins, test plugins, and current help docs.
- Wrote a new design guide focused on the author-facing SDK layer rather than the already-implemented host/runtime integration.
- Updated the task list and changelog to reflect the completed research deliverable.
- Validated the ticket with `docmgr doctor`.
- Uploaded the final bundle to reMarkable and verified the remote listing.

### Why

- The user explicitly asked for a full ticket plus a detailed intern-oriented design and implementation guide.
- The right next step is not another runtime refactor; it is improving the authoring experience on top of the stable host/runtime path.
- The existing examples and docs already contain enough evidence to design a good SDK layer without guesswork.

### What worked

- The current plugin packages are small and easy to inspect, which made the duplication obvious.
- The existing host/runtime design was already documented well enough that the SDK design could focus on preserving invariants rather than rediscovering them.
- `docmgr` and the reMarkable bundle flow worked cleanly for this ticket.

### What didn't work

- Nothing failed technically. The main challenge was conceptual: the SDK design needed to improve ergonomics without accidentally becoming a second host-policy layer.

### What I learned

- The most important authoring friction is not transport itself; it is repeated manifest/dispatch/conversion/serve boilerplate.
- The host-side validation rules in `pkg/hashiplugin/host/validate.go` are the real guardrails the SDK should align with.
- The built-in native-module tutorial in `pkg/doc/02-creating-modules.md` is the right usability baseline for plugin authoring, even though the runtime model is different.

### What was tricky to build

- The tricky part was choosing the right abstraction level. A too-thin helper would leave most of the current boilerplate in place, while a reflection-heavy magical layer would be harder to explain and stabilize. The recommended middle ground is an explicit SDK with module/export builders, handler functions, argument/result conversion helpers, and one `Serve(...)` wrapper around the existing shared transport code.

### What warrants a second pair of eyes

- Whether the package should be named `pkg/hashiplugin/sdk` or `pkg/hashiplugin/authoring`.
- Whether SDK-side namespace validation should always default to `plugin:` even though host policy is configurable.
- Whether the invalid fixture should remain handwritten to preserve raw-contract coverage after the example plugins migrate to the SDK.

### What should be done in the future

- Implement `pkg/hashiplugin/sdk` in phases, starting with manifest generation and dispatch.
- Migrate `plugins/examples/greeter` first, then the test fixtures as appropriate.
- Rewrite the plugin tutorial to teach `sdk.ServeModule(...)` as the happy path once the SDK exists.

### Code review instructions

- Start with the design doc in `design-doc/01-plugin-authoring-sdk-layer-for-hashicorp-go-go-goja-plugins-architecture-and-implementation-guide.md`.
- Then compare its recommendations against `pkg/hashiplugin/contract/contract.go`, `pkg/hashiplugin/shared/plugin.go`, `pkg/hashiplugin/host/validate.go`, `pkg/hashiplugin/host/reify.go`, and `plugins/examples/greeter/main.go`.
- Use `pkg/doc/02-creating-modules.md` as the comparison point for the desired authoring ergonomics.

### Technical details

- Commands run:
  - `docmgr status --summary-only`
  - `docmgr ticket create-ticket --ticket GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins --title "Create a plugin authoring SDK layer for HashiCorp go-go-goja plugins" --topics goja,go,js-bindings,architecture,tooling`
  - `docmgr doc add --ticket GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins --doc-type design-doc --title "Plugin authoring SDK layer for HashiCorp go-go-goja plugins architecture and implementation guide"`
  - `docmgr doc add --ticket GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins --doc-type reference --title "Investigation diary"`
  - `rg --files pkg/hashiplugin plugins cmd pkg/doc engine | sort`
  - `rg -n "hashiplugin|plugin:|go-plugin|GRPCPlugin|Manifest\\(|Invoke\\(|Export" pkg/hashiplugin plugins cmd pkg/doc engine -S`
  - `nl -ba engine/runtime_modules.go | sed -n '1,220p'`
  - `nl -ba engine/factory.go | sed -n '1,260p'`
  - `nl -ba engine/runtime.go | sed -n '1,240p'`
  - `nl -ba pkg/hashiplugin/contract/contract.go | sed -n '1,120p'`
  - `nl -ba pkg/hashiplugin/contract/jsmodule.proto | sed -n '1,220p'`
  - `nl -ba pkg/hashiplugin/shared/plugin.go | sed -n '1,240p'`
  - `nl -ba pkg/hashiplugin/host/config.go | sed -n '1,260p'`
  - `nl -ba pkg/hashiplugin/host/validate.go | sed -n '1,240p'`
  - `nl -ba pkg/hashiplugin/host/client.go | sed -n '1,260p'`
  - `nl -ba pkg/hashiplugin/host/reify.go | sed -n '1,260p'`
  - `nl -ba pkg/hashiplugin/host/registrar.go | sed -n '1,260p'`
  - `nl -ba pkg/hashiplugin/host/discover.go | sed -n '1,240p'`
  - `nl -ba pkg/hashiplugin/host/registrar_test.go | sed -n '1,260p'`
  - `nl -ba plugins/examples/greeter/main.go | sed -n '1,220p'`
  - `nl -ba plugins/testplugin/echo/main.go | sed -n '1,220p'`
  - `nl -ba pkg/doc/02-creating-modules.md | sed -n '1,260p'`
  - `nl -ba pkg/doc/13-plugin-developer-guide.md | sed -n '1,260p'`
  - `docmgr doctor --ticket GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins --stale-after 30`
  - `remarquee status`
  - `remarquee cloud account --non-interactive`
  - `remarquee upload bundle --dry-run ...`
  - `remarquee upload bundle ...`
  - `remarquee cloud ls /ai/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK --long --non-interactive`

## Step 2: Implement the richer SDK core, manifest builder, dispatch layer, and serve wrapper

This step turned the design ticket into real code. I started with the richer but still opinionated v1 surface rather than the stripped-down map DSL, because the user explicitly preferred the more extensive version. The resulting package is `pkg/hashiplugin/sdk`, and it provides the explicit authoring API that was described in the design doc: module construction, function/object/method declarations, metadata helpers, a `Call` helper type, manifest generation, invoke dispatch, result conversion, and the thin `Serve(...)` wrapper over the existing shared transport.

The important point is that this slice did not change the host/runtime architecture at all. The SDK sits above `contract` and `shared`, and below `host`. That means the runtime path still loads plugins the same way, while plugin authors now have a much smaller amount of code to write once the examples migrate in the next slice.

### Prompt Context

**User prompt (verbatim):**

```text
alright, work on it task by task, committing at appropriate intervals as you go, and keeping a diary.
```

**Assistant interpretation:** Implement the GOJA-09 backlog in reviewable slices, update the ticket as the work lands, and preserve enough detail that the ticket remains a continuation artifact.

**Inferred user intent:** Move beyond design and actually build the richer SDK surface while keeping the work easy to review and resume.

**Commit (code):** `3c2a591` — `hashiplugin: add authoring sdk core`

### What I did

- Added `pkg/hashiplugin/sdk/module.go` with `Module`, `ModuleOption`, `NewModule(...)`, `MustModule(...)`, module metadata helpers, validation, and manifest generation.
- Added `pkg/hashiplugin/sdk/export.go` with `Function(...)`, `Object(...)`, `Method(...)`, `ExportDoc(...)`, and `ObjectDoc(...)`. Later follow-up work also added explicit method metadata helpers: `MethodSummary(...)`, `MethodDoc(...)`, and `MethodTags(...)`.
- Added `pkg/hashiplugin/sdk/call.go` with the initial `Call` helper API:
  `Len`, `Value`, `String`, `StringDefault`, `Float64`, `Bool`, `Map`, and `Slice`.
- Added `pkg/hashiplugin/sdk/convert.go` with request decoding and result encoding helpers around `structpb.Value`.
- Added `pkg/hashiplugin/sdk/dispatch.go` with the `(exportName, methodName)` dispatch table and `Invoke(...)` implementation.
- Added `pkg/hashiplugin/sdk/serve.go` with `Serve(...)` and `ServeModule(...)`.
- Added `pkg/hashiplugin/sdk/errors.go` for consistent argument error messages.
- Added `pkg/hashiplugin/sdk/sdk_test.go` covering manifest generation, host-validation compatibility, duplicate detection, dispatch behavior, encode failures, and gRPC adapter compatibility through `shared.ServerPluginSet(...)`.
- Ran focused tests for `sdk`, `shared`, and `host`.
- Fixed the initial lint failure by adding explicit `EXPORT_KIND_UNSPECIFIED` switch handling so the repo's exhaustive lint rule passed cleanly.

### Why

- The SDK is not useful until it can generate a valid manifest and answer `Invoke(...)`, so the first slice needed to make the package functional rather than just create empty files.
- The richer explicit API keeps the future extension points visible without dragging in reflection or code generation.
- Testing against both `host.ValidateManifest(...)` and the shared gRPC adapter gives immediate proof that the SDK fits the existing plugin system rather than creating a parallel one.

### What worked

- The package stayed cleanly layered: production code depends on `contract` and `shared`, not on `host`.
- The richer API was still small enough to implement directly without overbuilding.
- The SDK-generated manifest passed the current host validation rules.
- The SDK-authored module worked over the existing shared gRPC transport using `plugin.TestPluginGRPCConn(...)`.

### What didn't work

- The first commit attempt failed the repo's exhaustive lint rule because the new `ExportKind` switches did not yet handle `EXPORT_KIND_UNSPECIFIED`.

Exact lint failure:

```text
pkg/hashiplugin/sdk/dispatch.go:20:3: missing cases in switch of type contract.ExportKind: contract.ExportKind_EXPORT_KIND_UNSPECIFIED (exhaustive)
pkg/hashiplugin/sdk/module.go:132:3: missing cases in switch of type contract.ExportKind: contract.ExportKind_EXPORT_KIND_UNSPECIFIED (exhaustive)
```

- The fix was straightforward: add explicit `UNSPECIFIED` cases returning clear SDK errors, rerun formatting/tests, and recommit.

### What I learned

- The richer API is still compact enough to be practical if it stays explicit and avoids reflection.
- The most valuable validation for this slice was not just unit tests; it was proving compatibility with the existing host validation and shared transport.
- `ServeModule(...)` can remain a convenience helper while `MustModule(...)` + `Serve(...)` stays the clearer explicit path for examples and docs.

### What was tricky to build

- The trickiest design point was finding the right place for validation. If too much validation stayed host-only, the SDK would still allow authors to build obviously broken modules. If too much host policy moved into the SDK, the layering would blur. The chosen split is structural validation in the SDK and deployment/runtime policy in `host`.

### What warrants a second pair of eyes

- Whether the v1 `Call` helper set is exactly right, or whether one or two helpers should be cut before the public surface is documented.
- Whether `ObjectDoc(...)` is worth keeping in v1 if no immediate example uses it.
- Whether `ServeModule(...)` should remain the convenience helper name, or whether the codebase would read more clearly with a `MustServeModule(...)` helper later.

### What should be done in the future

- Migrate the example plugin and at least one test plugin to the SDK next.
- Extend host integration coverage so the SDK-authored example is loaded through the existing runtime path.
- Update the help docs once the example migration proves the public surface is stable enough to teach.

### Code review instructions

- Start with `pkg/hashiplugin/sdk/module.go`, `export.go`, and `dispatch.go`.
- Then read `pkg/hashiplugin/sdk/call.go` and `convert.go` to understand the author-facing handler boundary.
- Finish with `pkg/hashiplugin/sdk/sdk_test.go`, especially the manifest compatibility test and the gRPC round-trip test.

### Technical details

- Commands run:
  - `gofmt -w pkg/hashiplugin/sdk/*.go`
  - `go test ./pkg/hashiplugin/sdk ./pkg/hashiplugin/shared ./pkg/hashiplugin/host -count=1`
  - `git add pkg/hashiplugin/sdk`
  - `git commit -m "hashiplugin: add authoring sdk core"`

## Step 3: Migrate the example plugins to the SDK, extend runtime integration coverage, and rewrite the help docs

This step turned the SDK from an isolated package into the actual recommended authoring path. I migrated the user-facing `greeter` example and the `echo` test plugin to the new `pkg/hashiplugin/sdk` API, deliberately kept the invalid fixture handwritten so the raw contract path still has coverage, and added a new host integration test that builds and loads the SDK-authored `greeter` example through the existing runtime registrar.

Once that path worked, I updated the plugin help pages so they stop teaching authors to hand-build `Manifest(...)`, `Invoke(...)`, and `plugin.Serve(...)` for the common case. The docs now present the SDK as the default authoring layer and describe the lower-level contract as the underlying transport boundary rather than the first thing a new author should touch.

### Prompt Context

**User prompt (verbatim):** (see Step 2)

**Assistant interpretation:** Continue the GOJA-09 backlog in reviewable slices, moving from core SDK internals into real examples, integration validation, and user-facing documentation.

**Inferred user intent:** Prove the SDK is not just internally plausible but is actually the path the repo now wants plugin authors to use.

**Commit (code):** `945a573` — `hashiplugin: migrate examples to sdk`

### What I did

- Rewrote `plugins/examples/greeter/main.go` to use `sdk.MustModule(...)`, `sdk.Function(...)`, `sdk.Object(...)`, `sdk.Method(...)`, and `sdk.Serve(...)`.
- Rewrote `plugins/testplugin/echo/main.go` to use the SDK while keeping the integration-test intent clear.
- Left `plugins/testplugin/invalid/main.go` handwritten so raw-contract failure coverage remains in the test suite.
- Updated `plugins/examples/README.md` so it describes the SDK surface instead of low-level `plugin.Serve(...)` wiring.
- Added `TestRegistrarLoadsSDKAuthoredExamplePlugin` to `pkg/hashiplugin/host/registrar_test.go`.
- Updated `pkg/doc/12-plugin-user-guide.md` to point plugin authors at the SDK-backed example.
- Updated `pkg/doc/13-plugin-developer-guide.md` to describe the new layering with `pkg/hashiplugin/sdk`.
- Rewrote the authoring portions of `pkg/doc/14-plugin-tutorial-build-install.md` around the richer SDK path.
- Updated the GOJA-09 design doc so it reflects that the SDK now exists and that the examples have already been migrated.
- Verified the help pages still render through `cmd/repl help ...`.

### Why

- The SDK only matters if the examples and tests actually use it.
- Migrating the examples early makes the public surface concrete and quickly exposes missing affordances.
- Leaving the invalid fixture manual is the cleanest way to retain raw-contract test coverage while still moving the happy path to the SDK.
- The help docs need to match the code, or new plugin authors will still learn the older low-level path first.

### What worked

- The migrated `greeter` example became materially shorter and easier to read.
- The existing host/runtime integration accepted the SDK-authored plugins without any host-side code changes.
- The new registrar test proved the SDK-authored example loads through the real runtime path, not just SDK-local tests.
- The help pages rendered successfully after the tutorial and architecture updates.

### What didn't work

- Nothing failed technically in this slice. The main judgment call was whether to migrate the invalid fixture as well, and I chose not to because the handwritten path is still useful coverage.

### What I learned

- The SDK is already expressive enough for the current example plugins; there was no missing feature that forced a fallback to the low-level contract.
- Keeping one handwritten invalid fixture is a good compromise between dogfooding the SDK and preserving transport-level test coverage.
- The developer guide needed a clearer statement that `sdk`, `contract/shared`, and `host` are three different layers with different responsibilities.

### What was tricky to build

- The subtle part was avoiding a documentation split-brain. Once the examples moved, the tutorial and developer guide immediately became stale because they still taught raw `Manifest(...)`/`Invoke(...)` code. Updating the example code and the authoring docs in the same slice kept the repo coherent.

### What warrants a second pair of eyes

- Whether `plugins/testplugin/echo` should remain SDK-authored permanently, or whether one small handwritten positive fixture is still useful alongside the handwritten invalid fixture.
- Whether the tutorial should also show a second object-method example using numeric arguments now that `sdk.Call.Float64(...)` exists.
- Whether the developer guide should add one tiny API reference section specifically for `pkg/hashiplugin/sdk`.

### What should be done in the future

- Run the final GOJA-09 validation pass, update the ticket closeout state, and re-upload the bundle to reMarkable.
- Decide whether to add a dedicated quickstart help page for the SDK itself, or keep the tutorial as the primary authoring entrypoint.

### Code review instructions

- Start with `plugins/examples/greeter/main.go` and compare it mentally to the pre-SDK shape described in the GOJA-09 design doc.
- Then read `pkg/hashiplugin/host/registrar_test.go` to see the real runtime path loading an SDK-authored example.
- Finish with `pkg/doc/14-plugin-tutorial-build-install.md` to confirm the help system now teaches the SDK-based path.

### Technical details

- Commands run:
  - `gofmt -w plugins/examples/greeter/main.go plugins/testplugin/echo/main.go pkg/hashiplugin/host/registrar_test.go`
  - `go test ./pkg/hashiplugin/sdk ./pkg/hashiplugin/host ./plugins/examples/... ./plugins/testplugin/... -count=1`
  - `go run ./cmd/repl help goja-plugin-user-guide | sed -n '1,120p'`
  - `go run ./cmd/repl help goja-plugin-developer-guide | sed -n '1,160p'`
  - `go run ./cmd/repl help plugin-tutorial-build-install | sed -n '1,200p'`
  - `git add plugins/examples/greeter/main.go plugins/examples/README.md plugins/testplugin/echo/main.go pkg/hashiplugin/host/registrar_test.go`
  - `git commit -m "hashiplugin: migrate examples to sdk"`

## Step 4: Run the final GOJA-09 closeout pass, sync the ticket state, and refresh the reMarkable bundle

This final step was about making the ticket truthful and reviewable rather than adding more code. I checked off the remaining closeout tasks, reran the full validation pass after the SDK and docs work had landed, and refreshed the uploaded bundle so the reMarkable copy matches the final ticket state instead of an earlier intermediate draft.

### Prompt Context

**User prompt (verbatim):** "alright, work on it task by task, committing at appropriate intervals as you go, and keeping a diary."

**Assistant interpretation:** Finish the remaining GOJA-09 execution loop by recording the final validation evidence, closing the ticket checklist, and publishing the final bundle state.

**Inferred user intent:** Leave the repository and the ticket artifacts in a state where another engineer can review the shipped SDK work and trust that the docs, checklist, and external bundle all reflect the code that is actually on the branch.

**Commit (code):** `N/A` for the validation/upload pass itself; the matching closeout docs commit is recorded separately in git history.

### What I did

- Updated `tasks.md` so the final closeout items are explicitly checked off.
- Added this closeout diary step so the validation and delivery evidence live in the ticket, not just in terminal history.
- Updated `changelog.md` with a final GOJA-09 closeout summary that points reviewers at the shipped SDK, migrated examples, tests, and docs.
- Re-ran the full repository test suite after the SDK, example, and documentation slices were complete.
- Re-ran `docmgr doctor` for the GOJA-09 ticket.
- Re-uploaded the GOJA-09 bundle to reMarkable and verified the remote listing in the target directory.

### Why

- The ticket checklist should not imply unfinished work when the implementation is already complete.
- The diary is most useful when it captures the full lifecycle, including the final validation and publication pass.
- Re-uploading after the last ticket updates avoids drift between the checked-in ticket state and the copy sent to reMarkable.

### What worked

- The full repository test suite passed after the SDK migration work.
- `docmgr doctor` passed without requiring structural fixes to the ticket.
- The bundle upload succeeded and the target reMarkable directory still resolved to the expected GOJA-09 document name.

### What didn't work

- Nothing failed in this closeout slice. The work here was mostly consistency and proof rather than debugging.

### What I learned

- The most fragile part of these ticket-driven workflows is usually not the code; it is keeping the ticket artifacts, uploaded bundle, and branch history aligned. Doing one explicit closeout pass is worth it.
- The richer SDK remained small enough that the final full test run was still the right validation level; no specialized or manual-only validation was needed beyond the existing integration coverage.

### What was tricky to build

- The main subtlety was deciding what counts as "done" for a research-plus-implementation ticket. The answer here was: code merged into the branch, examples migrated, docs updated, tests passing, ticket state current, and external bundle refreshed.

### What warrants a second pair of eyes

- Whether the GOJA-09 design guide should now be split into a shorter API quickstart plus a deeper intern document, or whether the current long-form guide is the right shape for this repo.
- Whether `sdk.ServeModule(...)` should remain a supported helper in examples, or whether future docs should standardize more strongly on `MustModule(...)` + `Serve(...)`.

### What should be done in the future

- If plugin authoring becomes a bigger user story, add a dedicated quickstart page focused purely on the SDK API.
- Consider whether the next plugin hardening ticket should build on the new SDK surface with authoring-time validation helpers or trust-policy integration.

### Code review instructions

- Review `pkg/hashiplugin/sdk/*.go` first to understand the shipped API surface.
- Then read `plugins/examples/greeter/main.go` and `pkg/hashiplugin/host/registrar_test.go` to confirm the example and real runtime path both use the SDK successfully.
- Finish with the GOJA-09 `tasks.md` and `changelog.md` to verify the ticket history lines up with the code.

### Technical details

- Commands run:
  - `go test ./... -count=1`
  - `docmgr doctor --ticket GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins --stale-after 30`
  - `remarquee status`
  - `remarquee cloud account --non-interactive`
  - `remarquee upload bundle --dry-run --bundle-dir ttmp/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins--create-a-plugin-authoring-sdk-layer-for-hashicorp-go-go-goja-plugins --name "GOJA-09 Plugin authoring SDK" --remote-dir "/ai/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK" --toc-depth 2`
  - `remarquee upload bundle --force --bundle-dir ttmp/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins--create-a-plugin-authoring-sdk-layer-for-hashicorp-go-go-goja-plugins --name "GOJA-09 Plugin authoring SDK" --remote-dir "/ai/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK" --toc-depth 2`
  - `remarquee cloud ls /ai/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK --long --non-interactive`

## Step 5: Expand the SDK example catalog with focused plugins for metadata, validation, state, nested results, and errors

The original `greeter` example was enough to show the happy path, but it was no longer enough to show the actual shape of the richer SDK. I added five more user-facing example plugins under `plugins/examples/...`, each designed to teach one aspect of the public authoring surface instead of repeating the same pattern with different string literals.

The goal in this slice was not to invent new SDK features. It was to turn the shipped SDK into something authors can actually learn from by example. That meant building examples around the existing v1 surface: metadata helpers, zero-argument handlers, `sdk.Call` accessors, stateful object methods, nested JSON-like results, and explicit handler errors.

### Prompt Context

**User prompt (verbatim):** "yes go ahead with all of them"

**Assistant interpretation:** Implement the full example set previously proposed rather than only a smaller subset.

**Inferred user intent:** Make the SDK feel real and teachable by providing concrete, copyable plugins for the main use cases instead of one minimal baseline plus a test fixture.

**Commit (code):** `9ed3b51` — `plugins: add sdk example catalog`

### What I did

- Added `plugins/examples/clock/main.go` to show module metadata, zero-argument calls, and structured object return values.
- Added `plugins/examples/validator/main.go` to show `sdk.Call` helpers, optional/default handling, `Map`, `Slice`, `Bool`, and `Float64` reads, plus explicit validation errors.
- Added `plugins/examples/kv/main.go` to show stateful object methods backed by plugin-process-local memory.
- Added `plugins/examples/system-info/main.go` to show mixed export shapes and nested JSON-like responses.
- Added `plugins/examples/failing/main.go` to show explicit handler errors and conditional failures.
- Kept all examples within the existing SDK and transport constraints rather than adding API surface just to support demos.

### Why

- `greeter` teaches the baseline shape, but not the breadth of the SDK surface.
- Examples are the fastest way to make an API understandable to new authors.
- A focused catalog makes it easier to copy the right starting point without reading long reference docs first.

### What worked

- The examples compiled cleanly as ordinary Go packages with no SDK changes required.
- The richer SDK surface was already expressive enough to cover the planned example types.
- The new examples stayed conceptually distinct instead of collapsing into variants of `greeter`.

### What didn't work

- Nothing failed in this first catalog slice. The only real design constraint was staying within the JSON-like value boundary that the transport already expects.

### What I learned

- The current SDK is already broad enough for real teaching examples; the missing piece was the catalog, not more API design.
- It is easier to understand the SDK by looking at several intentionally narrow examples than by growing one example into a kitchen sink.

### What was tricky to build

- The main challenge was avoiding fake diversity. Each example had to justify its existence by teaching one thing clearly: metadata, validation, state, nested results, or errors.

### What warrants a second pair of eyes

- Whether `system-info` is the right name versus something shorter like `sysinfo`.
- Whether any of the examples should gain tiny local READMEs later, or whether the top-level catalog README is the right amount of structure.

### What should be done in the future

- Add docs and runtime tests that explicitly call out which example to read for which SDK behavior.
- Consider whether one example should eventually demonstrate a more document-oriented or command-oriented plugin if those become real user stories.

### Code review instructions

- Read `plugins/examples/greeter/main.go` first as the baseline.
- Then compare `plugins/examples/validator/main.go` and `plugins/examples/kv/main.go` to see how the same SDK surface handles input validation versus stateful behavior.
- Finish with `plugins/examples/system-info/main.go` and `plugins/examples/failing/main.go` for nested results and explicit error returns.

### Technical details

- Commands run:
  - `gofmt -w plugins/examples/clock/main.go plugins/examples/validator/main.go plugins/examples/kv/main.go plugins/examples/system-info/main.go plugins/examples/failing/main.go`
  - `go test ./plugins/examples/... -count=1`
  - `git add plugins/examples/clock/main.go plugins/examples/validator/main.go plugins/examples/kv/main.go plugins/examples/system-info/main.go plugins/examples/failing/main.go`
  - `git commit -m "plugins: add sdk example catalog"`

## Step 6: Turn the example directory into a catalog, add focused runtime coverage, and record one transport-shape sharp edge

After the example binaries existed, I rewrote the example README and the plugin help pages so the broader catalog is discoverable, then extended the host integration tests to cover the two most important non-trivial behaviors: state surviving across calls and handler errors crossing the plugin boundary. During that work I hit one small but useful edge case: returning `[]string` from a handler did not encode through the current protobuf conversion path, while `[]any{...}` did.

I fixed that by keeping the example aligned with the transport contract instead of expanding the SDK or host logic. That is exactly the kind of detail examples are supposed to expose.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Finish the example-catalog slice end to end, including docs, tests, ticket updates, and validation.

**Inferred user intent:** Do not stop at adding source files; make the new examples discoverable, tested where it matters, and recorded in GOJA-09.

**Commit (code):** `N/A` until the docs/tests/ticket sync commit is recorded in git history.

### What I did

- Rewrote `plugins/examples/README.md` into a real catalog with module names, teaching goals, build examples, and quick JavaScript probes.
- Updated `pkg/doc/04-repl-usage.md`, `pkg/doc/12-plugin-user-guide.md`, `pkg/doc/13-plugin-developer-guide.md`, and `pkg/doc/14-plugin-tutorial-build-install.md` to reference the broader example set.
- Added `TestRegistrarLoadsStatefulKVExamplePlugin` to prove stateful object methods survive across calls in one runtime.
- Added `TestRegistrarSurfacesPluginHandlerErrors` to prove explicit plugin handler errors surface back to the JavaScript caller.
- Fixed `plugins/examples/kv/main.go` so `store.keys()` returns `[]any` instead of `[]string`, matching the current conversion path used by `sdk.encodeResult(...)`.
- Re-ran the focused host/example/doc validations and rendered the updated help pages through `cmd/repl help ...`.

### Why

- Examples that cannot be discovered from the docs might as well not exist.
- The two most important runtime behaviors beyond the baseline happy path are state retention and error propagation.
- The `[]string` result issue was a concrete reminder that examples should reflect the actual transport boundary rather than an idealized one.

### What worked

- The catalog README made the example set much easier to scan and choose from.
- The new host integration tests passed once the `kv` example returned a transport-friendly slice shape.
- The help system rendered the updated catalog and tutorial pages successfully.

### What didn't work

- The first version of `kv.store.keys()` returned `[]string`, which failed with `proto: invalid type: []string` in the current `structpb.NewValue(...)` path.

### What I learned

- The current SDK surface is fine; the practical limitation lives one layer down in the value-encoding boundary.
- It is useful to keep at least one example that bumps into a real edge case, because it documents what “JSON-like” means in practice rather than in theory.

### What was tricky to build

- The subtle part was deciding whether to “fix” the SDK for `[]string` or to keep the example faithful to the current transport constraints. I chose the latter because the purpose of the example catalog is to teach the shipped system, not an aspirational one.

### What warrants a second pair of eyes

- Whether `encodeResult(...)` should eventually normalize simple typed slices like `[]string` into a `structpb`-friendly shape automatically.
- Whether the example catalog should later grow one tiny note about supported result shapes instead of relying on the more general “JSON-like values” wording.

### What should be done in the future

- Decide whether automatic normalization for common typed slices and maps belongs in the SDK or should stay an author responsibility.
- If that normalization is added later, update `kv` and the docs to show the simpler return shape.

### Code review instructions

- Start with `plugins/examples/README.md`.
- Then read `pkg/hashiplugin/host/registrar_test.go` for the new `kv` and `failing` integration coverage.
- Finish with `pkg/doc/12-plugin-user-guide.md` and `pkg/doc/14-plugin-tutorial-build-install.md` to confirm the new examples are discoverable from the user-facing docs.

### Technical details

- Commands run:
  - `gofmt -w pkg/hashiplugin/host/registrar_test.go`
  - `go test ./pkg/hashiplugin/host ./plugins/examples/... ./pkg/doc -count=1`
  - `go run ./cmd/repl help goja-plugin-user-guide | sed -n '1,220p'`
  - `go run ./cmd/repl help goja-plugin-developer-guide | sed -n '420,520p'`
  - `go run ./cmd/repl help plugin-tutorial-build-install | sed -n '1,260p'`

## Step 7: Close out the reopened example-catalog slice and refresh the external deliverable

Once the example catalog, docs, and focused tests were committed, the remaining work was ticket hygiene again: rerun the full validation pass from the committed state, refresh the uploaded bundle, and make the GOJA-09 workspace reflect that the reopened slice is complete rather than still in progress.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Finish the reopened GOJA-09 slice all the way through validation, upload, and ticket closeout.

**Inferred user intent:** Leave no ambiguity about whether the expanded example catalog is just local branch work or a fully closed out ticket deliverable.

**Commit (code):** `88fc319` — `plugins: document and test sdk example catalog`

### What I did

- Re-ran the full repository test suite after the example-catalog docs and host-test work landed.
- Re-ran `docmgr doctor` for GOJA-09.
- Verified `remarquee` status and cloud account access.
- Dry-ran the bundle upload with the current CLI syntax.
- Re-uploaded the GOJA-09 bundle from the committed ticket state.
- Verified the remote listing for `/ai/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK`.
- Checked off the final Phase 9 upload item in `tasks.md`.

### Why

- The reopened work should meet the same standard as the original GOJA-09 closeout: passing tests, current ticket artifacts, and a refreshed external bundle.
- Re-running the full suite catches drift that focused tests can miss, especially after help-page and example changes.

### What worked

- The full repository tests still passed after the example-catalog expansion.
- `docmgr doctor` remained clean.
- The reMarkable bundle refresh succeeded and the remote directory still showed the expected document.

### What didn't work

- Nothing failed in this final closeout slice.

### What I learned

- Reopening a ticket for follow-on work is manageable as long as the diary and checklist keep explicit boundaries between the original closeout and the reopened slice.
- The example catalog now carries more of the teaching burden, which should reduce pressure to over-explain the SDK in every help page.

### What was tricky to build

- The subtle part was not code; it was keeping the reopened slice from making the earlier closeout record misleading. The answer was to treat the reopened work as an explicit new ticket phase with its own final evidence.

### What warrants a second pair of eyes

- Whether the GOJA-09 design guide should now mention the expanded example catalog explicitly in its recommended reading order.
- Whether one of the new examples should be promoted into the main tutorial as an optional second walkthrough, especially `validator` or `kv`.

### What should be done in the future

- If a future ticket improves value normalization in `sdk.encodeResult(...)`, revisit `kv` and the docs to see whether plain typed slices should become first-class return shapes.
- Consider adding tiny smoke scripts under `testdata/` that exercise one or two example plugins from JavaScript for copy-paste manual testing.

### Code review instructions

- Start with `git show 9ed3b51` for the pure example-source slice.
- Then read `git show 88fc319` for the docs, tests, and ticket updates that turned the examples into a real catalog.
- Finish with `tasks.md` and this diary step to verify the reopened Phase 9 is fully closed.

### Technical details

- Commands run:
  - `go test ./... -count=1`
  - `docmgr doctor --ticket GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins --stale-after 30`
  - `remarquee status`
  - `remarquee cloud account --non-interactive`
  - `remarquee upload bundle ttmp/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins--create-a-plugin-authoring-sdk-layer-for-hashicorp-go-go-goja-plugins --dry-run --name "GOJA-09 Plugin authoring SDK" --remote-dir "/ai/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK" --toc-depth 2`
  - `remarquee upload bundle ttmp/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins--create-a-plugin-authoring-sdk-layer-for-hashicorp-go-go-goja-plugins --force --name "GOJA-09 Plugin authoring SDK" --remote-dir "/ai/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK" --toc-depth 2`
  - `remarquee cloud ls /ai/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK --long --non-interactive`
