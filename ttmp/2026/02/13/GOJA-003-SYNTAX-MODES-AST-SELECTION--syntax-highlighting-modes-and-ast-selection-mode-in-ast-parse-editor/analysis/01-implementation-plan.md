---
Title: Implementation plan
Ticket: GOJA-003-SYNTAX-MODES-AST-SELECTION
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
      Note: Target file for mode switching, syntax highlighting, and AST-selection behavior
    - Path: go-go-goja/cmd/ast-parse-editor/app/model_test.go
      Note: Existing model tests to extend for mode and selection transitions
    - Path: go-go-goja/pkg/jsparse/index.go
      Note: NodeAtOffset and ancestor/sibling navigation primitives for AST selection
    - Path: go-go-goja/cmd/inspector/app/model.go
      Note: Existing AST selection/sync behavior to reuse conceptually
ExternalSources: []
Summary: Plan for syntax coloring plus interactive mode switching with AST parser selection in ast-parse-editor.
LastUpdated: 2026-02-13T16:30:00-05:00
WhatFor: Define implementation details for GOJA-003 UI interaction upgrades.
WhenToUse: Use when implementing or reviewing syntax/mode/AST-selection changes.
---

# GOJA-003 Implementation Plan

## Goal

Add color syntax highlighting and mode switching so we can reuse AST-parser node selection behavior in `ast-parse-editor`.

## Current Baseline

- Editor supports direct text editing with cursor movement.
- AST parse runs asynchronously and currently emits only S-expression text.
- No mode state exists besides focused pane cycling.

## Scope

In scope:

- Syntax coloring in editor pane from CST leaf tokens.
- Mode switching (edit vs AST-select).
- AST-select mode using `jsparse.Index.NodeAtOffset` and parent/child/sibling traversal.
- Status/help updates for discoverability.

Out of scope:

- Full tree pane like `inspector`.
- Go-to-definition/usages workflows.

## Design

### 1. Editor Modes

Add mode enum in `Model`:

- `editorModeInsert` (default typing)
- `editorModeASTSelect` (navigation-select mode)

Keybinding:

- `m` cycles between insert and AST-select modes.

### 2. AST Parse Message Enrichment

Extend async AST parse result to include index:

- Build `idx := jsparse.BuildIndex(program, source)` when parse-valid.
- Store `m.astIndex`.

### 3. AST Selection State

Add state:

- `selectedASTNodeID jsparse.NodeID`

Behaviors:

- Entering AST-select mode seeds selection from cursor offset using `NodeAtOffset`.
- In AST-select mode:
  - `h` parent
  - `l` first child
  - `j` next sibling
  - `k` previous sibling
- After AST selection changes, source cursor jumps to node start and node highlight updates.

### 4. Syntax Highlighting

Implement CST-driven token map:

- Walk leaf nodes and map spans to token classes.
- Style basic classes: keyword, string, number, comment, identifier/property, operator/punctuation.
- Add toggle key `s` for syntax color on/off.

### 5. View/Status/Help

- Show `mode: INSERT` or `mode: AST-SELECT` in status/header.
- Show selected AST node kind/span in status when available.
- Update help text with `m`, `s`, and AST-select navigation keys.

## Test Plan

Add tests in `cmd/ast-parse-editor/app/model_test.go`:

1. Mode toggle changes behavior/state.
2. AST-select mode seeds selection from cursor on valid AST.
3. AST-select navigation keys move selected node and sync cursor.
4. Syntax highlight toggle switches flag without breaking edits.

Run:

- `GOWORK=off go test ./cmd/ast-parse-editor/... ./pkg/jsparse -count=1`

## Risks and Mitigations

- Large-file per-render coloring cost:
  - Keep tokenization simple and bounded by visible lines.
- AST async staleness:
  - Reuse `pendingSeq` guard and ignore stale `astParsedMsg`.

## Delivery Steps

1. Add mode/AST state fields and parse message index wiring.
2. Implement AST-select navigation and source-sync hooks.
3. Implement syntax coloring and toggle.
4. Extend tests, run tmux suite, commit.
