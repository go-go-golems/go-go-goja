---
Title: Persistence correctness analysis, design, and implementation guide
Ticket: GOJA-040-PERSISTENCE-CORRECTNESS
Status: active
Topics:
    - goja
    - go
    - sqlite
    - repl
    - architecture
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Intern-oriented guide for fixing deleted-session semantics, durable session IDs, and SQLite connection integrity in the REPL stack."
LastUpdated: 2026-04-07T10:00:00-04:00
WhatFor: "Provide a detailed analysis and implementation guide for the persistence correctness PR."
WhenToUse: "Use when implementing, reviewing, or testing GOJA-040."
---

# Persistence correctness analysis, design, and implementation guide

## Executive summary

This ticket is the first PR because it fixes behavior that is semantically wrong today. It is not primarily a refactor. It is primarily a contract fix.

There are three defects:

1. Deleted sessions are soft-deleted in SQLite, but they still show up in normal read paths.
2. Default persistent session IDs are allocated in-memory, so two separate processes can pick the same ID.
3. SQLite foreign key enforcement is enabled during bootstrap, but not reliably across every pooled connection used later.

If you are a new intern, the most important thing to understand is that these are not "code quality opinions." These are cases where the system can tell the user the wrong thing or enforce the wrong integrity guarantees.

## What system are we changing?

The REPL stack is layered like this:

```text
CLI / HTTP / tests
        |
        v
pkg/replapi        application facade
        |
        v
pkg/replsession    live in-memory session kernel
        |
        +--> pkg/repldb     durable SQLite persistence
```

The persistence PR mainly touches the `pkg/repldb` and `pkg/replapi` boundary, with a small amount of `pkg/replsession` context because session creation and deletion start there.

Important files:

- [pkg/repldb/read.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/read.go)
- [pkg/repldb/write.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/write.go)
- [pkg/repldb/store.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/store.go)
- [pkg/replapi/app.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go)
- [pkg/replsession/service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go)

Key entry points to understand:

- `(*Store).ListSessions` in [read.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/read.go#L17)
- `(*Store).LoadSession` in [read.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/read.go#L51)
- `(*Store).DeleteSession` in [write.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/write.go)
- `Open` and `bootstrap` in [store.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/store.go#L21) and [store.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/store.go#L62)
- `(*App).Restore`, `(*App).DeleteSession`, and `(*App).ListSessions` in [app.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go#L85), [app.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go#L101), and [app.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go#L118)

## Problem 1: deleted sessions are still visible

### What happens today

Today, deleting a durable session sets `deleted_at`. That is a soft delete.

That part is fine.

The problem is that the normal read paths do not filter on `deleted_at`. So the same session can still be:

- listed
- loaded
- restored
- exported
- used for history lookups

Conceptually, the current behavior is:

```text
DeleteSession(sessionID):
    UPDATE sessions
    SET deleted_at = now()

ListSessions():
    SELECT * FROM sessions

LoadSession(sessionID):
    SELECT * FROM sessions WHERE session_id = ?
```

That is internally inconsistent because the database remembers the delete, but the public read contract ignores it.

### Why this matters

If a user deletes a session, they reasonably expect that session to stop appearing in normal product behavior. If the system still lists or restores it, the delete action becomes misleading.

That creates two bad outcomes:

- user trust issue: "delete" did not mean what users think it means
- API correctness issue: higher layers cannot trust storage semantics

### Correct design

Normal read paths should behave as if soft-deleted rows no longer exist.

That means:

- `ListSessions` returns only rows where `deleted_at IS NULL`
- `LoadSession` returns `ErrSessionNotFound` for rows where `deleted_at IS NOT NULL`
- all higher-level flows built on top of `LoadSession` inherit that behavior

The simple rule is:

```text
normal application reads never see deleted sessions
```

If you later want an admin or debugging path that can see deleted sessions, that should be an explicit separate API, not the default.

### Implementation sketch

Pseudocode:

```text
func ListSessions(ctx):
    SELECT ... FROM sessions
    WHERE deleted_at IS NULL
    ORDER BY created_at ASC, session_id ASC

func LoadSession(ctx, sessionID):
    SELECT ... FROM sessions
    WHERE session_id = ? AND deleted_at IS NULL
    if no row:
        return ErrSessionNotFound
```

Higher layers then become correct automatically:

```text
App.Restore(sessionID):
    record = store.LoadSession(sessionID)
    if record not found:
        return not found
    ...
```

### Tests to add

- delete a durable session, then verify `ListSessions` no longer returns it
- delete a durable session, then verify `LoadSession` returns not found
- delete a durable session, then verify `Restore` returns not found
- delete a durable session, then verify `Export` and `History` also fail consistently if they depend on the hidden session contract

## Problem 2: durable session ID generation is not safe across processes

### What happens today

The current system uses a process-local default session ID allocator. That means process A and process B can both start from the same initial state and both decide that `"session-1"` is the next ID.

This was acceptable only if sessions were purely in-memory and process-scoped. Once SQLite persistence exists, it is no longer acceptable.

Why?

Because two separate invocations of the CLI or server against the same SQLite file are now part of the supported mental model.

### Why this matters

This is a classic "state moved from process-local to durable, but identifier allocation did not move with it" bug.

The symptom is a unique-constraint failure that the user did not do anything wrong to trigger.

### Best design choice

Do not allocate durable IDs using a shared incrementing counter unless you truly need human-readable sequential identifiers.

For this system, the cleanest choice is:

- generate opaque collision-resistant IDs such as ULID or UUID for durable sessions

Why this is elegant:

- no DB locking dance
- no `SELECT MAX(...)`
- no process coordination
- easier to reason about in tests and production

### Recommended invariant

```text
session IDs must be globally unique enough that separate processes using the same DB never collide in normal operation
```

### Implementation shape

If the caller explicitly sets `opts.ID`, honor it.

If the caller does not set `opts.ID`, generate one with an opaque durable ID generator.

Pseudocode:

```text
func resolveSessionID(opts):
    if opts.ID != "":
        return opts.ID
    return newOpaqueSessionID()
```

Examples:

- `session_01HV...`
- `repl_01HV...`
- plain UUID strings if you do not care about lexical ordering

I would avoid bare UUIDs only if you want log readability. ULIDs are usually nicer for humans.

### Where to implement

Start by tracing session creation from:

- [pkg/replsession/service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go#L143)
- [pkg/replsession/policy.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go#L50)
- [pkg/replapi/config.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go#L30)

### Tests to add

- create two sessions without explicit IDs in one process and verify distinct IDs
- create a session in one process, then in a separate process against the same DB create another, and verify no collision
- preserve backward behavior for explicit IDs supplied by tests or callers

## Problem 3: SQLite foreign key enforcement is connection-local

### What happens today

In [store.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/store.go#L76), bootstrap enables `PRAGMA foreign_keys = ON` inside a transaction.

That sounds right at first glance, but SQLite PRAGMAs like this are connection-local. The problem is that `*sql.DB` is a pool, not a single forever-connection object.

So the system currently has this risk:

```text
Open():
    create DB pool
    bootstrap on one connection
    enable foreign_keys on that connection

later:
    pooled operation may use a different connection
    that connection may not have foreign_keys enabled
```

### Why this matters

This is subtle because tests can still pass. The system may behave correctly in small runs and then silently stop enforcing integrity guarantees in other runs depending on connection reuse.

This is the worst kind of storage bug:

- easy to miss
- not obvious from happy-path behavior
- corrodes trust in constraints

### Correct design

Make integrity-related SQLite settings part of connection-open configuration, not just schema bootstrap.

In practice, this usually means:

- configure the sqlite DSN so foreign keys are enabled for each connection
- also consider busy timeout and WAL at the same layer

Conceptual model:

```text
every new SQLite connection should start in the desired integrity mode
```

### Recommended policy

At minimum:

- foreign keys on

Likely also:

- busy timeout set
- WAL enabled, if it fits the project's concurrency story

### Implementation sketch

Pseudocode:

```text
func Open(path):
    dsn = buildSQLiteDSN(
        path,
        foreignKeys=true,
        busyTimeout=5000,
        wal=true,
    )
    db = sql.Open("sqlite3", dsn)
    bootstrapSchema(db)
```

Note the separation:

- open-time connection settings
- bootstrap-time schema creation

These are related, but they are not the same concern.

## Proposed PR structure

This PR should remain intentionally narrow.

### In scope

- fix default read filtering for deleted sessions
- fix default durable ID allocation
- fix SQLite connection integrity settings
- add focused regression tests

### Out of scope

- large refactors
- timeout and interruption work
- package renames
- CORS/auth/rate-limit concerns

That keeps the review about one thing:

```text
does persistence behave correctly and consistently?
```

## Suggested implementation order

1. Fix deleted-session read semantics.
2. Add tests proving restore/list behavior after delete.
3. Swap in opaque default durable IDs.
4. Add multi-process regression test.
5. Move SQLite integrity settings into connection-open configuration.
6. Add one targeted constraint enforcement test or probe.

This ordering is good because the first two fixes are the easiest to explain and review, and the SQLite connection fix is the most subtle.

## Review checklist

Use this during PR review:

- Does `ListSessions` hide deleted rows?
- Does `LoadSession` hide deleted rows?
- Does `Restore` now fail for deleted sessions?
- Are explicit caller-provided IDs still honored?
- Are default IDs collision-resistant across processes?
- Are SQLite foreign keys enabled on every connection, not just bootstrap connection?
- Did tests cover the negative cases, not just the happy cases?

## Manual testing guide

### Test 1: delete semantics

```bash
tmpdb="$(mktemp /tmp/goja-repl-XXXXXX.sqlite)"
go run ./cmd/goja-repl --db-path "$tmpdb" create
```

Then:

1. create a session
2. evaluate one cell
3. delete the session
4. list sessions
5. attempt restore/export/history for the deleted ID

Expected behavior after the fix:

- not shown in list
- restore fails with not found
- export/history behave consistently with the hidden session contract

### Test 2: cross-process IDs

```bash
tmpdb="$(mktemp /tmp/goja-repl-XXXXXX.sqlite)"
go run ./cmd/goja-repl --db-path "$tmpdb" create
go run ./cmd/goja-repl --db-path "$tmpdb" create
```

Expected behavior after the fix:

- both succeed
- IDs differ
- no unique-constraint failure

### Test 3: SQLite integrity config

Add a targeted test or probe that checks the effective `PRAGMA foreign_keys` setting on more than one connection. The exact test harness can vary, but the core idea is to prove the setting is not merely bootstrap-local.

## Final advice for the intern

Do not think of this ticket as a "database cleanup" ticket. Think of it as a "make storage mean what the API says it means" ticket.

That framing will help you make the right small decisions:

- prefer explicit semantics over cleverness
- prefer stable invariants over incidental behavior
- prefer durable ID safety over pretty sequential names
- prefer connection-open integrity configuration over bootstrap-only assumptions
