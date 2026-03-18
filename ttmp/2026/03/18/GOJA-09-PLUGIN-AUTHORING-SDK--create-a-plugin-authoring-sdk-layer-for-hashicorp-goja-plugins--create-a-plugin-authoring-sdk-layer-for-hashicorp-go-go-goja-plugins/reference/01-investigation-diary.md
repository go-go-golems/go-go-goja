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
- Added `pkg/hashiplugin/sdk/export.go` with `Function(...)`, `Object(...)`, `Method(...)`, `ExportDoc(...)`, and `ObjectDoc(...)`.
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
