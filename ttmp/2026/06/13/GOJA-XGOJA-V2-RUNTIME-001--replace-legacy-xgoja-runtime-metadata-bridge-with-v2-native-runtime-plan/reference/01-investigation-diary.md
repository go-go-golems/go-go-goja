---
Title: Investigation diary
DocType: reference
Ticket: GOJA-XGOJA-V2-RUNTIME-001
Status: active
Topics: [xgoja, architecture, codegen]
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
