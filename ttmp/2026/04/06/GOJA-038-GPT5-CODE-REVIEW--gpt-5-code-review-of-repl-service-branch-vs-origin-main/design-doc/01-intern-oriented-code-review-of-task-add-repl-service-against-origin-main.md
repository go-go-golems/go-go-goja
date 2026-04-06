---
Title: Intern-Oriented Code Review of task/add-repl-service against origin/main
Ticket: GOJA-038-GPT5-CODE-REVIEW
Status: complete
Topics:
    - goja
    - go
    - review
    - repl
    - architecture
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-repl/root.go
      Note: Canonical CLI
    - Path: cmd/goja-repl/tui.go
      Note: TUI profile defaults and replapi-backed Bobatea integration
    - Path: pkg/doc/04-repl-usage.md
      Note: User-facing REPL docs with persistence wording drift
    - Path: pkg/doc/13-plugin-developer-guide.md
      Note: Developer docs with stale adapter file reference
    - Path: pkg/repl/adapters/bobatea/replapi.go
      Note: Bobatea adapter for replapi-backed TUI execution
    - Path: pkg/repl/adapters/bobatea/runtime_assistance.go
      Note: Inspector-side runtime assistance reuse
    - Path: pkg/repl/evaluators/javascript/assistance.go
      Note: Shared completion/help extraction reused across evaluator and TUI paths
    - Path: pkg/replapi/app.go
      Note: App facade
    - Path: pkg/replapi/config.go
      Note: Profile and session override configuration model
    - Path: pkg/repldb/read.go
      Note: Deleted-session visibility and replay/export read paths
    - Path: pkg/repldb/store.go
      Note: SQLite bootstrap path and foreign-key configuration concern
    - Path: pkg/repldb/write.go
      Note: Soft-delete implementation and durable write flow
    - Path: pkg/replhttp/handler.go
      Note: HTTP lifecycle routes used for delete/restore reproduction
    - Path: pkg/replsession/policy.go
      Note: Declarative session policy model for eval
    - Path: pkg/replsession/rewrite.go
      Note: Instrumented REPL rewrite pipeline and static report generation
    - Path: pkg/replsession/service.go
      Note: Core live session kernel and main source of review findings
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-06T15:53:11.50818159-04:00
WhatFor: ""
WhenToUse: ""
---


# Intern-Oriented Code Review of task/add-repl-service against origin/main

## Executive Summary

This review covers the `task/add-repl-service` branch in `go-go-goja` against `origin/main`, with emphasis on the new REPL stack: `cmd/goja-repl`, `pkg/replapi`, `pkg/replsession`, `pkg/repldb`, `pkg/replhttp`, and the Bobatea/JavaScript assistance integration. I reviewed the branch directly from source and deliberately did not rely on prior review documents in `ttmp/`.

At a high level, the branch is trying to do four things at once:

1. Replace the older `cmd/repl` and `cmd/js-repl` entrypoints with one canonical `cmd/goja-repl` command surface.
2. Introduce a session-oriented API layer (`pkg/replapi`) above the raw runtime.
3. Add optional SQLite-backed persistence and replay (`pkg/repldb` + `pkg/replsession`).
4. Reuse the same runtime/session stack from the Bobatea TUI, the CLI, and a JSON HTTP server.

The direction is strong. The branch turns an evaluator-centric REPL into a session-centric system, which is the right architectural move if the project wants durable sessions, web clients, and richer introspection. The problem is that the branch also introduces a few correctness defects in persistence semantics, and it concentrates too much responsibility inside `pkg/replsession/service.go`, which makes the system harder for a new contributor to reason about than it needs to be.

The three most important code-level findings are:

1. Deleted sessions are not actually treated as deleted by the read and restore paths.
2. Persistent session IDs collide across separate processes because the default allocator is only in-memory.
3. SQLite foreign-key enforcement is disabled on pooled connections, so integrity assumptions are weaker than the code suggests.

Those are followed by documentation drift and a handful of maintainability issues around naming, duplication, and file organization.

## Problem Statement

An intern reading this branch needs two things:

1. A mental model of what the new REPL system is supposed to be.
2. A severity-ordered list of what is currently wrong, confusing, stale, or unnecessarily hard to maintain.

Without that orientation, the branch is difficult to approach because the user-facing surface looks simple while the implementation underneath is spread across several layers:

- CLI and TUI command wiring in `cmd/goja-repl`.
- App-level profile/config orchestration in `pkg/replapi`.
- Session lifecycle, runtime execution, static analysis, global diffing, JS doc extraction, and persistence marshaling in `pkg/replsession`.
- SQLite schema and read/write/export logic in `pkg/repldb`.
- HTTP route exposure in `pkg/replhttp`.
- TUI-facing runtime assistance and adapter code in `pkg/repl/adapters/bobatea`.
- Shared completion/help logic in `pkg/repl/evaluators/javascript/assistance.go`.

The branch delta is also large: `git diff --shortstat origin/main...HEAD` reports `88 files changed, 14149 insertions(+), 1167 deletions(-)`, and `48` non-`ttmp/` files changed. That means a reviewer needs a map, not just a list of files.

This document therefore has two goals:

1. Explain the system clearly enough that a new contributor can trace the main runtime flows.
2. Identify the highest-leverage fixes and cleanup work, with evidence anchored to concrete files and lines.

## Scope And Method

This review is evidence-based. I inspected the changed source directly, ran the automated test suite, and performed a few targeted runtime reproductions.

### Branch scope

- Review base: `origin/main`
- Review head: branch `task/add-repl-service`
- Non-`ttmp/` files reviewed in depth:
  - `cmd/goja-repl/*`
  - `pkg/replapi/*`
  - `pkg/repldb/*`
  - `pkg/replhttp/*`
  - `pkg/replsession/*`
  - `pkg/repl/adapters/bobatea/*`
  - `pkg/repl/evaluators/javascript/*`
  - selected docs updated in `README.md` and `pkg/doc/*`

### Validation performed

- `go test ./...`
- Reproduced repeated `goja-repl create` calls against one SQLite file
- Reproduced delete/restore behavior through the JSON server
- Reproduced SQLite `PRAGMA foreign_keys` state across pooled connections

### Important constraint

I did not use older review documents in `ttmp/` as source material. Any conclusions here come from direct inspection of the current branch.

## Current-State Architecture

### What the system is trying to be

The new design turns the REPL from "one evaluator attached to one terminal" into "a reusable session service with multiple frontends."

That is the right abstraction. A session service can support:

- a terminal UI,
- a scriptable CLI,
- an HTTP API,
- durable state and replay,
- richer diagnostics and binding metadata.

### High-level diagram

```text
                 +----------------------+
                 |   cmd/goja-repl      |
                 |  create/eval/serve   |
                 |       tui/help       |
                 +----------+-----------+
                            |
                            v
                 +----------------------+
                 |    pkg/replapi       |
                 | profiles + app API   |
                 +----------+-----------+
                            |
             +--------------+--------------+
             |                             |
             v                             v
   +----------------------+      +----------------------+
   |  pkg/replsession     |      |    pkg/replhttp      |
   | live session kernel  |      | JSON route exposure  |
   +----------+-----------+      +----------------------+
              |
      +-------+-----------------------------+
      |                                     |
      v                                     v
+-------------+                    +-----------------+
| engine.*    |                    |   pkg/repldb    |
| goja runtime|                    | sqlite storage  |
+-------------+                    +-----------------+
      ^
      |
      +---------------------------------------------+
                                                    |
                                        +---------------------------+
                                        | Bobatea adapter + shared  |
                                        | Assistance (completion,   |
                                        | help bar, help drawer)    |
                                        +---------------------------+
```

### Layer-by-layer explanation

#### 1. `cmd/goja-repl`: the command surface

`cmd/goja-repl/root.go` is the top-level command assembly point. It wires shared flags, shared help loading, and subcommands for:

- `sessions`
- `create`
- `eval`
- `snapshot`
- `history`
- `bindings`
- `docs`
- `export`
- `restore`
- `serve`
- `tui`

This is all in one file right now: `cmd/goja-repl/root.go:41-567`.

The important design choice is that the commands do not talk directly to a raw goja runtime. They all go through `replapi.App`, built by `commandSupport.newAppWithOptions` in `cmd/goja-repl/root.go:116-157`.

#### 2. `pkg/replapi`: the application facade

`pkg/replapi` is a policy/configuration wrapper around the session service.

Key concepts:

- Profiles: `raw`, `interactive`, `persistent` in `pkg/replapi/config.go:15-19`
- App config: `Config` in `pkg/replapi/config.go:22-27`
- Per-session override input: `SessionOptions` in `pkg/replapi/config.go:30-35`

This layer is what lets the same underlying service behave differently in different entrypoints:

- `raw`: minimal evaluation, little or no observation
- `interactive`: REPL-friendly in-memory behavior
- `persistent`: interactive behavior plus durable store-backed replay

The app methods are straightforward wrappers around the service plus store:

- create/snapshot/evaluate/bindings: mostly service calls
- history/export/docs/list: store-backed queries
- auto-restore: `pkg/replapi/app.go:187-199`

#### 3. `pkg/replsession`: the real kernel

This is the core of the branch.

`pkg/replsession/service.go` is where nearly all session behavior lives:

- create session
- evaluate source
- static analysis
- rewrite source for REPL semantics
- snapshot globals
- track bindings
- capture console output
- extract JSDoc-like docs
- persist evaluation records
- replay session history on restore

For a new reader, the most important mental model is:

```text
session = runtime + policy + cell history + tracked bindings + optional persistence
```

The service state is explicit:

- service-wide state in `pkg/replsession/service.go:39-47`
- per-session state in `pkg/replsession/service.go:49-62`
- per-cell state in `pkg/replsession/service.go:64-67`
- per-binding state in `pkg/replsession/service.go:69-79`

#### 4. `pkg/repldb`: durable storage

The SQLite schema is defined in `pkg/repldb/schema.go:3-88`. The tables tell you what the system considers durable:

- `sessions`
- `evaluations`
- `console_events`
- `bindings`
- `binding_versions`
- `binding_docs`

The intent is sensible:

- session metadata and lifecycle in `sessions`
- one evaluated cell per `evaluations` row
- emitted console output split into child rows
- logical binding identity in `bindings`
- time-versioned binding changes in `binding_versions`
- docs extracted from REPL-authored source in `binding_docs`

#### 5. `pkg/replhttp`: JSON server surface

`pkg/replhttp/handler.go:20-151` exposes the app over HTTP with routes such as:

- `GET /api/sessions`
- `POST /api/sessions`
- `GET /api/sessions/{id}`
- `DELETE /api/sessions/{id}`
- `POST /api/sessions/{id}/evaluate`
- `POST /api/sessions/{id}/restore`
- `GET /api/sessions/{id}/history`
- `GET /api/sessions/{id}/bindings`
- `GET /api/sessions/{id}/docs`
- `GET /api/sessions/{id}/export`

This gives the branch a machine-oriented surface, which is important if the longer-term plan includes a web UI or agent automation.

#### 6. Bobatea integration and shared assistance

The TUI path now uses:

- `cmd/goja-repl/tui.go`
- `pkg/repl/adapters/bobatea/replapi.go`
- `pkg/repl/adapters/bobatea/runtime_assistance.go`
- `pkg/repl/evaluators/javascript/assistance.go`

This is one of the better changes in the branch. The assistance logic was extracted out of the classic evaluator into a reusable component. That reduces duplication between:

- the original evaluator-backed REPL path
- the new session-backed Bobatea TUI path
- the smalltalk inspector's runtime assistance path

The extraction is visible in `pkg/repl/evaluators/javascript/evaluator.go`, where `CompleteInput`, `GetHelpBar`, and `GetHelpDrawer` now delegate into `Assistance`.

## Runtime Flows

### Flow 1: Create a persistent session

```text
CLI/HTTP/TUI
  -> replapi.App.CreateSession(...)
    -> replsession.Service.CreateSessionWithOptions(...)
      -> engine.Factory.NewRuntime(...)
      -> install console/doc helpers if enabled
      -> store.CreateSession(...) if persistence is enabled
      -> keep live session in memory
```

The implementation path is:

- `cmd/goja-repl/root.go:229-241`
- `pkg/replapi/app.go:63-72`
- `pkg/replsession/service.go:143-208`
- `pkg/repldb/write.go:14-57`

### Flow 2: Evaluate one cell

```text
source
  -> optional jsparse analysis
  -> optional rewrite for instrumented mode
  -> execute in goja runtime
  -> snapshot globals and bindings
  -> build CellReport
  -> optionally persist evaluation + binding versions + docs
```

The main branch point is in `pkg/replsession/service.go:210-272`.

Instrumented mode goes through:

- `evaluateInstrumented` in `pkg/replsession/service.go:329-422`

Raw mode goes through:

- `evaluateRaw` in `pkg/replsession/service.go:424-512`

### Flow 3: Restore a session

```text
app.Snapshot(id)
  -> live session exists? use it
  -> otherwise auto-restore if configured
      -> load session metadata
      -> load replay source history
      -> create temporary replay runtime
      -> re-evaluate each stored cell
      -> move reconstructed runtime into live map
```

The important code is:

- auto-restore gate in `pkg/replapi/app.go:187-199`
- persistent restore in `pkg/replapi/app.go:85-99`
- replay implementation in `pkg/replsession/service.go:561-624`

This is the heart of the persistence design. It is also where one of the biggest correctness issues shows up, because "deleted" sessions still pass through this path.

## API Reference For New Contributors

### Profiles

Profile behavior is encoded in:

- `pkg/replapi/config.go:45-84`
- `pkg/replsession/policy.go:62-107`

In prose:

- `raw`: direct execution, minimal session interpretation
- `interactive`: instrumented execution plus static/runtime observation
- `persistent`: interactive plus durable writes and replay metadata

### Session policy structure

The policy tree in `pkg/replsession/policy.go:17-47` is the main conceptual contract:

- `EvalPolicy`: how code is executed
- `ObservePolicy`: what extra metadata is collected
- `PersistPolicy`: what is written durably

That policy object is what connects configuration to actual behavior. If you want to understand why a session did or did not:

- capture console output,
- emit bindings,
- persist docs,
- support top-level await,

you look here first.

### Durable data model

`pkg/repldb/types.go:24-40` is the durable row model for evaluated cells. One naming wrinkle matters here:

- `EvaluationRecord.AnalysisJSON` sounds like raw analysis output
- but `persistCell` actually stores `cell.Static` into it at `pkg/replsession/service.go:639-641`

That is not a correctness defect, but it is a naming mismatch that makes the persistence layer harder to understand than necessary.

## Findings

### 1. High: deleted sessions are still listed and can be auto-restored

Problem:

Deletion is implemented as a soft delete in SQLite, but the read paths do not treat soft-deleted sessions as deleted. In practice, that means a deleted session can still appear in listings and can be resurrected by the app's auto-restore path.

Where to look:

- `pkg/repldb/write.go:59-89`
- `pkg/repldb/read.go:25-30`
- `pkg/repldb/read.go:66-68`
- `pkg/replapi/app.go:85-99`
- `pkg/replapi/app.go:187-199`

Example:

```go
// write.go
UPDATE sessions
SET deleted_at = ?, updated_at = ?
WHERE session_id = ?

// read.go
SELECT session_id, created_at, updated_at, deleted_at, engine_kind, metadata_json
FROM sessions
ORDER BY created_at ASC, session_id ASC
```

Observed behavior:

- I created a session through the JSON server.
- I evaluated one cell.
- I deleted the session with `DELETE /api/sessions/{id}`.
- A subsequent `GET /api/sessions/{id}` returned HTTP 200 and auto-restored the session.
- `GET /api/sessions` still listed the deleted session, now with `DeletedAt` populated.

Why it matters:

- User-facing semantics are wrong. "Delete" does not mean "unavailable".
- The app can silently resurrect state the user believed was gone.
- Any future UI will have to special-case "deleted but restorable" records unless the backend semantics are fixed.
- Tests currently validate that `deleted_at` is written, but not that deleted sessions disappear from all read surfaces.

Cleanup sketch:

```text
Store.ListSessions:
  SELECT ... FROM sessions
  WHERE deleted_at IS NULL

Store.LoadSession:
  SELECT ... FROM sessions
  WHERE session_id = ? AND deleted_at IS NULL

Store.LoadEvaluations / ExportSession / LoadReplaySource:
  reject deleted sessions early via LoadSession

App.Restore:
  if LoadSession returns ErrSessionNotFound for a deleted row,
  surface that as a real not-found and do not replay
```

Tests to add:

- delete -> list should omit session
- delete -> snapshot should return not found
- delete -> restore should return not found
- delete -> export/history/docs should return not found

### 2. High: default persistent session IDs collide across processes

Problem:

The default session ID allocator is an in-memory counter. Each new process starts from zero, so the first persistent session in each fresh process tries to use `session-1`. Because `cmd/goja-repl` creates a fresh app/service for every command invocation, repeated `create` commands against one SQLite database collide.

Where to look:

- `pkg/replsession/service.go:44`
- `pkg/replsession/service.go:153-158`
- `cmd/goja-repl/root.go:109-157`

Example:

```go
if id == "" {
    id = fmt.Sprintf("session-%d", atomic.AddUint64(&s.nextID, 1))
}
```

Observed behavior:

Running:

```bash
go run ./cmd/goja-repl --db-path "$tmpdb" create
go run ./cmd/goja-repl --db-path "$tmpdb" create
```

produced:

```text
Error: persist session: create session: UNIQUE constraint failed: sessions.session_id
exit status 1
```

Why it matters:

- The canonical persistent CLI path is broken for normal repeated usage.
- The bug is branch-specific and user-visible.
- The failure propagates as a storage error instead of a clean session creation failure.

Cleanup sketch:

Preferred option:

```text
use non-sequential durable IDs
  session-<uuid-or-ulid>
```

If sequential IDs are important:

```text
on service startup or before allocation:
  read highest existing durable id from store
  advance nextID

on create:
  loop until candidate id does not already exist
```

Tests to add:

- create session in app instance 1
- create session in app instance 2 with same store
- assert different durable session IDs

### 3. High: SQLite foreign-key enforcement is disabled on pooled connections

Problem:

The store enables `PRAGMA foreign_keys = ON` only inside the bootstrap transaction. In SQLite, that setting is connection-local. Because `database/sql` uses a connection pool, later queries and transactions may run on different connections with foreign keys off.

Where to look:

- `pkg/repldb/store.go:32`
- `pkg/repldb/store.go:70-88`

Example:

```go
db, err := sql.Open("sqlite3", path)
...
if _, err := tx.ExecContext(ctx, `PRAGMA foreign_keys = ON;`); err != nil {
    ...
}
```

Observed behavior:

I opened a store, expanded the pool, acquired fresh `database/sql` connections, and queried `PRAGMA foreign_keys;`. Those additional pooled connections reported `fk=0`.

Why it matters:

- The schema declares foreign keys in `pkg/repldb/schema.go:17-80`, but the runtime does not reliably enforce them.
- That weakens integrity guarantees for:
  - evaluations -> sessions
  - console events -> evaluations
  - binding rows -> sessions/evaluations
- The code reads as if foreign keys are guaranteed, but the operational behavior does not match.

Cleanup sketch:

```text
open sqlite with DSN that enables foreign keys per connection
  e.g. sqlite3 DSN with _foreign_keys=on

optionally also set:
  _busy_timeout
  WAL mode if concurrent access is expected

keep the bootstrap PRAGMA if desired,
but do not rely on it as the only enforcement mechanism
```

Tests to add:

- verify `PRAGMA foreign_keys` on multiple acquired connections
- attempt child-row insert without parent and assert failure

### 4. Medium: documentation drift already points readers at stale code and incorrect semantics

Problem:

The code moved toward a unified `goja-repl` and reusable assistance layer, but some docs still point to stale paths or overstate persistence behavior. This matters because the branch is explicitly trying to make the REPL more teachable and more discoverable.

Where to look:

- `pkg/doc/13-plugin-developer-guide.md:421-426`
- `pkg/doc/04-repl-usage.md:211-222`

Examples:

```md
- `pkg/repl/adapters/bobatea/javascript.go`
```

That file does not exist in the branch; the adapter now lives in:

- `pkg/repl/adapters/bobatea/replapi.go`
- `pkg/repl/adapters/bobatea/runtime_assistance.go`

And:

```md
Variables and functions persist across REPL sessions:
```

The example beneath that line only demonstrates persistence within a single in-memory interactive session. Cross-session persistence now depends on the persistent profile and session restore path, not on the default TUI profile.

Why it matters:

- New contributors will chase stale file names.
- The docs blur the distinction between "state persists within a live session" and "state survives process boundaries."
- This makes the branch harder to teach than the code structure alone requires.

Cleanup sketch:

```text
update docs to distinguish:
  live-session persistence
  durable persistence
  restore semantics

replace stale adapter file references

consider a docs check that greps for removed entrypoints and removed file paths
```

## Maintainability And Cleanup Opportunities

These are not all correctness bugs, but they are the main sources of confusion and future drift.

### A. `pkg/replsession/service.go` is carrying too many responsibilities

`pkg/replsession/service.go` is `1633` lines and currently mixes:

- lifecycle management,
- policy handling,
- execution,
- top-level-await support,
- AST/CST reporting,
- global diffing,
- runtime inspection,
- JSDoc extraction,
- persistence marshaling.

That makes local reasoning expensive. For an intern, this file feels like "the whole system," which is a sign that the abstraction boundaries are too implicit.

Suggested split:

```text
pkg/replsession/
  service.go            // public orchestration only
  service_create.go     // session creation and restore
  service_eval.go       // evaluateInstrumented/evaluateRaw
  service_runtime.go    // snapshots, global diffs, runtime inspection
  service_persist.go    // persistence marshaling
  service_docs.go       // doc sentinel + extraction
  rewrite.go            // already separated and should stay separated
  policy.go             // already separated and should stay separated
```

### B. `AnalysisJSON` is a confusing durable field name

Evidence:

- field name in `pkg/repldb/types.go:35`
- stored value in `pkg/replsession/service.go:639-641`

The field sounds like raw analyzer output, but the branch stores the rendered `StaticReport`, which is a higher-level summary view. Renaming it to something like `StaticReportJSON` would reduce confusion immediately.

### C. top-level-await wrapping logic is duplicated

Evidence:

- `pkg/repl/evaluators/javascript/evaluator.go:381-387`
- `pkg/replsession/service.go:1021-1027`

The helper bodies are effectively the same. This is small, but it is exactly the sort of duplication that drifts later when one path gets fixed and the other does not.

### D. command boilerplate in `cmd/goja-repl/root.go` is repetitive

The command file is readable, but it repeats the same pattern many times:

- decode flags
- create app/store
- defer store close
- call one app method
- write JSON

That repetition makes it easy for behavior to diverge subtly between subcommands. A thin helper for "persistent JSON subcommand" would cut a lot of noise.

Pseudocode:

```go
func runPersistentJSONCommand[T any](
    ctx context.Context,
    support commandSupport,
    fn func(*replapi.App) (T, error),
) error {
    app, store, err := support.newApp()
    if err != nil { return err }
    defer store.Close()
    payload, err := fn(app)
    if err != nil { return err }
    return writeJSON(support.out, payload)
}
```

### E. repository residue still includes `.orig` files

Observed files:

- `engine/config.go.orig`
- `engine/runtime.go.orig`
- `modules/exports.go.orig`

These are not part of the branch delta, but they are repo-level confusing artifacts that look like abandoned manual backups. They raise the cognitive cost of grepping the codebase and should either be deleted or moved into an explicit archive/scratch area outside the normal source tree.

## Why The Good Parts Still Matter

Despite the issues above, the branch contains several genuinely good architectural moves:

- `pkg/replapi` creates a stable seam between frontends and session execution.
- `pkg/repl/evaluators/javascript/assistance.go` is a good extraction and makes the completion/help stack reusable.
- The session policy tree in `pkg/replsession/policy.go` is a strong idea, because it expresses behavior declaratively instead of scattering booleans everywhere.
- The SQLite schema is directionally solid and captures enough structure to support replay, export, and future tooling.

In other words: this branch is not a dead end. It is a good design with a few sharp correctness bugs and an implementation that needs one more cleanup pass.

## Prioritized Implementation Plan

### Phase 1: fix observable correctness

1. Fix deleted-session read semantics in `pkg/repldb/read.go` and `pkg/replapi/app.go`.
2. Replace or harden session ID allocation for persistent mode.
3. Enable SQLite foreign keys at connection-open time, not just in bootstrap.
4. Add regression tests for all three defects.

### Phase 2: fix documentation drift

1. Update stale file references in `pkg/doc/13-plugin-developer-guide.md`.
2. Clarify the persistence wording in `pkg/doc/04-repl-usage.md`.
3. Ensure `README.md` and help pages describe the profile distinction consistently.

### Phase 3: reduce maintenance risk

1. Split `pkg/replsession/service.go` by responsibility.
2. Rename confusing durable fields such as `AnalysisJSON`.
3. Remove or centralize duplicated helpers.
4. Delete `.orig` repository residue if nothing still depends on it.

## Test Strategy

The branch already has a useful test base:

- `cmd/goja-repl/root_test.go`
- `pkg/replapi/app_test.go`
- `pkg/repldb/store_test.go`
- `pkg/replhttp/handler_test.go`
- `pkg/replsession/service_persistence_test.go`
- `pkg/replsession/service_policy_test.go`

What is missing are the tests that match the defects above.

Recommended new tests:

1. Delete semantics:
   - create session
   - evaluate
   - delete
   - assert list/snapshot/restore/export/history all treat it as gone

2. Cross-process persistent creation:
   - app1 create with store
   - app2 create with same store
   - assert unique IDs

3. Foreign keys:
   - verify `PRAGMA foreign_keys = 1` across multiple pooled connections
   - verify orphan insert fails

4. Docs drift:
   - a very small doc-link/path linter or at least a grep-based test in CI

## Alternatives Considered

### Keep the evaluator-centric design and bolt persistence on the side

Rejected because it would make the TUI, HTTP, and persistence layers each invent their own partial session concepts. The new `replapi` + `replsession` split is better.

### Keep soft delete but allow restore as an intentional feature

Possible, but then the API and docs must say that clearly. Right now the behavior is framed as deletion while the implementation acts like archival. That mismatch is the real bug.

### Keep sequential IDs and rely on one long-lived process

Rejected because the current CLI already constructs fresh app/service instances per invocation, and the HTTP path can also restart independently. The architecture now requires durable ID allocation, not just process-local allocation.

## Open Questions

1. Should deleted sessions be permanently inaccessible, or should there be an explicit archive/undelete feature? The current implementation implicitly chooses undelete-by-read, which is the least clear option.
2. Should persistent session IDs be human-readable sequential IDs, or would UUID/ULID-style IDs be acceptable?
3. Does the project want the docs module available only in `goja-repl tui`, or in all `goja-repl` runtimes including CLI `eval` and HTTP sessions?

## References

- Branch command assembly: `cmd/goja-repl/root.go:41-567`
- TUI entrypoint: `cmd/goja-repl/tui.go:41-145`
- App facade: `pkg/replapi/app.go:15-218`
- Profile config: `pkg/replapi/config.go:12-183`
- Session policy: `pkg/replsession/policy.go:9-173`
- Session kernel: `pkg/replsession/service.go:38-1633`
- Rewrite pipeline: `pkg/replsession/rewrite.go:13-427`
- Durable schema: `pkg/repldb/schema.go:3-88`
- Durable write paths: `pkg/repldb/write.go:13-335`
- Durable read paths: `pkg/repldb/read.go:13-359`
- HTTP routes: `pkg/replhttp/handler.go:14-175`
- Shared assistance: `pkg/repl/evaluators/javascript/assistance.go:17-676`
- Bobatea replapi adapter: `pkg/repl/adapters/bobatea/replapi.go:18-184`
- Runtime assistance for inspector reuse: `pkg/repl/adapters/bobatea/runtime_assistance.go:16-114`

<!-- Describe the proposed solution in detail -->

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
