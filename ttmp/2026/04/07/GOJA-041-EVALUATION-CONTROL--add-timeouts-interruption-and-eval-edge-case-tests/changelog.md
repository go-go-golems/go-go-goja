# Changelog

## 2026-04-07

- Created ticket `GOJA-041-EVALUATION-CONTROL`.
- Added a detailed design and implementation guide for timeout, interruption, and evaluation edge-case work.
- Scoped the PR around evaluation control only, not broader persistence or cleanup concerns.
- Validated the ticket with `docmgr doctor` and uploaded the bundle to reMarkable.

## 2026-04-08

- Added `timeoutMs` to `replsession.EvalPolicy` and set opinionated default deadlines in the stock raw and interactive session presets.
- Made promise waiting deadline-aware so top-level awaited promises can now time out cleanly with `status == "timeout"`.
- Added raw-mode tests that document the current top-level `await` contract: expression-style `await` works, declaration-style `await` still errors, and a never-settling promise now times out.
- Investigated synchronous runaway-evaluation interruption and recorded a blocker: `goja.Runtime.Interrupt(...)` did not unwind `RunString(...)` under the current `goja_nodejs/eventloop` execution model in a direct repro, so the "hang and recover the same session" slice is still unresolved.
- Added numbered retraceable experiment files under [00-scripts-index.md](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/00-scripts-index.md), including the plain-runtime success repro, the `eventloop` failure repro, and the same-VM check that explains the mismatch.
