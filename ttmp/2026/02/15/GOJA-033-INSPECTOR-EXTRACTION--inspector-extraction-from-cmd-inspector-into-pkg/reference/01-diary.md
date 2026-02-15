---
Title: Diary
Ticket: GOJA-033-INSPECTOR-EXTRACTION
Status: active
Topics:
    - go
    - goja
    - inspector
    - refactor
    - tui
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/pkg/inspector/tree/rows.go
      Note: Extracted UI-agnostic tree row view-model shaping
    - Path: go-go-goja/pkg/inspector/navigation/sync.go
      Note: Extracted source-tree synchronization helpers
    - Path: go-go-goja/cmd/inspector/app/model.go
      Note: Old inspector rewired to consume extracted navigation helpers
    - Path: go-go-goja/cmd/inspector/app/tree_list.go
      Note: Old inspector rewired to consume extracted tree row helpers
ExternalSources: []
Summary: Step-by-step implementation diary for GOJA-033 extraction work.
LastUpdated: 2026-02-15T10:57:00-05:00
WhatFor: Preserve exact implementation decisions, errors, and validation during extraction.
WhenToUse: Use when reviewing or continuing GOJA-033 extraction.
---

# Diary

## Goal

Capture the GOJA-033 extraction sequence from ticket scaffolding through code extraction, testing, and documentation updates.

## Step 1: Extract Tree and Navigation Logic into pkg/inspector

This step moved old inspector logic out of `cmd/inspector/app` and into transport-agnostic packages under `pkg/inspector`. The extraction was intentionally narrow: row shaping and source/tree sync logic only.

The command-side code was kept as thin adapters so later API design can move orchestration into reusable services without unpicking Bubble Tea rendering code first.

### Prompt Context

**User prompt (verbatim):** "create a new ticket for the inspector extraction. Add implementation plan and tasks and upload to remarkable.

then work through the tasks with commit and diary at appropriate places as usual.

We'll do a nice api design + packaging pass once everything is extracted."

**Assistant interpretation:** Create GOJA-033, write plan/tasks, upload docs to reMarkable, then execute extraction tasks with commits and diary updates.

**Inferred user intent:** Finish concrete extraction work now, while preserving a clean handoff into a later API/packaging design pass.

**Commit (code):** `b49af1f` — "inspector: extract reusable tree and navigation helpers"

### What I did

- Replaced duplicated tree-row formatting logic in `cmd/inspector/app/tree_list.go` with a thin adapter over `pkg/inspector/tree.BuildRow`.
- Added `pkg/inspector/tree` package:
  - `rows.go` (`Row`, `BuildRowsFromIndex`, `BuildRow`)
  - `rows_test.go` (row rendering, usage hints, scope hints).
- Added `pkg/inspector/navigation` package:
  - `sync.go` (`SourceOffset`, `SelectionAtSourceCursor`, `FindVisibleNodeIndex`, `SelectionFromVisibleTree`)
  - `sync_test.go` (offset clamping, source selection, visible index lookup, tree selection mapping).
- Rewired `cmd/inspector/app/model.go`:
  - `syncSourceToTree` now uses `SelectionAtSourceCursor` + `FindVisibleNodeIndex`.
  - `syncTreeToSource` now uses `SelectionFromVisibleTree`.
  - removed local `sourceCursorOffset`.
- Added cmd-level regression test `TestModelTreeListItemIncludesUsageHint` in `cmd/inspector/app/model_test.go`.
- Ran validation:
  - `go test ./pkg/inspector/... -count=1`
  - `go test ./cmd/inspector/... -count=1`
  - `go test ./... -count=1`

### Why

- Keep domain logic reusable across future UI/CLI/REST/LSP surfaces.
- Remove duplicate implementations and centralize behavior in package-level tests.
- Preserve old inspector behavior while improving layering.

### What worked

- Extraction boundaries were stable and clean: no Bubble Tea/Bubbles/lipgloss dependencies entered new pkg code.
- Existing cmd behavior stayed intact once rewired.
- Added tests caught mismatches quickly and gave confidence for commit.

### What didn't work

- First run of `go test ./pkg/inspector/... -count=1` failed:
  - `pkg/inspector/tree/rows_test.go:35:46: n.Text undefined`
  - `pkg/inspector/tree/rows_test.go:60:46: n.Text undefined`
  - Cause: test used outdated field name; fixed by using `strings.Contains(n.DisplayLabel(), "x")`.
- First run of `go test ./cmd/inspector/... -count=1` failed:
  - `TestModelTreeListItemIncludesUsageHint`: usage node not visible.
  - Cause: deep node row not visible without expanding ancestors.
  - Fix: `m.index.ExpandTo(usageID)` + `m.refreshTreeVisible()` before assertion.
- Pre-commit hook `go generate ./...` logged Dagger pull timeout for `node:20.18.1` and fell back to local npm build automatically; commit still completed.

### What I learned

- The old inspector’s source/tree sync logic can be extracted cleanly without changing state model ownership.
- Command-level tests still need explicit visibility setup for non-root tree nodes, even when extraction logic is correct.

### What was tricky to build

- The main sharp edge was visibility coupling: `treeSelectedIdx` is over visible nodes only, so source-selected nodes require ancestor expansion before index lookup.
- This manifested as a false-negative test failure, not a runtime panic, which made the issue easy to miss without explicit visibility setup.

### What warrants a second pair of eyes

- Cursor/offset behavior around UTF-8 multibyte characters is still byte-oriented; verify expected semantics for future editor integrations.
- Tree row ordering assumptions depend on `VisibleNodes()` and should be re-checked if index traversal semantics change.

### What should be done in the future

- API/packaging pass: define service-level interfaces around navigation/tree logic so non-TUI adapters can consume the same commands.
- Consider additional edge-case tests for source offsets near EOF and empty last lines.

### Code review instructions

- Start here:
  - `pkg/inspector/tree/rows.go`
  - `pkg/inspector/navigation/sync.go`
  - `cmd/inspector/app/model.go`
  - `cmd/inspector/app/tree_list.go`
- Validate with:
  - `go test ./pkg/inspector/... -count=1`
  - `go test ./cmd/inspector/... -count=1`
  - `go test ./... -count=1`

### Technical details

- Extracted APIs are pure-value in/out helpers:
  - `tree.BuildRow(*NodeRecord, []NodeID, *Resolution) Row`
  - `navigation.SelectionAtSourceCursor(*Index, []string, int, int) (SourceSelection, bool)`
  - `navigation.SelectionFromVisibleTree(*Index, []NodeID, int) (TreeSelection, bool)`
- Command model remains owner of UI state (`focus`, `treeSelectedIdx`, viewport offsets, etc.); pkg APIs only compute selections and row DTOs.
