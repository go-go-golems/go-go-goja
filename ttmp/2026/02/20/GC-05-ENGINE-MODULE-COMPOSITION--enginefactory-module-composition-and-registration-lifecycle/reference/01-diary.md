---
Title: Diary
Ticket: GC-05-ENGINE-MODULE-COMPOSITION
Status: active
Topics:
    - go
    - architecture
    - refactor
    - tooling
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/bun-demo/main.go
      Note: Demo command migrated to builder/factory runtime flow
    - Path: go-go-goja/cmd/repl/main.go
      Note: REPL command migrated to owned runtime construction and close
    - Path: go-go-goja/engine/factory.go
      Note: New builder/factory API with explicit Build and NewRuntime lifecycle
    - Path: go-go-goja/engine/factory_test.go
      Note: Factory tests moved to builder-based API and validation
    - Path: go-go-goja/engine/module_specs.go
      Note: RuntimeContext
    - Path: go-go-goja/engine/options.go
      Note: Option pipeline renamed for builder settings
    - Path: go-go-goja/engine/runtime.go
      Note: Owned runtime struct and explicit Close semantics
    - Path: go-go-goja/engine/runtime_test.go
      Note: Runtime tests updated for explicit factory runtime creation
    - Path: go-go-goja/perf/goja/bench_test.go
      Note: Bench helper migrated to new runtime API and cleanup closure
    - Path: go-go-goja/perf/goja/phase2_bench_test.go
      Note: Remaining bench callsites updated for 3-value runtime helper and close handling
    - Path: go-go-goja/pkg/repl/evaluators/javascript/evaluator.go
      Note: Evaluator reset now closes/recreates owned runtime
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-21T15:47:38.4626097-05:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Track the implementation work for the GC-05 no-compat EngineFactory rewrite, including API cut decisions, file-by-file changes, and commit-by-commit progress.

## Step 1: Expand ticket tasks for execution mode (no wrappers)

I converted the existing research backlog into an execution-oriented implementation checklist that matches the current direction: full API rewrite, no backwards compatibility wrappers, no dependency solver in this iteration.

This step intentionally front-loads scope discipline so subsequent code commits can be small and auditable against explicit milestones.

### Prompt Context

**User prompt (verbatim):** "go ahead, add detailed tasks to the ticket (no backwards compatibility / wrapper), and then do the tasks one by one, committing as you go and continuing keeping your diary"

**Assistant interpretation:** First update ticket tasks with a concrete no-compat plan, then start implementation with incremental commits while recording diary entries.

**Inferred user intent:** Execute the rewrite immediately with transparent progress tracking and minimal ambiguity about scope.

### What I did

- Updated `tasks.md` to a concrete no-compat sequence:
  - new builder and module/runtime contracts
  - owned runtime lifecycle
  - deletion of legacy API paths
  - migration of all in-repo call sites
  - test/doc updates
- Created this diary document and added the first step entry.

### Why

- The previous task list was architecture-research-oriented and included dependency solver work not required for the current implementation pass.
- Explicit no-wrapper tasks are necessary to prevent scope drift.

### What worked

- Task list now maps directly to executable code milestones.

### What didn't work

- N/A.

### What I learned

- Converting design goals to actionable tasks early reduces churn when doing multi-commit refactors.

### What was tricky to build

- Ensuring task wording is explicit about removals (legacy APIs) and not interpreted as additive/compat behavior.

### What warrants a second pair of eyes

- Confirm that task order is acceptable before irreversible API removals land.

### What should be done in the future

- Keep updating this diary after each code commit with exact files and rationale.

### Code review instructions

- Review `tasks.md` to confirm implementation order and no-compat policy are explicit.

### Technical details

- Ticket path:
  - `go-go-goja/ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/`

## Step 2: Land no-compat engine API rewrite and migrate call sites

I replaced the previous mixed constructor/wrapper model with a single builder -> factory -> runtime flow and migrated active callsites to use owned runtime lifecycle explicitly. This commit makes API shape deliberate: module registration is an explicit composition step, and runtime creation is a factory responsibility.

This step also captured a real migration break in perf benchmarks, then fixed it before finalizing the commit. The failure was useful because it validated that old signatures were fully removed and stale usage was surfaced by tooling.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement the next task batch directly, commit incrementally, and keep ticket diary/changelog updated with concrete technical details.

**Inferred user intent:** Drive execution instead of planning, with auditable progress and no backwards-compatibility detours.

**Commit (code):** `4059db5` — "GC-05: replace legacy runtime wrappers with builder and runtime ownership"

### What I did

- Reworked engine API surface around:
  - `NewBuilder(...)`
  - fluent builder methods for require options, module specs, and runtime initializers
  - `Build()` producing immutable factory
  - `factory.NewRuntime(ctx)` returning owned runtime object
- Added module composition contracts and explicit default-registry module spec.
- Removed legacy runtime wrapper entry points and moved runtime teardown into explicit `Close(ctx)`.
- Migrated callsites in demo/repl/evaluator/perf packages to the new API.
- Updated engine and runtime tests to use builder/factory/runtime flow.
- Fixed a benchmark typecheck break in `phase2_bench_test.go` caused by helper signature migration.

### Why

- The previous API mixed construction responsibilities and made module composition less explicit.
- No-compat mode requires direct cutover so stale API usage fails fast and is corrected immediately.

### What worked

- Compile-time migration pressure caught remaining legacy call patterns.
- Pre-commit lint/test hooks validated end-to-end engine + callsite consistency after fixes.

### What didn't work

- First commit attempt failed in pre-commit lint due old benchmark helper usage:
  - Command: `git commit -m "GC-05: replace legacy runtime wrappers with builder and runtime ownership"`
  - Error excerpts:
    - `perf/goja/phase2_bench_test.go:191:13: assignment mismatch: 2 variables but newRuntime returns 3 values`
    - `perf/goja/phase2_bench_test.go:203:12: assignment mismatch: 2 variables but newRuntime returns 3 values`
    - `perf/goja/phase2_bench_test.go:217:12: assignment mismatch: 2 variables but newRuntime returns 3 values`
- Resolution:
  - Updated those callsites to accept cleanup closure and invoke/defer runtime close.
  - Re-ran commit; hooks passed.

### What I learned

- Removing wrappers early is effective only if benchmark/test helpers are migrated in the same batch; otherwise API breakage appears as noisy late failures.
- Explicit runtime ownership improves benchmark hygiene by preventing leaked event loop/runtime objects.

### What was tricky to build

- The sharp edge was mixed helper signatures across benchmark files. The root cause was partial migration of `newRuntime` consumers; symptoms were typecheck failures in pre-commit lint.
- I addressed it by standardizing all benchmark callsites to the same `(vm, req, close)` contract and adding close calls where missing.

### What warrants a second pair of eyes

- `pkg/repl/evaluators/javascript/evaluator.go` reset semantics now close/recreate owned runtime; review for lifecycle correctness under repeated reset/use.
- `engine/module_specs.go` default registry module strategy should be reviewed for desired default surface area.

### What should be done in the future

- Add a focused design follow-up on dependency resolution hooks and optional DAG validation in builder phase.

### Code review instructions

- Start with:
  - `engine/factory.go`
  - `engine/module_specs.go`
  - `engine/runtime.go`
  - `pkg/repl/evaluators/javascript/evaluator.go`
- Then inspect migrated callsites:
  - `cmd/repl/main.go`
  - `cmd/bun-demo/main.go`
  - `perf/goja/bench_test.go`
  - `perf/goja/phase2_bench_test.go`
- Validate by re-running repo hooks (lint + test) on staged files.

### Technical details

- New core flow:
  - `builder := engine.NewBuilder()`
  - `factory, _ := builder.WithModules(engine.DefaultRegistryModules()).Build()`
  - `rt, _ := factory.NewRuntime(ctx)`
  - `defer rt.Close(ctx)`
