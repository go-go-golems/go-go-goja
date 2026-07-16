---
Title: Using replapi for Long-Lived REPL Sessions
Slug: replapi-guide
Short: Embed stateful Goja sessions, add SQLite replay, or expose the protobuf-JSON API
Topics:
- repl
- replapi
- replsession
- persistence
- http
Commands:
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

`replapi.App` is the high-level Go API for evaluating multiple JavaScript cells against a session. While an app is running, each session owns one Goja runtime, so declarations and mutations remain available to later evaluations. With the persistent profile, the app also records cells in SQLite and can reconstruct a missing runtime by replaying them.

These are different guarantees:

- **Live continuity:** repeated calls on the same app and session ID use the same runtime.
- **Durable reconstruction:** a new app can rebuild a runtime from persisted source cells.

Persistence does not serialize the Goja heap. A restored session is a new runtime produced by replay, not the original VM resumed from a snapshot.

## Architecture and Ownership

`replapi` is a facade over the packages that implement the actual work. Understanding these boundaries prevents callers from bypassing session ownership or mistaking database records for live state.

```text
caller / TUI / HTTP
        |
        v
   replapi.App          application facade and optional auto-restore
        |
        +----> replsession.Service   live sessions and evaluation pipeline
        |             |
        |             +----> engine.Runtime   one owned Goja VM per session
        |
        +----> repldb.Store          SQLite history, bindings, docs, exports

   replhttp.NewHandler               protobuf-JSON transport over replapi.App
```

The service serializes work within one session because a Goja runtime cannot be used concurrently. Different sessions have different runtimes and may progress independently. Keep the `App` alive for as long as callers need live continuity, and treat a session ID as the handle for all later operations.

## Choose a Profile

Profiles are complete policy presets, not merely storage switches. They select evaluation rewriting, observation, timeout, and persistence behavior together.

| Profile | Evaluation and observation | SQLite | Use it for |
|---|---|---:|---|
| `raw` | Raw execution, 5-second default timeout, no binding tracking or console capture | No | Minimal stateful Goja execution |
| `interactive` | Instrumented execution, last-expression capture, narrow top-level `await`, static/runtime reports, bindings, console, and JSDoc | No | TUI, editor, or process-local REPL |
| `persistent` | Interactive policy plus durable evaluations, binding versions, binding docs, and per-session ownership fencing | Required | Sequential CLI use or concurrent services that route each session to one lease owner |

`replapi.New` defaults to the **persistent** profile. Therefore, calling `replapi.New(parentCtx, factory, logger)` without a store fails. Choose `ProfileRaw` or `ProfileInteractive` explicitly for an in-memory app. The required parent context makes runtime ownership explicit.

The top-level-`await` support is intentionally narrow: it recognizes a source cell whose trimmed text starts with `await ` or `await(`. It is not general ECMAScript module top-level await; for example, a declaration containing `await` is not covered by that convenience wrapper.

## Embed an Interactive Session

An interactive app is the smallest useful long-lived host. Build the runtime factory and app once, create a session once, then retain its ID.

```go
package main

import (
    "context"
    "fmt"

    "github.com/go-go-golems/go-go-goja/pkg/engine"
    "github.com/go-go-golems/go-go-goja/pkg/replapi"
    "github.com/rs/zerolog"
)

func main() {
    // This context is the parent of runtime-owned resources.
    appCtx := context.Background()

    factory, err := engine.NewRuntimeFactoryBuilder().Build()
    if err != nil {
        panic(err)
    }
    app, err := replapi.New(
        appCtx,
        factory,
        zerolog.Nop(),
        replapi.WithProfile(replapi.ProfileInteractive),
    )
    if err != nil {
        panic(err)
    }

    session, err := app.CreateSession(context.Background())
    if err != nil {
        panic(err)
    }
    defer func() { _ = app.Close(context.Background()) }()

    if _, err := app.Evaluate(context.Background(), session.ID, `const answer = 41`); err != nil {
        panic(err)
    }
    response, err := app.Evaluate(context.Background(), session.ID, `answer + 1`)
    if err != nil {
        panic(err)
    }

    fmt.Println(response.Cell.Execution.Result) // 42
}
```

The context passed to `replapi.New` owns app and runtime lifetime. The context passed to `CreateSession` controls startup only, so an HTTP request may safely create a session without canceling its timers and asynchronous resources when the handler returns. Evaluation calls may use shorter request-scoped contexts; the session policy also applies its own evaluation timeout.

Use `UnloadSession` for non-destructive per-session cleanup and `App.Close` for app-wide cleanup. Unloading an in-memory session discards its only state. `DeleteSession` is a product deletion operation and soft-deletes durable history.

## Add Durable Replay

A persistent app needs a `repldb.Store`. The caller owns the store and must close it after all app operations have stopped.

```go
store, err := repldb.Open(appCtx, "repl.sqlite")
if err != nil {
    return err
}
defer store.Close()

app, err := replapi.New(
    appCtx,
    factory,
    logger,
    replapi.WithProfile(replapi.ProfilePersistent),
    replapi.WithStore(store),
)
if err != nil {
    return err
}
defer app.Close(context.Background()) // runs before store.Close because of LIFO defers
```

Creating a persistent session writes its ID, timestamp, profile, and policy. Every cell that reaches a completed cell report writes the raw source and its reports, including empty input and JavaScript parse/runtime failures. Persisting empty cells keeps durable cell IDs contiguous and replay numbering stable.

### How restore works

When `AutoRestore` is enabled, `Evaluate`, `Snapshot`, `Bindings`, and `WithRuntime` restore a session on demand if its ID is absent from the app's live-session map. `Restore` performs the same operation explicitly:

1. Load the durable session metadata and ordered raw source cells.
2. Create a fresh runtime with the current `engine.RuntimeFactory`.
3. Replay every stored source cell with persistence disabled during replay.
4. Install the reconstructed runtime under the original session ID.

Replay uses the policy saved with the session when that metadata is available. It does not restore serialized closures, pending promises, open files, timers, sockets, plugin processes, or other external resources. Module registration and host configuration come from the **current** factory, so the factory must remain compatible with the environment that produced the journal.

Replay can also repeat side effects. Code that writes files, calls services, reads time or randomness, or depends on changing external state may produce a different result when restored. Persistent sessions work best when setup cells are deterministic and external effects are explicit.

### Commit failure, exact retry, and recovery

JavaScript mutation cannot be rolled back if SQLite append fails after execution. `Evaluate` therefore returns both the populated cell response and a typed `replsession.CommitError`, marks the live session degraded, and rejects later JavaScript before it runs. Do not resubmit the source.

Choose one explicit recovery path:

1. `RetryPendingCommit` writes the exact retained `EvaluationRecord` without rerunning JavaScript. On success, the pending cell becomes committed history and evaluation may continue with the next contiguous cell ID.
2. `RecoverSession` unloads the suspect VM and restores only the last durable journal head. The uncommitted JavaScript mutation and any in-memory effects are discarded, but external side effects that already occurred cannot be undone.

`SessionHealth` reports `healthy`, `degraded`, or `fenced`. Both degraded and fenced states reject new evaluation. A fenced session must be unloaded/recovered after ownership can be reacquired; its stale VM must never continue writing.

### SQLite migration and backup runbook

`repldb.Open` reads `repldb_meta.schema_version`, rejects databases newer than the binary, and applies each pending migration in its own immediate SQLite transaction. Empty databases apply v1 followed by the v2 per-session lease schema; existing v1 databases preserve their journal data while adding `session_leases`. A failed migration rolls back its statements and does not advance the version. Concurrent openers serialize through SQLite and re-read the version while holding the migration transaction.

Before deploying a binary with a newer schema:

1. Stop writers and call `App.Close` before `Store.Close`.
2. Create a consistent SQLite backup, for example `sqlite3 repl.sqlite '.backup repl-before-upgrade.sqlite'`. Do not copy only the main file while WAL writers are active.
3. Verify the backup with `PRAGMA integrity_check` and retain it until the upgraded application has been exercised.
4. Start one upgraded instance and inspect startup errors before allowing normal traffic.

Schema upgrades are forward-only. Older binaries return `repldb.ErrDatabaseTooNew`; they do not relabel or downgrade the database. If migration fails, preserve the original error and database, correct the environmental or migration issue, and retry. Restore the verified backup rather than manually editing `repldb_meta`.

### Single active owner per session

Persistent apps use a schema-v2 SQLite lease for each live session. Every `App` receives a random process-unique owner ID. Create or restore acquires ownership before publishing a runtime; another unexpired owner receives typed `repldb.ErrSessionOwned`. Different sessions may be owned by different processes concurrently.

The default lease TTL is 30 seconds and can be changed with `WithLeaseTTL`. Ownership renews before evaluation and `WithRuntime`, and at one-third of the TTL during long evaluation or replay. Expired takeover increments a monotonic fencing epoch. Every durable append verifies owner ID, epoch, non-expiry, and expected next cell ID in the same transaction.

If another app takes over an expired lease, the stale app becomes `fenced` before its next JavaScript operation. `App.Close`, `UnloadSession`, and delete release ownership before the caller closes the store. `RecoverSession` can discard a fenced VM and restore after the current owner releases or expires.

Lease fencing protects SQLite, not arbitrary external systems. JavaScript that writes files, calls remote services, or emits messages still needs idempotency keys or an external fencing mechanism.

## Use the Persistent CLI

The `goja-repl` state commands create a persistent app for each invocation. This demonstrates durable reconstruction: every later command opens the database and restores the session before operating on live state.

```bash
DB=/tmp/goja-repl.sqlite

SESSION_ID=$(
  go run ./cmd/goja-repl --db-path "$DB" create |
  jq -r '.session.id'
)

go run ./cmd/goja-repl --db-path "$DB" eval \
  --session-id "$SESSION_ID" \
  --source 'const answer = 41; answer'

go run ./cmd/goja-repl --db-path "$DB" eval \
  --session-id "$SESSION_ID" \
  --source 'answer + 1'

go run ./cmd/goja-repl --db-path "$DB" history \
  --session-id "$SESSION_ID"
```

The second `eval` does not reconnect to the first process's VM; the first command releases its lease during app close, then the second command acquires the next epoch, reconstructs a new VM by replaying the first cell, and evaluates `answer + 1`.

Use `goja-repl tui --profile interactive` for process-local state, or `goja-repl tui --profile persistent --session-id <id>` to open a durable session. A `--session-id` is accepted only with the persistent TUI profile.

## Expose the HTTP API

`replhttp.NewHandler(app)` maps a persistent app to protobuf-JSON routes. Keep one app and one handler for the server lifetime.

```go
handler, err := replhttp.NewHandler(
    app,
    replhttp.WithMaxRequestBodyBytes(1<<20),
    replhttp.WithMaxSourceBytes(256<<10),
    replhttp.WithHandlerLogger(logger),
)
if err != nil {
    return err
}

server := &http.Server{
    Addr:              "127.0.0.1:3090",
    Handler:           handler,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       60 * time.Second,
    MaxHeaderBytes:    1 << 20,
}
```

The ready-made command uses this setup:

```bash
go run ./cmd/goja-repl --db-path /tmp/goja-repl.sqlite serve \
  --addr 127.0.0.1:3090
```

Create and evaluate a session with the `/api` routes:

```bash
SESSION_ID=$(
  curl -s -X POST http://127.0.0.1:3090/api/sessions |
  jq -r '.session.id'
)

curl -s -X POST \
  "http://127.0.0.1:3090/api/sessions/$SESSION_ID/evaluate" \
  -H 'Content-Type: application/json' \
  -d '{"schemaVersion":1,"source":"const answer = 41; answer"}'

curl -s -X POST \
  "http://127.0.0.1:3090/api/sessions/$SESSION_ID/evaluate" \
  -H 'Content-Type: application/json' \
  -d '{"schemaVersion":1,"source":"answer + 1"}'
```

The transport contract is `proto/goja/replapi/v1/replapi.proto`. Responses use protobuf JSON field names such as `schemaVersion` and `sessionExport`. Evaluate requests must use `Content-Type: application/json`, contain exactly the supported nonzero `schemaVersion`, and have no unknown fields. The default handler accepts at most 1 MiB of request body and 256 KiB of UTF-8 JavaScript source; handler options may lower or raise both limits, but the source limit cannot exceed the body limit. `ExecutionReport.resultJson` remains a string containing JSON, while protobuf `int64` fields decode to JavaScript `bigint` with `@bufbuild/protobuf` v2.

Transport and infrastructure failures use protobuf `ErrorResponse` with `schemaVersion`, stable `code`, safe `message`, and `requestId`. The handler accepts a valid caller-provided `X-Request-ID` or generates one, returns it on success and failure, sends `X-Content-Type-Options: nosniff` and `Cache-Control: no-store`, logs detailed failures through the configured logger, and redacts SQL, filesystem, plugin, panic, and runtime internals by default. `WithExposeInternalErrors(true)` exists only for trusted local debugging. JavaScript parse/runtime failures remain HTTP 200 cell reports because they are evaluation results, not transport failures.

| HTTP | Stable codes | Meaning |
|---:|---|---|
| 400 | `invalid_argument`, `invalid_content_type`, `unsupported_schema_version` | Malformed or incompatible request |
| 404 | `session_not_found` | Session is absent or deleted |
| 409 | `session_owned`, `session_not_writable` | Another owner holds the lease, or this VM is degraded/fenced |
| 413 | `request_too_large`, `source_too_large` | Configured resource limit exceeded |
| 500 | `internal` | Redacted unexpected infrastructure failure |
| 503 | `persistence_unavailable`, `service_shutting_down`, `service_unavailable` | Commit, lifecycle, or cancellation prevents service |

The handler does not add authentication, authorization, CORS policy, rate limiting, or tenant isolation. It can execute JavaScript with every host capability exposed by the runtime factory. The ready-made command binds to loopback by default and refuses a non-loopback address unless `--allow-remote` explicitly acknowledges the risk. An intentional remote deployment should also use `--safe-mode` or a minimal `--enable-module` allowlist and place authentication, authorization, TLS, rate limits, and audit middleware in front of the returned `http.Handler`.

`POST /api/sessions` uses its request context only for session startup. Runtime-owned timers and asynchronous work inherit the app context passed to `replapi.New`, so they remain alive after the create response and stop during app shutdown. Request cancellation still stops the individual queued or active operation without poisoning the long-lived session.

### HTTP routes

| Method | Path | Meaning |
|---|---|---|
| `GET` | `/api/sessions` | List non-deleted durable sessions |
| `POST` | `/api/sessions` | Create a persistent session |
| `GET` | `/api/sessions/{id}` | Snapshot the live session, restoring it if needed |
| `DELETE` | `/api/sessions/{id}` | Close and logically delete the session |
| `POST` | `/api/sessions/{id}/evaluate` | Evaluate one source cell |
| `POST` | `/api/sessions/{id}/restore` | Explicitly reconstruct the live runtime |
| `GET` | `/api/sessions/{id}/history` | Read durable evaluation records |
| `GET` | `/api/sessions/{id}/bindings` | Inspect current live bindings, restoring if needed |
| `GET` | `/api/sessions/{id}/docs` | Read persisted binding documentation |
| `GET` | `/api/sessions/{id}/export` | Export the durable session and evaluations |

## Migrate Existing Hosts

The hardened API intentionally makes ownership decisions compile-visible. There are no compatibility shims for the old constructor or runtime callback shapes.

| Before | Now | Required decision |
|---|---|---|
| `replapi.New(factory, logger, opts...)` | `replapi.New(appCtx, factory, logger, opts...)` | Choose the context that owns every runtime and lease in the app |
| `replapi.NewWithConfig(factory, logger, config)` | `replapi.NewWithConfig(appCtx, factory, logger, config)` | Use the same explicit app lifetime |
| `ConfigForProfile(profile)` | `ConfigForProfile(profile) (Config, error)` | Handle unknown profiles rather than accepting partial presets |
| `WithRuntime(ctx, id, func(*engine.Runtime) error)` | `WithRuntime(ctx, id, func(opCtx context.Context, rt *engine.Runtime) error)` | Honor cancellation from caller, unload, and shutdown |
| close only `repldb.Store` | call `App.Close`, then `Store.Close` | Stop runtime work and release per-session leases before SQLite closes |

A minimal shutdown sequence is:

```go
closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
appErr := app.Close(closeCtx)
storeErr := store.Close()
return errors.Join(appErr, storeErr)
```

Do not replace the required parent with `context.Background()` merely to make compilation pass unless the process lifetime truly owns the app. Servers should use a service context, cancel/stop inbound work, wait for handlers, close the app, and close the store last. TUI adapters do not own the app; the command or host that created it must perform shutdown.

For HTTP clients, every evaluate request must now include `Content-Type: application/json` and the exact `schemaVersion`. Decode non-2xx bodies as generated `ErrorResponse`; do not depend on historical `{"error":"..."}` strings or raw internal error text.

## Go API Reference

The app combines live-runtime methods and store-backed methods. The distinction matters when selecting an in-memory profile.

| Method | State used | Notes |
|---|---|---|
| `CreateSession` | Live; optionally durable | Uses app defaults |
| `CreateSessionWithOptions` | Live; optionally durable | Sparse ID, timestamp, profile, and full-policy overrides |
| `Evaluate` | Live, with optional auto-restore | Returns JavaScript errors in `Cell.Execution`; infrastructure errors use Go `error` |
| `Snapshot` | Live, with optional auto-restore | Returns current reports and bindings |
| `Bindings` | Live, with optional auto-restore | Requires observation to have tracked useful bindings |
| `WithRuntime` | Live, with optional auto-restore | Runs a context-aware callback while owning the session operation gate |
| `Restore` | Durable then live | Replays source into a new runtime |
| `SessionHealth` | Live | Reports healthy, degraded, or fenced state |
| `RetryPendingCommit` | Live and durable | Retries the exact retained record without rerunning JavaScript |
| `RecoverSession` | Durable then live | Discards suspect VM state and restores the last durable head |
| `UnloadSession` | Live | Closes and evicts the runtime without deleting durable history |
| `DeleteSession` | Live and, when persistent, durable | Closes runtime and soft-deletes the durable session |
| `Close` | All live sessions | Idempotently unloads all runtimes; does not close the caller-owned store |
| `ListSessions`, `History`, `Docs`, `Export`, `ReplaySource` | Durable | Return an error when no store is configured |

`WithRuntime` is the controlled escape hatch for host integrations. Its callback receives an operation context that is canceled by caller cancellation, unload, or app shutdown. Honor that context; do not retain the runtime pointer after the callback returns; and do not call another app method for the same session from inside the callback because the callback already owns that session's capacity-one operation gate.

A JavaScript parse error, thrown value, rejected promise, or timeout is normally represented by `response.Cell.Execution.Status` and `response.Cell.Execution.Error`; it is not necessarily returned as the Go `error`. Always inspect the cell status. Go errors represent failures such as missing sessions, canceled infrastructure work, runtime observation failures, or persistence failures.

## Profiles and Policy Overrides

Profiles are validated, canonical preset names. `ParseProfile` trims and lowercases user input, while `ConfigForProfile` returns the complete preset or `ErrUnknownProfile`:

```go
profile, err := replapi.ParseProfile(userInput)
if err != nil {
    return err
}
config, err := replapi.ConfigForProfile(profile)
if err != nil {
    return err
}
```

A bare explicit config such as `Config{Profile: ProfileRaw}` resolves the complete raw preset, including raw evaluation and its default timeout. If both `Config.Profile` and `SessionOptions.Profile` are present, they must name the same preset; contradictory labels return `ErrProfileMismatch` rather than silently mixing behavior. Unknown app and per-session profiles return `ErrUnknownProfile`.

Most callers should use `WithProfile` or a profile config helper. If a host needs custom behavior, use `WithDefaultSessionPolicy` for the app default or `SessionOverrides.Policy` for one new session.

A policy override replaces the full policy; it is not merged field-by-field with the selected profile. Zero-valued booleans therefore disable features. Start from a preset and modify it rather than constructing a partial policy accidentally:

```go
options := replsession.InteractiveSessionOptions()
options.Policy.Eval.TimeoutMS = 1500
options.Policy.Observe.JSDocExtraction = false

app, err := replapi.New(
    appCtx,
    factory,
    logger,
    replapi.WithProfile(replapi.ProfileInteractive),
    replapi.WithDefaultSessionPolicy(options.Policy),
)
```

Enabling `Persist.Enabled` requires a configured store. The subordinate persistence flags select which evaluation, binding-version, and binding-document records are written; they do not enable persistence by themselves.

## Unload, Deletion, and Shutdown

`DeleteSession` has product semantics, not merely resource-cleanup semantics. For a persistent session it:

1. removes the session from the live map,
2. sets `deleted_at` in SQLite when persistence is enabled,
3. closes its runtime even if the persistence update fails.

Normal list, load, history, export, and restore paths hide soft-deleted sessions. Deletion does not physically erase the database rows.

`UnloadSession` closes and removes only the live runtime. It leaves durable rows intact and releases the session lease, so a persistent app with auto-restore can reacquire and reconstruct that session on the next operation. `App.Close` performs this non-destructive unload for every live session, is safe to call repeatedly or concurrently, and aggregates runtime/lease errors. It deliberately does not close the caller-owned store; close the app first and the store second.

Shutdown rejects new app operations with typed `ErrAppClosing` or `ErrAppClosed` errors. A bounded unload/close that times out while an operation is active leaves the session reachable in a closing state. A later `UnloadSession` or `Close` call can retry after the operation releases its gate. Delete retains distinct product semantics and must not be used merely for eviction.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `replapi: auto-restore requires a store` | `replapi.New` used its persistent default without SQLite | Pass `WithStore(store)`, or explicitly choose `ProfileInteractive`/`ProfileRaw` |
| `replapi: persistence requested but no store configured` | A per-session profile or policy enabled persistence without a store | Configure `WithStore`, or use a non-persistent policy |
| A variable disappears between calls | The app or session ID changed | Reuse one app and session ID, or use persistent replay across app instances |
| Async module work stops unexpectedly | The parent passed to `replapi.New` was canceled | Keep the app parent alive until shutdown; request contexts belong on individual operations |
| Restore changes behavior or repeats an external action | Replay re-executes raw source against the current factory and environment | Keep replayable setup deterministic and isolate external side effects |
| Evaluation returns a cell and `ErrCommitFailed` | JavaScript executed but SQLite append failed | Do not rerun source; call `RetryPendingCommit` or `RecoverSession` |
| Session returns `ErrSessionDegraded` | An uncommitted executed cell is pending | Retry the exact commit or recover from durable history |
| Restore returns `ErrSessionOwned` | Another app has an unexpired lease for that session | Route to the current owner, close it cleanly, or wait for expiry before takeover |
| Session returns `ErrSessionFenced` | Its lease expired and another owner took over, or its durable head changed | Stop using that VM; unload/recover after ownership is available |
| Database returns `ErrDatabaseTooNew` | A newer binary already upgraded the schema | Use a compatible/newer binary; do not edit the version metadata |
| Fenced append returns `ErrWriteConflict` | Durable next cell differs from the VM's expected cell | Treat the VM as fenced and recover from the durable head |
| `WithRuntime` hangs | The callback re-entered the same session or ignored its operation context during shutdown | Keep it non-reentrant, honor `opCtx`, and use the provided runtime directly |
| `Evaluate` returned no Go error but JavaScript failed | JS failures are reported in the cell report | Check `Cell.Execution.Status` and `Cell.Execution.Error` |
| A deleted session cannot be restored | `DeleteSession` soft-deletes it and normal reads hide deleted rows | Use `UnloadSession` for persistent-runtime eviction |
| HTTP execution is reachable by untrusted clients | The built-in handler has no security middleware | Bind to loopback or add authentication, authorization, limits, and sandboxed modules |

## See Also

- `goja-repl help repl-usage` — interactive TUI, script execution, modules, and transport payload examples.
- `goja-repl help goja-plugin-user-guide` — configure plugin-backed modules used by REPL runtimes.
- `proto/goja/replapi/v1/replapi.proto` — authoritative protobuf transport schema.
- `pkg/replapi/app.go` and `pkg/replapi/config.go` — app methods, profiles, and validation.
- `pkg/replsession/service.go` and `pkg/replsession/policy.go` — live session ownership and evaluation policy.
- `cmd/goja-repl/cmd_serve.go` — current server construction and shutdown path.
