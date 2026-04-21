---
Title: REPL cleanup analysis, design, and implementation guide
Ticket: GOJA-042-REPL-CLEANUP
Status: active
Topics:
    - goja
    - go
    - repl
    - refactor
    - architecture
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Intern-oriented guide for splitting the session kernel by responsibility, clarifying API shape, and planning consolidation of the older evaluator path."
LastUpdated: 2026-04-07T10:00:00-04:00
WhatFor: "Provide a detailed analysis and implementation guide for the cleanup/refactor PR."
WhenToUse: "Use when implementing, reviewing, or testing GOJA-042."
---

# REPL cleanup analysis, design, and implementation guide

## Executive summary

This ticket exists because the REPL code is harder to understand than it needs to be, even after the behavior bugs are fixed.

The two reviews agreed on the basic maintainability problem:

- too much logic sits inside one giant file
- some type names are easy to confuse
- there is an older evaluator path that still exists and needs an explicit plan

This ticket is intentionally last because cleanup should follow semantics. Do not refactor the world while the correctness contract is still moving.

## What part of the system are we cleaning up?

The center of gravity is [pkg/replsession/service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go).

That file currently contains a large mix of responsibilities, including:

- service construction
- session lifecycle
- evaluation orchestration
- persistence record shaping
- restore/replay logic
- runtime snapshotting
- binding bookkeeping
- console capture
- doc-sentinel installation
- promise waiting
- summary building

Important related files:

- [pkg/replsession/rewrite.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/rewrite.go#L13)
- [pkg/replsession/policy.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go#L50)
- [pkg/replapi/config.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go#L30)
- [pkg/repl/adapters/bobatea/javascript.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repl/adapters/bobatea/javascript.go#L10)
- [pkg/repl/evaluators/javascript/evaluator.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go#L56)

## Cleanup goal 1: split `service.go` by responsibility

### Why this matters

Large files are not automatically bad. A large file becomes bad when it mixes different change reasons together.

That is the problem here. Different kinds of future work all have to touch the same file:

- persistence fixes
- evaluation fixes
- summary/report fixes
- restore behavior changes
- binding instrumentation changes

That makes code review harder and raises regression risk.

### Target shape

Do not split randomly. Split by responsibility boundaries.

A reasonable end-state is:

```text
pkg/replsession/
    service.go        service type, constructor, top-level public API
    lifecycle.go      create/delete/restore session lifecycle
    evaluate.go       main evaluation orchestration
    execute.go        raw/wrapped execution helpers and timeout wiring
    snapshot.go       snapshot and diff logic
    bindings.go       binding bookkeeping and runtime refresh
    persistence.go    persistence record shaping
    docs.go           doc sentinel installation and doc extraction helpers
    summary.go        summary building
```

You do not need this exact file list. The important idea is that each file should answer one question.

### Rule of thumb

If a future bug report says:

- "restore is wrong"
- "timeout is wrong"
- "binding persistence is wrong"

the engineer should be able to guess the right file.

## Cleanup goal 2: clarify the `SessionOptions` API shape

There are two `SessionOptions` concepts:

- app-layer override/config shape in [pkg/replapi/config.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go#L30)
- kernel-layer resolved session options in [pkg/replsession/policy.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go#L50)

This is not a correctness bug, but it is easy for new readers to get lost.

### Recommended naming direction

Preserve the kernel type as the "real resolved settings" type, and rename the app-layer type toward override semantics.

Examples:

- `SessionOverrides`
- `CreateSessionOverrides`
- `SessionRequest`

The goal is to teach the reader this distinction:

```text
app layer:
    "I may override some defaults"

kernel layer:
    "these are the resolved settings for one live session"
```

### Important caution

Only do this rename in the cleanup PR after the correctness PRs land. Otherwise you are mixing semantic behavior changes with API cleanup noise.

## Cleanup goal 3: decide what to do with the older evaluator path

The review-review found that calling the old evaluator "deprecated" was too strong, because it is still wired through the Bobatea adapter.

That means you need an explicit decision, not a vague one.

The current relationship looks like this:

```text
new REPL session kernel
    -> pkg/replsession/*

older evaluator path
    -> pkg/repl/evaluators/javascript/evaluator.go
    -> adapted by pkg/repl/adapters/bobatea/javascript.go
```

### Options

Option A: keep it and document it

- good if Bobatea still depends on it meaningfully
- add comments explaining why both paths exist

Option B: plan consolidation

- only if there is a clear migration target
- should become its own explicit follow-up task

Option C: partial extraction of shared helpers

- if both paths genuinely need the same utility logic
- but do not over-abstract just to reduce line count

### My recommendation

Do not force consolidation in the cleanup PR unless the destination architecture is clear. The safer cleanup move is:

- document the coexistence
- reduce confusion
- create a later migration ticket only if there is real value

## What this PR should not do

- do not change persistence semantics
- do not change timeout semantics
- do not widen product scope
- do not sneak in transport/API behavior changes

This PR should feel boring in the best possible way:

```text
same behavior
clearer structure
easier review
easier onboarding
```

## Suggested implementation sequence

### Step 1: extract without renaming behavior

Move cohesive helper groups into new files while keeping function names stable where possible.

### Step 2: run tests after each extraction slice

Because this file is central, small incremental moves are safer than one giant patch.

### Step 3: rename app-layer `SessionOptions` if still warranted

Do this after file extraction settles, not before.

### Step 4: add comments/docstrings where the architecture is non-obvious

Especially around:

- session creation flow
- restore flow
- persistence boundary
- legacy evaluator coexistence

## Pseudocode for the target architecture

```text
service.go:
    type Service
    NewService(...)
    public API methods call focused helpers

lifecycle.go:
    create session
    delete session
    restore session

evaluate.go:
    analyze
    choose execution path
    collect report
    persist if needed

snapshot.go:
    snapshot globals
    diff globals

bindings.go:
    upsert declared bindings
    refresh runtime details

persistence.go:
    build DB record payloads
```

## Review checklist

- Did behavior stay the same?
- Did file boundaries become easier to explain?
- Can a new engineer find restore logic quickly?
- Can a new engineer find persistence shaping quickly?
- Did the rename of any public-ish type improve clarity enough to justify churn?
- Did we avoid speculative abstraction?

## Manual testing guide

For this PR, manual testing is mostly regression testing:

```bash
go test ./...
```

Then a small smoke pass:

1. create a session
2. evaluate a normal cell
3. restore if persistence is enabled
4. inspect list/history/export if those paths are already stable from PR 1

The point is not to discover new behavior. The point is to prove cleanup did not break old behavior.

## Final advice for the intern

Refactors become messy when they are really secret behavior changes. Avoid that.

If you do this ticket well, the result should let the next engineer say:

"I understand where to look now."
