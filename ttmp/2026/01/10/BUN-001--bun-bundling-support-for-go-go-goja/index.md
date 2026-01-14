---
Title: Bun bundling support for go-go-goja
Ticket: BUN-001
Status: complete
Topics:
    - goja
    - bun
    - bundling
    - javascript
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Plan and design for bun-based bundling of JS packages for Goja, with go:embed packaging.
LastUpdated: 2026-01-14T16:03:32.220656227-05:00
WhatFor: Track the design and implementation steps for bundling npm-managed JS for Goja.
WhenToUse: When reviewing ticket status, links, and documentation.
---



# Bun bundling support for go-go-goja

## Overview
Define a bun-based packaging pipeline for npm-managed JS that produces an ES5-compatible bundle for Goja and embeds it in a Go application.

## Key Links
- [Bun bundling design + analysis](./design/01-bun-bundling-design-analysis.md)
- [Package Goja research](./reference/01-package-goja-research.md)
- [Diary](./reference/02-diary.md)

## Status
Current status: **active**

## Topics
- goja
- bun
- bundling
- javascript

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
