---
Title: Harden replapi runtime lifecycle, session ownership, and HTTP safety
Ticket: GOJA-068
Status: complete
Topics:
    - goja
    - repl
    - replapi
    - lifecycle
    - persistent-repl
    - http
    - security
    - sqlite
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Research and implementation plan for making long-lived replapi runtimes explicitly owned, gracefully closable, persistence-consistent, single-owner, cancellation-aware, and safe to expose through bounded HTTP transport.
LastUpdated: 2026-07-15T21:50:43.609717086-04:00
WhatFor: Track the architecture, evidence, implementation phases, and delivery of REPL API lifecycle and hardening work.
WhenToUse: Start here when implementing, reviewing, or splitting GOJA-068 into phased pull requests.
---


# Harden replapi runtime lifecycle, session ownership, and HTTP safety

## Overview

GOJA-068 addresses correctness and safety gaps in the long-lived REPL stack. The current subsystem provides stateful Goja sessions and SQLite replay, but operation contexts leak into runtime lifetimes, apps cannot close or non-destructively unload runtimes, separate app instances can create split-brain VMs, persistence failures can create durable cell gaps, canceled callers can remain queued, profile normalization can produce contradictory policies, and the HTTP surface lacks complete resource and error boundaries.

The ticket contains an intern-oriented implementation guide, a chronological investigation diary, and five executable probes. The implementation now provides explicit app/session lifecycle state machines, context-aware operation gates, fail-closed persistence behavior, SQLite lease fencing and migrations, strict profile resolution, hardened protobuf JSON handling, and deterministic CLI/TUI/server shutdown.

## Initial Findings (addressed through Phase 6)

- HTTP-created and restored runtimes inherited request cancellation.
- Two apps could execute the same next persistent cell before SQLite rejected one.
- A failed cell write could be followed by a successful later write, producing cell IDs such as `1, 3`.
- Partial `ProfileRaw` config yielded instrumented mode with no timeout.
- A canceled operation could block on the session mutex and return success after its deadline.
- `App` had no close-all or preserve-history unload operation.
- SQLite bootstrap was not an ordered migration system.
- HTTP input was unbounded and internal errors were returned directly.

Phases 1–6 now enforce complete profile presets, app-owned lifetimes, context-aware operation gates, retryable close/unload, fail-closed commits, transactional migrations, per-session lease fencing, and bounded/versioned/redacted HTTP protobuf behavior.

## Key Links

- [Primary implementation guide](./design-doc/01-repl-api-lifecycle-ownership-and-http-hardening-implementation-guide.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)
- [Probe scripts](./scripts/)

## Implementation Milestones

The tracker contains 81 completed implementation tasks, including one explicit completion gate per phase. Phases 0–7 and all gates are complete. Use the stable `[P#.N]` labels in `tasks.md` for progress reporting.

### Milestone A — Mandatory correctness core

1. **P0:** Convert probes into deterministic regression tests.
2. **P1:** Repair profile validation and normalization.
3. **P2:** Add app/service lifecycle, context separation, cancellation-aware gates, close, and unload.
4. **P3:** Fail sessions closed after persistence commit errors and add recovery.

Milestone A is required for local, embedded, TUI, and persistent use.

### Milestone B — Conditional persistent multi-process ownership

5. **P4:** Add transactional SQLite migrations.
6. **P5:** Select and implement per-session lease/fencing or a deliberately narrower database-wide exclusive-owner contract.

Milestone B is required when separate processes may access the same durable database/session. It may be skipped only if the supported deployment model explicitly prevents that access.

### Milestone C — Transport and release integration

7. **P6:** Bound and version HTTP/protobuf behavior and harden command-server defaults.
8. **P7:** Update CLI, TUI, adapters, docs, generated bindings, migration notes, and final validation.

See `tasks.md` for entry/exit gates and the design guide for dependencies, files, commands, and acceptance criteria.

## Status

Current status: **complete**. Research/design and Phases 0–7 are complete; P0.GATE through P7.GATE are closed and the final validation/delivery bundle is published.

## Topics

- goja
- repl
- replapi
- lifecycle
- persistent-repl
- http
- security
- sqlite
