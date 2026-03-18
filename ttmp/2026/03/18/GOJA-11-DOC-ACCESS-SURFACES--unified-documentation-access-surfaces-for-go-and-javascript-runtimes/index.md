---
Title: Unified documentation access surfaces for Go and JavaScript runtimes
Ticket: GOJA-11-DOC-ACCESS-SURFACES
Status: active
Topics:
    - goja
    - architecture
    - tooling
    - js-bindings
    - glazed
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Design ticket for a shared documentation access layer that can unify Glazed help, jsdoc stores, and plugin metadata for both Go callers and JavaScript runtimes.
LastUpdated: 2026-03-18T15:41:52.924626367-04:00
WhatFor: Plan a unified documentation access architecture and future implementation path.
WhenToUse: Use when implementing or reviewing doc access APIs spanning Glazed help, jsdoc, plugin metadata, and future docmgr integration.
---

# Unified documentation access surfaces for Go and JavaScript runtimes

## Overview

This ticket is about designing a unified documentation access layer for `go-go-goja`. The immediate goal is not to implement the full system, but to define a practical architecture that can expose rich documentation to both Go-side callers and JavaScript-side runtime users. The main sources in scope are Glazed help, jsdoc stores, and plugin manifests. `docmgr` is explicitly treated as a future influence, not a mandatory dependency for the first implementation.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- goja
- architecture
- tooling
- js-bindings
- glazed

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
