# Tasks

## Completed

- [x] Create ticket `GOJA-041-EVALUATION-CONTROL`
- [x] Write the evaluation control analysis/design/implementation guide
- [x] Record an investigation diary
- [x] Store numbered retraceable experiment scripts under `scripts/`
- [x] Validate the ticket with `docmgr doctor`
- [x] Upload the ticket bundle to reMarkable

## Planned implementation work

- [x] Define evaluation timeout policy in the session model
- [x] Add timeout-aware execution path for promise waiting
- [x] Add timeout-aware execution path for synchronous runaway raw and wrapped evaluation
- [x] Add VM interruption on timeout for synchronous runaway evaluation
- [x] Ensure session remains usable after timeout
- [x] Add tests for hung code paths
- [x] Add tests for promise waiting and cancellation behavior
- [x] Add tests for raw-mode top-level `await` edge cases

## Resolved blocker

- The earlier standalone `eventloop` repro was not exercising the same runtime ownership path as `replsession`.
- The decisive engine-path repro showed:
  - `sameVM true`
  - `errors.Is(err, interruptErr) == true`
  - the VM is reusable after `ClearInterrupt()`
- GOJA-041 now uses that engine/runtimeowner path directly for synchronous timeout interruption.
