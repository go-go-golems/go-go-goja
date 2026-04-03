---
Title: SQLite Persistence Replay and Export Implementation Plan
Ticket: GOJA-21-PERSISTENT-REPL-SQLITE
Status: active
Topics:
    - persistent-repl
    - sqlite
    - architecture
    - webrepl
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/jsdoc/exportsq/exportsq.go
      Note: Existing SQLite writing patterns in this repository
    - Path: pkg/jsdoc/extract/extract.go
      Note: Existing jsdocex extraction logic to reuse for REPL-authored binding docs
    - Path: pkg/jsdoc/model/store.go
      Note: Existing in-memory doc model that informs stored REPL doc payloads
    - Path: pkg/repldb/schema.go
      Note: Initial durable schema for sessions evaluations bindings and docs
    - Path: pkg/repldb/store.go
      Note: SQLite store bootstrap and lifecycle for the durable REPL database
    - Path: pkg/repldb/store_test.go
      Note: Bootstrap tests proving schema creation and version recording
    - Path: pkg/replsession/rewrite.go
      Note: Binding rewrite layer that determines which names and declarations are persisted
    - Path: pkg/replsession/service.go
      Note: Current session lifecycle and evaluation pipeline that must gain persistence hooks
    - Path: pkg/replsession/types.go
      Note: Transport-neutral cell and binding types that inform the persisted schema
ExternalSources:
    - https://github.com/mattn/go-sqlite3
Summary: Detailed phase-1 implementation plan for durable session history, binding metadata, replay, restore, export, and REPL-authored docs in SQLite.
LastUpdated: 2026-04-03T18:01:00-04:00
WhatFor: Use this guide to implement the SQLite-backed persistence layer for the extracted persistent REPL kernel.
WhenToUse: Use when building or reviewing the session store, replay flow, export path, and REPL-scoped JSDoc persistence work.
---


# SQLite Persistence Replay and Export Implementation Plan

## Executive Summary

`GOJA-20` extracted the transport-neutral session kernel into `pkg/replsession`. `GOJA-21` is the next real dependency in the roadmap: make that kernel durable. The system should keep running from memory during a live process, but every meaningful session event should also be written to SQLite so that we can inspect, export, replay, and partially restore a session after the fact.

The implementation should stay honest about what can be restored. We are not serializing an entire goja heap. We are storing enough structure to:

- query session history cheaply,
- reconstruct a session by replaying cells in order,
- restore selected binding values when they are JSON-like and explicitly exportable,
- preserve binding metadata and JSDoc written inside the REPL.

This ticket should end with a reusable `pkg/repldb` package, `pkg/replsession` persistence hooks, tests for write and replay behavior, and ticket docs that let the next engineer continue into CLI/server work without rediscovering the storage model.

## Problem Statement

The extracted `replsession.Service` currently behaves like a volatile notebook kernel. It can create sessions, evaluate cells, and retain binding metadata only while the process is alive. Once the process exits:

- evaluation history is gone,
- binding metadata is gone,
- console output history is gone,
- REPL-authored docs are gone,
- there is no query surface for later export or inspection,
- there is no reliable reconstruction path for a session.

That prevents the next phases from being built cleanly. A CLI history subcommand, a server-side session inspection endpoint, or an export command all need a durable system boundary. If we skip that boundary now, the later transports will either duplicate state management or expose a partial product that becomes harder to retrofit.

## Goals

This phase should deliver the following concrete behaviors:

1. Creating a session writes a durable session row.
2. Evaluating a cell writes a durable evaluation record plus the metadata needed to inspect that cell later.
3. Binding changes are tracked durably per evaluation, including whether the value is snapshot-exportable.
4. REPL-authored JSDoc extracted from a cell is stored and associated with the correct binding version.
5. A stored session can be reconstructed by replaying persisted evaluations in order.
6. A stored session can be exported as a structured notebook/history artifact without requiring a live runtime.

## Non-Goals

This phase should not attempt to:

1. serialize arbitrary closures, prototypes, class instances, or host objects,
2. finish the CLI or JSON server UX,
3. delete the old `pkg/webrepl` transport package yet,
4. build a complex migration framework beyond what this repository needs for a first durable schema.

## Recommended Architecture

### Package split

The implementation should add a new storage package and keep the dependency direction one-way:

```text
pkg/replsession/
  service.go             # runtime/session orchestration
  types.go               # reports and transport-neutral DTOs
  rewrite.go             # persistence-aware source rewrite

pkg/repldb/
  store.go               # DB open/bootstrap/transactions
  schema.go              # schema bootstrap SQL
  sessions.go            # session CRUD
  evaluations.go         # evaluation rows and cell history
  bindings.go            # binding and binding-version persistence
  docs.go                # persisted REPL-authored JSDoc
  replay.go              # ordered readback and export helpers
```

`replsession` should depend on `repldb` only through a narrow interface. That keeps the service testable and leaves room for a future in-memory test fake.

### Data flow

```text
source cell
   |
   v
replsession.Evaluate
   |
   +--> parse/rewrite/analyze
   |
   +--> execute in goja runtime
   |
   +--> diff globals / extract docs / classify bindings
   |
   +--> persist evaluation + binding versions + docs
   |
   v
EvaluationResult

SQLite becomes the durable event log, not the execution authority.
The live goja runtime remains authoritative for "what exists right now" during the process lifetime.
```

## Schema Design

### Why event-oriented tables

We need both current-state inspection and history. The cleanest shape is:

- one table for stable session identity,
- one table per evaluation,
- one table that records binding versions over time,
- one table for doc blocks derived from cells,
- optional child tables for console lines and replay/export artifacts.

This lets us answer:

- "what happened in session X?"
- "what did binding `foo` look like after cell 7?"
- "which bindings have user-authored docs?"
- "replay the session from cell 1 through N"

### Proposed tables

```sql
sessions (
  session_id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT,
  engine_kind TEXT NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}'
)

evaluations (
  evaluation_id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL,
  cell_id INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  raw_source TEXT NOT NULL,
  rewritten_source TEXT NOT NULL,
  ok INTEGER NOT NULL,
  result_json TEXT NOT NULL,
  error_text TEXT NOT NULL DEFAULT '',
  analysis_json TEXT NOT NULL,
  globals_before_json TEXT NOT NULL,
  globals_after_json TEXT NOT NULL,
  FOREIGN KEY(session_id) REFERENCES sessions(session_id),
  UNIQUE(session_id, cell_id)
)

console_events (
  console_event_id INTEGER PRIMARY KEY AUTOINCREMENT,
  evaluation_id INTEGER NOT NULL,
  stream TEXT NOT NULL,
  seq INTEGER NOT NULL,
  text TEXT NOT NULL,
  FOREIGN KEY(evaluation_id) REFERENCES evaluations(evaluation_id),
  UNIQUE(evaluation_id, seq)
)

bindings (
  binding_id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL,
  name TEXT NOT NULL,
  created_at TEXT NOT NULL,
  latest_evaluation_id INTEGER,
  latest_cell_id INTEGER,
  UNIQUE(session_id, name),
  FOREIGN KEY(session_id) REFERENCES sessions(session_id)
)

binding_versions (
  binding_version_id INTEGER PRIMARY KEY AUTOINCREMENT,
  binding_id INTEGER NOT NULL,
  evaluation_id INTEGER NOT NULL,
  cell_id INTEGER NOT NULL,
  action TEXT NOT NULL,
  runtime_type TEXT NOT NULL,
  display_value TEXT NOT NULL,
  summary_json TEXT NOT NULL,
  export_kind TEXT NOT NULL,
  export_json TEXT NOT NULL,
  doc_digest TEXT NOT NULL DEFAULT '',
  FOREIGN KEY(binding_id) REFERENCES bindings(binding_id),
  FOREIGN KEY(evaluation_id) REFERENCES evaluations(evaluation_id),
  UNIQUE(binding_id, evaluation_id)
)

binding_docs (
  binding_doc_id INTEGER PRIMARY KEY AUTOINCREMENT,
  binding_id INTEGER NOT NULL,
  evaluation_id INTEGER NOT NULL,
  cell_id INTEGER NOT NULL,
  symbol_name TEXT NOT NULL,
  source_kind TEXT NOT NULL,
  raw_doc TEXT NOT NULL,
  normalized_json TEXT NOT NULL,
  FOREIGN KEY(binding_id) REFERENCES bindings(binding_id),
  FOREIGN KEY(evaluation_id) REFERENCES evaluations(evaluation_id)
)
```

### Exportability model

Each binding version should carry an `export_kind` value:

- `none`: not exportable as a value snapshot
- `json`: can be represented as a JSON value
- `string`: representable only as a textual display form

This keeps the restore contract explicit:

- replay is the default restoration mechanism,
- JSON snapshots are an optimization and export aid,
- textual display values are for audit/export only, not restore.

## Service Integration Plan

### New service seam

Add an optional persistence dependency to `replsession.Service`:

```go
type Persistence interface {
    CreateSession(ctx context.Context, record SessionRecord) error
    DeleteSession(ctx context.Context, sessionID string, deletedAt time.Time) error
    PersistEvaluation(ctx context.Context, record EvaluationRecord) error
    LoadEvaluations(ctx context.Context, sessionID string) ([]EvaluationRecord, error)
}
```

The exact shape can evolve, but the important rule is that `replsession` must not embed raw SQL.

### Write ordering

The write path should follow this order:

1. Session is created in memory.
2. Session row is inserted in SQLite.
3. Cell is parsed, rewritten, executed, and summarized.
4. Extract docs from raw source.
5. Classify changed bindings into exportability buckets.
6. Persist evaluation, console events, binding rows, binding versions, and docs in one transaction.
7. Return the existing `EvaluationResult`.

### Failure policy

Persistence failures should fail the evaluation request. A partially durable notebook kernel is worse than an explicitly failing one, because the later replay/export assumptions would silently become false.

That means:

- if runtime execution fails, persist the failed evaluation with `ok=0`,
- if runtime execution succeeds but the DB write fails, return an error and do not pretend the evaluation is durably recorded.

## Replay and Restore Plan

### Replay contract

Restoration should rebuild a fresh session by replaying persisted `raw_source` cells in cell order. That keeps behavior closest to what the user actually typed and avoids committing to a serialization format for arbitrary runtime values.

### Replay algorithm

```pseudo
RestoreSession(sessionID):
  history = store.LoadEvaluations(sessionID ordered by cell_id)
  newSession = service.CreateSession()

  for each evaluation in history:
    result = service.Evaluate(newSession.ID, evaluation.raw_source)
    if result.error and evaluation.ok:
      return mismatch error

  return newSession
```

### Optional snapshot use

Snapshot values should not replace replay. They are useful for:

- faster exports,
- debugging binding changes without spinning up goja,
- future selective restore or cache-warm features.

## REPL-Authored JSDoc Plan

The REPL should accept jsdocex-style doc patterns inside a cell, extract them using `pkg/jsdoc/extract`, and store the normalized doc payload with the binding version written by that cell.

The contract for this ticket should be:

1. extract docs from the user’s raw source,
2. associate each extracted symbol doc with the binding of the same name when present,
3. store both raw and normalized forms,
4. surface the association in persisted query/export results.

This ticket does not need to finish the runtime `docs` command integration, but the stored shape should make that later wiring trivial.

## Concrete Task Breakdown

### Task 1: Finalize the storage model in the ticket

Deliverables:

- schema description in this design doc,
- ordered implementation tasks in `tasks.md`,
- initial diary scaffolding.

### Task 2: Add `pkg/repldb` bootstrap and schema creation

Deliverables:

- DB open helper,
- schema bootstrap path,
- smoke tests proving tables are created.

### Task 3: Add session and evaluation write APIs

Deliverables:

- create/delete session persistence,
- evaluation persistence with success/failure status,
- transactional write path.

### Task 4: Persist binding versions and exportability metadata

Deliverables:

- changed binding persistence,
- JSON export classification,
- tests for primitives, arrays/objects, and non-exportable values.

### Task 5: Persist REPL-authored docs

Deliverables:

- doc extraction from cell source,
- binding-to-doc association,
- durable storage and tests.

### Task 6: Add replay and export readers

Deliverables:

- ordered evaluation loader,
- replay helper,
- structured export DTOs.

### Task 7: Validate and document

Deliverables:

- targeted tests,
- ticket diary/changelog updates,
- focused commits per slice.

## Testing Plan

At minimum, this ticket should add:

1. store bootstrap test using a temp DB path,
2. session create/delete persistence test,
3. successful evaluation persistence test,
4. failed evaluation persistence test,
5. binding version persistence test for exportable and non-exportable values,
6. doc extraction persistence test,
7. replay test that reconstructs a session from persisted cells.

Recommended commands during the work:

```bash
go test ./pkg/repldb ./pkg/replsession
go test ./... 
golangci-lint run -v
```

## Risks and Design Decisions

### Why SQLite now instead of later

Because CLI and server work will immediately need stable history and query semantics. Deferring persistence would only move complexity into later phases.

### Why replay-first restore

Because it matches the user’s source of truth and avoids a misleading promise that arbitrary goja values can be snapshotted and recovered.

### Why preserve both raw and rewritten source

`raw_source` is the user’s truth for replay and export. `rewritten_source` is critical for debugging the lexical binding persistence layer.

### Why fail on DB write errors

Because later commands must be able to trust that "successful evaluation" means "durably recorded evaluation".

## Alternatives Considered

### Alternative 1: Persist only raw cells and nothing else

Rejected because it would make exports and queries weak, and force every inspection command to boot a replay runtime.

### Alternative 2: Serialize the full runtime heap

Rejected because it is not realistic for arbitrary goja values and would create a misleading product promise.

### Alternative 3: Store docs outside the session DB

Rejected because REPL-authored docs are session-scoped artifacts and need to version with binding changes.

## Open Questions

1. Whether `console_events` should be implemented in this ticket or deferred if the runtime currently does not expose console capture cleanly.
2. Whether binding deletions need an explicit tombstone action in `binding_versions` for phase 1.
3. Whether the schema should store a compact analysis summary only, or the full analysis payload from `replsession`.

The default implementation should choose the simplest option that preserves replay/export correctness.

## References

- `GOJA-20` design doc `02-cli-and-server-first-persistent-repl-architecture-and-implementation-guide.md`
- `go-go-goja/pkg/replsession/service.go`
- `go-go-goja/pkg/jsdoc/extract/extract.go`
- `go-go-goja/pkg/jsdoc/exportsq/exportsq.go`
