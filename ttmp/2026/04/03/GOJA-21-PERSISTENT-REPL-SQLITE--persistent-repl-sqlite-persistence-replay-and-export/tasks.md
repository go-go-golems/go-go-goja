# Tasks

## TODO

- [x] Re-read the extracted `replsession` package, the relevant `pkg/jsdoc` packages, and the `GOJA-20` architecture docs.
- [x] Replace the placeholder `GOJA-21` design doc with a detailed implementation plan for SQLite persistence, replay, restore, export, and REPL-authored docs.
- [x] Replace the placeholder task list and diary scaffold with detailed execution artifacts for this ticket.
- [x] Phase 1.1: Add `pkg/repldb` with SQLite open/bootstrap helpers and the initial durable schema.
- [ ] Phase 1.2: Add store APIs for creating/deleting sessions and writing evaluation rows transactionally.
- [ ] Phase 1.3: Extend `pkg/replsession.Service` so live session lifecycle and evaluation flow persist through `pkg/repldb`.
- [ ] Phase 1.4: Persist binding rows, binding version history, and exportability classification for changed globals.
- [ ] Phase 1.5: Persist REPL-authored JSDoc extracted from cell source and associate it with binding versions.
- [ ] Phase 1.6: Add read-side APIs for ordered history loading, structured export, and replay-oriented restore input.
- [ ] Phase 1.7: Add targeted tests for schema bootstrap, persisted evaluations, binding version writes, docs persistence, and replay behavior.
- [ ] Phase 1.8: Commit the implementation in focused slices and record each slice in the diary and changelog.
