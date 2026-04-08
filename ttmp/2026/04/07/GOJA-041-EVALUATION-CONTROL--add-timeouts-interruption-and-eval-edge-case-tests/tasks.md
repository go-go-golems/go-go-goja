# Tasks

## Completed

- [x] Create ticket `GOJA-041-EVALUATION-CONTROL`
- [x] Write the evaluation control analysis/design/implementation guide
- [x] Record an investigation diary
- [x] Validate the ticket with `docmgr doctor`
- [x] Upload the ticket bundle to reMarkable

## Planned implementation work

- [x] Define evaluation timeout policy in the session model
- [x] Add timeout-aware execution path for promise waiting
- [ ] Add timeout-aware execution path for synchronous runaway raw and wrapped evaluation
- [ ] Add VM interruption on timeout for synchronous runaway evaluation
- [ ] Ensure session remains usable after timeout
- [ ] Add tests for hung code paths
- [x] Add tests for promise waiting and cancellation behavior
- [x] Add tests for raw-mode top-level `await` edge cases

## Current blocker

- `goja.Runtime.Interrupt(...)` did not unwind a runaway `RunString(...)` under the repo's `goja_nodejs/eventloop` execution model in either the REPL code path or a standalone repro, so the synchronous hang-recovery portion remains blocked pending a larger design decision.
