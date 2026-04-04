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

- [ ] Refactor `cmd/goja-repl` app construction so TUI can choose `interactive` or `persistent` profiles cleanly
- [ ] Add `goja-repl tui` under the existing Cobra/Glazed root command
- [ ] Port the current `cmd/js-repl` Bubble Tea startup into the new subcommand
- [ ] Support startup flags for profile, session selection, and alt-screen behavior
- [ ] Add command-level tests where practical

## Phase 4: Cutover and Cleanup

- [ ] Switch the TUI path to the new `replapi`-backed adapter
- [ ] Remove the standalone `cmd/js-repl` entrypoint
- [ ] Remove old evaluator-owned execution code that is no longer needed
- [ ] Update docs/help text/examples to point users at `goja-repl tui`

## Phase 5: Validation and Ticket Hygiene

- [ ] Run targeted Go tests for adapter, assistance, and CLI surfaces
- [ ] Run end-to-end TUI smoke tests under `tmux`
- [ ] Update the diary after each material implementation step
- [ ] Update changelog/tasks as commits land
- [ ] Run `docmgr doctor --ticket GOJA-24-REPL-TUI-UNIFICATION --stale-after 30`
- [ ] Upload the final ticket bundle to reMarkable

## Completed Discovery

- [x] Create the ticket workspace and scaffold the primary design and diary documents
- [x] Analyze `cmd/js-repl`, `cmd/goja-repl`, the Bobatea adapter, the old evaluator stack, and `replapi`
- [x] Write the detailed analysis / design / implementation guide for the migration and binary unification
- [x] Write the initial diary entry capturing the evidence and main design decisions
