# Tasks

## Phase 1: Adapter Boundary

- [x] Add a narrow live-runtime access hook through `replsession` / `replapi` for frontend assistance use
- [x] Add a new Bobatea adapter backed by `replapi.App` and one session ID
- [x] Map `EvaluateResponse` into Bobatea timeline events with console/error/result parity
- [x] Add focused adapter tests for evaluation, console output, repeated session use, and runtime auto-restore

## Phase 2: Assistance Split

- [x] Extract completion/help logic from the old JavaScript evaluator into a dedicated assistance provider
- [x] Make the assistance provider consume runtime hints from the live `replapi` session runtime
- [x] Preserve docs-resolver behavior for plugin/module help
- [x] Add focused tests for completion, help bar, and help drawer on the extracted provider

## Phase 3: Unified Binary

- [x] Refactor `cmd/goja-repl` app construction so TUI can choose `interactive` or `persistent` profiles cleanly
- [x] Add `goja-repl tui` under the existing Cobra/Glazed root command
- [x] Port the current `cmd/js-repl` Bubble Tea startup into the new subcommand
- [x] Support startup flags for profile, session selection, and alt-screen behavior
- [x] Add command-level tests where practical

## Phase 4: Cutover and Cleanup

- [x] Switch the TUI path to the new `replapi`-backed adapter
- [x] Remove the standalone `cmd/js-repl` entrypoint
- [x] Remove evaluator-owned execution from the TUI path while keeping the evaluator available for non-TUI callers that still use it
- [x] Update docs/help text/examples to point users at `goja-repl tui`

## Phase 5: Validation and Ticket Hygiene

- [x] Run targeted Go tests for adapter, assistance, and CLI surfaces
- [x] Run end-to-end TUI smoke tests under `tmux`
- [x] Update the diary after each material implementation step
- [x] Update changelog/tasks as commits land
- [x] Run `docmgr doctor --ticket GOJA-24-REPL-TUI-UNIFICATION --stale-after 30`
- [x] Upload the final ticket bundle to reMarkable

## Phase 6: Inspector Assistance Cleanup

- [x] Add an assistance-only Bobatea adapter for existing non-`replapi` runtimes
- [x] Move `smalltalk-inspector` off the full JavaScript evaluator dependency
- [x] Preserve completion, help bar, help drawer, and declaration tracking behavior in the inspector REPL widgets
- [x] Add focused tests for the new assistance-only adapter where practical
- [x] Re-run targeted tests for `smalltalk-inspector` and the Bobatea adapter package

## Phase 7: Legacy Web Prototype Removal

- [x] Remove the `cmd/web-repl` binary
- [x] Remove the `pkg/webrepl` package and embedded static UI assets
- [x] Clean up any current user-facing references that still point at the legacy web prototype instead of `goja-repl serve`
- [x] Re-run full validation after the removal

## Completed Discovery

- [x] Create the ticket workspace and scaffold the primary design and diary documents
- [x] Analyze `cmd/js-repl`, `cmd/goja-repl`, the Bobatea adapter, the old evaluator stack, and `replapi`
- [x] Write the detailed analysis / design / implementation guide for the migration and binary unification
- [x] Write the initial diary entry capturing the evidence and main design decisions
