---
Title: Sobek ESM Research Logbook
Ticket: GOJA-053
Status: active
Topics:
    - xgoja
    - goja
    - javascript
    - module
    - research
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/others/sobek/AGENTS.md
      Note: |-
        Sobek repository guidance resource evaluated for architecture and event-loop warnings.
        Resource evaluated in Sobek ESM logbook
    - Path: ../../../../../../../../../../code/others/sobek/README.md
      Note: |-
        Sobek README resource evaluated for ESM caveats and documentation status.
        Resource evaluated in Sobek ESM logbook
    - Path: ../../../../../../../../../../code/others/sobek/modules.go
      Note: |-
        Sobek core ModuleRecord resource evaluated for ESM linking/evaluation APIs.
        Resource evaluated in Sobek ESM logbook
    - Path: ../../../../../../../../../../code/others/sobek/modules_integration_test.go
      Note: |-
        Sobek integration tests resource evaluated for custom native ModuleRecord examples.
        Resource evaluated in Sobek ESM logbook
    - Path: ../../../../../../../../../../code/others/sobek/modules_sourcetext.go
      Note: |-
        Sobek source-text module resource evaluated for ParseModule, Link, and Evaluate behavior.
        Resource evaluated in Sobek ESM logbook
    - Path: ../../../../../../../../../../code/others/sobek/modules_test.go
      Note: |-
        Sobek ESM unit tests resource evaluated for resolver and dynamic import examples.
        Resource evaluated in Sobek ESM logbook
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/go-go-parc/Projects/2026/04/12/PROJ - Goja vs Sobek Deep Analysis.md
      Note: |-
        Prior Goja-vs-Sobek analysis resource evaluated for fork relationship and ESM decision framing.
        Resource evaluated in Sobek ESM logbook
    - Path: go-go-goja/engine/factory.go
      Note: Current Goja/goja_nodejs engine construction resource evaluated for migration impact.
    - Path: go-go-goja/modules/common.go
      Note: |-
        Current CommonJS NativeModule resource evaluated for native module interface comparison.
        Resource evaluated in Sobek ESM logbook
    - Path: go-go-goja/pkg/doc/bun-goja-bundling-playbook.md
      Note: |-
        Bundling alternative resource evaluated for ESM authoring without Sobek migration.
        Bundling-to-CJS alternative evaluated in Sobek ESM logbook
ExternalSources: []
Summary: Tracks resources used for the Sobek/ESM native module analysis, including usefulness, stale/confusing areas, and update needs.
LastUpdated: 2026-06-04T00:00:00Z
WhatFor: Use when revising the Sobek ESM analysis, deciding whether to prototype a Sobek backend, or planning cleanup around xgoja's require/native module documentation.
WhenToUse: Before implementing a Sobek ESM spike, native ESM ModuleRecord adapter, CommonJS-to-ESM bridge, or goja/sobek engine abstraction.
---


# Sobek ESM Research Logbook

## Purpose

This logbook records the resources used for `design/06-sobek-esm-native-module-analysis.md`. It is specifically about the question: would Sobek's ECMAScript Module support reduce xgoja's current `require()` / module factory / native module machinery, and what would an ESM-based native module design look like?

Each entry includes:

- what was being researched,
- what was being looked for in the document or source file,
- why the resource was chosen,
- how the resource was found,
- what was useful,
- what was not useful,
- what is out of date or wrong,
- what would need updating.

---

## Resource 1: `/home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/04/12/PROJ - Goja vs Sobek Deep Analysis.md`

### What I was researching

The prior local analysis comparing Goja and Sobek, especially the fork relationship, ESM feature delta, and decision guidance.

### What I was looking for in this document in particular

- Whether Sobek is a maintained fork or a divergent engine.
- Whether ESM is the main functional difference.
- How much code ESM adds.
- Whether previous research recommended using Sobek for this class of problem.
- Any warnings about ESM complexity or API stability.

### Why I chose it

The user explicitly pointed to this note as existing project knowledge. It is the strongest local prior about whether Sobek is a good move.

### How I found the resource itself

The user provided the absolute path directly.

### What I found useful in the document

- It states three key findings: Sobek tracks Goja closely, ESM is the major difference, and Sobek is k6-driven.
- It quantified the ESM delta as modest in code size but significant in conceptual overhead.
- It listed the Sobek ESM files: `modules.go`, `modules_sourcetext.go`, `modules_namespace.go`, `modules_test.go`, and `modules_integration_test.go`.
- It included the important working rule: prefer Goja unless ESM is specifically needed.
- It identified `esmrefactor` as an open branch/API area to monitor, which means the current ESM API may not be final.

### What I didn't find useful

- It did not analyze xgoja's current provider/module factory architecture.
- It did not determine whether native xgoja provider modules should become ESM module records.
- It did not benchmark xgoja or run current go-go-goja code against Sobek.

### What is out of date / what was wrong

- The note refers to an `esmrefactor` branch and generated reports from an earlier project state. It may need refresh if Sobek's branch merged or API changed after that analysis.
- The note's line counts and commit sync claims should be rechecked before using them as final production migration evidence.

### What would need updating

- Re-run fork sync and diff analysis against current Goja/Sobek heads.
- Update the status of Sobek's ESM API if `esmrefactor` merged or was abandoned.
- Add a section specifically about xgoja/goja_nodejs migration impact.

---

## Resource 2: `/home/manuel/code/others/sobek/README.md`

### What I was researching

Sobek's official project-level description of ESM support, caveats, and runtime/event-loop expectations.

### What I was looking for in this document in particular

- Whether ESM is considered stable.
- Whether Sobek provides documentation or points to tests.
- Whether Sobek provides an event loop.
- How Sobek positions itself relative to Goja.

### Why I chose it

The README is the top-level statement from the Sobek repository and should be cited before relying on source internals.

### How I found the resource itself

The user provided `/home/manuel/code/others/sobek`; I listed and searched that repository, then read `README.md`.

### What I found useful in the document

- It states Sobek is a fork of Goja.
- It labels ESM support experimental.
- It says the ESM API is close to the specification but not great for actual usage.
- It warns the ESM API is likely to change in a breaking manner.
- It says there is no concrete documentation and recommends reading `modules_test.go` and `modules_integration_test.go`.
- It says ESM requires an event loop implementation and Sobek does not provide one.

### What I didn't find useful

- It does not provide an implementation guide for ESM.
- It does not explain how to expose native Go modules as ESM modules.
- It does not discuss xgoja, goja_nodejs, or CommonJS interop.

### What is out of date / what was wrong

- Not obviously wrong, but the “no concrete documentation” caveat means any implementation must rely on source/tests and may break later.

### What would need updating

- Add a concrete ESM usage guide if Sobek expects embedders beyond k6.
- Document a recommended host resolver/event-loop pattern.
- Document native ModuleRecord examples beyond tests.

---

## Resource 3: `/home/manuel/code/others/sobek/AGENTS.md`

### What I was researching

Repository-maintainer guidance summarizing Sobek architecture and gotchas.

### What I was looking for in this document in particular

- A concise architecture summary.
- Any direct warnings about ESM.
- Any warnings about runtime threading, event loops, promises, or object transfer.

### Why I chose it

AGENTS.md is often more direct than README prose and can surface operational constraints important for agent-driven design work.

### How I found the resource itself

It appeared in the file listing of `/home/manuel/code/others/sobek`.

### What I found useful in the document

- It states Sobek uses parse → compile → execute.
- It emphasizes one runtime per goroutine and no object transfer between runtimes.
- It states ESM support exists but is experimental.
- It states ESM requires the embedder to provide an event loop and Sobek has none.
- It warns promise job queues drain synchronously when the top-level script function returns.

### What I didn't find useful

- It is high-level and does not include APIs or code examples for modules.

### What is out of date / what was wrong

- Nothing obviously wrong.
- If Sobek adds a formal event loop or changes ESM API, this should be updated.

### What would need updating

- Add a pointer to the best current ESM tests/examples.
- Clarify how dynamic import should be hosted in production embedders.

---

## Resource 4: `/home/manuel/code/others/sobek/modules.go`

### What I was researching

Sobek's core ECMAScript Module interfaces and algorithms.

### What I was looking for in this document in particular

- Definitions of `ModuleRecord`, `CyclicModuleRecord`, `ModuleInstance`, and `CyclicModuleInstance`.
- How linking and evaluation work.
- Dynamic import hooks.
- import.meta hooks.
- TODO comments indicating instability.

### Why I chose it

This is the central Sobek ESM implementation file. Any native ESM module design must implement or interact with these interfaces.

### How I found the resource itself

Repository search for module APIs:

```bash
rg -n "ModuleRecord|ParseModule|SetImportModuleDynamically|HostResolveImportedModule" /home/manuel/code/others/sobek -S
```

### What I found useful in the document

- `HostResolveImportedModuleFunc` defines host import resolution.
- `ModuleRecord` requires `GetExportedNames`, `ResolveExport`, `Link`, and `Evaluate`.
- `CyclicModuleRecord` adds `RequestedModules`, `InitializeEnvironment`, and `Instantiate`.
- `CyclicModuleRecordConcreteLink` and `innerModuleLinking` show the DFS-style link algorithm.
- `CyclicModuleRecordEvaluate` evaluates modules and returns a Promise.
- Dynamic import uses `SetImportModuleDynamically` and `FinishLoadingImportModule`.
- import.meta hooks are exposed through `SetGetImportMetaProperties` and `SetFinalImportMeta`.

### What I didn't find useful

- Many comments are TODOs, and some names are acknowledged as likely to change.
- It is specification-oriented and not ergonomic as an embedder API.
- It does not provide a ready-made native module adapter.

### What is out of date / what was wrong

- The file itself says things like “most things here probably should be unexported and names should be revised before merged in master” and “TODO fix signature.” This confirms API instability.
- Typos such as “wether,” “dependancies,” and “type inferance” are harmless but signal rough edges.

### What would need updating

- Stabilize public API names before xgoja depends on them.
- Provide a documented synthetic/native module helper.
- Add clearer error handling around `ResolveExport` paths that currently panic in some cases.

---

## Resource 5: `/home/manuel/code/others/sobek/modules_sourcetext.go`

### What I was researching

How Sobek parses source-text ESM modules and turns import/export syntax into module records.

### What I was looking for in this document in particular

- `ParseModule` behavior.
- `ModuleFromAST` behavior.
- How requested modules, import entries, export entries, and star exports are derived.
- How `Link`, `Evaluate`, and `Instantiate` are implemented.

### Why I chose it

The design needed to explain source module flow: parse → link → instantiate/evaluate. This file implements `SourceTextModuleRecord`.

### How I found the resource itself

The prior analysis listed it as one of Sobek's five ESM files; `rg` also found `ParseModule` here.

### What I found useful in the document

- `ParseModule` appends `parser.IsModule` and calls `Parse`.
- `ModuleFromAST` builds `requestedModules`, `importEntries`, `localExportEntries`, `indirectExportEntries`, and `starExportEntries`.
- Duplicate export names are checked early.
- `InitializeEnvironment` compiles the module.
- `Instantiate` creates `SourceTextModuleInstance` with `exportGetters` and runs the compiled program.
- `Evaluate` delegates to `Runtime.CyclicModuleRecordEvaluate`.

### What I didn't find useful

- It does not explain host-level resolver design.
- It does not explain native module records; those appear only in tests.
- Some TODOs and comments are rough, so it needs careful reading.

### What is out of date / what was wrong

- Comments such as “TODO arguments to this need fixing,” “realm isn't implement,” and “TODO figure out omething less idiotic” indicate rough implementation/documentation areas.
- The API may not be stable enough for a large migration.

### What would need updating

- Add public examples for source module parsing/evaluation.
- Clarify which parts of `SourceTextModuleRecord` are intended public API.
- Fix comments and TODOs if Sobek ESM is promoted from experimental.

---

## Resource 6: `/home/manuel/code/others/sobek/modules_test.go`

### What I was researching

Sobek's ESM usage patterns in tests, especially source module resolution and dynamic imports.

### What I was looking for in this document in particular

- A runnable resolver pattern.
- How tests link and evaluate source modules.
- How dynamic import is wired.
- How event-loop queues are simulated.

### Why I chose it

Sobek README says tests are the recommended documentation for ESM usage.

### How I found the resource itself

The Sobek README explicitly points to `modules_test.go`.

### What I found useful in the document

- `runModules` implements a simple map-backed resolver with caching.
- It shows how to call `ParseModule`, `Link`, and `Evaluate`.
- It shows a minimal `SetImportModuleDynamically` callback.
- It shows use of `FinishLoadingImportModule`.
- It demonstrates missing/ambiguous export errors.
- It confirms top-level await and dynamic imports make evaluation Promise-based.

### What I didn't find useful

- It is test code, not a production event loop.
- The event-loop queue is explicitly minimal and likely not production-safe.
- It does not show xgoja provider module adaptation.

### What is out of date / what was wrong

- The test helper comments describe the event loop as “the most basic and likely buggy event loop,” which is not a production pattern.

### What would need updating

- Add a proper example package for ESM host resolution.
- Add production-quality dynamic import guidance.

---

## Resource 7: `/home/manuel/code/others/sobek/modules_integration_test.go`

### What I was researching

How Sobek supports non-source/native module records and custom module instances.

### What I was looking for in this document in particular

- Examples of custom `ModuleRecord` implementations.
- How a native module can export values.
- How cyclic custom module records work.
- How import.meta is configured.

### Why I chose it

The central question was whether native/provider modules can be exposed as ESM. This file contains the clearest local examples of custom non-source module records.

### How I found the resource itself

The Sobek README explicitly points to `modules_integration_test.go`; `rg` also found custom module implementations there.

### What I found useful in the document

- `simpleModuleImpl` implements `ModuleRecord` and exposes `coolStuff`.
- `simpleModuleInstanceImpl.GetBindingValue` returns `rt.ToValue(5)`.
- `cyclicModuleImpl` implements `CyclicModuleRecord` and models exports that resolve through other modules.
- Dynamic import tests show how `SetImportModuleDynamically` can resolve custom/native modules.
- `TestSourceMetaImport` shows `SetGetImportMetaProperties`.

### What I didn't find useful

- The custom modules are minimal test fixtures, not ergonomic helpers.
- The dynamic import example includes FIXME/hack comments.
- There is no reusable `NativeModuleRecord` helper in the Sobek package.

### What is out of date / what was wrong

- The test contains comments like `// FIXME haxx` and typo fields such as `bidning`, indicating roughness.
- It proves feasibility, not production readiness.

### What would need updating

- Extract or document a supported native/synthetic module helper.
- Clean up typos/comments if the file is the official ESM documentation.
- Add tests for default-only object modules because that is likely the easiest xgoja bridge.

---

## Resource 8: `/home/manuel/code/others/sobek/modules_namespace.go`

### What I was researching

How Sobek creates module namespace objects for `import * as ns` and dynamic import results.

### What I was looking for in this document in particular

- `Runtime.NamespaceObjectFor` behavior.
- How namespace objects read live binding values.
- Whether namespace objects are cached per ModuleRecord.

### Why I chose it

ESM native modules often expose namespace objects, and dynamic import resolves to a namespace object. Understanding this is needed for CJS/ESM interop design.

### How I found the resource itself

`rg` found `NamespaceObjectFor` references in Sobek module files and tests.

### What I found useful in the document

- Namespace objects are cached in `runtime.moduleNamespaces`.
- Namespace properties resolve through module bindings, not static copies.
- Dynamic import resolution in `modules.go` uses `NamespaceObjectFor(module)`.

### What I didn't find useful

- It did not directly answer how to implement native module records; it is mainly a runtime object implementation detail.

### What is out of date / what was wrong

- Nothing obvious from the limited inspection.

### What would need updating

- If Sobek ESM docs are created, add a section explaining namespace object behavior and live binding implications.

---

## Resource 9: `/home/manuel/code/others/sobek/go.mod`

### What I was researching

Sobek's module path and dependency footprint.

### What I was looking for in this document in particular

- The Go import path.
- Minimum Go version.
- Whether Sobek imports/provides goja_nodejs-like packages.

### Why I chose it

A migration needs to know whether Sobek is a drop-in Go import replacement and whether it brings Node compatibility packages.

### How I found the resource itself

It appeared in the Sobek repository root listing.

### What I found useful in the document

- Module path is `github.com/grafana/sobek`.
- Go version is `1.25`.
- Dependencies do not include a `goja_nodejs` equivalent.

### What I didn't find useful

- It does not explain API compatibility beyond the module path.

### What is out of date / what was wrong

- Nothing obvious.

### What would need updating

- If Sobek creates Node compatibility packages, the module dependency story should be revisited.

---

## Resource 10: `go-go-goja/modules/common.go`

### What I was researching

The current native module interface in go-go-goja and how it maps to CommonJS.

### What I was looking for in this document in particular

- The `NativeModule` interface.
- How modules register globally.
- How the default module registry works.
- How loaders are typed.

### Why I chose it

To compare current CommonJS native modules with hypothetical ESM native modules, I needed the current contract.

### How I found the resource itself

Repository search for `NativeModule`, `modules.Register`, and `RegisterNativeModule`.

### What I found useful in the document

- `NativeModule.Loader(*goja.Runtime, *goja.Object)` is explicitly goja/goja_nodejs-oriented.
- The loader receives a module object whose `exports` property is populated.
- `modules.DefaultRegistry` stores native modules and `Enable` registers them into a `require.Registry`.
- This is clearly CommonJS-shaped.

### What I didn't find useful

- It does not expose static export names, which are needed for named ESM imports.
- It cannot be reused directly with `*sobek.Runtime` because the types differ.

### What is out of date / what was wrong

- Not wrong for CommonJS.
- The docs point to old `ttmp/2025-06-21/01-goja-initial-plan.md`, which may not be the best current architecture doc.

### What would need updating

- Add an optional ESM/native export metadata interface if an ESM backend is pursued.
- Add docs explaining CommonJS-only assumptions.

---

## Resource 11: `go-go-goja/modules/exports.go`

### What I was researching

How modules currently attach individual exports to CommonJS `module.exports` objects.

### What I was looking for in this document in particular

- Whether export names are centrally declared.
- Whether there is a reusable export abstraction that could feed ESM named exports.

### Why I chose it

If xgoja already had a central export metadata abstraction, ESM named exports would be easier.

### How I found the resource itself

From `rg` results showing many modules call `modules.SetExport`.

### What I found useful in the document

- `SetExport` is a simple helper around `exports.Set(name, value)`.

### What I didn't find useful

- It does not retain metadata about export names/types.
- It does not provide a reusable value builder independent of `goja.Object`.

### What is out of date / what was wrong

- Not wrong; it is intentionally minimal.

### What would need updating

- For ESM named exports, introduce a richer export declaration type that can generate CommonJS exports, ESM bindings, and TypeScript declarations from one source.

---

## Resource 12: `go-go-goja/modules/fs/fs.go`

### What I was researching

A representative complex native module that uses runtimebridge and populates many CommonJS exports.

### What I was looking for in this document in particular

- How a current native module populates exports.
- Whether it needs runtime services.
- Whether its exports are static enough for ESM named export metadata.

### Why I chose it

`fs` is a central xgoja module and has both async and sync functions. It is a useful concrete example for ESM adaptation.

### How I found the resource itself

`rg` showed `modules/fs/fs.go` implementing `modules.NativeModule` and using `runtimebridge.Lookup`.

### What I found useful in the document

- The loader gets `exports` from `moduleObj.Get("exports")`.
- It calls `runtimebridge.Lookup(vm)`, so native modules need runtime services beyond raw VM access.
- It attaches many named functions with `modules.SetExport`.
- Export names are mostly static and could be represented as ESM named exports if refactored.

### What I didn't find useful

- The exports are attached procedurally, so there is no centralized export declaration list.
- The module is typed to Goja and `goja_nodejs/buffer`.

### What is out of date / what was wrong

- Not wrong for CommonJS.
- As an ESM source, it would need refactoring because current export population happens after the loader is called.

### What would need updating

- Extract export declarations or builder functions if named ESM exports are desired.
- Port buffer handling if Sobek mode needs async file APIs returning Buffer-like values.

---

## Resource 13: `go-go-goja/engine/factory.go`

### What I was researching

Current Goja/goja_nodejs runtime construction and what a Sobek backend would need to replace.

### What I was looking for in this document in particular

- Creation of `goja.Runtime`.
- Creation of `goja_nodejs/eventloop`.
- `runtimebridge.Store` timing.
- `require.NewRegistry` and `reg.Enable(vm)`.
- `console`, `buffer`, and `url` installation.

### Why I chose it

A Sobek migration would hit this file immediately. It is the center of current runtime construction.

### How I found the resource itself

It was already part of the lifecycle runthrough and appears in `app.RuntimeFactory.NewRuntime` flow.

### What I found useful in the document

- Current engine imports `goja`, `goja_nodejs/buffer`, `console`, `eventloop`, `require`, and `url`.
- Runtimebridge services are stored before module registration.
- `require.Registry` is created and enabled during runtime construction.
- This shows Sobek cannot be a one-file replacement unless all goja_nodejs pieces are also replaced.

### What I didn't find useful

- It does not contain ESM concepts because current engine is CommonJS/require-oriented.

### What is out of date / what was wrong

- Not wrong.
- It deserves documentation distinguishing `require.Registry` from any future ESM resolver/module graph.

### What would need updating

- If adding Sobek mode, this file should not become a pile of backend branches. Prefer a separate experimental package or a backend abstraction.
- Add comments around Node primitive installation to clarify migration impact.

---

## Resource 14: `go-go-goja/engine/module_specs.go`

### What I was researching

How current engine module specs register native modules into the require registry.

### What I was looking for in this document in particular

- `NativeModuleSpec` shape.
- `RegisterRuntimeModule` behavior.
- Default registry module registration.
- Data-only default modules and aliases.

### Why I chose it

It is the bridge from engine module selection to CommonJS native module registration.

### How I found the resource itself

From the lifecycle runthrough and `rg` results for `RegisterNativeModule`.

### What I found useful in the document

- `NativeModuleSpec` stores `ModuleName` and `require.ModuleLoader`.
- Registration is `reg.RegisterNativeModule(name, loader)`.
- `namedDefaultRegistryModulesSpec` fetches modules from `modules.DefaultRegistry`.
- Alias expansion for `node:fs`, `node:path`, etc. is implemented here.

### What I didn't find useful

- There is no ESM equivalent to `RuntimeModuleSpec` yet.
- It is tightly coupled to `require.Registry`.

### What is out of date / what was wrong

- Not wrong for current engine.

### What would need updating

- Add a parallel `ESMModuleSpec` only in a Sobek spike, not in production code first.
- If ESM native modules become real, decide whether aliases are resolver aliases rather than registration aliases.

---

## Resource 15: `go-go-goja/engine/module_roots.go`

### What I was researching

Current CommonJS module resolution roots for script execution.

### What I was looking for in this document in particular

- How script-local module roots are computed.
- How `node_modules` roots are included.
- Whether this resolver logic could inform an ESM resolver.

### Why I chose it

An ESM host resolver would need path resolution rules. Current CommonJS runner already derives roots from script paths.

### How I found the resource itself

From `run.go`, which calls `RequireOptionWithModuleRootsFromScript`.

### What I found useful in the document

- It includes script dir, parent dir, and `node_modules` folders by default.
- It dedupes absolute paths.
- It converts roots into `require.WithGlobalFolders`.

### What I didn't find useful

- ESM resolution rules differ from CommonJS and Node's ESM resolver, so this is only a starting point.
- It does not resolve file extensions or package exports for ESM.

### What is out of date / what was wrong

- Not wrong.

### What would need updating

- If ESM mode is added, create a separate ESM resolver with explicit rules instead of reusing CommonJS roots blindly.
- Document differences between CommonJS and ESM path/package resolution.

---

## Resource 16: `go-go-goja/pkg/xgoja/app/factory.go`

### What I was researching

Where xgoja provider modules become CommonJS `require.ModuleLoader` registrations.

### What I was looking for in this document in particular

- Where `providerapi.Module.New` is called.
- What `ModuleContext` contains at that point.
- How the returned loader is registered.
- Whether this can map to an ESM module record factory.

### Why I chose it

This file is the xgoja provider/native module adapter and is central to both GOJA-053 and any ESM migration.

### How I found the resource itself

It was previously identified in the codegen/runtime runthrough as the `Module.New` insertion point.

### What I found useful in the document

- `providerRuntimeModuleSpec.RegisterRuntimeModule` marshals `ModuleInstance.Config` and calls `s.module.New(...)`.
- It passes `RuntimeOwner`, `Host`, `Name`, `As`, and config into `ModuleContext`.
- It registers the returned `require.ModuleLoader` with `reg.RegisterNativeModule(alias, loader)`.
- This is exactly where an ESM adapter would need to produce/register a `ModuleRecord` instead of a CommonJS loader.

### What I didn't find useful

- It cannot show ESM because there is no current ESM backend.

### What is out of date / what was wrong

- Not wrong.
- For future ESM, `providerapi.Module.New` returning `require.ModuleLoader` is too specific.

### What would need updating

- Keep this CommonJS path stable.
- Add a separate ESM provider path only after a Sobek spike proves useful.
- Continue GOJA-053 pre-`Module.New` config work because ESM will need equivalent setup config.

---

## Resource 17: `go-go-goja/pkg/xgoja/providerapi/module.go`

### What I was researching

The provider module API and whether it can be reused for ESM native modules.

### What I was looking for in this document in particular

- `ModuleFactory` return type.
- `ModuleContext` fields.
- Whether the API is CommonJS-specific.

### Why I chose it

Provider modules are user-facing xgoja extension points. Any ESM redesign must decide whether to change or wrap this API.

### How I found the resource itself

From `app/factory.go` and earlier lifecycle analysis.

### What I found useful in the document

- `ModuleFactory` returns `require.ModuleLoader`, so it is tied to goja_nodejs/CommonJS.
- `ModuleContext.Config` is JSON and remains relevant for ESM factory setup.
- `ModuleContext.RuntimeOwner` and `Host` are backend-lifecycle concepts, not CommonJS concepts.

### What I didn't find useful

- It has no comments explaining which fields are runtime setup-only versus long-lived.
- It has no ESM abstraction.

### What is out of date / what was wrong

- `Context` remains a confusing field name and should likely become `StartupContext` or `SetupContext`.

### What would need updating

- Do not mutate this API for ESM immediately.
- If an ESM path is added, define a separate `ESMModuleFactory` rather than overloading `ModuleFactory`.
- Add comments to `ModuleContext` either way.

---

## Resource 18: `go-go-goja/pkg/xgoja/providerapi/registry.go`

### What I was researching

Whether the provider registry is CommonJS-specific or can remain for ESM.

### What I was looking for in this document in particular

- How packages/modules/capabilities are registered.
- Whether registry entries assume `require()`.
- Whether provider selection remains useful in ESM.

### Why I chose it

The user is trying to reduce module machinery. I needed to separate registry machinery that ESM could remove from registry machinery that remains necessary.

### How I found the resource itself

It was part of the previous lifecycle runthrough and current xgoja provider flow.

### What I found useful in the document

- `providerapi.Registry` stores provider metadata: modules, capabilities, command providers, help sources, and verb sources.
- This registry is not the same as Goja's `require.Registry`.
- ESM would still need provider package selection and capability lookup.

### What I didn't find useful

- `Module` entries currently carry a CommonJS `ModuleFactory`, so the registry can store them but cannot itself make them ESM-capable.

### What is out of date / what was wrong

- Not wrong.

### What would need updating

- Add comments distinguishing provider registry from CommonJS require registry and future ESM resolver/cache.
- Add new entry types only after the ESM spike.

---

## Resource 19: `geppetto/pkg/js/modules/geppetto/module.go`

### What I was researching

A complex real provider/native module and how much work it would be to port from CommonJS loader style to Sobek ESM.

### What I was looking for in this document in particular

- The current `NewLoader` return type.
- Use of `goja` and `goja_nodejs/require`.
- Runtimebridge and runtime owner usage.
- Dynamic API shape that may not map easily to static named exports.

### Why I chose it

Geppetto is a motivating provider for GOJA-053 and one of the more complex modules. If Geppetto cannot migrate easily, a full ESM switch is not a simplification.

### How I found the resource itself

The previous GOJA-053 work and searches for `NewLoader` / `require.ModuleLoader` identified this file.

### What I found useful in the document

- `NewLoader(opts Options) require.ModuleLoader` confirms CommonJS loader orientation.
- The module uses `goja.Runtime`, `require`, runtimebridge, runtime owner, event emitters, tool registries, and profile registries.
- Many APIs are built dynamically on objects, so default-object ESM facade is much easier than static named exports.

### What I didn't find useful

- The file is too large and domain-specific to serve as a generic module adapter example.

### What is out of date / what was wrong

- Not necessarily wrong, but the current config/options shape is already under reconsideration in GOJA-053.

### What would need updating

- If ESM is pursued, start with `import gp from "geppetto"`, not named imports.
- Avoid porting Geppetto until a small native ESM spike works.

---

## Resource 20: `geppetto/pkg/js/modules/geppetto/provider/provider.go`

### What I was researching

How an xgoja provider wraps a native loader and uses `ModuleContext.Config`, `Host`, and setup context.

### What I was looking for in this document in particular

- The provider `Module.New` implementation.
- Whether config is consumed before loader creation.
- Whether ESM would still need pre-module config.

### Why I chose it

The analysis needed to answer whether ESM changes GOJA-053 config timing. This provider demonstrates that config is needed before loader/module construction.

### How I found the resource itself

From earlier GOJA-053 work and from searching `providerapi.Module{` and `New: func(ctx providerapi.ModuleContext)`.

### What I found useful in the document

- The provider decodes `ctx.Config` before calling `geppettomodule.NewLoader(opts)`.
- It calls host services with `ctx.Context`.
- This confirms that an ESM module factory would still need merged config before it creates a native module record or instance.

### What I didn't find useful

- It does not address ESM directly.

### What is out of date / what was wrong

- The current Geppetto config fields may be simplified by GOJA-053 follow-up design work.

### What would need updating

- Keep GOJA-053 config merging backend-agnostic so it can feed future CommonJS or ESM factories.
- Rename/document `ModuleContext.Context` if the broader cleanup happens.

---

## Resource 21: `go-go-goja/pkg/doc/bun-goja-bundling-playbook.md`

### What I was researching

An alternative to Sobek: author modern TypeScript/ESM, bundle to CommonJS, and keep current Goja runtime.

### What I was looking for in this document in particular

- Whether the repository already has a recommended ESM/TypeScript workflow.
- Whether bundling avoids runtime ESM complexity.
- How native modules remain external to the bundle.

### Why I chose it

The user's real goal is reducing complexity. Bundling may satisfy modern JS authoring needs without migrating engines.

### How I found the resource itself

Repository file listing and search results showed `bun-goja-bundling-playbook.md`; it also appeared relevant from previous xgoja docs.

### What I found useful in the document

- It recommends CommonJS bundle output for Goja.
- It explains a simple flow: TS/ESM source → Bun/esbuild → `.cjs` bundle → current `require()` runtime.
- It states native/host modules can remain external (`fs`, plugins, etc.).
- It explicitly says this avoids shipping a full Node runtime.

### What I didn't find useful

- It is about bundled applications, not unbundled runtime ESM imports.
- It does not solve native modules as ESM.

### What is out of date / what was wrong

- Nothing obvious.

### What would need updating

- Add a section comparing bundling-to-CJS with Sobek ESM once the Sobek analysis is accepted.
- Include xgoja-specific examples with provider modules like Geppetto.

---

## Resource 22: `go-go-goja/pkg/doc/02-creating-modules.md`

### What I was researching

Existing documentation for writing go-go-goja native modules.

### What I was looking for in this document in particular

- Current instructions for `NativeModule` authors.
- Whether docs frame modules as `require()` only.
- Whether an ESM migration would require doc updates.

### Why I chose it

If native module architecture changes, this is one of the docs that new contributors will read first.

### How I found the resource itself

Repository docs listing and search results for `NativeModule` and `require()`.

### What I found useful in the document

- It clearly documents the current CommonJS pattern: implement `NativeModule`, populate `exports`, register from `init()`, use `require("module")` from JS.
- It is good evidence that current developer experience is intentionally CommonJS-shaped.

### What I didn't find useful

- It has no ESM discussion.
- It does not distinguish CommonJS as a design choice from JavaScript modules generally.

### What is out of date / what was wrong

- Not wrong today.
- If Sobek ESM is added, this doc would become incomplete.

### What would need updating

- Add a “CommonJS native modules” label.
- Add a future ESM section only after an ESM native module API exists.

---

## Resource 23: `go-go-goja/pkg/doc/16-nodejs-primitives.md`

### What I was researching

Current Node-style primitives and module composition docs.

### What I was looking for in this document in particular

- What primitives current engine installs by default.
- How `require()` is described to users.
- How explicit module selection/sandboxing works.
- What would be lost or need porting in Sobek mode.

### Why I chose it

Sobek does not provide `goja_nodejs` compatibility. This doc shows the current runtime capabilities that depend on the CommonJS/Node-style stack.

### How I found the resource itself

Repository docs listing and search for NodeJS primitives / require modules.

### What I found useful in the document

- It documents default primitives: console, Buffer, URL, performance, crypto, events, path, time, timer.
- It explains opt-in host-access modules like process.
- It documents CommonJS access through `require("...")` and `node:` aliases.
- It supports the claim that Sobek migration would require replacing more than just `require()`.

### What I didn't find useful

- It does not discuss ESM or Sobek.

### What is out of date / what was wrong

- Not wrong for current runtime.

### What would need updating

- If ESM mode is added, document which primitives exist in ESM mode and how they are imported.
- Clarify that Node-style `node:` aliases are CommonJS resolver aliases today.

---

## Resource 24: `go-go-goja/cmd/xgoja/doc/02-user-guide.md`

### What I was researching

How xgoja user docs explain runtime profiles, require aliases, and generated command behavior.

### What I was looking for in this document in particular

- `require()` alias semantics.
- How module selection is documented.
- Whether users already see module syntax as CommonJS.

### Why I chose it

A module system change affects user-facing xgoja docs and scripts.

### How I found the resource itself

Search results for `require(` and xgoja docs surfaced this file.

### What I found useful in the document

- It explains `as` as the JavaScript `require()` alias.
- It documents runtime profile module selection.
- It includes troubleshooting around `require("fs")` missing when a module is aliased to `fs:assets`.

### What I didn't find useful

- It does not discuss ESM.

### What is out of date / what was wrong

- Not wrong today.

### What would need updating

- If an ESM mode is added, explain `import` specifier aliases separately from CommonJS `require()` aliases.
- Add examples for default imports if the ESM spike uses default-object native modules.

---

## Resource 25: `go-go-goja/cmd/xgoja/doc/04-tutorial-providing-package-and-modules.md`

### What I was researching

Provider package documentation and how providers expose modules to JavaScript.

### What I was looking for in this document in particular

- User-facing explanation of provider packages.
- The current recommended provider module API.
- CommonJS assumptions in provider docs.

### Why I chose it

Provider docs would need substantial updates if provider modules gain an ESM surface.

### How I found the resource itself

Search results for provider modules and `require()` in xgoja docs.

### What I found useful in the document

- It explains that provider packages are Go-side composition units compiled into generated binaries.
- It describes provider modules as things made available through `require(...)`.
- It reinforces that provider selection remains useful even if JavaScript module syntax changes.

### What I didn't find useful

- It does not discuss ESM or synthetic module records.

### What is out of date / what was wrong

- Not wrong for current CommonJS xgoja.

### What would need updating

- If ESM is added, include a provider API matrix: CommonJS loader, ESM default export, ESM named exports.

---

## Resource 26: `design/05-xgoja-codegen-and-script-execution-runthrough.md`

### What I was researching

The lifecycle document created earlier in the ticket, to ensure the Sobek/ESM analysis uses the same xgoja build/runtime mental model.

### What I was looking for in this document in particular

- Where `providerapi.Module.New` is called.
- Which machinery is module-system-specific versus runtime/provider lifecycle-specific.
- How GOJA-053's pre-runtime config timing fits into the larger flow.

### Why I chose it

The Sobek/ESM question is partly about reducing confusion in the lifecycle. The existing runthrough is the baseline.

### How I found the resource itself

It was created earlier in the same GOJA-053 ticket.

### What I found useful in the document

- It distinguishes provider registry from require registry.
- It identifies `app.RuntimeFactory` as a profile-to-engine-runtime adapter.
- It identifies `providerRuntimeModuleSpec.RegisterRuntimeModule` as the `Module.New` timing point.
- It helps explain which concepts ESM would not remove.

### What I didn't find useful

- It does not discuss Sobek or ESM.

### What is out of date / what was wrong

- It is current as of the prior step.
- It will need updating if GOJA-053 adds `NewRuntimeFromSections` or if ESM mode is introduced.

### What would need updating

- Add a short note that ESM would replace require registry behavior, not provider registry/runtime lifecycle behavior.

---

## Overall findings

### Most useful resources

- Sobek README: best high-level warning that ESM is experimental, undocumented, and requires an event loop.
- Sobek `modules.go`: best API reference for ModuleRecord/link/evaluate/dynamic import.
- Sobek `modules_integration_test.go`: best evidence that native/custom ESM ModuleRecords are feasible.
- go-go-goja `engine/factory.go`: best evidence that goja_nodejs replacement is a large migration.
- go-go-goja `modules/common.go`: best evidence that current native modules are CommonJS-shaped.
- Geppetto module/provider files: best evidence that complex providers still need pre-module setup/config regardless of module syntax.

### Most stale/confusing resources

- Sobek ESM APIs: comments explicitly say names/signatures may need revision.
- Sobek tests: useful, but currently function as documentation and contain hacks/FIXMEs.
- xgoja `ConfigSectionCapability` naming: confusing, though not specific to Sobek.
- `ModuleContext.Context`: still confusing and should be renamed or documented.

### Main update candidates

1. Add comments/docs distinguishing provider registry, require registry, and any future ESM resolver.
2. Add an xgoja “CommonJS vs ESM” note to docs if the Sobek spike is accepted.
3. Keep bundling-to-CJS docs prominent as the low-complexity path for modern JS authoring.
4. Document Sobek ESM as experimental and tests-as-docs if any spike package is added.
5. Do not rewrite provider APIs until a small native ESM ModuleRecord experiment has tests.
