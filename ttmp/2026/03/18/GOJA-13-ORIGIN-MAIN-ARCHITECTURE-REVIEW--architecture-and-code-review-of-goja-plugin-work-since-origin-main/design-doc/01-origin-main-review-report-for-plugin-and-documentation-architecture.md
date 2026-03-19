---
Title: Origin main review report for plugin and documentation architecture
Ticket: GOJA-13-ORIGIN-MAIN-ARCHITECTURE-REVIEW
Status: active
Topics:
    - goja
    - analysis
    - architecture
    - tooling
    - refactor
    - repl
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/bun-demo/main.go
      Note: |-
        Third copy of plugin runtime setup
        Third copy of runtime plugin bootstrap
    - Path: cmd/js-repl/main.go
      Note: |-
        Bobatea REPL bootstrap and second copy of plugin/docs wiring
        Bobatea REPL bootstrap and second copy of plugin and docs wiring
    - Path: cmd/repl/main.go
      Note: |-
        Line REPL bootstrap and one copy of plugin/docs wiring
        Line REPL bootstrap and one copy of plugin and docs wiring
    - Path: engine/factory.go
      Note: Runtime creation flow and new runtime-scoped registrar setup
    - Path: engine/runtime.go
      Note: Owned runtime lifecycle and missing long-lived runtime-scoped state access
    - Path: engine/runtime_modules.go
      Note: Runtime module registrar API and temporary value-sharing seam
    - Path: modules/glazehelp/glazehelp.go
      Note: Legacy JS help surface still shipped alongside the new docs system
    - Path: modules/glazehelp/registry.go
      Note: Global help-system registry with entrypoint-specific state
    - Path: pkg/docaccess/plugin/provider.go
      Note: String-encoded plugin doc entry IDs and parser helpers
    - Path: pkg/docaccess/runtime/registrar.go
      Note: Runtime-scoped docs hub plus JS adapter concentrated in one file
    - Path: pkg/hashiplugin/contract/jsmodule.proto
      Note: Current plugin contract shape and unused method metadata fields
    - Path: pkg/hashiplugin/host/client.go
      Note: |-
        Plugin process startup, diagnostics handling, and manifest retrieval
        Plugin process startup diagnostics and manifest retrieval
    - Path: pkg/hashiplugin/host/config.go
      Note: Discovery defaults and silent fallback behavior
    - Path: pkg/hashiplugin/host/reify.go
      Note: Runtime invocation bridge from JS to plugin subprocesses
    - Path: pkg/hashiplugin/host/report.go
      Note: User-facing plugin status summary and error visibility
    - Path: pkg/hashiplugin/host/validate.go
      Note: Host-side manifest validation rules
    - Path: pkg/hashiplugin/sdk/export.go
      Note: SDK export and method options compared with the richer protobuf schema
    - Path: pkg/hashiplugin/sdk/module.go
      Note: SDK-side manifest construction and duplicate validation logic
    - Path: pkg/repl/evaluators/javascript/evaluator.go
      Note: |-
        Help/completion path that still relies mainly on static signatures
        Help and completion path that still relies mainly on static signatures
ExternalSources: []
Summary: Evidence-based review of the plugin, SDK, docs-hub, and REPL changes since origin/main, focused on duplication, deprecated paths, runtime complexity, error handling, and cleanup priorities for a long-lived codebase.
LastUpdated: 2026-03-18T18:05:00-04:00
WhatFor: Help decide what to simplify, consolidate, or tighten after the initial plugin and docs feature work so the codebase stays maintainable as the branch lands and evolves.
WhenToUse: Use when planning post-merge cleanup, reviewing the branch as a whole, or deciding which architecture debts should be paid before building more plugin and docs features.
---


# Origin main review report for plugin and documentation architecture

## Executive Summary

From `origin/main` to `HEAD`, this branch adds four substantial capabilities:

- runtime-scoped module registrars in `engine`
- HashiCorp `go-plugin` support plus a plugin authoring SDK
- example plugins and plugin-aware REPL wiring
- a runtime-scoped documentation hub exposed through `require("docs")`

The feature work is real and useful. The branch is not random churn. There is a coherent direction:

- move runtime setup into explicit composition
- move plugins into a real host/contract/sdk split
- move docs into a runtime-scoped hub instead of one-off wrappers

That said, the review turned up several maintainability risks that should be addressed before the architecture calcifies:

1. runtime-scoped state exists only during setup, not on the returned runtime
2. runtime/bootstrap wiring is duplicated across three entrypoints
3. the old `glazehelp` path is still live, global, and inconsistent across entrypoints
4. plugin validation rules are duplicated across authoring and host ingest
5. parts of the new plugin/doc contract are stringly and only partially surfaced
6. plugin diagnostics and cancellation behavior are weaker than the feature surface suggests
7. the evaluator help path still bypasses the new docs system in practice

My overall recommendation is not a rewrite. It is a consolidation pass:

- keep the new architecture
- reduce duplicated bootstrap code
- remove or explicitly deprecate the old help surface
- centralize validation and naming rules
- make runtime-scoped state first-class on the owned runtime
- tighten error visibility and cancellation semantics

## Scope And Review Method

This review covers the committed delta `origin/main..HEAD`.

### Surface area reviewed

- `git log --oneline origin/main..HEAD`
- `git diff --stat origin/main..HEAD`
- `git diff --name-only origin/main..HEAD`
- detailed reads of the changed code in:
  - `engine`
  - `pkg/hashiplugin`
  - `pkg/docaccess`
  - `pkg/repl/evaluators/javascript`
  - `cmd/repl`
  - `cmd/js-repl`
  - `cmd/bun-demo`
  - `modules/glazehelp`

### What this report optimizes for

- long-lived maintainability
- clarity of ownership and naming
- avoiding duplicate or deprecated surfaces
- identifying runtime and lifecycle sharp edges early
- preserving the useful new architecture instead of flattening it away

### What this report is not

- not a line-by-line bug hunt for every new test fixture or help page
- not a recommendation to undo the plugin/docs work
- not a request to oversimplify the system into one file or one package

## Change Inventory

The branch adds 30+ commits and about 90 changed files. Most non-ticket churn sits in:

- `pkg/hashiplugin/*`
- `pkg/docaccess/*`
- `engine/*`
- `cmd/repl`, `cmd/js-repl`, `cmd/bun-demo`

The important architectural layers now look like this:

```text
engine/
  FactoryBuilder
  Runtime
  RuntimeModuleRegistrar

pkg/hashiplugin/
  contract/   protobuf + shared interface
  shared/     go-plugin transport glue
  host/       discovery, load, validate, reify, reports
  sdk/        plugin authoring DSL

pkg/docaccess/
  model + hub + providers
  runtime/    builds a docs hub per runtime and exposes require("docs")

cmd/
  repl        line REPL
  js-repl     Bobatea TUI REPL
  bun-demo    embedded bundle runtime consumer
```

That is the right high-level decomposition. The cleanup work is mostly about the seams between those layers.

## Current-State Architecture Map

```mermaid
flowchart LR
    A[cmd/repl | cmd/js-repl | cmd/bun-demo] --> B[engine.FactoryBuilder]
    B --> C[RuntimeModuleRegistrars]
    C --> D[hashiplugin host registrar]
    C --> E[docaccess runtime registrar]
    D --> F[plugin processes]
    D --> G[loaded manifests + report collector]
    E --> H[docaccess.Hub]
    H --> I[Glazed provider]
    H --> J[jsdoc provider]
    H --> K[plugin provider]
    B --> L[engine.Runtime]
    L --> M[goja VM + require registry]
    M --> N[repl/js-repl evaluation]
```

### Strong parts of the current shape

- `engine.FactoryBuilder` gives runtime composition a real center.
- `pkg/hashiplugin` has a sensible host/shared/sdk split.
- `pkg/docaccess` is better than continuing to grow `modules/glazehelp`.
- the example plugins are useful and give the feature a concrete teaching surface.

### Weak parts of the current shape

- setup-time state and runtime-time state are not the same thing yet
- entrypoint bootstrap code is repeating and starting to drift
- the legacy and replacement doc surfaces both still exist
- some contracts are richer than the actual API that populates them

## Findings

## 1. Runtime-scoped state stops at setup time instead of becoming owned runtime state

Severity: medium

### Problem

The branch introduces `RuntimeModuleContext.Values` as a registrar-to-registrar coordination seam, but those values do not survive onto the returned `engine.Runtime`. That means runtime-scoped metadata is available only while the factory is building the runtime, not to later consumers of the owned runtime.

### Where to look

- `engine/factory.go:183-195`
- `engine/runtime.go:23-34`
- `engine/runtime_modules.go:19-44`
- `pkg/hashiplugin/host/registrar.go:64-74`
- `pkg/docaccess/runtime/registrar.go:78-125`

### Example

```go
moduleCtx := &RuntimeModuleContext{
    VM:        vm,
    Loop:      loop,
    Owner:     owner,
    AddCloser: rt.AddCloser,
    Values:    map[string]any{},
}
for _, registrar := range f.runtimeModuleRegistrars {
    if err := registrar.RegisterRuntimeModules(moduleCtx, reg); err != nil {
        ...
    }
}
```

### Why it matters

- The branch already needs runtime-scoped plugin manifest snapshots to build the docs hub.
- The next feature immediately wants evaluator-side access to the same docs hub.
- Without a first-class runtime value store, every later feature will need another workaround:
  - closure capture
  - repeated re-indexing
  - ad hoc package globals

This is exactly the sort of “temporary seam that becomes permanent debt” problem that grows quietly.

### Cleanup sketch

Make runtime-scoped setup values a first-class part of the owned runtime.

```go
type Runtime struct {
    VM      *goja.Runtime
    Require *require.RequireModule
    Loop    *eventloop.EventLoop
    Owner   runtimeowner.Runner
    Values  map[string]any
}

func (r *Runtime) Value(key string) (any, bool) { ... }
```

Then in `Factory.NewRuntime`:

```go
rt.Values = cloneValues(moduleCtx.Values)
```

This keeps the registrar pattern but removes the “setup-only” limitation.

## 2. Runtime/bootstrap wiring is duplicated across `repl`, `js-repl`, and `bun-demo`

Severity: medium

### Problem

Plugin discovery, allowlist wiring, and docs/help source setup are now repeated in three entrypoints. The branch also duplicates:

- `stringSliceFlag`
- `pluginStartupSummary`
- `printPluginReport`

### Where to look

- `cmd/repl/main.go:43-67`, `cmd/repl/main.go:168-203`
- `cmd/js-repl/main.go:53-85`, `cmd/js-repl/main.go:140-150`
- `cmd/bun-demo/main.go:47-64`

### Example

```go
type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
    return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
    *s = append(*s, value)
    return nil
}
```

This helper now exists separately in `cmd/js-repl` and `cmd/bun-demo`, while similar plugin setup logic exists in all three entrypoints.

### Why it matters

- Every new runtime-wide feature now costs three changes.
- Drift is already visible:
  - `repl` prints a different plugin startup summary than `js-repl`
  - `repl` also registers the old `glazehelp` path while `js-repl` does not
- Small drift becomes behavior drift very quickly in CLI apps.

### Cleanup sketch

Create a small shared bootstrap package, not a giant framework.

Suggested split:

```text
pkg/app/runtimecfg/
  plugin_flags.go
  plugin_bootstrap.go
  help_sources.go
  report.go
```

Then each entrypoint becomes:

```go
opts := runtimecfg.ParsePluginOptions(...)
builder := engine.NewBuilder().WithModules(engine.DefaultRegistryModules())
builder = runtimecfg.ApplyPluginRuntime(builder, opts)
builder = runtimecfg.ApplyDocsRuntime(builder, runtimecfg.DefaultHelpSources(...))
```

That keeps each command small while preserving command-specific UI behavior.

## 3. The old `glazehelp` path is still live, global, and inconsistent with the new docs architecture

Severity: medium

### Problem

The branch introduces `pkg/docaccess` and `require("docs")` as the new docs architecture, but the old `modules/glazehelp` module is still blank-imported into the default runtime and still relies on a package-global registry.

Worse, only `cmd/repl` registers a help system into that registry. `cmd/js-repl` does not. So the same runtime default module exists across entrypoints, but its behavior depends on entrypoint-specific setup.

### Where to look

- `engine/runtime.go:14-20`
- `modules/glazehelp/glazehelp.go:14-124`
- `modules/glazehelp/registry.go:10-63`
- `cmd/repl/main.go:175-185`
- `cmd/js-repl/main.go:72-85`

### Example

```go
// engine/runtime.go
_ "github.com/go-go-golems/go-go-goja/modules/glazehelp"

// cmd/repl/main.go
glazehelp.Register("default", appHelpSystem)
```

There is no equivalent `glazehelp.Register(...)` in `cmd/js-repl`.

### Why it matters

- The system now has two documentation surfaces:
  - old: `glazehelp`
  - new: `docs`
- One is runtime-scoped and provider-based.
- The other is global, mutable, and entrypoint-dependent.

This is a classic “deprecated but not actually deprecated” situation. It confuses future contributors and makes docs behavior harder to reason about.

### Cleanup sketch

Pick one explicit policy:

1. Deprecate and remove `glazehelp` from default runtime composition.
2. Keep it only as a thin compatibility module implemented on top of `docaccess`.

If compatibility is needed:

```go
require("glazehelp").section(key, slug)
```

should become a thin adapter over:

```go
require("docs").bySlug(sourceID, slug)
```

But the current mixed global/runtime model should not persist.

## 4. Plugin validation rules are duplicated across the SDK and host layers

Severity: medium

### Problem

The SDK validates module/export/method shape when building a plugin, and the host validates almost the same rules again when loading the plugin. The checks are not identical, and there is no shared contract-level validator.

### Where to look

- `pkg/hashiplugin/sdk/module.go:107-169`
- `pkg/hashiplugin/host/validate.go:10-75`

### Example

SDK side:

```go
if !strings.HasPrefix(def.name, DefaultNamespace) {
    return fmt.Errorf("sdk module %q must use namespace %q", def.name, DefaultNamespace)
}
...
if _, ok := methodNames[method.name]; ok {
    return fmt.Errorf("sdk object export %q in module %q has duplicate method %q", ...)
}
```

Host side:

```go
if !strings.HasPrefix(name, cfg.Namespace) {
    return fmt.Errorf("plugin module %q must use namespace %q", name, cfg.Namespace)
}
...
if _, ok := methodNames[methodName]; ok {
    return fmt.Errorf("object export %q in module %q has duplicate method %q", ...)
}
```

### Why it matters

- Drift is now a maintenance risk, not a hypothetical one.
- The SDK and the host can develop subtly different ideas of “valid plugin.”
- Future rules, like symbol naming restrictions or richer method metadata validation, will have to be updated in two places.

### Cleanup sketch

Create one contract-level validator package or function and let both layers call it.

```go
package contractvalidate

type Options struct {
    Namespace    string
    AllowModules []string
}

func ValidateManifest(manifest *contract.ModuleManifest, opts Options) error
```

Then:

- SDK validates the built manifest through that path
- host validates loaded manifests through the same path

Keep SDK-only checks only for pre-manifest concerns like nil handlers.

## 5. The plugin/doc contract is richer than the public authoring API, and some IDs are more stringly than the validation rules allow

Severity: medium

### Problem

The protobuf contract now includes richer method metadata:

- `summary`
- `doc`
- `tags`

and the SDK now exposes `MethodSummary(...)`, `MethodDoc(...)`, and `MethodTags(...)`. The remaining concern is that plugin doc entries are encoded into string IDs using `/` and `.` separators, but export and method names are not validated against those separators.

### Where to look

- `pkg/hashiplugin/contract/jsmodule.proto:29-40`
- `pkg/hashiplugin/sdk/export.go:37-120`
- `pkg/hashiplugin/sdk/module.go:118-165`
- `pkg/docaccess/plugin/provider.go:249-271`

### Example

Contract:

```proto
message MethodSpec {
  string name = 1;
  string summary = 2;
  string doc = 3;
  repeated string tags = 4;
}
```

SDK manifest construction:

```go
methods = append(methods, &contract.MethodSpec{
    Name: method.name,
    Doc:  method.doc,
})
```

Doc ID encoding:

```go
func exportID(moduleName, exportName string) string {
    return moduleName + "/" + exportName
}

func methodID(moduleName, exportName, methodName string) string {
    return exportID(moduleName, exportName) + "." + methodName
}
```

### Why it matters

- `summary` and `tags` are dead richness today.
- The schema suggests capabilities the public API does not actually provide.
- The string encoding for IDs assumes export and method names do not contain separators, but validation never enforces that assumption.

That is exactly how “mostly works” protocols turn brittle later.

### Cleanup sketch

Do one of these, explicitly:

1. Simplify the contract to match the current public API.
2. Finish the SDK surface to match the contract.

And separately, enforce name constraints:

```go
var validSymbol = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
```

or, if more permissive names are desired, stop using separator-encoded IDs and use structured refs internally.

## 6. Plugin diagnostics are weaker than the feature surface suggests

Severity: medium

### Problem

The system loads external subprocesses, but the default diagnostics path hides several failure details:

- plugin stdout/stderr are discarded
- default discovery silently returns no directories on stat/walk errors
- startup summaries ignore `LoadReport.Error`

### Where to look

- `pkg/hashiplugin/host/client.go:72-83`
- `pkg/hashiplugin/host/config.go:73-104`
- `pkg/hashiplugin/host/report.go:99-113`

### Example

```go
client := plugin.NewClient(&plugin.ClientConfig{
    ...
    SyncStdout: io.Discard,
    SyncStderr: io.Discard,
    Stderr:     io.Discard,
})
```

and:

```go
func (r LoadReport) Summary() string {
    switch {
    case len(r.Loaded) > 0:
        ...
    case len(r.Candidates) > 0:
        ...
    case len(r.Directories) > 0:
        ...
    default:
        return "no plugin directories configured"
    }
}
```

### Why it matters

- Plugin startup/debug failures become much harder to diagnose.
- A permission problem under `~/.go-go-goja/plugins` can degrade into “no plugins found.”
- The startup summary can hide the fact that an actual load error occurred.

This is especially risky for a system that spawns external binaries and expects users to author their own plugins.

### Cleanup sketch

- Keep the silent path for normal mode if desired.
- Add a debug/reporting mode that captures stderr in memory and exposes it.
- Make discovery distinguish:
  - no configured dirs
  - missing dirs
  - unreadable dirs
  - load errors

For summaries:

```go
if r.Error != "" {
    return "plugin loading error: " + r.Error
}
```

should probably win over generic “found candidates” messaging.

## 7. Plugin invocation is not tied to runtime shutdown or caller cancellation

Severity: medium

### Problem

JS calls into plugin exports use `context.Background()` at the reification boundary. That means invocation lifetime is controlled only by the `LoadedModule` timeout fallback, not by runtime shutdown or any evaluation-scoped context.

### Where to look

- `pkg/hashiplugin/host/reify.go:55-67`
- `pkg/hashiplugin/host/client.go:38-50`

### Example

```go
resp, err := loaded.Invoke(context.Background(), &contract.InvokeRequest{
    ExportName: exportName,
    MethodName: methodName,
    Args:       args,
})
```

### Why it matters

- Runtime shutdown and plugin call lifetime are only loosely coupled.
- Long-running or hung plugin calls cannot be cancelled early from a higher-level runtime context.
- This becomes more important as plugins move from demos to real IO or background-process integrations.

### Cleanup sketch

At minimum, thread a runtime-owned base context through reified invocations.

```go
type LoadedModule struct {
    BaseContext context.Context
    ...
}
```

or attach a cancelable runtime context to `engine.Runtime` and use it in reified calls.

Even if there is still a default timeout, the runtime should be able to say “all plugin calls stop now.”

## 8. The new docs system is architecturally present but still underused in the evaluator/help path

Severity: low-medium

### Problem

The branch lands `pkg/docaccess` and `require("docs")`, but `js-repl` help remains primarily driven by:

- static signature strings
- completion candidate detail text
- shallow runtime inspection

The new docs hub is available in the runtime but not yet used by the evaluator help path.

### Where to look

- `pkg/docaccess/runtime/registrar.go:56-189`
- `pkg/repl/evaluators/javascript/evaluator.go:478-939`

### Example

```go
if txt, ok := helpBarSymbolSignatures[token]; ok {
    return makeHelpBarPayload(txt, "signature")
}
```

### Why it matters

- The branch now has two “documentation systems” from a user perspective:
  - explicit `require("docs")`
  - implicit static REPL help
- That slows convergence toward one coherent developer experience.

This is not a reason to block the branch, but it is a good reason to avoid building more static help tables.

### Cleanup sketch

Use GOJA-12 as the convergence ticket:

- preserve static fallback for built-ins
- let plugin/module docs come from `docaccess`
- keep evaluator-side docs resolution Go-side, not JS-recursive

## Low-Risk Refactors

These are the cleanup items I would do first because they reduce future complexity without changing the core feature model.

- Introduce a shared runtime/bootstrap helper package for `repl`, `js-repl`, and `bun-demo`.
- Add `Runtime.Values` plus a `Value()` accessor on the owned runtime.
- Make `LoadReport.Summary()` error-aware.
- Stop silently swallowing discovery directory errors.
- Decide and document the `glazehelp` deprecation policy.
- Centralize manifest validation rules.

## Larger Architectural Changes

These are still worth doing, but they are one notch larger and should probably happen after the first consolidation pass.

- Replace separator-encoded plugin doc IDs with structured refs or stricter symbol validation.
- Finish or simplify the `MethodSpec` richness story.
- Thread runtime cancellation into plugin invocation.
- Break `pkg/docaccess/runtime/registrar.go` into:
  - provider assembly
  - runtime state storage
  - JS module adapter

## Recommended Sequence

1. Consolidate bootstrap and runtime value plumbing.
2. Choose the `glazehelp` deprecation/compatibility strategy.
3. Centralize plugin validation and naming rules.
4. Strengthen diagnostics and cancellation behavior.
5. Then build the next docs-aware REPL UX on top of the cleaner runtime seams.

## Open Questions

- Should `engine.Runtime` become the canonical home for all runtime-scoped setup values?
- Is `glazehelp` meant to survive as a compatibility layer, or should it be removed from default runtime composition?
- Are plugin export/method names intentionally unrestricted, or should the system define a stricter public naming contract now?
- Should the plugin contract really keep `summary` and `tags` before the SDK exposes them?

## References

- `git log --oneline origin/main..HEAD`
- `git diff --stat origin/main..HEAD`
- `ttmp/.../scripts/list-origin-main-review-surface.sh`
