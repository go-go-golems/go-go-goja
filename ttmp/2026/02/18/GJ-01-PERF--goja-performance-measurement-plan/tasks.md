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
