---
Title: Diary
Ticket: GC-04-ENGINE-FACTORY
Status: active
Topics:
    - go
    - refactor
    - tooling
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/changelog.md
      Note: Step milestones and artifacts
    - Path: ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/reference/01-implementation-plan.md
      Note: Implementation outcomes and profiling summary
    - Path: ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/runtime_spawn_profile_summary.yaml
      Note: Structured profiling result data
    - Path: perf/goja/bench_test.go
      Note: Benchmark cases profiled in this diary
ExternalSources: []
Summary: Diary for EngineFactory implementation and profiling verification.
LastUpdated: 2026-02-18T15:12:00-05:00
WhatFor: Track delivery and validation decisions for EngineFactory performance work.
WhenToUse: Use when reviewing implementation sequence, benchmark evidence, and profiling artifacts.
---


# Diary

## Goal

Record the implementation and profiling trail for `GC-04-ENGINE-FACTORY`,
including commit milestones, benchmark evidence, and flamegraph-style outputs.

## Step 1: Implement EngineFactory and Wire Compatibility Path

The first implementation segment introduced a reusable runtime factory while
keeping the existing constructor surface intact. The focus was to preserve
behavior while reducing repeated bootstrap overhead.

This delivered factory construction, runtime creation, tests, and benchmark
coverage in incremental commits.

### Prompt Context

**User prompt (verbatim):** "Tasks by task, implement, commit to git as you go."

**Assistant interpretation:** Execute GC-04 tasks incrementally and commit each
completed unit of work.

**Inferred user intent:** Get traceable, task-by-task delivery with minimal
risk and clear rollback points.

**Commit (code):** `b93a0fa892f563bcf6a19bb6453fd4f2d6535560` — "feat(engine): add EngineFactory runtime bootstrap path"

### What I did

- Added `engine/factory.go` with `NewFactory(...)` and `(*Factory).NewRuntime()`.
- Routed `engine.Open(...)` through the factory path in `engine/runtime.go`.
- Added `engine/factory_test.go` and benchmark case
  `BenchmarkRuntimeSpawn/EngineFactory_NoCallLog`.

### Why

- Avoid repeating expensive registry/module setup logic for high runtime churn.

### What worked

- Existing constructor entrypoints stayed compatible.
- Tests and benchmarks exercised both old and new runtime spawn paths.

### What didn't work

- N/A

### What I learned

- The new factory path primarily affects allocation pressure; wall-time impact
  needs repeated measurement to separate signal from noise.

### What was tricky to build

- Preserving compatibility while shifting internals required clean option
  mapping through `Open(...)` so legacy APIs kept equivalent behavior.

### What warrants a second pair of eyes

- Whether additional bootstrap work can move from `NewRuntime()` into factory
  construction without breaking runtime isolation semantics.

### What should be done in the future

- Extend profiling to isolate `require` and `console.Enable` cost in dedicated
  sub-benchmarks.

### Code review instructions

- Start with `engine/factory.go` and `engine/runtime.go`.
- Validate via `go test ./engine ./perf/goja -run '^$' -bench BenchmarkRuntimeSpawn`.

### Technical details

- Runtime-scoped calllog handling remains in per-runtime setup and is preserved
  in factory-created runtimes.

## Step 2: Profile Runtime Spawn Gains and Produce Flamegraph-Style Outputs

This step focused on the user's profiling request. I captured controlled A/B
benchmark samples, generated statistical deltas with `benchstat`, and exported
CPU profile SVG callgraphs plus diff graphs.

I also added a structured YAML summary to make the result set consumable by
tools and dashboards.

### Prompt Context

**User prompt (verbatim):** "ok, profile the perf gains then. Can you do a flamegraph even or something like that?"

**Assistant interpretation:** Quantify performance impact of EngineFactory and
produce flamegraph-like visual evidence.

**Inferred user intent:** Validate whether the design change materially improves
performance and inspect where time is spent.

**Commit (code):** Pending (profiling/docs commit in progress)

### What I did

- Ran controlled benchmark captures for each case:
  - `EngineNew_NoCallLog`
  - `EngineFactory_NoCallLog`
- Normalized benchmark names and used `benchstat` for direct A/B significance.
- Captured 5s CPU and mem profiles for each path and generated:
  - per-path SVG callgraphs
  - differential SVG callgraph
  - top and diff top text reports
- Added `runtime_spawn_profile_summary.yaml` for structured result data.

### Why

- Raw one-off benchmark lines are insufficient for confident conclusions under
  high runtime/GC noise.

### What worked

- Allocation improvements were stable and statistically significant.
- Flamegraph-style SVG artifacts were generated and stored under ticket
  `various/profiles/`.

### What didn't work

- CPU wall-time gain was not statistically significant in controlled run
  (`p=0.114`) and showed high variance.

### What I learned

- The dominant hotspots remain goja native function/object setup and require
  module load path; allocation reductions do not automatically convert into
  stable wall-time wins on this host.

### What was tricky to build

- `benchstat` requires aligned benchmark names across files for A/B mode; I
  solved this by normalizing benchmark names in copied result files.

### What warrants a second pair of eyes

- Whether additional runtime initialization can be cached safely, especially
  around module registration and console bootstrap.

### What should be done in the future

- Add targeted micro-benchmarks around `console.Enable` and registry setup to
  isolate the remaining dominant costs.

### Code review instructions

- Review structured summary:
  - `ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/runtime_spawn_profile_summary.yaml`
- Review flamegraph-style artifacts:
  - `ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/engine_new_cpu_5s.svg`
  - `ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/engine_factory_cpu_5s.svg`
  - `ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/cpu_diff_5s.svg`

### Technical details

- Interactive flamegraph view is available through:
  - `go tool pprof -http=:0 <profile.pprof>`
