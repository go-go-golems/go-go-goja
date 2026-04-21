---
Title: Persistent REPL SQLite Persistence Replay and Export
Ticket: GOJA-21-PERSISTENT-REPL-SQLITE
Status: active
Topics:
    - persistent-repl
    - sqlite
    - architecture
    - webrepl
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Phase-1 execution ticket for adding SQLite-backed persistence, replay, export, and REPL-authored docs to the extracted replsession kernel.
LastUpdated: 2026-04-03T17:23:56.487324029-04:00
WhatFor: Use this ticket when implementing the durable storage layer required before the new CLI and JSON server surfaces can be built cleanly.
WhenToUse: Use when working on session history persistence, binding version tracking, replay/restore, or export for the persistent REPL.
---

# Persistent REPL SQLite Persistence Replay and Export

## Overview

This ticket implements the durable storage layer for the extracted persistent REPL kernel. The main outcome is a SQLite-backed `pkg/repldb` package plus `pkg/replsession` integration that records sessions, evaluations, binding versions, and REPL-authored docs in a form that can later power CLI history commands, JSON server inspection endpoints, replay, and export.

The design contract for this ticket is replay-first restore. We persist enough metadata to inspect and export sessions without a live runtime, but we do not promise full VM serialization.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- persistent-repl
- sqlite
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
