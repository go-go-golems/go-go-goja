---
Title: Event emitter module for go-go-goja
Ticket: EVT-001
Status: active
Topics:
    - goja
    - javascript
    - event-emitter
    - module
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - local:01-event-emitter.md
Summary: Ticket workspace for designing a Go-owned event bus with a JavaScript EventEmitter façade in go-go-goja.
LastUpdated: 2026-04-26T09:31:00-04:00
WhatFor: Track the event-emitter module implementation design, source brief, investigation evidence, diary, validation, and delivery.
WhenToUse: Use when implementing or reviewing EVT-001 event-emitter work.
---


# Event emitter module for go-go-goja

## Overview

EVT-001 captures the design for adding a Node-style `events` module and a Go-owned runtime event bus to `go-go-goja`.

The primary deliverable is an intern-oriented implementation guide that explains goja runtime ownership, module registration, `runtimeowner.Runner` scheduling, the EventEmitter façade, Watermill message settlement, and fsnotify event delivery. The imported source brief is stored under `sources/local/01-event-emitter.md`, and reproducible investigation output is stored under `sources/local/evidence.txt`.

## Key Links

- **Design guide**: [design-doc/01-event-emitter-module-implementation-guide.md](design-doc/01-event-emitter-module-implementation-guide.md)
- **Diary**: [reference/01-diary.md](reference/01-diary.md)
- **Imported brief**: [sources/local/01-event-emitter.md](sources/local/01-event-emitter.md)
- **Evidence capture**: [sources/local/evidence.txt](sources/local/evidence.txt)
- **Evidence script**: [scripts/01-gather-event-emitter-evidence.sh](scripts/01-gather-event-emitter-evidence.sh)

## Status

Current status: **active**

## Topics

- goja
- javascript
- event-emitter
- module

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
