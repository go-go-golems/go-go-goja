---
Title: Diary
Ticket: GOJA-TIMER-MODULE
Status: active
Topics:
    - goja
    - javascript
    - modules
    - async
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: Diary step 1 references runtime creation and loop setup
    - Path: engine/runtime.go
      Note: Diary step 1 references the currently shipped built-in modules
    - Path: modules/database/database.go
      Note: Diary step 1 used database as a reference for configurable module design
    - Path: modules/fs/fs.go
      Note: Diary step 1 used fs as the reference module shape
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-23T14:56:29.640158128-04:00
WhatFor: Record the implementation work for the go-go-goja timer module.
WhenToUse: Use when reviewing what changed, why it changed, and how the module was validated.
---


# Diary

## Goal

Capture the implementation of the shipped `timer` module, including the planning decisions, the exact code/test slices, and the validation steps used to prove the module works in a fresh runtime.

## Step 1: Ticket setup and architecture capture

This step established the documentation workspace and verified the current runtime shape before any code changes. The main goal was to confirm whether the repository already shipped timer functionality or only documented it, because that determines whether this is a wiring task or a new module task.

The inspection showed that the runtime always creates an event loop, but a fresh runtime still reports `typeof setTimeout === "undefined"` and there is no `modules/timer` package in the repository. That means the work should be treated as a proper new built-in module rather than a small docs fix.

### Prompt Context

**User prompt (verbatim):** "do the timer module, straight in go-go-goja. Createa a new ticket with docmgr --root go-go-goja/ttmp and then add implementation plan and tasks and work them off one by one, comitting as you go, keeping a detailed diary"

**Assistant interpretation:** Create a dedicated `go-go-goja` ticket, document the plan, then implement the timer module in small committed steps while keeping the ticket diary current.

**Inferred user intent:** Add a real, shipped timer primitive to `go-go-goja` and leave behind enough documentation and history that the work is easy to review and continue.

### What I did

- Created the ticket workspace `GOJA-TIMER-MODULE` under `go-go-goja/ttmp`.
- Added a design doc for the implementation plan and a reference diary document.
- Inspected the existing runtime and module registration path in:
  - [factory.go](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/engine/factory.go)
  - [runtime.go](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/engine/runtime.go)
  - [module_specs.go](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/engine/module_specs.go)
  - [fs.go](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/modules/fs/fs.go)
  - [database.go](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/modules/database/database.go)
- Verified via a temporary Go probe that a fresh runtime currently exposes no timer globals.

### Why

- The ticket and plan needed to exist before the implementation work so the code and docs could evolve together.
- The runtime inspection was necessary to avoid implementing the wrong thing. If a timer module had already existed but was unregistered, the solution would have been different.

### What worked

- `docmgr ticket create-ticket --root /home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/ttmp --ticket GOJA-TIMER-MODULE --title "Add timer module with Promise-based sleep primitive" --topics goja,javascript,modules,async`
- `docmgr doc add ... --doc-type design-doc --title "Implementation plan for timer module"`
- `docmgr doc add ... --doc-type reference --title "Diary"`
- Source inspection showed that the default runtime currently blank-imports only `database`, `exec`, and `fs`.

### What didn't work

- A first quick probe program had type mistakes and failed with:

```text
/tmp/goja-timer-check-u0IB.go:16:10: declared and not used: expr
/tmp/goja-timer-check-u0IB.go:17:60: cannot use func(ctx context.Context, vm any) (any, error) {…} (value of type func(ctx context.Context, vm any) (any, error)) as runtimeowner.CallFunc value in argument to rt.Owner.Call
/tmp/goja-timer-check-u0IB.go:24:89: undefined: goja
/tmp/goja-timer-check-u0IB.go:28:39: out.Export undefined (type any has no field or method Export)
```

- The corrected probe then printed:

```text
typeof setTimeout = undefined
typeof setInterval = undefined
typeof clearTimeout = undefined
typeof clearInterval = undefined
```

### What I learned

- The async docs and README already describe a `timer` module shape, but the repository code has not caught up with that documentation yet.
- `go-go-goja` already has the right substrate for the work: a per-runtime event loop plus the owner-runner abstraction.

### What was tricky to build

- The only subtle part in this setup step was distinguishing repository intent from shipped behavior. The docs made it easy to assume that `timer` already existed, but the source tree and a real runtime probe showed that the docs were ahead of the implementation.

### What warrants a second pair of eyes

- The eventual module implementation should be reviewed for owner-thread correctness during Promise settlement and for any runtime shutdown edge cases.

### What should be done in the future

- Implement the actual module and prove it works in tests.

### Code review instructions

- Start with [factory.go](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/engine/factory.go) and [runtime.go](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/engine/runtime.go) to see how runtimes are created.
- Then read [01-implementation-plan-for-timer-module.md](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/ttmp/2026/03/23/GOJA-TIMER-MODULE--add-timer-module-with-promise-based-sleep-primitive/design-doc/01-implementation-plan-for-timer-module.md) for the planned module shape.

### Technical details

- The fresh runtime probe used `engine.NewBuilder().Build().NewRuntime(context.Background())`.
- The current default registry behavior is driven by `modules.EnableAll(reg)` plus the blank imports in [runtime.go](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/engine/runtime.go).
