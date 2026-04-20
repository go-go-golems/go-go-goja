---
Title: Diary
Ticket: GOJA-051
Status: active
Topics:
    - goja
    - engine
    - lifecycle
    - repl
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: |-
        Runtime construction path that wires owner, loop, bridge, and AddCloser exposure.
        Shows how lifecycle participants are created and wired
    - Path: engine/runtime.go
      Note: |-
        Main source of current close-path complexity and lifecycle sequencing.
        Primary source of close-path complexity inspected during the diary step
    - Path: engine/runtime_modules.go
      Note: |-
        Current module registrar extension point.
        Shows the current AddCloser-only registrar extension point
    - Path: pkg/hashiplugin/host/registrar.go
      Note: |-
        Existing runtime-scoped cleanup consumer.
        Real cleanup consumer used to justify phase-aware lifecycle hooks
    - Path: pkg/runtimeowner/runner.go
      Note: |-
        Concrete owner shutdown behavior.
        Concrete runner shutdown behavior reviewed during analysis
    - Path: pkg/runtimeowner/types.go
      Note: |-
        Owner interface contract.
        Owner abstraction reviewed during analysis
ExternalSources: []
Summary: Chronological investigation diary for GOJA-051, covering ticket creation, evidence gathering, design authoring, validation, and reMarkable delivery.
LastUpdated: 2026-04-20T11:31:00-04:00
WhatFor: Continuation-friendly record of how the lifecycle-object design ticket was created and validated.
WhenToUse: Use when reviewing or extending GOJA-051 documentation work.
---


# Diary

## Goal

Record the evidence-gathering and document-authoring work for GOJA-051 so a future contributor can see exactly how the lifecycle-object proposal was grounded in the current engine/runtimeowner code and how the deliverable was prepared for reMarkable.

## Step 1: Create the ticket and map the current runtime shutdown surface

I created a new docmgr ticket for the runtime lifecycle cleanup request and then inspected the current engine/runtimeowner shutdown path before writing any recommendation. The point of this step was to avoid writing a generic “clean up Close()” note and instead anchor the guide to the exact code paths that currently encode shutdown sequencing and cleanup registration.

I focused first on `engine/runtime.go`, `engine/factory.go`, `engine/runtime_modules.go`, `pkg/runtimeowner/types.go`, `pkg/runtimeowner/runner.go`, and `pkg/hashiplugin/host/registrar.go`. Those files are the minimum surface needed to explain why `Runtime.Close()` is carrying too much lifecycle responsibility, how runtime-scoped modules currently register cleanup, and why a lifecycle object would be a better abstraction than one generic closer stack.

### Prompt Context

**User prompt (verbatim):** "Ok, create a ticket with a design and implementation guide to clean up the OwnerCall with a nice lifecycle object. Upload to remarkable."

**Assistant interpretation:** Create a new ticket-style documentation workspace, write an evidence-backed design and implementation guide for replacing the current owner/close sequencing with a better lifecycle abstraction, and deliver the result to reMarkable.

**Inferred user intent:** Capture the lifecycle-cleanup idea as a real implementation ticket that another engineer could pick up later, not just as an ephemeral chat suggestion.

### What I did

- Ran `docmgr status --summary-only` to confirm the repository docmgr setup.
- Created ticket `GOJA-051` with docmgr.
- Added a design doc and a diary doc under the new ticket workspace.
- Gathered file-backed evidence with:
  - `rg -n "AddCloser\(|Close\(ctx context.Context\)|Shutdown\(ctx context.Context\)|type Runner interface|runtimebridge.Delete|Loop.Stop|closeOnce" engine pkg/runtimeowner pkg/runtimebridge modules pkg/replsession -S`
  - `nl -ba engine/runtime.go | sed -n '1,180p'`
  - `nl -ba engine/factory.go | sed -n '140,260p'`
  - `nl -ba engine/runtime_modules.go | sed -n '1,120p'`
  - `nl -ba pkg/runtimeowner/types.go | sed -n '1,120p'`
  - `nl -ba pkg/runtimeowner/runner.go | sed -n '1,180p'`
  - `nl -ba pkg/hashiplugin/host/registrar.go | sed -n '60,110p'`
  - `nl -ba engine/runtime_modules_test.go | sed -n '40,170p'`

### Why

- The user explicitly asked for a ticket and implementation guide, which means the result needed to be persistent, structured, and actionable.
- The recommendation had to be based on the current repository shape, especially the close path and the runtime-scoped cleanup extension point.

### What worked

- The docmgr ticket workflow created the expected workspace and primary docs cleanly.
- The runtime ownership and cleanup evidence was concentrated in a small enough set of files to support a focused design doc.
- The existing tests made it clear that the current public contract is “generic reverse-order closers,” which is useful context for designing a migration path.

### What didn't work

- The phrase “OwnerCall” from the user prompt did not map to an exact symbol name in the repository. I verified that by searching for variants like `Owner.Close`, `Owner.Shutdown`, `closeOnce`, and `runtime owner` instead of looking for a literal `OwnerCall` identifier.
- Command used during that clarification pass:
  - `grep -Rsn "Owner\.Close\|Owner\.Cl[a-z]*\|closeOnce\|runtime owner\|Shutdown(ctx)\|Owner.Shutdown" . | head -80`

### What I learned

- The real architectural issue is broader than the owner object itself. The current complexity sits at the boundary where `engine.Runtime` coordinates context cancellation, bridge cleanup, generic cleanup hooks, owner shutdown, and loop stop.
- Runtime-scoped modules already have a real cleanup consumer (`pkg/hashiplugin/host/registrar.go`), so the lifecycle-object proposal is justified by existing code, not hypothetical future needs.

### What was tricky to build

The tricky part was scoping the proposal correctly. It would have been easy to overreach and redesign `runtimeowner.Runner` itself, but the evidence in the code shows that the immediate modeling problem lives in `engine.Runtime.Close()` and the `AddCloser` registration surface exposed through `RuntimeModuleContext`. The solution needed to improve shutdown modeling without pretending that the owner runner is the only lifecycle participant.

### What warrants a second pair of eyes

- Whether the proposed default phase order is exactly right for all runtime-scoped module integrations.
- Whether any current cleanup hook implicitly depends on the bridge still being present when runtime cleanup runs.
- Whether `AddCloser` should remain as a compatibility wrapper or be removed in one migration pass.

### What should be done in the future

- If this ticket gets implemented, add lifecycle-focused tests first so the refactor is driven by contract rather than by method-body cleanup alone.
- Revisit plugin/runtime integration after phase-aware hooks exist and decide whether any cleanup needs owner-thread execution guarantees.

### Code review instructions

- Start with `engine/runtime.go` and compare the current close path against the phase model proposed in the design doc.
- Then read `engine/runtime_modules.go` and `pkg/hashiplugin/host/registrar.go` to see why generic `AddCloser` is now too weak an extension point.
- Validate the evidence trail by rerunning the `rg` and `nl -ba` commands listed above.

### Technical details

- Ticket path: `ttmp/2026/04/20/GOJA-051--introduce-a-lifecycle-object-for-runtime-owner-shutdown-and-cleanup`
- Primary design doc: `ttmp/2026/04/20/GOJA-051--introduce-a-lifecycle-object-for-runtime-owner-shutdown-and-cleanup/design-doc/01-design-and-implementation-guide-for-replacing-runtime-close-sequencing-with-a-lifecycle-object.md`
- Diary: `ttmp/2026/04/20/GOJA-051--introduce-a-lifecycle-object-for-runtime-owner-shutdown-and-cleanup/reference/01-diary.md`
