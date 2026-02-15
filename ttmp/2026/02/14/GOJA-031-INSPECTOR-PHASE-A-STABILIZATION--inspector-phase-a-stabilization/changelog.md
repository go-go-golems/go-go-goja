# Changelog

## 2026-02-14

- Initial workspace created


## 2026-02-14

Step 1: Added Phase A implementation plan, task breakdown, and diary scaffold with explicit UI/core architecture boundary requirements.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-031-INSPECTOR-PHASE-A-STABILIZATION--inspector-phase-a-stabilization/design/01-phase-a-implementation-plan.md — Detailed plan for extraction + stabilization
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-031-INSPECTOR-PHASE-A-STABILIZATION--inspector-phase-a-stabilization/tasks.md — Executable task list
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-031-INSPECTOR-PHASE-A-STABILIZATION--inspector-phase-a-stabilization/reference/01-diary.md — Diary started


## 2026-02-14

Step 2: Extracted class/function member analysis from `cmd/smalltalk-inspector` into `pkg/inspector/core` and rewired UI model call sites to use core APIs.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/pkg/inspector/core/members.go — New UI-independent member analysis API
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/pkg/inspector/core/members_test.go — Core package tests for extraction behavior
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/model.go — Rewired to consume core package


## 2026-02-14

Step 3: Added depth guard to inherited-member traversal and expanded core regression tests to include deep inheritance chain bounds.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/pkg/inspector/core/members.go — Added `maxInheritanceDepth`-bounded recursion
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/pkg/inspector/core/members_test.go — Added deep-chain guard test


## 2026-02-14

Step 4: Added command-level regression tests for cyclic inheritance input and validated all relevant test suites.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/model_members_test.go — New command-level no-panic regressions
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-031-INSPECTOR-PHASE-A-STABILIZATION--inspector-phase-a-stabilization/reference/01-diary.md — Updated with command outputs and commit trace

## 2026-02-14

Ticket closed

