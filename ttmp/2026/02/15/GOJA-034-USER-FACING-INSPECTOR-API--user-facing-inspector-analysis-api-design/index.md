---
Title: User-Facing Inspector Analysis API Design
Ticket: GOJA-034-USER-FACING-INSPECTOR-API
Status: active
Topics:
    - go
    - goja
    - inspector
    - api
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/ttmp/2026/02/15/GOJA-034-USER-FACING-INSPECTOR-API--user-facing-inspector-analysis-api-design/design/01-user-facing-inspector-api-analysis-and-design-guide.md
      Note: Primary 8+ page onboarding, analysis, and API design document
    - Path: go-go-goja/pkg/inspector/analysis
      Note: Existing high-level analysis helpers to be wrapped by user-facing service layer
    - Path: go-go-goja/pkg/inspector/runtime
      Note: Runtime inspection and eval helpers for API packaging
    - Path: go-go-goja/pkg/inspector/navigation
      Note: Extracted source/tree sync primitives for reusable API endpoints
    - Path: go-go-goja/pkg/inspector/tree
      Note: Extracted tree row shaping primitives for reusable API endpoints
ExternalSources: []
Summary: Ticket for designing a stable user-facing API over extracted analysis and inspection functionality.
LastUpdated: 2026-02-15T11:10:00-05:00
WhatFor: Track architecture decisions and contracts for a reusable inspector service layer.
WhenToUse: Use when implementing shared APIs for TUI, CLI, REST, and future LSP adapters.
---

# User-Facing Inspector Analysis API Design

## Overview

This ticket defines how to expose extracted inspector functionality as a clean user-facing API. The design document maps current capabilities, explains architecture for new developers, and proposes concrete API contracts and migration paths.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

Progress: design and brainstorming guide delivered (8+ pages), ready for implementation ticket(s).

## Topics

- go
- goja
- inspector
- api
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
