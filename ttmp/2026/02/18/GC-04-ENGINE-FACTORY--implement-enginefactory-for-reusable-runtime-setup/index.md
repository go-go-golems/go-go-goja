---
Title: Implement EngineFactory for reusable runtime setup
Ticket: GC-04-ENGINE-FACTORY
Status: active
Topics:
    - go
    - refactor
    - tooling
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: Core EngineFactory implementation
    - Path: engine/factory_test.go
      Note: Factory correctness tests
    - Path: engine/options.go
      Note: Existing Open options API to integrate with Factory
    - Path: engine/runtime.go
      Note: Current runtime creation flow to be split into reusable bootstrap
    - Path: perf/goja/bench_test.go
      Note: Spawn benchmark comparison includes EngineFactory
    - Path: pkg/calllog/calllog.go
      Note: Runtime-scoped calllog behavior Factory must preserve
    - Path: ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/runtime-spawn-enginefactory-bench.txt
      Note: Recorded benchmark output
ExternalSources: []
Summary: Plan ticket for introducing EngineFactory to optimize repeated runtime creation while preserving option-driven configuration and calllog scoping.
LastUpdated: 2026-02-18T10:16:27.02490808-05:00
WhatFor: Define design and implementation work for EngineFactory.
WhenToUse: Use when planning or reviewing EngineFactory implementation work.
---


# Implement EngineFactory for reusable runtime setup

## Overview

This ticket defines the design and implementation plan for adding an
`EngineFactory` that can precompute reusable runtime setup (module registry,
require options, calllog policy) and create fresh runtimes more efficiently
than rebuilding all setup on every `engine.Open()` call.

## Key Links

- Implementation plan: `reference/01-implementation-plan.md`
- Design plan: `reference/02-design-plan.md`
- Diary: `reference/03-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

## Topics

- go
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
