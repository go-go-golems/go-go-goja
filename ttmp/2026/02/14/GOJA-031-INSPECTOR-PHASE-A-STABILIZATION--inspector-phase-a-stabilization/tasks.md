# Tasks

## TODO

- [x] Create GOJA-031 ticket workspace
- [x] Add Phase A implementation plan and diary scaffold

- [x] Extract class/function member analysis from Bubble Tea model into `pkg/inspector/core`
- [x] Add cycle-safe inheritance traversal (visited set + depth guard) in core layer
- [x] Rewire `cmd/smalltalk-inspector` member building to call core APIs
- [x] Add core regression tests for self/indirect inheritance cycles and inherited-member behavior
- [x] Add command-level regression test to ensure no panic on cyclic inheritance input
- [x] Run validation suite (`go test ./cmd/smalltalk-inspector/...`, `go test ./pkg/inspector/...`, `go test ./...`)
- [x] Update GOJA-031 diary/changelog/index with outcomes and commit references
