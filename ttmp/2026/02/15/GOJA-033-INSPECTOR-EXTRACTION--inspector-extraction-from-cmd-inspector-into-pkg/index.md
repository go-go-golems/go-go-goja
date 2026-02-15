---
Title: Inspector Extraction from cmd/inspector into pkg
Ticket: GOJA-033-INSPECTOR-EXTRACTION
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
RelatedFiles:
    - Path: go-go-goja/pkg/inspector/tree/rows.go
      Note: Reusable row-view model shaping extracted from old inspector
    - Path: go-go-goja/pkg/inspector/navigation/sync.go
      Note: Reusable source/tree synchronization helpers
    - Path: go-go-goja/cmd/inspector/app/model.go
      Note: cmd adapter now consuming extracted navigation helpers
    - Path: go-go-goja/cmd/inspector/app/tree_list.go
      Note: cmd adapter now consuming extracted tree helper
ExternalSources: []
Summary: Extraction ticket that moved old inspector tree/sync domain logic into reusable pkg helpers with regression coverage.
LastUpdated: 2026-02-15T15:20:04.974656937-05:00
WhatFor: Track GOJA-033 extraction implementation and handoff context for the later API/packaging pass.
WhenToUse: Use when reviewing what was extracted from cmd/inspector and what remains deferred.
---


# Inspector Extraction from cmd/inspector into pkg

## Overview

GOJA-033 extracts stable, reusable logic from `cmd/inspector/app` into `pkg/inspector/*` while preserving existing old-inspector behavior. This pass focused on:

- tree row/view-model shaping (`pkg/inspector/tree`)
- source/tree sync helpers (`pkg/inspector/navigation`)
- cmd rewiring + regression tests

The packaging/API design pass remains intentionally separate.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

Implementation status: **extraction complete**, ready for API design pass.

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
