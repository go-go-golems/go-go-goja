---
Title: XGOJA-014 tasks
Ticket: XGOJA-014
Status: active
Topics:
  - xgoja
  - command-providers
DocType: tasks
Intent: planning
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Task list for adding command providers to go-minitrace, loupedeck, and css-visual-diff."
LastUpdated: 2026-05-25T22:05:00-04:00
WhatFor: "Track implementation progress package by package."
WhenToUse: "Use while executing or reviewing XGOJA-014."
---

# XGOJA-014 tasks

## Phase 0 — Ticket and planning

- [x] Create XGOJA-014 ticket workspace.
- [x] Inspect current provider/command surfaces in `go-minitrace`, `loupedeck`, and `css-visual-diff`.
- [x] Write package-specific implementation guide for `go-minitrace`.
- [x] Write package-specific implementation guide for `loupedeck`.
- [x] Write package-specific implementation guide for `css-visual-diff`.
- [x] Start detailed implementation diary.
- [x] Relate key implementation files to the ticket with docmgr.
- [x] Commit initial ticket docs.

## Phase 1 — go-minitrace command provider

- [x] Update `go-minitrace` dependency to `github.com/go-go-golems/go-go-goja v0.5.0`.
- [x] Add reusable catalog-to-Glazed command helper that returns `[]cmds.Command` with `Parents` populated from catalog folders.
- [x] Register `providerapi.CommandSetProvider{Name: "queries"}` in `pkg/minitracejs/provider`.
- [x] Decode command provider config (`appName`, `queryRepositories`).
- [x] Add provider tests for command provider resolution and command construction.
- [x] Run focused go-minitrace validation.
- [x] Commit go-minitrace implementation.

## Phase 2 — loupedeck command provider

- [x] Update `loupedeck` dependency to `github.com/go-go-golems/go-go-goja v0.5.0`.
- [x] Export a non-Cobra annotated verb command-list helper from `cmd/loupedeck/cmds/verbs`.
- [x] Register `providerapi.CommandSetProvider{Name: "scenes"}` in the loupedeck xgoja provider.
- [x] Decode command provider config (`includeRun`, `repositories`).
- [x] Add construction-only tests that do not open hardware sessions.
- [x] Run focused loupedeck validation.
- [x] Commit loupedeck implementation.

## Phase 3 — css-visual-diff provider and command provider

- [x] Update `css-visual-diff` dependency to `github.com/go-go-golems/go-go-goja v0.5.0`.
- [x] Extract loader-friendly module installation helpers for `css-visual-diff`, `diff`, and `report`.
- [x] Add public `pkg/xgoja/provider` package registering modules and command provider.
- [x] Export a non-Cobra verb command-list helper from `internal/cssvisualdiff/verbcli`.
- [x] Implement `css-visual-diff.verbs` command provider using xgoja `RuntimeFactory` when available.
- [x] Add provider/module/command construction tests.
- [x] Run focused css-visual-diff validation.
- [x] Commit css-visual-diff implementation.

## Phase 4 — Generated command-provider smoke tests

- [x] Write generated smoke-test design for all three command providers.
- [x] Add generated xgoja go-minitrace command-provider example with JS Markdown report writer.
- [x] Smoke go-minitrace generated binary and assert Markdown output.
- [x] Upgrade loupedeck command provider to support xgoja RuntimeFactory-based verb execution.
- [x] Add generated xgoja loupedeck command-provider example with Express-driven scene switching.
- [x] Smoke loupedeck generated binary and assert HTTP-triggered scene/report output.
- [x] Add generated xgoja css-visual-diff command-provider example with visual artifact output.
- [x] Smoke css-visual-diff generated binary and assert artifacts.
- [x] Commit smoke-test implementation at package boundaries.

## Phase 5 — Cross-repo validation and closeout

- [x] Update diary and changelog after each phase.
- [x] Run all focused package validations again.
- [x] Run `docmgr doctor --ticket XGOJA-014 --stale-after 30`.
- [x] Commit final docs.
- [ ] Optionally upload final guide bundle to reMarkable.
- [ ] Close XGOJA-014 when all package work is complete.
