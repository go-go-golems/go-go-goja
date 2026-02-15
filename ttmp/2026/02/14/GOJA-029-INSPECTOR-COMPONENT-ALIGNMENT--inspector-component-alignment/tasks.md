# Tasks

## TODO

- [x] Create GOJA-029 ticket workspace
- [x] Add component-alignment implementation plan document

- [x] Baseline: capture current interaction behavior and test outputs before refactor
- [x] Implement mode-keymap alignment for smalltalk-inspector key handling and help wiring
- [x] Extract reusable list pane helper and migrate globals/members panes
- [x] Extract reusable viewport pane helper and migrate inspect/stack panes
- [x] Ensure inspect/stack selected row visibility with explicit scroll offsets
- [x] Migrate source pane to shared viewport helper where feasible
- [ ] Consolidate duplicated helper logic (`padRight`, min/max, status formatting) into shared utilities
- [ ] Add/expand tests for pane navigation, visibility, and mode-specific key behavior
- [x] Run full regression suite (`cmd/inspector`, `cmd/smalltalk-inspector`, `pkg/inspector`, full repo)
- [x] Update docs/changelog with migration notes and residual follow-ups

## Done Criteria

- [ ] Smalltalk inspector uses shared component primitives for key mode/list/viewport/status
- [ ] Finding #3 visibility issue is resolved and covered by tests
- [ ] No regression in existing `cmd/inspector` behavior
