---
Title: CLI and JSON Server Implementation Plan
Ticket: GOJA-22-PERSISTENT-REPL-CLI-SERVER
Status: active
Topics:
    - persistent-repl
    - cli
    - rest-api
    - architecture
    - webrepl
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-repl/root.go
      Note: Single-binary persistent CLI command surface
    - Path: cmd/goja-repl/root_test.go
      Note: CLI flow validation across create eval and history commands
    - Path: cmd/repl/main.go
      Note: Existing simple CLI REPL to supersede with a persistent command surface
    - Path: cmd/web-repl/main.go
      Note: Existing browser-oriented server entry point that should stop owning the long-term API shape
    - Path: pkg/replapi/app.go
      Note: Restore-aware orchestration layer used by both CLI and HTTP transports
    - Path: pkg/replapi/app_test.go
      Note: Restore-aware app validation across process boundaries
    - Path: pkg/repldb/read.go
      Note: Read-side export and replay inputs that the new CLI and server should consume
    - Path: pkg/repldb/write.go
      Note: Durable write path already available to the new transports
    - Path: pkg/replhttp/handler.go
      Note: JSON-only transport package for the persistent REPL server
    - Path: pkg/replhttp/handler_test.go
      Note: HTTP handler lifecycle and history/export validation
    - Path: pkg/replsession/service.go
      Note: Live session kernel that needs restore-aware orchestration for CLI/server use
    - Path: pkg/webrepl/server.go
      Note: Existing mixed static UI plus JSON API transport to avoid using as the long-term boundary
ExternalSources: []
Summary: Detailed implementation plan for a single-binary persistent REPL CLI and a JSON-only HTTP transport built on the extracted replsession and repldb layers.
LastUpdated: 2026-04-03T18:05:00-04:00
WhatFor: Use this guide when implementing the first-class CLI and JSON server interfaces on top of the durable persistent REPL core.
WhenToUse: Use when designing or reviewing the command surface, restore flow, HTTP routes, and the transition away from the web-first prototype.
---


# CLI and JSON Server Implementation Plan

## Executive Summary

`GOJA-20` extracted the shared session kernel. `GOJA-21` made it durable with SQLite. `GOJA-22` should now expose that core through the two product surfaces we actually care about first:

- a single-binary CLI for humans and automation,
- a JSON-only HTTP server for agents and future clients.

The key design point for this ticket is that CLI commands must work across process boundaries. A plain in-memory `replsession.Service` is not enough for that. The command surface therefore needs a small orchestration layer that can:

1. open the durable store,
2. restore a session into a fresh runtime by replaying persisted cells when needed,
3. evaluate new source against that restored runtime,
4. persist the new evaluation back into SQLite.

This ticket should end with a new `goja-repl` binary, a new JSON transport package that does not depend on browser assets, and enough restore-aware orchestration that later cleanup can retire the `webrepl` prototype without losing functionality.

## Problem Statement

The repo currently has three separate entry points:

- `cmd/repl`: basic in-process REPL loop,
- `cmd/js-repl`: rich Bobatea UI,
- `cmd/web-repl`: browser-oriented server plus prototype JSON routes.

None of those is the durable first-class interface we want.

Current gaps:

1. There is no single binary that exposes persistent session lifecycle plus history/export/restore commands.
2. The HTTP transport is still packaged as `webrepl` and mixed with static asset serving.
3. There is no shared restore-aware orchestration for "load persisted session, replay it, then continue".
4. There is no JSON-only API surface designed for agent use without browser baggage.

## Goals

This ticket should deliver:

1. A new `cmd/goja-repl` binary.
2. Persistent CLI subcommands for:
   - `create`
   - `eval`
   - `snapshot`
   - `history`
   - `bindings`
   - `docs`
   - `export`
   - `restore`
   - `serve`
3. A new JSON-only HTTP package with routes for session lifecycle, evaluation, history, bindings, docs, export, and restore.
4. Restore-aware orchestration so CLI and server can operate on persisted sessions even after process restart.

## Non-Goals

This ticket does not need to:

1. remove the browser UI static assets yet,
2. replace the Bobatea REPL,
3. design authentication or multi-tenant policy,
4. promise that the old `cmd/repl` or `cmd/web-repl` binaries disappear in this exact ticket.

## Proposed Architecture

### New package split

```text
cmd/goja-repl/
  main.go
  root.go
  support.go
  cmd_*.go

pkg/replapi/
  app.go              # restore-aware orchestration over replsession + repldb

pkg/replhttp/
  handler.go          # JSON API only
```

### Why add `pkg/replapi`

The transport surfaces need more than raw `replsession.Service` and more than raw SQL helpers. They need a small app-level seam that knows how to:

- create a live session,
- restore a persisted session into a live runtime,
- auto-restore on snapshot/eval when appropriate,
- expose store-backed history/export operations.

Without that seam, the CLI and HTTP server would duplicate restore logic.

## Restore Contract

### Rule

If a caller asks to evaluate or snapshot a session that is not currently live in memory, but does exist in SQLite, the system should restore it by replaying persisted raw source cells in order.

### Implications

- `create` always creates a fresh live session and a durable row.
- `eval` should auto-restore when the target session is persisted but not live.
- `snapshot` should auto-restore for the same reason.
- `history`, `docs`, and `export` can stay store-backed and do not require a live runtime.
- `restore` should explicitly rebuild a live runtime and return the resulting live snapshot.

## CLI Design

### Root command

Use a new binary `goja-repl` with a Cobra root that follows the repo’s Glazed/help/logging conventions.

Persistent flags should include:

- `--db-path`
- `--plugin-dir`
- `--allow-plugin-module`
- `--log-level`

### Output shape

Default command output should be machine-friendly JSON. That makes the CLI immediately useful to agents and scripts.

### Initial subcommands

```text
goja-repl create
goja-repl eval --session-id <id> --source 'const x = 1; x'
goja-repl snapshot --session-id <id>
goja-repl history --session-id <id>
goja-repl bindings --session-id <id>
goja-repl docs --session-id <id>
goja-repl export --session-id <id>
goja-repl restore --session-id <id>
goja-repl serve --addr 127.0.0.1:3090
```

## HTTP Design

### Routes

The JSON server should expose:

```text
GET    /api/sessions
POST   /api/sessions
GET    /api/sessions/{id}
DELETE /api/sessions/{id}
POST   /api/sessions/{id}/evaluate
POST   /api/sessions/{id}/restore
GET    /api/sessions/{id}/history
GET    /api/sessions/{id}/bindings
GET    /api/sessions/{id}/docs
GET    /api/sessions/{id}/export
```

### Response model

Reuse `replsession` and `repldb` DTOs where possible instead of inventing parallel transport-only shapes.

## Concrete Task Breakdown

### Task 1: Finalize the ticket plan and task list

Deliverables:

- this design doc,
- concrete tasks,
- diary scaffold.

### Task 2: Add restore-aware orchestration

Deliverables:

- `pkg/replapi`,
- session restore helper path,
- session list/export/history wrappers.

### Task 3: Add the JSON-only HTTP transport

Deliverables:

- `pkg/replhttp`,
- handler tests,
- no embedded browser assets.

### Task 4: Add the `goja-repl` binary

Deliverables:

- root command,
- persistent flags,
- create/eval/snapshot/history/bindings/docs/export/restore/serve subcommands.

### Task 5: Validate and document

Deliverables:

- targeted tests,
- ticket diary/changelog/task updates,
- focused commits.

## Risks and Design Decisions

### Why JSON-only server instead of extending `webrepl`

Because we want a transport boundary that is not biased toward browser assets and can later be used by any client.

### Why auto-restore in the orchestration layer

Because that behavior is needed by both CLI and server, but it is not the responsibility of raw SQL helpers or of the bare in-memory session service.

### Why keep old binaries for now

Because removing them is cleanup work, not the critical path for getting the CLI/server product surface working first.

## Open Questions

1. Whether to add an interactive `repl` subcommand in this ticket or keep the first CLI surface command-oriented.
2. Whether `bindings` and `docs` should always use store-backed exports, or prefer live snapshot state when a session is already loaded.

The default implementation should bias toward simpler agent- and script-friendly commands first.
