---
Title: Multi-format jsdoc export + batch API (json/yaml/sqlite/markdown)
Ticket: GOJA-02-JSDOC-EXPORT-API
Status: active
Topics:
    - goja
    - tooling
    - architecture
    - analysis
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-05T04:04:50.656820155-05:00
WhatFor: ""
WhenToUse: ""
---

# Multi-format jsdoc export + batch API (json/yaml/sqlite/markdown)

## Overview

This ticket designs (and will later implement) batch extraction and multi-format export for the migrated jsdoc system:

- batch inputs (multiple files; optionally inline content),
- exports to JSON, YAML, SQLite, and Markdown (with ToC),
- a web API to request batch extract/export without breaking the existing doc browser routes.

Primary design/implementation plan:
- `reference/01-design-implementation-plan-batch-jsdoc-api-and-multi-format-exporters.md`

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- goja
- tooling
- architecture
- analysis

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
