---
Title: Inspector Phase A Stabilization
Ticket: GOJA-031-INSPECTOR-PHASE-A-STABILIZATION
Status: active
Topics:
    - go
    - goja
    - tui
    - inspector
    - refactor
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-031-INSPECTOR-PHASE-A-STABILIZATION--inspector-phase-a-stabilization/design/01-phase-a-implementation-plan.md
      Note: Execution plan for this phase
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-031-INSPECTOR-PHASE-A-STABILIZATION--inspector-phase-a-stabilization/reference/01-diary.md
      Note: Detailed implementation diary
    - Path: go-go-goja/cmd/smalltalk-inspector/app/model.go
      Note: UI code path consuming extracted core functionality
    - Path: go-go-goja/pkg/inspector
      Note: Core extraction target for non-UI logic
ExternalSources: []
Summary: Phase A stabilization ticket to fix critical recursion crash and extract core inspector logic from Bubble Tea UI for future REST/CLI reuse.
LastUpdated: 2026-02-14T19:10:00Z
WhatFor: Track implementation of safety and architecture-boundary improvements identified in GOJA-028.
WhenToUse: Use as entrypoint for GOJA-031 execution and review.
---

# Inspector Phase A Stabilization

## Overview

This ticket implements Phase A stabilization work: fixing cyclic-inheritance crash risk, adding regression coverage, and separating core analysis functionality from UI/Bubble Tea concerns.

## Key Links

- Implementation plan: `design/01-phase-a-implementation-plan.md`
- Diary: `reference/01-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

## Topics

- go
- goja
- tui
- inspector
- refactor

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
