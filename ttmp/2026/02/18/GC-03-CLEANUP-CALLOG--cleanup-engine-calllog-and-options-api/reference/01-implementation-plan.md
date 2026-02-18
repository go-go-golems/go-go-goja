---
Title: Implementation Plan
Ticket: GC-03-CLEANUP-CALLOG
Status: active
Topics:
    - go
    - refactor
    - tooling
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-perf/serve_command.go
      Note: Graceful shutdown plan
    - Path: engine/runtime.go
      Note: Target API and runtime init changes
    - Path: pkg/calllog/calllog.go
      Note: Runtime-scoped logger design
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-18T10:03:38.652626444-05:00
WhatFor: ""
WhenToUse: ""
---


# Implementation Plan

## Goal

Replace split/awkward engine startup APIs with one option-driven entrypoint,
make calllog configuration scoped to an engine instance (instead of process
global toggling during each runtime startup), and fix `goja-perf serve` so
Ctrl-C terminates immediately.

## Context

- Current engine constructors are split between:
  - `engine.NewWithOptions(...require.Option)` for require registry behavior
  - `engine.NewWithConfig(RuntimeConfig, ...require.Option)` for calllog +
    require.
- Calllog currently uses package-global default logger state, and
  runtime creation can enable/disable it globally.
- User requirement: require options should be part of engine open options, and
  calllog should be another option scoped to that engine.

## Quick Reference

### Target API shape

```go
// Primary constructor
func Open(opts ...Option) (*goja.Runtime, *require.RequireModule)

type Option func(*openSettings)

func WithRequireOptions(opts ...require.Option) Option
func WithCallLog(path string) Option
func WithCallLogDisabled() Option
```

### Compatibility approach

- Keep existing constructors (`New`, `NewWithOptions`, `NewWithConfig`) as
  wrappers to reduce immediate breakage.
- Internals route through `Open(...)`.
- `RuntimeConfig` remains for wrapper compatibility while new code uses options.

### Calllog scoping approach

- Add runtime-scoped logger binding in `pkg/calllog` (runtime -> logger map).
- `WrapGoFunction` and `CallJSFunction` log through runtime logger first, then
  fallback to package default logger only when runtime binding is absent.
- Engine open attaches logger for that runtime when `WithCallLog(path)` is set.
- Engine open no longer globally disables/enables calllog as side effect.

### Serve Ctrl-C fix

- `serve` command must listen for context cancellation and call
  `http.Server.Shutdown(...)`.
- Treat `http.ErrServerClosed` as normal shutdown path.

## Usage Examples

```go
vm, req := engine.Open(
  engine.WithRequireOptions(require.WithLoader(loader)),
  engine.WithCallLog("/tmp/goja-calllog.sqlite"),
)
_ = vm
_ = req
```

```go
vm, req := engine.Open(
  engine.WithRequireOptions(require.WithLoader(loader)),
  engine.WithCallLogDisabled(),
)
_ = vm
_ = req
```

## Related

- `ttmp/2026/02/18/GC-03-CLEANUP-CALLOG--cleanup-engine-calllog-and-options-api/tasks.md`
- `ttmp/2026/02/18/GC-03-CLEANUP-CALLOG--cleanup-engine-calllog-and-options-api/reference/02-diary.md`
