---
Title: Inspector Bubbles Component Refactor
Ticket: GOJA-025-INSPECTOR-BUBBLES-REFACTOR
Status: complete
Topics:
    - go
    - goja
    - tui
    - tooling
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/inspector/app/model.go
      Note: Primary refactor target for component adoption
    - Path: cmd/inspector/app/keymap.go
      Note: Mode-aware keymap and help integration
    - Path: cmd/inspector/app/tree_list.go
      Note: bubbles/list adapter for AST tree pane
    - Path: ttmp/2026/02/14/GOJA-025-INSPECTOR-BUBBLES-REFACTOR--inspector-bubbles-component-refactor/reference/01-inspector-refactor-design-guide.md
      Note: Detailed implementation guide
    - Path: ttmp/2026/02/14/GOJA-025-INSPECTOR-BUBBLES-REFACTOR--inspector-bubbles-component-refactor/reference/02-diary.md
      Note: Per-task execution diary
ExternalSources: []
Summary: Refactor ticket to adopt reusable Bubble Tea components in cmd/inspector before Smalltalk inspector implementation.
LastUpdated: 2026-02-14T20:00:08.906362554-05:00
WhatFor: Track component-driven refactor tasks, commits, and documentation for inspector UI groundwork.
WhenToUse: Use as entrypoint for GOJA-025 to review scope, links, and completion state.
---


# Inspector Bubbles Component Refactor

## Overview

This ticket refactors existing inspector machinery to reusable Bubble Tea components before implementing new Smalltalk browser features. It introduces mode-aware key bindings and standardizes core UI surfaces around Bubbles components.

## Key Links

- Design guide: `reference/01-inspector-refactor-design-guide.md`
- Diary: `reference/02-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`
- Main code target: `cmd/inspector/app/model.go`

## Status

Current status: **active**

## Topics

- go
- goja
- tui
- tooling

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
