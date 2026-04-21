---
Title: Web REPL Architecture Analysis and Third-Party Integration Guide
Ticket: GOJA-20-WEBREPL-ARCHITECTURE
Status: active
Topics:
    - webrepl
    - architecture
    - rest-api
    - llm-agent-integration
    - persistent-repl
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-03T15:36:33.291764707-04:00
WhatFor: ""
WhenToUse: ""
---

# Web REPL Architecture Analysis and Third-Party Integration Guide

## Overview

This ticket now contains two architecture documents with different roles.

The earlier design doc is a detailed walkthrough of the staged `pkg/webrepl` prototype and is still useful as evidence when reading the code.
The newer design doc is the recommended implementation plan and explicitly reorients the work around `shared session kernel + CLI + JSON server first`, with SQLite-backed persistence and REPL-authored JSDoc support as first-class requirements.

## Key Links

- Recommended design:
  - `design-doc/02-cli-and-server-first-persistent-repl-architecture-and-implementation-guide.md`
- Historical prototype analysis:
  - `design-doc/01-web-repl-architecture-analysis-and-third-party-integration-design.md`
- Investigation diary:
  - `reference/01-investigation-diary.md`
- Related Files: See frontmatter `RelatedFiles`
- External Sources: See frontmatter `ExternalSources`

## Status

Current status: **active**

Recommended direction:

1. Extract a transport-neutral persistent REPL core from the current prototype.
2. Bring up CLI and JSON server on top of that core before investing more in browser UI.
3. Persist sessions, evaluations, binding versions, and REPL-authored docs in SQLite.

## Topics

- webrepl
- architecture
- rest-api
- llm-agent-integration
- persistent-repl

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

## Current Reading Order

1. `design-doc/02-cli-and-server-first-persistent-repl-architecture-and-implementation-guide.md`
2. `design-doc/01-web-repl-architecture-analysis-and-third-party-integration-design.md`
3. `reference/01-investigation-diary.md`
