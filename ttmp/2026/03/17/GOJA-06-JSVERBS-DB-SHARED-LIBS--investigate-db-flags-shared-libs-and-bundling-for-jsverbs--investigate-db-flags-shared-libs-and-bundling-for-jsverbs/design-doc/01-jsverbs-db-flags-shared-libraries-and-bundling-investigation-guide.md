---
Title: jsverbs db flags, shared libraries, and bundling investigation guide
Ticket: GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs
Status: active
Topics:
    - go
    - glazed
    - js-bindings
    - sqlite
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/jsverbs-example/main.go
      Note: Runner used for all experiments
    - Path: modules/database/database.go
      Note: Database module configuration contract
    - Path: pkg/jsverbs/binding.go
      Note: Binding plan and file-local section validation
    - Path: pkg/jsverbs/runtime.go
      Note: Runtime construction, source overlay loading, and invoke flow
    - Path: pkg/jsverbs/scan.go
      Note: Scanner entrypoints, top-level extractor rules, and function/metadata finalization
    - Path: pkg/jsverbs/command.go
      Note: Glazed command description generation from verb metadata
    - Path: ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts/exp03-bundled/dist/bundle.cjs
      Note: Generated CommonJS bundle proving the working bundle shape
    - Path: ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts
      Note: Ticket-local experiments for db flags, shared helpers, and bundling
ExternalSources: []
Summary: Evidence-backed intern guide explaining how jsverbs works today, what is already possible for db-backed verbs and shared require()-loaded helpers, where cross-file shared flag sections fail, and what API additions would make the system more ergonomic.
LastUpdated: 2026-03-17T13:42:45.158508858-04:00
WhatFor: Explain the current jsverbs architecture, answer whether db flags and shared libraries are possible today, document current limitations around shared sections, and recommend a concrete evolution path.
WhenToUse: Use when integrating jsverbs into a real CLI, teaching a new engineer how the system works, or planning runner-level support for shared sections or database bootstrap behavior.
---


# jsverbs db flags, shared libraries, and bundling investigation guide

## Executive Summary

`pkg/jsverbs` already supports the important half of the desired workflow. A jsverb can expose a `--db` flag today, bind that flag into a JavaScript object, and call the native `database` module from inside the verb. Shared JavaScript runtime logic is also already possible today through `require()` as long as the helper files are inside the scanned module tree or bundled into the final CommonJS artifact.

The part that does not work today is cross-file sharing of `__section__` metadata. Sections are collected per file during scanning, and the binding-plan validation step only looks up referenced sections in `verb.File.Sections`. That means a verb in `verbs.js` cannot reference a section declared in `common.js`, even if both files are scanned into the same registry.

Bundling is viable, but there is one extra rule that is easy to miss: the bundled output must still contain a top-level function declaration or a top-level variable-assigned function that matches the `__verb__` function name. My first bundle attempt failed because esbuild tree-shook the command function away. Exporting the function fixed that and produced a working bundle.

For an intern, the right mental model is this:

1. `jsverbs` is a static scanner plus a runtime bridge.
2. Metadata discovery is strict, file-local, and declarative.
3. Runtime code sharing is broader than metadata sharing because `require()` can load any scanned helper file.
4. If we want runner-level shared flag catalogs or host-managed database bootstrapping, we need a small API expansion on the Go side rather than more clever JavaScript.

## Problem Statement And Scope

The user asked two concrete questions:

1. Can jsverbs expose a `--db ...` flag that opens a database and lets the JavaScript implementation use it?
2. Can jsverbs load shared libraries with `require()` so functionality and possibly common flag sections can be shared across verbs?

Those questions sound similar, but they touch different layers of the system:

- CLI schema generation is controlled by static metadata scanning.
- Argument-to-JS binding is controlled by the binding-plan layer.
- Actual code reuse is controlled by the runtime loader and `require()`.
- Database behavior is controlled by the native `database` module plus whatever bootstrapping policy the verb or host applies.

This report focuses on those four layers, answers what is possible right now, and recommends the smallest changes that would make the system more production-friendly.

Out of scope:

- implementing new production APIs in `pkg/jsverbs`,
- redesigning the `database` native module,
- replacing the current promise polling bridge.

## Current System Overview

The current code path is a five-stage pipeline:

```text
JS files or in-memory sources
  -> scanner builds FileSpec / VerbSpec / SectionSpec
  -> binding planner decides how Glazed values map into JS parameters
  -> command compiler creates Glazed command descriptions
  -> runtime bridge constructs a Goja runtime and loads source via require()
  -> selected JS function executes and returns rows or text
```

More concretely:

1. `ScanDir`, `ScanFS`, `ScanSource`, and `ScanSources` gather JS source inputs and normalize them into module paths in [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go#L17).
2. The extractor walks top-level AST nodes and records functions, `__package__`, `__section__`, `__verb__`, and `doc` blocks in [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go#L356).
3. `finalizeVerb` resolves each verb metadata record against a real function name in that same file in [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go#L289).
4. `buildVerbBindingPlan` computes the schema/runtime binding contract once and validates that every referenced section exists in the file-local section map in [pkg/jsverbs/binding.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/binding.go#L40).
5. `buildDescription` turns the binding plan into Glazed sections and fields in [pkg/jsverbs/command.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/command.go#L72).
6. `invoke` creates a runtime with default native modules, injects an overlay loader, requires the verb file, and then calls the captured JS function in [pkg/jsverbs/runtime.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/runtime.go#L18).

That split is why some reuse patterns work and others do not. Runtime module loading sees the whole scanned tree. Metadata validation does not.

## Architecture Diagram

```text
                Scan Time                                          Run Time

  verbs.js / helper.js / bundle.cjs                  parsed Glazed values from CLI flags
              |                                                       |
              v                                                       v
  tree-sitter top-level extraction                            buildArguments(...)
              |                                                       |
              v                                                       v
  FileSpec { functions, sections, verb meta }             JS args / section maps / context
              |                                                       |
              v                                                       v
  finalizeVerb(function must exist in file)      engine.NewBuilder + DefaultRegistryModules()
              |                                                       |
              v                                                       v
  buildVerbBindingPlan(validate sections)      require.WithLoader(sourceLoader) + overlay prelude
              |                                                       |
              v                                                       v
  buildDescription -> Cobra/Glazed flags            require(verb module) -> captured JS fn -> result
```

## API Reference For The Current System

### Scan-time entrypoints

- `ScanDir(root string, opts ...ScanOptions)` reads a real directory tree and records each `.js` or `.cjs` file as a `sourceInput` in [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go#L17-L74).
- `ScanFS(...)` does the same for an arbitrary `fs.FS`, which matters for embedded or virtual fixtures in [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go#L76-L124).
- `ScanSource(...)` and `ScanSources(...)` allow synthetic or already-bundled source strings to be scanned without a disk directory in [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go#L126-L150).

### Core model types

- `Registry` is the top-level container for scanned files, diagnostics, and compiled verbs in [pkg/jsverbs/model.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/model.go#L74-L82).
- `FileSpec` stores source bytes, package metadata, file-local sections, explicit verb metadata, and discovered functions in [pkg/jsverbs/model.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/model.go#L84-L96).
- `SectionSpec` and `FieldSpec` describe reusable CLI schema fragments and field metadata in [pkg/jsverbs/model.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/model.go#L118-L136).
- `VerbSpec` describes the command-facing view of a JS function, including `UseSections`, `Fields`, and resolved `Params` in [pkg/jsverbs/model.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/model.go#L138-L150).

### Binding modes

Observed binding modes in the code:

- positional binding: a field value becomes a direct JS argument,
- section binding: a whole section map becomes one JS object,
- `bind: "all"`: every resolved Glazed value becomes one object,
- `bind: "context"`: runtime metadata such as `rootDir`, `sourceFile`, and `sections` becomes one object.

These modes are encoded in [pkg/jsverbs/binding.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/binding.go#L11-L38) and materialized into runtime arguments in [pkg/jsverbs/runtime.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/runtime.go#L89-L143).

### Runtime API and modules

- `engine.DefaultRegistryModules()` enables all native modules registered in the default module registry in [engine/module_specs.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/engine/module_specs.go#L64-L82).
- Blank imports in [engine/runtime.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/engine/runtime.go#L12-L19) ensure built-in modules such as `database`, `exec`, `fs`, and `glazehelp` are registered at process startup.
- The database module itself requires explicit `configure(driver, dsn)` before `query(...)` or `exec(...)` in [modules/database/database.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/modules/database/database.go#L49-L80) and [modules/database/database.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/modules/database/database.go#L139-L180).
- `engine.WithModuleRootsFromScript(...)` exists for script-path-based module-root layering in [engine/module_roots.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/engine/module_roots.go#L11-L118), but `pkg/jsverbs/runtime.go` currently uses its own `sourceLoader` instead of that helper.

## How The Example Runner Fits

The example CLI in [cmd/jsverbs-example/main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/cmd/jsverbs-example/main.go#L20-L88) is intentionally thin. It:

1. discovers the `--dir` scan root,
2. scans that directory once,
3. converts the registry into Glazed commands,
4. adds those commands to a Cobra root,
5. leaves runtime configuration entirely to `pkg/jsverbs`.

That is important because the example runner does not add any runner-specific policy for database setup, shared sections, or bundle loading. All of the behavior observed in the experiments therefore comes from `pkg/jsverbs` plus the generic engine/module layer, not from ad hoc logic in the example command.

## Evidence From The Ticket Experiments

### Experiment 1: Unbundled `--db` flag plus shared helper works today

Files:

- [scripts/exp01-unbundled-db/verbs.js](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts/exp01-unbundled-db/verbs.js)
- [scripts/exp01-unbundled-db/lib/sql.js](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts/exp01-unbundled-db/lib/sql.js)
- [scripts/run-exp01.sh](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts/run-exp01.sh)

Command run:

```bash
/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts/run-exp01.sh
```

Observed result:

- the command accepted `--db <sqlite-file>`,
- the JS implementation called `require("database")`,
- the JS implementation also loaded a sibling helper file with `require("./lib/sql")`,
- the command returned rows seeded through the database module.

Conclusion:

- exposing a `--db` flag is already possible today,
- the simplest current design is to treat database configuration as ordinary jsverb metadata,
- the actual database open still happens in JS by calling `database.configure(...)`.

### Experiment 2: Cross-file shared sections do not work today

Files:

- [scripts/exp02-cross-file-sections/common.js](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts/exp02-cross-file-sections/common.js)
- [scripts/exp02-cross-file-sections/verbs.js](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts/exp02-cross-file-sections/verbs.js)
- [scripts/run-exp02.sh](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts/run-exp02.sh)

Command run:

```bash
/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts/run-exp02.sh
```

Observed failure:

```text
verbs.js#probe references unknown section "db"
exit status 1
```

This matches the code exactly. `buildVerbBindingPlan` validates referenced section slugs against `verb.File.Sections`, not against a registry-global section catalog, in [pkg/jsverbs/binding.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/binding.go#L117-L129).

Conclusion:

- runtime helper sharing is broader than metadata sharing,
- `require("./common")` can share executable JS code,
- `__section__` is file-local for command-schema purposes.

### Experiment 3: Bundling works if scanner-visible functions survive the bundle

Files:

- [scripts/exp03-bundled/src/index.js](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts/exp03-bundled/src/index.js)
- [scripts/exp03-bundled/src/shared.js](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts/exp03-bundled/src/shared.js)
- [scripts/exp03-bundled/dist/bundle.cjs](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts/exp03-bundled/dist/bundle.cjs)
- [scripts/run-exp03.sh](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts/run-exp03.sh)

Bundling command:

```bash
bun x esbuild src/index.js --bundle --platform=node --format=cjs --external:database --outfile=dist/bundle.cjs
```

Observed intermediate failure on the first attempt:

```text
bundle.cjs references unknown function "countUsers"
exit status 1
```

Reason:

- the bundle kept `__verb__("countUsers", ...)`,
- but the command function was tree-shaken away because it was not exported or otherwise kept alive,
- `finalizeVerb` requires a scanner-discovered function of the same name in [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go#L289-L329).

After adding `exports.countUsers = countUsers;`, the bundle contained a real top-level `function countUsers(...)` in [scripts/exp03-bundled/dist/bundle.cjs](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts/exp03-bundled/dist/bundle.cjs#L38-L55), and the bundled jsverb ran successfully.

Conclusion:

- bundling is a good packaging model for jsverbs,
- but the bundled output must preserve scanner-visible metadata and function definitions,
- exporting the command functions is a practical rule that keeps esbuild from removing them.

## Direct Answers To The User Questions

### A) Can jsverbs expose a `--db` flag and open a database for JS use?

Yes, with one important precision.

What works today:

- define a file-local section or field for `db`,
- bind that section into a JS parameter with `bind: "db"`,
- call `require("database").configure(...)` in the JS function,
- then call `query(...)` or `exec(...)`.

What does not exist today:

- a host-side automatic database bootstrap hook that reads `--db` and preconfigures the module before the JS code runs,
- a first-class runner option in `pkg/jsverbs` for saying "all verbs in this registry get a shared database flag and an already-open database handle."

Why it does not exist today:

- the `database` module requires explicit configuration before use in [modules/database/database.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/modules/database/database.go#L49-L80),
- `Registry.invoke` only builds JS arguments from parsed Glazed values and does not apply any host-side runtime initializer based on those values in [pkg/jsverbs/runtime.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/runtime.go#L18-L43).

### B) Can jsverbs load shared libs with `require()` and share common sections for flags?

Split answer:

- shared executable JS libraries: yes,
- shared metadata sections across files: no, not in the current design.

Why helpers work:

- every scanned file is added to `filesByModule`,
- `sourceLoader` resolves module paths from that map,
- the runtime uses `require.WithLoader(r.sourceLoader)` so sibling helper files load correctly in [pkg/jsverbs/runtime.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/runtime.go#L18-L25) and [pkg/jsverbs/runtime.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/runtime.go#L146-L152).

Why cross-file sections fail:

- sections live on `FileSpec.Sections`,
- binding validation only reads `verb.File.Sections`,
- there is no registry-level `SharedSections` map today.

## Recommended Design Direction

I recommend a staged design rather than one large rewrite.

### Phase 0: Adopt a clear “what works now” policy

For immediate production use:

1. Keep each verb's shared sections in the same source file as the verbs that use them.
2. Put shared runtime logic in sidecar helper modules loaded with `require()`.
3. If you want a deployable single file, bundle each entrypoint to CommonJS and ensure verb functions are explicitly exported so the scanner can still see them.
4. Treat database setup as ordinary verb logic for now.

This policy requires no Go changes.

### Phase 1: Add host-provided shared sections

This is the smallest Go-side API change that unlocks real cross-file flag reuse.

Suggested sketch:

```go
type HostSectionCatalog map[string]*jsverbs.SectionSpec

type RegistryOptions struct {
    SharedSections HostSectionCatalog
}

func (r *Registry) WithSharedSections(sections HostSectionCatalog) *Registry
```

Binding-plan lookup would change from:

```go
section := verb.File.Sections[slug]
```

to:

```go
section := verb.File.Sections[slug]
if section == nil {
    section = r.SharedSections[slug]
}
if section == nil {
    return error("unknown section")
}
```

This keeps JS metadata strict while letting the host inject reusable schemas such as `db`, `profile`, `auth`, or `output`.

### Phase 2: Add runtime initializers or invocation hooks

If the product goal is "the runner owns database bootstrapping, not every verb," then `pkg/jsverbs` needs a host hook before function invocation.

Suggested sketch:

```go
type InvokeHook interface {
    BeforeInvoke(ctx context.Context, rt *engine.Runtime, verb *VerbSpec, values *values.Values) error
}

type RuntimeOptions struct {
    Hooks []InvokeHook
}
```

A database hook could:

1. read the shared `db` section from `values`,
2. obtain the runtime's `database` module,
3. call `configure("sqlite3", dsn)` before the JS function runs.

That would let JS verbs write:

```js
const db = require("database");
function queryUsers(prefix) {
  return db.query("SELECT ...", prefix + "%");
}
```

instead of manually calling `configure(...)` every time.

### Phase 3: Make bundle production a documented contract

The bundle experiment suggests one explicit rule should become part of the jsverbs contract:

> Bundled artifacts must preserve scanner-visible command functions and static metadata calls.

Recommended documentation points:

- bundle format should be CommonJS,
- host/native modules like `database` stay external,
- command functions should be explicitly exported to avoid tree-shaking,
- metadata calls such as `__verb__` and `__section__` must survive bundling as static top-level calls.

The existing bundling playbook in [pkg/doc/bun-goja-bundling-playbook.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/doc/bun-goja-bundling-playbook.md#L21-L58) is already close to this; jsverbs-specific guidance should simply add the scanner-preservation rules.

## Recommended End-State API

This is the architecture I would recommend if the repo wants a first-class reusable runner package instead of only example code:

```go
type RunnerConfig struct {
    ScanOptions     jsverbs.ScanOptions
    SharedSections  map[string]*jsverbs.SectionSpec
    InvokeHooks     []jsverbs.InvokeHook
    ModuleSpecs     []engine.ModuleSpec
}

func LoadRegistry(root string, cfg RunnerConfig) (*jsverbs.Registry, error)
func AddCommands(rootCmd *cobra.Command, registry *jsverbs.Registry, cfg RunnerConfig) error
```

Runtime behavior would then look like:

```text
runner boot
  -> scan sources
  -> merge host shared sections
  -> compile commands
  -> on each invocation:
       create runtime
       register default + custom modules
       run invoke hooks (db, auth, profiles, tracing)
       require verb module
       call JS function
```

This keeps the scanner strict and predictable, while giving real CLIs one obvious place to add policy.

## Risks And Tradeoffs

### Keeping everything file-local

Pros:

- very simple model,
- easy to reason about,
- no host/JS schema split.

Cons:

- duplicated `db`/`auth`/`profile` sections across many files,
- harder to keep large verb sets consistent.

### Adding shared section catalogs

Pros:

- removes duplication,
- lets Go runners own cross-cutting flags,
- improves consistency across large command trees.

Cons:

- introduces one more place where schema can come from,
- must define precedence rules when file-local and host-provided sections share a slug.

### Host-side database bootstrapping

Pros:

- fewer repetitive `database.configure(...)` calls in JS,
- more uniform operational behavior.

Cons:

- hides some behavior from the JS author,
- requires explicit lifecycle rules for reconnect, close, and error propagation.

## Open Questions

1. Should shared section precedence favor the JS file or the host runner when both define the same slug?
2. Should the database hook expose only a DSN string, or a richer config object with driver, read-only mode, and pooling options?
3. Should jsverbs eventually gain a dedicated bundle loader mode, or is `ScanSource`/`ScanSources` sufficient as the public API for bundled artifacts?
4. Should the scanner become aware of exported functions as a stronger convention for bundling, or is documentation enough?

## Recommended Implementation Plan

1. Document the current rules clearly in `pkg/doc`:
   - file-local sections only,
   - helpers may live in separate required files,
   - bundles must preserve command functions.
2. Introduce registry-level shared sections and update binding-plan validation.
3. Add runtime invoke hooks so database bootstrapping can be host-managed.
4. Create a small production-grade runner package that wraps `ScanDir`, `Registry.Commands()`, and hook registration.
5. Add tests for:
   - host shared sections,
   - db hook preconfiguration,
   - bundled artifacts with exported command functions,
   - bundle failure when command functions are tree-shaken.

## References

- [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go)
- [pkg/jsverbs/model.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/model.go)
- [pkg/jsverbs/binding.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/binding.go)
- [pkg/jsverbs/command.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/command.go)
- [pkg/jsverbs/runtime.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/runtime.go)
- [modules/database/database.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/modules/database/database.go)
- [engine/module_specs.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/engine/module_specs.go)
- [engine/module_roots.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/engine/module_roots.go)
- [cmd/jsverbs-example/main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/cmd/jsverbs-example/main.go)
- [pkg/doc/09-jsverbs-example-fixture-format.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/doc/09-jsverbs-example-fixture-format.md)
- [pkg/doc/bun-goja-bundling-playbook.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/doc/bun-goja-bundling-playbook.md)
- [Obsidian project note](/home/manuel/code/wesen/obsidian-vault/Projects/2026/03/16/PROJ%20-%20go-go-goja%20jsverbs%20-%20JavaScript%20to%20Glazed%20Commands.md)

<!-- Describe the proposed solution in detail -->

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
