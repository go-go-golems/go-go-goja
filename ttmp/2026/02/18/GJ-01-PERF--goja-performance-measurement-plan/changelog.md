# Changelog

## 2026-02-18

- Created ticket workspace and reference docs for `GJ-01-PERF`.
- Added a dedicated benchmark section at `perf/goja/`.
- Implemented benchmark suites for VM spawn/execute, JS load cost, JS<->Go call overhead, Go->JS call overhead, and require cold/warm loading.
- Ran functional benchmark smoke validation with `go test ./perf/goja -run '^$' -bench . -benchtime=1x -count=1`.
- Authored implementation plan documenting benchmark matrix, execution protocol, and phased follow-up plan.

## 2026-02-18

Implemented perf/goja benchmark harness and documented phased performance measurement plan with diary notes.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/perf/goja/README.md — Benchmark runbook
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/perf/goja/bench_test.go — Initial benchmark suites
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/reference/01-goja-performance-benchmark-plan.md — Implementation plan
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/reference/02-diary.md — Investigation diary


## 2026-02-18

Uploaded bundled benchmark plan/diary package to reMarkable at /ai/2026/02/18/GJ-01-PERF.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/reference/01-goja-performance-benchmark-plan.md — Bundled in uploaded PDF
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/reference/02-diary.md — Bundled in uploaded PDF


## 2026-02-18

- Commit `039a49f959ebbec004cb4da6f90da760a1388fb8`: baseline performance ticket and benchmark harness landed; call logging default switched to disabled with guard test.
- Commit `deb40211326e13fc503c3ef6311353a78828a530`: added Glazed phase-1 runner commands and produced YAML task definitions + YAML run report + per-task raw outputs.
- Completed Phase-1 execution tasks P1-T1 through P1-T5 in `tasks.md`.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/cmd/goja-perf/main.go — CLI root for phase-1 runner commands
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/cmd/goja-perf/phase1_tasks_command.go — YAML task definition command
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/cmd/goja-perf/phase1_run_command.go — YAML execution/report command
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/various/phase1-task-definitions.yaml — Phase-1 command/flag definitions
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/various/phase1-run-results.yaml — Phase-1 structured run results

## 2026-02-18

Added Glazed phase-1 commands, generated YAML task/result artifacts, and recorded detailed incremental diary with commit hashes 039a49f and deb4021.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/cmd/goja-perf/phase1_run_command.go — Phase-1 execution command
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/cmd/goja-perf/phase1_tasks_command.go — Phase-1 task definitions command
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/reference/02-diary.md — Detailed implementation diary updates
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/various/phase1-run-results.yaml — Structured run output

