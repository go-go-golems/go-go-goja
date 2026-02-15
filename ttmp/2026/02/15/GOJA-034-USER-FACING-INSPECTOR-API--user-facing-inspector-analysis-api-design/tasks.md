# Tasks

## TODO

- [x] Map current analysis/inspection capabilities and package boundaries
- [x] Design user-facing API options (library-first, service-first, hybrid)
- [x] Write detailed onboarding and decision guide with pseudocode and API sketches (8+ pages)
- [x] Write GOJA-034 phase-A hybrid implementation plan with clean-cutover assumptions
- [ ] Create `pkg/inspectorapi` service foundation (contracts, registry, open/update/close)
- [ ] Implement static analysis facade methods (globals, members, declaration lookups)
- [ ] Implement runtime merge and REPL declaration facade helpers
- [ ] Implement navigation/tree facade methods (sync + rows)
- [ ] Cut over `cmd/smalltalk-inspector/app` static orchestration to `pkg/inspectorapi`
- [ ] Add/adjust tests for new facade and smalltalk cutover behavior
- [ ] Run validation suite (`go test ./pkg/inspectorapi/...`, `go test ./cmd/smalltalk-inspector/...`, `go test ./...`)
- [ ] Update GOJA-034 changelog/index/diary/tasks with commit-linked progress
