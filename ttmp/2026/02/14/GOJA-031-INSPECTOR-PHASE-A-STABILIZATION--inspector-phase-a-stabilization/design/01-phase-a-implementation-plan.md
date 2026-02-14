---
Title: Phase A Implementation Plan
Ticket: GOJA-031-INSPECTOR-PHASE-A-STABILIZATION
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
      Note: Current UI-centric member-building logic to split
    - Path: go-go-goja/cmd/smalltalk-inspector/app/update.go
      Note: Bubble Tea event flow that should consume core APIs
    - Path: go-go-goja/pkg/inspector
      Note: Domain package home for UI-independent core logic
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-028-CLEANUP-INSPECTOR--cleanup-inspector/reference/01-inspector-cleanup-review.md
      Note: Source findings for Phase A stabilization priorities
ExternalSources: []
Summary: Phase A stabilization and extraction plan: fix critical recursion crash, add regression tests, and split core logic from Bubble Tea UI boundaries.
LastUpdated: 2026-02-14T19:10:00Z
WhatFor: Provide an execution-ready plan for Phase A safety and architecture boundary improvements.
WhenToUse: Use while implementing GOJA-031 tasks.
---

# Phase A Implementation Plan

## Objective

Deliver Phase A stabilization with explicit architecture boundaries:

1. Fix critical correctness issue (inheritance recursion crash).
2. Add regression tests around the fixed behavior.
3. Extract core non-UI behavior into `pkg/inspector/*` so it can later be exposed via CLI/REST.
4. Keep Bubble Tea command code (`cmd/smalltalk-inspector`) focused on UI orchestration only.

## Architecture Boundary Rule (Mandatory)

UI layer (`cmd/smalltalk-inspector`):
- Bubble Tea model/update/view.
- Keyboard/navigation/rendering concerns.

Core layer (`pkg/inspector/core`):
- Member extraction from AST.
- Inheritance traversal and cycle-safety.
- Function/member structural analysis.
- No Bubble Tea imports.

Future API exposure implication:
- Anything in `pkg/inspector/core` can be reused by REST handlers or non-TUI CLI commands without UI coupling.

## Phase A Work Packages

### WP1: Extract class/function member analysis into core package

- Create `pkg/inspector/core/members.go`.
- Move AST member extraction logic from UI model into core package.
- Expose API oriented around analysis inputs/outputs (plain Go structs).

### WP2: Fix recursion safety with cycle detection

- Implement explicit visited set in inheritance traversal.
- Add depth guard as a second safety net.
- Ensure self-cycle and indirect-cycle class hierarchies cannot panic/overflow.

### WP3: Wire UI to core APIs

- Replace direct class/function AST traversal in `cmd/smalltalk-inspector/app/model.go` with calls into `pkg/inspector/core`.
- Preserve current UI fields and behavior mapping.

### WP4: Test hardening

- Add core package tests for:
  - self-cycle (`class A extends A`)
  - indirect cycle (`A extends B`, `B extends A`)
  - inherited method dedupe behavior.
- Add command-level regression test ensuring `buildMembers` no longer panics on cycle input.

### WP5: Validation and documentation

- Run targeted and full test suites.
- Update GOJA-031 diary/changelog/tasks with per-step results and commit IDs.

## Acceptance Criteria

1. No stack overflow for cyclic inheritance source.
2. New core package exists and is used by smalltalk-inspector UI.
3. Bubble Tea code no longer owns inheritance traversal internals.
4. Tests cover the crash regression and pass.
5. Ticket diary documents each implementation step.

## Execution Sequence

1. Create core package and move logic (WP1).
2. Add safety guards (WP2).
3. Integrate UI call sites (WP3).
4. Add tests (WP4).
5. Validate and document (WP5).
