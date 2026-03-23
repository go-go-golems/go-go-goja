---
Title: Plugin-author-facing TypeScript declaration generation for go-go-goja plugins
Ticket: GOJA-15-GEN-DTS-PLUGINS
Status: active
Topics:
    - goja
    - typescript
    - plugins
    - docs
    - analysis
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/hashiplugin/sdk/export.go
      Note: |-
        Export and method option APIs that should carry author-owned signature metadata
        Ticket now centers on plugin-author signature metadata
    - Path: pkg/hashiplugin/sdk/module.go
      Note: |-
        Core plugin module definition API and the primary place to attach TypeScript metadata
        Ticket now centers on plugin-author module definitions
    - Path: pkg/tsgen/spec/types.go
      Note: |-
        Existing declaration model to reuse for plugin-author generation
        Ticket reuses the shared TypeScript declaration model
ExternalSources: []
Summary: Corrected-scope ticket for plugin-author-facing `.d.ts` generation from source-owned SDK metadata.
LastUpdated: 2026-03-20T09:06:01.640118578-04:00
WhatFor: Analyze and document how plugin writers should generate `.d.ts` files directly from source-owned SDK metadata.
WhenToUse: Use this ticket when planning or reviewing plugin-author-facing declaration generation work in go-go-goja.
---



# Plugin-author-facing TypeScript declaration generation for go-go-goja plugins

## Overview

This ticket was revised after scope clarification. It now answers how plugin writers should generate TypeScript declarations for their own plugins from source-owned SDK metadata, rather than how a host tool should discover installed plugins and emit declarations after the fact.

The ticket contains:

1. a detailed design document written for an intern who is new to the repository,
2. a chronological diary that records the original misread and the corrected direction,
3. ticket bookkeeping and delivery evidence for the revised reMarkable upload.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- goja
- typescript
- plugins
- docs
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
