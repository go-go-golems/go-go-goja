---
Title: Investigation diary
Ticket: GOJA-041-EVALUATION-CONTROL
Status: active
Topics:
    - goja
    - go
    - repl
    - analysis
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological notes for the evaluation-control design guide."
LastUpdated: 2026-04-07T10:00:00-04:00
WhatFor: "Record the reasoning behind the timeout and interruption design."
WhenToUse: "Use when retracing why GOJA-041 was separated from persistence and cleanup."
---

# Investigation diary

## Why this ticket exists

This ticket exists because evaluation-control problems are real, but they should not be mixed into the persistence correctness PR.

The main issues are:

- incomplete timeout policy
- promise polling without an explicit loop-level cancellation check
- uncertain behavior for hung or runaway execution
- narrow raw-mode top-level `await` handling

## Files to understand first

- [pkg/replsession/service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go)
- [pkg/replsession/policy.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go)
- [pkg/replsession/rewrite.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/rewrite.go)
- [pkg/replapi/config.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go)

## Main design stance

The system must have a clear answer to this question:

"What happens when user code never returns?"

Without that answer, the REPL has no durable execution contract.

## 2026-04-08 implementation session

This session produced a safe partial GOJA-041 slice rather than the full timeout-and-recovery design. The code now has an explicit evaluation-timeout field in session policy, and promise waiting honors that deadline, but synchronous runaway JavaScript execution is still blocked on the runtime stack behavior described below.

The main reason for stopping short is that the straightforward design did not hold under test. A direct `goja.Runtime.Interrupt(...)` attempt did not unwind a runaway `RunString(...)` when execution was happening under the repository's `goja_nodejs/eventloop` model, so shipping a "timeout" response for synchronous infinite loops would have been misleading and unsafe.

### Step 1: Add policy-level deadlines and promise-wait timeout handling

**User prompt (verbatim):** "Alright, add tasks for GOJA-040, then work on them one by one, committing at appropriate intervals, keeping a diary as you go."

**Assistant interpretation:** Continue the ticket workflow by implementing the next reviewable GOJA-041 slice, keeping code and ticket docs in sync.

**Inferred user intent:** Make steady, reviewable progress on the planned REPL hardening work rather than leaving the tickets as documentation only.

### What I did

- Added `timeoutMs` to `EvalPolicy` in [pkg/replsession/policy.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go).
- Set default evaluation deadlines in the stock raw and interactive session presets.
- Updated promise waiting in [pkg/replsession/service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go) to check `ctx.Done()` on each polling loop and classify timeout errors as `status == "timeout"`.
- Added raw-mode coverage in [pkg/replsession/service_policy_test.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service_policy_test.go) for:
  - expression-style top-level `await`
  - declaration-style top-level `await`
  - never-settling awaited promises
- Validated the safe subset with:
  - `go test ./pkg/replsession -run 'TestServiceRawSessionAwaitExpressionWorksButDeclarationDoesNot|TestServiceRawAwaitPromiseTimeoutUsesEvalDeadline|TestServiceRawSessionUsesDirectExecution|TestServiceInteractiveSessionTracksBindingsWithoutPersistence'`
  - `go test ./pkg/replsession ./pkg/replapi`

### Why

- The timeout setting needs to live in the session policy because it is part of the execution contract, not just a transport-level flag.
- Promise waiting already had a polling loop, so it was the lowest-risk place to make timeouts real without claiming that runaway synchronous JS was solved.
- The raw-mode `await` behavior was previously only implied by code; now it is documented by tests.

### What worked

- Policy normalization and JSON shape changes were straightforward.
- Promise waiting can now time out cleanly and deterministically.
- The raw-mode tests confirmed the current contract clearly enough to prevent future confusion.

### What didn't work

- A direct attempt to interrupt synchronous runaway execution did not work.
- I first tried a context-driven interrupt path in `runString(...)`, then a direct timer-driven interrupt path. Both left an instrumented `while (true) {}` evaluation hanging.
- I also ran a standalone repro outside the REPL path using `goja` plus `goja_nodejs/eventloop`, and it printed:
  - `scheduled true`
  - `interrupting`
  - `timed out waiting`
- That result strongly suggests the limitation is in the current runtime/event-loop execution model, not just in REPL wrapper code.

### What I learned

- Promise deadlines are implementable now.
- Synchronous runaway-evaluation recovery needs a deeper design than "call `vm.Interrupt(...)` and keep going" in this repo's runtime model.
- The right next step is probably architectural, not another small patch in `replsession/service.go`.

### What was tricky to build

- The sharp edge is that there are really two timeout problems:
  - waiting on a `Promise`
  - interrupting CPU-bound synchronous JS
- The first one is cooperative polling and easy to control.
- The second one depends on runtime/event-loop semantics below the REPL layer, and the obvious mechanism did not work in practice.

### What warrants a second pair of eyes

- Whether the repo should keep trying to recover the same runtime after a synchronous hang, or instead treat timeout as a runtime-replacement boundary.
- Whether `engine.Factory.NewRuntime()` should expose a different execution mode for REPL sessions that need reliable interruption semantics.

### What should be done in the future

- Decide whether synchronous timeout recovery should:
  - work inside the existing event-loop-backed runtime
  - rebuild the runtime/session after timeout
  - move evaluation into a more strongly isolated execution boundary

### Code review instructions

- Start with [pkg/replsession/policy.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go) to see the new policy surface.
- Then read [pkg/replsession/service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go) around `executeRaw`, `executeWrapped`, `waitPromise`, `executionStatus`, and `evaluationContext`.
- Finish with [pkg/replsession/service_policy_test.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service_policy_test.go) to see the contract the current code actually enforces.

### Technical details

- Current safe behavior:
  - `await Promise.resolve(9)` in raw mode can succeed when `SupportTopLevelAwait` is enabled.
  - `const x = await Promise.resolve(3)` in raw mode still fails because the wrapper only supports the expression-prefix case.
  - `await new Promise(() => {})` now times out when `timeoutMs` is set.
- Current unresolved behavior:
  - synchronous runaway `while (true) {}` execution is still not safely recoverable under the current event-loop-backed runtime stack.
