# Tasks

## TODO

- [x] Create GOJA-033 ticket workspace
- [x] Add GOJA-033 implementation plan document
- [x] Add GOJA-033 diary scaffold
- [x] Upload implementation plan + tasks to reMarkable
- [x] Extract tree row/view-model shaping from `cmd/inspector/app/tree_list.go` into `pkg/inspector/tree`
- [x] Add unit tests for extracted `pkg/inspector/tree` behavior
- [x] Extract source/tree sync helpers from `cmd/inspector/app/model.go` into `pkg/inspector/navigation`
- [x] Add unit tests for `pkg/inspector/navigation` offset/sync helpers
- [x] Rewire `cmd/inspector/app` to consume extracted tree/navigation helpers
- [x] Add/adjust command-level regression tests to guard behavior parity
- [x] Run validation suite (`go test ./pkg/inspector/...`, `go test ./cmd/inspector/...`, `go test ./...`)
- [x] Update GOJA-033 diary/changelog/index/tasks with commit-linked progress

## Handoff Checklist

- [x] Extracted pkg code has no Bubble Tea/Bubbles/lipgloss imports
- [x] Old inspector behavior is preserved (tested)
- [x] Remaining non-extracted surfaces are documented for later API packaging pass
