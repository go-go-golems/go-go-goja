---
Title: Syntax Highlighting Improvements
Ticket: GOJA-030-SYNTAX-HIGHLIGHTING-IMPROVEMENTS
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
    - Path: cmd/smalltalk-inspector/app/model.go
      Note: Current syntax-span rebuild and REPL source tracking flow
    - Path: cmd/smalltalk-inspector/app/view.go
      Note: Current render hot path using syntax spans
    - Path: pkg/jsparse/highlight.go
      Note: Current highlight span/lookup implementation target
    - Path: ttmp/2026/02/14/GOJA-027-SYNTAX-HIGHLIGHT--syntax-highlighting-for-smalltalk-inspector-source-pane/tasks.md
      Note: Original feature ticket context and carry-over work
    - Path: ttmp/2026/02/14/GOJA-030-SYNTAX-HIGHLIGHTING-IMPROVEMENTS--syntax-highlighting-improvements/design/01-syntax-highlighting-implementation-plan.md
      Note: Primary implementation plan for highlighting upgrades
    - Path: ttmp/2026/02/14/GOJA-030-SYNTAX-HIGHLIGHTING-IMPROVEMENTS--syntax-highlighting-improvements/design/02-syntax-highlighting-algorithm-research.md
      Note: Deep algorithm research and recommendation memo used to drive implementation tasks
ExternalSources: []
Summary: Execution ticket for syntax-highlighting algorithm and performance improvements in smalltalk-inspector.
LastUpdated: 2026-02-14T19:03:00Z
WhatFor: Plan and track benchmark-driven optimization and correctness hardening of highlighting.
WhenToUse: Use as entrypoint when implementing GOJA-030.
---


# Syntax Highlighting Improvements

## Overview

This ticket improves syntax highlighting internals after GOJA-027 by replacing expensive lookup/render paths, adding benchmark coverage, and hardening correctness and cache invalidation for both file and REPL source views.

## Key Links

- Implementation plan: `design/01-syntax-highlighting-implementation-plan.md`
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
