# REPL Hardening Project Report

Date: 2026-04-08
Branch: `task/add-repl-service`
Scope: `GOJA-040`, `GOJA-041`, `GOJA-042`

## Executive Summary

This project took the REPL work from review findings to implemented fixes and cleanup across three coordinated tracks:

- persistence correctness
- evaluation timeout and recovery behavior
- structural cleanup and boundary clarification

The result is a more reliable and reviewable REPL subsystem. The persistent-session path now respects deletion semantics and uses collision-resistant IDs. SQLite integrity settings are applied at runtime-open time instead of relying on one bootstrap connection. Evaluation control now covers both promise timeouts and synchronous runaway code, and sessions remain usable after timeout recovery. The `replsession` package was also split into clearer responsibility boundaries so the core lifecycle file is no longer a multi-purpose catch-all.

## Problem Statement

The branch review found a mix of correctness bugs and cleanup debt in the REPL/session stack:

- deleted sessions could still be listed or restored
- generated persistent session IDs could collide across processes
- SQLite foreign key enforcement was not reliably applied to pooled connections
- evaluation timeout behavior was incomplete
- synchronous runaway JavaScript did not have a clear recovery contract
- `pkg/replsession/service.go` had grown too large and responsibility-dense
- the boundary between `replapi` and `replsession` session configuration was easy to misread

These issues made the system harder to trust operationally and harder to review architecturally.

## Work Completed

### 1. Persistence Correctness (`GOJA-040`)

Implemented fixes:

- hide deleted sessions from normal read paths
- generate collision-resistant default session IDs
- configure SQLite integrity settings at open time

Key commits:

- `30eab79` `fix(repldb): hide deleted sessions from normal reads`
- `9213f13` `fix(replsession): use generated default session ids`
- `0ec8f0b` `fix(repldb): configure sqlite integrity at open time`

Important code references:

- [read.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/read.go)
- [store.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/store.go)
- [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go)

Outcome:

- deleted now means hidden from normal reads
- default durable IDs no longer depend on process-local counters
- database constraint behavior is much closer to the intended invariant on every opened runtime connection

### 2. Evaluation Control (`GOJA-041`)

Implemented fixes:

- add `timeoutMs` to session evaluation policy
- make promise waiting deadline-aware
- interrupt synchronous runaway execution on timeout
- clear interrupt state after unwind so the session stays usable
- add recovery tests for both raw and interactive sessions

Key commits:

- `055814a` `feat(replsession): add eval policy deadlines for awaited promises`
- `4c2399e` `feat(replsession): interrupt synchronous evaluations on timeout`

Important code references:

- [policy.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go)
- [evaluate.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go)
- [service_policy_test.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service_policy_test.go)

Important investigation result:

- the earlier `eventloop` interrupt repro was misleading because it was not targeting the same VM used by `engine.Runtime`
- the decisive engine-path repro showed that `rt.VM.Interrupt(...)` plus `ClearInterrupt()` works for the real `runtimeowner.Runner.Call(...)` path and allows reuse after timeout

Supporting experiment:

- [main.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/04-engine-runtimeowner-interrupt-sync-loop/main.go)

Outcome:

- promise timeouts are real
- synchronous timeout recovery is real
- a timed-out session can still evaluate subsequent cells

### 3. Cleanup and Refactor (`GOJA-042`)

Implemented cleanup:

- split evaluation flow out of `service.go`
- split persistence shaping out of `service.go`
- split runtime observation and summary shaping out of `service.go`
- clarify the `replapi.SessionOptions` vs `replsession.SessionOptions` boundary
- document that the Bobatea evaluator path is still active, not deprecated dead code
- add a focused test for create-session option resolution

Key commits:

- `4fcce77` `refactor(replsession): extract evaluation pipeline from service`
- `cb2d952` `refactor(replsession): extract persistence helpers from service`
- `5ab661b` `refactor(replsession): extract observation and summary helpers`
- `a35fc04` `refactor(replapi): clarify session option and evaluator boundaries`
- `96264e0` `test(replapi): cover create-session option resolution`

Important code references:

- [service.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go)
- [evaluate.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go)
- [persistence.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/persistence.go)
- [observe.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/observe.go)
- [config.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go)
- [config_test.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config_test.go)
- [replapi.go](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repl/adapters/bobatea/replapi.go)

Structural result:

- `service.go`: 467 lines
- `evaluate.go`: 523 lines
- `persistence.go`: 299 lines
- `observe.go`: 434 lines

This is not a reduction in total code size. It is a separation of concerns improvement that makes the subsystem easier to review and change safely.

## Validation

The work was validated continuously during implementation and again through pre-commit hooks.

Main validations used:

- `go test ./pkg/repldb ./pkg/replapi ./pkg/replsession`
- `go test ./pkg/replapi ./pkg/repl/adapters/bobatea ./pkg/repl/evaluators/javascript`
- `go test ./...`
- `golangci-lint run -v`
- `docmgr doctor --ticket GOJA-040-PERSISTENCE-CORRECTNESS --stale-after 30`
- `docmgr doctor --ticket GOJA-041-EVALUATION-CONTROL --stale-after 30`
- `docmgr doctor --ticket GOJA-042-REPL-CLEANUP --stale-after 30`

All three tickets were also refreshed on reMarkable after implementation:

- `GOJA-040 Persistence Correctness`
- `GOJA-041 Evaluation Control`
- `GOJA-042 REPL Cleanup`

## Remaining Limitations

This project resolved the planned tickets, but a few constraints remain:

- raw-mode declaration-style top-level `await` is still unsupported by design
- the timeout/recovery path now works for the current runtime architecture, but any future runtime ownership redesign should re-verify the interrupt contract
- the original timing-sensitive timeout test should still be watched in CI even though the broader validation passed repeatedly

## Overall Assessment

This project was successful.

The bug-fix portion materially improved correctness in persistence and evaluation behavior. The refactor portion materially improved readability and change safety without bundling hidden semantics changes into cleanup commits. The final state is substantially stronger than the reviewed starting point: the persistence layer has clearer invariants, the evaluation layer has a real timeout-and-recovery story, and the session subsystem now has an architecture that is easier to reason about in review.

## Commit Summary

Implemented code changes on this branch:

- `30eab79` `fix(repldb): hide deleted sessions from normal reads`
- `9213f13` `fix(replsession): use generated default session ids`
- `0ec8f0b` `fix(repldb): configure sqlite integrity at open time`
- `055814a` `feat(replsession): add eval policy deadlines for awaited promises`
- `4c2399e` `feat(replsession): interrupt synchronous evaluations on timeout`
- `4fcce77` `refactor(replsession): extract evaluation pipeline from service`
- `cb2d952` `refactor(replsession): extract persistence helpers from service`
- `5ab661b` `refactor(replsession): extract observation and summary helpers`
- `a35fc04` `refactor(replapi): clarify session option and evaluator boundaries`
- `96264e0` `test(replapi): cover create-session option resolution`

Supporting documentation and ticket updates were committed alongside them.
