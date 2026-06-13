---
Title: Investigation diary
Ticket: GOJA-XGOJA-V2-RUNTIME-001
Status: active
Topics:
    - xgoja
    - architecture
    - code-generation
DocType: reference
Intent: ""
Owners: []
RelatedFiles:
    - Path: cmd/xgoja/internal/generate/templates.go
      Note: Step 3 preserves command sources during current generator bridge (commit 556ed5c)
    - Path: cmd/xgoja/root_test.go
      Note: Step 3 generated-binary regression and temporary fixture helper (commit 556ed5c)
    - Path: pkg/xgoja/app/command_providers.go
      Note: Step 3 passes filtered source set to provider command contexts (commit 556ed5c)
    - Path: pkg/xgoja/app/jsverb_sources.go
      Note: Step 3 command-scoped JS verb source filtering (commit 556ed5c)
    - Path: pkg/xgoja/app/runtime_spec.go
      Note: Step 3 interim source IDs on command-provider metadata (commit 556ed5c)
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

This diary records the investigation and documentation work for replacing the legacy xgoja generated runtime metadata bridge with a v2-native runtime plan.

## Step 1: Create ticket and write the hard-cutover design guide

I created a dedicated docmgr ticket under `go-go-goja/ttmp` to capture the architecture analysis and implementation guide for an xgoja v2-native runtime cutover. The design focuses on removing the legacy runtime metadata bridge entirely rather than patching it or keeping a parallel compatibility path.

The investigation started from a concrete failure in the sessionstream xgoja chatbot demo: v2 YAML allowed `commands[].sources`, but generated runtime metadata dropped that field while converting the v2 command into legacy `commandProviders`. I used that symptom to map the current pipeline from `specv2` through the planner, generator, `pkg/xgoja/app`, provider command-set setup, and the HTTP serve provider.

### Prompt Context

**User prompt (verbatim):** "create a new docmgr ticket in go-go-goja/ttmp with `docmgr --root go-go-goja/ttmp ...` and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a go-go-goja ticket, investigate the xgoja v2 runtime bridge architecture deeply, write an intern-facing design/implementation guide with evidence, diagrams, pseudocode, API references, file references, and upload the final bundle to reMarkable.

**Inferred user intent:** Preserve the architectural reasoning in a structured ticket so a new engineer can implement the xgoja v2-native runtime cutover without rediscovering the system from scratch.

**User prompt (verbatim):** "make sure there is no legacy left and that all docs will be updated and the migration guide also gets updated."

**Assistant interpretation:** Strengthen the guide so the target is a hard cutover with no legacy runtime path left behind and with documentation/migration updates as part of the definition of done.

**Inferred user intent:** Avoid an incremental compatibility shim that leaves the confusing v2-to-legacy bridge in place; require a clean implementation and complete documentation refresh.

**User prompt (verbatim):** "we can do a hard cutover, no need to keep a legacy path and compare"

**Assistant interpretation:** Remove earlier dual-plan/compare migration language and document a direct replacement strategy.

**Inferred user intent:** Keep the implementation simpler and more decisive by deleting the obsolete bridge instead of maintaining a temporary parallel runtime pipeline.

### What I did

- Created ticket `GOJA-XGOJA-V2-RUNTIME-001` with docmgr under `go-go-goja/ttmp`.
- Added the primary design document:
  - `design-doc/01-xgoja-v2-native-runtime-plan-design-and-implementation-guide.md`
- Added this investigation diary.
- Gathered evidence from:
  - `cmd/xgoja/internal/specv2/types.go`
  - `cmd/xgoja/internal/specv2/defaults.go`
  - `cmd/xgoja/internal/plan/plan.go`
  - `cmd/xgoja/internal/generate/templates.go`
  - `cmd/xgoja/internal/generate/gomod.go`
  - `pkg/xgoja/app/runtime_spec.go`
  - `pkg/xgoja/app/host.go`
  - `pkg/xgoja/app/root.go`
  - `pkg/xgoja/app/command_providers.go`
  - `pkg/xgoja/app/jsverb_sources.go`
  - `pkg/xgoja/providerapi/module.go`
  - `pkg/xgoja/providers/http/serve.go`
  - `cmd/xgoja/doc/*`
  - `examples/xgoja/*`
- Wrote a long-form intern-facing guide covering:
  - current architecture;
  - current v2 schema/planner/generator/app-runtime split;
  - exact gap analysis;
  - hard-cutover architecture;
  - proposed runtime plan DTOs;
  - source registry API;
  - provider command-set API;
  - generated binary/runtime-package shape;
  - hard removal targets;
  - documentation update requirements;
  - decision records;
  - phased implementation plan;
  - testing strategy;
  - file reference map.
- Added docmgr tasks for setup, evidence gathering, guide writing, bookkeeping, validation, and reMarkable upload.

### Why

- The existing xgoja v2 YAML schema is already conceptually correct, but the generated runtime still speaks an older DTO shape.
- The bridge has now caused a real behavior bug: provider command-set `sources` are present in v2 YAML but dropped before runtime command construction.
- A new intern needs a coherent map of all relevant code paths before attempting a hard cutover, because the work spans spec parsing, planning, generation, app runtime construction, provider APIs, examples, and docs.

### What worked

- `docmgr --root go-go-goja/ttmp ticket create-ticket` created the ticket workspace successfully.
- `docmgr --root go-go-goja/ttmp doc add` created the design doc and diary.
- Evidence gathering with `nl -ba` and `rg` produced concrete line references showing:
  - v2 schema fields already exist;
  - planner keeps command plans;
  - generator reshapes v2 into legacy fields;
  - legacy runtime DTO lacks command-provider source bindings;
  - HTTP serve provider requires configured jsverb sources.
- The final guide explicitly treats documentation and migration-guide updates as part of implementation completion.

### What didn't work

- The current generated xgoja runtime path cannot support the desired sessionstream chatbot server through the provider command-set `serve` flow without either patching the bridge or replacing it. The design therefore documents the replacement path rather than pretending the current path is sufficient.
- I initially considered a dual-plan transition strategy, but the user clarified that a hard cutover is preferred. I updated the design to reject dual runtime paths and comparison-mode migration.

### What I learned

- The xgoja v2 planner is much closer to the desired architecture than the generated runtime layer is.
- The lossy part is concentrated around the conversion from `plan.Plan` / `specv2.Config` into `pkg/xgoja/app.RuntimeSpec` and then the runtime app builder's dependence on that legacy DTO.
- The codebase already has enough abstractions to make a clean cutover practical: provider graph, source graph, workspace resolution, runtime factory, host services, and provider command sets are all present.

### What was tricky to build

- The tricky part was distinguishing "legacy user config" from "legacy generated runtime metadata." The user-facing schema is already v2; the legacy part is the internal generated runtime DTO and app builder path.
- Another tricky part was avoiding a small tactical fix. Adding `Sources []string` to `CommandProviderInstanceSpec` would fix the immediate bug, but it would leave the confusing architecture intact. The final design explicitly documents that this ticket should remove the bridge rather than extend it.
- The documentation plan also needed to be strict: if the implementation changes internals but leaves the migration guide and examples using old terms, future contributors will continue learning the wrong model.

### What warrants a second pair of eyes

- Whether the new runtime plan should live in `pkg/xgoja/app` directly or in a temporary internal package during implementation. The design recommends ending in `pkg/xgoja/app` with no permanent `appv2` parallel path.
- Whether the generated runtime JSON schema marker should be `xgoja/runtime/v2` or another name.
- Whether any external provider command-set users need a short migration notice beyond the in-repo docs.

### What should be done in the future

- Implement the hard cutover in go-go-goja.
- Finish the real sessionstream xgoja chatbot server once provider command-set `sources` survive to runtime.
- Add tests that fail if generated v2 runtime metadata reintroduces legacy fields such as `commandProviders`, top-level `jsverbs`, or `packages`.

### Code review instructions

- Start with the design doc's "Current-state architecture" and "Gap analysis" sections.
- Then inspect these files in order:
  1. `cmd/xgoja/internal/specv2/types.go`
  2. `cmd/xgoja/internal/plan/plan.go`
  3. `cmd/xgoja/internal/generate/templates.go`
  4. `pkg/xgoja/app/runtime_spec.go`
  5. `pkg/xgoja/app/command_providers.go`
  6. `pkg/xgoja/providers/http/serve.go`
- Validate the docs with:

```bash
docmgr --root go-go-goja/ttmp doctor --ticket GOJA-XGOJA-V2-RUNTIME-001 --stale-after 30
```

### Technical details

The key lossy conversion is the provider command-set case in `cmd/xgoja/internal/generate/templates.go`. The v2 command has `Sources []string`, but the legacy runtime DTO does not. The design recommends deleting this conversion path entirely and replacing runtime command construction with a loop over v2-native command plans.

Target flow:

```text
xgoja/v2 YAML
  -> specv2 validation/defaults
  -> plan.Compile
  -> generated xgoja.v2.runtime.json
  -> app.RuntimePlan
  -> v2-native host/root/runtime construction
```

Rejected flow:

```text
xgoja/v2 YAML
  -> plan.Compile
  -> legacy RuntimeSpec { packages, modules, commandProviders, jsverbs }
  -> app.Host legacy command builders
```


## Step 2: Reproduce provider command-set source loss

I started the implementation work by creating a dedicated branch and then writing a small reproduction script for the first real xgoja v2 runtime bug. The script builds a temporary xgoja/v2 app with two jsverb source sets, attaches only one source set to an HTTP `provider.command-set`, and inspects the generated metadata.

The reproduction proves the architectural gap without introducing a permanently failing test yet: `commands[].sources` is present in the v2 YAML, but the generated legacy `commandProviders` metadata has no source binding, while both jsverb sources remain in a global top-level `jsverbs` list. This is exactly the bridge behavior the cutover must remove.

### Prompt Context

**User prompt (verbatim):** "then work on the tasks one by one, keeping a detailed diary as you work, commit at appropriate intervals."

**Assistant interpretation:** Begin executing the newly-created xgoja v2 hard-cutover ticket task list sequentially, update the diary as work proceeds, and make focused commits.

**Inferred user intent:** Move from design into implementation while preserving reviewable history and continuation context.

**Commit (code):** 64523c433117c1e84f4ef11f59189c0b69af95c1 — "Docs: reproduce xgoja command source loss"

### What I did

- Renamed the local go-go-goja branch from the deleted post-merge `task/goja-sessionstream` branch to `task/xgoja-v2-runtime-cutover`.
- Marked task 7 complete after recording the dedicated branch baseline.
- Ran the existing HTTP serve xgoja example smoke test:

```bash
make -C examples/xgoja/13-http-serve-jsverbs smoke
```

- Added `scripts/01-reproduce-provider-command-source-loss.sh` under the ticket workspace.
- The script creates a temporary xgoja/v2 spec with:
  - two jsverb sources, `site-a` and `site-b`;
  - one HTTP provider command set with `sources: [site-a]`;
  - an embedded binary artifact containing both sources.
- The script builds with a fixed temporary work directory and inspects generated `xgoja.gen.json`.
- Marked task 8 complete and updated the changelog.

### Why

- Task 8 asks for a concrete reproduction of the provider command-set source bug before changing runtime internals.
- A script is appropriate at this stage because it captures the current behavior without committing an intentionally failing unit test into the branch before the implementation exists.
- The reproduction gives future tests exact expected facts to assert after the v2-native runtime plan lands.

### What worked

The existing HTTP serve example still passed:

```bash
make -C examples/xgoja/13-http-serve-jsverbs smoke
```

That is important because it shows the current legacy behavior can still work in simple cases by scanning all global jsverb sources.

The reproduction script then succeeded and printed:

```text
generated command provider metadata: {"id": "serve", "mount": "serve", "name": "serve", "package": "http"}
generated top-level jsverb sources: [{"embed": true, "id": "site-a", "path": "xgoja_embed/jsverbs/site_a"}, {"embed": true, "id": "site-b", "path": "xgoja_embed/jsverbs/site_b"}]
OK: reproduced source loss: command sources [site-a] were dropped, while both jsverb sources remain global
```

### What didn't work

The first version of the reproduction script computed the repository root incorrectly and ascended one directory too far. It failed with:

```text
go: go.mod file not found in current directory or any parent directory; see 'go help modules'
```

I fixed the script's `repo_root` calculation from seven parent traversals to six parent traversals from the ticket `scripts/` directory.

### What I learned

- The current HTTP serve example passes because command providers see every runtime jsverb source through the legacy global `RuntimeSpec.JSVerbs` field.
- The deeper v2 bug is source scoping, not total inability to run HTTP serve.
- A minimal repro needs at least two source sets. If there is only one source set, global scanning hides the bug.

### What was tricky to build

- The reproduction needed to demonstrate an architectural data-loss bug without relying on fragile CLI help output. Inspecting generated `xgoja.gen.json` is more precise: it shows `commandProviders[0]` lacks `sources` and that both sources are global.
- The temporary xgoja fixture must use an absolute path for source directories because the generated build runs from a separate work directory.

### What warrants a second pair of eyes

- Whether the reproduction script should remain as a ticket artifact only or be promoted into a permanent regression test once the v2 runtime plan implementation starts.
- Whether future tests should assert both metadata shape and CLI command visibility. The design likely needs both: metadata guard tests catch legacy reintroduction, while CLI tests prove end-user behavior.

### What should be done in the future

- Task 9 should add active regression tests that initially fail against current runtime behavior and later pass with the v2-native runtime plan.
- Task 10 should add generated metadata guard assertions that fail if legacy `commandProviders`, `packages`, or top-level `jsverbs` return.

### Code review instructions

- Review `scripts/01-reproduce-provider-command-source-loss.sh`.
- Run it from `go-go-goja`:

```bash
ttmp/2026/06/13/GOJA-XGOJA-V2-RUNTIME-001--replace-legacy-xgoja-runtime-metadata-bridge-with-v2-native-runtime-plan/scripts/01-reproduce-provider-command-source-loss.sh
```

- Confirm it prints that command sources were dropped and both jsverb sources remain global.

### Technical details

The temporary v2 command in the reproduction is:

```yaml
commands:
  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [site-a]
```

The generated metadata currently becomes:

```json
{
  "commandProviders": [
    { "id": "serve", "package": "http", "name": "serve", "mount": "serve" }
  ],
  "jsverbs": [
    { "id": "site-a" },
    { "id": "site-b" }
  ]
}
```

The missing `sources: ["site-a"]` on the command provider is the data-loss boundary.

## Step 3: Add command-scoped HTTP serve regression coverage and interim source propagation

I turned the ticket reproduction into an active end-to-end regression: a generated xgoja binary now builds with two jsverb sources, mounts the HTTP `serve` provider command with `sources: [site-a]`, and proves the generated CLI exposes only the selected site while still carrying the HTTP serve flags. This gives us a behavior-level guard for the user-visible failure mode before the larger RuntimePlan hard cutover begins.

I also made the smallest runtime-path change needed for that regression to pass: v2 command `sources` are preserved into the current generated command-provider metadata, and provider command contexts receive a filtered JS verb source set. This is deliberately an interim stop on the way to the no-legacy RuntimePlan shape; it fixes the immediate source-loss bug without changing the final hard-cutover requirement.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue the xgoja v2 runtime cutover implementation from task 9, adding regression coverage and committing at an appropriate interval.

**Inferred user intent:** Keep advancing the ticket task list one step at a time while preserving a detailed implementation diary and focused commits.

**Commit (code):** 556ed5cb31b55bab893e2e7641a2a7442e3ef2da — "Fix xgoja command-scoped jsverb sources"

### What I did

- Added `TestBuildCommandProviderServeUsesCommandScopedSources` in `cmd/xgoja/root_test.go`.
- The test builds a generated binary from a temporary xgoja/v2 spec with two jsverb sources and one HTTP `provider.command-set` scoped to `site-a`.
- The test asserts `scoped-serve serve sitea start --help --long-help` includes `Start A`, `--http-listen`, and `--hot-reload`.
- The test asserts `scoped-serve serve --help` does not expose `siteb` / `Start B`.
- Added `Sources []string` to `app.CommandProviderInstanceSpec`.
- Updated `cmd/xgoja/internal/generate/templates.go` so `applyPlanRuntimeCommand` preserves `command.Sources` for provider command sets.
- Added `newScopedJSVerbSourceSet` and used it from `Host.newCommandSet` so command providers see only their selected JS verb sources when sources are declared.
- Marked ticket task 9 complete and updated the ticket changelog.

### Why

- The reproduction script proved metadata data loss, but it did not prove the generated end-user CLI behavior.
- HTTP `serve` is the real blocked use case for the sessionstream chatbot demo, so the regression needs to verify both jsverb command visibility and HTTP serve flag propagation.
- The test intentionally uses two sources so global source scanning cannot hide source-scope bugs.

### What worked

- The first version of the test failed before the fix because `site-b` was visible through the HTTP serve command even though the v2 command declared `sources: [site-a]`.
- After preserving command source IDs and filtering the JS verb source set, the targeted regression passed.
- The broader focused validation passed:

```text
go test ./cmd/xgoja ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./pkg/xgoja/providers/http -count=1
ok  	github.com/go-go-golems/go-go-goja/cmd/xgoja	14.645s
ok  	github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate	0.104s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/app	0.659s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http	0.478s
```

- The pre-commit hook also ran lint, generation, and `go test ./...` successfully before creating commit `556ed5c`.

### What didn't work

The initial test used provider ID `http`, but the HTTP provider registers itself under package ID `go-go-goja-http`. That produced an error stub instead of real serve subcommands:

```text
Error: unknown command provider http.serve
unknown command provider http.serve
```

After switching the v2 spec provider ID, runtime module provider, and command provider to `go-go-goja-http`, the generated serve commands appeared.

The next version checked short help for `--http-listen`, but Glazed hides detailed flags behind long help. The failure was:

```text
expected scoped serve help to contain "--http-listen", got:
# start - Start A
...
Use scoped-serve serve sitea start --help --long-help for information about all flags.
```

I changed the assertion to call `--help --long-help` for the leaf command.

A final test-shape issue was that asking Cobra/Glazed for `serve siteb start --help` returned parent `serve` help rather than a hard error. I changed the negative assertion to inspect `serve --help` and ensure the unselected source is absent from the command tree.

### What I learned

- Provider IDs in xgoja specs must match the provider package ID registered by the provider, not an arbitrary local alias, in the current runtime path.
- Glazed command help exposes detailed provider-added flags only in long-help output.
- Behavioral CLI tests are useful but must avoid relying on Cobra's exact unknown-argument error behavior; command-tree visibility is a more stable assertion.

### What was tricky to build

- The runtime bug had two layers: generated metadata dropped `commands[].sources`, and the app runtime gave provider command sets a global JS verb source set. Fixing only one layer was insufficient.
- The regression needed to prove HTTP serve flag propagation too. That required checking the leaf command's long help, because short help intentionally hides most flags.
- This fix is intentionally temporary with respect to representation: it extends the legacy bridge enough to preserve source scoping now, but the final ticket still requires replacing `RuntimeSpec`/`CommandProviderInstanceSpec` with the v2-native `RuntimePlan` rather than keeping this shape.

### What warrants a second pair of eyes

- The interim fallback behavior for command providers with no `sources` remains “all JS verb sources”. That preserves current behavior, but the RuntimePlan design should decide whether provider commands should require explicit source selection or keep all-by-kind as a documented default.
- The test currently verifies command-tree absence via `serve --help`; reviewers should confirm this is the most robust user-visible assertion for Glazed/Cobra command mounting.
- The generated metadata still uses legacy fields (`commandProviders`, top-level `jsverbs`, `packages`). This is expected for this step only and must be removed in later hard-cutover tasks.

### What should be done in the future

- Task 10 should add a generated metadata guard that fails while legacy top-level runtime fields remain.
- The upcoming RuntimePlan implementation should replace this interim `Sources` field on `CommandProviderInstanceSpec` with a v2-native command/source representation.
- Provider command context should eventually carry command identity and a unified source registry, not just the legacy `JSVerbSourceSet` adapter.

### Code review instructions

- Start with `cmd/xgoja/root_test.go`, especially `TestBuildCommandProviderServeUsesCommandScopedSources` and `writeHTTPServeScopedSourcesSpec`.
- Then review `cmd/xgoja/internal/generate/templates.go` at `applyPlanRuntimeCommand` to confirm v2 command sources are preserved.
- Then review `pkg/xgoja/app/command_providers.go` and `pkg/xgoja/app/jsverb_sources.go` to confirm provider command contexts receive filtered sources.
- Validate with:

```bash
go test ./cmd/xgoja ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./pkg/xgoja/providers/http -count=1
```

### Technical details

The regression's core v2 command shape is:

```yaml
commands:
  - id: serve
    type: provider.command-set
    provider: go-go-goja-http
    name: serve
    mount: serve
    sources: [site-a]
```

The generated binary must expose:

```text
scoped-serve serve sitea start --help --long-help
```

and must not expose `siteb` through:

```text
scoped-serve serve --help
```
