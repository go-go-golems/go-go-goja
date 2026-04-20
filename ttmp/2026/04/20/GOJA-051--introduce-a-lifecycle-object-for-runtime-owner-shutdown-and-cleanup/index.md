---
Title: Introduce a lifecycle object for runtime owner shutdown and cleanup
Ticket: GOJA-051
Status: active
Topics:
    - goja
    - engine
    - lifecycle
    - repl
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/runtime.go
      Note: Current runtime close orchestration.
    - Path: engine/factory.go
      Note: Runtime construction and module-context wiring.
    - Path: engine/runtime_modules.go
      Note: Runtime-scoped registrar context.
    - Path: pkg/runtimeowner/types.go
      Note: Owner runner abstraction.
    - Path: pkg/hashiplugin/host/registrar.go
      Note: Existing cleanup consumer.
ExternalSources: []
Summary: "Ticket workspace for designing a phase-aware lifecycle object to replace the current generic runtime close sequencing model."
LastUpdated: 2026-04-20T11:32:00-04:00
WhatFor: "Track the design, diary, tasks, and delivery artifacts for the runtime lifecycle cleanup proposal."
WhenToUse: "Use when implementing or reviewing the lifecycle-object cleanup for engine.Runtime."
---

# Introduce a lifecycle object for runtime owner shutdown and cleanup

## Overview

This ticket captures the design work for replacing the current `engine.Runtime.Close()` sequencing and generic `AddCloser(...)` API with a dedicated lifecycle object and phase-aware cleanup registration model.

The primary deliverable is a long-form design and implementation guide that explains:

1. the current runtime shutdown model,
2. why the current closer stack is too implicit,
3. what lifecycle abstraction should replace it,
4. and how to migrate existing cleanup consumers safely.

## Key Links

- Design doc: [design-doc/01-design-and-implementation-guide-for-replacing-runtime-close-sequencing-with-a-lifecycle-object.md](./design-doc/01-design-and-implementation-guide-for-replacing-runtime-close-sequencing-with-a-lifecycle-object.md)
- Diary: [reference/01-diary.md](./reference/01-diary.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**

## Deliverables

- Evidence-backed design doc
- Chronological investigation diary
- Ticket bookkeeping updates
- reMarkable bundle upload

## Topics

- goja
- engine
- lifecycle
- repl
- architecture
