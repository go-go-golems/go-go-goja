---
Title: Migrate js-repl onto replapi and unify under goja-repl tui
Ticket: GOJA-24-REPL-TUI-UNIFICATION
Status: complete
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
    - Path: pkg/replhttp/handler.go
      Note: Replacement JSON server path after retiring the legacy web prototype
    - Path: cmd/smalltalk-inspector/app/repl_widgets.go
      Note: Non-replapi inspector assistance integration now decoupled from the full evaluator
    - Path: pkg/repl/adapters/bobatea/runtime_assistance.go
      Note: Assistance-only Bobatea adapter for existing runtimes
    - Path: pkg/repl/evaluators/javascript/evaluator.go
      Note: Execution and assistance logic split point
    - Path: pkg/replapi/app.go
      Note: Target shared session API
ExternalSources: []
Summary: "The Bubble Tea frontend now runs as `goja-repl tui`, the legacy web prototype is gone, `smalltalk-inspector` no longer depends on the full JavaScript evaluator for assistance, and the old standalone `cmd/repl` command has been removed so the interactive surface is unified under `goja-repl`."
LastUpdated: 2026-04-06T15:21:30-04:00
WhatFor: ""
WhenToUse: ""
---


# Migrate js-repl onto replapi and unify under goja-repl tui

## Overview

This ticket covers the integration phase that moved the Bubble Tea `js-repl` onto the shared `replapi` session architecture, merged the interactive TUI into the unified `goja-repl` binary as `goja-repl tui`, and removed the old standalone REPL frontends. The end-state is one execution/session core, one primary CLI binary, and one maintained interactive REPL surface.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Primary Design Doc**: `design-doc/01-js-repl-migration-to-replapi-and-goja-repl-tui-unification-guide.md`
- **Diary**: `reference/01-diary.md`

## Status

Current status: **complete**

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
