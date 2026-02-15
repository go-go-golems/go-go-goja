---
Title: Diary
Ticket: GOJA-028-CLEANUP-INSPECTOR
Status: active
Topics:
    - go
    - goja
    - tui
    - inspector
    - refactor
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/inspector/app/model.go
      Note: Audited for salvageable reusable patterns
    - Path: cmd/smalltalk-inspector/app/model.go
      Note: Critical crash path identified during review
    - Path: cmd/smalltalk-inspector/app/update.go
      Note: Interaction-state logic reviewed for cleanup risks
    - Path: cmd/smalltalk-inspector/app/view.go
      Note: Rendering/scrolling behavior analyzed in depth
    - Path: pkg/inspector/analysis/globals_merge.go
      Note: Extracted globals merge/sort policy for reuse
    - Path: pkg/inspector/analysis/repl_declarations.go
      Note: Extracted parser-backed REPL declaration analysis
    - Path: pkg/inspector/runtime/globals.go
      Note: Extracted runtime global enumeration helpers
    - Path: ttmp/2026/02/14/GOJA-028-CLEANUP-INSPECTOR--cleanup-inspector/reference/01-inspector-cleanup-review.md
      Note: Main review output produced in this execution
    - Path: ttmp/vocabulary.yaml
      Note: Added topic entries to satisfy doctor checks
ExternalSources: []
Summary: Execution diary for GOJA-028 review work, including inventory, validation, crash reproduction, and deliverable publication steps.
LastUpdated: 2026-02-14T18:52:00Z
WhatFor: Preserve command-level traceability and review workflow context for the cleanup ticket.
WhenToUse: Use when auditing how GOJA-028 findings were produced or reproducing the analysis workflow.
---


# Diary

## Step 1: Ticket setup and scope inventory

Created `GOJA-028-CLEANUP-INSPECTOR` and enumerated ticket workspaces from GOJA-024 through GOJA-027 to establish review scope.

Commands used:

```bash
docmgr ticket create-ticket --ticket GOJA-028-CLEANUP-INSPECTOR --title "Cleanup Inspector" --topics go,goja,tui,refactor,inspector
find go-go-goja/ttmp -maxdepth 4 -type d | rg 'GOJA-0(2[4-9]|3[0-9])' | sort
```

## Step 2: Commit and file-surface mapping

Collected commit history and file-level diffs since the pre-ticket baseline to understand exactly what was implemented.

Commands used:

```bash
git -C go-go-goja log --oneline -n 40
git -C go-go-goja diff --name-status b1e9add..HEAD
git -C go-go-goja diff --shortstat b1e9add..HEAD
```

## Step 3: Baseline validation

Ran tests and vet across the repository to establish current health before deeper findings analysis.

Commands used:

```bash
cd go-go-goja && go test ./... -count=1
cd go-go-goja && go vet ./...
```

Outcome:

- All tests passed.
- `go vet` passed.

## Step 4: Deep file inspection and targeted issue checks

Inspected key files in `cmd/smalltalk-inspector/app`, `pkg/inspector/runtime`, `pkg/inspector/analysis`, and `pkg/jsparse/highlight.go` with line-numbered reads.

Collected concrete evidence for architecture drift, scroll/render issues, parser/regex brittleness, and reuse gaps.

## Step 5: Critical crash reproduction

Validated a high-severity crash candidate: self-referential inheritance in class member traversal.

Method:

- Created a temporary test within `cmd/smalltalk-inspector/app` package to trigger `buildMembers()` on `class A extends A {}`.
- Ran `go test` with timeout.
- Observed runtime stack overflow with repeated frames in `addInheritedMembers`.

Representative command:

```bash
cd go-go-goja && timeout 8s go test ./cmd/smalltalk-inspector/app -run TestTmpSelfExtends -count=1
```

Result:

- `fatal error: stack overflow`
- Repeated recursion at `cmd/smalltalk-inspector/app/model.go` in `addInheritedMembers`.

Cleanup:

- Removed temporary test file immediately after reproduction.

## Step 6: Authored primary review deliverable

Wrote `reference/01-inspector-cleanup-review.md` with:

- severity-ranked findings first,
- architecture and implementation analysis,
- deprecated/unidiomatic items,
- reusable component extraction opportunities,
- phased cleanup roadmap and task backlog.

## Step 7: Ticket metadata/task/changelog updates

Updated:

- `tasks.md` with completed analysis items + actionable cleanup backlog.
- `index.md` with concrete summary and link map.
- `changelog.md` with execution steps and related files.

## Step 8: Validation and vocabulary hygiene

Validated frontmatter for key GOJA-028 docs and ran `docmgr doctor`.

Doctor initially reported unknown topics (`inspector`, `refactor`) in vocabulary.

Fixed by adding vocabulary entries:

```bash
docmgr vocab add --category topics --slug inspector --description "Inspector tooling and workflows"
docmgr vocab add --category topics --slug refactor --description "Codebase refactor and cleanup work"
docmgr doctor --ticket GOJA-028-CLEANUP-INSPECTOR --stale-after 30
```

Final doctor status:

- GOJA-028 checks passed.

## Step 9: Extracted general-purpose REPL/runtime merge logic into pkg and audited old inspector salvage

Implemented a targeted extraction pass focused only on logic that is genuinely reusable outside Bubble Tea (including possible LSP/CLI/REST integration), and deliberately avoided moving TUI-specific behavior. The main goals were:

1. remove heuristic REPL declaration parsing from UI layer,
2. centralize runtime global discovery and merge policy in `pkg/`,
3. fix REPL fallback syntax invalidation gap,
4. audit `cmd/inspector` for additional salvageable components.

Code changes made:

- Added parser-backed declaration extraction:
  - `pkg/inspector/analysis/repl_declarations.go`
- Added reusable globals merge/sort policy:
  - `pkg/inspector/analysis/globals_merge.go`
- Added runtime global discovery helpers:
  - `pkg/inspector/runtime/globals.go`
- Rewired app layer to use the new pkg APIs:
  - `cmd/smalltalk-inspector/app/model.go`
  - `cmd/smalltalk-inspector/app/update.go`
- Fixed fallback path to rebuild REPL syntax spans after runtime `toString()` append:
  - `cmd/smalltalk-inspector/app/model.go`

Added tests:

- `pkg/inspector/analysis/repl_declarations_test.go`
- `pkg/inspector/analysis/globals_merge_test.go`
- `pkg/inspector/runtime/globals_test.go`

Validation:

```bash
go test ./pkg/inspector/... -count=1
go test ./cmd/smalltalk-inspector/... -count=1
go test ./... -count=1
```

All passed.

### Old inspector salvage audit (`cmd/inspector`)

Reviewed old inspector architecture and identified components worth salvaging next:

1. Tree list + metadata table composition model (`bubbles/list` + `bubbles/table`) for hierarchical inspectors.
2. Drawer/editor split model (`drawer.go`) with completion popup interaction patterns.
3. Mode-keymap state machine patterns (already partially reused) for multi-pane navigation.
4. Cursor/source-to-tree sync logic (`syncSourceToTree`, `syncTreeToSource`) useful as a generic cross-pane synchronization pattern.
5. Status/help/command-line composition strategy for consistent inspector UX scaffolding.

Items **not** generally salvageable (or not worth immediate extraction):

1. Styling/layout specifics tightly coupled to old inspector visual design.
2. JS-parse bridge details that duplicate newer `pkg/inspector/*` abstractions.
3. Highly app-specific drawer behavior that doesn’t translate to LSP/server contexts.

Key decision:

- Keep extraction scope focused on domain/runtime logic first (now in `pkg/inspector/*`).
- Treat old-inspector UI salvage as a separate componentization pass, not mixed with core logic extraction.
