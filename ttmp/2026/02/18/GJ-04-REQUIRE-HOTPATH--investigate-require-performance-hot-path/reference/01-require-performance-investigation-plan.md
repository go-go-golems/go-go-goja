---
Title: Require Performance Investigation Plan
Ticket: GJ-04-REQUIRE-HOTPATH
Status: active
Topics:
    - analysis
    - goja
    - tooling
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-perf/phase1_run_command.go
      Note: |-
        Reusable command execution path to gather benchmark evidence
        Benchmark execution/reporting pipeline for evidence collection
    - Path: cmd/goja-perf/phase1_types.go
      Note: |-
        Existing phase task grouping for loading/require measurements
        Task definitions that include BenchmarkRequireLoading
    - Path: engine/runtime.go
      Note: |-
        Runtime bootstrap flow likely invoking require and module enable paths
        Runtime bootstrap path where require setup cost appears
    - Path: perf/goja/bench_test.go
      Note: |-
        BenchmarkRequireLoading and runtime lifecycle benchmarks under investigation
        Require and runtime lifecycle benchmark source
    - Path: ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/runtime_spawn_profile_summary.yaml
      Note: |-
        Prior evidence that require/native loading appears in dominant stacks
        Prior profile evidence used for hypothesis framing
    - Path: ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/require-investigation-summary.yaml
      Note: Structured summary artifact for current evidence
    - Path: ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/require_cold_vs_warm_benchstat_cpu1_count8.txt
      Note: Controlled statistical result supporting interim findings
    - Path: ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/runtime_spawn_engine_new_cpu_focus_require_console_5s.txt
      Note: Runtime-spawn focused require/console CPU share evidence
    - Path: ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/runtime_spawn_engine_new_mem_focus_require_console_5s.txt
      Note: Runtime-spawn focused allocation share evidence
ExternalSources: []
Summary: Hypothesis-driven plan to validate whether require/module bootstrap is the dominant runtime initialization bottleneck.
LastUpdated: 2026-02-18T17:00:00-05:00
WhatFor: Define repeatable investigation steps, commands, and evidence criteria for require hot-path analysis.
WhenToUse: Use while running or reviewing require performance profiling sessions.
---




# Require Performance Investigation Plan

## Goal

Determine with high confidence whether `require` path work (registry setup, native
module loading, `console.Enable`) is the main performance culprit in runtime
spawn/initialization, and quantify how much of total cost it contributes.

## Context

- Existing GC-04 profiling shows hotspots in:
  - `goja_nodejs/require.(*RequireModule).loadNative`
  - `goja_nodejs/console.Enable`
  - goja native function/object setup and GC scanning
- Existing phase-1 benchmark suite already contains `BenchmarkRequireLoading`,
  so we can start without adding new benchmark code.
- Current question: whether require is the dominant root cause versus a secondary
  contributor behind broader runtime bootstrap work.

## Quick Reference

### Investigation Hypotheses

1. H1: Cold runtime require/module bootstrap is a major share of spawn cost.
2. H2: Warm require path is materially cheaper; cold-vs-warm delta is large.
3. H3: Even with require, non-require work (goja object setup + GC) remains a
   significant floor.

### Evidence Thresholds

1. Strong support for "culprit":
- require-related stacks appear consistently in top cumulative CPU/allocation
  contributors across controlled runs.
- removing/bypassing require path yields a meaningful runtime spawn drop.

2. Partial support:
- require is clearly non-trivial, but not dominant over goja core setup/GC.

### Command Set (initial pass)

```bash
# 1) Focused benchmark for require loading behavior
go test ./perf/goja -run '^$' -bench '^BenchmarkRequireLoading$' -benchmem -count=8 -benchtime=300ms

# 2) Focused cpu/mem profiles for require loading benchmark
go test ./perf/goja -run '^$' -bench '^BenchmarkRequireLoading$' -benchmem -count=1 -benchtime=5s \
  -cpuprofile ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/require_loading_cpu.pprof \
  -memprofile ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/require_loading_mem.pprof

# 3) Compare against runtime spawn benchmark for context
go test ./perf/goja -run '^$' -bench '^BenchmarkRuntimeSpawn/(EngineNew_NoCallLog|EngineFactory_NoCallLog)$' -benchmem -cpu=1 -count=8 -benchtime=300ms

# 4) Extract top stacks and svg callgraphs
go tool pprof -top ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/require_loading_cpu.pprof
go tool pprof -sample_index=alloc_objects -top ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/require_loading_mem.pprof
go tool pprof -svg ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/require_loading_cpu.pprof > \
  ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/require_loading_cpu.svg
```

### Data to Capture

1. Raw benchmark outputs (`.txt`) for each run.
2. CPU/memory profiles (`.pprof`) and derived outputs (`.svg`, `*_top.txt`).
3. A concise YAML or markdown summary of:
- cold vs warm require deltas
- require stack share in profiles
- confidence level and caveats

### Decision Outcomes

1. If require is dominant:
- open optimization ticket focused on require bootstrap (registry/module cache strategy).

2. If require is not dominant:
- prioritize broader runtime bootstrap and allocation/GC optimization plan.

## Interim Findings (2026-02-18)

1. Controlled cold-vs-warm require comparison indicates a very large gap:
- `69.094us` -> `1.051us` (`-98.48%`, `p=0.000`)
- `34655 B/op` -> `552 B/op` (`-98.41%`, `p=0.000`)
- `500 allocs/op` -> `10 allocs/op` (`-98.00%`, `p=0.000`)

2. Cold-path profiles show substantial require-related work:
- `RequireModule.Require` cumulative ~`33%`
- `RequireModule.resolve` cumulative ~`33%`
- `RequireModule.loadModuleFile` cumulative ~`25%`
- `Registry.getCompiledSource` cumulative ~`18%`

3. Conclusion so far:
- `require` is a major culprit for cold require/runtime paths.
- It is not the only cost center: parser, allocation churn, and GC are still
  meaningful contributors in cold path.

## Candidate Optimization Experiments

1. Precompiled source cache for `require` loader inputs
- Goal: avoid repeated parse/compile of stable module text across runtimes.
- Measurement: rerun `BenchmarkRequireLoading/ColdRequire_NewRuntime` and compare
  `ns/op`, `B/op`, `allocs/op`, plus parser/require stack share in CPU/alloc profiles.

2. Registry/bootstrap template reuse
- Goal: move more module/registry initialization out of per-runtime hot path.
- Measurement: compare `BenchmarkRuntimeSpawn/EngineNew_NoCallLog` against
  template-enabled path and inspect require/console-focused sample share.

3. Optional lean runtime mode (without `console.Enable` for non-console workloads)
- Goal: validate cost of always enabling console-related native modules.
- Measurement: spawn benchmark and require benchmark with and without console setup.

4. Path-cleaning and resolution micro-optimization
- Goal: reduce warm-path `path.Clean`/`path.Join`/resolution overhead visible in
  warm profiles.
- Measurement: `BenchmarkRequireLoading/WarmRequire_ReusedRuntime` throughput and
  focused profile deltas.

## Usage Examples

### Example Research Session

1. Run require benchmark with high enough sample count to reduce noise.
2. Capture 5s profile for stable stack distribution.
3. Cross-check with runtime spawn profile to quantify require share.
4. Record findings in diary with exact command lines and timestamps.

### Example Handoff Statement

"Require is/is not the dominant contributor because X% of cumulative CPU and Y%
of alloc_objects are in require/console paths across N controlled runs."

## Related

- `reference/02-diary.md`
- `ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/reference/01-implementation-plan.md`
