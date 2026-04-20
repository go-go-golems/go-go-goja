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

## Recommended implementation phases

### Phase 1: Root command and package scaffolding
- [ ] Decide whether the canonical binary should be `cmd/go-go-goja` or a transitional example binary.
- [ ] Add a `pkg/botcli` package for command orchestration rather than putting UX-specific logic into `pkg/jsverbs`.
- [ ] Add a `bots` root command with shared `--bot-repository` flags.

### Phase 2: Repository bootstrap and discovery
- [ ] Implement repository normalization and scanning helpers.
- [ ] Scan repositories with `jsverbs.ScanDir(...)`.
- [ ] Decide whether v1 should require explicit `__verb__` annotations or allow public-function inference.
- [ ] Reject duplicate full verb paths with clear source reporting.

### Phase 3: `bots list`
- [ ] Implement a sorted list view of discovered verbs.
- [ ] Print each verb path with a stable source label.
- [ ] Add tests for empty, duplicate, and multi-repository cases.

### Phase 4: `bots run <verb>`
- [ ] Implement selector resolution for one requested verb.
- [ ] Build a single-verb Glazed/Cobra parser from `CommandDescriptionForVerb(...)`.
- [ ] Create a runtime with the registry overlay loader and module roots.
- [ ] Invoke the selected verb through `registry.InvokeInRuntime(...)`.
- [ ] Render text and structured output correctly.

### Phase 5: `bots help <verb>`
- [ ] Build an ephemeral Cobra command from the selected description.
- [ ] Reuse the same parser/description source as `bots run`.
- [ ] Verify that help text stays aligned with actual runtime parsing.

### Phase 6: Validation and examples
- [ ] Add fixture JS files under `testdata/` for bot verbs.
- [ ] Add end-to-end tests for list, run, help, async Promise results, and relative `require()`.
- [ ] Add a README or help page showing how to author bot scripts with `__verb__`.

## Open follow-ups

- [ ] Decide whether later work should add config/env repository discovery similar to `loupedeck`.
- [ ] Decide whether sandbox-defined bots need a separate wrapper or adapter story for CLI execution.
- [ ] Decide whether `glaze` output mode should remain JSON-first in v1 or be integrated with richer Glazed renderers.
