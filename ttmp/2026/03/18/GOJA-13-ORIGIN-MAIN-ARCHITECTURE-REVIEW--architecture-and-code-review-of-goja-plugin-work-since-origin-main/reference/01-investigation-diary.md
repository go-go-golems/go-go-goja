---
Title: Investigation diary
Ticket: GOJA-13-ORIGIN-MAIN-ARCHITECTURE-REVIEW
Status: active
Topics:
    - goja
    - analysis
    - architecture
    - tooling
    - refactor
    - repl
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/bun-demo/main.go
      Note: Third runtime consumer reviewed for repeated plugin bootstrap logic
    - Path: cmd/js-repl/main.go
      Note: Bobatea REPL bootstrap reviewed for duplicated runtime and docs wiring
    - Path: cmd/repl/main.go
      Note: Line REPL bootstrap reviewed for duplicated runtime and docs wiring
    - Path: engine/factory.go
      Note: Runtime creation flow reviewed for setup-time versus runtime-time state ownership
    - Path: engine/runtime.go
      Note: Owned runtime lifecycle reviewed for missing runtime-scoped value access
    - Path: modules/glazehelp/glazehelp.go
      Note: Legacy help module inspected as a still-live parallel docs surface
    - Path: pkg/docaccess/runtime/registrar.go
      Note: Runtime docs adapter inspected for concentration of provider and JS bridge logic
    - Path: pkg/hashiplugin/host/client.go
      Note: Plugin subprocess startup and diagnostics handling reviewed for hidden failures
    - Path: pkg/hashiplugin/contract/validate.go
      Note: Shared manifest validation extracted here to remove duplicated namespace/export/method checks
    - Path: pkg/hashiplugin/host/validate.go
      Note: Host-side validation reviewed first, then simplified into a thin wrapper over shared contract validation
    - Path: pkg/hashiplugin/sdk/module.go
      Note: SDK-side manifest validation and construction compared with host rules, then trimmed down to authoring-only checks
    - Path: pkg/repl/evaluators/javascript/evaluator.go
      Note: |-
        Evaluator help/completion path inspected for incomplete integration with docaccess
        Evaluator help and completion path inspected for incomplete integration with docaccess
ExternalSources: []
Summary: Chronological record of how the origin-main review ticket was scoped, what evidence was gathered, and how the main maintainability findings were derived from the committed branch delta.
LastUpdated: 2026-03-18T18:05:00-04:00
WhatFor: Preserve the reasoning and command history behind the architecture review so future cleanup work can start from evidence instead of memory.
WhenToUse: Use when revisiting the review findings, continuing cleanup tickets, or validating that GOJA-13 covered the right branch surface.
---


# Investigation diary

## Goal

Record how the architecture/code review of the `origin/main..HEAD` plugin and docs work was conducted, which files shaped the conclusions, and what commands were used to map the branch surface.

## Context

The user asked for a full review of the branch relative to `origin/main`, specifically looking for:

- duplicated code
- deprecated or wrapping layers
- overcomplexity
- unclear naming
- unidiomatic patterns
- missing error handling
- complicated runtime behavior
- code that needs documentation

The correct response was not a shallow diff summary. The branch has introduced new runtime architecture, plugin transport, authoring SDK, example plugins, and a docs hub. The review needed to distinguish between:

- structural improvements worth keeping
- cleanup opportunities that should happen before the design hardens

## Quick Reference

### User prompt (verbatim)

```text
Ok, let's do a full review of where we have gone from origin/main to now. THe point is to find any duplicated / deprecated / wrapping / overcomplex / unclear naming / unidiomatic / missing error handling / complicated runtime behavior / undocumented but needs documentation. 

Basically, making something that will help us create a long lived codebase without overcomplexifying things, and without oversimplifying either. 

Create a detailed code review / architecture review report in a new review ticket.

Feel free to run any experiments you want in the ticket's scripts/ folder.
```

### Assistant interpretation

Create a new review ticket, inspect the full committed branch delta against `origin/main`, map the new architecture, identify concrete maintainability and complexity risks with evidence, and publish a long-form review report plus supporting diary.

### Main findings

- runtime-scoped values currently stop at setup time instead of becoming owned runtime state
- runtime/bootstrap logic is duplicated across `repl`, `js-repl`, and `bun-demo`
- the old `glazehelp` module is still live and inconsistent with the new docs architecture
- plugin validation and naming rules are duplicated and slightly under-specified
- diagnostics and cancellation behavior around plugin processes need strengthening

## Usage Examples

### If you are creating follow-up cleanup tickets

Start from the review report and turn each finding into a focused ticket:

1. runtime value plumbing
2. entrypoint bootstrap consolidation
3. `glazehelp` deprecation or compatibility policy
4. plugin validation centralization
5. plugin diagnostics and cancellation hardening

### If you are reviewing the review

Validate that every major claim is backed by:

- a concrete file
- a concrete command from the branch review
- an explanation of why the current code shape matters at runtime or in maintenance

## Related

- Review report: `design-doc/01-origin-main-review-report-for-plugin-and-documentation-architecture.md`
- Task list: `tasks.md`
- Changelog: `changelog.md`
- Reproduction script: `scripts/list-origin-main-review-surface.sh`

## Step 1: Create GOJA-13 and map the committed branch surface

The first step was to reduce the review target to something objective: the committed delta from `origin/main` to `HEAD`. I created a dedicated review ticket, then collected the commit list, diff stat, directory distribution, and changed-file inventory before reading implementation details.

This mattered because the branch is large enough that intuition is unreliable. The diff surface shows where the real architecture work happened:

- `pkg/hashiplugin`
- `pkg/docaccess`
- `engine`
- REPL entrypoints

It also shows how much of the branch is ticket documentation versus runtime code.

### What I did

- Created `GOJA-13-ORIGIN-MAIN-ARCHITECTURE-REVIEW`.
- Collected:
  - `git log --oneline origin/main..HEAD`
  - `git diff --stat origin/main..HEAD`
  - `git diff --name-only origin/main..HEAD`
  - `git diff --dirstat=files,0 origin/main..HEAD`
- Added `scripts/list-origin-main-review-surface.sh` to the ticket for reproducibility and ran it.

### Why

- The user asked for a full review of the branch, not a package-by-package review picked from memory.
- A reproducible command script makes the review easier to revisit later.

### What worked

- The branch surface divided cleanly into runtime composition, plugin architecture, docs architecture, and entrypoint integration.
- The command script is enough to regenerate the main git evidence quickly.

### What didn't work

- Nothing failed technically in this step.

### What I learned

- The branch architecture is coherent overall. The main risks are at the seams between new subsystems, not inside one obviously broken package.

### Technical details

- Commands run:
  - `git branch --show-current && git rev-parse --short HEAD && git rev-parse --short origin/main`
  - `git log --oneline --decorate origin/main..HEAD`
  - `git diff --stat origin/main..HEAD`
  - `git diff --name-only origin/main..HEAD`
  - `git diff --dirstat=files,0 origin/main..HEAD`
  - `ttmp/2026/03/18/GOJA-13-ORIGIN-MAIN-ARCHITECTURE-REVIEW--architecture-and-code-review-of-goja-plugin-work-since-origin-main/scripts/list-origin-main-review-surface.sh`

## Step 2: Inspect the runtime and plugin seams for long-term architectural debt

After inventory, I focused first on the runtime and plugin layers because that is where lifecycle, cancellation, state ownership, and validation duplication live. I read the engine builder/runtime flow, plugin discovery/load/reification, and SDK-side manifest construction side by side.

This quickly surfaced the first group of review findings:

- runtime-scoped values exist during setup but not on the final owned runtime
- plugin validation rules are duplicated across SDK and host
- plugin process diagnostics are weaker than the surface suggests
- plugin invocation does not use a runtime-owned cancellation context

### What I did

- Read:
  - `engine/factory.go`
  - `engine/runtime.go`
  - `engine/runtime_modules.go`
  - `pkg/hashiplugin/host/client.go`
  - `pkg/hashiplugin/host/config.go`
  - `pkg/hashiplugin/host/discover.go`
  - `pkg/hashiplugin/host/reify.go`
  - `pkg/hashiplugin/host/report.go`
  - `pkg/hashiplugin/host/validate.go`
  - `pkg/hashiplugin/sdk/module.go`
  - `pkg/hashiplugin/sdk/export.go`
  - `pkg/hashiplugin/sdk/convert.go`
  - `pkg/hashiplugin/contract/jsmodule.proto`

### Why

- The runtime and plugin architecture are the part most likely to become expensive to change later.
- Long-lived maintenance pain often comes from setup-time/runtime-time mismatches and duplicated validation logic.

### What worked

- The host/shared/sdk split in `pkg/hashiplugin` is a solid base.
- The main issues are cleanup and consolidation problems, not evidence that the split itself was wrong.

### What didn't work

- One review concern turned into a stronger finding than expected: the protobuf contract now exposes method metadata richness that the SDK does not yet populate, which is exactly the kind of half-used schema that drifts over time.

### What warrants a second pair of eyes

- Whether runtime-scoped values should be stored raw on `engine.Runtime` or behind a narrower accessor interface.
- Whether stricter symbol validation should be enforced now or whether structured IDs should replace separator encoding.

## Step 3: Inspect docs and REPL integration for duplication and deprecation drift

The next pass looked at the entrypoints, docs hub, legacy `glazehelp` module, and evaluator help path together. This is where the “too much wrapping / deprecated but still live / duplicated bootstrap” concerns were easiest to verify.

This pass confirmed:

- `repl`, `js-repl`, and `bun-demo` now repeat runtime setup logic
- `glazehelp` is still globally registered while the new `docs` module exists
- `js-repl` still relies mainly on static signature help rather than the new docs hub

### What I did

- Read:
  - `cmd/repl/main.go`
  - `cmd/js-repl/main.go`
  - `cmd/bun-demo/main.go`
  - `pkg/docaccess/runtime/registrar.go`
  - `pkg/docaccess/plugin/provider.go`
  - `pkg/docaccess/hub.go`
  - `pkg/repl/evaluators/javascript/evaluator.go`
  - `modules/glazehelp/glazehelp.go`
  - `modules/glazehelp/registry.go`
- Searched for duplicated helpers and entrypoint-local plugin reporting.

### Why

- The branch now has more than one runtime consumer, so bootstrap duplication has real cost.
- The user explicitly asked for deprecated/wrapping/overcomplex paths, and the old/new docs surfaces are the clearest example of that.

### What worked

- The new docs hub itself is a real improvement over growing `glazehelp`.
- The remaining problem is not the new hub. It is the coexistence story.

### What didn't work

- The current mixed docs state is not yet honest from a product standpoint: the old path still exists, but it is no longer the architectural center.

### What I learned

- The branch’s biggest design risk is not “too much abstraction” in isolation. It is “new architecture added without fully retiring the old one.”

## Step 4: Write the report and package it as a durable review artifact

Once the evidence and findings were stable, I wrote the review report as a cleanup-oriented architecture document rather than a generic changelog. The report is organized around findings, each with:

- problem statement
- file-backed evidence
- why it matters
- cleanup sketch

That format makes it directly usable as a refactor agenda.

### What I did

- Wrote the main review report.
- Grouped recommendations into:
  - low-risk refactors
  - larger architectural changes
- Added a recommended cleanup sequence.

### Why

- The user asked for something that helps make this a long-lived codebase.
- That means the report must help prioritize cleanup, not just criticize the current state.

## Step 5: Validate and publish the review ticket

The final step is the usual ticket hygiene: relate files, run `docmgr doctor`, dry-run the bundle upload, upload it, and verify the remote listing. This keeps the review durable and easy to hand off.

### What I did

- Related the core runtime/plugin/docs files back into the ticket docs.
- Ran `docmgr doctor --ticket GOJA-13-ORIGIN-MAIN-ARCHITECTURE-REVIEW --stale-after 30`.
- Performed a dry-run bundle upload to reMarkable.
- Uploaded the final bundle and verified the remote listing.

### Technical details

- Commands run:
  - `docmgr doc relate --doc ... --file-note "..."`
  - `docmgr doctor --ticket GOJA-13-ORIGIN-MAIN-ARCHITECTURE-REVIEW --stale-after 30`
  - `remarquee upload bundle ... --dry-run ...`
  - `remarquee upload bundle ... --force ...`
  - `remarquee cloud ls /ai/2026/03/18/GOJA-13-ORIGIN-MAIN-ARCHITECTURE-REVIEW --long --non-interactive`

## Step 6: Turn the review into implementation work and land the first cleanup slices

After the review ticket existed as a stable artifact, the next request was to turn the findings into action. The first pass was deliberately narrow:

- write a concrete implementation plan for registrar state and diagnostics/cancellation
- turn the report findings into a cleanup backlog
- execute two low-risk/high-value cleanups immediately:
  - remove the legacy `glazehelp` module path
  - centralize duplicated HashiPlugin manifest validation

### What I did

- Added `design-doc/02-registrar-state-and-plugin-diagnostics-hardening-implementation-plan.md`.
- Expanded `tasks.md` with execution phases covering runtime state persistence, cancellation, diagnostics, `glazehelp` removal, bootstrap deduplication, and validation cleanup.
- Removed:
  - `modules/glazehelp/glazehelp.go`
  - `modules/glazehelp/registry.go`
  - the related tests
  - the blank import in `engine/runtime.go`
  - the `glazehelp.Register(...)` path in `cmd/repl/main.go`
- Added shared manifest validation in `pkg/hashiplugin/contract/validate.go`.
- Simplified `pkg/hashiplugin/host/validate.go` into a thin wrapper over the shared validator.
- Trimmed `pkg/hashiplugin/sdk/module.go` so it now keeps SDK-specific handler/nil checks and defers shared manifest shape rules to the contract validator.
- Added direct tests for the shared validator.

### Why

- `glazehelp` had become exactly the kind of deprecated parallel surface the review warned about.
- The validation duplication was small enough to fix immediately and likely to drift more if left alone.

### What worked

- Removing `glazehelp` was straightforward because the live path had already collapsed to one default-runtime import and one `repl` registration call.
- The shared validation split is cleaner than the previous host-vs-SDK duplication:
  - shared namespace/export/method-shape rules live in `contract`
  - host still layers allowlist/namespace config through options
  - SDK still owns nil-handler/nil-definition concerns

### What didn't work

- One SDK test initially failed because it was asserting the exact old error wording from the SDK-local validation path. Updating the assertion to the new shared-validation wording fixed it.

### Technical details

- Commands run:
  - `rg -n "glazehelp|ValidateManifest\\(|validateModuleDefinition\\(" -S cmd engine modules pkg`
  - `gofmt -w ...`
  - `go test ./pkg/hashiplugin/contract ./pkg/hashiplugin/host ./pkg/hashiplugin/sdk ./cmd/repl -count=1`
  - `go test ./... -count=1`

### What I learned

- The review findings were accurate enough to turn into code quickly. Neither cleanup required redesigning the surrounding architecture.
- The remaining deeper work from GOJA-13 is runtime-state persistence and diagnostics/cancellation hardening, not more cleanup of obviously obsolete surfaces.
