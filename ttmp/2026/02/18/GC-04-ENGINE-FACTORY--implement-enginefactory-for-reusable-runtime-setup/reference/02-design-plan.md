---
Title: Design Plan
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
    - Path: engine/runtime.go
      Note: Existing constructor flow to decompose
    - Path: modules/common.go
      Note: Module registration behavior and registry interactions
    - Path: /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/perf/goja/bench_test.go
      Note: Runtime spawn benchmark used to evaluate design
ExternalSources: []
Summary: Design plan for EngineFactory architecture and compatibility strategy.
LastUpdated: 2026-02-18T10:16:33.500000000-05:00
WhatFor: Capture architecture and tradeoffs for EngineFactory.
WhenToUse: Use when reviewing design choices before implementation.
---

# Design Plan

## Goal

Define a robust EngineFactory architecture that improves repeated runtime spawn
performance without breaking existing APIs.

## Context

Factory must preserve:
- fresh per-runtime VM isolation
- runtime-scoped calllog behavior
- require/module semantics
- backward compatibility for existing `engine.New*` and `engine.Open(...)` callers

## Quick Reference

### Proposed API

```go
type Factory struct {
  // internal reusable setup data
}

func NewFactory(opts ...Option) *Factory
func (f *Factory) NewRuntime() (*goja.Runtime, *require.RequireModule, error)
```

### Design constraints

- No shared mutable JS runtime state across created runtimes.
- No process-global calllog toggles as a side effect of runtime creation.
- Module registration side effects should happen at factory creation, not per runtime.
- `engine.Open(...)` remains primary simple API; factory is an advanced/perf API.

### Risks and mitigations

- Risk: shared module instance state leaks across runtimes.
  - Mitigation: audit module registration/loading paths and isolate mutable runtime bindings.
- Risk: hidden compatibility regressions in require behavior.
  - Mitigation: keep existing tests, add require-option parity tests for factory path.

## Usage Examples

```go
// Simple path stays available
vm, req := engine.Open(engine.WithCallLogDisabled())

// High-throughput path uses factory
factory := engine.NewFactory(engine.WithCallLogDisabled())
vm2, req2, err := factory.NewRuntime()
if err != nil {
  panic(err)
}
_ = vm
_ = req
_ = vm2
_ = req2
```

## Related

- `reference/01-implementation-plan.md`
- `tasks.md`
