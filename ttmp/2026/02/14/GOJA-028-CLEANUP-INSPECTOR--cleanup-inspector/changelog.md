# Changelog

## 2026-02-14

- Initial workspace created


## 2026-02-14

Step 1: Inventoried GOJA-024 through GOJA-027 tickets and commit history, then mapped all implementation files changed since the pre-ticket baseline.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-024-SMALLTALK-INSPECTOR--smalltalk-inspector/index.md — Baseline design ticket context
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-025-INSPECTOR-BUBBLES-REFACTOR--inspector-bubbles-component-refactor/reference/01-inspector-refactor-design-guide.md — Reusable component baseline context
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-026-INSPECTOR-BUGS--smalltalk-inspector-bug-fixes-empty-members-and-repl-definitions/reference/02-bug-report.md — Bug-fix wave context
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-027-SYNTAX-HIGHLIGHT--syntax-highlighting-for-smalltalk-inspector-source-pane/index.md — Syntax-highlighting ticket context


## 2026-02-14

Step 2: Validated the current codebase with tests/vet and reproduced a critical stack-overflow crash for self-referential inheritance during member-building recursion.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/model.go — Recursion path (`addInheritedMembers`) responsible for stack overflow
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/update.go — Runtime behavior and state transition review
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/view.go — Rendering and scrolling behavior review


## 2026-02-14

Step 3: Authored comprehensive cleanup review document with severity-ranked findings, architecture map, reuse/refactor opportunities, and phased remediation roadmap.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-028-CLEANUP-INSPECTOR--cleanup-inspector/reference/01-inspector-cleanup-review.md — Primary review deliverable
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-028-CLEANUP-INSPECTOR--cleanup-inspector/tasks.md — Cleanup backlog derived from findings
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-028-CLEANUP-INSPECTOR--cleanup-inspector/index.md — Ticket overview and navigation links


## 2026-02-14

Step 4: Validated frontmatter and doctor checks for GOJA-028, then added missing topic vocabulary entries (`inspector`, `refactor`) so the ticket passes hygiene checks.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/vocabulary.yaml — Added topics `inspector` and `refactor`
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-028-CLEANUP-INSPECTOR--cleanup-inspector/index.md — Topic metadata validated successfully

## 2026-02-15

Cross-ticket update: GOJA-028 task #13 (analysis-layer integration) has been executed in GOJA-032-ANALYSIS-INTEGRATION with implementation guide, staged refactor commits, and regression validation.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/15/GOJA-032-ANALYSIS-INTEGRATION--smalltalk-inspector-analysis-session-integration/design/01-implementation-guide-integrate-pkg-inspector-analysis-into-smalltalk-inspector.md — Execution plan and architecture for the extracted task
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/15/GOJA-032-ANALYSIS-INTEGRATION--smalltalk-inspector-analysis-session-integration/tasks.md — Task-level execution tracking for the extracted work


## 2026-02-15

Backlog reconciliation update: marked completed cleanup items delivered via GOJA-029/031/032 and closed GOJA-027 hygiene gap after doctor validation.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-027-SYNTAX-HIGHLIGHT--syntax-highlighting-for-smalltalk-inspector-source-pane/changelog.md — GOJA-027 closure and hygiene validation
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-027-SYNTAX-HIGHLIGHT--syntax-highlighting-for-smalltalk-inspector-source-pane/tasks.md — Final GOJA-027 task completion
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-028-CLEANUP-INSPECTOR--cleanup-inspector/tasks.md — Reconciled completed-vs-open backlog tasks


## 2026-02-15

Extracted reusable REPL/global merge logic into pkg/inspector (analysis + runtime), rewired smalltalk-inspector to consume pkg APIs, fixed REPL fallback syntax-span rebuild path, and audited cmd/inspector salvage opportunities.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/model.go — Fallback syntax span rebuild and pkg API adoption
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/update.go — Parser-backed REPL declaration tracking
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/pkg/inspector/analysis/globals_merge.go — General merge/sort policy for globals
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/pkg/inspector/analysis/repl_declarations.go — Parser-backed declaration extraction reusable by CLI/LSP
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/pkg/inspector/runtime/globals.go — Runtime global classification primitives

