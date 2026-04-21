---
Title: CLI and Server First Persistent REPL Architecture and Implementation Guide
Ticket: GOJA-20-WEBREPL-ARCHITECTURE
Status: active
Topics:
    - webrepl
    - architecture
    - rest-api
    - llm-agent-integration
    - persistent-repl
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/js-repl/main.go
      Note: |-
        Existing richer TUI REPL surface
        Existing richer REPL surface with help and completion
    - Path: go-go-goja/cmd/repl/main.go
      Note: |-
        Existing simple CLI REPL surface
        Existing basic CLI REPL surface to preserve or subsume
    - Path: go-go-goja/cmd/web-repl/main.go
      Note: |-
        Existing HTTP entry point
        Existing HTTP prototype entry point
    - Path: go-go-goja/engine/factory.go
      Note: Runtime construction seam
    - Path: go-go-goja/engine/runtime.go
      Note: Runtime lifecycle and closers
    - Path: go-go-goja/pkg/docaccess/runtime/registrar.go
      Note: Runtime-scoped docs module registration
    - Path: go-go-goja/pkg/jsdoc/exportsq/exportsq.go
      Note: |-
        Existing SQLite export patterns for docs
        SQLite export patterns that inform the new store design
    - Path: go-go-goja/pkg/jsdoc/extract/extract.go
      Note: |-
        Current jsdocex-compatible extraction engine
        jsdocex-compatible extraction engine to reuse
    - Path: go-go-goja/pkg/jsdoc/model/store.go
      Note: In-memory doc store shape
    - Path: go-go-goja/pkg/repl/evaluators/javascript/docs_resolver.go
      Note: Existing plugin-doc resolution path
    - Path: go-go-goja/pkg/repl/evaluators/javascript/evaluator.go
      Note: |-
        Existing docs/autocomplete/help integration
        Current docs/autocomplete/help runtime surface
    - Path: go-go-goja/pkg/runtimeowner/runner.go
      Note: Single-goroutine access discipline for goja
    - Path: go-go-goja/pkg/webrepl/rewrite.go
      Note: Current lexical-binding persistence rewrite
    - Path: go-go-goja/pkg/webrepl/server.go
      Note: Prototype HTTP transport
    - Path: go-go-goja/pkg/webrepl/service.go
      Note: |-
        Prototype persistent session pipeline
        Prototype session evaluation pipeline and binding tracking
ExternalSources:
    - https://github.com/dop251/goja
    - https://pkg.go.dev/github.com/dop251/goja_nodejs
    - https://pkg.go.dev/net/http
    - https://github.com/mattn/go-sqlite3
Summary: Replaces the prototype-centered plan with a shared persistent REPL kernel built for CLI and JSON server first, with SQLite-backed evaluation history and REPL-scoped JSDoc metadata.
LastUpdated: 2026-04-03T16:30:04.776552774-04:00
WhatFor: Use this document when designing or implementing the long-lived persistent JavaScript REPL that should power both command-line and server workflows, while keeping rich introspection, persistence, and documentation.
WhenToUse: Use when deciding how to evolve the current webrepl prototype into a reusable subsystem, especially when SQLite persistence, export/replay, or JSDoc-on-bindings are in scope.
---


# CLI and Server First Persistent REPL Architecture and Implementation Guide

## Executive Summary

The current ticket already contains a strong prototype analysis, but its center of gravity is still the staged `pkg/webrepl` package. That is the wrong target for the next implementation step. The codebase already has three partial user surfaces:

1. `cmd/repl/main.go` is the smallest interactive entry point and proves the runtime can be used from a plain terminal.
2. `cmd/js-repl/main.go` plus `pkg/repl/evaluators/javascript/evaluator.go` shows that the project already has a richer evaluator surface with docs, completion, and help integration.
3. `cmd/web-repl/main.go` plus `pkg/webrepl/service.go` proves that long-lived HTTP sessions, rewrite-based persistence, and runtime diffing are feasible.

The correct next design is therefore not "make the web REPL bigger". The correct next design is "extract a shared persistent REPL kernel that exposes two first-class transports immediately: CLI and JSON server". The browser UI should become optional and come later.

This document recommends a new layered architecture:

1. A runtime/session core package that owns session lifecycle, evaluation, rewrite, static analysis, runtime diffing, and session-scoped documentation.
2. A SQLite-backed persistence package that records sessions, evaluations, console output, binding metadata, binding snapshots where possible, and JSDoc metadata extracted from REPL cells.
3. Two thin transports on top of that core:
   - a CLI surface for humans and automation,
   - a JSON HTTP server for agents and future browser clients.
4. An optional browser UI after the core is stable.

The most important addition beyond the previous plan is that persistence is not treated as an export-only afterthought. SQLite becomes part of the core system boundary from the beginning, because it gives us queryability, exportability, auditability, and the only realistic path to later restoration. We should not promise true VM serialization. We should promise deterministic replay plus partial value snapshotting for serializable bindings.

The second important addition is REPL-scoped `jsdocex` support. The user should be able to submit `__doc__(...)` and `doc\`...\`` metadata in a cell, have that metadata attached to the relevant binding, store it in SQLite, and surface it through the same runtime `docs` module and transport APIs that the richer TUI evaluator already knows how to consume.

## Problem Statement

### What we are actually building

We are not building "a web page that can run JavaScript". We are building a reusable subsystem inside `go-go-goja` that lets a long-lived goja runtime be:

1. addressed through stable sessions,
2. evaluated incrementally across cells,
3. introspected statically and at runtime,
4. persisted and queried later,
5. documented with REPL-authored symbol docs,
6. exposed through both CLI and server interfaces.

### Why the current plan is insufficient

The staged prototype is useful evidence, but it is still shaped around the HTTP demo package:

1. `pkg/webrepl/service.go:121-257` contains the richest evaluation pipeline, but it is packaged as "webrepl" even though most of the logic is not web-specific.
2. `pkg/webrepl/server.go:16-115` mixes static asset serving and API routing directly into the transport layer.
3. `cmd/web-repl/main.go:38-74` is a standalone binary instead of one transport on top of a shared product surface.
4. `pkg/repl/evaluators/javascript/evaluator.go:103-176` already has a second, partially overlapping runtime composition story for docs/help/autocomplete.
5. `pkg/webrepl/service.go:378-401` uses promise polling with `time.Sleep(5 * time.Millisecond)`, which is acceptable for a prototype but not a design center.
6. `pkg/webrepl/service.go:95-119` and `260-283` keep all session state in memory only. Once the process exits, the session history is gone.

### Constraints we must respect

Observed constraints from the codebase:

1. `goja.Runtime` is not safe for concurrent goroutine access. The codebase already handles this with `pkg/runtimeowner/runner.go:62-159`.
2. Runtime creation and module wiring already have a clean seam in `engine/factory.go:141-205`.
3. Runtime lifecycle and cleanup hooks already exist in `engine/runtime.go:22-109`.
4. Static analysis already exists and is quite capable through `pkg/jsparse` and the inspector analysis packages.
5. Runtime-scoped docs already have a registration pattern through `pkg/docaccess/runtime/registrar.go:58-130`.
6. `pkg/jsdoc/extract/extract.go:13-87` and `140-240` already parse jsdocex-style sentinel patterns.
7. `pkg/jsdoc/exportsq/exportsq.go:55-258` already demonstrates the repository is comfortable with SQLite-backed durable doc exports.

### Non-goals

This design does not attempt to:

1. serialize arbitrary goja heap objects or closures bit-for-bit,
2. make the browser UI the primary interface,
3. design a public multi-tenant cloud service,
4. preserve backward compatibility with the prototype package layout if a cleaner split is better.

## Proposed Solution

### Recommendation in one sentence

Create a new shared package family centered on a persistent session kernel, then reimplement CLI and server as thin adapters over it, using SQLite as the durable event log and JSDoc store.

### Proposed package split

The cleanest structure is to stop growing `pkg/webrepl` directly and instead split responsibilities like this:

```text
cmd/goja-repl/
  main.go                  # one binary with subcommands: repl, eval, serve, history, export

pkg/replsession/
  service.go               # session lifecycle API
  session.go               # per-session state
  evaluate.go              # evaluation pipeline
  rewrite.go               # lexical persistence rewrite
  snapshot.go              # runtime snapshots and diffs
  docs.go                  # REPL-scoped docs ingestion and lookup
  types.go                 # transport-neutral result structs

pkg/repldb/
  schema.sql or migrations/
  store.go                 # open DB, migrations, transactions
  sessions.go              # session CRUD
  evaluations.go           # cell/event persistence
  bindings.go              # binding metadata and snapshots
  docs.go                  # persisted JSDoc metadata
  export.go                # replay/export helpers

pkg/replhttp/
  handler.go               # JSON API only
  routes.go

pkg/replcli/
  app.go                   # shared CLI command wiring
  commands_*.go

pkg/replui/                # optional later browser assets and handlers
```

### Why a new core package instead of extending `pkg/webrepl`

Because the current prototype package already violates the boundary we want:

1. `pkg/webrepl/service.go` is mostly transport-neutral business logic.
2. `pkg/webrepl/server.go` is the actual web layer.
3. The package name pressures the implementation toward browser concerns even when the user now wants CLI and server first.

The redesign should preserve good code from `pkg/webrepl`, not preserve the package boundary.

### High-level runtime architecture

```text
                     +------------------------------+
                     |        CLI / HTTP API        |
                     |  repl, eval, serve, export   |
                     +---------------+--------------+
                                     |
                                     v
                     +------------------------------+
                     |      replsession.Service     |
                     | session create/eval/query    |
                     +---+----------------------+---+
                         |                      |
                         v                      v
               +----------------+      +-------------------+
               |  engine.Runtime |      |    repldb.Store   |
               |  goja + loop +  |      | sessions/cells/   |
               |  runtimeowner    |      | bindings/docs     |
               +----------------+      +-------------------+
                         |
                         v
         +-----------------------------------------------+
         | jsparse + inspector + docaccess + jsdoc       |
         | static analysis, runtime diff, docs lookup    |
         +-----------------------------------------------+
```

### Core design principles

1. The session kernel is the product. CLI and server are adapters.
2. Every evaluation is persisted as an event, even if replay is the only restore mechanism initially.
3. Session state is authoritative in memory while the process is alive; SQLite is authoritative for history, querying, export, and reconstruction.
4. Binding restoration is best-effort and capability-based:
   - primitives and JSON-like values can be snapshotted,
   - functions, classes, closures, and complex host objects are restored by replay, not value serialization.
5. JSDoc metadata is treated like session data, not like a separate documentation product.
6. The HTTP API should be JSON-first and UI-agnostic.

### Evaluation pipeline

The new core should keep the useful steps from `pkg/webrepl/service.go`, but make the order explicit and persistence-aware.

```pseudo
Evaluate(sessionID, source, options):
  session = loadSession(sessionID)
  lock session

  cellID = allocateCellID()
  rawSource = source

  tsTree = maybeParseTreeSitter(rawSource)
  analysis = jsparse.Analyze(cellFilename(cellID), rawSource, nil)

  docExtraction = repldocs.ExtractFromCell(rawSource)
  sanitizedSource = repldocs.StripOrNoopDocSentinels(rawSource)

  rewrite = buildRewrite(sanitizedSource, analysis, cellID)
  beforeGlobals = snapshotGlobals()

  start transaction
    persist evaluation_requested event
    persist raw source, analysis summary, extracted docs
  commit

  runtimeResult = execute rewritten cell through runtimeowner
  afterGlobals = snapshotGlobals()
  diff = compare(beforeGlobals, afterGlobals)
  update session binding catalog
  refresh binding runtime metadata
  attach extracted docs to matching bindings

  exportableSnapshots = collectSerializableBindingValues()

  start transaction
    persist evaluation_completed event
    persist console events
    persist binding changes
    persist serializable binding snapshots
    persist doc rows
  commit

  return transport-neutral evaluation report
```

### SQLite as a first-class subsystem

#### What SQLite is for

SQLite is not only for "save history". It should support:

1. session and cell audit trail,
2. querying what happened and when,
3. exporting a session cleanly,
4. deterministic rebuild by replay,
5. attaching documentation and metadata to bindings,
6. later tooling such as `history`, `search`, `diff`, `explain`, and notebook export.

#### What SQLite is not for

SQLite is not a magic snapshot of a live goja heap. We should explicitly document that:

1. closures cannot be serialized generically,
2. host-backed objects cannot be reconstructed safely from raw pointers,
3. prototype state is best observed and replayed, not marshaled blindly.

The restoration contract should therefore be:

1. replay all successful cells in order,
2. optionally preload serializable binding values when they are marked exportable,
3. warn when a session depends on non-exportable bindings.

### Proposed SQLite schema

The schema below is deliberately normalized enough for querying, but still practical for a first intern-friendly implementation.

```sql
CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL,
  closed_at TEXT,
  runtime_profile TEXT,
  metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE evaluations (
  session_id TEXT NOT NULL,
  cell_id INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  status TEXT NOT NULL,
  source_raw TEXT NOT NULL,
  source_sanitized TEXT NOT NULL,
  source_rewritten TEXT NOT NULL,
  result_preview TEXT NOT NULL DEFAULT '',
  error_text TEXT NOT NULL DEFAULT '',
  awaited INTEGER NOT NULL DEFAULT 0,
  duration_ms INTEGER NOT NULL DEFAULT 0,
  static_report_json TEXT NOT NULL,
  runtime_report_json TEXT NOT NULL,
  PRIMARY KEY (session_id, cell_id)
);

CREATE TABLE console_events (
  session_id TEXT NOT NULL,
  cell_id INTEGER NOT NULL,
  seq INTEGER NOT NULL,
  kind TEXT NOT NULL,
  message TEXT NOT NULL,
  PRIMARY KEY (session_id, cell_id, seq)
);

CREATE TABLE bindings (
  session_id TEXT NOT NULL,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  origin TEXT NOT NULL,
  declared_in_cell INTEGER,
  last_updated_cell INTEGER,
  declared_line INTEGER,
  declared_snippet TEXT NOT NULL DEFAULT '',
  docs_summary TEXT NOT NULL DEFAULT '',
  docs_prose TEXT NOT NULL DEFAULT '',
  exportability TEXT NOT NULL DEFAULT 'unknown',
  PRIMARY KEY (session_id, name)
);

CREATE TABLE binding_versions (
  session_id TEXT NOT NULL,
  name TEXT NOT NULL,
  cell_id INTEGER NOT NULL,
  change_kind TEXT NOT NULL,
  value_kind TEXT NOT NULL,
  preview TEXT NOT NULL DEFAULT '',
  snapshot_json TEXT,
  snapshot_format TEXT NOT NULL DEFAULT '',
  function_mapping_json TEXT NOT NULL DEFAULT '',
  runtime_metadata_json TEXT NOT NULL DEFAULT '{}',
  PRIMARY KEY (session_id, name, cell_id)
);

CREATE TABLE jsdoc_packages (
  session_id TEXT NOT NULL,
  cell_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  prose TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (session_id, cell_id, name)
);

CREATE TABLE jsdoc_symbols (
  session_id TEXT NOT NULL,
  cell_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  prose TEXT NOT NULL DEFAULT '',
  docpage TEXT NOT NULL DEFAULT '',
  line INTEGER NOT NULL DEFAULT 0,
  raw_json TEXT NOT NULL DEFAULT '{}',
  PRIMARY KEY (session_id, cell_id, name)
);

CREATE TABLE jsdoc_examples (
  session_id TEXT NOT NULL,
  cell_id INTEGER NOT NULL,
  id TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  raw_json TEXT NOT NULL DEFAULT '{}',
  PRIMARY KEY (session_id, cell_id, id)
);
```

#### Query examples the schema enables

```sql
-- Show the last 20 cells in a session.
SELECT cell_id, status, result_preview, created_at
FROM evaluations
WHERE session_id = ?
ORDER BY cell_id DESC
LIMIT 20;

-- Show when binding "foo" changed.
SELECT cell_id, change_kind, value_kind, preview
FROM binding_versions
WHERE session_id = ? AND name = 'foo'
ORDER BY cell_id;

-- Search JSDoc summaries authored inside a session.
SELECT name, summary, cell_id
FROM jsdoc_symbols
WHERE session_id = ? AND summary LIKE '%' || ? || '%'
ORDER BY cell_id;
```

### Binding restoration model

The system should classify bindings by exportability on each update:

1. `json`
   - primitives, arrays, plain objects that survive `Value.Export()` plus JSON marshal.
2. `js-literal`
   - values that are not JSON but can be rendered as a JavaScript initializer safely.
3. `replay-only`
   - functions, classes, closures, promises, host objects, or values tied to external modules.
4. `unknown`
   - classification failed.

Restore logic:

```pseudo
RestoreSession(sessionID):
  create fresh runtime
  load session metadata

  for each successful cell in evaluations ordered by cell_id:
    evaluate source_raw or source_sanitized
    stop on failure unless restore mode is "best-effort"

  for each latest binding_version with exportability in ('json', 'js-literal'):
    optionally compare replay result vs stored snapshot

  produce restore report
```

This design deliberately chooses replay as the baseline because it is honest, inspectable, and aligned with how notebooks and build systems normally recover state.

### REPL-scoped JSDoc support

#### Why this matters

The user specifically wants the REPL author to be able to document bindings while exploring code. That is a high-leverage feature because it turns the REPL from a transient execution tool into a living notebook with explainable symbols.

#### Existing support we should reuse

1. `pkg/jsdoc/extract/extract.go:13-87` parses files or sources into `FileDoc`.
2. `pkg/jsdoc/extract/extract.go:140-240` already understands `__package__`, `__doc__`, `__example__`, and `doc` tagged templates.
3. `pkg/jsdoc/model/store.go:3-89` gives us an indexed in-memory doc store.
4. `pkg/docaccess/runtime/registrar.go:37-130` already knows how to expose JSDoc sources through the runtime `docs` module.

#### Proposed REPL behavior

For REPL cells, support these patterns from day one:

```js
__doc__("answer", {
  summary: "The stable answer binding for this session",
  tags: ["demo", "repl"]
})
const answer = 42

doc`
---
symbol: answer
---
This binding is used by the tutorial examples below.
`
```

Implementation approach:

1. Run jsdoc extraction on the raw cell source before evaluation.
2. Persist extracted package/symbol/example docs to SQLite with `(session_id, cell_id)` ownership.
3. Merge extracted symbol docs onto the current binding catalog if names match declared or existing session bindings.
4. Provide a session-scoped `DocStore` assembled from persisted rows and feed it into `docaccessruntime.NewRegistrar(...)`.
5. Keep sentinel statements runtime-safe by one of these two methods:
   - preferred: install no-op sentinel functions and tag helpers in the runtime,
   - acceptable fallback: strip recognized doc-only statements before execution.

The preferred method is runtime no-ops because it preserves source fidelity. The stripped version is more invasive and makes source-to-runtime comparison harder.

#### Minimal REPL doc runtime helpers

```js
globalThis.__doc__ = (...args) => undefined
globalThis.__package__ = (...args) => undefined
globalThis.__example__ = (...args) => undefined
globalThis.doc = (strings, ...values) => String.raw({ raw: strings }, ...values)
```

The important point is that the extraction path is authoritative, not the runtime helper behavior. The helpers only make the submitted cell executable.

### Shared docs model across CLI, TUI, and server

One of the strongest arguments for this redesign is that the codebase already has a good docs path, but it is split across product surfaces.

Observed facts:

1. `pkg/repl/evaluators/javascript/evaluator.go:121-125` can register runtime docs sources.
2. `pkg/repl/evaluators/javascript/docs_resolver.go:25-55` can resolve runtime docs entries into completion/help.
3. `cmd/web-repl/main.go:53-56` does not currently use that path.

The new session kernel should therefore expose a single runtime docs builder:

```go
type DocsSourceBuilder interface {
    BuildSessionSources(ctx context.Context, sessionID string) ([]docaccessruntime.JSDocSource, error)
}
```

That builder can draw from SQLite and be used by:

1. CLI help and search commands,
2. future TUI completion/help drawers,
3. server endpoints such as `/api/sessions/{id}/docs/search`,
4. runtime JavaScript via `require("docs")`.

### Transport design

#### CLI first-class commands

The CLI should be a stable automation surface, not just an interactive shell. Use one binary with clear subcommands:

```text
goja-repl repl        # interactive session
goja-repl eval        # one-shot cell into a named session
goja-repl serve       # JSON server
goja-repl history     # query persisted cells
goja-repl bindings    # query current or historical bindings
goja-repl docs        # query session-authored docs
goja-repl export      # export session history / notebook bundle
goja-repl restore     # replay a prior session
```

Recommended initial flags:

```text
--session
--sqlite-path
--plugin-dir
--allow-plugin-module
--log-level
```

#### JSON HTTP API

Do not make the initial server depend on a web page. Start with JSON only.

```text
POST   /api/sessions
GET    /api/sessions/{id}
DELETE /api/sessions/{id}

POST   /api/sessions/{id}/evaluate
GET    /api/sessions/{id}/history
GET    /api/sessions/{id}/history/{cellID}
GET    /api/sessions/{id}/bindings
GET    /api/sessions/{id}/bindings/{name}
GET    /api/sessions/{id}/docs/search?q=...
GET    /api/sessions/{id}/docs/{kind}/{name}
POST   /api/sessions/{id}/export
POST   /api/sessions/{id}/restore
```

Minimal example:

```json
POST /api/sessions/session-1/evaluate
{
  "source": "__doc__(\"answer\", {\"summary\":\"main tutorial value\"})\nconst answer = 42\nanswer"
}
```

Response:

```json
{
  "session": { "...": "session summary" },
  "cell": { "...": "evaluation report" },
  "docs": {
    "symbolsAdded": ["answer"]
  }
}
```

The actual payload shape can reuse much of `pkg/webrepl/types.go:5-299`, but the transport-neutral types should live outside a web-named package.

## Design Decisions

### 1. Use a new shared core package instead of growing `pkg/webrepl`

Rationale:

1. The existing service logic is already more general than the package name.
2. The user explicitly wants CLI and server before UI expansion.
3. A neutral core package reduces conceptual drift and lets us keep HTTP, TUI, and CLI adapters thin.

### 2. Make SQLite mandatory for the persistent session product

Rationale:

1. The feature request is explicitly about storing evaluations and bindings for later query and export.
2. Without a durable store, every future feature becomes a side channel or a log scrape.
3. SQLite is already present in the repo and fits local-tool workflows.

### 3. Treat replay as the default restore mechanism

Rationale:

1. It is technically honest.
2. It matches notebook semantics.
3. It keeps failure modes explainable.

### 4. Install no-op JSDoc sentinel helpers in the runtime

Rationale:

1. It keeps raw cell source executable.
2. It preserves source fidelity between stored input and executed input.
3. It makes doc authoring feel natural inside the REPL.

### 5. Keep the current rewrite strategy initially, but isolate it behind a stable interface

Rationale:

1. `pkg/webrepl/rewrite.go:13-87` already proves the basic lexical-persistence trick works.
2. The rewrite layer is where future alternatives can be tested without destabilizing the session API.
3. The rewrite implementation should be swappable once we learn more from real usage.

### 6. Prefer JSON server first, browser UI second

Rationale:

1. Agents and tests benefit more from a stable JSON contract than from a browser.
2. The current UI is prototype scaffolding, not the differentiator.
3. This reduces startup complexity for the first delivery.

## Alternatives Considered

### Alternative 1: Keep `pkg/webrepl` as the main architecture and add features in place

Rejected because:

1. it keeps the wrong package boundary,
2. it encourages UI-first thinking,
3. it makes CLI and TUI feel secondary instead of peer surfaces.

### Alternative 2: Persist only raw cells and ignore binding snapshots

Rejected because:

1. it makes queryability weak,
2. it cannot answer "when did binding X change?",
3. it wastes the runtime diff work we are already doing.

### Alternative 3: Promise full VM/session serialization

Rejected because:

1. goja values are not generically serializable,
2. host objects and closures make this unreliable,
3. it creates a contract we are unlikely to honor safely.

### Alternative 4: Add JSDoc support only in the browser UI layer

Rejected because:

1. the request is about parsing input itself,
2. the docs must be part of the session model,
3. CLI and HTTP callers should get the same capability.

### Alternative 5: Reuse `pkg/jsdoc/exportsq` tables directly for REPL docs

Partially rejected.

We should reuse the patterns, not the exact schema. The REPL needs session and cell ownership, temporal history, and binding attachment. The existing exporter tables are useful inspiration, but they model static document exports, not evolving live sessions.

## Implementation Plan

### Phase 0: Stabilize the ticket and architecture agreement

Deliverables:

1. Land this design doc as the recommended plan.
2. Mark the earlier prototype-centered document as historical reference only.
3. Agree on the package naming and the single-binary CLI approach.

### Phase 1: Extract the shared session kernel

Goal: make evaluation logic transport-neutral.

Implementation steps:

1. Create `pkg/replsession` and move or port the following concepts from `pkg/webrepl`:
   - session lifecycle,
   - cell reports,
   - rewrite pipeline,
   - runtime snapshots and diffs,
   - console capture,
   - binding catalog.
2. Preserve `runtimeowner` usage for every VM call.
3. Preserve `engine.Factory` as the runtime composition seam.
4. Keep the current in-memory session state model initially, but define store interfaces from day one.

Suggested interfaces:

```go
type Store interface {
    CreateSession(ctx context.Context, session SessionMeta) error
    AppendEvaluation(ctx context.Context, record EvaluationRecord) error
    UpsertBindings(ctx context.Context, sessionID string, bindings []BindingRecord) error
    QueryHistory(ctx context.Context, sessionID string) ([]EvaluationRecord, error)
}

type Service interface {
    CreateSession(ctx context.Context, opts CreateSessionOptions) (*SessionSummary, error)
    Evaluate(ctx context.Context, sessionID, source string, opts EvaluateOptions) (*EvaluateResult, error)
    Snapshot(ctx context.Context, sessionID string) (*SessionSummary, error)
    DeleteSession(ctx context.Context, sessionID string) error
}
```

### Phase 2: Add SQLite persistence and replay/export

Goal: durable history with useful queries.

Implementation steps:

1. Create `pkg/repldb`.
2. Add schema creation and migration bootstrap.
3. Write all session and evaluation events transactionally.
4. Persist binding version rows after every successful evaluation.
5. Add exportability classification for bindings.
6. Implement `history`, `bindings`, and `export` CLI commands against the DB.
7. Implement `restore` by replay from stored evaluations.

Important engineering detail:

1. Use a single writer connection pattern or explicit transactions.
2. Prefer WAL mode and a sensible busy timeout.
3. Keep JSON payload columns for reports to avoid premature table explosion.

### Phase 3: Integrate REPL-scoped JSDoc

Goal: bind docs authoring directly to session evaluation.

Implementation steps:

1. Add `repldocs.ExtractFromCell(source string)` using `pkg/jsdoc/extract`.
2. Install no-op doc sentinel helpers in the runtime initializer path.
3. Persist extracted package/symbol/example docs in SQLite.
4. Merge symbol docs into binding summaries.
5. Build a session `DocStore` from persisted rows.
6. Feed that store into `docaccessruntime.NewRegistrar(...)`.
7. Add CLI and HTTP docs query surfaces.

Verification cases:

1. document a binding before declaration in the same cell,
2. document a binding after declaration in a later cell,
3. update a symbol summary in a later cell and verify replacement behavior,
4. query docs via both CLI and HTTP.

### Phase 4: Build the JSON server

Goal: stable agent-facing transport.

Implementation steps:

1. Create `pkg/replhttp`.
2. Port only the necessary JSON routes from `pkg/webrepl/server.go`.
3. Remove embedded static UI from the core server package.
4. Add server endpoints for history, bindings, docs, export, and restore.
5. Add graceful shutdown using the same pattern already present in `cmd/web-repl/main.go:59-74`.

### Phase 5: Optional browser UI

Goal: human-friendly interface after the data model is stable.

Implementation steps:

1. Add a thin UI client that consumes the JSON API.
2. Keep all logic on the API side.
3. Treat the UI as replaceable.

## Implementation Guide for a New Intern

If a new intern were starting tomorrow, I would tell them to read the system in this order:

1. `engine/factory.go:141-205`
   - understand how a runtime is created.
2. `engine/runtime.go:22-109`
   - understand lifecycle and closers.
3. `pkg/runtimeowner/runner.go:62-159`
   - understand why all VM access must be serialized.
4. `pkg/webrepl/service.go:121-257`
   - understand the current prototype evaluation pipeline.
5. `pkg/webrepl/rewrite.go:13-87`
   - understand how lexical bindings survive across cells.
6. `pkg/repl/evaluators/javascript/evaluator.go:121-176`
   - understand the existing docs/help composition path.
7. `pkg/docaccess/runtime/registrar.go:58-130`
   - understand how docs sources become runtime modules.
8. `pkg/jsdoc/extract/extract.go:140-240`
   - understand jsdocex-compatible extraction.
9. `pkg/jsdoc/model/store.go:3-89`
   - understand the in-memory docs shape.
10. `pkg/jsdoc/exportsq/exportsq.go:55-258`
   - understand the existing SQLite export style.

Then I would tell them to implement in this exact order:

1. extract transport-neutral session types,
2. extract service logic,
3. add the DB interface,
4. add SQLite writes,
5. add restore/export commands,
6. add JSDoc extraction and session docs,
7. wire the JSON server,
8. only then revisit UI work.

## Testing Strategy

### Unit tests

Add tests for:

1. rewrite behavior across `let`, `const`, `class`, and final-expression capture,
2. exportability classification for binding values,
3. SQLite round-trip for sessions, evaluations, bindings, and docs,
4. JSDoc extraction from REPL-style cells,
5. session doc store assembly from SQLite rows.

### Integration tests

Add end-to-end tests for:

1. `repl -> eval -> history -> export -> restore`,
2. `server -> create session -> evaluate -> query bindings`,
3. JSDoc authing flow:
   - submit docs,
   - query docs,
   - verify binding summary is updated.

### Concurrency and lifecycle tests

Add tests for:

1. session deletion during idle state,
2. cancellation of long-running evaluation,
3. shutdown with open sessions,
4. repeated create/evaluate/delete loops against the same SQLite path.

### Manual validation commands

These are the first commands I would expect an intern to run once the first phases land:

```bash
go test ./...
go run ./cmd/goja-repl repl --sqlite-path /tmp/goja-repl.sqlite
go run ./cmd/goja-repl eval --session demo --sqlite-path /tmp/goja-repl.sqlite --code 'const answer = 42; answer'
go run ./cmd/goja-repl history --session demo --sqlite-path /tmp/goja-repl.sqlite
go run ./cmd/goja-repl export --session demo --sqlite-path /tmp/goja-repl.sqlite
go run ./cmd/goja-repl serve --sqlite-path /tmp/goja-repl.sqlite
```

## Open Questions

1. Should `cmd/repl`, `cmd/js-repl`, and `cmd/web-repl` be kept temporarily as compatibility wrappers, or should we cut over directly to one new binary?
2. Do we want to persist full `StaticReport` and `RuntimeReport` JSON blobs forever, or prune heavyweight fields like flattened AST/CST rows by configuration?
3. Should `__example__` be supported in phase 1 of REPL docs, or delayed until notebook export exists?
4. How aggressively do we want to auto-merge later-cell docs onto an existing binding with the same name?
5. Do we want session expiration and garbage collection in the first server version, or only manual close/delete?

## References

### Current code references

1. `cmd/repl/main.go`
2. `cmd/js-repl/main.go`
3. `cmd/web-repl/main.go`
4. `engine/factory.go`
5. `engine/runtime.go`
6. `pkg/runtimeowner/runner.go`
7. `pkg/webrepl/service.go`
8. `pkg/webrepl/server.go`
9. `pkg/webrepl/rewrite.go`
10. `pkg/webrepl/types.go`
11. `pkg/repl/evaluators/javascript/evaluator.go`
12. `pkg/repl/evaluators/javascript/docs_resolver.go`
13. `pkg/docaccess/runtime/registrar.go`
14. `pkg/jsdoc/extract/extract.go`
15. `pkg/jsdoc/model/store.go`
16. `pkg/jsdoc/exportsq/exportsq.go`

### External API references

1. `goja` runtime project: <https://github.com/dop251/goja>
2. `goja_nodejs` module docs: <https://pkg.go.dev/github.com/dop251/goja_nodejs>
3. Go `net/http` package docs: <https://pkg.go.dev/net/http>
4. `go-sqlite3` project: <https://github.com/mattn/go-sqlite3>

## Bottom Line

The repository does not need "a bigger webrepl". It needs a durable persistent REPL product surface with one shared kernel, two first-class transports, SQLite-backed history, and REPL-authored docs. The staged `pkg/webrepl` code is the proof that the core ideas work. It should now be mined, split, and rebuilt around the boundaries we actually care about.

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
