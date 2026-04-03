---
Title: Persistent REPL CLI and JSON Server Surfaces
Ticket: GOJA-22-PERSISTENT-REPL-CLI-SERVER
Status: active
Topics:
    - persistent-repl
    - cli
    - rest-api
    - architecture
    - webrepl
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Implementation ticket for the first-class persistent REPL CLI and JSON-only HTTP server built on replsession and repldb.
LastUpdated: 2026-04-03T17:23:56.4868507-04:00
WhatFor: Use this ticket when implementing or reviewing the new persistent REPL command surface and JSON server transport.
WhenToUse: Use when working on restore-aware CLI commands, agent-facing HTTP routes, or the transition away from the browser-oriented prototype boundary.
---

# Persistent REPL CLI and JSON Server Surfaces

## Overview

This ticket turns the durable persistent REPL core into usable product surfaces. The main outcomes are a new `goja-repl` binary, a JSON-only HTTP package, and a shared restore-aware orchestration layer that can load persisted sessions back into live runtimes when needed.

The design center is not the browser UI. The design center is command and API workflows that remain useful across process restarts because they can rebuild live runtime state from SQLite-backed history.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- persistent-repl
- cli
- rest-api
- architecture
- webrepl

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
