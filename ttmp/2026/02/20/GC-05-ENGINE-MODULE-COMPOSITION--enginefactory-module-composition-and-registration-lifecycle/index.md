---
Title: EngineFactory module composition and registration lifecycle
Ticket: GC-05-ENGINE-MODULE-COMPOSITION
Status: complete
Topics:
    - go
    - architecture
    - refactor
    - tooling
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/design/01-enginefactory-withmodules-architecture-deep-dive.md
      Note: Original architecture investigation and dependency-solver oriented analysis
    - Path: ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/design/02-pragmatic-mvp-api-for-module-composition-and-runtime-ownership.md
      Note: Finalized no-compat v2 API design used for implementation
    - Path: ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/reference/01-diary.md
      Note: Step-by-step execution diary with commit-level implementation details
ExternalSources: []
Summary: Completed ticket for no-compat EngineFactory rewrite introducing explicit builder/factory/runtime lifecycle, module specs, runtime initializers, and migrated callsites.
LastUpdated: 2026-02-21T15:58:09.471594532-05:00
WhatFor: Preserve both design rationale and implementation record for the completed EngineFactory no-compat composition rewrite.
WhenToUse: Use when reviewing or extending explicit runtime composition APIs and lifecycle ownership in go-go-goja.
---



# EngineFactory module composition and registration lifecycle

## Overview

This ticket now captures both the architecture analysis and the completed
no-backward-compatibility implementation that replaced legacy runtime wrappers
with explicit builder/factory/runtime ownership.

The shipped model introduces explicit module and runtime initializer contracts,
immutable factory construction, and callsite migration across repl/demo/tests.

## Key Links

- **Design Doc (initial)**: `design/01-enginefactory-withmodules-architecture-deep-dive.md`
- **Design Doc (v2, implemented API)**: `design/02-pragmatic-mvp-api-for-module-composition-and-runtime-ownership.md`
- **Implementation Diary**: `reference/01-diary.md`
- **Related Files**: See frontmatter/linked docs for code references
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **complete**

## Topics

- go
- architecture
- refactor
- tooling

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
