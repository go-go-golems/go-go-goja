---
Title: Component Alignment Implementation Plan
Ticket: GOJA-029-INSPECTOR-COMPONENT-ALIGNMENT
Status: active
Topics:
    - go
    - goja
    - tui
    - inspector
    - refactor
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/smalltalk-inspector/app/model.go
      Note: Current monolithic state and member-building behavior
    - Path: go-go-goja/cmd/smalltalk-inspector/app/update.go
      Note: Current key handling and message routing
    - Path: go-go-goja/cmd/smalltalk-inspector/app/view.go
      Note: Current pane rendering and manual scrolling logic
    - Path: go-go-goja/cmd/smalltalk-inspector/app/keymap.go
      Note: Local keymap implementation to align with mode-keymap approach
    - Path: go-go-goja/cmd/inspector/app/keymap.go
      Note: GOJA-025 mode-keymap baseline to reuse
    - Path: go-go-goja/cmd/inspector/app/tree_list.go
      Note: GOJA-025 reusable list adapter pattern
    - Path: go-go-goja/cmd/inspector/app/model.go
      Note: Existing shared use of help/list/table/viewport/spinner/textinput
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-028-CLEANUP-INSPECTOR--cleanup-inspector/reference/01-inspector-cleanup-review.md
      Note: Source of finding #5 and downstream cleanup priorities
ExternalSources: []
Summary: Incremental migration plan to align smalltalk-inspector with reusable inspector UI components and reduce architecture drift.
LastUpdated: 2026-02-14T19:02:00Z
WhatFor: Define a low-risk execution strategy for reusing GOJA-025 component primitives and reducing duplicated pane/state logic.
WhenToUse: Use before and during GOJA-029 implementation work.
---

# Component Alignment Implementation Plan

## Goal

Address GOJA-028 finding #5 by aligning `cmd/smalltalk-inspector` with the reusable component baseline introduced in GOJA-025 (`help`, `viewport`, `list`, `table`, `spinner`, `textinput`, `mode-keymap`), while preserving existing behavior.

## Why This Ticket Exists

Current state:

- `cmd/smalltalk-inspector` reimplements pane scrolling/selection/layout logic manually.
- `cmd/inspector` already contains reusable patterns and tested component integrations.
- Equivalent logic now diverges across two commands, increasing bug risk and maintenance cost.

This ticket narrows that gap.

## Scope

In scope:

1. Mode-aware key handling parity with GOJA-025 (`mode-keymap` style).
2. Shared scrollable pane abstractions for list-style panes and long content panes.
3. Refactor stack and inspect panes to viewport-backed rendering.
4. Incremental extraction of shared UI helpers to a reusable package (`internal/inspectorui` preferred).
5. Regression tests for navigation visibility and pane behavior.

Out of scope:

1. New end-user features unrelated to component alignment.
2. Syntax-highlighting algorithm upgrades (covered by GOJA-030).
3. Large runtime-domain rewrites (`pkg/inspector/runtime`) unless required for compatibility.

## Target Architecture

### Command Layer

`cmd/smalltalk-inspector/app` should become orchestration-focused:

- route messages,
- coordinate feature models,
- compose views from reusable components.

### Shared UI Layer

Add `go-go-goja/internal/inspectorui` with:

1. `keymode` helper:
- wraps mode switching and key filtering.
- exposes consistent help rendering contract.

2. `listpane` helper:
- generic selected-index + scroll-window behavior.
- optional `bubbles/list` adapter mode for richer list rendering.

3. `viewportpane` helper:
- long content + selected line visibility.
- reusable for source, stack, inspect views.

4. `statusline` helper:
- consistent status formatting and transient notice composition.

### Feature Models (Smalltalk)

Keep feature-specific behavior local, but consume shared UI primitives:

- globals pane model
- members pane model
- inspect pane model
- stack pane model
- source pane model
- repl model

## Implementation Strategy (Incremental)

## Phase 1: Mode + Keymap Alignment

1. Replace local ad-hoc mode handling with explicit mode-keymap approach modeled on `cmd/inspector/app/keymap.go`.
2. Ensure `ShortHelp`/`FullHelp` reflect per-mode behavior.
3. Keep existing key bindings to avoid UX churn; adjust internal dispatch only.

Deliverable:

- Smalltalk key handling aligned structurally with GOJA-025 baseline.

## Phase 2: Pane Scrolling Abstractions

1. Introduce shared `listpane` and `viewportpane` helpers.
2. Migrate globals and members panes first.
3. Migrate stack and inspect panes next (this also resolves GOJA-028 finding #3 if implemented with visibility guarantees).

Deliverable:

- No manual repeated `ensureXVisible`/window math per pane.

## Phase 3: Source + Status + Command Alignment

1. Reuse viewport component for source pane consistently.
2. Consolidate status and command-line rendering helpers.
3. Remove duplicate utility functions (`minInt`, `maxInt`, `padRight`) where possible.

Deliverable:

- Reduced duplicate logic in `model.go`, `update.go`, `view.go`.

## Phase 4: Hardening and Cleanup

1. Add tests for cross-pane navigation and visibility.
2. Verify no regression in `cmd/inspector`.
3. Remove dead code and stale compatibility shims.

Deliverable:

- Stable aligned component architecture with test coverage.

## Testing Plan

Mandatory:

1. `go test ./cmd/smalltalk-inspector/... -count=1`
2. `go test ./cmd/inspector/... -count=1`
3. `go test ./pkg/inspector/... -count=1`
4. `go test ./... -count=1`

Add/expand tests:

1. selection visibility in inspect/stack panes for long lists.
2. pane switching and mode-specific key behavior.
3. command mode open/close and help consistency.
4. regression test for stack overflow cycle once bug fix lands.

## Risks and Mitigations

Risk: breaking interaction behavior during refactor.  
Mitigation: phase-by-phase migration with behavior snapshots/tests.

Risk: over-abstraction early.  
Mitigation: extract only duplicated patterns used in at least two panes.

Risk: conflating alignment with feature additions.  
Mitigation: keep ticket scope strictly cleanup/refactor.

## Definition of Done

1. Smalltalk inspector uses shared component primitives for key mode, list panes, viewport panes, and status/help composition.
2. Manual duplicated pane scrolling logic is removed from at least inspect+stack+globals/members paths.
3. `cmd/inspector` and `cmd/smalltalk-inspector` behavior remains stable.
4. Tests cover key aligned behaviors and pass in CI/local.

## Notes on Relationship to GOJA-028 Finding #3

Yes: completing this ticket should directly address finding #3 if stack/inspect pane migration to viewport/list abstractions includes explicit visibility management.

No: it is not automatic. The implementation must include `selected item must remain visible` assertions and tests.
