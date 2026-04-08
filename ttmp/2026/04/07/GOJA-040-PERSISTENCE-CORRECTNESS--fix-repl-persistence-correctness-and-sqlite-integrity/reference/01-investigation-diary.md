---
Title: Investigation diary
Ticket: GOJA-040-PERSISTENCE-CORRECTNESS
Status: active
Topics:
    - goja
    - go
    - sqlite
    - repl
    - analysis
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological notes for the persistence correctness design guide."
LastUpdated: 2026-04-07T10:00:00-04:00
WhatFor: "Record why the persistence-correctness work was split into its own PR and how the design was chosen."
WhenToUse: "Use when retracing the design decisions for GOJA-040."
---

# Investigation diary

## Why this ticket exists

The review and review-review both pointed to a simple conclusion: persistence correctness is the first PR because it fixes behavior that is wrong today, not just code that is messy today.

The three concrete defects are:

- soft-deleted sessions are still returned by normal read paths
- process-local default IDs collide across separate processes
- SQLite foreign key enablement is done during bootstrap but not guaranteed on every later pooled connection

## Files to understand first

- [pkg/repldb/read.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/read.go)
- [pkg/repldb/write.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/write.go)
- [pkg/repldb/store.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/store.go)
- [pkg/replapi/app.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go)
- [pkg/replsession/service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go)

## Main design stance

The persistence layer should behave as if it were a small database-backed API, not just an internal cache. That means reads must have a clear contract, deletes must have a clear contract, and identifiers must remain safe across multiple processes.

## 2026-04-08 implementation session

Implementation is being done in three commits, not one large patch.

Planned commit order:

1. deleted-session semantics and tests
2. collision-resistant default session IDs and tests
3. connection-open SQLite integrity configuration and tests

Reasoning:

- the first slice changes the visible API contract
- the second slice changes identifier safety across processes
- the third slice changes storage integrity setup

That keeps review focused and makes regressions easier to localize.

### Commit 1: deleted-session contract

Implemented the first behavior slice by changing the durable read paths to treat soft-deleted sessions as absent from normal application reads.

Changes made:

- `ListSessions` now filters to rows where `deleted_at IS NULL`
- `LoadSession` now only loads rows where `deleted_at IS NULL`
- `LoadEvaluations` now first confirms the session is still visible through `LoadSession`
- added store-level regression coverage for list/load/export/replay behavior after delete
- added app-level regression coverage for restore/history/export behavior after delete

Validation:

```bash
go test ./pkg/repldb ./pkg/replapi ./pkg/replsession
```

Result:

- all focused persistence packages passed
- the first slice is ready to commit independently

### Commit 2: collision-resistant default IDs

Implemented the second behavior slice by removing the in-memory `session-%d` counter from `replsession.Service` and replacing the default ID path with generated opaque IDs.

Design decision:

- explicit caller-provided IDs still win
- default IDs are now generated as `session-<uuid>`

Why this design was chosen:

- it avoids cross-process coordination entirely
- it preserves log readability better than a bare UUID
- it removes the need for the old `nextID` and `noteSessionID` machinery

Validation:

```bash
go test ./pkg/repldb ./pkg/replapi ./pkg/replsession
```

Additional regression coverage:

- two independent services sharing the same store now generate different IDs
- explicit `SessionOptions.ID` is still honored unchanged
