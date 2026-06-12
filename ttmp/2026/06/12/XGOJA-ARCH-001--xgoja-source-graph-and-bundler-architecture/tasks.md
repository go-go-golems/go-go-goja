# Tasks

## Completed ticket setup and architecture work

- [x] Create XGOJA-ARCH-001 architecture ticket.
- [x] Capture the architecture reassessment prompt as a source note.
- [x] Review current buildspec, runtime spec, provider API, jsverbs, TypeScript, generation, declarations, and hot reload paths.
- [x] Write detailed source graph and bundler architecture design.
- [x] Write investigation diary.
- [x] Update architecture design v2 with Go workspace resolution for local provider modules.
- [x] Add second architecture document for xgoja v2 spec and migration strategy.
- [x] Simplify v2 spec document around goja-executed source and external asset bundles.

## Hard cutover implementation plan for xgoja v2

### Phase 0: Freeze v1 semantics as migration input only

- [ ] Inventory every v1 `xgoja.yaml` example and test fixture under `examples/xgoja`, `cmd/xgoja`, and `pkg/xgoja`.
- [ ] Add golden v1 input fixtures for representative specs:
  - minimal generated binary;
  - provider module selection;
  - command provider mount;
  - jsverbs source;
  - TypeScript jsverbs source;
  - embedded jsverbs/help/assets;
  - generated runtime package;
  - target adapter/cobra if still needed.
- [ ] Decide the exact hard-cutover rule: v1 specs are accepted only by `xgoja migrate-spec`, not by `xgoja build`, `xgoja doctor`, or `xgoja gen-dts` after cutover.
- [ ] Add a clear diagnostic for v1 specs passed to normal commands: `xgoja.yaml appears to be v1; run xgoja migrate-spec -f xgoja.yaml --out xgoja.v2.yaml`.
- [ ] Confirm whether any downstream packages require temporary dual support; if not, keep the implementation v2-native with no compatibility branch in planner/runtime code.

### Phase 1: Define the simplified v2 schema package

- [ ] Create `cmd/xgoja/internal/specv2`.
- [ ] Add `types.go` with v2 DTOs:
  - `Config`;
  - `AppSpec`;
  - `GoSpec`;
  - `WorkspaceSpec`;
  - `ProviderSpec`;
  - `RuntimeSpec` / `RuntimeModuleSpec`;
  - `SourceSpec`;
  - `SourceFromSpec`;
  - `CompileSpec`;
  - `CommandSurfaceSpec`;
  - `ArtifactSpec`;
  - optional `ProfileSpec` placeholder if needed.
- [ ] Keep the v2 DTO intentionally small: no ordinary `engine`, `platform`, `target`, `format`, browser bundle, package manager, loader, or polyfill fields.
- [ ] Add `load.go` for YAML loading and schema detection.
- [ ] Add `defaults.go` for v2 defaults:
  - `schema: xgoja/v2` required in rendered output;
  - provider `register` defaults to `Register`;
  - source `language` inferred only if omitted and safe;
  - source `compile.mode` defaults by source kind;
  - workspace defaults to `auto` or `off` as decided before implementation.
- [ ] Add `validate.go` for structural validation:
  - unique provider IDs;
  - unique source IDs;
  - unique command IDs;
  - unique artifact IDs;
  - runtime module provider exists;
  - command source references exist;
  - command provider references exist;
  - provider command names are non-empty;
  - artifact source references exist;
  - no unsupported broad bundler fields are accepted silently.
- [ ] Add `render.go` to write stable, formatted v2 YAML for migration output and tests.
- [ ] Add schema unit tests and golden render tests.

### Phase 2: Build v1-to-v2 migration tooling

- [ ] Add `cmd/xgoja/internal/specv2/migrate_v1.go` that converts `buildspec.BuildSpec` to `specv2.Config`.
- [ ] Implement v1 `packages[]` → v2 `providers[]`.
- [ ] Implement v1 `modules[]` → v2 `runtime.modules[]`.
- [ ] Implement v1 builtin `commands` → v2 command surfaces:
  - `eval` → `type: builtin.eval` if still kept;
  - `run` → `type: builtin.run`;
  - `repl` → `type: builtin.repl` if still kept;
  - `jsverbs` → `type: builtin.jsverbs` with migrated source references.
- [ ] Implement v1 `commandProviders[]` → v2 `commands[]` with `type: provider.command-set`.
- [ ] Implement v1 `jsverbs[]` → v2 `sources[]` with `kind: jsverbs`.
- [ ] Implement v1 TypeScript settings migration:
  - `typescript.enabled` → `language: typescript`;
  - `typescript.bundle` → `compile.bundle`;
  - `typescript.checkCommand` → `compile.check.command`;
  - remove runtime-module aliases from `typescript.external` where they can be derived;
  - preserve non-runtime externals only if an explicit future field is added, otherwise warn.
- [ ] Implement v1 `help.sources[]` → v2 `sources[]` with `kind: help`.
- [ ] Implement v1 `assets[]` → v2 `sources[]` with `kind: assets` plus `artifacts[]` entries when embedded.
- [ ] Implement v1 `target` → v2 `artifacts[]`:
  - binary;
  - runtime package;
  - adapter/cobra targets if still supported.
- [ ] Implement v1 local replacements migration:
  - `packages[].replace` becomes a provider module local override or a migration warning recommending `workspace.mode: auto`;
  - `--xgoja-replace` remains a CLI concern and is represented in migration docs, not v2 spec output by default.
- [ ] Add migration warnings with file paths and v1 field references.
- [ ] Add golden migration tests for all Phase 0 fixtures.

### Phase 3: Add `xgoja migrate-spec`

- [ ] Add a new Cobra/Glazed command `migrate-spec`.
- [ ] Support flags:
  - `-f, --file`;
  - `--out`;
  - `--in-place`;
  - `--backup`;
  - `--check`;
  - `--from v1`;
  - `--to v2`;
  - `--profile dev|release` only if profiles are included.
- [ ] Print migration warnings in a stable, grep-friendly format.
- [ ] In `--check` mode, return non-zero if migration would change the file.
- [ ] In `--in-place --backup` mode, write `xgoja.yaml.bak` before overwriting.
- [ ] Add tests for normal output, check mode, in-place mode, backup mode, and warning output.
- [ ] Add a migration doc page under `cmd/xgoja/doc`, e.g. `16-migrating-to-xgoja-v2.md`.

### Phase 4: Implement Go workspace resolution for generated builds

- [ ] Create `cmd/xgoja/internal/workspace` or `pkg/xgoja/workspace` depending on whether it is needed outside the CLI.
- [ ] Implement upward `go.work` discovery from the v2 spec directory.
- [ ] Parse workspace module directories using `go work edit -json` first.
- [ ] Read each workspace module's `go.mod` to map module path → local directory.
- [ ] Define `GoModulePlan`:
  - module path;
  - version;
  - local dir;
  - required-by list;
  - resolution kind (`versioned`, `replace`, `workspace`);
  - resolution source (`explicit-replace`, `cli-replace`, `go-work`, `version`).
- [ ] Implement precedence:
  - explicit provider local override / migrated replace;
  - `--xgoja-replace` for `github.com/go-go-golems/go-go-goja` during transition if the flag remains;
  - explicit workspace module mapping if added;
  - detected `go.work` module;
  - versioned requirement.
- [ ] Wire workspace-derived replacements into generated `go.mod` rendering.
- [ ] Apply the same module plan to `xgoja build` and `xgoja gen-dts` sidecars.
- [ ] Add doctor/plan diagnostics showing module path, local dir, version, and resolution source.
- [ ] Add tests for workspace auto, workspace off, explicit file, replace precedence, missing version/local module errors, and sidecar rendering.

### Phase 5: Build the v2 provider graph without changing provider runtime APIs by default

- [ ] Create provider graph types around existing `providerapi.ProviderRegistry`.
- [ ] Keep existing provider Go APIs initially:
  - `providerapi.Module`;
  - `providerapi.CommandSetProvider`;
  - `providerapi.VerbSource`;
  - `providerapi.HelpSource`;
  - asset provider APIs.
- [ ] Add a provider API audit task: verify whether v2 requires any provider API changes, especially around static descriptors, command dependencies, source descriptions, and future provider manifests.
- [ ] Prefer not to change provider APIs in the hard cutover unless a concrete planner requirement demands it.
- [ ] Resolve v2 `providers[]` against the provider registry.
- [ ] Resolve v2 `runtime.modules[]` against provider modules.
- [ ] Validate duplicate runtime module aliases centrally.
- [ ] Expose runtime module aliases to the source compiler as automatic externals.
- [ ] Reuse provider graph for declaration generation instead of re-resolving modules independently.
- [ ] Add tests for missing provider, missing module, duplicate alias, missing TypeScript descriptor in strict declaration mode, and selected command set validation.

### Phase 6: Build the v2 source graph

- [ ] Create `pkg/xgoja/sourcegraph` or `pkg/xgoja/internal/sourcegraph` depending on intended API stability.
- [ ] Implement source origins:
  - disk directory;
  - provider `fs.FS`;
  - embedded `fs.FS` internal origin;
  - future workspace directory origin.
- [ ] Implement source kinds:
  - `jsverbs`;
  - `script`;
  - `assets`;
  - `help`.
- [ ] Implement include/exclude filtering for source graph discovery.
- [ ] Preserve origin metadata needed by TypeScript runtime bundling and hot reload.
- [ ] Implement local import resolution with extension probing for goja-executed JS/TS sources.
- [ ] Implement runtime module alias classification as automatic externals.
- [ ] Reject unknown bare imports by default with actionable diagnostics.
- [ ] Add tests for disk source sets, provider `fs.FS` source sets, embedded source sets, path escape rejection, helper import resolution, runtime module alias resolution, and unknown bare import diagnostics.

### Phase 7: Port TypeScript/jsverbs execution to v2 graph/plans

- [ ] Implement XGOJA-TS-002 or incorporate it here: fs.FS-backed runtime bundling for embedded/provider TypeScript jsverbs.
- [ ] Replace direct xgoja calls to `jsverbs.ScanDir`/`ScanFS` with graph-backed scan adapters.
- [ ] Keep `jsverbs.ScanDir` and `ScanFS` as lower-level convenience APIs if still useful, but normal xgoja v2 execution should use source graph adapters.
- [ ] Make jsverbs runtime transforms use source graph origin metadata.
- [ ] Make TypeScript runtime module externals derive from provider graph/runtime modules, not per-source config.
- [ ] Preserve overlay-before-bundling behavior.
- [ ] Add tests for v2 filesystem TypeScript jsverbs, v2 embedded TypeScript jsverbs, and v2 provider TypeScript jsverbs.

### Phase 8: Replace build/generate/gen-dts command paths with v2 planner

- [ ] Create a v2 `Plan` type that includes:
  - `GoModulePlan`;
  - provider graph;
  - source graph;
  - command surfaces;
  - artifact plan;
  - runtime module aliases;
  - declaration plan.
- [ ] Update `xgoja doctor` to load v2 and validate through the planner.
- [ ] Update `xgoja build` to generate from v2 artifact plan.
- [ ] Update `xgoja gen-dts` to use v2 provider graph and declaration artifact plan.
- [ ] Update generated `go.mod` rendering to consume `GoModulePlan`.
- [ ] Update embedded source copying to consume source/artifact plans.
- [ ] Update generated runtime spec rendering to consume the runtime plan.
- [ ] Add tests for build dry-run, build with workspace auto, gen-dts with workspace auto, and generated runtime package output.

### Phase 9: Cut over examples and docs

- [ ] Migrate `examples/xgoja/15-typescript-jsverbs` first.
- [ ] Migrate HTTP serve jsverbs example.
- [ ] Migrate generated runtime package example.
- [ ] Migrate provider examples and tutorials.
- [ ] Migrate asset/help examples.
- [ ] Update `examples/xgoja/README.md` for v2 examples.
- [ ] Add `cmd/xgoja/doc/17-xgoja-v2-reference.md`.
- [ ] Update existing tutorials to v2 or move v1 content into migration docs.
- [ ] Add clear docs for externally-built frontend/browser bundles as `kind: assets` sources.
- [ ] Add clear docs for goja-runtime package bundling as future behavior under `compile.bundle: true`.

### Phase 10: Hard remove normal v1 execution paths

- [ ] Make normal commands reject v1 specs with a migration diagnostic.
- [ ] Remove v1 planner/generation code paths from `build`, `doctor`, `gen-dts`, and example workflows.
- [ ] Keep only enough v1 DTO/loading code for `migrate-spec`.
- [ ] Remove or quarantine v1-only tests that no longer apply.
- [ ] Ensure repository examples are all v2.
- [ ] Run full validation:
  - `go test ./cmd/xgoja/... ./pkg/xgoja/... ./pkg/jsverbs ./pkg/tsscript -count=1`;
  - relevant example smoke targets;
  - `docmgr doctor --ticket XGOJA-ARCH-001 --stale-after 30`.

### Phase 11: Optional provider manifest/catalog follow-up, not part of hard cutover MVP

- [ ] Create a separate ticket for provider manifests and provider catalog search.
- [ ] Keep provider manifest work out of the hard cutover critical path unless v2 provider graph implementation proves it is required.
- [ ] Design provider manifests as static metadata that complements, not replaces, provider Go registration.

## Open implementation questions

- [ ] Should `workspace.mode` default to `auto` for local development or `off` for release safety?
- [ ] Should v2 require `language` for goja-executed source, or infer from extensions by default?
- [ ] Should `compile.mode: build-time` ship in the first v2 cutover, or should MVP support only `runtime` plus asset embedding?
- [ ] Should builtin `eval` and `repl` remain in v2 command surfaces, or should v2 start with `run`, `jsverbs`, and provider command sets only?
- [ ] Should v2 `artifacts` allow multiple output artifacts in the first implementation, or should MVP start with one binary artifact plus declarations?
- [ ] Should migrated v1 comments be discarded, preserved best-effort, or copied into a generated migration report?
- [ ] How long should `migrate-spec` keep v1 parsing after normal v1 execution is removed?

## Suggested first follow-up tickets

- [ ] `XGOJA-V2-001`: Implement simplified specv2 DTOs, validation, rendering, and migrate-spec command.
- [ ] `XGOJA-V2-002`: Implement Go workspace resolution and generated go.mod module planning.
- [ ] `XGOJA-V2-003`: Implement provider graph and centralized runtime module alias resolution.
- [ ] `XGOJA-V2-004`: Implement source graph and v2 jsverbs/TypeScript source execution.
- [ ] `XGOJA-V2-005`: Port build, gen-dts, doctor, examples, and docs to v2 and remove normal v1 execution.
