# Tasks

## Phase 1: Ticket Setup And Design Deliverable

- [x] Create GOJA-11 for unified documentation access surfaces.
- [x] Inspect the existing Glazed help, jsdoc, and plugin metadata code paths.
- [x] Write a detailed intern-facing architecture and implementation guide.
- [x] Seed an investigation diary capturing reasoning and context.
- [x] Relate the key files and validate the ticket with `docmgr doctor`.
- [x] Upload the design bundle to reMarkable.

## Phase 2: Architecture Decisions

- [x] Decide that the initial JS module name is `docs`.
- [x] Decide not to keep backward compatibility for plugin method docs; update the manifest contract directly.
- [ ] Create a shared `pkg/docaccess` core package with provider and hub abstractions.
- [ ] Define the shared Go-side model for sources, entry refs, entries, and search queries.
- [ ] Decide whether `modules/glazehelp` becomes a wrapper over the new hub or is deprecated after rollout.

## Phase 3: Source Providers

- [ ] Extend `pkg/hashiplugin/contract/jsmodule.proto` with first-class method doc metadata and regenerate protobuf code.
- [ ] Update the plugin SDK manifest builder and validation rules for method docs.
- [ ] Update plugin host reporting/reification support to preserve the richer manifest metadata in memory.
- [ ] Implement a Glazed help provider over `*help.HelpSystem`.
- [ ] Implement a jsdoc provider over `*model.DocStore`.
- [ ] Implement a plugin metadata provider over retained plugin manifests rather than just CLI summaries.
- [ ] Expose module, export, and method entries from the plugin provider.

## Phase 4: Runtime And JavaScript Surface

- [ ] Add a runtime-scoped registrar that builds a documentation hub for each runtime.
- [ ] Add a new native module for JavaScript-side documentation access.
- [ ] Wire the docs module into `repl` runtime setup.
- [ ] Wire the docs module into `js-repl` evaluator/runtime setup.
- [ ] Decide how the line REPL and `js-repl` should surface this module or related built-in commands.

## Phase 5: Validation And Future Expansion

- [ ] Add Go tests for the shared hub and provider adapters.
- [ ] Add tests for the new plugin method-doc manifest shape.
- [ ] Add integration tests for JS-side access through `require("docs")`.
- [ ] Revisit optional `docmgr` provider support after the core three sources are stable.
- [ ] Refresh this ticket with implementation notes once work begins.
