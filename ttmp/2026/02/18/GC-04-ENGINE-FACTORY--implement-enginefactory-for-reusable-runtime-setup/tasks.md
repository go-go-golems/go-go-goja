# Tasks

## Done

- [x] Create ticket `GC-04-ENGINE-FACTORY`
- [x] Add implementation and design plan documents

## Planned

- [x] F1 Define `EngineFactory` API and lifecycle semantics
- [x] F2 Refactor engine bootstrap internals for reusable registry/module setup
- [x] F3 Implement `EngineFactory.NewRuntime()` with runtime-scoped calllog handling
- [x] F4 Keep `engine.Open(...)` and legacy constructors compatible while enabling factory path
- [x] F5 Add tests for correctness, calllog isolation, and require option behavior
- [x] F6 Add benchmarks comparing `engine.Open(...)` vs `EngineFactory.NewRuntime()`
- [x] F7 Update docs/examples to include factory usage
- [x] F8 Profile performance gains and generate flamegraph-style artifacts
