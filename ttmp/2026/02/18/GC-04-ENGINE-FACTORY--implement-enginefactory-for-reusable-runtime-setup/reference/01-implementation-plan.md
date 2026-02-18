---
Title: Implementation Plan
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
    - Path: engine/options.go
      Note: Option model that factory should reuse
    - Path: engine/runtime.go
      Note: Existing Open constructor implementation
    - Path: perf/goja/README.md
      Note: Implementation plan references benchmark scope update
    - Path: pkg/calllog/calllog.go
      Note: Runtime logger binding behavior
    - Path: pkg/doc/bun-goja-bundling-playbook.md
      Note: Implementation plan references factory usage docs
ExternalSources: []
Summary: Implementation plan for adding EngineFactory with measurable runtime spawn improvements.
LastUpdated: 2026-02-18T10:16:32.5-05:00
WhatFor: Sequence implementation work for EngineFactory.
WhenToUse: Use while implementing and validating EngineFactory.
---


# Implementation Plan

## Goal

Implement an `EngineFactory` path that reduces per-runtime setup cost for
repeated runtime creation while preserving current behavior and compatibility.

## Context

`engine.Open(...)` currently builds runtime setup each call. This is clean but
expensive when creating many runtimes in loops, workers, or benchmark harnesses.
We already have option-based API and runtime-scoped calllog; EngineFactory
should reuse those semantics.

## Quick Reference

### Target milestones

1. Introduce `EngineFactory` type with constructor from `engine.Option`s.
2. Prebuild reusable setup state once (require registry config and module registration path).
3. Add `factory.NewRuntime()` to create a fresh runtime with lightweight per-runtime wiring.
4. Keep `engine.Open(...)` behavior intact; optionally route through a default factory internally.
5. Add tests and benchmarks proving correctness and performance delta.

### Validation commands

```bash
go test ./engine ./pkg/calllog -count=1
go test ./perf/goja -run '^$' -bench BenchmarkRuntimeSpawn -count=3 -benchtime=200ms
```

## Usage Examples

```go
factory := engine.NewFactory(
  engine.WithRequireOptions(require.WithLoader(loader)),
  engine.WithCallLogDisabled(),
)

vm, req := factory.NewRuntime()
_ = vm
_ = req
```

## Current Outcome

- Implemented in commit `b93a0fa892f563bcf6a19bb6453fd4f2d6535560`.
- Benchmark artifact captured at:
  - `ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/runtime-spawn-enginefactory-bench.txt`

## Related

- `reference/02-design-plan.md`
- `tasks.md`
