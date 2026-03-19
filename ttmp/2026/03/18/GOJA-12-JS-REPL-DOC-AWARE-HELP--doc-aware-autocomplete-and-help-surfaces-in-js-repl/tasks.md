# Tasks

## Phase 1: Ticket Setup And Architectural Grounding

- [x] Create GOJA-12 for doc-aware js-repl autocomplete and help.
- [x] Inspect the current evaluator, docs hub, plugin provider, and runtime ownership seams.
- [x] Write a detailed intern-facing design and implementation guide.
- [x] Seed an investigation diary with the design rationale and repository evidence.

## Phase 2: Runtime State And Ownership Design

- [x] Decide whether to persist the raw docs hub or a narrower resolver on the owned runtime.
- [x] Add stable runtime-scoped value access on `engine.Runtime`.
- [x] Teach the runtime factory to preserve runtime module context values after setup.
- [x] Persist the docs hub under a named runtime-scoped key during docs-module registration.

## Phase 3: Evaluator-Side Documentation Resolution

- [x] Add a docs resolver helper near the JavaScript evaluator.
- [x] Implement module-level doc resolution for plugin modules through `require()` aliases.
- [x] Implement export-level doc resolution for plugin object exports.
- [x] Implement method-level doc resolution for plugin methods.
- [x] Keep the resolver API generic enough to admit future non-plugin module docs.

## Phase 4: UI Surface Integration

- [x] Enrich autocomplete suggestion text with docs-derived summaries when available.
- [x] Prefer docs-derived summaries in the help bar over generic candidate detail strings.
- [x] Render full documentation bodies in the help drawer for resolved docs entries.
- [x] Preserve current static-signature and runtime-inspection fallbacks for unsupported contexts.

## Phase 5: Validation And Coverage

- [x] Add focused resolver tests for module, export, and method lookup.
- [x] Add integration tests for doc-aware autocomplete display text.
- [x] Add integration tests for doc-aware help-bar output.
- [x] Add integration tests for doc-aware help-drawer markdown.
- [x] Add at least one failure/edge-case test for unknown aliases or unsupported objects.

## Phase 6: Broader Module-Docs Follow-Up

- [x] Decide how native module docs like `fs`, `path`, `url`, and `docs` should plug into the same resolver seam.
- [x] Decide whether source IDs such as `plugin-manifests` should be centralized before broader rollout.
- [x] Document what remains intentionally out of scope for the first implementation slice.

## Phase 7: Ticket Closeout

- [x] Relate the implementation files back into the GOJA-12 docs once work starts.
- [x] Run `docmgr doctor --ticket GOJA-12-JS-REPL-DOC-AWARE-HELP --stale-after 30`.
- [x] Upload the refreshed bundle to reMarkable and verify the remote listing.
