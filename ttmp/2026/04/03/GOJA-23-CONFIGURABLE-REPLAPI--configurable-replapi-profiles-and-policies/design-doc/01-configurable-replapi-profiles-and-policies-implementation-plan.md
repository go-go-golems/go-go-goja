---
Title: Configurable replapi profiles and policies implementation plan
Ticket: GOJA-23-CONFIGURABLE-REPLAPI
Status: active
Topics:
    - persistent-repl
    - architecture
    - repl
    - refactor
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Detailed implementation plan for refactoring replapi into a profile-based API spanning raw, interactive, and persistent session behavior.
LastUpdated: 2026-04-03T19:40:11.336412253-04:00
WhatFor: Use this plan to implement a configurable replapi surface that can be used as raw Goja, as an interactive REPL kernel, or as the fully persistent restore-aware REPL stack.
WhenToUse: Use when changing replapi constructors, replsession evaluation flow, or CLI/TUI adoption of profile-based REPL behavior.
---

# Configurable replapi profiles and policies implementation plan

## Executive Summary

`replapi` currently hard-codes one product behavior: restore-aware persistent sessions backed by SQLite and the full `replsession` analysis/rewrite/persistence path. That is a good default for the new CLI/server, but it is too rigid for other consumers. The refactor in this ticket introduces a profile-based API with explicit policies for evaluation, observation, persistence, and restore.

The key design choice is to keep the API opinionated while avoiding "mystery meat" behavior. Most callers should choose a profile such as `raw`, `interactive`, or `persistent`. Advanced callers can then override specific sub-policies like auto-restore or JSDoc extraction. Under the hood, `replsession` becomes policy-driven rather than unconditionally instrumented.

## Problem Statement

The current `replapi` constructor in `pkg/replapi/app.go` requires a SQLite store and always enables the full persistent behavior. That means:

- Callers that want an in-memory REPL still pay the mental cost of the persistence-oriented API shape.
- The current evaluation path in `pkg/replsession/service.go` always performs analysis, rewrite, runtime diffing, and persistence-oriented bookkeeping even when the caller only wants execution.
- `cmd/goja-repl` and the traditional interactive REPLs do not have a clean way to declare their intended behavior through one shared API.

This causes two concrete risks:

- feature overreach, where every consumer is forced into the heaviest behavior;
- architecture drift, where some consumers bypass `replapi` entirely because it is too opinionated in the wrong way.

## Proposed Solution

Introduce explicit app/session configuration with two levels:

- app defaults, chosen at construction time;
- per-session overrides, chosen when creating a session.

The API should center on named profiles:

- `raw`
- `interactive`
- `persistent`

Those profiles expand into a small number of policy groups:

- execution policy
- observation policy
- persistence policy
- restore policy

### Proposed Types

Pseudocode:

```go
type Profile string

const (
    ProfileRaw         Profile = "raw"
    ProfileInteractive Profile = "interactive"
    ProfilePersistent  Profile = "persistent"
)

type Config struct {
    Profile Profile

    Store   *repldb.Store
    Restore RestorePolicy
    Eval    EvalPolicy
    Observe ObservePolicy
    Persist PersistPolicy
}

type SessionOptions struct {
    Profile  *Profile
    Restore  *RestorePolicy
    Eval     *EvalPolicy
    Observe  *ObservePolicy
    Persist  *PersistPolicy
    ID       string
    CreateAt time.Time
}
```

### Policy Breakdown

Execution policy decides whether the cell runs through the current instrumented rewrite path or through a simpler direct path:

```go
type EvalMode string

const (
    EvalModeRaw          EvalMode = "raw"
    EvalModeInstrumented EvalMode = "instrumented"
)

type EvalPolicy struct {
    Mode                 EvalMode
    CaptureLastExpression bool
    SupportTopLevelAwait  bool
}
```

Observation policy governs non-durable introspection:

```go
type ObservePolicy struct {
    StaticAnalysis    bool
    RuntimeSnapshots  bool
    BindingTracking   bool
    ConsoleCapture    bool
    JSDocExtraction   bool
}
```

Persistence policy governs durable side effects:

```go
type PersistPolicy struct {
    Enabled         bool
    Sessions        bool
    Evaluations     bool
    BindingVersions bool
    BindingDocs     bool
}
```

Restore policy governs replay behavior:

```go
type RestorePolicy struct {
    AutoRestore bool
}
```

### Profile Semantics

`raw`:

- no persistence
- no auto-restore
- raw execution path
- no JSDoc extraction
- no binding/version persistence
- minimal or no snapshots

`interactive`:

- no persistence by default
- no auto-restore by default
- instrumented execution for nice REPL semantics
- console capture and binding tracking enabled
- static analysis enabled where useful for UI affordances
- JSDoc extraction optional but on by default if inexpensive

`persistent`:

- persistence enabled
- auto-restore enabled
- instrumented execution enabled
- binding tracking enabled
- JSDoc extraction enabled
- durable session/evaluation/binding/doc writes enabled

### Implementation Shape

The refactor should separate `replsession` evaluation into explicit stages:

```text
source
  -> optional parse/analyze
  -> optional rewrite
  -> execute
  -> optional runtime observation
  -> optional persistence write
```

That avoids smearing conditionals across one giant `Evaluate` method.

## Design Decisions

### Use profiles first, overrides second

Most callers should not hand-assemble a matrix of booleans. Profiles give a small, understandable API surface, while overrides keep it flexible.

### Keep persistence and observation separate

Observation answers "what metadata do we compute?" Persistence answers "which metadata do we durably store?" Those are related but not identical concerns.

### Allow per-session behavior

Different sessions within one app may need different semantics. A long-running daemon or TUI should not need a separate app instance just to create one raw scratch session.

### Keep current persistent behavior as the persistent preset

`cmd/goja-repl` and the JSON server should keep their current behavior by explicitly choosing `persistent`. The refactor should not silently weaken the durable path.

## Alternatives Considered

### Only add a few booleans to `replapi.New`

Rejected because it would create an opaque constructor with fragile interactions between flags.

### Keep `replapi` persistent-only and tell other callers to use `replsession`

Rejected because it makes `replapi` less useful as the shared product boundary and encourages more bypass paths.

### Create separate packages like `rawreplapi`, `interactiveapi`, `persistentapi`

Rejected because the underlying domain is the same. The difference is policy, not ownership or protocol.

## Implementation Plan

### Step 1: Add config and profile types to `pkg/replapi`

- Introduce the config, profile, and session option types.
- Add preset builders for `raw`, `interactive`, and `persistent`.
- Keep the public API narrow and document the preset semantics in code comments and tests.

### Step 2: Make `replsession` session creation policy-aware

- Add session-level policy state.
- Let `CreateSession` accept a policy-aware record/options struct rather than always using one implicit mode.
- Preserve the current session summary shape where possible.

### Step 3: Split evaluation into raw and instrumented paths

- Keep the current instrumented path for the persistent preset.
- Add a direct/raw path for callers that want minimal transformation.
- Share result formatting and error handling where it actually matches.

### Step 4: Gate persistence and restore behavior explicitly

- Allow `replapi.App` to exist without a store.
- Only use auto-restore if both the policy and a store are present.
- Fail fast if a caller requests persistence without providing a store.

### Step 5: Adopt the new API in existing entry points

- `cmd/goja-repl` should choose the persistent profile.
- `cmd/js-repl` should get an interactive profile constructor or seam.
- Existing tests should be updated to assert profile behavior rather than accidental defaults.

### Step 6: Add focused coverage

- raw profile evaluates without persistence requirements;
- interactive profile keeps REPL-friendly semantics without requiring SQLite;
- persistent profile still writes/reads durable state;
- per-session overrides beat app defaults in the expected direction.

## Open Questions

- Whether `interactive` should enable JSDoc extraction by default or leave it off until the UI asks for it.
- Whether raw mode should still support a narrow top-level-await wrapper or be literally "direct `RunString` only".
- Whether `cmd/repl` should be updated in this ticket or only after `cmd/js-repl` adopts the new API.

## References

- `pkg/replapi/app.go`
- `pkg/replsession/service.go`
- `pkg/repldb/store.go`
- `cmd/goja-repl/root.go`
- `cmd/js-repl/main.go`
