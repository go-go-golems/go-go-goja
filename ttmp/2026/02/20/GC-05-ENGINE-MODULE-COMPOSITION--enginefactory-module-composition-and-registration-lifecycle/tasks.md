# Tasks

## TODO

- [x] Finalize no-compat rewrite scope and freeze migration policy (explicitly no wrappers, no legacy New/Open path).
- [x] Introduce canonical builder API in `engine`:
  - `NewBuilder(...)`
  - `WithRequireOptions(...)`
  - `WithModules(...)`
  - `WithRuntimeInitializers(...)`
  - `Build()`
- [x] Implement module contracts without dependency solver:
  - `ModuleSpec` with stable `ID()` and `Register(*require.Registry)`.
  - `RuntimeInitializer` with stable `ID()` and `InitRuntime(*RuntimeContext)`.
  - duplicate-ID fail-fast validation for both sets.
- [x] Implement immutable built `Factory` and owned runtime lifecycle:
  - `Factory.NewRuntime(ctx)` returns `*Runtime` (VM, require, loop, owner runner).
  - `Runtime.Close(ctx)` shuts down owner and event loop safely.
- [x] Remove legacy runtime creation APIs and implicit module path:
  - delete `engine.New()`, `engine.NewWithOptions(...)`, `engine.Open(...)`, and implicit `modules.EnableAll(...)` behavior.
  - keep only explicit composition through builder.
- [x] Migrate all in-repo call sites to new API:
  - `cmd/repl`
  - `cmd/bun-demo`
  - `pkg/repl/evaluators/javascript`
  - engine tests and perf benches.
- [x] Update tests for new API surface (no dependency solver assertions, only duplicate-ID and lifecycle semantics).
- [x] Update GC-05 ticket docs:
  - diary entries per implementation step/commit.
  - changelog entries tied to concrete code changes.
