---
Title: Hybrid Implementation Plan (Clean Cutover)
Ticket: GOJA-034-USER-FACING-INSPECTOR-API
Status: active
Topics:
    - go
    - goja
    - inspector
    - api
    - architecture
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/pkg/inspectorapi
      Note: New user-facing API service layer (to be implemented in this ticket)
    - Path: go-go-goja/cmd/smalltalk-inspector/app/model.go
      Note: Primary clean-cutover consumer for API layer adoption
    - Path: go-go-goja/cmd/smalltalk-inspector/app/update.go
      Note: File load and REPL flows will be switched to API service methods
    - Path: go-go-goja/pkg/inspector/analysis
      Note: Existing static analysis helpers used by the new facade
    - Path: go-go-goja/pkg/inspector/runtime
      Note: Existing runtime helpers used by the new facade
    - Path: go-go-goja/pkg/inspector/navigation
      Note: Existing source/tree sync helpers wrapped by new facade methods
    - Path: go-go-goja/pkg/inspector/tree
      Note: Existing tree row builder wrapped by new facade methods
ExternalSources: []
Summary: Detailed implementation plan for hybrid user-facing inspector API with clean cutover and no backward-compatibility shims.
LastUpdated: 2026-02-15T11:26:00-05:00
WhatFor: Guide execution of the first implementation slice of the hybrid architecture.
WhenToUse: Use while implementing GOJA-034 code changes and migration.
---

# Hybrid Implementation Plan (Clean Cutover)

## Decision and Scope

We will implement the hybrid approach as a **clean cutover**:

1. No compatibility shims for old adapter-local static analysis orchestration.
2. Frontend command models should consume the new facade directly.
3. Existing low-level packages (`pkg/jsparse`, `pkg/inspector/*`) remain the implementation substrate.

This plan intentionally prioritizes tightening and simplification over preserving intermediate APIs.

## Target Architecture (Phase A)

Introduce a new package:

1. `pkg/inspectorapi`

Phase A capabilities:

1. Document session lifecycle:
   1. open from source or existing analysis result,
   2. update source,
   3. close document.
2. Static analysis use cases:
   1. list globals,
   2. list members for selected global,
   3. declaration jumps for globals/members.
3. Runtime-integrated global/member behavior:
   1. merge runtime globals and REPL declarations,
   2. runtime-derived member listing for value globals.
4. Navigation/tree wrappers:
   1. source->tree sync,
   2. tree->source sync,
   3. tree row building.
5. Smalltalk inspector cutover:
   1. replace direct `inspector/analysis` orchestration calls with `inspectorapi.Service`.

Out of scope for this slice:

1. REST adapter implementation.
2. LSP adapter implementation.
3. Deeper runtime object-handle protocol.
4. Tree-state decoupling from `jsparse.Index.Expanded`.

## Data and Contract Direction

Phase A contract strategy:

1. Expose explicit request/response structs in `pkg/inspectorapi`.
2. Keep DTOs UI-framework agnostic.
3. Reuse existing domain enums where stable (`jsparse.BindingKind`).
4. Keep coordinate conventions explicit in comments (0-based vs 1-based).

## Clean Cutover Rules

1. `cmd/smalltalk-inspector/app` should stop depending on `inspectoranalysis.Session` for globals/members/jumps.
2. Static analysis orchestration logic should move into `pkg/inspectorapi`.
3. No adapter-local duplicate logic should remain for:
   1. global list creation,
   2. member list creation,
   3. declaration line lookup,
   4. runtime-global merge.

## Implementation Phases

## Phase 1: Service Foundation

1. Add `pkg/inspectorapi` package with:
   1. service struct and in-memory document registry,
   2. core contracts and error types,
   3. open/update/close/get document operations.

## Phase 2: Static + Runtime Use Cases

1. Implement facade methods:
   1. list globals,
   2. list members,
   3. go-to declaration helpers,
   4. runtime-global merge helper.
2. Add parser-backed REPL declaration extraction helper in facade namespace.

## Phase 3: Navigation/Tree Facade Methods

1. Wrap existing extracted packages:
   1. `pkg/inspector/navigation`,
   2. `pkg/inspector/tree`.
2. Expose service methods for source/tree sync and tree rows.

## Phase 4: Smalltalk Inspector Cutover

1. Introduce `inspectorapi.Service` in model state.
2. Route file-load setup through service open/register path.
3. Switch globals/members/jump and merge logic to service methods.
4. Remove obsolete adapter-local static orchestration dependencies.

## Phase 5: Validation and Documentation

1. Add/adjust service unit tests.
2. Update smalltalk inspector tests for new service wiring.
3. Run:
   1. `go test ./pkg/inspectorapi/... -count=1`
   2. `go test ./cmd/smalltalk-inspector/... -count=1`
   3. `go test ./... -count=1`
4. Update GOJA-034 tasks/changelog/diary with commit-linked progress.

## Risk Assessment

Risk:

1. Cutover introduces behavior drift in globals/members panels.

Mitigation:

1. Keep existing tests green; add service-level tests for the moved logic.

Risk:

1. Mixed index/line conventions create off-by-one regressions.

Mitigation:

1. Make service method coordinate behavior explicit and test both sync directions.

Risk:

1. Runtime-dependent member listing differs due to orchestration move.

Mitigation:

1. Add regression tests for runtime-derived value members and global merge.

## Done Criteria

1. `pkg/inspectorapi` exists with phase-A methods and tests.
2. `cmd/smalltalk-inspector/app` uses the new service for static analysis orchestration.
3. No compatibility shims added for prior adapter-local static orchestration.
4. Full tests pass and GOJA-034 docs are updated.
