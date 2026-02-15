---
Title: 'Implementation Guide: Integrate pkg/inspector/analysis into smalltalk-inspector'
Ticket: GOJA-032-ANALYSIS-INTEGRATION
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
    - Path: cmd/smalltalk-inspector/app/model.go
      Note: Current UI-level static analysis traversal that will migrate to analysis session methods
    - Path: cmd/smalltalk-inspector/app/update.go
      Note: File-load and state transition points impacted by analysis session wiring
    - Path: pkg/inspector/analysis/session.go
      Note: Existing session wrapper to extend
    - Path: pkg/inspector/core/members.go
      Note: Reusable member extraction primitives consumed by analysis layer
ExternalSources: []
Summary: Detailed implementation guide for extracting smalltalk-inspector static-analysis access behind pkg/inspector/analysis session APIs.
LastUpdated: 2026-02-15T00:45:00Z
WhatFor: 'Break out GOJA-028 task #13 into an explicit plan that reduces UI coupling to jsparse internals and enables reuse from future CLI/REST frontends.'
WhenToUse: Use when implementing or reviewing analysis-layer integration work for smalltalk-inspector.
---


# Implementation Guide: Integrate pkg/inspector/analysis into smalltalk-inspector

## Goal

Move `cmd/smalltalk-inspector` from direct `jsparse.AnalysisResult` graph access to `pkg/inspector/analysis` session APIs, so analysis behavior is reusable and UI-independent.

This ticket is the extracted implementation work for GOJA-028 task `#13`.

## Problem Statement

Current `smalltalk-inspector` code still reaches directly into:

- `analysis.Resolution.Scopes[...]`
- `analysis.Index.Nodes[...]`
- `analysis.Program.Body`

This creates three problems:

1. UI package owns domain traversal logic.
2. Analysis behavior is harder to reuse from non-TUI surfaces.
3. Future parser/index changes require broad UI edits.

## Existing Architecture Snapshot

### Current flow

1. `loadFile` runs `jsparse.Analyze(...)`.
2. Result is stored in `Model.analysis` (`*jsparse.AnalysisResult`).
3. UI model methods use raw result internals for:
- globals list construction,
- class/function member lookup,
- source jump targets.

### Existing reusable pieces

- `pkg/inspector/core` already provides:
- `BuildClassMembers(...)`
- `BuildFunctionMembers(...)`
- `ClassExtends(...)`
- `pkg/inspector/analysis` already provides:
- `Session` wrapper around `AnalysisResult`,
- baseline xref/method-symbol helpers.

## Target Architecture

### Layering

1. `cmd/smalltalk-inspector/app` (UI orchestration):
- calls analysis-session methods only.
2. `pkg/inspector/analysis` (domain API):
- exposes stable methods for globals/members/jump metadata.
3. `pkg/jsparse` (parsing/index engine):
- remains an implementation dependency of analysis layer, not UI.

### New/expanded analysis-session API

Add methods on `pkg/inspector/analysis.Session` (or small helper types in same package):

1. `Globals() []GlobalBinding`
- sorted class/function/value,
- includes extends name when known.

2. `BindingDeclLine(name string) (line int, ok bool)`
- for global declaration jumps.

3. `ClassMembers(className string) []core.Member`
- delegates to `pkg/inspector/core`.

4. `FunctionMembers(funcName string) []core.Member`
- delegates to `pkg/inspector/core`.

5. `MemberDeclLine(className, memberName string) (line int, ok bool)`
- encapsulates AST/index traversal currently in UI.

6. `ParseError() error`
- lightweight status access for UI status bar decisions.

The UI keeps runtime-only value inspection behavior in place (that part is not static analysis).

## Migration Plan

## Phase 1: Add analysis APIs + tests

1. Extend `pkg/inspector/analysis` with the methods above.
2. Add unit tests for:
- globals sorting semantics,
- class/function member extraction wiring,
- declaration line resolution and not-found behavior.
3. Keep methods deterministic and nil-safe.

## Phase 2: Switch smalltalk model state

1. Introduce `analysisSession *analysis.Session` in smalltalk model.
2. Set session in file-load path using `analysis.NewSessionFromResult(...)` (or direct constructor).
3. Keep existing `*jsparse.AnalysisResult` temporarily for compatibility while migrating method call sites.

## Phase 3: Replace direct graph access call sites

Refactor these areas to use analysis-session methods:

1. `buildGlobals`
2. `buildClassMembers`
3. `buildFunctionMembers`
4. `jumpToBinding`
5. `jumpToMember`
6. status parse-error check

Remove UI-level direct scope/index traversal once equivalent session methods exist.

## Phase 4: Remove compatibility fields and harden

1. Remove leftover `Model.analysis` usage where session fully covers static-analysis needs.
2. Ensure runtime-only paths remain unchanged:
- runtime value inspection,
- REPL evaluation/stack inspection.
3. Update docs and related ticket references (GOJA-028, GOJA-032).

## Testing Strategy

Required:

1. `go test ./pkg/inspector/analysis -count=1`
2. `go test ./cmd/smalltalk-inspector/... -count=1`
3. `go test ./cmd/inspector/... -count=1`
4. `go test ./... -count=1`

Add or update smalltalk tests to verify:

1. globals list behavior is unchanged,
2. class/function member list behavior is unchanged,
3. source jump behavior remains correct.

## Risk and Mitigations

1. Risk: semantic drift in globals/member ordering.
- Mitigation: snapshot-like tests on representative source fixtures.

2. Risk: line mapping regressions for source jumps.
- Mitigation: method-level tests for `BindingDeclLine` and `MemberDeclLine`.

3. Risk: partial migration leaving mixed abstractions.
- Mitigation: explicit checklist in tasks; do not close ticket until direct graph reads are removed from smalltalk app static-analysis paths.

## Out of Scope

1. Runtime value inspection redesign.
2. Syntax-highlighting algorithm optimization (GOJA-030 scope).
3. Command parser improvements for `:load`.

## Deliverables

1. New analysis-session API surface with tests.
2. Smalltalk inspector using session methods for static analysis paths.
3. Updated docs/tasks/changelog in GOJA-032 and cross-reference note in GOJA-028.
