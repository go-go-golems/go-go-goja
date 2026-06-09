---
Title: HTTP Serve Hot Reload Implementation Guide
Ticket: XGOJA-EXTERNAL-HTTP-HOST
Status: active
Topics:
    - xgoja
    - goja
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/pkg/xgoja/app/factory.go
      Note: Runtime factory needs per-runtime host service injection
    - Path: go-go-goja/pkg/xgoja/hotreload/manager.go
      Note: Reusable blue/green reload manager to embed in serve --hot-reload
    - Path: go-go-goja/pkg/xgoja/providerapi/commands.go
      Note: Command provider runtime factory interface will be extended
    - Path: go-go-goja/pkg/xgoja/providers/http/serve.go
      Note: Serve command provider that will gain opt-in hot reload flags and execution branch
    - Path: go-go-goja/pkg/xgoja/providers/http/serve_test.go
      Note: Focused serve command provider tests
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# HTTP Serve Hot Reload Implementation Guide

## Executive summary

The external-host work made hot reload possible for embedding applications, but generated xgoja binaries still run `serve` commands as a single long-lived runtime. This document designs the next step: an opt-in `--hot-reload` mode for the built-in HTTP `serve` command provider.

In hot-reload mode, `xgoja serve ...` should own the public TCP listener once, create a fresh `gojahttp.Host` and xgoja runtime for each reload attempt, run the selected JavaScript serve verb against that candidate, optionally smoke-test a configured path such as `/api/widget/health`, and atomically swap only successful candidates into service. Broken reloads must keep the previous active runtime alive.

## Problem statement

Today a generated binary can serve a JavaScript site:

```bash
./dist/minitrace-viz serve site start --http-listen 127.0.0.1:8787
```

That path creates one runtime, invokes the selected serve verb once, and then waits for shutdown. When JavaScript files change, the user must restart the process. The new `pkg/xgoja/hotreload` manager proves the desired blue/green behavior, but it is currently only usable from a Go embedding host.

Generated binary users should be able to opt into the same behavior without writing a Go host:

```bash
./dist/minitrace-viz serve site start \
  --http-listen 127.0.0.1:8787 \
  --hot-reload \
  --hot-reload-watch-root . \
  --hot-reload-smoke-path /api/widget/health
```

## Current-state architecture

`pkg/xgoja/providers/http/serve.go` builds commands from configured JSVerb sources. Each command invokes `serveVerb`, which currently:

1. creates a runtime through `commandCtx.RuntimeFactory.NewRuntimeFromSections(...)`;
2. initializes selected modules from parsed Glazed values;
3. invokes the selected JSVerb in that runtime;
4. prints that the runtime is alive;
5. waits for `SIGINT` / `SIGTERM`.

The HTTP provider module now supports external hosts via `ExternalHostService{Host, OwnsListen}`. However, the current `providerapi.RuntimeFactory` interface cannot add per-runtime host services at serve-command time. The factory captures a base service bag when the generated app starts.

`pkg/xgoja/hotreload` provides the reusable runtime-swap behavior but deliberately delegates candidate runtime construction to a caller-provided `LoadFunc`. The `serve` command can become such a caller once it can inject a candidate `*gojahttp.Host` into each runtime creation.

## Proposed solution

### User-facing flags

Add a serve-specific Glazed section to each generated HTTP serve command:

| Field | Type | Default | Meaning |
|---|---|---|---|
| `hot-reload` | bool | `false` | Enable blue/green hot reload for this serve invocation. |
| `hot-reload-watch-root` | string list | empty | Files/directories to poll for changes. Empty defaults to JSVerb source roots where possible. |
| `hot-reload-watch-ext` | string list | `js,json,md,yaml,yml` | File extensions that trigger reload. |
| `hot-reload-smoke-path` | string | empty | Optional path to GET on the candidate host before swap. |
| `hot-reload-poll` | string | `500ms` | Poll interval parsed with `time.ParseDuration`. |
| `hot-reload-debounce` | string | `250ms` | Debounce delay parsed with `time.ParseDuration`. |
| `hot-reload-close-grace` | string | `2s` | Delay before closing a retired runtime. |
| `hot-reload-status-path` | string | `/__xgoja/status` | Go-owned status endpoint path. Empty disables status endpoint. |

Names intentionally use the `hot-reload-` prefix because command users see flattened Cobra flags.

### Runtime factory extension

Add an optional interface in `providerapi`:

```go
type RuntimeFactoryWithHostServices interface {
    RuntimeFactory
    NewRuntimeFromSectionsWithHostServices(
        ctx context.Context,
        vals *values.Values,
        hostServices HostServices,
        opts ...require.Option,
    ) (*engine.Runtime, error)
}
```

Implement it on `app.RuntimeFactory` by creating a runtime with a service collector seeded from the factory's base services plus the per-runtime services. Existing `RuntimeFactory` callers stay unchanged.

### Serve execution branch

`serveVerb` decodes the new hot-reload settings. If disabled, it keeps the current behavior.

If enabled:

1. Resolve `--http-listen` from the existing HTTP section.
2. Build a `hotreload.Manager` with a `LoadFunc` that:
   - creates `app.HostServices` containing `ExternalHostService{Host: candidate.Host, OwnsListen: false}`;
   - creates a runtime through `NewRuntimeFromSectionsWithHostServices`;
   - initializes selected modules from parsed values;
   - invokes the selected serve verb against the candidate runtime.
3. Configure `SmokeFunc` when `--hot-reload-smoke-path` is set.
4. Run an initial `manager.Reload(ctx)` before binding the public listener.
5. Start one Go-owned `net/http.Server` with handler routes:
   - optional `hot-reload-status-path` handled by Go;
   - all other paths delegated to `manager`.
6. Start `manager.Watch` if watch roots can be resolved or were supplied.
7. Wait for shutdown, then close the manager and HTTP server.

The HTTP provider's own listener must not bind in this path because each candidate runtime receives `OwnsListen: false`.

## Design decisions

### Decision: add hot reload to `serve`, not `run`

`serve` already means “invoke a JavaScript verb that registers an HTTP site and keep it alive.” That is the natural command for hot-reloadable sites. `run` can stay a direct script runner.

### Decision: keep hot reload opt-in

Default `serve` behavior must remain unchanged to avoid surprising production users. Hot reload is primarily a development workflow.

### Decision: extend runtime factory with per-runtime host services

The generated binary path cannot call generated-package `NewBundle`. Per-runtime host services are the missing abstraction that lets command providers reuse the same selected module set while adding a candidate external HTTP host.

### Decision: candidate smoke test uses HTTP path

A smoke path validates the behavior users care about: did the candidate register routes and can it answer a representative request? A path such as `/api/widget/health` is better than only checking that the JavaScript verb returned.

## Detailed implementation tasks

### Phase 1: Design and planning

- [x] Add this design document.
- [x] Add a task checklist to the ticket.
- [ ] Relate the new design document to implementation files.
- [ ] Commit planning docs.

### Phase 2: Runtime factory per-runtime host services

- [ ] Add `RuntimeFactoryWithHostServices` to `pkg/xgoja/providerapi/commands.go`.
- [ ] Add `NewRuntimeFromSectionsWithHostServices` to `pkg/xgoja/app/factory.go`.
- [ ] Keep existing `NewRuntime` and `NewRuntimeFromSections` behavior unchanged.
- [ ] Add tests proving a command-time service is visible to provider `ModuleSetupContext.Host`.
- [ ] Run focused app/providerapi tests.
- [ ] Update diary and commit.

### Phase 3: Serve hot-reload command flags

- [ ] Add serve hot-reload settings and a serve-specific Glazed section in `pkg/xgoja/providers/http/serve.go`.
- [ ] Decode settings from parsed values.
- [ ] Add unit tests proving generated serve commands expose the new flags.
- [ ] Run focused HTTP provider tests.
- [ ] Update diary and commit.

### Phase 4: Serve hot-reload execution path

- [ ] Add a hot-reload branch in `serveVerb` while preserving the current non-hot path.
- [ ] Inject `ExternalHostService{Host: candidate.Host, OwnsListen: false}` per candidate runtime.
- [ ] Start a Go-owned HTTP server around `hotreload.Manager` using `--http-listen`.
- [ ] Implement optional status endpoint and optional smoke path.
- [ ] Wire watcher roots/extensions/poll/debounce/close grace.
- [ ] Add tests for reload success, broken reload last-known-good, status, and smoke failure.
- [ ] Run focused HTTP provider and hotreload tests.
- [ ] Update diary and commit.

### Phase 5: Generated binary integration test

- [ ] Add or update a generated-binary integration test that runs `serve ... --hot-reload`.
- [ ] Verify an HTTP health endpoint responds.
- [ ] Modify a watched JS source and verify `__xgoja/status.activeVersion` increments.
- [ ] Introduce a broken JS edit and verify the previous runtime still serves.
- [ ] Run generator integration tests.
- [ ] Update diary and commit.

### Phase 6: Documentation and final validation

- [ ] Update `cmd/xgoja/doc` user docs for `serve --hot-reload`.
- [ ] Update examples or tutorial notes where appropriate.
- [ ] Run focused tests plus `go test ./...`.
- [ ] Run `docmgr doctor`.
- [ ] Mark ticket tasks complete.
- [ ] Update diary and commit final docs.

## Test strategy

Focused commands during implementation:

```bash
go test ./pkg/xgoja/app ./pkg/xgoja/providerapi -count=1
go test ./pkg/xgoja/providers/http ./pkg/xgoja/hotreload ./pkg/gojahttp -count=1
go test ./cmd/xgoja/internal/generate -count=1
```

Before PR:

```bash
go test ./...
docmgr --root ttmp doctor --ticket XGOJA-EXTERNAL-HTTP-HOST --stale-after 30
```

## Risks and open questions

- The per-runtime service extension must not duplicate provider-contributed services incorrectly.
- Watch roots derived from JSVerb sources need careful behavior for provider-shipped or embedded sources; those may not be watchable and should require explicit roots or skip watching.
- `--hot-reload-status-path` must not shadow JavaScript application routes unless the user opts into that path.
- Long-running old requests rely on runtime close grace, but exact in-flight behavior depends on `runtimeowner` waiting/interrupt semantics.
