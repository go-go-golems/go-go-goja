---
Title: Diary
Ticket: XGOJA-003
Status: active
Topics:
    - xgoja
    - goja
    - engine
    - runtime
    - modules
    - refactor
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: Builder/factory runtime module registration and default module controls (commits 29e07e7
    - Path: engine/options.go
      Note: Explicit opt-outs for implicit default module exposure (commit 5911a8d)
    - Path: engine/runtime_modules.go
      Note: Unified RuntimeModuleSpec contract (commit 29e07e7)
    - Path: pkg/doc/13-plugin-developer-guide.md
      Note: Public docs updated for RuntimeModuleSpec API (commit 7ff4dac)
    - Path: pkg/xgoja/app/factory.go
      Note: xgoja provider modules adapted into engine.Runtime (commits 29e07e7
    - Path: pkg/xgoja/app/root.go
      Note: xgoja jsverbs now invoke through engine runtime path (commit 29e07e7)
    - Path: ttmp/2026/05/23/XGOJA-003--unify-engine-runtime-module-registration/design-doc/01-runtime-aware-module-registration-design-and-implementation-guide.md
      Note: Primary design and migration guide for the hard cutover
ExternalSources: []
Summary: Implementation diary for unifying engine module registration around one runtime-aware RegisterRuntimeModule contract and adapting xgoja to use engine runtimes safely.
LastUpdated: 2026-05-23T18:15:07-04:00
WhatFor: Use this diary to resume or review XGOJA-003 implementation work.
WhenToUse: Read before changing engine module registration APIs, xgoja runtime construction, or jsverbs invocation paths.
---


# Diary

## Goal

This diary records the hard cutover from split engine module registration APIs to one runtime-aware module API, plus the xgoja adaptation that should let generated apps use `engine.Runtime` safely from xgoja specs.

## Step 1: Create the runtime-aware module registration ticket and design

This step created the follow-up ticket for the engine/xgoja runtime abstraction work. The previous review discussion showed that xgoja's separate lightweight runtime existed partly because `engine.ModuleSpec` could not receive runtime context. The new design replaces the split between static modules and runtime registrars with one runtime-aware `RegisterRuntimeModule` contract.

The design is intentionally a hard cutover. It does not preserve compatibility wrappers or legacy builder methods. That keeps the API smaller and makes each module registration path explicit: if a value registers modules into a runtime, it receives `RuntimeModuleContext`.

### Prompt Context

**User prompt (verbatim):** "Ok, create a ticket for that, hard cutover, add small migration guide. Add design + implementation guide, then tasks to the ticket, and implement them one by one, keeping a diary as you work, committing at appropriat eintervals. No backwards compatibility or legacy wrappers or so."

**Assistant interpretation:** Create a new docmgr ticket for the hard cutover to a runtime-aware engine module registration API, write a design/migration guide, add tasks, then implement incrementally with commits and diary updates.

**Inferred user intent:** Reduce xgoja/engine runtime duplication by making engine module registration always runtime-aware, without carrying old APIs forward.

### What I did

- Created ticket `XGOJA-003 — Unify engine runtime module registration`.
- Added design doc `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/23/XGOJA-003--unify-engine-runtime-module-registration/design-doc/01-runtime-aware-module-registration-design-and-implementation-guide.md`.
- Added this diary.
- Added implementation tasks for API cutover, built-in module updates, call-site migration, xgoja runtime adaptation, and validation.

### Why

- The existing split between `ModuleSpec.Register(reg)` and `RuntimeModuleRegistrar.RegisterRuntimeModules(ctx, reg)` makes runtime-aware modules a special case.
- xgoja provider modules naturally need runtime context and config. A unified runtime-aware engine module API should let xgoja use `engine.Runtime` without losing spec-selected module semantics.

### What worked

- The ticket, design doc, diary, and tasks were created.
- The design identifies the hard-cutover migration path and the validation set before implementation starts.

### What didn't work

- The generated task list initially included the default placeholder task `Add tasks here`. This should be removed or marked obsolete during task cleanup.

### What I learned

- The cleanest API is not an additional wrapper structure. It is replacing both existing module registration concepts with one runtime-aware module concept.

### What was tricky to build

- The design has to preserve engine's current default builder behavior while allowing xgoja to avoid default module exposure. The distinction is in how the builder is used: a plain engine builder can still select default modules, while xgoja must always pass explicit spec-selected modules.

### What warrants a second pair of eyes

- Review whether `RuntimeModuleSpec` should replace every old registrar name in one commit or whether a mechanical rename commit should be separated from xgoja runtime adaptation.
- Review whether `RuntimeInitializer` should remain separate. The current design keeps it separate because it runs after `require` is enabled.

### What should be done in the future

- Implement tasks in order and commit after focused phases.

### Code review instructions

- Start with the design guide before reviewing code changes.
- During implementation, review engine API changes before xgoja adaptation.

### Technical details

Target interface:

```go
type RuntimeModuleSpec interface {
    ID() string
    RegisterRuntimeModule(ctx *RuntimeModuleContext, reg *require.Registry) error
}
```

## Step 2: Cut over engine modules and adapt xgoja to engine runtimes

This step implemented the hard cutover described in the design guide. The engine now has one module registration contract, `RuntimeModuleSpec`, and every in-repo runtime-scoped module path uses `RegisterRuntimeModule(ctx, reg)` plus `WithModules(...)`.

The xgoja runtime factory now adapts spec-selected provider modules into engine runtime modules and returns `*engine.Runtime`. A follow-up hardening pass made that engine reuse explicit: generated xgoja runtimes disable implicit engine default modules so their `require()` surface remains defined by the xgoja buildspec.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue implementing the XGOJA-003 hard cutover, finish the remaining migration tasks, validate the result, and keep ticket docs current.

**Inferred user intent:** Complete the runtime-aware engine module API migration without legacy wrappers and preserve xgoja's explicit module-selection semantics.

**Commit (code):** 29e07e7b73b282750225f64a5a86bea90f302348 — "Unify engine runtime module registration"

**Commit (code):** 5911a8dd87d58bd34cb8f1ca5b2eafb4b75552cd — "Keep xgoja engine runtimes explicit"

**Commit (docs):** 7ff4dac383b02739711b91b73e84fdf5ea44775b — "Docs: update runtime module API references"

### What I did

- Replaced the engine module split with `RuntimeModuleSpec` in `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/runtime_modules.go`.
- Updated built-in module specs in `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/module_specs.go` to implement `RegisterRuntimeModule`.
- Updated `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/factory.go` so `FactoryBuilder` and `Factory` store `[]RuntimeModuleSpec` and invoke `RegisterRuntimeModule` during runtime creation.
- Removed the `WithRuntimeModuleRegistrars` call path and migrated call sites in express, ui.dsl, HashiCorp plugin host setup/tests, doc access runtime tests, goja HTTP tests, REPL evaluator configuration, jsverbs CLI runtime creation, and `cmd/goja-repl`.
- Reworked `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/factory.go` so xgoja provider modules are adapted into engine runtime module specs and xgoja returns `*engine.Runtime` via a type alias.
- Updated `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/root.go` to invoke jsverbs through `jsverbs.InvokeInRuntime`.
- Added engine builder options in `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/options.go` to disable implicit default-registry fallback and automatic data-only default modules.
- Configured xgoja runtime creation to use those options so generated runtimes expose only buildspec-selected modules.
- Added regression tests in `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/runtime_modules_test.go` and `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/root_test.go`.
- Updated public bundled docs in `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/doc/13-plugin-developer-guide.md`, `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/doc/18-express-module.md`, and `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/doc/19-uidsl-module.md`.
- Checked off XGOJA-003 tasks 2 through 6 and updated the changelog.

### Why

- The old `ModuleSpec` / `RuntimeModuleRegistrar` split made runtime-aware modules a special-case path and prevented xgoja from cleanly reusing `engine.Runtime`.
- xgoja needed the engine lifecycle substrate, owner, runtimebridge bindings, and `InvokeInRuntime` path, but it also needed to keep module exposure spec-driven.

### What worked

- The mechanical API migration compiled cleanly after renaming remaining in-repo call sites from `RegisterRuntimeModules` to `RegisterRuntimeModule` and from `WithRuntimeModuleRegistrars` to `WithModules`.
- Focused validation passed with:
  - `GOWORK=off go test ./engine ./pkg/runtimebridge ./pkg/jsverbs ./pkg/xgoja/app ./cmd/xgoja/internal/generate ./cmd/xgoja ./cmd/xgoja/internal/buildspec ./pkg/xgoja/providerapi ./pkg/xgoja/testprovider ./pkg/xgoja/testcobra ./pkg/xgoja/testadapter ./modules/express ./modules/uidsl ./pkg/hashiplugin/host ./pkg/repl/evaluators/javascript ./pkg/docaccess/runtime ./pkg/jsverbscli ./pkg/gojahttp ./pkg/doc -count=1`
- The runnable xgoja smoke examples passed:
  - `make -C examples/xgoja/runtime-filesystem smoke`
  - `make -C examples/xgoja/embedded-jsverbs smoke`
  - `make -C examples/xgoja/provider-shipped-jsverbs smoke`

### What didn't work

- The first xgoja engine adaptation still inherited engine's unconditional data-only default modules. That preserved engine behavior, but it was not strict enough for generated xgoja binaries because xgoja module exposure should come from the buildspec.
- I fixed this by adding explicit builder controls and configuring xgoja with `WithImplicitDefaultRegistryModules(false)` and `WithDataOnlyDefaultRegistryModules(false)`.

### What I learned

- Reusing `engine.Runtime` safely required two separate decisions: unify the module registration interface, and make implicit engine defaults configurable at builder construction time.
- The hard cutover simplified module call sites, but preserving xgoja policy still needed a small explicit engine option surface rather than relying on convention.

### What was tricky to build

- The main sharp edge was default module exposure. `engine.NewBuilder().Build()` historically exposes default modules, and `engine.NewRuntime` also installed data-only modules. Those defaults are useful for normal engine callers, but xgoja generated runtimes must not gain extra modules just because they moved onto `engine.Runtime`.
- The solution was to preserve default behavior for normal engine callers while adding explicit opt-out options for generated runtimes. The regression test checks both the engine opt-out path and the xgoja no-implicit-defaults path by verifying that `require("path")` fails while a spec-selected provider module still loads.

### What warrants a second pair of eyes

- Review the names and semantics of `WithImplicitDefaultRegistryModules(false)` and `WithDataOnlyDefaultRegistryModules(false)`; they are intentionally explicit, but they are new public engine builder options.
- Review whether automatic data-only module installation should remain enabled by default for all non-xgoja engine callers.
- Review xgoja provider module IDs for duplicate detection; aliases are included in the ID to keep same provider/name instances distinct when mounted under different aliases.

### What should be done in the future

- Consider closing XGOJA-003 after `docmgr doctor` passes and any desired final review docs are updated.
- Historical ticket docs still mention the old registrar API; update only if those historical docs are expected to serve as current reference material.

### Code review instructions

- Start with `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/runtime_modules.go`, `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/factory.go`, and `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/options.go` to review the engine API and default-module controls.
- Then review `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/factory.go` and `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/root.go` to confirm generated xgoja runtimes now use `engine.Runtime` and `InvokeInRuntime`.
- Validate with the focused `GOWORK=off go test ...` command above and the three `examples/xgoja/*` smoke commands.

### Technical details

The final runtime module contract is:

```go
type RuntimeModuleSpec interface {
    ID() string
    RegisterRuntimeModule(ctx *RuntimeModuleContext, reg *require.Registry) error
}
```

The xgoja runtime factory now creates an engine builder like this:

```go
builder := engine.NewBuilder(
    engine.WithImplicitDefaultRegistryModules(false),
    engine.WithDataOnlyDefaultRegistryModules(false),
).WithModules(modules...)
```
