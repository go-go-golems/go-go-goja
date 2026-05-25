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
LastUpdated: 2026-05-25T20:05:00-04:00
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
- [ ] Commit initial ticket docs.

## Phase 1 — go-minitrace command provider

- [ ] Update `go-minitrace` dependency to `github.com/go-go-golems/go-go-goja v0.5.0`.
- [ ] Add reusable catalog-to-Glazed command helper that returns `[]cmds.Command` with `Parents` populated from catalog folders.
- [ ] Register `providerapi.CommandSetProvider{Name: "queries"}` in `pkg/minitracejs/provider`.
- [ ] Decode command provider config (`appName`, `queryRepositories`).
- [ ] Add provider tests for command provider resolution and command construction.
- [ ] Run focused go-minitrace validation.
- [ ] Commit go-minitrace implementation.

## Phase 2 — loupedeck command provider

- [ ] Update `loupedeck` dependency to `github.com/go-go-golems/go-go-goja v0.5.0`.
- [ ] Export a non-Cobra annotated verb command-list helper from `cmd/loupedeck/cmds/verbs`.
- [ ] Register `providerapi.CommandSetProvider{Name: "scenes"}` in the loupedeck xgoja provider.
- [ ] Decode command provider config (`includeRun`, `repositories`).
- [ ] Add construction-only tests that do not open hardware sessions.
- [ ] Run focused loupedeck validation.
- [ ] Commit loupedeck implementation.

## Phase 3 — css-visual-diff provider and command provider

- [ ] Update `css-visual-diff` dependency to `github.com/go-go-golems/go-go-goja v0.5.0`.
- [ ] Extract loader-friendly module installation helpers for `css-visual-diff`, `diff`, and `report`.
- [ ] Add public `pkg/xgoja/provider` package registering modules and command provider.
- [ ] Export a non-Cobra verb command-list helper from `internal/cssvisualdiff/verbcli`.
- [ ] Implement `css-visual-diff.verbs` command provider using xgoja `RuntimeFactory` when available.
- [ ] Add provider/module/command construction tests.
- [ ] Run focused css-visual-diff validation.
- [ ] Commit css-visual-diff implementation.

## Phase 4 — Cross-repo validation and closeout

- [ ] Update diary and changelog after each phase.
- [ ] Run all focused package validations again.
- [ ] Run `docmgr doctor --ticket XGOJA-014 --stale-after 30`.
- [ ] Commit final docs.
- [ ] Optionally upload final guide bundle to reMarkable.
- [ ] Close XGOJA-014 when all package work is complete.
