---
Title: Generic TypeScript declaration generator for Goja native modules
Ticket: GC-06-GOJA-DTS-GENERATOR
Status: active
Topics:
    - goja
    - js-bindings
    - modules
    - tooling
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: workspaces/2026-02-22/add-gepa-optimizer/go-go-goja/cmd/bun-demo/js/src/types/goja-modules.d.ts
      Note: Current manual declaration file targeted for migration
    - Path: workspaces/2026-02-22/add-gepa-optimizer/go-go-goja/engine/factory.go
      Note: Factory composition lifecycle impacted by module typing design
    - Path: workspaces/2026-02-22/add-gepa-optimizer/go-go-goja/modules/common.go
      Note: Core NativeModule registry constraints for generator design
ExternalSources: []
Summary: Ticket workspace for designing a generic go-go-goja TypeScript declaration generator for native modules with intern-ready implementation guidance.
LastUpdated: 2026-03-01T06:14:50.969638885-05:00
WhatFor: Define architecture and execution plan for reusable Goja bindings to TypeScript declaration generation.
WhenToUse: Use when implementing or reviewing the GC-06 generator and migration tasks.
---


# Generic TypeScript declaration generator for Goja native modules

## Overview

This ticket defines a non-breaking, descriptor-driven `.d.ts` generation architecture for go-go-goja native modules and provides an implementation-ready guide for intern execution.

## Key Links

- Design doc:
  - `design-doc/01-generic-goja-typescript-declaration-generator-architecture-and-implementation-guide.md`
- Diary:
  - `reference/01-implementation-diary.md`
- Tasks:
  - `tasks.md`
- Changelog:
  - `changelog.md`

## Status

Current status: **active**

## Topics

- goja
- js-bindings
- modules
- tooling
- architecture

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design working docs
- design-doc/ - Primary technical deliverable
- reference/ - Diary and quick references
- playbooks/ - Command sequences and runbooks
- scripts/ - Temporary scripts used for this ticket
- various/ - Supporting notes
- archive/ - Deprecated artifacts
