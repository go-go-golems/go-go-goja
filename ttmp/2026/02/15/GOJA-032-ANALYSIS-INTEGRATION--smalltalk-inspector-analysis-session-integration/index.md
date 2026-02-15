---
Title: Smalltalk Inspector Analysis Session Integration
Ticket: GOJA-032-ANALYSIS-INTEGRATION
Status: complete
Topics:
    - go
    - goja
    - inspector
    - refactor
    - tui
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: 'Execution ticket for extracting GOJA-028 task #13 into a dedicated implementation stream that integrates pkg/inspector/analysis with smalltalk-inspector.'
LastUpdated: 2026-02-15T09:18:30.686187497-05:00
WhatFor: Isolate analysis-layer integration work from broader cleanup tasks and provide a clear implementation path.
WhenToUse: Use when implementing, reviewing, or tracking analysis-session integration in smalltalk-inspector.
---


# Smalltalk Inspector Analysis Session Integration

## Overview

This ticket isolates GOJA-028 cleanup task `#13` into a dedicated implementation effort.

Primary objective:

- route smalltalk-inspector static-analysis behavior through `pkg/inspector/analysis` instead of direct `jsparse` graph traversal in UI code.

Implementation guide:

- `design/01-implementation-guide-integrate-pkg-inspector-analysis-into-smalltalk-inspector.md`

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- go
- goja
- inspector
- refactor
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
