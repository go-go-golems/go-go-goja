---
Title: REPL API lifecycle, ownership, and HTTP hardening implementation guide
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
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/goja-repl/cmd_serve.go
      Note: |-
        P6 complete HTTP server limits and explicit non-loopback acknowledgement
        P7 HTTP shutdown/wait before app and store cleanup
    - Path: repo://cmd/goja-repl/root.go
      Note: P7 shared bounded app-before-store cleanup and command error aggregation
    - Path: repo://cmd/goja-repl/shutdown_test.go
      Note: P7 lease-release ordering and shutdown diagnostic tests
    - Path: repo://cmd/goja-repl/tui.go
      Note: P7 TUI owner shutdown after adapter/UI completion
    - Path: repo://pkg/doc/34-replapi-guide.md
      Note: P7 public breaking-API migration and final lifecycle/HTTP guidance
    - Path: repo://pkg/engine/runtime.go
      Note: Owned runtime context, close hooks, interruption, and shutdown
    - Path: repo://pkg/replapi/app.go
      Note: Application facade, explicit parent context, auto-restore, and lifecycle enforcement
    - Path: repo://pkg/replapi/config.go
      Note: Profile presets, partial config normalization, and validation
    - Path: repo://pkg/repldb/store.go
      Note: Transactional schema bootstrap, schema-v2 lease migration, version validation, and WAL configuration
    - Path: repo://pkg/replhttp/handler.go
      Note: P6 bounded handler defaults, stable typed error mapping, request IDs, security headers, panic recovery, and redaction
    - Path: repo://pkg/replhttp/proto_handler.go
      Note: |-
        Protobuf JSON routes, request context propagation, and body parsing
        P6 strict content/body/source/schema parsing and protobuf-JSON route responses
    - Path: repo://pkg/replhttp/security_test.go
      Note: P6 real HTTP acceptance tests for limits, cancellation, mapping, redaction, ownership, and headers
    - Path: repo://pkg/replsession/evaluate.go
      Note: Serialized evaluation pipeline, delayed commit publication, and degraded-session gating
    - Path: repo://pkg/replsession/service.go
      Note: Live session ownership, lease transfer/heartbeat, app-derived runtime contexts, restore publication, and lifecycle integration
    - Path: repo://proto/goja/replapi/v1/replapi.proto
      Note: P6 public ErrorResponse transport contract
ExternalSources: []
Summary: Evidence-backed architecture and intern-oriented implementation guide for fixing replapi context ownership, graceful shutdown, persistent-session split brain, persistence divergence, cancellation, configuration, SQLite migration, and HTTP safety.
LastUpdated: 2026-07-15T14:45:00-04:00
WhatFor: Implement and review a safe long-lived replapi service without confusing live Goja runtime ownership with durable replay.
WhenToUse: Read before changing pkg/replapi, pkg/replsession, pkg/repldb, pkg/replhttp, or the goja-repl persistent CLI and server lifecycle.
---




# REPL API lifecycle, ownership, and HTTP hardening implementation guide

## Executive Summary

The current REPL subsystem has a sound high-level package split: `replsession` owns live Goja sessions, `replapi` combines that live kernel with optional SQLite persistence, `replhttp` exposes protobuf JSON, and `cmd/goja-repl` provides CLI, TUI, and server surfaces. The evaluator already serializes calls within a session, interrupts runaway JavaScript, records rich analysis, and reconstructs persistent sessions by replaying stored source. Those are useful foundations and should be preserved.

The subsystem is not yet a safe long-lived service boundary. Five executable probes written for this ticket confirmed that lifecycle and durable-ownership assumptions leak across package boundaries:

1. HTTP session creation uses the request context as the runtime lifetime context. It is canceled as soon as `ServeHTTP` returns.
2. Two `replapi.App` instances can restore the same durable session into independent VMs. Both can execute the same next cell; one mutates its VM and then loses on the SQLite unique constraint.
3. A post-execution persistence failure leaves the VM mutated and the in-memory cell accepted. The next cell can persist, producing a durable sequence such as cells `1, 3` with cell `2` missing.
4. `NewWithConfig(..., Config{Profile: ProfileRaw})` labels the session `raw` but gives it instrumented execution and a zero timeout rather than the raw preset.
5. A caller whose context expires while waiting for a busy session remains blocked on `sync.Mutex`; the operation can later return success after its deadline.

Two previously identified limitations compound these findings:

- `replapi.App` and `replsession.Service` have no app-wide close operation and no non-destructive way to unload a runtime while retaining durable history.
- The HTTP transport has no request body limit, schema-version validation, authentication boundary, safe error contract, or complete server timeout policy.

The recommended design establishes explicit ownership at four levels:

```text
host process
  owns replapi.App and its lifetime context
      owns one replsession.Service
          owns zero or more live session entries
              owns exactly one engine.Runtime per live session

repldb.Store
  owns durable history and a per-session lease/fencing record

HTTP request
  owns only one operation context
  never owns the session runtime lifetime
```

The implementation should add:

- an app lifetime context set at construction;
- idempotent `App.Close`, `Service.Close`, and non-destructive `UnloadSession` operations;
- context-aware per-session operation gates and explicit lifecycle states;
- fail-closed handling after a persistence write fails;
- SQLite schema migrations, per-session leases, and fencing tokens;
- strict profile validation and correct preset resolution;
- bounded protobuf-JSON HTTP requests, stable error responses, schema-version validation, and hardened server defaults;
- regression, race, failure-injection, migration, CLI, TUI, and real HTTP-server tests.

The work should be delivered in phases. Lifecycle and persistence correctness must land before broader transport polish. A new intern should not try to implement this as one large patch.

## Problem Statement and Scope

### What `replapi` is supposed to provide

A REPL session is a stateful sequence of JavaScript evaluations. Within one live process, later cells see declarations and mutations made by earlier cells. In persistent mode, the source journal and reports are stored in SQLite so another runtime can be reconstructed later.

The public mental model should be:

```text
live session = one owned Goja runtime + one serialized operation stream
persistent session = live session + durable replay journal + one active owner
```

The current code implements the first half of each sentence but does not fully enforce the ownership terms. It serializes operations inside one `replsession.Service`, but it does not coordinate the same durable session across apps. It stores a replay journal, but it permits the live VM to advance after a journal write fails. It creates runtime lifetime contexts, but it derives them from individual create/restore operation contexts.

### Goals

This ticket should make the following contracts explicit and enforceable:

1. A session runtime is parented by the app lifetime, not by a request.
2. Every live runtime can be closed deterministically.
3. Unloading a runtime is distinct from deleting durable session history.
4. One persistent session has at most one active app owner.
5. A stale owner cannot append to the durable journal.
6. A failed durable write prevents later VM/journal divergence.
7. Waiting for a busy session honors caller cancellation.
8. Profile names and policies cannot silently disagree.
9. HTTP requests are bounded, versioned, and safe to compose behind authentication.
10. SQLite schema changes are migrated rather than silently declared current.

### Non-goals

This ticket does not attempt to:

- serialize a Goja heap, closure, event loop, pending promise, open file, socket, watcher, or plugin process;
- make replay side-effect-free;
- provide a multi-tenant authentication product inside `pkg/replhttp`;
- implement distributed execution or transparent failover;
- preserve stale constructor APIs with permanent compatibility shims;
- redesign JavaScript analysis, rewriting, completion, or JSDoc behavior except where lifecycle changes require tests.

Authentication remains a host concern, but this ticket must make unsafe deployment harder and define middleware seams clearly.

## Reader Orientation: Terms and Invariants

### Runtime

`engine.Runtime` wraps a `*goja.Runtime`, Node-style event loop, require registry, `runtimeowner.RuntimeOwner`, runtime-scoped values, a lifetime context, and close hooks. It is the resource that must never be accessed concurrently without its owner and must eventually be closed (`pkg/engine/runtime.go:33-127`).

### Runtime owner

`runtimeowner.RuntimeOwner` serializes actual VM calls. Its `Call` operation accepts a context, queues work, and returns when the call executes or the context expires (`pkg/runtimeowner/runner.go:90-156`). This protects Goja itself, but it does not replace the higher-level `sessionState.mu` used by `replsession` to serialize the complete analyze/execute/observe/persist pipeline.

### Live session

A live session is a `replsession.sessionState` in `Service.sessions`. It owns a runtime, cell counter, reports, binding catalog, console sink, policy, and a `sync.Mutex` (`pkg/replsession/service.go:35-73`). The session mutex covers much more than the VM call: it prevents two evaluations from interleaving analysis, execution, snapshots, and persistence.

### Durable session

A durable session is a row in `repldb.sessions` plus ordered rows in `evaluations` and child tables (`pkg/repldb/schema.go:8-88`). Durable state is a replay journal and audit record. It is not the live runtime.

### Restore

Restore loads a durable session and ordered raw source, creates a temporary live session, replays every stored source cell with persistence disabled, and publishes the resulting state under the original ID (`pkg/replapi/app.go:84-98`, `pkg/replsession/service.go:260-324`).

### Delete, unload, and close

These terms must remain distinct:

- **Delete:** close a live runtime and logically delete durable history from normal product views.
- **Unload:** close and remove a live runtime but retain durable history for later restore.
- **Close app:** reject new work, unload all live runtimes, release ownership, and leave durable history intact.

The current API only implements delete.

### Required invariants

Use these as review criteria for every implementation phase:

```text
I1. One session entry owns one runtime.
I2. One durable session has at most one active app lease.
I3. One session executes at most one high-level operation at a time.
I4. A caller canceled while waiting does not execute later.
I5. Closing prevents new operations before resources are released.
I6. App close and unload do not soft-delete durable records.
I7. Delete does soft-delete persistent records.
I8. A durable cell write is fenced by owner epoch and expected cell number.
I9. After an uncommitted evaluation, no later cell may execute in that VM.
I10. A profile name resolves to exactly one documented preset.
I11. HTTP request context bounds an operation, never the runtime lifetime.
I12. External JavaScript side effects are not transactionally rolled back.
```

## Current-State Architecture

### Package map

The current architecture is layered correctly enough to evolve without a wholesale rewrite:

```text
cmd/goja-repl
  root.go            constructs factory, store, app, commands
  tui.go             adapts one replapi session to Bobatea
  cmd_serve.go       exposes replhttp handler through net/http

pkg/repl/adapters/bobatea
  replapi.go         maps Bobatea evaluator/help calls to replapi

pkg/replhttp
  proto_handler.go   method/path routing and protobuf JSON conversion
  handler.go         recovery, status mapping, JSON errors

pkg/replapi
  config.go          profiles, app config, session overrides
  app.go             live/durable facade and auto-restore

pkg/replsession
  service.go         session map, creation, restore, deletion
  evaluate.go        analyze/rewrite/execute/observe/persist pipeline
  persistence.go     conversion to repldb records
  policy.go          evaluation/observation/persistence policy
  observe.go         runtime snapshots and summaries

pkg/repldb
  store.go           SQLite open/bootstrap
  schema.go          schema v1 DDL
  write.go           sessions, evaluations, bindings, docs
  read.go            list/load/history/replay/export

pkg/engine + pkg/runtimeowner
  runtime.go         concrete runtime lifecycle
  factory.go         module wiring and runtime context creation
  runtimeowner       serialized VM execution
```

### Live session creation flow

The app resolves a profile and delegates to the service. The service creates the runtime before publishing the session in the map.

```text
caller context
    |
    v
App.CreateSession(ctx)
    |
    v
resolve profile/policy
    |
    v
Service.CreateSessionWithOptions(ctx)
    |
    +--> RuntimeFactory.NewRuntime(
    |      WithStartupContext(ctx),
    |      WithLifetimeContext(ctx))
    |
    +--> install console/JSDoc hooks
    +--> optionally insert durable session row
    +--> publish sessionState in Service.sessions
```

The critical defect is visible in the two identical context arguments at `pkg/replsession/service.go:146`. A create request's deadline is appropriate for setup. It is not an appropriate parent for timers, filesystem async work, plugin invocations, or other runtime-owned resources.

### Evaluation flow

Evaluation holds `sessionState.mu` for the complete operation (`pkg/replsession/evaluate.go:30-37`):

```text
get session pointer
lock session mutex
increment nextCellID
analyze source
rewrite source
snapshot globals before
execute through RuntimeOwner
snapshot globals after
update bindings and reports
append in-memory cell
persist SQLite evaluation
return response
unlock session mutex
```

This ordering preserves a coherent in-memory report when persistence succeeds. It cannot roll back the VM if persistence fails after execution.

### Persistent restore flow

`App.ensureLiveSession` first attempts a live snapshot. If the session is absent and auto-restore is enabled, it calls `Restore` (`pkg/replapi/app.go:186-199`). Restore loads the record and replay source, then asks `Service.RestoreSession` to build the runtime.

```text
request context
    |
    v
load session metadata + raw source from SQLite
    |
    v
create temporary session runtime using request context
    |
    v
for each source cell:
    evaluate with persistence disabled
    |
    v
rename temporary session to durable session ID
publish in live map
```

Restore has an in-app race defense: if two goroutines in the same service restore concurrently, one map insertion wins and the loser closes its temporary runtime (`pkg/replsession/service.go:308-318`). That defense does not extend across two `App` instances.

### Deletion flow

`Service.DeleteSession` removes the entry from the map, writes durable deletion metadata if enabled, and closes the runtime (`pkg/replsession/service.go:234-258`). It does not acquire the session mutex before closing. An evaluation may already hold a pointer and be operating on the same state.

The database uses soft deletion. `ListSessions` and `LoadSession` hide rows whose `deleted_at` is non-null (`pkg/repldb/read.go:17-88`). This is correct product behavior for delete, but it makes `DeleteSession` unsuitable as a cleanup-only or eviction API.

### HTTP flow

`replhttp.NewHandler` registers Go 1.22 method/path handlers under `/api` (`pkg/replhttp/proto_handler.go:22-149`). The evaluate route calls `io.ReadAll`, unmarshals strict protobuf JSON, invokes the app, converts the response, and writes JSON.

```text
net/http request context
      |
      +--> POST /api/sessions --> App.CreateSession(request context)
      |
      +--> evaluate --> unbounded io.ReadAll --> strict protojson --> App.Evaluate
      |
      +--> errors --> plain {"error":"internal details"}
```

The command server sets only `ReadHeaderTimeout` (`cmd/goja-repl/cmd_serve.go:57-61`). It has no app close after HTTP shutdown; it closes only the SQLite store through a defer.

## Evidence and Reproduction Results

The ticket includes five standalone probes under `scripts/`. They are deliberately small, use public APIs where possible, and should be converted into regression tests during implementation.

### Probe 1: HTTP request context cancels runtime resources

Command:

```bash
go run ./ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/01-probe-http-session-context
```

Observed output:

```text
session=session-8a973f8b-2559-46ce-a222-bec0b38ddea2 runtime_lifetime_error=context canceled
```

The probe uses a real `httptest.Server`, creates a session through `POST /api/sessions`, waits for the handler to return, and inspects `engine.Runtime.Context()` through `App.WithRuntime`. The VM remains in the live map, but its resource lifetime has already been canceled.

### Probe 2: two apps create persistent split brain

Command:

```bash
go run ./ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/02-probe-persistent-split-brain
```

Observed output:

```text
appA: cell=2 status=ok error=<nil>
appB: response=nil error=persist cell: write evaluation: persist evaluation: insert evaluation: UNIQUE constraint failed: evaluations.session_id, evaluations.cell_id
durable cell ids: 1 2
```

Both apps restored cell 1 into separate VMs. App A and app B each executed cell 2. App B's JavaScript mutation happened before its database insert failed. The durable journal contains app A's cell 2, while app B remains a mutated, stale live VM.

### Probe 3: partial raw config is not raw

Command:

```bash
go run ./ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/03-probe-partial-profile-config
```

Observed output:

```text
profile=raw eval_mode=instrumented timeout_ms=0 static_analysis=false binding_tracking=false
```

`Config{Profile: ProfileRaw}` propagates the profile string into zero-valued session options, then policy normalization defaults the missing mode to instrumented. It does not apply `RawSessionOptions`, whose documented timeout is 5000 ms (`pkg/replapi/config.go:145-161`, `pkg/replsession/policy.go:62-73`).

### Probe 4: persistence failure creates a durable gap

Command:

```bash
go run ./ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/04-probe-post-execution-persistence-failure
```

Observed output:

```text
cell2_response_nil=true cell2_error=persist cell: write evaluation: injected persistence failure
cell3_id=3 cell3_result=3 cell3_error=<nil>
snapshot_cell_count=3 snapshot_error=<nil> persisted_cell_ids=[1 3]
```

The service permits cell 3 after cell 2 fails to persist. A later restore replays two records but assigns temporary live cell numbers by replay order. The durable maximum remains 3, so future writes can conflict or attach reports to a VM built from a different sequence.

### Probe 5: canceled waiter still succeeds late

Command:

```bash
go run ./ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/05-probe-canceled-waiter
```

Observed output:

```text
snapshot_error=<nil> context_error=context deadline exceeded elapsed=301ms
```

A `WithRuntime` callback holds the session mutex for 300 ms. `Snapshot` receives a 20 ms context deadline but blocks on `sync.Mutex.Lock`, then returns success after the lock is released. The context cannot cancel queue wait because `sync.Mutex` has no context-aware acquisition.

### Race test baseline

The existing suite passes the race detector:

```bash
go test -race ./pkg/replapi ./pkg/replsession ./pkg/replhttp ./pkg/repldb ./pkg/repl/adapters/bobatea
```

This is useful but does not invalidate the findings. Split brain and persistence gaps are logical consistency failures, not necessarily Go memory races. The current test suite has no app-close, unload, lease, body-limit, or canceled-lock-wait tests.

## Findings and Gap Analysis

### Finding F1: operation context is used as runtime lifetime

- **Severity:** High
- **Evidence:** `pkg/replsession/service.go:146`; probe 1.
- **Affected flows:** direct create, explicit restore, auto-restore through evaluate/snapshot/bindings/`WithRuntime`, HTTP create, HTTP restore.
- **Consequence:** runtime-owned asynchronous resources observe cancellation when the create/restore operation ends.

The fix must separate startup context from lifetime context at the service boundary. Merely changing the HTTP handler to pass `context.Background()` would hide cancellation but create an unowned resource lifetime.

### Finding F2: no app-wide close or non-destructive unload

- **Severity:** High
- **Evidence:** `App` exposes no close method (`pkg/replapi/app.go:16-217`); `Store` is the only relevant type with `Close` (`pkg/repldb/store.go:49-55`).
- **Consequence:** embedded hosts cannot deterministically release all VMs; persistent callers cannot evict a VM without soft-deleting the durable session.

The fix must add both app-wide close and per-session unload. These are different from delete.

### Finding F3: persistent sessions have no single-owner coordination

- **Severity:** Critical
- **Evidence:** app-local maps only; `UNIQUE(session_id, cell_id)` at `pkg/repldb/schema.go:31`; probe 2.
- **Consequence:** two valid app instances can mutate divergent VMs. SQLite detects only the losing append, after JavaScript has run.

The fix needs ownership before execution, not only a clearer error after an insert conflict.

### Finding F4: persistence failure does not fail the live session closed

- **Severity:** Critical
- **Evidence:** state is mutated and appended before `persistCell` at `pkg/replsession/evaluate.go:316-345` and `403-448`; probe 4.
- **Consequence:** journal gaps, replay renumbering, future unique conflicts, and live/durable semantic divergence.

JavaScript effects cannot be rolled back. The minimum safe behavior is to mark the session persistence-degraded and reject every later evaluation until the runtime is discarded or the exact pending record is persisted.

### Finding F5: session queue wait ignores context

- **Severity:** High
- **Evidence:** `state.mu.Lock()` in evaluate, snapshot, and `WithRuntime` (`pkg/replsession/evaluate.go:36`; `pkg/replsession/service.go:212,228`); probe 5.
- **Consequence:** canceled HTTP requests remain queued, consume resources, and may execute later.

The fix needs a context-aware operation gate, not a context check after a blocking mutex returns.

### Finding F6: delete and active operations lack a lifecycle protocol

- **Severity:** High
- **Evidence:** delete removes from map and closes runtime without taking `state.mu` (`pkg/replsession/service.go:234-258`).
- **Consequence:** an operation that already resolved the state pointer can overlap runtime close. New app-wide shutdown would amplify this race unless the session has explicit active/closing/closed states.

The fix must first reject new operations, cancel active operation contexts, wait for operation ownership, and then close the runtime.

### Finding F7: partial and unknown profiles are not validated correctly

- **Severity:** Medium
- **Evidence:** `ConfigForProfile` falls through to persistent for unknown values (`pkg/replapi/config.go:51-70`); partial raw config probe.
- **Consequence:** profile labels, policy behavior, timeout, and persistence requirements can disagree. A typo can silently choose persistent behavior.

The fix must reject unknown profiles and define whether `Config.Profile` is authoritative or redundant.

### Finding F8: HTTP resource and security boundary is incomplete

- **Severity:** High when remotely exposed
- **Evidence:** unbounded `io.ReadAll` at `pkg/replhttp/proto_handler.go:70`; internal error text returned by `pkg/replhttp/handler.go:45-50`; only `ReadHeaderTimeout` in `cmd/goja-repl/cmd_serve.go:57-61`.
- **Consequence:** memory amplification, expensive static analysis from oversized source, leaked internal errors, slow-client exposure, and unauthenticated execution with host-access modules.

The root factory loads the default registry unless module middleware restricts it (`cmd/goja-repl/root.go:115-176`). Server operators can use `--safe-mode`, `--enable-module`, and `--disable-module`, but the transport itself does not enforce a safe policy.

### Finding F9: schema version is emitted but not enforced

- **Severity:** Medium
- **Evidence:** `SchemaVersion = 1` at `pkg/replapi/pbconv/codec.go:16`; evaluate unmarshal does not validate it; no error response exists in `replapi.proto`.
- **Consequence:** clients can send zero or arbitrary versions, and failures return a hand-written error shape outside the protobuf contract.

### Finding F10: SQLite bootstrap is not a migration system

- **Severity:** High as soon as leases are added
- **Evidence:** schema consists of `CREATE TABLE IF NOT EXISTS`; bootstrap writes the current version unconditionally with `INSERT OR REPLACE` (`pkg/repldb/store.go:59-92`).
- **Consequence:** adding lease tables or columns can mark an incompletely upgraded database as current. Newer databases are not rejected by older binaries.

Lease/fencing work must not land before ordered, transactional migrations.

### Finding F11: `WithRuntime` is cancellation-weak and easy to misuse

- **Severity:** Medium
- **Evidence:** service explicitly ignores its context before locking (`pkg/replsession/service.go:219-230`); callback receives no context; runtime pointer can be retained.
- **Consequence:** shutdown cannot force a callback that ignores lifecycle, and callers can escape ownership after the callback.

The API should pass an operation context into the callback and document that the runtime cannot escape.

### Finding F12: replay necessarily repeats source and side effects

- **Severity:** Inherent design constraint
- **Evidence:** `LoadReplaySource` returns raw source (`pkg/repldb/read.go:201-213`); restore evaluates it cell by cell (`pkg/replsession/service.go:289-297`).
- **Consequence:** file writes, network calls, randomness, time, plugin effects, and non-idempotent host calls can differ or repeat.

This is not fully fixable without a different persistence model. The implementation should preserve the replay contract, surface provenance, and avoid describing persistence as VM resume.

## Proposed Architecture

### Ownership diagram

```text
+---------------------------------------------------------------+
| Host process                                                  |
|                                                               |
|  appParentCtx                                                 |
|      |                                                        |
|      v                                                        |
|  +-------------------- replapi.App ------------------------+  |
|  | state: open -> closing -> closed                         |  |
|  | ownerID: process-unique                                  |  |
|  | appCtx/appCancel                                         |  |
|  |                                                          |  |
|  |  +------------- replsession.Service ------------------+  |  |
|  |  |                                                    |  |  |
|  |  | session A                    session B             |  |  |
|  |  | active/closing/closed        active/...            |  |  |
|  |  | operation gate               operation gate        |  |  |
|  |  | sessionCtx/cancel            sessionCtx/cancel     |  |  |
|  |  | engine.Runtime               engine.Runtime        |  |  |
|  |  +----------------------------------------------------+  |  |
|  +----------------------------------------------------------+  |
|              |                         |                      |
+--------------|-------------------------|----------------------+
               | fenced writes           | acquire/release lease
               v                         v
       +------------------------------------------------+
       | repldb.Store / SQLite                          |
       | sessions, evaluations, bindings, docs          |
       | session_leases(owner_id, epoch, lease_until)   |
       +------------------------------------------------+
```

### Context hierarchy

```text
host parent context
    |
    +-- app context (canceled by App.Close or host shutdown)
          |
          +-- session A lifetime context
          |     |
          |     +-- evaluation operation context
          |     +-- WithRuntime callback context
          |
          +-- session B lifetime context

HTTP request context
    |
    +-- startup/restore operation only
    +-- evaluation deadline only
    X  never parent of app or session lifetime
```

### Lifecycle state machines

App state:

```text
open
  | Close begins
  v
closing -- reject Create/Evaluate/Restore/WithRuntime --> ErrAppClosing
  | all sessions stopped and leases released
  v
closed -- every operation except repeated Close --> ErrAppClosed
```

Session state:

```text
restoring -> active -> closing -> closed
                |
                +-- persistence append fails --> degraded
                |                                  |
                |                                  +-- unload + restore --> active
                |                                  +-- delete ----------> closed
                |
                +-- lease/fence lost ----------> fenced -> unload/restore
```

`degraded` and `fenced` reject new JavaScript before it executes.

## Proposed Public APIs

### `replapi` constructor and lifecycle

Make the app parent context explicit. There are few repository call sites, and a mechanical migration is safer than a hidden `context.Background()` default.

```go
func New(
    parent context.Context,
    factory *engine.RuntimeFactory,
    logger zerolog.Logger,
    opts ...Option,
) (*App, error)

func NewWithConfig(
    parent context.Context,
    factory *engine.RuntimeFactory,
    logger zerolog.Logger,
    config Config,
) (*App, error)

type App struct {
    // private lifecycle state, child context, cancel, owner ID,
    // service, store, and close-once/result synchronization
}

// Close rejects new operations, unloads every live runtime, releases leases,
// cancels app-owned resources, and leaves durable session rows intact.
func (a *App) Close(ctx context.Context) error

// UnloadSession removes and closes one live runtime without soft deletion.
// Persistent history remains restorable.
func (a *App) UnloadSession(ctx context.Context, sessionID string) error

// RecoverSession discards a degraded/fenced live runtime and restores the
// last valid durable journal under a fresh lease.
func (a *App) RecoverSession(ctx context.Context, sessionID string) (*replsession.SessionSummary, error)

// DeleteSession retains current product semantics: close + soft delete.
func (a *App) DeleteSession(ctx context.Context, sessionID string) error
```

`Close` should be idempotent and return the same aggregated result to concurrent callers. Use `errors.Join` so one failing runtime closer does not prevent the remaining sessions from closing.

### `replsession` service lifecycle

```go
type ServiceOption func(*Service)

// The service derives one child context per created/restored session.
func WithLifetimeContext(ctx context.Context) ServiceOption

func NewService(
    factory *engine.RuntimeFactory,
    logger zerolog.Logger,
    opts ...ServiceOption,
) *Service

func (s *Service) UnloadSession(ctx context.Context, sessionID string) error
func (s *Service) Close(ctx context.Context) error

func (s *Service) WithRuntime(
    ctx context.Context,
    sessionID string,
    fn func(context.Context, *engine.Runtime) error,
) error
```

The context passed to `CreateSession` remains the startup context. `WithLifetimeContext` becomes the lifetime parent used in `RuntimeFactory.NewRuntime`.

### Error contracts

Introduce sentinel or typed errors that callers and HTTP status mapping can inspect without matching strings:

```go
var (
    ErrAppClosing          = errors.New("replapi: app is closing")
    ErrAppClosed           = errors.New("replapi: app is closed")
    ErrSessionClosing      = errors.New("replsession: session is closing")
    ErrSessionDegraded     = errors.New("replsession: persistence is degraded")
    ErrSessionFenced       = errors.New("replsession: session ownership was lost")
    ErrSessionOwned        = errors.New("repldb: session has another active owner")
    ErrUnsupportedVersion  = errors.New("replhttp: unsupported schema version")
    ErrRequestTooLarge     = errors.New("replhttp: request body too large")
)
```

Typed errors should carry safe metadata such as session ID, active owner expiry, expected/current cell, or supported schema version. They must not expose database paths, SQL text, stack traces, or plugin internals to remote clients.

## Context-Aware Operation Gate

Replacing `sync.Mutex` with a context-aware gate solves only queue cancellation unless lifecycle state is integrated. Implement the gate together with session state and session cancellation.

A weighted semaphore with capacity one is already available through `golang.org/x/sync/semaphore`. The exact primitive is less important than the contract.

```go
type sessionState struct {
    gate *semaphore.Weighted // capacity 1

    lifecycleMu sync.Mutex
    phase       sessionPhase
    health      sessionHealth

    ctx    context.Context
    cancel context.CancelCauseFunc

    runtime *engine.Runtime
    // existing cells, bindings, policy, counters, lease, pending record...
}
```

Operation acquisition pseudocode:

```text
beginOperation(callerCtx):
    if callerCtx is nil:
        callerCtx = Background

    acquire gate with callerCtx
    if acquisition fails:
        return callerCtx error

    lock lifecycleMu
    if phase is not active:
        unlock lifecycleMu
        release gate
        return closing/closed error
    if health is degraded or fenced and operation is Evaluate:
        unlock lifecycleMu
        release gate
        return health error
    unlock lifecycleMu

    opCtx = context canceled when either callerCtx or sessionCtx is canceled
    return operation token containing opCtx and release function
```

Shutdown pseudocode:

```text
stopSession(shutdownCtx, disposition):
    lock lifecycleMu
    if already closed:
        return previous close result
    if active:
        phase = closing
        cancel sessionCtx with ErrSessionClosing
    unlock lifecycleMu

    acquire gate with shutdownCtx
      # active evaluation sees session cancellation and should unwind
    if acquisition fails:
        return shutdown timeout; retain a reachable closing entry for retry

    close engine.Runtime with shutdownCtx
    release persistent lease if disposition != delete
    soft-delete durable session if disposition == delete

    mark closed
    release gate
```

Do not remove the map entry so early that a failed close leaves an unreachable runtime. Either keep a closing tombstone until close completes or move the state into a dedicated closing registry owned by `Service.Close`.

### Cancellation inside evaluation

Evaluation currently derives its timeout only from the caller context. Compose it with the session context so unload/app close can interrupt it:

```text
operationCtx = merge(callerCtx, sessionCtx)
evaluationCtx = apply policy timeout to operationCtx
run JavaScript using evaluationCtx
```

Use `context.AfterFunc` or a small helper to avoid leaking a goroutine for every operation. Preserve the existing `goja.Runtime.Interrupt` behavior and post-timeout session usability tests.

## Durable Ownership: Lease and Fencing Design

### Why a lease is required

A unique cell constraint detects duplicate append after execution. It does not prevent two live VMs. A process-local lock cannot coordinate CLI invocations or separate servers. A per-session SQLite lease is the smallest coordination primitive that matches the current deployment model.

### Schema v2

After a real migration framework exists, add:

```sql
CREATE TABLE session_leases (
  session_id   TEXT PRIMARY KEY,
  owner_id     TEXT NOT NULL,
  epoch        INTEGER NOT NULL,
  lease_until  TEXT NOT NULL,
  updated_at   TEXT NOT NULL,
  FOREIGN KEY(session_id) REFERENCES sessions(session_id)
);

CREATE INDEX idx_session_leases_until
  ON session_leases(lease_until);
```

Definitions:

- `owner_id`: random ID created once per `replapi.App` instance.
- `epoch`: monotonically increasing fencing number; every takeover increments it.
- `lease_until`: expiry used to recover after process death.
- `WriteFence`: `(session_id, owner_id, epoch, expected_cell_id)` attached to every append.

### Lease API

```go
type SessionLease struct {
    SessionID  string
    OwnerID    string
    Epoch      int64
    LeaseUntil time.Time
}

type WriteFence struct {
    OwnerID       string
    Epoch         int64
    ExpectedCellID int
}

func (s *Store) AcquireSessionLease(
    ctx context.Context,
    sessionID string,
    ownerID string,
    ttl time.Duration,
) (SessionLease, error)

func (s *Store) RenewSessionLease(
    ctx context.Context,
    lease SessionLease,
    ttl time.Duration,
) (SessionLease, error)

func (s *Store) ReleaseSessionLease(
    ctx context.Context,
    lease SessionLease,
) error

func (s *Store) PersistEvaluationFenced(
    ctx context.Context,
    record EvaluationRecord,
    fence WriteFence,
) error
```

### Acquire transaction

```text
BEGIN IMMEDIATE
load active session
load lease row

if no lease:
    insert owner_id, epoch=1, lease_until=now+ttl
    COMMIT success

if lease.owner_id == this owner:
    renew lease_until
    COMMIT success

if lease.lease_until <= now:
    update owner_id=this owner,
           epoch=old epoch+1,
           lease_until=now+ttl
    COMMIT success

otherwise:
    ROLLBACK
    return ErrSessionOwned(active owner and expiry)
```

### Fenced append transaction

```text
BEGIN IMMEDIATE
read lease
require owner_id == fence.ownerID
require epoch == fence.epoch
require lease_until > now
require MAX(evaluations.cell_id)+1 == fence.expectedCellID
insert evaluation and children
update session timestamp
COMMIT
```

The epoch prevents an old process from writing after its expired lease is taken over. The expected cell check prevents journal gaps and stale heads.

### Lease lifetime and renewal

The app should acquire a lease before restoring source or publishing a new persistent session. It should renew before evaluation and during long replay. Choose a TTL comfortably above the maximum normal evaluation timeout, then renew at a fraction such as one-third of TTL.

Use a clock interface in tests:

```go
type Clock interface {
    Now() time.Time
}
```

Do not write sleep-heavy lease tests.

A lease cannot undo external side effects from an owner that continues executing after losing connectivity or pausing beyond expiry. Fencing protects SQLite consistency. Document that external systems require their own idempotency keys or fencing strategy.

## Persistence Failure State

### Minimum safe state machine

A persistent evaluation runs JavaScript before all reports can be written. If `PersistEvaluationFenced` fails, the service cannot make the VM match the durable journal by rollback. It must stop accepting new evaluations.

```text
active
  |
  | JavaScript completed, durable append failed
  v
degraded
  |
  +-- RetryPendingPersistence succeeds --> active
  |
  +-- RecoverSession --> unload runtime, restore durable head --> active
  |
  +-- DeleteSession --> closed/deleted
```

Store the exact pending `EvaluationRecord` and write fence in memory before returning the error. This enables an explicit retry for transient errors without recomputing observations or rerunning JavaScript.

```go
type persistenceFailure struct {
    Cause   error
    Record  repldb.EvaluationRecord
    Fence   repldb.WriteFence
    At      time.Time
}
```

### Evaluation pseudocode

```text
Evaluate(sessionID, source):
    op = begin context-aware session operation
    reject unless phase=active and health=healthy

    if persistent:
        renew/verify lease
        verify durable head == state.nextCellID

    compute candidateCellID = nextCellID + 1
    analyze, rewrite, execute, observe
    build complete cell and persistence record

    if persistent:
        err = persist fenced record
        if err:
            append cell to in-memory diagnostics if useful
            mark health degraded or fenced
            retain exact pending record
            return infrastructure error

    publish nextCellID and cell as committed
    return response
```

Ideally, delay publishing `nextCellID` and the committed cell list until persistence succeeds. Runtime mutations still exist, but the state health prevents later use. The response may include a cell report together with a typed commit error so operators can diagnose what executed. Decide this public shape before implementation; returning only `nil, error` hides useful execution evidence.

### Recovery choices

- **Retry pending write:** safe only with the exact original record and same valid fence.
- **Discard live runtime and restore durable head:** restores journal consistency but cannot undo external side effects.
- **Force continue:** reject this option. It recreates the current gap bug.

## SQLite Migration Design

The lease table and future lifecycle metadata require real migrations.

### Required behavior

1. Detect an empty database and apply schema v1 then later migrations.
2. Read the stored version before changing it.
3. Apply ordered migrations in one transaction per migration or one encompassing transaction when SQLite permits.
4. Update the version only after migration statements succeed.
5. Reject a database whose version is newer than the binary supports.
6. Preserve existing v1 data.
7. Make migration tests use real v1 fixtures.

### API sketch

```go
type migration struct {
    Version int
    Name    string
    Up      []string
}

var migrations = []migration{
    {Version: 1, Name: "initial repl schema", Up: schemaV1},
    {Version: 2, Name: "session ownership leases", Up: schemaV2Lease},
}

func (s *Store) migrate(ctx context.Context) error
```

Pseudocode:

```text
ensure repldb_meta table exists
read schema_version; absent means 0
if version > CurrentSchemaVersion:
    fail with ErrDatabaseTooNew

for each migration where migration.version > version:
    BEGIN IMMEDIATE
    execute migration.Up statements
    update schema_version to migration.version
    COMMIT
```

Do not use `INSERT OR REPLACE` to announce the latest version before verifying the shape.

## Configuration Repair

### Make profile resolution authoritative

Unknown profiles must fail validation:

```go
func ValidateProfile(profile Profile) error {
    switch profile {
    case ProfileRaw, ProfileInteractive, ProfilePersistent:
        return nil
    default:
        return fmt.Errorf("%w: %q", ErrUnknownProfile, profile)
    }
}
```

For an explicit `Config.Profile` with zero `SessionOptions`, apply the full profile preset. Do not merely copy the string into zero options.

```text
normalize config:
    if profile empty:
        use DefaultConfig
    validate profile

    if SessionOptions is zero:
        preset, err = ConfigForProfile(profile)
        if err != nil: return err
        SessionOptions = preset.SessionOptions
    else:
        validate SessionOptions.Profile if set
        require profile/policy relationship to be intentional
```

A cleaner longer-term option is to remove duplicate profile fields and store only normalized `SessionOptions`. For this ticket, preserve the public shape but reject disagreement rather than silently mixing it.

### Policy replacement semantics

`WithDefaultSessionPolicy` and `SessionOverrides.Policy` currently replace the complete policy. Keep that behavior unless a separate patch introduces an explicit merge API. Document it and add tests so zero booleans do not look like “unspecified.”

## HTTP Hardening Design

### Handler options

Keep authentication composable, but make parsing and resource behavior safe by default:

```go
type HandlerConfig struct {
    MaxRequestBodyBytes int64 // default 1 MiB
    MaxSourceBytes      int   // default 256 KiB
    ExposeInternalErrors bool // false by default
}

type HandlerOption func(*HandlerConfig)

func NewHandler(app *replapi.App, opts ...HandlerOption) (http.Handler, error)
```

The handler should:

1. require `Content-Type: application/json` for evaluate requests;
2. wrap the body with `http.MaxBytesReader` before reading;
3. reject oversized input with HTTP 413;
4. reject unsupported `schemaVersion` with HTTP 400;
5. reject unknown fields through existing strict `protojson` behavior;
6. enforce `MaxSourceBytes` after decoding;
7. map typed domain errors to stable status and error codes;
8. log internal details server-side and return a generic message by default;
9. attach `X-Content-Type-Options: nosniff` consistently, including errors;
10. add request IDs or accept one from an outer middleware.

### Protobuf error contract

Add a public error message to `proto/goja/replapi/v1/replapi.proto`:

```proto
message ErrorResponse {
  uint32 schema_version = 1;
  string code = 2;
  string message = 3;
  string request_id = 4;
}
```

Suggested mappings:

| Domain error | HTTP | Code |
|---|---:|---|
| invalid protobuf JSON/profile/version | 400 | `invalid_argument` |
| request/source too large | 413 | `request_too_large` |
| session not found/deleted | 404 | `session_not_found` |
| session owned elsewhere | 409 | `session_owned` |
| session degraded/fenced | 409 | `session_not_writable` |
| app closing | 503 | `service_shutting_down` |
| internal persistence/runtime failure | 500 | `internal` |

JavaScript parse/runtime errors remain successful transport responses with a non-`ok` cell execution status. That distinction is already part of the DTO model and should be documented in tests.

### Server defaults

Harden `cmd/goja-repl serve`:

```go
srv := &http.Server{
    Addr:              settings.Addr,
    Handler:           securedHandler,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       60 * time.Second,
    MaxHeaderBytes:    1 << 20,
}
```

The exact write timeout must exceed the maximum configured evaluation timeout plus response serialization margin.

Retain loopback as the default bind address. If the user selects a non-loopback address, either require an explicit `--allow-remote` acknowledgement or emit a prominent error unless an authentication integration is configured. Also encourage `--safe-mode` or an explicit module allowlist; default module registration includes host-access capabilities.

### Authentication boundary

Do not hard-code one identity system into `pkg/replhttp`. Instead, expose a normal `http.Handler` so applications can wrap it:

```text
authentication middleware
    -> authorization/session ownership middleware
        -> request limits and audit middleware
            -> replhttp handler
```

Document that `replhttp.NewHandler` is an execution transport, not a secure multi-tenant service.

## CLI and TUI Lifecycle Integration

### CLI commands

`commandSupport.runWithApp` currently closes only the store. After `App.Close` exists, every command must close the app before the store:

```go
app, store, err := s.newApp(ctx)
if err != nil { return err }
defer store.Close()
defer app.Close(context.Background())
```

Use a bounded shutdown context rather than an unbounded background context in production code. The ordering is important: runtimes and lease operations may still need the database during app close.

### TUI

The Bobatea adapter explicitly does not own the app (`pkg/repl/adapters/bobatea/replapi.go:168-172`). Keep that contract. The TUI command owns and closes the app after the event bus and Bubble Tea program stop.

`WithRuntime` callback signature changes will require updating completion/help assistance integration. The callback must use the operation context passed by the service.

### HTTP server

Shutdown order:

```text
receive SIGINT/SIGTERM
stop accepting HTTP requests with Server.Shutdown
wait for in-flight handlers
App.Close with bounded context
Store.Close
return process status
```

If `App.Close` reports errors, log all of them and return a non-zero command error after still attempting store close.

## Decision Records

### Decision: app owns runtime lifetime

- **Context:** Create and restore operation contexts previously became runtime lifetime parents.
- **Options considered:** use `context.Background`; fix only HTTP; add a service context option; make app construction take a parent context.
- **Decision:** `replapi.New`/`NewWithConfig` accept a parent context and pass the app child context to `replsession.Service` as the lifetime parent. Create/restore contexts control startup work only.
- **Rationale:** ownership is explicit, all transports behave consistently, and app close has one cancellation root.
- **Consequences:** constructor call sites migrated; direct `replsession` users can use `WithLifetimeContext`; no compatibility shim obscures the contract.
- **Status:** accepted in P2

### Decision: unload is separate from delete

- **Context:** persistent runtimes need cleanup without making durable history disappear.
- **Options considered:** overload `DeleteSession` with flags; use delete then clear `deleted_at`; add `UnloadSession`.
- **Decision:** expose non-destructive `UnloadSession`; retain delete as close plus soft delete. Unload is allowed for in-memory sessions and intentionally discards their only state.
- **Rationale:** the names encode product intent and avoid boolean disposition flags.
- **Consequences:** callers must choose deliberately; documentation warns that in-memory unload is destructive.
- **Status:** accepted in P2

### Decision: context-aware gate plus lifecycle states

- **Context:** `sync.Mutex` ignored canceled waiters and delete could race active operations.
- **Options considered:** check context before/after mutex; channel worker per session; weighted semaphore; capacity-one channel gate with lifecycle state.
- **Decision:** use a context-aware capacity-one channel gate plus explicit phases and a session cancellation context.
- **Rationale:** it minimally changes the synchronous service while enforcing queue cancellation and shutdown ordering without another dependency.
- **Consequences:** acquisition and release are centralized in operation tokens; bounded shutdown leaves a reachable closing state for retry; race tests are mandatory.
- **Status:** accepted in P2

### Decision: fail closed after durable append failure

- **Context:** JavaScript mutation cannot be rolled back after SQLite failure.
- **Options considered:** continue and allow gaps; rerun source; retry exact record; unload immediately; mark degraded.
- **Decision:** retain the exact pending record, mark the session degraded, reject later evaluation, and support retry or discard-and-restore recovery. `Evaluate` returns both the executed cell response and typed `CommitError`.
- **Rationale:** it prevents silent divergence while preserving diagnostic evidence and avoids rerunning source during commit retry.
- **Consequences:** external side effects may already have happened; successful retry publishes the pending cell, while recovery discards the suspect VM and restores only durable source.
- **Status:** accepted in P3

### Decision: SQLite lease with fencing epoch

- **Context:** two apps could own independent VMs for one durable session, while unrelated sessions should remain concurrent.
- **Options considered:** documentation only; process-local mutex; database-wide exclusive lock; optimistic cell check; per-session lease and fence.
- **Decision:** support concurrent processes on distinct sessions and enforce one writable owner per session with a schema-v2 SQLite lease, random per-app owner ID, expiry, monotonic epoch, heartbeat, and fenced append.
- **Rationale:** it supports sequential CLI processes, deterministic expired takeover, stale-owner rejection, and session-granular concurrency without a database-wide bottleneck.
- **Consequences:** external effects still need idempotency/fencing; owner IDs cannot be caller-configured; app close/unload/delete release leases before store close.
- **Status:** accepted in P5

### Decision: migrations precede lease schema

- **Context:** the old bootstrap could not safely evolve v1 databases.
- **Options considered:** add another `CREATE TABLE IF NOT EXISTS`; reset databases; implement migrations.
- **Decision:** use ordered per-version transactions, record the version only after statements succeed, and reject newer databases. P4 retained schema v1; P5 subsequently added per-session ownership as migration v2.
- **Rationale:** production data must not be silently mislabeled as upgraded, and the ownership version was consumed only when ownership landed.
- **Consequences:** the real v1 fixture and migration rollback/concurrent-open tests are permanent; upgrades are forward-only and require operator backups.
- **Status:** accepted in P4

### Decision: transport hardening without bundled authentication

- **Context:** the handler executes powerful JavaScript but reusable libraries cannot assume one identity provider.
- **Options considered:** leave all safety to callers; embed auth; provide bounded parsing and middleware seams.
- **Decision:** enforce body/source/version/error safety in `replhttp`, harden the CLI server, and keep authentication/authorization composable outside the package.
- **Rationale:** parsing safety belongs to the transport; identity policy belongs to the host.
- **Consequences:** `replhttp` remains normal middleware-compatible `http.Handler`; the CLI defaults to loopback and requires `--allow-remote` acknowledgement for any non-loopback bind, but that acknowledgement does not provide authentication.
- **Status:** accepted in P6

### Decision: use a breaking constructor migration, not a shim

- **Context:** lifecycle ownership cannot be inferred reliably from existing `New` calls.
- **Options considered:** `NewWithContext` plus old `New`; option with background default; change `New` directly.
- **Decision:** change the constructors to require the parent context and update repository callers atomically.
- **Rationale:** compilation failures identify every ownership decision. A compatibility wrapper would preserve unsafe omission.
- **Consequences:** downstream users receive a deliberate API migration documented in the Glazed guide; every repository caller now supplies an intentional lifetime parent.
- **Status:** accepted in P2 and documented in P7

## Phased Implementation Plan

### How to use this plan

The phase plan is a dependency graph and tracking contract, not a suggestion to put every change into one pull request. The authoritative checkbox IDs live in `tasks.md`; this section explains why each task exists, what may run in parallel, and what evidence closes the phase.

Rules for implementation:

1. Do not check a `P*.GATE` task until every non-optional task in that phase is checked and its validation commands pass.
2. Record the commit or pull request for each completed task in the diary/changelog.
3. Keep behavior-changing phases separate from generated-code and documentation-only commits where practical.
4. If implementation evidence changes a proposed decision, update the decision record before changing later-phase tasks.
5. Do not start Phase 2 behavior changes before Phase 0 captures the current failures.
6. Phases 4–5 are conditional on the persistent deployment contract described below; skipping them requires an explicit documented single-owner alternative, not silence.

### Phase dependency and necessity matrix

| Phase | Classification | Depends on | Deliverable |
|---|---|---|---|
| 0 — Regression safety net | Mandatory | Research complete | Deterministic failing tests for every reproduced defect |
| 1 — Configuration correctness | Mandatory | Phase 0 | Validated profile/policy/store contract |
| 2 — Runtime lifecycle and cancellation | Mandatory | Phase 1 | App-owned lifetime, close/unload/delete semantics, cancelable queue |
| 3 — Fail-closed persistence | Mandatory for persistent profile | Phase 2 | No later cell after uncommitted mutation; retry/recover path |
| 4 — SQLite migrations | Conditional prerequisite | Phase 3 + schema evolution decision | Safe v1-to-vNext migration framework |
| 5 — Ownership and fencing | Conditional on multi-process persistence | Phase 4 for lease design | One writable owner per durable session or explicit exclusive DB owner |
| 6 — HTTP/protobuf hardening | Baseline mandatory if `serve` remains | Phase 2; some mappings need 3/5 | Bounded requests, safe errors, hardened server |
| 7 — Host/release integration | Mandatory for release | Every phase selected for release | Correct shutdown ordering, docs, generated outputs, full validation |

### Milestone A: mandatory correctness core

Milestone A comprises Phases 0–3. It is required for the in-process TUI, embedded applications, and persistent applications. A local-only deployment does not make premature context cancellation, uncancelable queue waits, or persistence gaps acceptable.

#### Phase 0: regression safety net

**Status:** Complete on 2026-07-15 (`P0.GATE` checked). Eight desired-behavior tests are isolated behind the `replapi_hardening` build tag; the exact red baseline and promotion workflow are documented in `reference/02-phase-0-red-test-baseline-and-execution-guide.md`.

**Goal:** Convert every ticket probe into a deterministic package-level test before changing behavior.

**Entry criteria:** The five standalone probes reproduce on the current baseline. Existing focused and race tests pass.

**Tracked work:**

- **P0.1:** Add a real `httptest.Server` test that creates through HTTP, waits for `ServeHTTP` completion, and proves the runtime lifetime context is canceled today. A direct `ResponseRecorder` test is insufficient.
- **P0.2:** Add a barrier-based canceled-waiter test. Hold the session operation with `WithRuntime`, cancel a queued `Snapshot` or evaluation, and assert it does not execute after release.
- **P0.3:** Add table tests for `Config{Profile: ProfileRaw}`, all valid presets, and unknown app/session profiles.
- **P0.4:** Add a persistence stub that fails cell 2, permits cell 3, and proves the current durable sequence becomes `[1,3]`.
- **P0.5:** Add two apps over one SQLite store, restore the same session into both, and prove the second cell mutates both VMs before one unique insert fails.
- **P0.6:** Characterize evaluate-versus-delete behavior with channels/barriers rather than sleeps. Record the desired behavior separately from the current result.
- **P0.7:** Mark expected failures narrowly and show unrelated REPL tests remain green. Do not disable whole packages.
- **P0.8:** Run and record the focused race baseline.

**Likely files:**

- `pkg/replhttp/proto_handler_test.go`
- `pkg/replapi/app_test.go`
- `pkg/replapi/config_test.go`
- `pkg/replsession/service_persistence_test.go`
- new `lifecycle_test.go` or `concurrency_test.go` files where clearer

**Validation:**

```bash
go test ./pkg/replapi ./pkg/replsession ./pkg/repldb ./pkg/replhttp ./pkg/repl/adapters/bobatea

go test -race ./pkg/replapi ./pkg/replsession ./pkg/repldb ./pkg/replhttp ./pkg/repl/adapters/bobatea
```

**Gate P0.GATE:** Every reproduced defect has a deterministic regression test that fails for the intended assertion, not from timeout flakiness or unrelated setup errors.

**Recommended PR boundary:** Test-only pull request or first commit in Phase 1, with expected failures clearly isolated so the default branch remains buildable.

#### Phase 1: configuration correctness

**Status:** Complete on 2026-07-15 (`P1.GATE` checked). Three P0 profile regressions were promoted into normal CI; five non-configuration hardening regressions remain red behind `replapi_hardening`.

**Goal:** Make the selected profile an enforceable policy preset rather than a label that can contradict execution behavior.

**Entry criteria:** P0.GATE checked. Config regressions demonstrate the current mismatch.

**Tracked work:**

- **P1.1:** Add typed validation for unknown app and per-session profile values.
- **P1.2:** When `Config.Profile` is explicit and `SessionOptions` are zero, resolve the complete matching preset, including timeout and observation flags.
- **P1.3:** Reject contradictory nonzero `Config.Profile`, `SessionOptions.Profile`, and policy combinations unless an explicitly documented override API permits them.
- **P1.4:** Preserve the existing full-policy replacement behavior for `WithDefaultSessionPolicy` and `SessionOverrides.Policy`; test that zero booleans mean disabled, not unspecified.
- **P1.5:** Keep persistent-profile store and auto-restore validation strict.
- **P1.6:** Reuse the shared validator in TUI parsing where this removes duplicate accepted-value logic without coupling UI errors to internals.
- **P1.7:** Update API comments and profile tables in tests/docs touched by the change.

**Likely files:**

- `pkg/replapi/config.go`
- `pkg/replapi/config_test.go`
- `cmd/goja-repl/tui.go`
- `cmd/goja-repl/root_test.go`

**Validation:**

```bash
go test ./pkg/replapi ./cmd/goja-repl
```

**Gate P1.GATE:** For every supported constructor and session override, profile name, eval mode, timeout, observation, persistence, and store requirements are consistent. Unknown values return a typed error.

**Recommended PR boundary:** One focused config/API correctness PR.

#### Phase 2: runtime lifecycle and cancellation

**Status:** Complete on 2026-07-15 (`P2.GATE` checked). Runtime lifetime is app-owned, queued operations are cancelable, and close/unload/delete share a tested lifecycle gate. Three P0 lifecycle regressions were promoted; two non-P2 regressions remain tagged red.

**Goal:** Give every runtime an explicit owner and deterministic shutdown path while making queued operations honor cancellation.

**Entry criteria:** P1.GATE checked. The lifecycle API shape and breaking constructor decision are accepted in review.

**Tracked work:**

- **P2.1:** Define app/session lifecycle phases and typed closing/closed errors.
- **P2.2:** Change `replapi.New` and `NewWithConfig` to accept a parent context and migrate direct callers so ownership decisions are compile-visible.
- **P2.3:** Add a service lifetime context and derive a child context for each created/restored session.
- **P2.4:** Pass caller context only as runtime startup context; pass the app/session context as runtime lifetime.
- **P2.5:** Replace blocking session mutex acquisition with a context-aware capacity-one operation gate.
- **P2.6:** Compose caller cancellation, session cancellation, and evaluation timeout without one goroutine per operation leaking.
- **P2.7:** Pass the operation context into `WithRuntime` callbacks and document non-escape/non-reentrancy.
- **P2.8:** Implement non-destructive unload in service and app.
- **P2.9:** Implement idempotent `Service.Close` and `App.Close`, closing all runtimes and aggregating errors.
- **P2.10:** Serialize delete, unload, and close with active operations; preserve a reachable retryable closing state if shutdown times out.
- **P2.11:** Test context lifetime, canceled queue wait, repeated/concurrent close, unload versus delete, active evaluation shutdown, and close hooks exactly once.

**Likely files:**

- `pkg/replapi/app.go`
- `pkg/replapi/config.go`
- `pkg/replsession/service.go`
- `pkg/replsession/evaluate.go`
- `pkg/repl/adapters/bobatea/replapi.go`
- `pkg/engine/runtime.go` only if a narrower interrupt primitive is necessary

**Implementation order inside the phase:**

```text
lifecycle enums/errors
  -> parent/session contexts
  -> operation gate
  -> operation context composition
  -> unload
  -> close-all
  -> delete integration
  -> adapter/caller migration
  -> concurrency/race tests
```

**Validation:**

```bash
go test ./pkg/replapi ./pkg/replsession ./pkg/replhttp ./pkg/repl/adapters/bobatea

go test -race ./pkg/replapi ./pkg/replsession ./pkg/replhttp ./pkg/repl/adapters/bobatea
```

**Gate P2.GATE:** HTTP request completion does not cancel runtime-owned resources; canceled queued calls do not execute later; every runtime is unloadable/closable; delete remains soft-delete; repeated close is safe.

**Recommended PR boundary:** This may require two reviewable PRs: context/gate first, then close/unload/delete. Do not expose `App.Close` until service shutdown semantics are tested.

#### Phase 3: fail-closed persistence and recovery

**Status:** Complete on 2026-07-15 (`P3.GATE` checked). Executed-but-uncommitted cells return a response plus typed commit error, block later JavaScript, and support exact retry or discard-and-restore recovery. The P0 persistence regression is promoted; only the Phase 5 ownership regression remains tagged red.

**Goal:** Prevent an executed but uncommitted cell from being followed by later source in the same live VM.

**Entry criteria:** P2.GATE checked. Reviewers agree on the public shape for an executed-but-uncommitted result.

**Tracked work:**

- **P3.1:** Define healthy, degraded, and fenced health states with typed errors.
- **P3.2:** Build and retain the exact `EvaluationRecord` before attempting durable commit.
- **P3.3:** Delay publishing the committed cell ID/history until append success where the current report pipeline permits.
- **P3.4:** Mark the live session degraded on any post-execution append failure.
- **P3.5:** Reject all later JavaScript before analysis/execution while degraded or fenced.
- **P3.6:** Retry the exact pending record without rerunning JavaScript.
- **P3.7:** Recover by unloading the suspect VM and restoring the last durable head.
- **P3.8:** Define whether `Evaluate` returns both cell report and typed commit error, or exposes diagnostics through another method; update transport mapping accordingly.
- **P3.9:** Inject failures at evaluation insert, child rows, commit, retry, and recovery; assert contiguous IDs and no later execution.

**Likely files:**

- `pkg/replsession/service.go`
- `pkg/replsession/evaluate.go`
- `pkg/replsession/persistence.go`
- `pkg/replsession/types.go`
- `pkg/replapi/app.go`
- protobuf/adapters only if health or commit results become public in this phase

**Validation:**

```bash
go test ./pkg/replsession ./pkg/replapi ./pkg/repldb
```

Required property:

```text
if durable commit for cell N fails,
then no cell N+1 executes until N is committed exactly or the VM is discarded.
```

**Gate P3.GATE:** Failure injection cannot produce durable gaps such as `[1,3]`, and recovery reconstructs the last committed journal without claiming external side effects were rolled back.

**Recommended PR boundary:** One persistence state-machine PR; keep lease/fencing out until Phase 5.

### Milestone B: conditional persistent multi-process ownership

Milestone B comprises Phases 4–5. It is required if two OS processes may open the same persistent database or if a server and CLI may touch the same session concurrently. It may be skipped only when the product explicitly enforces a narrower ownership model, such as one exclusive process for the entire database. “Users probably will not do that” is not an ownership mechanism.

#### Decision gate before Phase 4

Record one of these supported contracts in the design/changelog:

| Contract | Required mechanism | Trade-off |
|---|---|---|
| In-memory only | No SQLite ownership | No cross-process persistence |
| One process per database | Database-wide OS/SQLite ownership lock | Simple, but blocks all other CLI/server access |
| Concurrent processes, distinct sessions | Per-session lease/fencing | More schema and lifecycle complexity; best concurrency |
| Distributed service beyond shared SQLite | External coordinator plus fencing | Out of scope for this ticket |

**P5.1 decision (accepted 2026-07-15):** support concurrent processes on distinct sessions and enforce one writable owner per persistent session with SQLite lease/fencing. The database-wide exclusive-owner alternative is rejected because it serializes unrelated sessions. Phases 4 and 5 therefore apply as written.

#### Phase 4: transactional SQLite migrations

**Status:** Complete on 2026-07-15 (`P4.GATE` checked). Schema v1 is represented by an ordered transactional migration, empty and concurrent opens share that path, real v1 data is preserved, failed hypothetical upgrades roll back, and newer versions are rejected. P5 later consumed schema v2 for per-session ownership leases.

**Goal:** Replace version stamping with a migration system capable of safely adding ownership schema.

**Entry criteria:** P3.GATE checked; durable schema evolution selected; backup/compatibility expectations reviewed.

**Tracked work:**

- **P4.1:** Commit a real v1 SQLite fixture containing representative session and evaluation data.
- **P4.2:** Define ordered migration descriptors and read the stored version before modifying schema.
- **P4.3:** Bootstrap an empty database through the same migration path.
- **P4.4:** Apply migration statements transactionally and update version only after success.
- **P4.5:** Reject a database newer than the binary supports.
- **P4.6:** Test data-preserving v1 upgrade, failed migration rollback, and reopen idempotence.
- **P4.7:** Test concurrent opens during migration.
- **P4.8:** Document backup, upgrade, downgrade/non-downgrade, and failure recovery expectations.

**Likely files:**

- `pkg/repldb/store.go`
- `pkg/repldb/schema.go`
- new `pkg/repldb/migrations.go`
- `pkg/repldb/store_test.go`
- `pkg/repldb/testdata/repl-v1.sqlite`

**Validation:**

```bash
go test ./pkg/repldb -count=1
```

**Gate P4.GATE:** A real v1 database upgrades without data loss, a failed migration leaves the old version intact, and a newer database is rejected instead of relabeled.

**Recommended PR boundary:** Migration-only PR before any lease table is introduced.

#### Phase 5: persistent ownership and fencing

**Status:** Complete on 2026-07-15 (`P5.GATE` checked). The supported contract is concurrent processes owning distinct sessions with one SQLite lease/fencing epoch per writable session. Schema v2, heartbeat renewal, stale-owner fencing, fenced append, release, recovery, and deterministic takeover tests are complete; all P0 regressions now run green in normal CI.

**Goal:** Ensure one durable session cannot have two writable live VM owners under the selected deployment contract.

**Entry criteria:** P4.GATE checked for lease design, or P5.1 explicitly selects and specifies a database-wide exclusive-owner alternative.

**Tracked work:**

- **P5.1:** Decide and document the supported ownership contract. Adjust P5.2–P5.10 only through an explicit design/task update.
- **P5.2:** For leases, add schema-v2 `session_leases` migration and indexes.
- **P5.3:** Give every app a process-unique owner ID and inject a clock for tests.
- **P5.4:** Acquire atomically for absent, same-owner, expired, and conflicting-owner rows.
- **P5.5:** Renew, release, expire, and increment epoch on takeover.
- **P5.6:** Fence append by owner, epoch, expiry, and expected next cell in one transaction.
- **P5.7:** Acquire before persistent create/restore publication and renew during long replay/evaluation.
- **P5.8:** Mark stale owners fenced before later JavaScript; discard or recover their VMs.
- **P5.9:** Release ownership during unload/app close before store close while retaining delete semantics.
- **P5.10:** Test fake-clock takeover, stale epochs, simultaneous owners, expected-cell conflict, crash expiry, and sequential CLI use.

**Likely files:**

- `pkg/repldb/migrations.go`
- new `pkg/repldb/lease.go`
- `pkg/repldb/write.go`
- `pkg/replapi/app.go`
- `pkg/replsession/service.go`
- `pkg/replsession/persistence.go`

**Validation:**

```bash
go test ./pkg/repldb ./pkg/replapi ./pkg/replsession -count=1

go test -race ./pkg/repldb ./pkg/replapi ./pkg/replsession
```

**Gate P5.GATE:** A second active owner is rejected before JavaScript mutation, stale epochs cannot append, takeover after expiry is deterministic, and orderly close releases ownership.

**Recommended PR boundary:** Store lease primitives first, then service/app integration.

### Milestone C: transport hardening and release integration

Milestone C contains Phases 6–7. If `goja-repl serve` remains a supported command, request/body limits, redacted errors, complete server timeouts, and remote-bind safety are mandatory. A generated protobuf `ErrorResponse` is recommended as part of the same coherent public transport change; if split out, the temporary error contract must still be explicit and tested.

#### Phase 6: HTTP and protobuf hardening

**Goal:** Bound resource use, make version/error behavior stable, and make unsafe remote deployment deliberate.

**Entry criteria:** P2.GATE checked. Typed degraded/ownership status mappings require P3/P5 respectively; independent body-limit work may begin earlier on a separate branch.

**Status:** Complete on 2026-07-15 (`P6.GATE` checked). Evaluate parsing is bounded to 1 MiB bodies and 256 KiB source by default, exact schema/content rules are enforced, all domain and infrastructure failures use generated redacted `ErrorResponse` payloads with request IDs, server timeouts/header limits are complete, and non-loopback CLI binding requires `--allow-remote` acknowledgement. Real HTTP cancellation, ownership, redaction, header, limit, and compatibility tests plus Go/TypeScript generated-binding checks pass.

**Tracked work:**

- **P6.1:** Add nonzero default maximum body and source sizes.
- **P6.2:** Apply `http.MaxBytesReader` before any full-body allocation and return 413.
- **P6.3:** Validate JSON content type, decoded source size, and exact schema version.
- **P6.4:** Add protobuf `ErrorResponse` with version, stable code, safe message, and request ID.
- **P6.5:** Map typed domain errors to documented HTTP statuses/codes.
- **P6.6:** Redact internal SQL, filesystem, panic, runtime, and plugin details while logging diagnostics server-side.
- **P6.7:** Regenerate Go/TypeScript bindings and update conversion/golden fixtures.
- **P6.8:** Add read, write, idle, header, and maximum-header settings compatible with evaluation timeouts.
- **P6.9:** Require explicit acknowledgement or a strong warning for non-loopback binding; document module allowlist/safe-mode expectations.
- **P6.10:** Add real HTTP tests for limits, cancellation, version, status, error redaction, request IDs, and headers.

**Likely files:**

- `proto/goja/replapi/v1/replapi.proto`
- `pkg/replapi/pbconv`
- `pkg/replhttp/proto_handler.go`
- `pkg/replhttp/handler.go`
- `cmd/goja-repl/cmd_serve.go`
- generated Go and `web/packages/replapi-types` files/tests

**Validation:**

```bash
go test ./pkg/replhttp ./pkg/replapi ./cmd/goja-repl
buf lint
buf generate
pnpm replapi-types:typecheck
pnpm replapi-types:test
```

**Gate P6.GATE:** Oversized requests fail before allocation, unsupported versions have a stable response, internal details are hidden, server timeouts are complete, and documentation does not imply authentication is built in.

**Recommended PR boundary:** Body/server safety may land first; protobuf error schema and generated clients should be one atomic transport PR.

#### Phase 7: host integration, documentation, and release validation

**Goal:** Apply the selected lifecycle/ownership/transport changes consistently to every host and publish an accurate migration path.

**Entry criteria:** Every phase selected for this release has its gate checked.

**Status:** Complete on 2026-07-15 (`P7.GATE` checked). Every constructor caller supplies an explicit lifetime parent; CLI, TUI, and server hosts aggregate operation/shutdown failures while closing app before store; Bobatea assistance observes the operation context; public API comments and migration guidance reflect the breaking contracts; and the complete Go, race, vet, Glazed, Buf, TypeScript, docmgr, and zero-red validation matrix passes. The final design/diary bundle was refreshed on reMarkable.

**Tracked work:**

- **P7.1:** Update every constructor call site with an intentional parent context.
- **P7.2:** Close app before store in CLI helpers using a bounded shutdown context.
- **P7.3:** Close the TUI-owned app while keeping the Bobatea adapter explicitly non-owning.
- **P7.4:** Enforce server shutdown order: stop HTTP, wait handlers, close app/release ownership, close store.
- **P7.5:** Migrate `WithRuntime` assistance and other callbacks to the context-bearing contract.
- **P7.6:** Update Glazed help, usage help, examples, and API comments.
- **P7.7:** Add constructor/lifecycle migration notes and remove unsafe examples rather than adding indefinite shims.
- **P7.8:** Run Go formatting, full tests, focused race, vet, and Glazed lint.
- **P7.9:** Run Buf generation and TypeScript checks when protobuf changed.
- **P7.10:** Update diary/changelog/relations, run doctor, and refresh the reMarkable bundle.

**Likely files:**

- `cmd/goja-repl/root.go` and command files
- `cmd/goja-repl/tui.go`
- `cmd/goja-repl/cmd_serve.go`
- `pkg/repl/adapters/bobatea/replapi.go`
- `pkg/doc/34-replapi-guide.md`
- `pkg/doc/04-repl-usage.md`
- fuzz harnesses, examples, generated bindings, and release notes

**Validation:**

```bash
gofmt -w <changed-go-files>
go test ./...
go test -race ./pkg/replapi ./pkg/replsession ./pkg/repldb ./pkg/replhttp ./pkg/repl/adapters/bobatea
go vet ./...
glazed-lint ./cmd/goja-repl/...
docmgr doctor --ticket GOJA-068 --stale-after 30
```

Run Buf and pnpm commands from Phase 6 when transport files changed.

**Gate P7.GATE:** Every host closes in the documented order, all tests/generation/docs pass, no stale constructor or help example remains, and ticket/reMarkable delivery matches the implementation.

### Progress reporting format

At the end of each implementation session, report progress with stable IDs rather than percentages guessed from prose:

```text
Phase: P2 — Runtime lifecycle and cancellation
Completed: P2.1, P2.2, P2.3
In progress: P2.4
Blocked: P2.5 pending operation-gate API review
Gate: P2.GATE open
Validation: go test ./pkg/replapi ./pkg/replsession (pass)
```

The phase gate is the release signal. A phase with 9 of 10 subtasks checked is still incomplete if its invariant is not demonstrated.

## Testing and Validation Strategy

### Unit tests

Cover pure config, lifecycle transitions, lease decisions, migration version selection, error mapping, and size/version validation.

Use table-driven tests for:

- profile/policy combinations;
- app/session lifecycle transitions;
- HTTP error-to-status/code mapping;
- lease row states: absent, same owner, other unexpired owner, expired owner, stale epoch;
- migration source and target versions.

### Integration tests

Use real SQLite and real runtimes for contracts that cross package boundaries:

1. create persistent session, evaluate, unload, restore, continue;
2. close app, construct new app, restore session;
3. two apps contend for one session;
4. lease expires with fake clock, second app takes over, first is fenced;
5. injected persistence error degrades session and blocks later source;
6. retry pending persistence succeeds without rerunning JavaScript;
7. app close during pending promise and synchronous runaway code;
8. TUI adapter close does not own app but command close does.

### Real HTTP tests

Use `httptest.NewServer`, not only direct `ServeHTTP`, for request-lifetime assertions. Test:

- runtime context remains live after response;
- request cancellation interrupts evaluation only;
- body one byte above limit returns 413;
- unknown field and wrong version return 400 protobuf error;
- missing session returns 404;
- session owned/degraded returns 409;
- internal injected store error returns generic 500 while logger receives details;
- response headers are correct.

### Concurrency and race tests

Run:

```bash
go test -race ./pkg/replapi ./pkg/replsession ./pkg/repldb ./pkg/replhttp ./pkg/repl/adapters/bobatea
```

Add deterministic barriers rather than sleeps for:

- evaluate versus unload;
- evaluate versus app close;
- two simultaneous restore calls in one app;
- two app lease acquisition;
- canceled waiter behind `WithRuntime`;
- concurrent repeated `Close`.

### Persistence failure injection

Keep a stub persistence implementation that can fail:

- session create;
- lease acquire/renew/release;
- evaluation append before transaction;
- child row insertion;
- commit;
- soft delete.

For every failure, assert:

- runtime close behavior;
- session phase/health;
- whether a retry is legal;
- no later cell executes when journal consistency is unknown.

### Migration tests

Check a committed v1 fixture into `pkg/repldb/testdata`. Avoid building the “old” database with current schema helpers, because that can mask migration omissions.

Tests should inspect both schema and data after migration, then open the upgraded database a second time to prove idempotence.

### Fuzz and property tests

Existing fuzz infrastructure can add properties:

```text
persisted cell IDs are contiguous from 1..N
one durable session has at most one valid lease epoch owner
stale fence never writes
Close is idempotent
no operation starts after closing transition
unknown profile never normalizes successfully
```

### Full validation commands

```bash
gofmt -w <changed-go-files>
go test ./...
go test -race ./pkg/replapi ./pkg/replsession ./pkg/repldb ./pkg/replhttp ./pkg/repl/adapters/bobatea
go vet ./...
buf lint
buf generate
pnpm replapi-types:typecheck
pnpm replapi-types:test
glazed-lint ./cmd/goja-repl/...
```

## Code Review Guide for an Intern

Review in this order:

1. `pkg/replsession/service.go`: identify who owns the map, session state, gate, context, runtime, and close result.
2. `pkg/replsession/evaluate.go`: verify no JavaScript runs before health/lease checks and no later cell runs after commit failure.
3. `pkg/replapi/app.go`: verify app state rejects new work during close and close preserves durable history.
4. `pkg/repldb/migrations.go` and lease code: verify transaction predicates and epoch checks.
5. `pkg/replhttp`: verify limits happen before allocation and internal errors are not returned.
6. `cmd/goja-repl`: verify shutdown order is HTTP, app, store.
7. Tests: look for deterministic coordination rather than timing-only assertions.

Questions to ask during review:

- Can a state pointer escape the session map immediately before close?
- Can a canceled caller acquire the gate later?
- Can any code execute after `degraded` or `fenced`?
- Can an old lease epoch append?
- Can app close return while a runtime remains reachable but unmanaged?
- Can store close happen before lease release or runtime closers finish?
- Can a remote error expose SQL, paths, plugin messages, or panic details?
- Does a test prove behavior with a real HTTP request lifecycle?

## Risks and Mitigations

### Risk: lifecycle refactor introduces deadlocks

Mitigation:

- centralize gate acquisition in one helper;
- document lock order (`app state` -> `service map` only briefly; never hold map lock while waiting on a session gate);
- use context-aware acquisition;
- add close/evaluate race tests under `-race`.

### Risk: close hangs on callbacks that ignore context

Mitigation:

- pass an operation context into `WithRuntime` callbacks;
- define bounded close behavior;
- retain a retryable closing entry if shutdown times out;
- avoid silently dropping the runtime pointer.

### Risk: lease expiry during long evaluation

Mitigation:

- renew before evaluation;
- TTL exceeds policy timeout with margin;
- use fencing on commit;
- mark stale owner fenced and discard its VM;
- document limits for external side effects.

### Risk: migration damages existing databases

Mitigation:

- real v1 fixture;
- transactional migrations;
- backup guidance for operators;
- reject newer versions;
- test failed statement rollback.

### Risk: API break surprises downstream users

Mitigation:

- compile-time constructor change;
- migration example in help/release notes;
- update repository callers in the same release;
- do not retain an unsafe context-free wrapper indefinitely.

### Risk: HTTP defaults break large legitimate cells

Mitigation:

- configurable limits;
- actionable 413 error with documented defaults;
- choose initial limits from observed payload sizes;
- separate source limit from response/report size considerations.

## Alternatives Considered

### Fix only `POST /api/sessions`

Rejected because restore and auto-restore also create runtimes from operation contexts. The ownership bug is in `replsession.Service`, not only the route.

### Use `context.Background()` for runtime lifetime

Rejected because it avoids premature cancellation but creates unowned resources and does not provide deterministic close.

### Treat SQLite unique constraints as ownership

Rejected because the losing VM has already executed JavaScript. Detection after mutation is too late.

### Use one lock for the entire database

Rejected because unrelated sessions should progress independently and CLI/server deployments need session-level ownership.

### Continue after persistence errors and repair later

Rejected because journal gaps make replay numbering and semantics ambiguous. Fail closed immediately.

### Physically delete sessions on cleanup

Rejected because cleanup and product deletion are different intentions. Durable history must survive app close and unload.

### Add authentication directly to `pkg/replhttp`

Rejected because the repository already supports multiple host/auth environments. The transport should be safely bounded and middleware-friendly rather than tied to one identity system.

## Open Questions

1. **Resolved in P2:** `UnloadSession` is allowed for raw/interactive sessions and intentionally destroys their only state; the public API and help topic warn callers.
2. Should a commit failure return both a non-nil cell report and a typed error, or should the report be retrievable through a separate diagnostics method?
3. **Resolved in P5:** default TTL is 30 seconds and heartbeat renewal runs at one-third TTL during evaluation, `WithRuntime`, and replay; hosts may configure TTL explicitly.
4. **Resolved in P5:** owner IDs remain diagnostic through the Go `App.OwnerID()` API and typed local errors; session summaries and current HTTP payloads do not expose lease rows.
5. Should `RecoverSession` automatically retry a pending exact record before discarding the runtime?
6. Should `schemaVersion: 0` remain temporarily accepted, or should v1 become strict immediately?
7. What default body/source limits fit current real REPL workloads and static report response sizes?
8. Should non-loopback `goja-repl serve` fail without `--allow-remote`, or warn loudly?
9. **Resolved in P2:** app close cancels session operation contexts immediately, then waits for the capacity-one gate within the caller's shutdown deadline.
10. Should persistent replay gain an explicit “replay-safe initialization” convention or metadata marker in a later ticket?

## References

### Primary implementation files

- `pkg/replapi/app.go` — application facade, auto-restore, and current missing lifecycle boundary.
- `pkg/replapi/config.go` — profiles, partial config normalization, and policy/store validation.
- `pkg/replsession/service.go` — live session map, runtime creation, restore, delete, and service locks.
- `pkg/replsession/evaluate.go` — complete evaluation pipeline and persist-after-mutation ordering.
- `pkg/replsession/persistence.go` — conversion from cell reports to durable records.
- `pkg/replsession/policy.go` — raw, interactive, and persistent policy presets.
- `pkg/repldb/store.go` — SQLite open/bootstrap and current schema-version handling.
- `pkg/repldb/schema.go` and `pkg/repldb/migrations.go` — v1 journal tables, schema-v2 leases, and ordered migration contract.
- `pkg/repldb/write.go` — session soft delete and transactional evaluation writes.
- `pkg/repldb/read.go` — soft-delete filtering and raw-source replay journal.
- `pkg/replhttp/proto_handler.go` — protobuf-JSON routes and unbounded request read.
- `pkg/replhttp/handler.go` — current error/status/recovery behavior.
- `cmd/goja-repl/cmd_serve.go` — current server timeouts and shutdown sequence.
- `cmd/goja-repl/root.go` — app/store/factory construction and module policy.
- `cmd/goja-repl/tui.go` — interactive/persistent TUI ownership.
- `pkg/repl/adapters/bobatea/replapi.go` — runtime callback adapter and non-owning close contract.
- `pkg/engine/options.go` — startup versus lifetime context definitions.
- `pkg/engine/runtime.go` — runtime context, close hooks, interruption, and shutdown.
- `pkg/runtimeowner/runner.go` — serialized VM calls and context-aware owner queue.
- `proto/goja/replapi/v1/replapi.proto` — public transport schema.

### Existing tests

- `pkg/replapi/app_test.go`
- `pkg/replapi/config_test.go`
- `pkg/replsession/service_persistence_test.go`
- `pkg/replsession/service_policy_test.go`
- `pkg/replsession/service_edge_test.go`
- `pkg/repldb/store_test.go`
- `pkg/replhttp/proto_handler_test.go`
- `cmd/goja-repl/root_test.go`

### Ticket evidence

- `scripts/01-probe-http-session-context/main.go`
- `scripts/02-probe-persistent-split-brain/main.go`
- `scripts/03-probe-partial-profile-config/main.go`
- `scripts/04-probe-post-execution-persistence-failure/main.go`
- `scripts/05-probe-canceled-waiter/main.go`

### Prior design context

- `ttmp/2026/04/03/GOJA-20-WEBREPL-ARCHITECTURE--web-repl-architecture-analysis-and-third-party-integration-guide/design-doc/02-cli-and-server-first-persistent-repl-architecture-and-implementation-guide.md`
- `ttmp/2026/04/03/GOJA-23-CONFIGURABLE-REPLAPI--configurable-replapi-profiles-and-policies/design-doc/01-configurable-replapi-profiles-and-policies-implementation-plan.md`
- `ttmp/2026/04/07/GOJA-040-PERSISTENCE-CORRECTNESS--fix-repl-persistence-correctness-and-sqlite-integrity/design-doc/01-persistence-correctness-analysis-design-and-implementation-guide.md`
- `ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/design-doc/01-evaluation-control-analysis-design-and-implementation-guide.md`
- `ttmp/2026/07/01/GOJA-067--protobuf-schema-for-replapi-payloads-and-typescript-generation/design-doc/01-protobuf-replapi-schema-and-typescript-generation-implementation-guide.md`

### Audit baseline

The source audit and probes were run against commit `cc9f18656f02f42e945806bcb6e3b1d86c0658ad` on 2026-07-15. The working tree also contains the separately requested REPL API help-page improvements in `pkg/doc/34-replapi-guide.md`, `pkg/doc/04-repl-usage.md`, and `cmd/goja-repl/root_test.go`.
