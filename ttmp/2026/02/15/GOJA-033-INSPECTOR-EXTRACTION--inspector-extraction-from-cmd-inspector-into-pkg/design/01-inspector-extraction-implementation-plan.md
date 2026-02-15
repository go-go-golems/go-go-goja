---
Title: Inspector Extraction Implementation Plan
Ticket: GOJA-033-INSPECTOR-EXTRACTION
Status: active
Topics:
    - go
    - goja
    - inspector
    - refactor
    - tui
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/inspector/app/model.go
      Note: Source/tree sync and selection logic extraction source
    - Path: go-go-goja/cmd/inspector/app/tree_list.go
      Note: Tree row/title/description shaping logic extraction source
    - Path: go-go-goja/cmd/inspector/app/drawer.go
      Note: Candidate salvage surface for future extraction passes
    - Path: go-go-goja/pkg/inspector/analysis
      Note: Existing extracted reusable analysis utilities
    - Path: go-go-goja/pkg/inspector/runtime
      Note: Existing extracted runtime utilities
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-028-CLEANUP-INSPECTOR--cleanup-inspector/reference/01-inspector-cleanup-review.md
      Note: Original findings that motivated extraction work
ExternalSources: []
Summary: Implementation plan for extracting old inspector reusable logic from cmd layer into pkg/inspector while keeping UI-framework coupling out of pkg APIs; now updated with completed extraction slices and deferred surfaces.
LastUpdated: 2026-02-15T10:55:00-05:00
WhatFor: Guide incremental extraction of tree/sync logic so it can be reused by CLI/LSP/REST adapters later.
WhenToUse: Use while implementing GOJA-033 extraction tasks.
---

# Inspector Extraction Implementation Plan

## Goal

Extract generally reusable logic from `cmd/inspector/app` into `pkg/inspector/*` with these constraints:

1. no Bubble Tea/Bubbles/lipgloss types in extracted APIs,
2. pure data + pure logic functions only,
3. old inspector behavior preserved by regression tests.

This ticket is extraction-focused only. Final API design/packaging polish is intentionally deferred to a later pass.

## Scope

In scope:

1. Tree row/view-model shaping from AST index/usage context.
2. Source cursor <-> tree node synchronization helpers.
3. Rewiring `cmd/inspector/app` to consume extracted helpers.
4. Package-level tests and command-level regression checks.

Out of scope:

1. UI theme/layout restyling.
2. Complete drawer component extraction.
3. Broad API redesign and versioned public contracts.
4. Syntax-highlighting algorithm redesign (GOJA-030).

## Extraction Principles

1. Extract only logic that is stable and domain-level.
2. Keep serialization-friendly structs as boundaries.
3. Keep UI adapters thin in `cmd/`.
4. Add tests at extraction site (`pkg`) and integration site (`cmd`).

## Target Package Additions

## `pkg/inspector/tree`

Purpose:

1. Convert visible AST nodes into reusable row data structures.
2. Attach optional hints (scope/reference/usage) without UI dependencies.

Expected API shape (initial):

1. Row DTO with `NodeID`, `Title`, `Description`.
2. `BuildRows(index, usageHighlights)` style helper.

## `pkg/inspector/navigation`

Purpose:

1. Source-byte offset calculation from `(line, col)`.
2. Selection/sync helpers between source cursor and visible tree node lists.
3. Cursor placement helpers from selected node spans.

Expected API shape (initial):

1. `SourceOffset(lines, line, col)` helper.
2. `FindVisibleNodeIndex(visibleIDs, targetID)` helper.
3. Sync helper(s) for source-to-tree and tree-to-source transitions.

## Implementation Phases

## Phase 0: Baseline + docs scaffolding

1. Create GOJA-033 plan/tasks/diary.
2. Upload plan/tasks to reMarkable.

## Phase 1: Tree extraction

1. Add `pkg/inspector/tree` DTO + row builder.
2. Add unit tests for row shaping and usage hints.

## Phase 2: Navigation extraction

1. Add `pkg/inspector/navigation` utilities.
2. Add unit tests for offset conversion, visible index lookup, sync outputs.

## Phase 3: Rewire old inspector

1. Replace local tree-list item shaping to call `pkg/inspector/tree`.
2. Replace local sync helpers in `cmd/inspector/app/model.go` to call `pkg/inspector/navigation`.
3. Keep behavior stable.

## Phase 4: Validate + document

1. Run targeted and full tests.
2. Update GOJA-033 diary/changelog/tasks with per-step outcomes and commit IDs.

## Validation Commands

```bash
go test ./pkg/inspector/... -count=1
go test ./cmd/inspector/... -count=1
go test ./... -count=1
```

## Risks and Mitigation

Risk:

1. subtle sync behavior drift while moving logic out of model methods.

Mitigation:

1. preserve old behavior with command-level regression tests and selective snapshots.

Risk:

1. extracting UI-coupled logic into pkg by accident.

Mitigation:

1. reject any `pkg` code that imports Bubble Tea/Bubbles/lipgloss.

## Done Criteria

1. Reusable tree + navigation logic lives in `pkg/inspector/*`.
2. `cmd/inspector` compiles and tests pass with extracted dependencies.
3. No Bubble Tea/Bubbles imports in newly extracted pkg code.
4. Diary and changelog reflect each extraction slice with commit references.

## Deferred Surfaces for API/Packaging Pass

These areas intentionally remain in `cmd/inspector/app` and should be addressed in the next API design ticket:

1. Drawer editing session state and tree-sitter lifecycle (`drawer.go`), including completion UI decisions.
2. Bubble Tea mode/keymap orchestration (`keymap.go`, mode switching in `model.go`) as an adapter layer over future transport-agnostic commands.
3. View rendering and layout composition (`View`/pane rendering methods in `model.go`) which should stay UI-only, but consume pkg contracts.
4. Command palette textinput + status messaging behavior that could later map to CLI/REST endpoints via an application service layer.
