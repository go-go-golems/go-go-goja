---
Title: Diary
Ticket: GOJA-034-USER-FACING-INSPECTOR-API
Status: active
Topics:
    - go
    - goja
    - inspector
    - api
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/smalltalk-inspector/app/model.go
      Note: Smalltalk static-flow cutover to inspectorapi service
    - Path: cmd/smalltalk-inspector/app/model_members_test.go
      Note: Updated tests for service-backed model setup
    - Path: cmd/smalltalk-inspector/app/update.go
      Note: File load and REPL declaration flow switched to inspectorapi
    - Path: pkg/doc/07-inspectorapi-hybrid-service-guide.md
      Note: New Glazed-compliant user-facing help page documented in Step 3
    - Path: pkg/inspectorapi/contracts.go
      Note: New hybrid facade contracts and request/response DTOs
    - Path: pkg/inspectorapi/service.go
      Note: Service implementation for document lifecycle and inspector use cases
    - Path: pkg/inspectorapi/service_test.go
      Note: Regression tests for phase-A service behavior
ExternalSources: []
Summary: Implementation diary for GOJA-034 hybrid API plan and phase-A cutover.
LastUpdated: 2026-02-15T11:46:00-05:00
WhatFor: Track implementation steps, decisions, regressions, and validation for GOJA-034.
WhenToUse: Use when reviewing or continuing GOJA-034 implementation work.
---


# Diary

## Goal

Capture execution of GOJA-034 from implementation planning through the first hybrid service-layer cutover into `cmd/smalltalk-inspector`.

## Step 1: Plan Phase-A Cutover and Expand Executable Tasks

This step converted the prior design-only ticket state into an executable implementation plan with concrete phases and clean-cutover constraints. The plan explicitly removed compatibility-shim expectations to keep refactoring boundaries tight and direct.

The output was a task graph that could be executed in one pass: service creation, use-case methods, smalltalk cutover, tests, and documentation closure.

### Prompt Context

**User prompt (verbatim):** "Create an implementation plan for the hybrid approach. We can do clean cutover, no need to maintain compatibility, which should make refactoring and tightening easier. Then add detailed tasks to the ticket, and work on them, committing and updating your diary at appropriate points, checking off tasks as you go."

**Assistant interpretation:** Author a detailed GOJA-034 implementation plan with explicit clean-cutover assumptions, convert it to detailed tasks, then implement and document progress with commit-linked diary updates.

**Inferred user intent:** Move from architecture discussion to concrete delivery of the first hybrid API slice and migration, without being slowed by backwards-compatibility constraints.

**Commit (code):** `b31157b` — "docs(goja-034): add clean-cutover implementation plan and tasks"

### What I did

- Added implementation plan:
  - `ttmp/2026/02/15/GOJA-034-USER-FACING-INSPECTOR-API--user-facing-inspector-analysis-api-design/design/02-hybrid-implementation-plan-clean-cutover.md`
- Expanded tasks in ticket `tasks.md` into executable phases.
- Added GOJA-034 diary scaffold and updated ticket index/changelog.

### Why

- The ticket needed a delivery-grade plan and task decomposition before code execution.
- Clean-cutover constraints needed to be documented explicitly to avoid accidental compatibility work.

### What worked

- The implementation phases mapped cleanly onto existing extraction work and package boundaries.
- The plan naturally translated into concrete coding tasks for immediate execution.

### What didn't work

- N/A.

### What I learned

- The existing extraction baseline was sufficient to start the facade layer directly without intermediate scaffolding tickets.

### What was tricky to build

- The key planning challenge was balancing breadth (hybrid architecture) and a shippable first slice. The resolution was to lock “phase-A” boundaries and keep adapters thin.

### What warrants a second pair of eyes

- Ensure phase-A scope remains aligned with product priorities before expanding into REST/LSP adapters.

### What should be done in the future

- Follow-on ticket for phase-B service expansion (runtime object handles, transport adapters).

### Code review instructions

- Read:
  - `ttmp/2026/02/15/GOJA-034-USER-FACING-INSPECTOR-API--user-facing-inspector-analysis-api-design/design/02-hybrid-implementation-plan-clean-cutover.md`
  - `ttmp/2026/02/15/GOJA-034-USER-FACING-INSPECTOR-API--user-facing-inspector-analysis-api-design/tasks.md`

### Technical details

- Plan codifies:
  - service package target: `pkg/inspectorapi`,
  - phase-A capability set,
  - clean cutover rules for `cmd/smalltalk-inspector/app`.

## Step 2: Implement Hybrid Service Layer and Cut Over Smalltalk Static Flows

This step implemented the first production slice of the hybrid architecture. A new `pkg/inspectorapi` package now provides a task-oriented facade over extracted analysis/runtime/navigation/tree functionality.

Smalltalk inspector static orchestration was migrated from adapter-local `inspectoranalysis.Session` wiring to `inspectorapi.Service`, matching the clean-cutover requirement and removing duplicate orchestration logic from the command model.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Execute the planned tasks end-to-end with commits, tests, and diary updates.

**Inferred user intent:** Deliver actual code migration and verify that the new service boundary is practical and stable.

**Commit (code):** `cd823ad` — "inspectorapi: add hybrid service layer and cut over smalltalk static flows"

### What I did

- Added new package:
  - `pkg/inspectorapi/contracts.go`
  - `pkg/inspectorapi/service.go`
  - `pkg/inspectorapi/service_test.go`
- Implemented service features:
  - document lifecycle: open/update/close,
  - globals and members retrieval,
  - declaration line lookups,
  - REPL declaration extraction wrapper,
  - runtime global merge,
  - tree row builder wrapper,
  - source<->tree sync wrappers,
  - analysis retrieval.
- Cut over `cmd/smalltalk-inspector/app`:
  - `model.go`: replaced `inspectoranalysis.Session` orchestration with `inspectorapi.Service`,
  - `update.go`: register file loads via service and use service-level declaration extraction,
  - `view.go`: parse-error status derived from `m.analysis`,
  - `model_members_test.go`: updated model setup to open service document sessions.
- Validated with:
  - `go test ./pkg/inspectorapi/... -count=1`
  - `go test ./cmd/smalltalk-inspector/... -count=1`
  - `go test ./... -count=1`

### Why

- Centralize reusable inspector use cases in a single user-facing API layer.
- Remove adapter-local duplication and tighten architecture for upcoming CLI/REST/LSP adapters.

### What worked

- Service API was sufficient to absorb existing smalltalk static workflows without compatibility shims.
- Existing smalltalk tests passed after focused updates.
- Full repository tests remained green.

### What didn't work

- Pre-commit `go generate ./...` Dagger image fetch for `node:20.18.1` timed out (`lookup registry-1.docker.io: i/o timeout`) and automatically fell back to local npm build; this did not block commit.

### What I learned

- The extracted packages were mature enough for immediate composition into a higher-level facade.
- Clean cutover reduced code complexity versus carrying temporary bridges.

### What was tricky to build

- The main complexity was preserving runtime-derived member behavior while moving static orchestration to the new service.
- This required keeping runtime session as an explicit argument in service calls that need it, instead of coupling runtime ownership into document sessions prematurely.

### What warrants a second pair of eyes

- Concurrency semantics in `inspectorapi.Service`: current in-memory registry uses coarse locking and returns document pointers; this is fine for current TUI use but should be reviewed before server-grade concurrency.
- Coordinate conventions in sync wrappers (0-based cursor inputs, 1-based index offsets) should receive dedicated API-contract tests as surface area expands.

### What should be done in the future

- Add phase-B contracts for runtime object handles and transport-safe inspection responses.
- Add one non-TUI adapter (CLI or REST) on top of `pkg/inspectorapi` to validate transport neutrality.

### Code review instructions

- Start with:
  - `pkg/inspectorapi/service.go`
  - `pkg/inspectorapi/contracts.go`
  - `cmd/smalltalk-inspector/app/model.go`
  - `cmd/smalltalk-inspector/app/update.go`
- Validate with:
  - `go test ./pkg/inspectorapi/... -count=1`
  - `go test ./cmd/smalltalk-inspector/... -count=1`
  - `go test ./... -count=1`

### Technical details

- Service methods currently implemented:
  - `OpenDocument`
  - `OpenDocumentFromAnalysis`
  - `UpdateDocument`
  - `CloseDocument`
  - `Analysis`
  - `ListGlobals`
  - `ListMembers`
  - `BindingDeclarationLine`
  - `MemberDeclarationLine`
  - `MergeRuntimeGlobals`
  - `BuildTreeRows`
  - `SyncSourceToTree`
  - `SyncTreeToSource`
  - `DeclaredBindingsFromSource` (package-level helper).

## Step 3: Author Glazed-Compliant Help Documentation for inspectorapi

This step produced user-facing documentation for the hybrid service layer in the repository help-doc system (`pkg/doc`). The page was written against the same rules used by Glazed help entries, including strict frontmatter, section-intro paragraphs, troubleshooting table, and `See Also` references.

The content targets new contributors who need both conceptual orientation and practical integration steps for `pkg/inspectorapi`.

### Prompt Context

**User prompt (verbatim):** "Now write a great documentation for it, following the glazed writing guidelines glazed/pkg/doc/topics/01-help-system.md and glazed/pkg/doc/topics/how-to-write-good-documentation-pages.md and glazed/pkg/doc/topics/14-writing-help-entries.md"

**Assistant interpretation:** Write a high-quality help page for the new hybrid inspector API and align structure/style with the specific Glazed guideline topics.

**Inferred user intent:** Ensure this new architecture is documented in the same quality and discoverability model as existing Glazed CLI help pages.

### What I did

- Located and reviewed guideline docs from the pinned Glazed version in `go.mod` (`v0.7.13`):
  - `.../glazed@v0.7.13/pkg/doc/topics/01-help-system.md`
  - `.../glazed@v0.7.13/pkg/doc/topics/how-to-write-good-documentation-pages.md`
  - `.../glazed@v0.7.13/pkg/doc/topics/14-writing-help-entries.md`
- Added new help page:
  - `pkg/doc/07-inspectorapi-hybrid-service-guide.md`
- Ensured page follows key conventions:
  - YAML frontmatter with Glazed metadata fields,
  - no top-level `#` title in body,
  - section-opening explanatory paragraphs,
  - practical code examples and migration checklist,
  - troubleshooting table,
  - `See Also` with `glaze help <slug>` references.

### Why

- `pkg/inspectorapi` is now a primary integration boundary and needs first-class help-system documentation, not only ticket docs.
- Keeping docs in `pkg/doc` makes content discoverable through existing help loading via `pkg/doc/doc.go`.

### What worked

- Existing `pkg/doc` structure already matched Glazed help conventions, so adding one new section required no loader changes.
- The guideline-driven format naturally aligned with this repository’s prior help pages.

### What didn't work

- N/A.

### What I learned

- The most important style constraints for help usability here are:
  - strong opening paragraph per section,
  - concise metadata,
  - explicit troubleshooting and cross-links.

### What was tricky to build

- Balancing reference depth and practical onboarding in one page required careful section ordering. The final structure starts with intent/architecture, then workflows/API surface, then migration/testing/ops guidance.

### What warrants a second pair of eyes

- Verify command tags and topic tags in frontmatter are exactly what maintainers want for discoverability and filtering.

### What should be done in the future

- Add companion Example/Application help entries demonstrating:
  - a minimal non-TUI CLI consumer,
  - a thin REST adapter over `pkg/inspectorapi`.

### Code review instructions

- Review content and frontmatter in:
  - `pkg/doc/07-inspectorapi-hybrid-service-guide.md`
- Optionally validate load path by reviewing:
  - `pkg/doc/doc.go`

### Technical details

- The new page was intentionally authored as `SectionType: GeneralTopic` and `IsTopLevel: true` to keep it visible during early adoption.
