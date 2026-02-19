# Changelog

## 2026-02-18

- Created ticket `GJ-04-REQUIRE-HOTPATH` for targeted require-path performance investigation.
- Added investigation plan with hypotheses, command set, evidence thresholds,
  and decision criteria.
- Added detailed diary with Step 1 setup and rationale.
- Added investigation task checklist and ticket overview links.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/index.md — Ticket overview and related files
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/tasks.md — Investigation task plan
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/reference/01-require-performance-investigation-plan.md — Main investigation plan
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/reference/02-diary.md — Detailed investigation diary

## 2026-02-18

- Uploaded investigation plan to reMarkable:
  - `01-require-performance-investigation-plan.pdf` -> `/ai/2026/02/18/GJ-04-REQUIRE-HOTPATH`
- Completed first profiling pass for require cold vs warm behavior.
- Captured controlled benchmark stats plus CPU/memory profiles and SVG callgraph
  artifacts.
- Added structured YAML summary of current findings.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/require_cold_vs_warm_benchstat_cpu1_count8.txt — Statistical cold vs warm comparison
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/require_cold_cpu_5s.pprof — Cold-path CPU profile
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/require_warm_cpu_5s.pprof — Warm-path CPU profile
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/require_cold_warm_cpu_diff_5s.svg — Cold vs warm CPU diff callgraph
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/require-investigation-summary.yaml — Structured current-state findings

## 2026-02-18

- Added runtime-spawn correlation profile to connect require findings to
  `BenchmarkRuntimeSpawn/EngineNew_NoCallLog`.
- Captured focused require/console sample shares for runtime spawn:
  - CPU focused share: `20.81%`
  - alloc_objects focused share: `84.95%`
- Added candidate optimization experiments to the plan and marked investigation
  checklist complete for first pass.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/runtime_spawn_engine_new_cpu_5s.pprof — Runtime spawn CPU profile
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/runtime_spawn_engine_new_cpu_focus_require_console_5s.txt — Runtime spawn CPU focused share report
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/various/runtime_spawn_engine_new_mem_focus_require_console_5s.txt — Runtime spawn allocation focused share report
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/reference/01-require-performance-investigation-plan.md — Added candidate optimization experiments
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-04-REQUIRE-HOTPATH--investigate-require-performance-hot-path/tasks.md — Investigation checklist completion

## 2026-02-18

- Initial workspace created
