---
Title: Inspector Component Alignment
Ticket: GOJA-029-INSPECTOR-COMPONENT-ALIGNMENT
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
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-029-INSPECTOR-COMPONENT-ALIGNMENT--inspector-component-alignment/design/01-component-alignment-implementation-plan.md
      Note: Primary implementation plan for this ticket
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-029-INSPECTOR-COMPONENT-ALIGNMENT--inspector-component-alignment/reference/01-diary.md
      Note: Step-by-step implementation diary
    - Path: go-go-goja/cmd/smalltalk-inspector/app/model.go
      Note: Current monolithic state/model logic targeted for alignment
    - Path: go-go-goja/cmd/smalltalk-inspector/app/update.go
      Note: Current key and message routing to be componentized
    - Path: go-go-goja/cmd/smalltalk-inspector/app/view.go
      Note: Current rendering/scroller logic to migrate to reusable components
    - Path: go-go-goja/cmd/inspector/app/keymap.go
      Note: GOJA-025 mode-aware baseline to align with
    - Path: go-go-goja/cmd/inspector/app/tree_list.go
      Note: GOJA-025 reusable list adapter pattern
ExternalSources: []
Summary: Execution ticket for aligning smalltalk-inspector with reusable GOJA-025 component architecture and reducing duplicated UI logic.
LastUpdated: 2026-02-14T19:03:00Z
WhatFor: Plan and track incremental migration from monolithic pane logic to shared component primitives.
WhenToUse: Use as entrypoint when implementing GOJA-029 refactor tasks.
---

# Inspector Component Alignment

## Overview

This ticket addresses GOJA-028 finding #5 by aligning `cmd/smalltalk-inspector` with reusable component patterns already established in GOJA-025. It focuses on mode-aware key handling, shared pane abstractions, scroll/visibility correctness, and regression-safe migration.

## Key Links

- Implementation plan: `design/01-component-alignment-implementation-plan.md`
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
