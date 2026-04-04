# Tasks

## TODO

- [x] Create the ticket workspace and scaffold the primary design and diary documents
- [x] Analyze `cmd/js-repl`, `cmd/goja-repl`, the Bobatea adapter, the old evaluator stack, and `replapi`
- [x] Write the detailed analysis / design / implementation guide for the migration and binary unification
- [x] Write the initial diary entry capturing the evidence and main design decisions
- [ ] Implement a `replapi`-backed Bobatea adapter for execution/session ownership
- [ ] Extract completion/help behavior into a separate assistance provider or equivalent helper layer
- [ ] Add `goja-repl tui` and move Bubble Tea startup under the unified root command
- [ ] Remove or replace the standalone `cmd/js-repl` entrypoint
- [ ] Clean up old evaluator-owned execution code that is no longer needed
- [ ] Add focused adapter/TUI/persistent-flow validation coverage
- [ ] Run `docmgr doctor` cleanly and keep ticket bookkeeping current as implementation lands
- [ ] Upload the finished ticket bundle to reMarkable
