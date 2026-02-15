# Changelog

## 2026-02-14

- Initial workspace created


## 2026-02-14

Step 1: Added detailed component alignment implementation plan with migration phases, target architecture, and test strategy.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-029-INSPECTOR-COMPONENT-ALIGNMENT--inspector-component-alignment/design/01-component-alignment-implementation-plan.md — Primary plan document
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-029-INSPECTOR-COMPONENT-ALIGNMENT--inspector-component-alignment/tasks.md — Task breakdown derived from plan


## 2026-02-14

Step 2: Implemented mode-keymap alignment in smalltalk-inspector key handling and mode updates; added diary baseline/test trace.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/keymap.go — Added keymap-mode tags and inspect/stack modes
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/update.go — Wired mode enablement in update transitions
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/model.go — Enabled initial mode-keymap state
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-029-INSPECTOR-COMPONENT-ALIGNMENT--inspector-component-alignment/reference/01-diary.md — Added implementation diary step

## 2026-02-14

Step 3: Migrated inspect/stack panes to viewport-backed rendering, added reusable row-visibility helper, and verified full regression suite.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/model.go — Added inspect/stack viewport state and pane-height helpers
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/update.go — Wired visibility maintenance and viewport offset resets
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/view.go — Migrated inspect/stack pane bodies to viewport rendering
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/internal/inspectorui/viewportpane.go — Reusable viewport row visibility helper
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-029-INSPECTOR-COMPONENT-ALIGNMENT--inspector-component-alignment/reference/01-diary.md — Recorded Step 2 implementation diary


## 2026-02-14

Step 4: Extracted reusable listpane helper for globals/members scroll windows and added listpane unit tests.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/update.go — Globals/members visibility now uses shared listpane helper
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/view.go — Globals/members render window now derived from shared helper
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/internal/inspectorui/listpane.go — Shared list pane selection visibility and visible-range logic
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/internal/inspectorui/listpane_test.go — Unit tests for listpane helper invariants
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-029-INSPECTOR-COMPONENT-ALIGNMENT--inspector-component-alignment/reference/01-diary.md — Recorded Step 3 implementation diary


## 2026-02-14

Step 5: Migrated source pane to viewport-backed scrolling and added shared viewport offset clamping helper.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/model.go — Added source viewport state and active-source helper
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/update.go — Source key handling now drives source viewport offset
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/view.go — Source pane now renders via viewport body
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/internal/inspectorui/viewportpane.go — Added shared viewport offset clamping
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-029-INSPECTOR-COMPONENT-ALIGNMENT--inspector-component-alignment/reference/01-diary.md — Recorded Step 4 implementation diary


## 2026-02-14

Step 6: Consolidated duplicated layout/status utility helpers into internal inspectorui utilities and rewired app callsites.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/model.go — Removed duplicated local helper implementations
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/update.go — Min/max callsites migrated to shared utilities
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/view.go — Padding and status formatting migrated to shared utilities
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/internal/inspectorui/util.go — Shared utility helpers for min/max
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/internal/inspectorui/util_test.go — Utility helper tests
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-029-INSPECTOR-COMPONENT-ALIGNMENT--inspector-component-alignment/reference/01-diary.md — Recorded Step 5 implementation diary


## 2026-02-14

Step 7: Added navigation/mode visibility tests for smalltalk-inspector and completed all GOJA-029 tasks with full regression validation.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/navigation_mode_test.go — New pane navigation and mode behavior tests
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-029-INSPECTOR-COMPONENT-ALIGNMENT--inspector-component-alignment/reference/01-diary.md — Recorded Step 6 completion diary
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-029-INSPECTOR-COMPONENT-ALIGNMENT--inspector-component-alignment/tasks.md — All remaining tasks checked complete

