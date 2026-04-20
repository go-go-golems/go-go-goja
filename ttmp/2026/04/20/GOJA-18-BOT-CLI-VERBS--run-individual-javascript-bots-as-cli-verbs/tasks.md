# Tasks

## Completed

- [x] Create ticket `GOJA-18-BOT-CLI-VERBS` for the requested bot CLI verbs work.
- [x] Inspect the current `go-go-goja` `jsverbs` architecture and identify the scan -> describe -> invoke pipeline.
- [x] Inspect `loupedeck` for reusable repository bootstrap, duplicate detection, and runtime command wrapper patterns.
- [x] Compare the `jsverbs` path to the new sandbox `defineBot(...)` path and document the boundary clearly.
- [x] Write a detailed intern-friendly design / analysis / implementation guide with diagrams, pseudocode, API references, and file references.
- [x] Write a quick-reference command surface and API companion document.
- [x] Update the ticket diary, index, and changelog.
- [x] Relate the key source files to the focused ticket docs.
- [x] Validate the ticket with `docmgr doctor`.
- [x] Upload the final ticket bundle to reMarkable and verify the remote path.
- [x] Add a real example bot repository under `examples/bots` that exercises the major bot CLI features end to end.
- [x] Add a smoke-test playbook with exact commands for validating the example repository manually.

## Recommended implementation phases

### Phase 1: Root command and package scaffolding
- [x] Decide whether the canonical binary should be `cmd/go-go-goja` or a transitional example binary.
- [x] Add a `pkg/botcli` package for command orchestration rather than putting UX-specific logic into `pkg/jsverbs`.
- [x] Add a `bots` root command with shared `--bot-repository` flags.

### Phase 2: Repository bootstrap and discovery
- [x] Implement repository normalization and scanning helpers.
- [x] Scan repositories with `jsverbs.ScanDir(...)`.
- [x] Decide whether v1 should require explicit `__verb__` annotations or allow public-function inference.
- [x] Reject duplicate full verb paths with clear source reporting.

### Phase 3: `bots list`
- [x] Implement a sorted list view of discovered verbs.
- [x] Print each verb path with a stable source label.
- [x] Add tests for empty, duplicate, and multi-repository cases.

### Phase 4: `bots run <verb>`
- [x] Implement selector resolution for one requested verb.
- [x] Build a single-verb Glazed/Cobra parser from `CommandDescriptionForVerb(...)`.
- [x] Create a runtime with the registry overlay loader and module roots.
- [x] Invoke the selected verb through `registry.InvokeInRuntime(...)`.
- [x] Render text and structured output correctly.

### Phase 5: `bots help <verb>`
- [x] Build an ephemeral Cobra command from the selected description.
- [x] Reuse the same parser/description source as `bots run`.
- [x] Verify that help text stays aligned with actual runtime parsing.

### Phase 6: Validation and examples
- [x] Add fixture JS files under `testdata/` for bot verbs.
- [x] Add end-to-end tests for list, run, help, async Promise results, and relative `require()`.
- [x] Add a README or help page showing how to author bot scripts with `__verb__`.

## Open follow-ups

- [ ] Decide whether later work should add config/env repository discovery similar to `loupedeck`.
- [ ] Decide whether sandbox-defined bots need a separate wrapper or adapter story for CLI execution.
- [ ] Decide whether `glaze` output mode should remain JSON-first in v1 or be integrated with richer Glazed renderers.
