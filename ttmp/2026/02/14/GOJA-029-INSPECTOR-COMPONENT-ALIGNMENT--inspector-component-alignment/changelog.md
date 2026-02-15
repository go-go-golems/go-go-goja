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

