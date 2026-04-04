---
Title: Migrate js-repl onto replapi and unify under goja-repl tui
Ticket: GOJA-24-REPL-TUI-UNIFICATION
Status: active
Topics:
    - repl
    - tui
    - cli
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-repl/root.go
      Note: Unified CLI root and future tui command host
    - Path: cmd/goja-repl/tui.go
      Note: Unified Bubble Tea entrypoint
    - Path: cmd/smalltalk-inspector/app/repl_widgets.go
      Note: Non-replapi inspector assistance integration now decoupled from the full evaluator
    - Path: pkg/repl/adapters/bobatea/runtime_assistance.go
      Note: Assistance-only Bobatea adapter for existing runtimes
    - Path: pkg/repl/evaluators/javascript/evaluator.go
      Note: Execution and assistance logic split point
    - Path: pkg/replapi/app.go
      Note: Target shared session API
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-04T17:51:52-04:00
WhatFor: ""
WhenToUse: ""
---


# Migrate js-repl onto replapi and unify under goja-repl tui

## Overview

This ticket covers the integration phase that moves the Bubble Tea `js-repl` onto the shared `replapi` session architecture and merges the interactive TUI into the unified `goja-repl` binary as `goja-repl tui`. The intended end-state is one execution/session core, one primary CLI binary, and removal of the remaining standalone TUI bootstrap path.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Primary Design Doc**: `design-doc/01-js-repl-migration-to-replapi-and-goja-repl-tui-unification-guide.md`
- **Diary**: `reference/01-diary.md`

## Status

Current status: **active**

## Topics

- repl
- tui
- cli
- architecture

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
