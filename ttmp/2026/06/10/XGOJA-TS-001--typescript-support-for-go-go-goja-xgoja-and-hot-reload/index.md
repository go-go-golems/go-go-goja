---
Title: TypeScript support for go-go-goja xgoja and hot reload
Ticket: XGOJA-TS-001
Status: active
Topics:
    - goja
    - xgoja
    - typescript
    - tooling
    - developer-experience
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - local:01-goja-typescript-esbuild-note.md
Summary: "Ticket workspace for designing TypeScript source support across go-go-goja, generated xgoja runtimes, jsverbs, and HTTP hot reload."
LastUpdated: 2026-06-10T21:35:00-04:00
WhatFor: "Use to find the TypeScript support design, diary, imported source note, tasks, and changelog."
WhenToUse: "When planning or implementing TypeScript execution support for go-go-goja/xgoja/hot reload."
---

# TypeScript support for go-go-goja xgoja and hot reload

## Overview

This ticket captures the analysis and proposed implementation plan for adding TypeScript authoring support to go-go-goja, generated xgoja runtimes, jsverbs, and HTTP hot reload.

The central recommendation is to compile TypeScript to JavaScript with esbuild's Go API, then keep using the existing goja runtime, goja_nodejs `require()` integration, xgoja provider modules, jsverbs command metadata, generated `.d.ts` declarations, and blue/green hot reload manager.

## Key Links

- [Design: TypeScript support analysis and implementation guide](./design/01-typescript-support-analysis-and-implementation-guide.md)
- [Diary: Investigation diary](./reference/01-investigation-diary.md)
- [Imported source note](./sources/local/01-goja-typescript-esbuild-note.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Status

Current status: **active**.

The research/design deliverable is complete. Implementation is not started; the task list records proposed implementation phases.

## Topics

- goja
- xgoja
- typescript
- tooling
- developer-experience

## Structure

- `design/` - architecture and implementation guide
- `reference/` - diary and continuation notes
- `sources/` - imported source material
- `tasks.md` - implementation checklist
- `changelog.md` - ticket updates
