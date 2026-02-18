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

## 2026-02-18

- Captured side-by-side CPU and allocation profiles for:
  - `BenchmarkRuntimeSpawn/EngineNew_NoCallLog`
  - `BenchmarkRuntimeSpawn/EngineFactory_NoCallLog`
- Generated SVG flamegraph-style call graphs and diff graph.
- Added head-to-head benchmark sample sets (default CPU, `-cpu=1`, and `GOGC=off`) for variability analysis.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/engine_new_cpu.svg — CPU call graph for EngineNew path
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/engine_factory_cpu.svg — CPU call graph for EngineFactory path
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/cpu_diff.svg — Differential CPU graph (factory minus new baseline)
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/alloc_diff_top.txt — Allocation delta summary
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/runtime_spawn_headtohead.txt — Multi-sample comparison run

## 2026-02-18

- Added controlled head-to-head benchmark comparison for runtime spawn paths using
  `benchstat` (`-cpu=1`, `-count=12`, `-benchtime=300ms`).
- Generated fresh 5-second CPU/memory profiles for both paths and produced
  additional SVG callgraph outputs and diff summaries.
- Added a structured YAML summary of benchmark and profiling outcomes.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/runtime_spawn_engine_new_cpu1_count12.txt — Baseline benchmark samples
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/runtime_spawn_engine_factory_cpu1_count12.txt — Factory benchmark samples
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/runtime_spawn_enginefactory_vs_new_benchstat_cpu1_count12.txt — Statistical delta report
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/engine_new_cpu_5s.svg — 5s CPU callgraph for EngineNew
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/engine_factory_cpu_5s.svg — 5s CPU callgraph for EngineFactory
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/cpu_diff_5s.svg — 5s differential CPU callgraph
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/runtime_spawn_profile_summary.yaml — Structured profile/benchmark summary
