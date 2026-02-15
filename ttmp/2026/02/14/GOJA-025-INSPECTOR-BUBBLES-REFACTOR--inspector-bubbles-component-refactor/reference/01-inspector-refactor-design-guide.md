---
Title: Inspector Refactor Design Guide
Ticket: GOJA-025-INSPECTOR-BUBBLES-REFACTOR
Status: active
Topics:
    - go
    - goja
    - tui
    - tooling
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/inspector/app/drawer_test.go
      Note: Drawer/completion baseline tests
    - Path: go-go-goja/cmd/inspector/app/keymap.go
      Note: Mode-aware keymap and help metadata
    - Path: go-go-goja/cmd/inspector/app/model.go
      Note: Primary implementation file for component integration
    - Path: go-go-goja/cmd/inspector/app/model_test.go
      Note: Behavior baseline tests
    - Path: go-go-goja/cmd/inspector/app/tree_list.go
      Note: List adapter for AST tree pane
ExternalSources: []
Summary: Detailed implementation guide for refactoring cmd/inspector to reusable Bubble Tea components before Smalltalk inspector implementation.
LastUpdated: 2026-02-14T12:10:00Z
WhatFor: Document architecture decisions, task sequence, and validation strategy for component-driven inspector refactor.
WhenToUse: Use before extending inspector UX so new work builds on shared help/viewport/list/table/spinner/textinput/mode-keymap plumbing.
---


# Inspector Refactor Design Guide

## Goal

Refactor `cmd/inspector` to use reusable Bubble Tea/Bubbles primitives (`help`, `viewport`, `list`, `table`, `spinner`, `textinput`) and mode-aware key binding control (`bobatea/pkg/mode-keymap`) so future Smalltalk-browser features build on stable UI infrastructure instead of ad-hoc rendering/state code.

## Context

Pre-refactor state in `cmd/inspector/app/model.go` had:

- Manual help string rendering.
- Manual tree and source scroll state bookkeeping.
- No standard command-input component.
- No structured tabular metadata panel.
- Key behavior encoded mostly as string comparisons without mode-gated binding metadata.

Refactor constraints:

- Keep current inspector behavior (source/tree sync, go-to-def/usages, drawer actions) working.
- Avoid breaking existing tests in `cmd/inspector/app/model_test.go` and `cmd/inspector/app/drawer_test.go`.
- Preserve command startup flow in `cmd/inspector/main.go`.

## Quick Reference

### Components Introduced

- `help.Model`: unified help footer from key bindings.
- `spinner.Model`: status activity indicator while completion popup is active.
- `viewport.Model`: source pane scrolling/render viewport.
- `list.Model`: AST tree pane list rendering/selection shell.
- `textinput.Model`: command mode (`:`) input line.
- `table.Model`: selected-node metadata panel in tree pane.
- `mode-keymap`: enable/disable key bindings by active UI mode (`source`, `tree`, `drawer`).

### Files Changed (Code)

- `cmd/inspector/app/keymap.go` (new)
- `cmd/inspector/app/tree_list.go` (new)
- `cmd/inspector/app/model.go` (refactored)

### Task/Commit Mapping

1. Mode-aware keymap + help/spinner plumbing
- Commit: `3339aa86b12bbe450ebb01e184241fe2ff47a541`

2. Source/tree refactor to viewport/list
- Commit: `13d7bbfcecd801d6bd75743826ac5e360f13062b`

3. Drawer-adjacent command input + metadata table
- Commit: `8e1e1ce4b5c2f4caf3e202fb521bf3b4d2919f99`

### Implementation Notes By Task

Task 1 details:

- Added `KeyMap` structure with mode tags and full/short help integration.
- Wired `help.View(m.keyMap)` in footer.
- Wired `spinner.TickMsg` update cycle and status rendering for completion activity.
- Added `updateInteractionMode()` and mode sync calls on focus transitions.

Task 2 details:

- Added `sourceViewport` and rewired source pane rendering through viewport content.
- Added `treeList` with `treeListItem` builder for AST rows and scope hints.
- Refactored `refreshTreeVisible` and selection sync methods to keep list index aligned.

Task 3 details:

- Added `textinput` command mode (`:`) with commands: `drawer`, `clear`, `help`, `quit`.
- Added `table` metadata pane below tree list for selected node fields.
- Added command-line render section and command status feedback string.

## Usage Examples

Focused validation loop:

```bash
cd go-go-goja
go test ./cmd/inspector/... -count=1
```

Repo hook-equivalent validation performed during commits:

```bash
cd go-go-goja
make lint
make test
```

## Related

- `cmd/inspector/app/model.go`
- `cmd/inspector/app/keymap.go`
- `cmd/inspector/app/tree_list.go`
- `cmd/inspector/app/model_test.go`
- `cmd/inspector/app/drawer_test.go`
- `reference/02-diary.md`
- `tasks.md`
- `changelog.md`
