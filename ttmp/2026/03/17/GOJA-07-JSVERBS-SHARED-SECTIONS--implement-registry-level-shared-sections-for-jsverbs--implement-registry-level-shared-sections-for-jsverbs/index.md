---
Title: Implement registry-level shared sections for jsverbs
Ticket: GOJA-07-JSVERBS-SHARED-SECTIONS--implement-registry-level-shared-sections-for-jsverbs
Status: complete
Topics:
    - go
    - glazed
    - js-bindings
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/jsverbs/model.go
      Note: Registry and section data model that needs to expand for shared sections
    - Path: pkg/jsverbs/binding.go
      Note: Current file-local validation bottleneck
    - Path: pkg/jsverbs/command.go
      Note: Current command-description path that fetches sections from the file only
    - Path: pkg/jsverbs/scan.go
      Note: Current scan-time section extraction ownership
    - Path: ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/design-doc/01-jsverbs-db-flags-shared-libraries-and-bundling-investigation-guide.md
      Note: Earlier investigation that motivated this implementation ticket
ExternalSources: []
Summary: Ticket for designing the registry-level shared-sections feature in jsverbs so multiple scripts can reuse a shared flag catalog without depending on runtime metadata imports.
LastUpdated: 2026-03-17T15:08:54.808850252-04:00
WhatFor: Plan and explain the implementation of shared sections in jsverbs.
WhenToUse: Use when implementing or reviewing the shared-sections feature.
---


# Implement registry-level shared sections for jsverbs

## Overview

This ticket turns the earlier shared-sections limitation analysis into a concrete design and implementation guide. The goal is to add registry-level shared sections to `pkg/jsverbs` so a Go runner can register reusable section schemas like `db`, `auth`, or `profile` and let multiple jsverb files reference them by slug.

The primary deliverable is a detailed intern-oriented design guide that explains the current architecture first, then walks through the recommended API, resolution rules, file-by-file changes, tests, and rollout strategy.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Primary analysis**: [design-doc/01-jsverbs-shared-sections-design-and-implementation-guide.md](./design-doc/01-jsverbs-shared-sections-design-and-implementation-guide.md)
- **Diary**: [reference/01-diary.md](./reference/01-diary.md)

## Status

Current status: **active**

## Topics

- go
- glazed
- js-bindings
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
