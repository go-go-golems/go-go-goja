---
Title: Export d.ts type definitions from xgoja-generated binaries
Ticket: XGOJA-019
Status: closed
Topics:
    - xgoja
    - typescript
    - modules
    - tooling
    - developer-experience
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/modules/typing.go
      Note: TypeScriptDeclarer interface — the contract modules implement
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/tsgen/spec/types.go
      Note: spec.Module data model for d.ts descriptors
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/tsgen/render/dts_renderer.go
      Note: Renders spec.Bundle into .d.ts string
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providerapi/module.go
      Note: providerapi.Module — needs DTSDescriptor field added
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/core/core.go
      Note: Core provider — needs TypeScriptDeclarer extraction
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/host/host.go
      Note: Host provider — needs TypeScriptDeclarer extraction
ExternalSources: []
Summary: Make TypeScript type definitions for native modules available to JS developers working with xgoja-generated binaries. The d.ts generation pipeline exists (TypeScriptDeclarer → spec → render → gen-dts) but is disconnected from the xgoja provider layer and generated binaries.
LastUpdated: 2026-06-10T10:42:43.145079464-04:00
WhatFor: Enable JS developers to get accurate TypeScript type definitions for the native modules available in any xgoja-generated binary.
WhenToUse: When implementing d.ts export from xgoja, adding the gen-dts subcommand, or wiring runtime type-definition exposure.
---


# Export d.ts type definitions from xgoja-generated binaries

## Overview

This ticket addresses the developer-experience gap where JavaScript developers writing code against xgoja-generated binaries have no way to discover the TypeScript types of available native modules. The d.ts generation pipeline already exists but is disconnected from the xgoja build system.

## Key Links

- Design doc: `design-doc/01-d-ts-export-architecture-and-implementation-plan.md`
- Investigation diary: `reference/01-investigation-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Three Gaps Identified

1. **Gap A:** `providerapi.Module` loses the TypeScript descriptor — `nativeModuleEntry()` in core.go/host.go wraps `NativeModule` but discards `TypeScriptDeclarer`
2. **Gap B:** No `xgoja gen-dts` subcommand — can't map xgoja.yaml module selections to d.ts output
3. **Gap C:** No runtime d.ts exposure — generated binaries cannot emit or serve type definitions

## Prior Tickets

- `GC-06-GOJA-DTS-GENERATOR` (ttmp/2026/03/01/) — Original d.ts generator design (completed)
- `GOJA-15-GEN-DTS-PLUGINS` (ttmp/2026/03/20/) — Extending gen-dts for plugins

## Status

Current status: **active** — investigation complete, design doc written, awaiting implementation.
