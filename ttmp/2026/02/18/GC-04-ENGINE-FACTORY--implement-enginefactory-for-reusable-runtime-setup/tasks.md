# Tasks

## Done

- [x] Create ticket `GC-04-ENGINE-FACTORY`
- [x] Add implementation and design plan documents

## Planned

- [ ] F1 Define `EngineFactory` API and lifecycle semantics
- [ ] F2 Refactor engine bootstrap internals for reusable registry/module setup
- [ ] F3 Implement `EngineFactory.NewRuntime()` with runtime-scoped calllog handling
- [ ] F4 Keep `engine.Open(...)` and legacy constructors compatible while enabling factory path
- [ ] F5 Add tests for correctness, calllog isolation, and require option behavior
- [ ] F6 Add benchmarks comparing `engine.Open(...)` vs `EngineFactory.NewRuntime()`
- [ ] F7 Update docs/examples to include factory usage
