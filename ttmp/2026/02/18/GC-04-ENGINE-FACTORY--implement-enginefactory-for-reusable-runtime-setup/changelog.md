# Changelog

## 2026-02-18

- Created ticket workspace `GC-04-ENGINE-FACTORY`.
- Added implementation and design planning documents for EngineFactory.
- Added phased task list for API, implementation, testing, and benchmarking.

## 2026-02-18

- Commit `b93a0fa892f563bcf6a19bb6453fd4f2d6535560`: implemented `EngineFactory` and routed `engine.Open(...)` through factory-based runtime bootstrap.
- Added factory-path correctness tests and runtime spawn benchmark coverage.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/engine/factory.go — New EngineFactory implementation
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/engine/runtime.go — `Open(...)` compatibility routing via factory
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/engine/factory_test.go — Factory behavior tests
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/perf/goja/bench_test.go — EngineFactory runtime spawn benchmark case

## 2026-02-18

- Commit `52343302f45e77b2145c7f5d57c8cae789f96db1`: updated docs/examples to include EngineFactory usage guidance.
- Captured benchmark artifact comparing runtime spawn paths at:
  - `ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/runtime-spawn-enginefactory-bench.txt`

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/perf/goja/README.md — Benchmark scope now includes EngineFactory path
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/pkg/doc/bun-goja-bundling-playbook.md — Added factory usage example for repeated runtime creation
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/runtime-spawn-enginefactory-bench.txt — Recorded benchmark output
