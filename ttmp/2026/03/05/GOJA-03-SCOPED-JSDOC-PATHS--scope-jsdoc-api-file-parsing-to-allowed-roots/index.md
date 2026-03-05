---
Title: Scope jsdoc API file parsing to allowed roots
Ticket: GOJA-03-SCOPED-JSDOC-PATHS
Status: active
Topics:
    - goja
    - tooling
    - security
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-05T14:49:22.263912989-05:00
WhatFor: ""
WhenToUse: ""
---

# Scope jsdoc API file parsing to allowed roots

## Overview

This ticket scopes jsdoc API file parsing to an allowed filesystem root so untrusted request paths no longer flow into a generic file-reading helper.

The implementation focus is:

- add an `fs.FS`-scoped extractor entrypoint,
- refactor batch/server code to use it for API requests,
- preserve trusted CLI behavior,
- and improve the CodeQL/static-analysis story by making the trust boundary explicit in the code structure.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- goja
- tooling
- security
- architecture

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
