---
Title: Cleanup Inspector
Ticket: GOJA-028-CLEANUP-INSPECTOR
Status: active
Topics:
    - go
    - goja
    - tui
    - refactor
    - inspector
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-028-CLEANUP-INSPECTOR--cleanup-inspector/reference/01-inspector-cleanup-review.md
      Note: Primary 8+ page analysis and review document
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-028-CLEANUP-INSPECTOR--cleanup-inspector/reference/02-diary.md
      Note: Execution diary and command trace for this review cycle
    - Path: go-go-goja/cmd/smalltalk-inspector/app/model.go
      Note: Main state and member-building logic reviewed
    - Path: go-go-goja/cmd/smalltalk-inspector/app/update.go
      Note: Message handling and interaction logic reviewed
    - Path: go-go-goja/cmd/smalltalk-inspector/app/view.go
      Note: Render and scrolling behavior reviewed
    - Path: go-go-goja/pkg/inspector/runtime/introspect.go
      Note: Runtime introspection helpers reviewed
    - Path: go-go-goja/pkg/jsparse/highlight.go
      Note: Syntax highlighting implementation reviewed
ExternalSources: []
Summary: Thorough cleanup review of inspector work from GOJA-024 to GOJA-027 with severity-ranked findings, architecture mapping, and refactor plan.
LastUpdated: 2026-02-14T18:48:00Z
WhatFor: Capture implementation quality assessment and prioritized cleanup roadmap before additional feature expansion.
WhenToUse: Use as the primary starting point for GOJA-028 cleanup and refactor execution.
---

# Cleanup Inspector

## Overview

GOJA-028 provides a deep post-implementation review of the inspector work delivered in GOJA-024 through GOJA-027. It focuses on correctness risks, architecture drift, reuse opportunities, unidiomatic patterns, and a concrete cleanup roadmap with implementation tasks.

## Key Links

- Review doc: `reference/01-inspector-cleanup-review.md`
- Diary: `reference/02-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`
- Key code surface:
  - `go-go-goja/cmd/smalltalk-inspector/app/model.go`
  - `go-go-goja/cmd/smalltalk-inspector/app/update.go`
  - `go-go-goja/cmd/smalltalk-inspector/app/view.go`
  - `go-go-goja/pkg/inspector/runtime/introspect.go`
  - `go-go-goja/pkg/jsparse/highlight.go`

## Status

Current status: **active**

## Topics

- go
- goja
- tui
- refactor
- inspector

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
