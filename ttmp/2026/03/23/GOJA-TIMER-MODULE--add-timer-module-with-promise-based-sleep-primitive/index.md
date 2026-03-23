---
Title: Add timer module with Promise-based sleep primitive
Ticket: GOJA-TIMER-MODULE
Status: active
Topics:
    - goja
    - javascript
    - modules
    - async
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-23T14:56:17.216570387-04:00
WhatFor: "Track the design and implementation of a built-in timer module for go-go-goja."
WhenToUse: "Use when adding Promise-based timing primitives or reviewing how async owner-thread work is exposed to JavaScript."
---

# Add timer module with Promise-based sleep primitive

## Overview

This ticket adds a real `timer` native module to `go-go-goja`. The immediate goal is to ship `require(\"timer\").sleep(ms)` as the first supported timing primitive, using the existing event loop and owner-thread settlement model instead of adding browser-like globals directly.

The work is split into small verified phases: planning and ticket setup, module implementation, test coverage, and final documentation/validation. The ticket also records why the module-based shape is preferred over mounting `setTimeout` globals first.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- goja
- javascript
- modules
- async

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
