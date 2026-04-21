---
Title: Investigation diary
Ticket: GOJA-042-REPL-CLEANUP
Status: active
Topics:
    - goja
    - go
    - repl
    - refactor
    - analysis
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological notes for the REPL cleanup/refactor design guide."
LastUpdated: 2026-04-08T18:19:30-04:00
WhatFor: "Record the reasoning behind the cleanup/refactor follow-up design."
WhenToUse: "Use when retracing why GOJA-042 is intentionally separated from correctness work."
---

# Investigation diary

## Why this ticket exists

The reviews found real cleanup work, but that work should not be mixed into the correctness PRs.

The main cleanup topics are:

- `pkg/replsession/service.go` is too large and too responsibility-dense
- app-level and kernel-level `SessionOptions` are easy to confuse
- the older evaluator path still exists through the Bobatea adapter and needs an explicit plan

## Files to understand first

- [pkg/replsession/service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go)
- [pkg/replsession/rewrite.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/rewrite.go)
- [pkg/replsession/policy.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go)
- [pkg/replapi/config.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go)
- [pkg/repl/adapters/bobatea/javascript.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repl/adapters/bobatea/javascript.go)
- [pkg/repl/evaluators/javascript/evaluator.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go)

## Main design stance

This ticket should improve readability and responsibility boundaries without changing semantics unnecessarily. It is a cleanup ticket, not a hidden behavior-change ticket.

## 2026-04-08 implementation session

This first implementation step focused on the highest-signal cleanup target without mixing in naming or product-facing API work. The goal was to make `pkg/replsession` easier to read by lifting the evaluation-specific code out of the giant service file while keeping the public shape and behavior stable.

The result is a new [evaluate.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go) file that now owns the evaluation pipeline. [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go) still holds lifecycle, persistence shaping, snapshotting, summary building, and binding/runtime support code, but it is materially smaller and less responsibility-dense than before.

### Step 1: Extract evaluation orchestration into `evaluate.go`

**User prompt (verbatim):** "ok, continue with GOJA-042"

**Assistant interpretation:** Start implementing the cleanup ticket, using the same staged workflow as the earlier tickets instead of leaving GOJA-042 as design-only documentation.

**Inferred user intent:** Reduce the structural complexity of the REPL kernel in reviewable increments while keeping the work traceable and low-risk.

### What I did

- Added [evaluate.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go).
- Moved the evaluation cluster out of [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go):
  - `Service.Evaluate`
  - `shouldAnalyze`
  - `buildRewriteReport`
  - `provenanceForPolicy`
  - `evaluateInstrumented`
  - `evaluateRaw`
  - `wrapTopLevelAwaitExpression`
  - `executeRaw`
  - `executeWrapped`
  - `runString`
  - `waitPromise`
  - `persistWrappedReturn`
  - `promisePreview`
  - evaluation-only helper structs
- Kept the utility helpers `executionStatus`, `errorString`, `evaluationContext`, `evaluationContextError`, and `gojaValuePreview` in `service.go` because they are still referenced from other non-evaluation helpers.
- Formatted the files and validated with:
  - `go test ./pkg/replsession ./pkg/replapi`

### Why

- This cluster is the most coherent responsibility boundary in `service.go`.
- Extracting it first reduces file size and cognitive load without forcing any naming or architectural decisions elsewhere.
- It is the best first slice because future timeout/rewrite/runtime-eval work should no longer require reading through persistence and summary code in the same file.

### What worked

- The move was mostly mechanical and did not require function renames.
- Package tests passed after the extraction.
- The resulting split is more legible: evaluation flow is now grouped together in one file.

### What didn't work

- During the first full package validation, `TestServiceRawAwaitPromiseTimeoutUsesEvalDeadline` failed once with `expected timeout status, got "runtime-error"`.
- The same test passed immediately when run in isolation, and the full package run passed on rerun.
- Because the extracted code path did not intentionally change the timeout logic, I treated this as a pre-existing timing-sensitive edge rather than proof of a cleanup regression.

### What I learned

- `pkg/replsession` is ready for incremental file extraction without a broad redesign.
- The evaluation cluster is cohesive enough to live in its own file cleanly.
- The timeout test likely has some timing sensitivity that predates the cleanup work and should be stabilized separately rather than hidden inside GOJA-042.

### What was tricky to build

- The main risk in a cleanup PR like this is accidental semantic drift from a "mechanical" move.
- To avoid that, I moved the whole evaluation cluster together instead of partially splitting it across multiple files, and I avoided renaming symbols in the same step.
- Another subtle point was deciding which small helpers should stay in `service.go`; I left shared utilities there instead of forcing a second split in the same patch.

### What warrants a second pair of eyes

- Verify that the file split really stayed mechanical and did not subtly alter the order of evaluation-side effects.
- Keep an eye on the promise-timeout test behavior in future runs; if it flakes again, stabilize it in a dedicated change instead of smuggling test semantics work into GOJA-042.

### What should be done in the future

- Extract the persistence/snapshot helpers next.
- Then revisit the `replapi.SessionOptions` vs `replsession.SessionOptions` naming boundary.
- Keep the older evaluator-path decision as a separate explicit cleanup step rather than bundling it into the file split.

### Code review instructions

- Start with [evaluate.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go) to review the new responsibility boundary.
- Then compare it against the slimmer [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go).
- Validate with:
  - `go test ./pkg/replsession ./pkg/replapi`

### Technical details

- Before this change, `service.go` contained both lifecycle and evaluation orchestration.
- After this change:
  - `evaluate.go` owns the evaluation pipeline
  - `service.go` still owns lifecycle, persistence shaping, snapshots, binding/runtime details, and summaries

### Step 2: Extract persistence shaping into `persistence.go`

After the evaluation split, the next most coherent cluster was the code that turns a finished cell report into durable database records. That code had nothing to do with session lifecycle, but it still lived in `service.go` because the original file had grown organically.

The result is a new [persistence.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/persistence.go) file. It now owns:

- `persistCell`
- `bindingPersistenceRecords`
- `snapshotBindingExports`
- `classifyBindingExport`
- `bindingVersionRecord`
- `bindingRemovalRecord`
- `extractBindingDocs`
- export snapshot helper types and defaults

### What I did

- Added [persistence.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/persistence.go).
- Moved the persistence-only helpers out of [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go) without renaming them.
- Kept `resolveSessionOptions` and session creation logic in `service.go` because those are still lifecycle concerns.
- Validated with:
  - `go test ./pkg/replsession ./pkg/replapi`

### Why

- Persistence shaping is a distinct responsibility from evaluation and session lifecycle.
- This split makes it much easier to review database write behavior without mentally filtering out runtime bootstrap code.
- It also makes future persistence changes lower-risk because there is now one focused file for that behavior.

### What worked

- The split was fully mechanical.
- Package tests passed immediately after the move.
- `service.go` dropped from 1175 lines to 890 lines after this slice.

### What I learned

- The persistence path is cohesive enough to stand alone cleanly.
- The REPL session package can be split by responsibility without introducing awkward circular dependencies.

### Step 3: Extract runtime observation and summary shaping into `observe.go`

Once persistence was separate, the remaining large cluster in [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go) was the runtime-observation code: global snapshots, binding runtime refresh, property/prototype inspection, and summary assembly. That is also a coherent subsystem, so it became the next cleanup slice.

The result is a new [observe.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/observe.go) file. It now owns:

- runtime global snapshot helpers
- binding/runtime detail refresh
- property and prototype inspection helpers
- summary shaping (`buildSummary*`)
- binding view shaping

### What I did

- Added [observe.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/observe.go).
- Moved the observation and summary helpers out of [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go).
- Left smaller shared helpers like `executionStatus`, `evaluationContext`, `firstDiagnosticMessage`, and `dedupeSortedStrings` in `service.go` because they are still used across the package.
- Validated with:
  - `go test ./pkg/replsession ./pkg/replapi`

### Why

- This split finally makes the remaining `service.go` file read like a service rather than a dump of every helper in the subsystem.
- The runtime-observation code is easier to review when it is not interleaved with session construction and persistence writes.
- This file boundary matches how a new engineer would actually reason about the subsystem: evaluate, persist, observe, then lifecycle.

### What worked

- The refactor stayed mechanical.
- Focused package tests passed.
- The pre-commit hook reran lint and full `go test ./...` successfully.
- `service.go` dropped again, from 890 lines to 467 lines.

### What was tricky

- The only failed attempt in this slice was my first oversized patch. It mismatched the live file because `bindingViewFromState` had gained provenance fields that were not reflected in the removal hunk I prepared.
- I stopped, reread the file, and reapplied the extraction in smaller patches instead of forcing the edit.

### What I learned

- The file layout is now much closer to the intended mental model:
  - [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go): lifecycle and setup
  - [evaluate.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go): evaluation flow
  - [persistence.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/persistence.go): durable write shaping
  - [observe.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/observe.go): runtime observation and summaries
- At this point, the remaining cleanup work is no longer “split the giant file”; it is mostly naming and legacy-path decisions.

### Updated code review instructions

- Start with [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go) and confirm that it now reads as lifecycle/setup code.
- Review [evaluate.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go), [persistence.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/persistence.go), and [observe.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/observe.go) as three explicit responsibility areas.
- Validate with:
  - `go test ./pkg/replsession ./pkg/replapi`
  - `go test ./...`

### Step 4: Clarify the app-layer option boundary instead of breaking it

The review correctly identified that `replapi.SessionOptions` and `replsession.SessionOptions` are easy to confuse. After revisiting the boundary in code, I decided not to rename the public API in GOJA-042. That would create churn across callers for a cleanup ticket, and the package-qualified names already carry the layer distinction if the comments are explicit enough.

So the cleanup decision was:

- keep the public type names as they are
- improve comments to explain the layer split
- rename the internal helper from `resolveSessionOptions` to `resolveCreateSessionOptions`
- add a focused test to pin the app-layer override semantics

### Why this is the right cleanup level

- It improves readability without creating a migration chore for callers.
- It makes the app-layer type read as "create-session overrides" instead of "yet another full session config object."
- It leaves any future public rename as an explicit product/API decision instead of sneaking it into a refactor ticket.

### What I changed

- Updated [pkg/replapi/config.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go) comments to distinguish:
  - app-layer session creation overrides
  - kernel/session-layer default policy objects
- Renamed the internal helper in [pkg/replapi/config.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go) and [pkg/replapi/app.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go) from `resolveSessionOptions` to `resolveCreateSessionOptions`.
- Added [config_test.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config_test.go) to cover:
  - profile preset override behavior
  - preservation of explicit `ID` and `CreatedAt`
  - explicit policy override behavior

### Step 5: Decide the status of the Bobatea evaluator path

I rechecked the old evaluator path before accepting the earlier review claim that it looked deprecated. The code says otherwise.

Relevant usage points include:

- [cmd/goja-repl/tui.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/goja-repl/tui.go)
- [cmd/smalltalk-inspector/app/repl_widgets.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/smalltalk-inspector/app/repl_widgets.go)
- [pkg/repl/adapters/bobatea/javascript.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repl/adapters/bobatea/javascript.go)
- [pkg/repl/adapters/bobatea/replapi.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repl/adapters/bobatea/replapi.go)
- [pkg/repl/evaluators/javascript/evaluator.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go)

### Decision

- Do not mark this path deprecated.
- Do not remove or consolidate it inside GOJA-042.
- Document that it remains the Bobatea/TUI-facing evaluator surface, while `replsession` is the session-kernel path.

### Why

- It still has real call sites.
- It still has its own tests.
- It solves a different problem: TUI assistance/evaluator contracts rather than durable session lifecycle.

### What I changed

- Added clarifying comments to:
  - [pkg/repl/evaluators/javascript/evaluator.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go)
  - [pkg/repl/adapters/bobatea/replapi.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repl/adapters/bobatea/replapi.go)

### Final state after this session

- The giant `pkg/replsession/service.go` split work is done.
- The app-vs-kernel option naming confusion is reduced without a breaking rename.
- The Bobatea evaluator path has an explicit "kept and documented" decision.
- There is now a small test guarding the option-resolution boundary.
