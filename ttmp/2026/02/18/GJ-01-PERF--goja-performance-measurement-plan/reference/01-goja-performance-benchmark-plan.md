---
Title: Goja Performance Benchmark Plan
Ticket: GJ-01-PERF
Status: active
Topics:
    - goja
    - analysis
    - tooling
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../goja/compiler_test.go
      Note: Upstream compile benchmark reference
    - Path: ../../../../../../../goja/runtime_test.go
      Note: Upstream goja benchmark patterns for call boundaries
    - Path: ../../../../../../../goja/vm_test.go
      Note: Upstream VM benchmark reference
    - Path: engine/runtime.go
      Note: Runtime bootstrap path and calllog mode behavior
    - Path: modules/exports.go
      Note: JS->Go exported function wrapping path
    - Path: perf/goja/README.md
      Note: Benchmark execution and comparison runbook
    - Path: perf/goja/bench_test.go
      Note: Primary benchmark harness implemented for this ticket
    - Path: pkg/calllog/calllog.go
      Note: Call logging wrappers and bridge serialization overhead
ExternalSources: []
Summary: Implementation plan and benchmark architecture for measuring Goja and go-go-goja performance.
LastUpdated: 2026-02-18T13:45:00-05:00
WhatFor: Design and operate repeatable Goja performance measurement across runtime lifecycle, JS loading, and Go<->JS boundary calls.
WhenToUse: Use when adding or interpreting go-go-goja Goja benchmarks or planning perf regressions checks.
---


# Goja Performance Benchmark Plan

## Goal

Define a concrete, repeatable benchmark strategy for `go-go-goja` + upstream `goja` focused on:

- repeatedly spawning and executing code in a VM
- JS -> Go call overhead vs direct Go call overhead
- Go -> JS call overhead vs direct Go call overhead
- JS loading/compile/execute cost
- `require()` cold/warm loading behavior

This plan also establishes a dedicated repository section for benchmarks: `perf/goja`.

## Context

### Existing behavior that influences benchmark design

- `engine.New()` enables call logging by default via `DefaultRuntimeConfig()` and `NewWithConfig` (`engine/runtime.go:25`, `engine/runtime.go:37`).
- JS module exports go through `modules.SetExport`, which wraps functions with `calllog.WrapGoFunction` (`modules/exports.go:10`).
- `calllog.WrapGoFunction` and `calllog.CallJSFunction` both serialize arguments/results for log entries (`pkg/calllog/calllog.go:348`, `pkg/calllog/calllog.go:392`).
- Upstream `goja` already includes benchmark patterns for call boundaries, VM loops, and compile cost (`goja/runtime_test.go:3067`, `goja/compiler_test.go:5906`, `goja/vm_test.go:247`).

### Repository section created for this work

```text
perf/goja/
  bench_test.go
  README.md
```

`perf/goja/bench_test.go` is implemented and runnable now.

## Quick Reference

### Benchmark suites in `perf/goja/bench_test.go`

| Suite | Question | Core sub-benchmarks |
|---|---|---|
| `BenchmarkRuntimeSpawn` | Cost of runtime creation | `GojaNew`, `EngineNew_NoCallLog`, `EngineNew_WithCallLog` |
| `BenchmarkRuntimeSpawnAndExecute` | Cost of spawn + one execution | `RunString_FreshRuntime`, `RunProgram_FreshRuntime` |
| `BenchmarkRuntimeReuse` | Parse+exec vs precompiled exec in reused VM | `RunString_ReusedRuntime`, `RunProgram_ReusedRuntime` |
| `BenchmarkJSLoading` | JS load/compile cost by script size | `Compile_*`, `RunString_FreshRuntime_*`, `RunProgram_ReusedRuntime_*` |
| `BenchmarkJSCallingGo` | JS->Go boundary cost | `GoDirect`, `JS_vm_Set`, `JS_module_SetExport` |
| `BenchmarkGoCallingJS` | Go->JS boundary cost | `GoDirect`, `GojaAssertFunction`, `CallLogCallJSFunction_*` |
| `BenchmarkRequireLoading` | `require()` cold vs warm behavior | `ColdRequire_NewRuntime`, `WarmRequire_ReusedRuntime` |

### Standard execution protocol

1. Run stable samples with memory metrics:

```bash
go test ./perf/goja -run '^$' -bench . -benchmem -count=10 > /tmp/goja-perf-baseline.txt
```

2. After changes, rerun and compare:

```bash
go test ./perf/goja -run '^$' -bench . -benchmem -count=10 > /tmp/goja-perf-candidate.txt
benchstat /tmp/goja-perf-baseline.txt /tmp/goja-perf-candidate.txt
```

3. For quick functional smoke:

```bash
go test ./perf/goja -run '^$' -bench . -benchtime=1x -count=1
```

### Measurement dimensions and interpretation rules

- Primary metrics: `ns/op`, `B/op`, `allocs/op`
- Treat `<5%` deltas as noise unless they are persistent across repeated runs.
- Flag regressions when both are true:
  - `benchstat` confidence indicates a real delta.
  - regression appears in at least 2 independent runs.
- Always compare like-for-like mode:
  - call logging disabled vs disabled
  - call logging enabled vs enabled

## Implementation Plan

### Phase 1 (done in this ticket): establish benchmark foundation

- Create `perf/goja` benchmark package.
- Implement lifecycle, loading, boundary-call, and require-cache benchmarks.
- Add runbook in `perf/goja/README.md`.

### Phase 2: add deeper cost decomposition

- Add value conversion microbenchmarks:
  - `Runtime.ToValue` for primitives, structs, maps, arrays
  - `Value.Export` / `Runtime.ExportTo` for matching payloads
- Add payload-size sweeps for JS<->Go calls:
  - tiny payloads (2 ints)
  - medium payloads (map with 20 fields)
  - large payloads (nested map/slice payload)
- Add GC-sensitive scenarios:
  - long-lived reused VM under mixed allocation pressure
  - spawn-heavy VM churn

### Phase 3: operationalize for regression tracking

- Add Makefile target (e.g. `make perf-goja`) for standardized command invocation.
- Persist benchmark outputs under ticket artifacts or CI artifacts.
- Add periodic benchmark CI job (non-blocking first, then gate selected metrics when stable).

## Risks and Controls

- `engine.New()` default call logging can dominate measurements; benchmark both modes and do not mix them in comparisons.
- Startup logs can add noise; benchmark harness silences logging output.
- `-benchtime=1x` numbers are for sanity only, not decision-quality analysis.
- Full-system thermal/governor variance can distort small deltas; use repeated runs and benchstat.

## Usage Examples

Run only JS<->Go boundary benchmarks:

```bash
go test ./perf/goja -run '^$' -bench 'Benchmark(JSCallingGo|GoCallingJS)$' -benchmem -count=10
```

Run only loading/compile benchmarks:

```bash
go test ./perf/goja -run '^$' -bench 'Benchmark(JSLoading|RequireLoading)$' -benchmem -count=10
```

## Related

- Benchmark harness: `perf/goja/bench_test.go`
- Harness usage notes: `perf/goja/README.md`
- Runtime bootstrap and logging toggle: `engine/runtime.go`
- Module export wrapping: `modules/exports.go`
- Call logging wrappers: `pkg/calllog/calllog.go`
- Upstream benchmark references:
  - `../goja/runtime_test.go`
  - `../goja/compiler_test.go`
  - `../goja/vm_test.go`
