---
Title: Runtime Ergonomics Polish Implementation Guide
Ticket: XGOJA-RUNTIME-POLISH
Status: active
Topics:
    - xgoja
    - goja
    - modules
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/modules/express/express.go
      Note: |-
        Express module API where app/route registration/listen behavior should be exposed.
        Express app API and route registration
    - Path: go-go-goja/modules/fs/backend_embed.go
      Note: |-
        Embedded read-only backend and EROFS mutation behavior.
        Read-only embedded filesystem backend
    - Path: go-go-goja/modules/fs/fs.go
      Note: |-
        Filesystem module exports and TypeScript declaration surface.
        Filesystem JavaScript export surface
    - Path: go-go-goja/pkg/xgoja/app/root.go
      Note: |-
        Generated runtime root commands, including provider module catalog command.
        Generated runtime commands
    - Path: go-go-goja/pkg/xgoja/app/runtime_spec.go
      Note: |-
        Embedded runtime selected module spec used for selected-module inventory.
        Selected runtime module spec
    - Path: go-go-goja/pkg/xgoja/providers/http/http.go
      Note: |-
        HTTP provider currently owns express host lifecycle and listener startup.
        HTTP provider listener lifecycle
ExternalSources: []
Summary: 'Design guide for three go-go-goja runtime ergonomics fixes: lazy Express listener startup, read-only fs:assets discovery, and structured selected-module inventory.'
LastUpdated: 2026-06-09T00:20:00-04:00
WhatFor: Use this to implement small go-go-goja runtime polish items after JSVerb filtering.
WhenToUse: Read before changing the HTTP provider, Express module, fs module, or generated runtime module inventory commands.
---


# Runtime Ergonomics Polish Implementation Guide

## Executive summary

Yes: these issues belong in the `go-go-goja` package/repository. They are framework/runtime ergonomics issues surfaced by ClubMedMeetup, but the underlying behavior lives in reusable xgoja providers and generated runtime commands.

This ticket tracks three small, separable fixes:

1. `require("express")` should not eagerly bind an HTTP listener just because the module was imported.
2. Embedded asset filesystem aliases such as `fs:assets` should advertise that they are read-only.
3. Generated runtime module inventory should clearly distinguish compiled provider modules from selected runtime aliases, and the selected inventory should be a structured Glazed command with normal `--output json` support.

## Scope

In scope:

- `go-go-goja/pkg/xgoja/providers/http/http.go`
- `go-go-goja/modules/express/express.go`
- `go-go-goja/modules/fs/*`
- `go-go-goja/pkg/xgoja/app/root.go`
- `go-go-goja/pkg/xgoja/app/runtime_spec.go`
- tests and docs for those behaviors

Out of scope:

- ClubMedMeetup application-specific route changes.
- go-minitrace provider consolidation.
- goja-text document helpers.
- Any broad module architecture rewrite.

## Problem 1: Express eagerly binds on require

Current behavior is surprising for introspection. Requiring the Express module can start or touch the HTTP listener before a script registers routes or explicitly asks to listen. A user running an eval such as:

```bash
./dist/app eval 'Object.keys(require("express"))'
```

should not fail because `127.0.0.1:8787` is already occupied.

### Proposed behavior

- `require("express")` creates exports only; no network bind.
- `express.app()` creates an app object only; no network bind by itself.
- First route/static registration may autostart the listener when HTTP is enabled, preserving existing `run server.js --keep-alive` behavior.
- `app.listen([addr])` should be considered as an explicit start path if it does not already exist.
- `--http-enabled=false` should allow route registration for tests/introspection without binding.

### Implementation sketch

Keep dependency direction clean: `modules/express` should not import `pkg/xgoja/providers/http`. Instead, add an optional callback option to the express registrar:

```go
type StartFunc func(*goja.Runtime) error

func WithOnUse(fn StartFunc) Option { ... }
```

Call the hook from route/static registration and `listen`, not from module load. The HTTP provider passes `c.start(vm, entry)` as the hook.

### Tests

- Requiring `express` alone does not bind the port.
- Registering a route in normal mode still starts serving.
- Registering a route with HTTP disabled succeeds and does not bind.
- Existing generated app run behavior still works.

## Problem 2: fs:assets should advertise read-only behavior

The embedded filesystem backend already rejects writes with read-only errors. The issue is discoverability: `Object.keys(require("fs:assets"))` lists write methods, and users have no obvious programmatic way to know the backend is read-only.

### Proposed API

Expose stable metadata on every fs module:

```js
const fs = require("fs:assets")
fs.isReadOnly // true
fs.capabilities()
// { backend: "embedded", read: true, write: false, embedded: true, mounts: [...] }
```

For host fs:

```js
require("fs:host").capabilities()
// { backend: "host", read: true, write: true, embedded: false }
```

The first implementation can expose booleans and backend kind before detailed mount metadata if that keeps the patch small.

### Tests

- Embedded backend reports `isReadOnly === true` and `capabilities().write === false`.
- Host backend reports `isReadOnly === false` and `capabilities().write === true`.
- Existing EROFS mutation behavior remains unchanged.

## Problem 3: runtime module inventory is confusing

Generated binaries currently have a `modules` command that lists compiled provider modules. That is useful, but it is not the same as selected runtime aliases. Users want to know whether `require("fs:assets")` or `require("fs")` is valid in this generated runtime.

The fix should use Glazed command machinery so users automatically get structured output modes such as `--output json`.

### Proposed command shape

Add a selected inventory command, for example:

```bash
./dist/app selected-modules --output json
```

Rows:

- `package`: provider package ID
- `module`: provider module name
- `alias`: actual CommonJS `require()` alias
- `provider_ref`: `<package>.<module>`
- `config`: static config summary or JSON string

Keep the existing `modules` command as a provider catalog, but rename/clarify columns so it does not imply its provider refs are valid `require()` names.

### Tests

- A runtime spec with two `fs` module instances emits two rows with distinct aliases.
- The command is implemented as a Glazed command and supports `--output json` through normal middleware.
- Existing `modules` command still works as provider catalog.

## Implementation order

1. Add selected-module inventory command first. It is additive and low risk.
2. Add fs read-only metadata. It is local to `modules/fs` and docs/tests.
3. Change Express lifecycle last. It has the highest compatibility risk and needs careful tests.

## Implemented outcome

The ticket has been implemented in three focused code commits after the branch rebase:

1. `dc1b74f` — `Add selected xgoja module inventory`
   - Adds the structured `selected-modules` Glazed command.
   - Clarifies that `modules` is a provider catalog by using `provider_ref` for provider identities.
2. `ef2fb81` — `Expose fs backend capabilities`
   - Adds backend capability metadata and JavaScript exports `fs.isReadOnly` and `fs.capabilities()`.
   - Reports embedded asset backends as read-only while preserving existing EROFS mutation failures.
3. `f16430e` — `Defer Express HTTP listener binding`
   - Makes `require("express")` side-effect-light.
   - Defers HTTP listener startup until route/static registration or explicit `app.listen()`.

The documentation wrap-up is recorded separately so reviewers can inspect code behavior and ticket bookkeeping independently.

## Validation

Run focused tests after each phase, then broader package tests:

```bash
cd /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja
go test ./pkg/xgoja/app ./pkg/xgoja/providers/http ./modules/express ./modules/fs -count=1
go test ./pkg/jsverbs ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1
```

Before PR, run the normal pre-commit hooks or `go test ./...` as appropriate.
