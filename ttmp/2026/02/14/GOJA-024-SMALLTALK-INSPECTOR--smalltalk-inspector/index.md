---
Title: Smalltalk Inspector
Ticket: GOJA-024-SMALLTALK-INSPECTOR
Status: complete
Topics:
    - go
    - goja
    - tui
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/smalltalk-inspector/app/model.go
      Note: Root model with all inspector state
    - Path: go-go-goja/cmd/smalltalk-inspector/app/update.go
      Note: Key handling and REPL eval
    - Path: go-go-goja/pkg/inspector/runtime/session.go
      Note: Runtime eval session
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-024-SMALLTALK-INSPECTOR--smalltalk-inspector/reference/01-diary.md
      Note: Detailed execution diary for this ticket
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-024-SMALLTALK-INSPECTOR--smalltalk-inspector/reference/02-smalltalk-goja-inspector-interface-and-component-design.md
      Note: Main implementation-oriented design analysis deliverable
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-024-SMALLTALK-INSPECTOR--smalltalk-inspector/scripts/goja-runtime-probe.go
      Note: Runtime capability probe for symbols, prototypes, and error stack output
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-024-SMALLTALK-INSPECTOR--smalltalk-inspector/scripts/jsparse-index-probe.go
      Note: Static analysis capability probe for globals/classes
ExternalSources:
    - local:smalltalk-goja-inspector.md
Summary: Ticket workspace for Smalltalk-style Goja inspector analysis and implementation blueprint generation.
LastUpdated: 2026-02-15T15:20:04.853537187-05:00
WhatFor: Track source import, screen-by-screen analysis, component-system design, and supporting probe artifacts.
WhenToUse: Use as the entry page for GOJA-024 to find the final design document, diary, scripts, and changelog.
---




# Smalltalk Inspector

## Overview

This ticket contains an implementation-oriented architecture package for a Smalltalk-style Goja inspector TUI. The imported source mockup is preserved, each target screen was analyzed, reusable Bubble Tea model decomposition was proposed, and supporting probe scripts were added to validate key runtime/static-analysis assumptions.

## Key Links

- Design doc: `reference/02-smalltalk-goja-inspector-interface-and-component-design.md`
- Diary: `reference/01-diary.md`
- Imported source: `sources/local/smalltalk-goja-inspector.md`
- Runtime probe: `scripts/goja-runtime-probe.go`
- Static probe: `scripts/jsparse-index-probe.go`
- Changelog: `changelog.md`

## Status

Current status: **active**

## Topics

- go
- goja
- tui

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
