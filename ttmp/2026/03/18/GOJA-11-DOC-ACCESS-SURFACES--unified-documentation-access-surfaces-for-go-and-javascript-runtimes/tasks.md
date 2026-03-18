# Tasks

## Phase 1: Ticket Setup And Design Deliverable

- [x] Create GOJA-11 for unified documentation access surfaces.
- [x] Inspect the existing Glazed help, jsdoc, and plugin metadata code paths.
- [x] Write a detailed intern-facing architecture and implementation guide.
- [x] Seed an investigation diary capturing reasoning and context.
- [x] Relate the key files and validate the ticket with `docmgr doctor`.
- [x] Upload the design bundle to reMarkable.

## Phase 2: Architecture Decisions

- [ ] Create a shared `pkg/docaccess` core package with provider and hub abstractions.
- [ ] Define the shared Go-side model for sources, entry refs, entries, and search queries.
- [ ] Decide whether the initial JS module name should be `docs` or a compatibility alias should ship alongside it.
- [ ] Decide whether `modules/glazehelp` becomes a wrapper over the new hub or is deprecated after rollout.

## Phase 3: Source Providers

- [ ] Implement a Glazed help provider over `*help.HelpSystem`.
- [ ] Implement a jsdoc provider over `*model.DocStore`.
- [ ] Implement a plugin metadata provider over retained plugin manifests rather than just CLI summaries.
- [ ] Document what plugin metadata cannot yet be represented because method-level docs are not in the protobuf contract.

## Phase 4: Runtime And JavaScript Surface

- [ ] Add a runtime-scoped registrar that builds a documentation hub for each runtime.
- [ ] Add a new native module for JavaScript-side documentation access.
- [ ] Decide how the line REPL and `js-repl` should surface this module or related built-in commands.

## Phase 5: Validation And Future Expansion

- [ ] Add Go tests for the shared hub and provider adapters.
- [ ] Add integration tests for JS-side access through `require("docs")`.
- [ ] Revisit optional `docmgr` provider support after the core three sources are stable.
- [ ] Refresh this ticket with implementation notes once work begins.
