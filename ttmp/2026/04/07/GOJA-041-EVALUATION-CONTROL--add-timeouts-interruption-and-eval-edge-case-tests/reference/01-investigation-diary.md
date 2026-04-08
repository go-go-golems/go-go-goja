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
LastUpdated: 2026-04-08T18:29:00-04:00
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

This session initially produced a safe partial GOJA-041 slice rather than the full timeout-and-recovery design. The code first gained an explicit evaluation-timeout field in session policy, and promise waiting began honoring that deadline, while synchronous runaway JavaScript execution remained blocked pending a better understanding of the runtime stack.

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
- Stored the interruption experiments retroactively under [00-scripts-index.md](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/00-scripts-index.md):
  - [01-goja-plain-runtime-interrupt/main.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/01-goja-plain-runtime-interrupt/main.go)
  - [02-goja-nodejs-eventloop-interrupt/main.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/02-goja-nodejs-eventloop-interrupt/main.go)
  - [03-eventloop-same-vm-check/main.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/03-eventloop-same-vm-check/main.go)

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
- The direct reason is now clearer: the `eventloop` creates its own runtime instance, and our earlier failing repro was interrupting a different `*goja.Runtime`.
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
- At that point in the investigation, synchronous runaway `while (true) {}` execution still appeared unresolved.
- Repro scripts:
  - [01-goja-plain-runtime-interrupt/main.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/01-goja-plain-runtime-interrupt/main.go): plain upstream `goja` interrupt works for async IIFE runaway code.
  - [02-goja-nodejs-eventloop-interrupt/main.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/02-goja-nodejs-eventloop-interrupt/main.go): `eventloop` repro that times out waiting.
  - [03-eventloop-same-vm-check/main.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/03-eventloop-same-vm-check/main.go): confirms the loop runtime is not the same pointer as an independently created runtime.

## 2026-04-08 follow-up session

The earlier blocker turned out to be real evidence, but only for the wrong path. The key correction was noticing that the repository's `engine.Runtime` does not evaluate against the `eventloop`'s internal VM. It creates its own `goja.New()` runtime, then uses the event loop as a scheduler while `runtimeowner.Runner` invokes closures against `rt.VM`.

That means the right question was not:

"Can I interrupt `eventloop.NewEventLoop().vm` from outside?"

It was:

"Can I interrupt `engine.Runtime.VM` while `runtimeowner.Runner.Call(...)` is executing `vm.RunString(...)` on the owner goroutine?"

### Step 2: Build the decisive engine-path repro

I added [04-engine-runtimeowner-interrupt-sync-loop/main.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/04-engine-runtimeowner-interrupt-sync-loop/main.go) to test the actual runtime stack used by `replsession`.

That script prints:

- whether the callback VM is the same object as `rt.VM`
- whether `Interrupt(...)` unwinds a synchronous `while (true) {}`
- whether `errors.Is(...)` still sees the original interrupt cause
- whether `ClearInterrupt()` makes the VM reusable

### What it showed

The result was:

- `sameVM true`
- the interrupt error came back from `RunString(...)`
- `errors.Is(err, interruptErr) == true`
- `postInterrupt value=2 err=<nil>`

That is the decisive missing fact. The real engine/runtimeowner path is interruptible and recoverable after clearing the interrupt flag.

### Step 3: Implement synchronous timeout interruption in `replsession`

With that experiment in hand, I updated [pkg/replsession/evaluate.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go).

The implementation change is:

- create one evaluation deadline context per cell
- use that same context for both:
  - `RunString(...)`
  - later promise waiting, if the result is a promise
- start a watcher goroutine that waits for the deadline context to finish
- on timeout/cancellation, call `s.runtime.VM.Interrupt(cause)`
- keep `Owner.Call(...)` itself on `context.WithoutCancel(ctx)` so it waits for the evaluation to actually unwind instead of returning early just because the timeout context fired
- after the call returns, clear the interrupt flag if an interrupt was delivered

That last point matters. If `Owner.Call(...)` were allowed to return immediately on `ctx.Done()`, the service could report timeout before the owner goroutine had actually unwound `RunString(...)`, which would make `ClearInterrupt()` unsafe and would not truly guarantee session recovery.

### Why this design is correct

- It uses the actual runtime object being evaluated.
- It lets `goja` surface the interrupt as the evaluation error.
- `errors.Is(...)` still sees the original timeout cause, so status classification remains sound.
- The runtime is explicitly reset with `ClearInterrupt()` only after the interrupted call returns.

### Step 4: Prove recovery with tests

I added recovery tests in [pkg/replsession/service_policy_test.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service_policy_test.go):

- raw session:
  - `while (true) {}` returns `status == "timeout"`
  - a subsequent `1 + 1` succeeds
- interactive session:
  - `while (true) {}` returns `status == "timeout"`
  - a subsequent `const x = 41; x + 1` succeeds
  - binding tracking still works after recovery

### What I learned

- The original eventloop repro was still useful, but only as a clue that VM identity mattered.
- The actual production path is interruptible today.
- The important engineering detail is not just "call Interrupt", but "wait for the owner call to finish unwinding before clearing the interrupt and reusing the runtime."

### Updated current state

- Promise deadlines are implemented.
- Synchronous runaway timeout interruption is implemented.
- Session reuse after timeout is covered by tests.
- The remaining GOJA-041 question is no longer "is this possible?" but whether there are any additional cancellation semantics or edge cases worth tightening beyond the current design.
