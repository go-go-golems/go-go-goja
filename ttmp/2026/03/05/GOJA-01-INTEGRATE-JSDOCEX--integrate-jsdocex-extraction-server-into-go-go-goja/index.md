---
Title: Integrate jsdocex extraction + server into go-go-goja
Ticket: GOJA-01-INTEGRATE-JSDOCEX
Status: complete
Topics:
    - goja
    - migration
    - architecture
    - tooling
    - analysis
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: >
  Migrated jsdocex doc extraction and web server into go-go-goja as reusable
  packages (pkg/jsdoc/*) plus a Glazed-wired CLI (cmd/goja-jsdoc), validated via
  extractor tests and a parity runbook.
LastUpdated: 2026-03-05T01:19:33.181832748-05:00
WhatFor: ""
WhenToUse: ""
---

# Integrate jsdocex extraction + server into go-go-goja

## Overview

This ticket is about migrating the `jsdocex/` JavaScript documentation extractor + doc-browser web server into the `go-go-goja/` repository, in a way that:

- makes the extraction/parsing reusable as a proper Go package (not `internal/`),
- keeps the existing HTTP server + UI behavior intact (same routes and JSON),
- adds Glazed-based CLI commands for `extract` and `serve`,
- leaves room to later connect docs to the broader JS parsing / AST analysis facilities already present in `go-go-goja/pkg/jsparse`.

Primary design/implementation guide:
- `reference/01-design-implementation-guide-integrate-jsdocex-into-go-go-goja.md`

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- goja
- migration
- architecture
- tooling
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
