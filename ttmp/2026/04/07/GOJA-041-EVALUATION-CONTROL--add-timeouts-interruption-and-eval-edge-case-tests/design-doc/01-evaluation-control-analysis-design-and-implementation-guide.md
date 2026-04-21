---
Title: Evaluation control analysis, design, and implementation guide
Ticket: GOJA-041-EVALUATION-CONTROL
Status: active
Topics:
    - goja
    - go
    - repl
    - architecture
    - analysis
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Intern-oriented guide for adding timeouts, interruption, and edge-case evaluation tests to the REPL kernel."
LastUpdated: 2026-04-07T10:00:00-04:00
WhatFor: "Provide a detailed analysis and implementation guide for the evaluation control PR."
WhenToUse: "Use when implementing, reviewing, or testing GOJA-041."
---

# Evaluation control analysis, design, and implementation guide

## Executive summary

This ticket makes the REPL execution model safe and understandable under failure. The main theme is not "performance." The main theme is "control."

If user code:

- runs forever
- waits forever
- returns a promise that never settles
- uses raw-mode top-level `await` in edge-case syntax

the system needs a clear and testable contract.

This ticket should produce that contract.

## What part of the system are we changing?

This work centers on the live session kernel in [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go).

Important entry points:

- top-level evaluation flow around [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go#L243)
- raw-mode rewrite selection around [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go#L279)
- `wrapTopLevelAwaitExpression` at [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go#L1021)
- `waitPromise` at [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go#L1101)
- session policy definitions in [policy.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go#L50)

Supporting files:

- [pkg/replsession/rewrite.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/rewrite.go#L13)
- [pkg/replapi/config.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go#L30)

## Mental model for an intern

The REPL currently has two broad execution modes:

```text
instrumented mode:
    parse
    analyze
    optionally rewrite source
    run code
    collect reports and persistence data

raw mode:
    skip most instrumentation
    maybe wrap top-level await
    run code
    maybe wait on promise
```

The difficulty is not just running code. The difficulty is deciding how long we are willing to wait and what we do when execution never finishes.

## Problem 1: there is no first-class timeout policy

### Why this matters

Without a timeout policy, the REPL does not define what happens to:

- `while(true){}`
- recursive runaway code
- promises that never settle
- user code that accidentally hangs

Even if you think "some REPLs hang forever," that is still a policy. Right now the system does not make the policy explicit.

### Correct design

Timeout belongs in the session policy model, not as an ad hoc handler or CLI-only knob.

That means the configuration story should look like:

```text
session options
    -> session policy
        -> eval policy
            -> timeout
            -> top-level await support
            -> execution mode
```

This matters because the timeout is part of what kind of REPL session the user asked for.

### Recommended design choice

Add an evaluation timeout field at the policy level, likely inside `EvalPolicy` or an adjacent execution-policy structure.

Keep the semantics simple:

- zero duration means "no timeout" only if that behavior is explicitly allowed
- otherwise pick a conservative default for interactive/persistent profiles

I would strongly prefer explicit defaults by profile rather than leaving this undefined.

## Problem 2: `waitPromise` is a polling loop with incomplete control semantics

Today, `waitPromise` repeatedly checks promise state and sleeps:

```text
loop:
    read promise state
    if pending:
        sleep 5ms
        continue
    if fulfilled:
        return value
    if rejected:
        return error
```

This is not automatically wrong. Polling can be acceptable in a small REPL.

The real problem is that the system does not clearly define:

- how polling stops on cancellation
- how timeout interacts with promise waiting
- whether a hung promise should mark the session unusable

### Design goal

Treat promise waiting as part of the same execution contract as normal code execution. Do not treat it as an unrelated helper.

That means:

- timeout applies to promise waiting too
- cancellation applies to promise waiting too
- errors should be reported consistently

### Pseudocode

```text
func evaluateWithControl(ctx, source):
    execCtx = maybeWithTimeout(ctx, policy.evalTimeout)
    result = run code under execCtx
    if result is promise:
        return waitForPromise(execCtx, promise)
    return result
```

And:

```text
func waitForPromise(ctx, promise):
    loop:
        if ctx done:
            interrupt VM
            return timeout or cancellation error
        snapshot = promise state
        switch snapshot.state:
            pending   -> sleep briefly
            fulfilled -> return value
            rejected  -> return error
```

## Problem 3: raw-mode top-level `await` handling is heuristic

`wrapTopLevelAwaitExpression` currently recognizes a narrow source shape:

```text
if trimmed source starts with "await ":
    wrap in async IIFE
```

This is good as a minimal compatibility feature, but it is not a general "raw mode supports top-level await" implementation. It is a narrow syntax convenience.

### Why this matters

If the public mental model says "raw mode supports top-level await," users will expect more than prefix-only support.

Examples:

```javascript
await Promise.resolve(1)
const x = await Promise.resolve(1)
foo(await Promise.resolve(1))
```

Those are not equivalent from a heuristic wrapper's point of view.

### Design choice to make explicitly

Pick one of these:

1. raw mode only supports the narrow leading-`await` expression case
2. raw mode has broader support and you improve detection
3. raw mode does not promise top-level `await` except in documented limited cases

The bad option is leaving it ambiguous.

### My recommendation

Do not overbuild parser logic into this PR. Keep the implementation modest and strengthen the contract:

- either document the narrow support
- or slightly improve it if there is a very clear safe extension

The most important thing is test coverage that matches the chosen contract.

## Problem 4: session survivability after timeout

This is the practical requirement many teams forget.

If a cell times out, what happens next?

The desired answer is:

```text
that evaluation fails
the session remains alive
the next simple evaluation still works
```

If timeout breaks the runtime permanently, the feature is only half-implemented.

### Required invariant

```text
a timed-out evaluation must fail in a controlled way without corrupting the session's ability to handle later evaluations
```

That requirement should drive both implementation and tests.

## Proposed implementation plan

### Step 1: define timeout in policy

Add timeout configuration close to the rest of evaluation policy in:

- [pkg/replsession/policy.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go#L50)
- [pkg/replapi/config.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go#L30)

### Step 2: route evaluation through timeout-aware execution context

Wrap raw and instrumented evaluation flows in timeout-aware contexts before runtime execution starts.

### Step 3: interrupt the VM on timeout

This is the key implementation detail. Timeout must not merely stop waiting in Go. It must also stop the underlying JS execution path.

### Step 4: align promise waiting with the same timeout/cancellation contract

`waitPromise` should respect the same execution context and error shape.

### Step 5: add regression tests

Focus on behavior, not internal implementation.

## Tests to write

### Test group A: timeout of runaway code

- evaluate `while(true){}`
- assert timeout error
- assert session remains usable

### Test group B: promise waiting

- evaluate a promise that resolves quickly
- assert success
- evaluate a promise that never settles
- assert timeout or cancellation behavior

### Test group C: raw-mode top-level `await`

Write tests for exactly the supported contract:

- `await Promise.resolve(1)`
- `const x = await Promise.resolve(1)` if you choose to support or reject it explicitly
- one negative case to make behavior intentional rather than accidental

### Test group D: post-timeout recovery

Sequence:

```text
create session
run hanging cell -> timeout
run "1 + 1" -> should still succeed
```

This is the most important user-facing regression test after the timeout itself.

## What not to do in this PR

- do not refactor `service.go` heavily here
- do not rename packages or session types
- do not mix in persistence fixes
- do not turn this into a general "modernize the REPL" PR

Keep this PR about execution control.

## Manual testing guide

### Test 1: timeout path

Create a session, then evaluate:

```javascript
while(true){}
```

Expected result:

- evaluation fails with timeout
- process remains healthy
- session remains usable

### Test 2: post-timeout recovery

Immediately after the timeout, evaluate:

```javascript
1 + 1
```

Expected result:

- returns `2`

### Test 3: raw-mode top-level await

Evaluate:

```javascript
await Promise.resolve(1)
```

Then evaluate your chosen edge case:

```javascript
const x = await Promise.resolve(1)
```

Expected result:

- behavior matches documented contract exactly

## Final advice for the intern

This ticket is not about making the runtime "fast." It is about making it governable.

When you work on it, keep asking:

- What happens if the code never finishes?
- How does the caller learn that?
- Can the session continue afterward?
- Did we test the unhappy path directly?
