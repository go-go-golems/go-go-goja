# Tasks

## TODO

- [x] Re-read the current ticket material and the staged prototype code.
- [x] Re-analyze the architecture with CLI and server as the primary delivery surfaces.
- [x] Produce a replacement design guide that treats SQLite persistence as a first-class subsystem.
- [x] Add a REPL-scoped JSDoc strategy for binding documentation and docs queries.
- [x] Update the ticket index, changelog, and diary to reflect the new recommended design.
- [x] Upload the refreshed ticket bundle to reMarkable.
- [x] Phase 1.1: Finalize the extraction boundary for the shared session kernel and map which `pkg/webrepl` files move now versus later.
- [x] Phase 1.2: Create the new shared package for transport-neutral session/report types and rewrite helpers.
- [x] Phase 1.3: Move the persistent session service implementation out of `pkg/webrepl` into the shared package.
- [x] Phase 1.4: Rewire the current `pkg/webrepl` HTTP layer so it depends on the shared package instead of owning the session logic.
- [x] Phase 1.5: Rewire `cmd/web-repl` to use the shared package directly through the HTTP transport.
- [x] Phase 1.6: Run targeted build/test validation for the extracted packages and fix any regressions caused by the move.
- [x] Phase 1.7: Commit the code refactor in focused commits and record each step in the diary and changelog.
- [ ] Create follow-on ticket for SQLite persistence, replay/restore, and export.
- [ ] Create follow-on ticket for the new CLI and JSON server surfaces after the phase-1 extraction lands.
