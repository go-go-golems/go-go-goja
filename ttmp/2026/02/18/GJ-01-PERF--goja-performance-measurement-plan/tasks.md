# Tasks

## Done

- [x] Create ticket `GJ-01-PERF` and initialize docs workspace
- [x] Analyze `go-go-goja` runtime/module/calllog paths and upstream `goja` benchmarks
- [x] Create dedicated benchmark section in repository: `perf/goja/`
- [x] Implement initial benchmark suites in `perf/goja/bench_test.go`
- [x] Write implementation plan and investigation diary
- [x] Upload bundled ticket deliverable to reMarkable

## Next

- [ ] Add payload-size sweep benchmarks for JS<->Go boundary calls
- [ ] Add value conversion benchmarks (`ToValue`, `Export`, `ExportTo`)
- [ ] Add stable CI/perf artifact workflow (bench outputs + benchstat diff)

## Phase 1 (Execution Tasks)

- [x] P1-T1 Define Glazed command/flag definitions for phase-1 performance execution
- [x] P1-T2 Emit phase-1 task definitions as YAML (command + flags only)
- [x] P1-T3 Execute phase-1 benchmark task set via the new command
- [x] P1-T4 Persist phase-1 run results as YAML under ticket artifacts
- [x] P1-T5 Record detailed diary entries and changelog updates with commit hashes

## Phase 2 (Execution Tasks)

- [x] P2-T1 Define Glazed command/flag definitions for phase-2 performance execution
- [x] P2-T2 Implement phase-2 benchmark suites (payload sweeps, value conversion, GC sensitivity)
- [x] P2-T3 Emit phase-2 task definitions as YAML (command + flags only)
- [x] P2-T4 Execute phase-2 benchmark task set via the new command
- [x] P2-T5 Persist phase-2 run results as YAML under ticket artifacts
- [x] P2-T6 Record detailed diary entries and changelog updates with commit hashes
