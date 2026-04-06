---
Title: Comprehensive code review of REPL architecture work since origin/main
Ticket: GOJA-25-CODE-REVIEW
Status: active
Topics:
    - code-review
    - architecture
    - repl
    - persistent-repl
    - sqlite
    - tui
    - cli
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-repl/root.go
      Note: 'Unified CLI root: subcommands'
    - Path: cmd/goja-repl/tui.go
      Note: Bubble Tea TUI subcommand backed by replapi
    - Path: pkg/repl/adapters/bobatea/replapi.go
      Note: Bobatea evaluator adapter bridging replapi sessions to Bubble Tea UI
    - Path: pkg/repl/adapters/bobatea/runtime_assistance.go
      Note: Assistance-only adapter for callers that own their own runtime
    - Path: pkg/replapi/app.go
      Note: Application facade combining live session kernel with durable store
    - Path: pkg/replapi/config.go
      Note: Profile model and config normalization/validation
    - Path: pkg/repldb/read.go
      Note: 'Read-side queries: list sessions'
    - Path: pkg/repldb/schema.go
      Note: Schema DDL for sessions
    - Path: pkg/repldb/store.go
      Note: SQLite store bootstrap and lifecycle
    - Path: pkg/repldb/write.go
      Note: 'Transactional write path: sessions'
    - Path: pkg/replhttp/handler.go
      Note: 'JSON-only HTTP transport: routes'
    - Path: pkg/replsession/policy.go
      Note: 'Session policy model: eval mode'
    - Path: pkg/replsession/rewrite.go
      Note: Async IIFE rewrite pipeline
    - Path: pkg/replsession/service.go
      Note: 'Core session kernel: session lifecycle'
    - Path: pkg/replsession/types.go
      Note: Transport-neutral DTOs for cell reports
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-06T15:31:53.823113576-04:00
WhatFor: ""
WhenToUse: ""
---


# Comprehensive code review of REPL architecture work since origin/main

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- code-review
- architecture
- repl
- persistent-repl
- sqlite
- tui
- cli

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
