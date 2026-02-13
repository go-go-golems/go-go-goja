---
Title: Implementation plan
Ticket: GOJA-002-CURSOR-NODE-HIGHLIGHT
Status: active
Topics:
    - goja
    - tooling
    - ui
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/ast-parse-editor/app/model.go
      Note: Current editor rendering and cursor handling implementation
    - Path: go-go-goja/cmd/ast-parse-editor/app/model_test.go
      Note: Existing editor model tests to extend for highlight behavior
    - Path: go-go-goja/pkg/jsparse/treesitter.go
      Note: NodeAtPosition and node span primitives for cursor-node mapping
    - Path: go-go-goja/cmd/inspector/app/model.go
      Note: Prior source-range highlight rendering pattern to reuse
ExternalSources: []
Summary: Plan for adding live source-pane highlight of the tree-sitter node under the cursor in ast-parse-editor.
LastUpdated: 2026-02-13T16:30:00-05:00
WhatFor: Define concrete implementation and test strategy for cursor-node highlighting.
WhenToUse: Use when implementing or reviewing GOJA-002 cursor-node highlight behavior.
---

# GOJA-002 Implementation Plan

## Goal

Add a visible highlight in the editor pane for the CST node currently under the cursor while typing.

## Current Baseline

- Editor cursor rendering exists in `go-go-goja/cmd/ast-parse-editor/app/model.go`.
- CST parse snapshot exists (`m.tsRoot`) and supports `NodeAtPosition` via `pkg/jsparse/treesitter.go`.
- No persistent node-range highlight state is currently tracked in `ast-parse-editor`.

## Scope

In scope:

- Highlight currently selected CST node range in editor pane.
- Keep highlight updated on cursor movement and edits.
- Show selected CST node metadata in status for observability.
- Add regression tests for highlight state updates.

Out of scope:

- AST-node keyboard navigation modes (handled in GOJA-003).
- Full lexical syntax coloring (handled in GOJA-003).

## Design

### 1. Model State Additions

Add fields in `Model`:

- `cursorNode *jsparse.TSNode` for latest CST node under cursor
- `highlightStartLine`, `highlightStartCol`, `highlightEndLine`, `highlightEndCol` as 1-based range bounds

### 2. Highlight Resolution

Add helper `updateCursorNodeHighlight()`:

- If `tsRoot == nil`, clear highlight.
- Probe `NodeAtPosition(cursorRow, cursorCol)` and fallback `cursorCol-1`.
- If node found, store pointer + convert node range into 1-based line/col highlight bounds.

### 3. Update Triggers

Call highlight refresh after:

- cursor movement (`moveCursor`)
- CST reparse after edits (`reparseCST`)
- initial model creation

### 4. Rendering

Enhance `renderEditorPane`:

- Apply highlight style over characters inside highlight range.
- Keep cursor style precedence above highlight style.

### 5. Status Line

Append `node: <kind> (<start..end>)` when a node is resolved at cursor.

## Test Plan

Add tests in `go-go-goja/cmd/ast-parse-editor/app/model_test.go`:

1. Initial cursor resolves highlight range on valid source.
2. Moving cursor to another token updates highlight range.
3. Empty source keeps sane highlight behavior (no panic, no invalid range).

Run:

- `GOWORK=off go test ./cmd/ast-parse-editor/... ./pkg/jsparse -count=1`

## Risks and Mitigations

- Boundary condition at end-of-line cursor:
  - Mitigate with `cursorCol` and `cursorCol-1` lookup fallback.
- Multiline node ranges:
  - Reuse inspector-style inclusive/exclusive per-line checks.

## Delivery Steps

1. Implement model highlight state + resolver.
2. Wire render and status updates.
3. Add/adjust tests.
4. Run tmux validation and commit.
