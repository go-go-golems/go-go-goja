---
Title: Investigation diary
Ticket: XGOJA-ARCH-001
Status: active
Topics:
    - goja
    - xgoja
    - typescript
    - tooling
    - developer-experience
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/xgoja/internal/buildspec/build_spec.go
      Note: Diary Step 1 build-time schema evidence
    - Path: cmd/xgoja/internal/generate/gomod.go
      Note: Diary Step 2 workspace resolution evidence and integration target
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/build_spec.go
      Note: Step 1 build-time xgoja schema evidence.
    - Path: go-go-goja/cmd/xgoja/internal/generate/generate.go
      Note: Step 1 generated output and embedded copy evidence.
    - Path: go-go-goja/cmd/xgoja/internal/generate/main.go
      Note: Step 1 generated runtime spec and embedded path rewrite evidence.
    - Path: go-go-goja/pkg/jsverbs/runtime.go
      Note: Step 1 current jsverbs require loader and invocation behavior evidence.
    - Path: go-go-goja/pkg/jsverbs/scan.go
      Note: Step 1 current ScanDir/ScanFS behavior evidence.
    - Path: go-go-goja/pkg/tsscript/compiler.go
      Note: Step 1 current esbuild facade evidence.
    - Path: go-go-goja/pkg/xgoja/app/runtime_spec.go
      Note: Step 1 runtime spec evidence.
    - Path: go-go-goja/pkg/xgoja/app/typescript.go
      Note: Step 1 TypeScript transform adapter evidence.
    - Path: go-go-goja/pkg/xgoja/providerapi/commands.go
      Note: Step 1 provider command and jsverb source-set API evidence.
    - Path: go-go-goja/pkg/xgoja/providerapi/module.go
      Note: Step 1 provider module API evidence.
    - Path: pkg/jsverbs/scan.go
      Note: Diary Step 1 source scanning evidence
    - Path: pkg/tsscript/compiler.go
      Note: Diary Step 1 TypeScript bundling evidence
    - Path: pkg/xgoja/app/runtime_spec.go
      Note: Diary Step 1 runtime schema evidence
    - Path: ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/design/02-xgoja-v2-spec-and-migration-architecture.md
      Note: Diary Step 3 v2 spec and migration design
ExternalSources:
    - local:01-architecture-reassessment-prompt.md
Summary: Chronological diary for the xgoja source graph and bundler architecture reassessment.
LastUpdated: 2026-06-12T12:35:00-04:00
WhatFor: Use to understand why the architecture ticket exists and how the source-graph/bundler design was produced.
WhenToUse: Read before implementing or reviewing the xgoja source graph, provider graph, build plan, runtime plan, or resolver architecture.
---




# Diary

## Goal

This diary records the architecture reassessment for xgoja after the TypeScript support and embedded TypeScript runtime bundling work exposed repeated source-resolution, provider-resolution, bundling, generation, and hot reload concerns.

## Step 1: Create the architecture ticket and write the source graph/bundler RFC

I created `XGOJA-ARCH-001` to step back from individual TypeScript fixes and define a larger architecture for xgoja. The resulting design treats xgoja as a Go-backed JavaScript runtime compiler: it resolves Go provider packages, Go-backed runtime modules, JavaScript/TypeScript source files, jsverb metadata, help, assets, declarations, generated outputs, and hot reload inputs into explicit source graphs and plans.

The design intentionally avoids a full rewrite. It preserves the goja runtime factory, provider module API, jsverbs command model, and generated runtime DTO, then introduces a planning layer that can be migrated into the current codebase incrementally.

### Prompt Context

**User prompt (verbatim):** "Ok, write that architecture docuemnt, in great depth and detail.

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new docmgr architecture ticket and write a detailed intern-oriented design/RFC explaining how xgoja should evolve into a source-graph and bundler-style compiler for Go-backed JavaScript runtimes, then upload the deliverable to reMarkable.

**Inferred user intent:** Pause feature-by-feature work long enough to define the fundamental architecture xgoja should grow toward before more TypeScript, embedded source, provider, and hot reload features are added.

**Commit (code):** N/A — documentation and architecture design only; no production code changed in this step.

### What I did

- Created ticket `XGOJA-ARCH-001` with:
  - `docmgr ticket create-ticket --ticket XGOJA-ARCH-001 --title "XGoja source graph and bundler architecture" --topics goja,xgoja,typescript,tooling,developer-experience`
- Created the primary design document:
  - `ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/design/01-xgoja-source-graph-and-bundler-architecture.md`
- Created this diary:
  - `ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/reference/01-investigation-diary.md`
- Captured the triggering prompt in:
  - `ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/sources/local/01-architecture-reassessment-prompt.md`
- Reviewed the current implementation and prior ticket designs, including:
  - `XGOJA-TS-001` TypeScript support design;
  - `XGOJA-TS-002` embedded/provider TypeScript runtime bundling design;
  - buildspec/runtime spec files;
  - provider API files;
  - engine runtime factory;
  - jsverbs scan/runtime files;
  - tsscript compiler facade;
  - xgoja TypeScript adapter;
  - xgoja run command;
  - generated output code;
  - declaration generation;
  - hot reload manager and HTTP provider integration.
- Wrote a detailed architecture document covering:
  - current-state architecture with file evidence;
  - problem statement and goals;
  - source graph design;
  - import resolver design;
  - provider bundle graph;
  - build plan and runtime plan;
  - compilation modes;
  - data-flow diagrams;
  - decision records;
  - staged migration plan;
  - API sketches;
  - testing strategy;
  - risks and near-term recommendations.

### Why

- The same unresolved source-resolution problem has appeared across TypeScript, embedded sources, provider sources, declarations, and hot reload.
- xgoja is now more than a generator; it is becoming a compiler pipeline for Go-backed JavaScript runtimes.
- A source graph and plan layer can reduce future local patches by making source origins, imports, runtime module aliases, and compilation modes explicit.

### What worked

- The current code already has good boundaries that the architecture can preserve:
  - build-time schema and runtime schema are separated;
  - provider modules are registered through a clear provider API;
  - runtime construction is centralized in `pkg/engine/factory.go`;
  - jsverbs scanning and invocation are isolated in `pkg/jsverbs`;
  - TypeScript compilation is isolated in `pkg/tsscript`;
  - hot reload has a blue/green manager with smoke-test support.
- The architecture could therefore recommend a planning layer rather than a runtime rewrite.

### What didn't work

- No command failed while creating this ticket or writing the design.
- The investigation confirmed architecture gaps already documented in prior work:
  - source origin is not first-class;
  - import resolution is scattered;
  - runtime-module externalization differs across `xgoja run` and jsverbs;
  - embedded source compilation policy is implicit;
  - hot reload watches roots/extensions rather than graph dependencies.

### What I learned

- The correct architectural framing is not "xgoja should become a JavaScript bundler" in the usual npm-only sense. It should become a Go-backed JavaScript runtime compiler.
- Provider packages are bundle inputs. They provide Go implementations of JavaScript modules, TypeScript descriptors, command sets, jsverb source trees, help, assets, and host services.
- The first practical migration should be jsverbs and embedded/provider TypeScript bundling because that path has a concrete review issue and exercises source origin, import resolution, runtime module aliases, and hot reload.

### What was tricky to build

- The design needed to separate source discovery from compilation mode. A source graph should answer what files exist and how they import each other. A build/runtime plan should answer whether those files are compiled at build time, compiled at runtime, preserved, copied, embedded, or rejected.
- The design also needed to avoid overreach. It proposes sourcegraph/provider-graph/plan packages, but it explicitly preserves the current runtime factory, provider registry, jsverbs command model, and generated runtime spec during migration.

### What warrants a second pair of eyes

- Review whether `sourcegraph`, `plan`, and `compile` are the right package boundaries or whether `plan` should initially live under `cmd/xgoja/internal` until stable.
- Review the decision that unknown bare imports should fail by default rather than be externalized by default.
- Review whether runtime plan data should eventually be serialized into generated JSON or kept as Go-only derived structures for longer.
- Review whether build-time prebundling should be a near-term ticket or remain behind runtime `fs.FS` bundling and provider-graph work.

### What should be done in the future

- Implement `XGOJA-TS-002` first to solve the immediate code review issue.
- Create `XGOJA-ARCH-002` for the first `sourcegraph` package and jsverb source discovery migration.
- Create `XGOJA-ARCH-003` for provider graph and centralized runtime module alias resolution.
- Add debug commands such as `xgoja plan` and `xgoja graph` after the planning layer exists.

### Code review instructions

- Start with the design document:
  - `ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/design/01-xgoja-source-graph-and-bundler-architecture.md`
- Then inspect the current evidence files in this order:
  - `cmd/xgoja/internal/buildspec/build_spec.go`
  - `pkg/xgoja/app/runtime_spec.go`
  - `pkg/xgoja/providerapi/module.go`
  - `pkg/xgoja/providerapi/commands.go`
  - `pkg/jsverbs/scan.go`
  - `pkg/jsverbs/runtime.go`
  - `pkg/tsscript/compiler.go`
  - `pkg/xgoja/app/typescript.go`
  - `cmd/xgoja/internal/generate/main.go`
  - `cmd/xgoja/internal/generate/generate.go`
- Validate ticket hygiene with:
  - `docmgr doctor --ticket XGOJA-ARCH-001 --stale-after 30`

### Technical details

- Proposed core package: `pkg/xgoja/sourcegraph`.
- Proposed planning package: `pkg/xgoja/plan`.
- Proposed migration order: jsverbs/sourcegraph first, provider graph second, build/runtime plan third, graph-based hot reload fourth, optional embedded prebundling fifth.
- Primary conceptual split:
  - source graph records what exists and imports what;
  - provider graph records what Go packages contribute;
  - build plan records generated artifacts and build-time compilation policy;
  - runtime plan records executable runtime behavior and loader policy.


## Step 2: Add Go workspace resolution to the architecture

I updated the architecture design to cover local Go workspace support. This extension treats `go.work` as the Go-module counterpart to the JavaScript/TypeScript source graph: source graph resolution answers where `import "./helper"` comes from, while workspace resolution answers where generated Go imports such as `github.com/acme/provider/pkg/xgoja/http` and `github.com/go-go-golems/go-go-goja/pkg/xgoja/app` come from during generated builds and DTS sidecars.

The design recommends deriving generated `replace` directives from discovered `go.work` modules as the first implementation. That preserves the current temporary generated `go.mod` workflow while eliminating repeated manual `packages[].replace` and `--xgoja-replace` entries during local workspace development.

### Prompt Context

**User prompt (verbatim):** "one thing we want to add as well is the support for workspace, where for example go packages for goja are coming from local workspaces, without having to put manual replaces in the xgoja file, a bit like go.work. 

Can you analyze that in context of the new design as well?"

**Assistant interpretation:** Extend the source graph architecture with a Go workspace/module resolution layer so generated xgoja builds can use local provider modules from `go.work` automatically.

**Inferred user intent:** Avoid duplicating local development replacement paths in every `xgoja.yaml` and make workspace-local Go provider development a first-class planning concern.

**Commit (code):** N/A — documentation update only in this step.

### What I did

- Reviewed current generated Go module rendering:
  - `cmd/xgoja/internal/generate/gomod.go`
  - `cmd/xgoja/internal/generate/generate.go`
  - `cmd/xgoja/internal/buildspec/build_spec.go`
- Added a `Go workspace resolution` section to the architecture design.
- Updated the executive summary, terms, problem statement, goals, proposed architecture diagram, package layout, build plan, decision records, implementation phases, tests, risks, recommendations, and references.
- Added proposed concepts:
  - `pkg/xgoja/workspace`
  - `GoModulePlan`
  - `GoModuleResolutionKind`
  - `GoModuleResolutionSource`
  - workspace config under `go.workspace`
- Recommended first implementation strategy:
  - discover `go.work`;
  - map module paths to local directories by reading local `go.mod` files;
  - apply explicit replacement precedence;
  - render derived `replace` directives in generated `go.mod` first;
  - defer temporary generated `go.work` output to a later option.

### Why

- The new architecture should model Go provider module origins, not only JavaScript/TypeScript source origins.
- xgoja generated builds and `gen-dts` sidecars currently need manual local replacements even when `go.work` already knows the local modules.
- Workspace resolution belongs in the build/provider graph layer because generated binaries do not need local workspace paths at runtime.

### What worked

- The current `RenderGoMod` implementation has a clear place where workspace-derived replacements can be added.
- The existing `providerModulePath` helper already performs the necessary import-path-to-module-path inference for provider packages.
- The architecture could add workspace support without changing the source graph thesis: xgoja now has both a Go provider graph and a JS/TS source graph.

### What didn't work

- No command failed in this step.
- The current code has no workspace resolver yet; this remains a design update.

### What I learned

- Workspace support should not be stored in `app.RuntimeSpec`. It is a generated-build concern, not runtime behavior.
- Emitting derived `replace` directives is a better first implementation than synthesizing temporary `go.work` files because it fits the existing generated module workflow and remains easy to inspect with `--keep-work`.
- Release validation still needs a workspace-off mode so local workspace convenience does not hide missing released versions.

### What was tricky to build

- The precedence rules matter. Explicit `packages[].replace` and `--xgoja-replace` must remain stronger than auto-detected workspace entries. Otherwise a user could not override a stale or undesired workspace mapping.
- The design needed to keep Go workspace resolution separate from the JS/TS import resolver while showing how they meet at provider module aliases.

### What warrants a second pair of eyes

- Review whether the first schema should be `go.workspace: auto` shorthand only or the fuller `go.workspace.mode/file/include/exclude` object.
- Review whether derived generated `replace` directives are sufficient for temporary builds or whether a generated `go.work` mode should be implemented at the same time.
- Review how `GOWORK=off` validation should be documented for release workflows.

### What should be done in the future

- Create an implementation ticket for Go workspace resolution, likely `XGOJA-ARCH-002`.
- Add doctor output showing module path, workspace path, resolution source, and status.
- Add plan output that includes `GoModulePlan` entries.

### Code review instructions

- Start with the new `Go workspace resolution` section in the design doc.
- Compare it to current generated module rendering in `cmd/xgoja/internal/generate/gomod.go`.
- Verify that workspace support is represented as build-time/provider-graph input, not runtime spec input.

### Technical details

- Current xgoja replacement option: `generate.Options.XGojaReplace`.
- Current provider replacement field: `buildspec.PackageSpec.Replace`.
- Current generated module renderer: `cmd/xgoja/internal/generate/gomod.go`.
- Proposed first output strategy: render workspace-derived `replace` directives in generated `go.mod`.


## Step 3: Add the v2 spec and migration architecture document

I added a second architecture document that assumes backwards compatibility is not a hard requirement if migration documentation and tooling are strong. This changes the architecture materially: instead of keeping the v1 `xgoja.yaml` schema as the long-term internal shape, the design defines a v2 schema that directly matches the proposed provider graph, source graph, workspace resolver, command-surface model, artifact model, and build/runtime planner.

The document treats v1 as legacy input that should be converted at the CLI boundary. The planner should be v2-native. That keeps future internals simpler and makes the configuration easier to explain because user-facing concepts and internal concepts use the same vocabulary.

### Prompt Context

**User prompt (verbatim):** "Ok, create a second document, then, with the new architecture with a v2 spec, that will make it much easier, we can migrate the packages that use xgoja when we'll need it"

**Assistant interpretation:** Add a separate design document to the existing architecture ticket that describes a v2 xgoja spec and migration strategy, assuming packages can migrate when they need the new architecture.

**Inferred user intent:** Prefer a clean future architecture and v2 config over preserving the old schema indefinitely, as long as migration tooling/documentation makes adoption manageable.

**Commit (code):** N/A — documentation update only in this step.

### What I did

- Added `design/02-xgoja-v2-spec-and-migration-architecture.md` to `XGOJA-ARCH-001`.
- Defined the v2 top-level schema:
  - `schema`
  - `app`
  - `go`
  - `workspace`
  - `providers`
  - `runtime.modules`
  - `sources`
  - `commands`
  - `artifacts`
  - optional `profiles`
- Added a complete v2 rewrite of the recent TypeScript jsverbs example.
- Added detailed mappings from v1 fields to v2 fields.
- Added migration tooling design for `xgoja migrate-spec`.
- Added decision records for:
  - making v2 the native planner schema;
  - using command surfaces;
  - unifying jsverbs/help/assets under source sets;
  - providing migration tooling instead of permanent internal compatibility.
- Added implementation, testing, and documentation plans.

### Why

- The source graph architecture is cleaner if the user-facing spec uses the same concepts as the internal planner.
- Keeping v1 as the permanent internal shape would require repeated compatibility inference for providers, sources, commands, artifacts, workspace resolution, and TypeScript compilation policy.
- A migration tool gives downstream packages a controlled path without forcing every new architecture feature to carry v1-specific semantics.

### What worked

- The v2 spec maps directly to the architecture from the first document:
  - `providers` maps to provider graph inputs;
  - `sources` maps to source graph inputs;
  - `commands` maps to command surfaces;
  - `workspace` maps to Go module resolution;
  - `artifacts` maps to build outputs.
- The TypeScript jsverbs example becomes clearer because runtime module aliases can be used as automatic compiler externals instead of duplicating `external: [express]` under TypeScript config.

### What didn't work

- No command failed in this step.
- This remains design-only; no v2 parser or migration command exists yet.

### What I learned

- Dropping strict backwards compatibility changes the architecture from an adapter-heavy design into a cleaner native planner design.
- The migration boundary should be CLI/spec loading, not planner internals.
- The v2 schema makes command dependencies explicit, which should simplify validation and future plan/debug output.

### What was tricky to build

- The v2 shape needed to be explicit without becoming too verbose for small examples. The design chooses clarity over minimal YAML because xgoja is now coordinating Go providers, source compilation, generated artifacts, and workspace resolution.
- Profiles are useful but should not block v2. The design reserves `profiles` as optional future structure rather than requiring it in the first implementation.

### What warrants a second pair of eyes

- Review whether `sources` should include help/assets immediately or whether v2 should keep dedicated `help` and `assets` sections for readability.
- Review whether `artifacts` should replace `target` fully in v2 or whether a shorthand binary target should exist for simple specs.
- Review whether `xgoja migrate-spec` should preserve comments through a YAML AST library or prioritize clean generated output.

### What should be done in the future

- Implement `cmd/xgoja/internal/specv2` DTOs and validation.
- Implement v1-to-v2 migration with golden tests.
- Add `xgoja migrate-spec` before building too much planner logic.
- Migrate `examples/xgoja/15-typescript-jsverbs/xgoja.yaml` as the first substantial example.

### Code review instructions

- Start with `design/02-xgoja-v2-spec-and-migration-architecture.md`.
- Compare the v2 TypeScript jsverbs example to `examples/xgoja/15-typescript-jsverbs/xgoja.yaml`.
- Review the migration mapping table and the decision records.

### Technical details

- Proposed new package: `cmd/xgoja/internal/specv2`.
- Proposed migration command: `xgoja migrate-spec`.
- Proposed compatibility stance: v1 is migratable legacy input; v2 is the native planner schema.


## Step 4: Simplify the v2 spec around goja-executed source only

I simplified the second architecture document after clarifying the intended scope: xgoja should be opinionated and should only compile or bundle code that runs inside the xgoja/goja runtime. Browser bundles, frontend applications, and other non-goja JavaScript outputs should be built outside xgoja and then embedded or served as assets.

This removes the need for broad bundler configuration in the v2 spec. The simplified design hides esbuild engine/platform/target/format settings as xgoja-owned runtime-profile defaults. Users describe providers, runtime modules, sources, commands, artifacts, workspace behavior, language, and compile intent.

### Prompt Context

**User prompt (verbatim):** "simplify the v2 docs and upload them again."

**Assistant interpretation:** Rewrite the v2 architecture document to reflect the simplified, opinionated scope and upload the new version to reMarkable.

**Inferred user intent:** Keep xgoja's future config focused on its actual runtime responsibilities and avoid carrying unnecessary frontend/general-bundler complexity.

**Commit (code):** N/A — documentation update only in this step.

### What I did

- Rewrote `design/02-xgoja-v2-spec-and-migration-architecture.md` around the rule:
  - If code runs in goja, xgoja may compile or bundle it.
  - If code runs somewhere else, build it outside xgoja and embed/serve the output as assets.
- Removed normal-user v2 config for:
  - `engine`;
  - `platform`;
  - `target`;
  - `format`;
  - package manager installation;
  - browser bundle artifacts;
  - loaders and polyfills.
- Kept concise source compile intent:
  - `language: javascript|typescript`;
  - `compile.mode: runtime|build-time|preserve`;
  - `compile.bundle: true|false`;
  - optional `compile.check.command`.
- Added explicit guidance that frontend/Redux browser bundles are external build outputs represented as `kind: assets` sources.
- Preserved future room for package imports such as Redux in goja-executed source under the existing `bundle: true` semantics.

### Why

- The execution engine for xgoja-managed code is always the xgoja/goja runtime, so ordinary specs should not expose low-level compiler backend settings.
- Non-goja bundles have different concerns—package managers, browser platform, loaders, polyfills, CSS, SVG, DOM assumptions—that do not belong in xgoja's core file format.
- A smaller spec is easier to implement, validate, document, and migrate to.

### What worked

- The simplified example is much shorter while still covering the TypeScript jsverbs use case.
- The v2 spec still supports future goja-runtime package bundling by keeping `compile.bundle: true` as an intent-level field.
- The external asset example cleanly covers browser/frontend build outputs without making xgoja responsible for building them.

### What didn't work

- No command failed in this step.
- The design remains documentation-only; no v2 parser or migration tooling exists yet.

### What I learned

- The important distinction is not TypeScript vs JavaScript. It is goja-executed source vs non-goja output.
- xgoja should own a small set of compiler profiles internally rather than making users configure esbuild mechanics.
- Deferring npm package bundling implementation does not require schema complexity now; `bundle: true` can grow from local imports to package imports later.

### What was tricky to build

- The document needed to preserve future Redux/package-bundling possibilities without adding package-manager fields now. The solution is semantic: define `bundle: true` as bundling dependencies for goja-executed source, initially local dependencies and later package dependencies.
- The document also needed to explain external frontend assets clearly so future readers do not assume xgoja must build every JavaScript artifact it embeds.

### What warrants a second pair of eyes

- Review whether `sources.kind: assets` plus `artifacts.type: embedded-assets` is the right minimal shape for external build outputs.
- Review whether `compile.mode: build-time` should be allowed in the MVP or initially limited to runtime TypeScript behavior plus generated artifacts.
- Review whether `language` should be inferred by default or required for goja-executed source.

### What should be done in the future

- Upload the simplified v2 document to reMarkable.
- Commit the simplified document after validation.
- Implement `specv2` DTOs using the simplified schema, not the broader bundler schema.

### Code review instructions

- Start with the `Core design rule` and `Complete v2 example` sections.
- Verify that low-level esbuild settings are absent from normal v2 examples.
- Compare the asset section to current asset embedding docs to ensure externally-built frontend bundles are still covered.

### Technical details

- `compile.bundle: true` remains the extension point for future goja-runtime package bundling.
- Runtime module aliases should be derived from `runtime.modules` and externalized automatically during bundling.
- Browser/frontend bundles are represented as `sources.kind: assets` after external build tooling produces their output directory.
