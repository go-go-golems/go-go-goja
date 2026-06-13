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
    - Path: cmd/xgoja/internal/generate/generate_test.go
      Note: |-
        Step 4 metadata guard against legacy top-level generated keys (commit 617b977)
        Step 6 workspace.mode:auto replacement regression (commit f09788a)
    - Path: cmd/xgoja/internal/generate/templates.go
      Note: |-
        Step 3 preserves command sources during current generator bridge (commit 556ed5c)
        Step 4 emits v2-native runtime plan JSON (commit 617b977)
    - Path: cmd/xgoja/internal/generate/templates/main.go.tmpl
      Note: Step 4 generated main decodes RuntimePlan (commit 617b977)
    - Path: cmd/xgoja/root_test.go
      Note: Step 3 generated-binary regression and temporary fixture helper (commit 556ed5c)
    - Path: examples/xgoja/13-http-serve-jsverbs/Makefile
      Note: Step 7 runnable HTTP serve smoke target validated (commit 08f1264)
    - Path: examples/xgoja/14-generated-runtime-package/internal/xgojaruntime/xgoja_runtime.gen.go
      Note: Step 4 checked-in runtime package fixture updated for RuntimePlan (commit 617b977)
    - Path: pkg/xgoja/app/assets.go
      Note: Step 6 asset setup through SourceRegistry kind=assets (commit f09788a)
    - Path: pkg/xgoja/app/command_providers.go
      Note: |-
        Step 3 passes filtered source set to provider command contexts (commit 556ed5c)
        Step 5 provider command contexts receive scoped source registries (commit 8bcc367)
    - Path: pkg/xgoja/app/command_providers_test.go
      Note: Step 5 scoped source registry regression coverage (commit 8bcc367)
    - Path: pkg/xgoja/app/framework.go
      Note: Step 6 help loading through SourceRegistry kind=help (commit f09788a)
    - Path: pkg/xgoja/app/host.go
      Note: Step 6 SourceRegistry ownership and CommandPlan dispatch loop (commit f09788a)
    - Path: pkg/xgoja/app/jsverb_sources.go
      Note: Step 3 command-scoped JS verb source filtering (commit 556ed5c)
    - Path: pkg/xgoja/app/root.go
      Note: Step 6 JS verb scanning via SourceRegistry (commit f09788a)
    - Path: pkg/xgoja/app/runtime_spec.go
      Note: |-
        Step 3 interim source IDs on command-provider metadata (commit 556ed5c)
        Step 4 v2 RuntimePlan DTO and transitional decode path (commit 617b977)
    - Path: pkg/xgoja/app/source_registry.go
      Note: Step 5 runtime SourceRegistry implementation (commit 8bcc367)
    - Path: pkg/xgoja/providerapi/commands.go
      Note: Step 5 CommandSetContext carries SourceRegistry (commit 8bcc367)
    - Path: pkg/xgoja/providerapi/sources.go
      Note: Step 5 provider-facing SourceRegistry API (commit 8bcc367)
    - Path: pkg/xgoja/providers/http/serve.go
      Note: Step 7 HTTP serve SourceRegistry and hot reload scoped source cleanup (commit 08f1264)
    - Path: pkg/xgoja/providers/http/serve_test.go
      Note: Step 7 HTTP serve SourceRegistry test contexts (commit 08f1264)
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

## Step 4: Cut generated runtime metadata over to RuntimePlan

I started the hard cutover by moving generated xgoja metadata to a v2-native `RuntimePlan` JSON shape. Generated metadata now emits `schema: "xgoja/runtime/v2"`, `providers`, `runtime.modules`, unified `sources`, and `commands[]`; the generator test now fails if legacy top-level keys such as `packages`, `modules`, `commandProviders`, `jsverbs`, `help`, or `assets` reappear in generated v2 output.

I also moved the generated main/runtime-package templates and the runtime app wiring to decode and consume `app.RuntimePlan`. To keep the repository buildable while migrating older in-repo tests and checked-in generated fixtures, `RuntimePlan` currently has non-emitted compatibility decode paths for old JSON/test literals; these are not produced by the generator and must be removed in the later legacy-removal sweep.

### Prompt Context

**User prompt (verbatim):** "let's do the full cutover. continue. We'll do sessionstream later on, you can break it"

**Assistant interpretation:** Continue with the hard xgoja v2 runtime cutover now, prioritizing go-go-goja and deferring sessionstream compatibility until later.

**Inferred user intent:** Remove the legacy generated runtime metadata bridge even if downstream/sessionstream work has to wait.

**Commit (code):** 617b977 — "Cut over generated xgoja metadata to RuntimePlan"

### What I did

- Replaced generated runtime metadata rendering in `cmd/xgoja/internal/generate/templates.go` with v2-native `app.RuntimePlan` output.
- Added a generated metadata guard in `cmd/xgoja/internal/generate/generate_test.go` that rejects legacy top-level keys: `packages`, `modules`, `commandProviders`, `jsverbs`, `help`, and `assets`.
- Added `app.RuntimePlan`, `AppPlan`, `RuntimeSection`, `RuntimeModulePlan`, `SourcePlan`, `CommandPlan`, and `ArtifactPlan` in `pkg/xgoja/app/runtime_spec.go`.
- Updated generated templates to decode `*app.RuntimePlan`.
- Updated app runtime wiring, source resolution, command providers, asset/help loading, dts generation, HTTP provider tests, host provider tests, and the generated runtime-package fixture to use RuntimePlan-shaped concepts.
- Marked tasks 10, 12, 16, 17, 19, and 21 complete.

### Why

- The source-scoping bug was a symptom of generated v2 specs being flattened into a legacy runtime DTO.
- The hard cutover requires generated output to stop emitting old top-level concepts and preserve v2 concepts directly.
- Starting at the generated metadata boundary gives every later runtime cleanup task a concrete v2 payload to consume.

### What worked

- The focused validation passed before commit:

```text
go test ./cmd/xgoja ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./pkg/xgoja/providers/http -count=1
ok  	github.com/go-go-golems/go-go-goja/cmd/xgoja	9.322s
ok  	github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate	0.038s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/app	0.177s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http	0.288s
```

- The pre-commit hook then passed lint, `go generate ./...`, and `go test ./...`.
- The generated runtime-package example was updated enough to compile against `RuntimePlan`.

### What didn't work

The first commit attempt failed because checked-in generated runtime-package code still referenced removed names:

```text
examples/xgoja/14-generated-runtime-package/internal/xgojaruntime/xgoja_runtime.gen.go:66:19: undefined: app.RuntimeSpec
examples/xgoja/14-generated-runtime-package/internal/xgojaruntime/xgoja_runtime.gen.go:80:25: undefined: app.RuntimeSpec
examples/xgoja/14-generated-runtime-package/internal/xgojaruntime/xgoja_runtime.gen.go:81:22: undefined: app.RuntimeSpec
```

The full pre-commit test pass then also exposed host provider tests still using the old names:

```text
pkg/xgoja/providers/host/host_test.go:58:22: undefined: app.RuntimeSpec
pkg/xgoja/providers/host/host_test.go:59:17: undefined: app.AssetSourceSpec
pkg/xgoja/providers/host/host_test.go:60:18: undefined: app.ModuleInstanceSpec
```

I updated those files to use `RuntimePlan`, `SourcePlan`, and `RuntimeModulePlan` and reran the commit hook successfully.

### What I learned

- The generator boundary can now enforce the most important guarantee: v2-generated runtime JSON no longer emits the legacy top-level metadata shape.
- Some older tests and fixtures still used legacy JSON or struct literals directly. Keeping them buildable required a temporary non-emitted decode/struct compatibility layer inside `RuntimePlan`.
- The generated DTS sidecar template also needed a RuntimePlan update; otherwise v2 generated binaries worked but generated declaration sidecars were empty.

### What was tricky to build

- Go's default JSON unmarshalling made mixed old/new tests fail in both directions: old `commands` objects could not unmarshal into `[]CommandPlan`, and new `commands[]` arrays could not unmarshal into `CommandsSpec`. I added custom decode logic while keeping generated output strictly v2-shaped.
- Runtime module provider identity changed from old `package` terminology to v2 `provider`; test literals and generated code needed to preserve provider IDs through module resolution, dts generation, and provider command setup.
- `go generate ./...` in the pre-commit hook catches checked-in generated examples, so template changes must be reflected in the generated runtime-package fixture before the commit can land.

### What warrants a second pair of eyes

- `pkg/xgoja/app/runtime_spec.go` still contains temporary compatibility fields and decode paths for old in-repo tests/fixtures. These are not emitted in generated JSON, but they must be removed in the later legacy-removal phase.
- The task checklist now has the generated metadata cutover tasks checked, but follow-up work is still needed to replace compatibility decode paths with purely v2 test fixtures.
- Reviewers should verify that `assertNoLegacyRuntimeKeys` is strict enough for the generated v2 JSON contract.

### What should be done in the future

- Remove compatibility fields/decode paths from `RuntimePlan` after all tests and historical fixtures are migrated to v2 JSON/literals.
- Continue with the unified source registry and provider `CommandSetContext` cleanup so command-scoped source handling no longer depends on a JSVerb-only adapter.
- Run a grep guard in CI for active legacy runtime DTO names after the final removal sweep.

### Code review instructions

- Start with `cmd/xgoja/internal/generate/templates.go` and `cmd/xgoja/internal/generate/generate_test.go` to verify the generated metadata contract.
- Then review `pkg/xgoja/app/runtime_spec.go` and the app wiring changes in `host.go`, `root.go`, `factory.go`, `command_providers.go`, `framework.go`, and `assets.go`.
- Validate with:

```bash
go test ./cmd/xgoja ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./pkg/xgoja/providers/http -count=1
go test ./... -count=1
```

### Technical details

Generated runtime JSON now uses this top-level shape:

```json
{
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "app": { "name": "fixture" },
  "target": { "kind": "xgoja", "output": "dist/fixture" },
  "providers": [{ "id": "fixture" }],
  "runtime": { "modules": [{ "provider": "fixture", "name": "hello", "as": "hello" }] },
  "sources": [{ "id": "verbs", "kind": "jsverbs", "path": "xgoja_embed/jsverbs/verbs", "embed": true }],
  "commands": [{ "id": "eval", "type": "builtin.eval", "name": "eval" }]
}
```

The generator test rejects these old top-level keys in v2 output:

```text
packages
modules
commandProviders
jsverbs
help
assets
```

## Step 5: Add v2 runtime SourceRegistry for provider command sets

I added the provider-facing runtime source registry layer so command providers no longer need to reason only in terms of the legacy JSVerb-specific adapter. Provider command contexts now receive a `SourceRegistry` that can list all command-scoped sources, filter by source kind, look up sources by ID, and expose a JS verb adapter for providers such as HTTP serve that still consume JS verb registries.

This step keeps HTTP serve working while moving the provider API toward the v2 model: commands own source selection, the runtime builds a command-scoped registry, and JS verbs become one adapter on top of the generic source registry rather than the only source API.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Continue advancing all xgoja v2 cutover phases, with sessionstream deferred until the go-go-goja runtime path is ready.

**Inferred user intent:** Keep removing the legacy runtime bridge and move provider/source APIs toward the v2-native design.

**Commit (code):** 8bcc367 — "Add xgoja runtime SourceRegistry"

### What I did

- Added `pkg/xgoja/providerapi/sources.go` with:
  - `RuntimeSourceKind`
  - `RuntimeSourceDescriptor`
  - `SourceRegistry`
- Added `pkg/xgoja/app/source_registry.go` with runtime source listing, kind filtering, ID lookup, command scoping, and `JSVerbs()` adapter support.
- Extended `providerapi.CommandSetContext` with `Sources providerapi.SourceRegistry`.
- Updated `Host.newCommandSet` to construct a command-scoped `SourceRegistry` from `CommandPlan.Sources`.
- Kept `ctx.JSVerbs` populated from the scoped registry so existing providers continue working while the API migrates.
- Added test coverage proving provider command sets receive only selected sources through both `ctx.Sources` and the JS verb adapter.
- Marked tasks 22, 26, 32, 33, and 34 complete.

### Why

- The v2 runtime plan represents all sources uniformly. Provider commands should receive that uniform view instead of only a global JS verb set.
- HTTP serve still needs a `JSVerbSourceSet` today, so the registry supplies an adapter while the provider implementation migrates in later phases.
- Command-scoped source filtering is now centralized in one runtime source registry instead of being hand-built per provider context.

### What worked

- Focused tests passed:

```text
go test ./cmd/xgoja ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./pkg/xgoja/providerapi ./pkg/xgoja/providers/http -count=1
```

- The pre-commit hook passed lint, `go generate ./...`, and `go test ./...` before commit `8bcc367` was created.

### What didn't work

The first commit attempt failed lint because the old helper became unused after `SourceRegistry.Scoped` replaced it:

```text
pkg/xgoja/app/jsverb_sources.go:26:6: func newScopedJSVerbSourceSet is unused (unused)
```

I removed that helper and reran the commit hook successfully.

A focused test also initially failed because compatibility-only `RuntimePlan.JSVerbs`/`Assets` fields did not have `Kind` set when converted to the unified source view. I adjusted `RuntimePlan.allSources()` to tag those compatibility sources as `jsverbs` or `assets` before filtering.

### What I learned

- The generic source registry can coexist with the JSVerb adapter cleanly: `SourceRegistry.JSVerbs()` returns a provider-compatible `JSVerbSourceSet` built from the same scoped source list.
- Compatibility fields are still a liability because they require normalization logic. Removing those fields remains important for the final cleanup.
- Centralizing command source filtering makes provider command tests simpler and avoids duplicating filtering behavior across HTTP serve and future providers.

### What was tricky to build

- The registry had to serve both the new provider-neutral API and the old JSVerb scanning API without widening the old API further.
- The source kind information is guaranteed in generated v2 output, but not in old test literals. That mismatch surfaced in tests and required transitional normalization.
- `CommandSetContext` now contains both `Sources` and `JSVerbs`; reviewers should treat `JSVerbs` as an adapter for current providers, not the long-term primary API.

### What warrants a second pair of eyes

- Whether `SourceRegistry.JSVerbs()` should return nil or an empty adapter when no JS verb sources exist. Current behavior returns an adapter over an empty source list.
- Whether provider-facing `RuntimeSourceDescriptor` should expose more origin metadata before docs are finalized.
- The temporary compatibility path in `RuntimePlan.allSources()` should be removed once old tests are migrated.

### What should be done in the future

- Update HTTP serve to consume `ctx.Sources` directly and only use `ctx.JSVerbs` as a compatibility adapter during transition.
- Remove compatibility `RuntimePlan` fields and decode paths.
- Update provider docs to describe `CommandSetContext.Sources` as the primary source API.

### Code review instructions

- Review `pkg/xgoja/providerapi/sources.go` for the provider-facing API.
- Review `pkg/xgoja/app/source_registry.go` for scoping and descriptor conversion.
- Review `pkg/xgoja/app/command_providers.go` to see where each provider command receives a scoped registry.
- Validate with:

```bash
go test ./cmd/xgoja ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./pkg/xgoja/providerapi ./pkg/xgoja/providers/http -count=1
go test ./... -count=1
```

### Technical details

A provider command scoped like this:

```yaml
commands:
  - id: serve
    type: provider.command-set
    provider: go-go-goja-http
    name: serve
    sources: [site-a]
```

now receives a `CommandSetContext` where:

```go
ctx.Sources.SourceByID("site-a") // true
ctx.Sources.SourceByID("site-b") // false
ctx.Sources.ListSourcesByKind(providerapi.RuntimeSourceKindJSVerbs) // only site-a
ctx.JSVerbs.ListJSVerbSources() // only site-a
```

## Step 6: Finish requested Phase 2–4 RuntimePlan source consumer cleanup

I completed the requested Phase 2 cleanup, Phase 3 source-registry consumer migration, and Phase 4 runtime app cleanup slice. The runtime host now owns a `SourceRegistry`, JS verb command construction scans through that registry, help and asset setup consume registry-filtered source descriptors, and default command attachment goes through a single `CommandPlan` dispatch loop.

I also added a workspace module replacement regression so `workspace.mode:auto` behavior stays tied to the planner's `GoModules` output instead of requiring provider-local `module.replace` entries in specs. This keeps the generated app path aligned with the intended v2 workspace behavior.

### Prompt Context

**User prompt (verbatim):** "do these:  - Phase 2 cleanup                                                                                                                                                 \n     - Task 20: Verify workspace.mode:auto / plan.GoModules still avoids unnecessary provider replace entries.                                                     \n - Phase 3 source registry completion                                                                                                                              \n     - Task 23: Port JS verb scanning to use SourceRegistry handles directly.                                                                                      \n     - Task 24: Port help loading to SourceRegistry kind=help.                                                                                                     \n     - Task 25: Port asset resolution to SourceRegistry kind=assets.                                                                                                \n - Phase 4 runtime app cleanup                                                                                                                                     \n     - Task 27: Confirm NewRootCommand / NewHost are fully RuntimePlan-based and remove old RuntimeSpec dependency.                                                \n     - Task 28: Confirm runtime factory uses RuntimePlan.Runtime.Modules.                                                                                          \n     - Task 29: Rewrite built-in and provider command attachment as one loop over []CommandPlan.                                                                    \n     - Task 30: Preserve/test root vs nested CLI mount semantics.                                                                                                  \n     - Task 31: Confirm framework/help/logging config uses RuntimePlan.App."

**Assistant interpretation:** Implement and validate the next listed cleanup tasks across generation, source consumers, host command attachment, and RuntimePlan app wiring.

**Inferred user intent:** Keep pushing the hard cutover forward in go-go-goja before returning to sessionstream.

**Commit (code):** f09788a — "Finish RuntimePlan source consumer cleanup"

### What I did

- Updated `TestRenderGoModPlanUsesWorkspaceModulePlan` so a provider import under a workspace-local module uses the planner's `GoModules` replacement and does not need a provider `module.replace` entry.
- Added `Host.SourceRegistry` and build it in `NewHostWithOptions` from `RuntimePlan` sources.
- Changed JS verb command construction to scan through `SourceRegistry` instead of directly iterating `RuntimePlan` sources.
- Changed `verbs sources` listing to use `SourceRegistry.ListSourcesByKind(jsverbs)`.
- Changed help loading to prefer `SourceRegistry.ListSourcesByKind(help)`.
- Changed asset store setup to use `SourceRegistry.ListSourcesByKind(assets)`.
- Reworked `AttachDefaultCommands` to iterate `RuntimePlan.runtimeCommands()` and dispatch through `AttachCommandPlan` for built-in and provider commands.
- Marked tasks 20, 23, 24, 25, 27, 28, 29, 30, and 31 complete.

### Why

- The previous step introduced `SourceRegistry`, but several consumers still read source slices directly from `RuntimePlan`.
- A hard cutover needs a single runtime plan path: host setup, source lookup, and command attachment should all flow from RuntimePlan concepts.
- The command dispatch loop makes `[]CommandPlan` the active command representation rather than treating provider commands as a separate attach phase.

### What worked

- Focused validation passed:

```text
go test ./cmd/xgoja ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./pkg/xgoja/providerapi ./pkg/xgoja/providers/http -count=1
```

- The pre-commit hook passed lint, `go generate ./...`, and `go test ./...` before commit `f09788a` landed.

### What didn't work

The first build after changing JS verb scanning failed because a local variable named `jsverbs` shadowed the imported `jsverbs` package in `root.go`:

```text
pkg/xgoja/app/root.go:296:94: jsverbs.Registry is not a type
pkg/xgoja/app/root.go:296:118: jsverbs.VerbSpec is not a type
```

I renamed the local variable to `jsverbSources`.

The first asset-store refactor also passed `[]SourcePlan` to a function that expected provider-facing descriptors:

```text
cannot use runtimePlan.sourcesByKind(SourceKindAssets) (value of type []SourcePlan) as []providerapi.RuntimeSourceDescriptor value in argument to NewAssetStoreFromSources
```

I changed `NewAssetStore` to construct a `SourceRegistry` and request asset descriptors from it.

### What I learned

- Once `SourceRegistry` exists at `Host`, the runtime consumers become simpler: each subsystem asks for a source kind instead of knowing the RuntimePlan layout.
- `go test ./...` through the pre-commit hook remains useful because generated fixtures and example packages catch stale template/runtime assumptions.
- The command loop can coexist with the older explicit `AttachEval`/`AttachRun` methods, but `AttachDefaultCommands` no longer needs separate built-in/provider phases.

### What was tricky to build

- The migration still has transitional compatibility fields in `RuntimePlan`, so source consumers must avoid accidentally depending on them directly. Routing through `SourceRegistry` helps isolate that compatibility normalization.
- Help and asset sources have different embedded filesystems from JS verbs, so the registry exposes descriptors while the actual embedded FS remains owned by the subsystem (`EmbeddedHelp`, `EmbeddedAssets`).
- The command dispatch loop had to preserve existing root-mounted JS verb semantics and provider mount semantics while changing attachment order.

### What warrants a second pair of eyes

- The `AttachDefaultCommands` ordering now follows `runtimeCommands()` order for configured commands, then adds utility commands (`modules`, `selected-modules`, `types`). Reviewers should confirm this is acceptable for generated CLIs.
- Help and asset loading use registry descriptors but still rely on their subsystem-specific embedded FS values. That is intentional, but worth checking for provider-source edge cases.
- Task 30 is satisfied by existing root/nested JS verb mount tests plus the preserved dispatch behavior; a future cleanup could add a new pure-v2 JSON mount test after compatibility fields are removed.

### What should be done in the future

- Remove the remaining transitional compatibility fields/decode paths from `RuntimePlan`.
- Update HTTP serve to consume `ctx.Sources` directly instead of the JSVerb adapter.
- Continue with providerutil/runtime initializer cleanup and docs.

### Code review instructions

- Start with `pkg/xgoja/app/host.go` to review SourceRegistry ownership and the `AttachCommandPlan` loop.
- Review `pkg/xgoja/app/root.go` for JS verb scanning through SourceRegistry.
- Review `pkg/xgoja/app/framework.go` and `pkg/xgoja/app/assets.go` for help/assets source kind migration.
- Review `cmd/xgoja/internal/generate/generate_test.go` for workspace-mode replacement coverage.
- Validate with:

```bash
go test ./cmd/xgoja ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./pkg/xgoja/providerapi ./pkg/xgoja/providers/http -count=1
go test ./... -count=1
```

### Technical details

`AttachDefaultCommands` now follows this pattern:

```go
for _, command := range h.RuntimePlan.runtimeCommands() {
    h.AttachCommandPlan(root, command)
}
h.AttachModules(root)
h.AttachSelectedModules(root)
h.AttachTypes(root)
```

Source consumers now request source descriptors by kind from the host registry:

```go
sourceRegistry.ListSourcesByKind(providerapi.RuntimeSourceKindJSVerbs)
sourceRegistry.ListSourcesByKind(providerapi.RuntimeSourceKindHelp)
sourceRegistry.ListSourcesByKind(providerapi.RuntimeSourceKindAssets)
```

## Step 7: Port HTTP serve to the command SourceRegistry

I completed the next hard-cutover cleanup slice by removing HTTP serve's direct dependence on the transitional `ctx.JSVerbs` command adapter. The provider command-set now requires `CommandSetContext.Sources`, derives its JS verb source set from that source registry, and uses that same scoped source set for command discovery, hot reload rescans, default watch roots, and TypeScript watch extension detection.

This also validated the surrounding cleanup tasks: the generator no longer has active v2-to-legacy conversion helper paths, provider/runtime initialization continues to run through the final module descriptor path, the `examples/xgoja/13-http-serve-jsverbs` smoke target builds and serves a generated binary with `provider.command-set sources`, and the existing `app.mount`/mountable-handler package tests still pass.

### Prompt Context

**User prompt (verbatim):** "Implement cleanup tasks:
- Phase 2 Task 18: remove old generator helper remnants, especially v2-to-legacy conversion helpers.
- Phase 5 Task 35: update providerutil/runtime initializer paths for final `RuntimeModulePlan` model.
- Phase 6 Tasks 36–39: update HTTP serve to use `ctx.Sources` directly, scope hot reload rescans/watches to command sources, repair smoke tests for `examples/xgoja/13-http-serve-jsverbs`, and validate `app.mount` docs/examples."

**Assistant interpretation:** Finish the remaining generator/providerutil/HTTP serve cleanup slice by eliminating HTTP serve's transitional source adapter usage, validating scoped hot reload and example smoke behavior, and checking mountable HTTP composition still works.

**Inferred user intent:** Continue the hard cutover so active runtime/provider code depends on v2-native `RuntimePlan` and `SourceRegistry` concepts rather than legacy or adapter-era structures.

**Commit (code):** 08f1264 — "Port HTTP serve to SourceRegistry context"

### What I did

- Added `serveCommandJSVerbSources` in `pkg/xgoja/providers/http/serve.go` to require `ctx.Sources` and derive JS verb sources via `ctx.Sources.JSVerbs()`.
- Updated HTTP serve command discovery to scan the derived command-scoped source set instead of `ctx.JSVerbs`.
- Updated hot reload to rescan, select default watch roots, and detect TypeScript support from the same scoped source set.
- Updated `pkg/xgoja/providers/http/serve_test.go` to provide a fake `SourceRegistry` instead of direct `JSVerbs` fields for HTTP serve setup.
- Checked tasks 18, 35, 36, 37, 38, and 39 in docmgr and updated the ticket changelog.

### Why

- HTTP serve is a provider command-set, so its visible verbs and hot reload watch set must be scoped by `commands[].sources` in the v2 runtime plan.
- Direct `ctx.JSVerbs` access was the last adapter-era path in this provider and made it easier to accidentally rescan unselected source roots.
- The hard cutover requires runtime/provider code to express source intent through `SourceRegistry`, not legacy top-level JS verb metadata.

### What worked

- Focused validation passed:

```bash
go test ./pkg/xgoja/providers/http ./pkg/xgoja/app ./cmd/xgoja/internal/generate -count=1
go test ./cmd/xgoja ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./pkg/xgoja/providerapi ./pkg/xgoja/providers/http ./pkg/gojahttp ./modules/express -count=1
```

- The runnable HTTP serve example passed:

```bash
cd examples/xgoja/13-http-serve-jsverbs && make smoke
```

- Mountable HTTP handler and Express mount behavior remained green:

```bash
go test ./pkg/gojahttp ./modules/express -count=1
```

- The pre-commit hook passed lint, `go generate ./...`, and `go test ./...` before commit `08f1264` landed.

### What didn't work

N/A. This cleanup slice was straightforward after the `SourceRegistry` abstraction existed.

### What I learned

- The HTTP serve provider only needs one source-entry point: `ctx.Sources.JSVerbs()`. Keeping all scans and watch root derivation behind that method makes command scoping explicit.
- The existing `examples/xgoja/13-http-serve-jsverbs` smoke target already covered the important generated-binary behavior: `serve sites demo --http-listen ...` with `provider.command-set sources`.
- The app.mount validation did not require new code because the relevant `pkg/gojahttp` and `modules/express` tests still cover the mount ABI and Express composition behavior.

### What was tricky to build

- The provider API still carries `ctx.JSVerbs` as a temporary adapter for other consumers, so the cleanup had to be deliberate: HTTP serve tests now construct `Sources` to prevent accidental regression back to direct adapter use.
- Hot reload has multiple source-dependent paths: rescan, default watch roots, and TypeScript extension detection. All three needed to use the same scoped source set or command-scoped reload would still be leaky.
- Task 35 was mostly validation-oriented in this slice because runtime initialization already receives selected module descriptors from the final RuntimePlan-derived command context; the relevant check was that HTTP serve still initializes runtimes correctly after removing direct JS verb adapter access.

### What warrants a second pair of eyes

- Review whether `serveCommandJSVerbSources` should allow an empty source registry for edge cases, or whether failing fast is the right provider command-set contract.
- Confirm that the fake test SourceRegistry mirrors the production `JSVerbs()` behavior closely enough for future provider tests.
- Re-check hot reload behavior once the transitional `ctx.JSVerbs` field is removed entirely from `CommandSetContext`.

### What should be done in the future

- Continue Phase 7 generated runtime-package cleanup.
- Remove remaining transitional compatibility fields/decode paths from `RuntimePlan`.
- Remove `CommandSetContext.JSVerbs` once all providers and tests use `ctx.Sources`.

### Code review instructions

- Start with `pkg/xgoja/providers/http/serve.go`, especially `serveCommandJSVerbSources`, `newServeCommandSet`, and `serveVerbHotReload`.
- Then review `pkg/xgoja/providers/http/serve_test.go` for the test registry shape and the updated command contexts.
- Validate with:

```bash
go test ./cmd/xgoja ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./pkg/xgoja/providerapi ./pkg/xgoja/providers/http ./pkg/gojahttp ./modules/express -count=1
cd examples/xgoja/13-http-serve-jsverbs && make smoke
```

### Technical details

HTTP serve now follows this dependency path:

```go
func serveCommandJSVerbSources(ctx providerapi.CommandSetContext) (providerapi.JSVerbSourceSet, error) {
    jsverbSources := ctx.Sources.JSVerbs()
    return jsverbSources, nil
}
```

Hot reload uses that same scoped set for every source-derived operation:

```go
resolveServeHotReloadVerb(jsverbSources, registry, verb, verbPath)
defaultServeHotReloadWatchRoots(jsverbSources)
sourceSetHasTypeScript(jsverbSources)
```
