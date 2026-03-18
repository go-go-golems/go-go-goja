---
Title: Doc-aware js-repl autocomplete and help architecture and implementation guide
Ticket: GOJA-12-JS-REPL-DOC-AWARE-HELP
Status: active
Topics:
    - goja
    - repl
    - ui
    - architecture
    - tooling
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/js-repl/main.go
      Note: Current Bobatea entrypoint and user-facing help/autocomplete configuration
    - Path: engine/runtime.go
      Note: Owned runtime object that should expose stable runtime-scoped docs state after setup
    - Path: engine/runtime_modules.go
      Note: Runtime module context value-sharing seam used by module registrars during setup
    - Path: pkg/docaccess/plugin/provider.go
      Note: Existing plugin module/export/method docs provider and first implementation target
    - Path: pkg/docaccess/runtime/registrar.go
      Note: Existing runtime-scoped docs hub construction and docs-module registration
    - Path: pkg/jsparse/repl_completion.go
      Note: Alias-aware completion support that docs-aware help should enrich rather than replace
    - Path: pkg/repl/evaluators/javascript/evaluator.go
      Note: Current completion and contextual-help logic that GOJA-12 will enrich
ExternalSources: []
Summary: Design for making js-repl autocomplete, one-line help, and drawer help consume the same runtime-scoped documentation hub that already powers require("docs"), starting with plugin docs and leaving a clean path for broader module documentation later.
LastUpdated: 2026-03-18T16:57:00-04:00
WhatFor: Explain how to extend js-repl so completion rows, contextual help, and help drawers can show plugin and module documentation without evaluating JavaScript or building a second parallel documentation index.
WhenToUse: Use when implementing GOJA-12, reviewing the architecture for doc-aware REPL UX, or extending the docs hub to power richer module discovery in the TUI.
---


# Doc-aware js-repl autocomplete and help architecture and implementation guide

## Executive Summary

`js-repl` already has three partially overlapping systems:

- a parser-aware completion engine in `pkg/repl/evaluators/javascript/evaluator.go`
- a runtime-scoped documentation hub and `require("docs")` module from GOJA-11
- plugin manifests with rich module, export, and method docs

These systems are not wired together yet. The current TUI help path still relies mainly on:

- static signature strings in `helpBarSymbolSignatures`
- `jsparse` completion candidates
- shallow runtime inspection

That is enough for `console.log` and `Math.max`, but it does not surface plugin docs and it does not scale to a broader “modules in general” story.

The recommendation for GOJA-12 is:

1. Keep one source of truth for documentation: the Go-side `docaccess.Hub`.
2. Persist that hub, or a stable resolver over it, as runtime-scoped state owned by the runtime.
3. Teach the JavaScript evaluator to resolve contextual symbols back into `docaccess.Entry` records.
4. Feed those resolved docs into:
   - autocomplete row text
   - one-line help bar text
   - help drawer markdown
5. Start with plugin modules first because they already have rich structured docs.
6. Add a clean abstraction for “module docs in general” so native modules can join later without reworking the evaluator again.

This is intentionally not a JavaScript-level solution. The evaluator should not call `require("docs")` internally. It should use the same Go-side hub directly.

## Problem Statement

The user experience gap is straightforward:

- `js-repl` can complete plugin properties because `require()` aliases and runtime objects are visible.
- `js-repl` cannot explain those completions with the rich docs that are already present in plugin manifests and exposed through `require("docs")`.
- `js-repl` can show some static help for built-in objects, but that help is hand-maintained and disconnected from the new docs system.

For an intern, the most important thing to understand is that this is not “just a UI tweak.” It is an ownership and architecture problem:

- the parser knows what the user is typing
- the runtime knows what modules and aliases exist
- the docs hub knows what documentation exists
- the evaluator currently joins only the first two

GOJA-12 is about joining all three without creating circular dependencies or duplicated indexes.

### Symptoms visible today

- Typing `kv.store.g` can produce `get`, but help does not show the method doc body from the plugin manifest.
- Typing `const docs = require("docs")` and then `docs.byID(...)` works, but that path is separate from the TUI help path.
- The help bar can show `console.log(...args): void`, but that knowledge lives in a static map, not in the docs system.

### Why this matters

- Plugin users need discoverability inside the TUI, not only through manual docs queries.
- The new docs hub will not feel like a core feature if the primary interactive surface ignores it.
- A second documentation lookup path inside the evaluator would drift from `docaccess` and create more maintenance work.

## Current System Inventory

An intern implementing this ticket should read the following files first:

- `cmd/js-repl/main.go`
- `pkg/repl/evaluators/javascript/evaluator.go`
- `pkg/docaccess/runtime/registrar.go`
- `pkg/docaccess/plugin/provider.go`
- `engine/runtime.go`
- `engine/runtime_modules.go`
- `pkg/jsparse/repl_completion.go`

### What each part does

#### `cmd/js-repl/main.go`

This is the Bobatea entrypoint. It configures:

- plugin discovery flags
- the evaluator
- help system sources
- UI knobs like autocomplete and help drawer behavior

It already passes `HelpSources` into the evaluator config, which means the runtime-side docs registrar is active for the TUI. That is important: the data exists already. The missing piece is evaluator-side consumption.

#### `pkg/repl/evaluators/javascript/evaluator.go`

This is the core of the current TUI intelligence. It handles:

- JS execution
- parser-backed completions
- `require()` alias extraction
- one-line help bar text
- help drawer markdown

This file is the main implementation target for GOJA-12.

#### `pkg/docaccess/runtime/registrar.go`

This builds a `docaccess.Hub` per runtime and registers `require("docs")`.

This file already solves the “documentation source aggregation” problem for JavaScript callers. It does not yet solve the “evaluator-side direct access” problem.

#### `pkg/docaccess/plugin/provider.go`

This exposes plugin module/export/method docs as `docaccess.Entry` values. Because GOJA-11 made method docs first-class in the plugin manifest, this provider is already rich enough for the first version of GOJA-12.

#### `engine/runtime_modules.go`

This exposes `RuntimeModuleContext.Values`, which allows runtime module registrars to exchange runtime-scoped data during setup.

That is the right seam for registrar-to-registrar coordination. It is not yet a stable long-lived surface visible from the owned runtime object after creation.

#### `engine/runtime.go`

This is the owned runtime wrapper returned by the engine factory. If the evaluator wants direct access to runtime-scoped documentation state after creation, this type needs a clean field or accessor for that state.

#### `pkg/jsparse/repl_completion.go`

This file contains completion candidate shaping and `require()` alias-aware module completion helpers. GOJA-12 should not replace this system. It should enrich it with docs.

## Design Goals

### Primary goals

- Show plugin docs in `js-repl` autocomplete rows, help bar, and help drawer.
- Reuse the existing `docaccess` system instead of building another documentation index.
- Keep the system runtime-scoped so plugin/runtime differences are respected.
- Preserve the current completion engine and alias logic.

### Secondary goals

- Create an abstraction that can later include native module docs and jsdoc-backed module docs.
- Avoid evaluating JavaScript from the evaluator just to retrieve docs.
- Keep help behavior deterministic and fast enough for interactive typing.

### Non-goals for v1

- Replacing every static signature in `helpBarSymbolSignatures`
- Full docmgr-backed navigation inside the TUI
- A universal static module documentation registry for every built-in module
- Runtime reflection that invents docs for arbitrary user-defined objects

## Proposed Solution

The proposed architecture has four parts:

1. Persist the runtime-scoped docs hub, or a resolver derived from it, on the owned runtime.
2. Add an evaluator-side docs resolver that maps parsed symbols and aliases to `docaccess.Entry`.
3. Feed resolved docs into completion/help surfaces.
4. Keep plugin docs as the first-class initial source, then expand to module docs in general behind the same resolver abstraction.

### High-level data flow

```mermaid
flowchart LR
    A[cmd/js-repl/main.go] --> B[evaluator config]
    B --> C[engine factory]
    C --> D[plugin registrar]
    C --> E[docaccess runtime registrar]
    D --> F[loaded plugin manifests]
    F --> E
    E --> G[docaccess hub]
    E --> H[require(\"docs\") module]
    E --> I[runtime-scoped resolver state]
    J[user input in js-repl] --> K[jsparse completion context]
    K --> L[evaluator doc resolver]
    I --> L
    L --> M[autocomplete rows]
    L --> N[help bar]
    L --> O[help drawer]
```

### Core architectural decision

The evaluator must read docs from Go-side runtime state, not from JavaScript.

That means:

- good: `Evaluator` reads a `*docaccess.Hub` or `DocsResolver` from the owned runtime
- bad: `Evaluator` runs `require("docs").byID(...)` through the VM to ask for docs

Why the “bad” path is wrong:

- it is recursive
- it couples help behavior to JS execution semantics
- it adds overhead and failure modes during completion
- it creates awkward locking and reentrancy questions inside the evaluator

## Detailed Design

## Part 1: Runtime-scoped documentation state

### Problem

`RuntimeModuleContext.Values` exists during registrar execution, but `Evaluator` currently only holds:

- `runtime *goja.Runtime`
- `ownedRuntime *engine.Runtime`

There is no stable field on `engine.Runtime` for the values map created during module registration.

### Recommendation

Promote runtime-scoped values into `engine.Runtime`.

#### Proposed API shape

In `engine/runtime.go`:

```go
type Runtime struct {
    VM      *goja.Runtime
    Require *require.RequireModule
    Loop    *eventloop.EventLoop
    Owner   runtimeowner.Runner

    Values map[string]any

    // existing closer fields...
}

func (r *Runtime) Value(key string) (any, bool) {
    if r == nil || r.Values == nil || key == "" {
        return nil, false
    }
    v, ok := r.Values[key]
    return v, ok
}
```

Then in the runtime factory, copy the final `RuntimeModuleContext.Values` into the returned `Runtime`.

### Why this matters

This gives every later subsystem one stable place to read runtime-scoped state after construction:

- docs hub
- plugin load reports
- future inspector indexes
- future docmgr runtime navigation, if that ever exists

### Key design rule

Treat `Runtime.Values` as immutable after setup unless there is a strong reason not to. Interactive help should read stable snapshots, not chase mutating registries.

## Part 2: Persist the docs hub or resolver

### Option A: Persist the raw `*docaccess.Hub`

In `pkg/docaccess/runtime/registrar.go`:

```go
const RuntimeDocHubContextKey = "docaccess.hub"

func (r *Registrar) RegisterRuntimeModules(ctx *engine.RuntimeModuleContext, reg *require.Registry) error {
    hub, err := r.buildHub(ctx)
    if err != nil {
        return err
    }
    ctx.SetValue(RuntimeDocHubContextKey, hub)
    reg.RegisterNativeModule(r.moduleName(), loader(hub))
    return nil
}
```

Pros:

- simplest
- evaluator can query the same hub used by `require("docs")`

Cons:

- evaluator must know more about `docaccess` internals
- symbol-to-entry mapping logic still has to live elsewhere

### Option B: Persist a resolver object instead

Example:

```go
type SymbolDocsResolver interface {
    ResolveModule(moduleName string) (*docaccess.Entry, bool)
    ResolveExport(moduleName, exportName string) (*docaccess.Entry, bool)
    ResolveMethod(moduleName, exportName, methodName string) (*docaccess.Entry, bool)
}
```

Pros:

- narrower evaluator dependency
- easier to optimize and cache
- cleaner evolution path for module docs in general

Cons:

- slightly more code now

### Recommendation

Store the hub and build a resolver over it in the evaluator package or a small adapter package. That gives:

- one source of truth
- a narrow runtime lookup seam
- no need to invent a second persistence layer

## Part 3: Evaluator-side doc resolution

This is the heart of the ticket.

The evaluator needs to map current cursor context into documentation entities.

### Cases to support in v1

#### Case A: module alias base object

Input:

```javascript
const kv = require("plugin:examples:kv")
kv.
```

Interpretation:

- alias `kv` maps to module `plugin:examples:kv`
- user is exploring the module surface
- help should show module doc

#### Case B: object export

Input:

```javascript
const kv = require("plugin:examples:kv")
kv.store.
```

Interpretation:

- alias `kv` maps to module `plugin:examples:kv`
- `store` is an export on that module
- help should show export doc

#### Case C: object method

Input:

```javascript
const kv = require("plugin:examples:kv")
kv.store.get
```

Interpretation:

- alias `kv` maps to module `plugin:examples:kv`
- `store` is an export
- `get` is a method on that export
- help should show method doc

#### Case D: direct `require("plugin:...")`

Input:

```javascript
require("plugin:examples:kv")
```

This can be deferred for v1 if alias support covers the common path. It is useful, but not critical.

### Proposed evaluator helper

Add a new helper file near the evaluator, for example:

- `pkg/repl/evaluators/javascript/dochelp.go`

Suggested types:

```go
type docsResolver struct {
    hub *docaccess.Hub
}

type resolvedDoc struct {
    entry      *docaccess.Entry
    moduleName string
    exportName string
    methodName string
}
```

Suggested methods:

```go
func (e *Evaluator) docsHub() *docaccess.Hub
func (e *Evaluator) resolveDocsForContext(ctx jsparse.CompletionContext, aliases map[string]string, candidates []jsparse.CompletionCandidate) *resolvedDoc
func (e *Evaluator) resolveDocsForProperty(baseExpr, property string, aliases map[string]string) *resolvedDoc
func (e *Evaluator) resolveDocsForIdentifier(name string) *resolvedDoc
```

### Resolution algorithm for plugin docs

Pseudocode:

```go
func resolveDocsForProperty(baseExpr, property string, aliases map[string]string) *resolvedDoc {
    parts := splitExpr(baseExpr) // "kv.store" -> ["kv", "store"]
    if len(parts) == 0 {
        return nil
    }

    moduleName, ok := aliases[parts[0]]
    if !ok {
        return nil
    }

    if len(parts) == 1 && property == "" {
        return resolveModule(moduleName)
    }

    exportName := parts[1]
    if len(parts) == 2 && property == "" {
        return resolveExport(moduleName, exportName)
    }

    if len(parts) == 2 && property != "" {
        return resolveMethod(moduleName, exportName, property)
    }

    return nil
}
```

### Important constraint

This only works reliably for module APIs with stable shapes:

- module object
- export object
- method on export

That is acceptable for plugins because the manifest explicitly describes those shapes. It is not yet a generic answer for arbitrary runtime objects, and that is fine.

## Part 4: UI behavior by surface

The evaluator currently feeds three UI surfaces with different information density. GOJA-12 should preserve that pattern.

### Autocomplete popup

Goal:

- stay compact
- add one useful doc hint

Recommended display format:

- module: `plugin:examples:kv - Plugin Module - Example in-memory key-value store`
- export: `store - Plugin Export - Mutable store API`
- method: `get - Plugin Method - Return a stored value by key`

Implementation note:

- do not put full markdown here
- clip summaries to a short line, for example 60 to 90 characters

### Help bar

Goal:

- one-line contextual explanation under the input

Recommended format:

- `plugin:examples:kv - Example in-memory key-value store`
- `store - Mutable store API`
- `store.get - Return a stored value by key`

Fallback order:

1. docs-derived summary
2. existing static signature
3. existing runtime fallback

That ordering matters. Plugin docs should override generic `candidate.Detail`.

### Help drawer

Goal:

- rich markdown body
- more source metadata
- related links/refs

Recommended sections:

- Symbol
- Source
- Summary
- Body
- Related
- Completion candidates
- require() aliases

Example drawer body:

```markdown
### Symbol
- `plugin:examples:kv / store.get`

### Source
- Source: `plugin-manifests`
- Kind: `Plugin Method`
- Path: `/home/.../goja-plugin-examples-kv`

### Summary
- Return a stored value by key

### Documentation
Returns the current value associated with the given key. Returns null when the key is absent.

### Related
- Module: `plugin:examples:kv`
- Export: `store`
```

## Module Docs In General

The user explicitly asked about plugin docs “or modules in general.” The right design is:

- implement plugin docs first
- keep the resolver API generic enough to admit more module sources later

### Why plugin-first is correct

- plugin docs already exist in structured form
- alias resolution is already module-aware
- plugin module/export/method hierarchy matches the evaluator context model well

### What “modules in general” should mean later

At least three broader sources may exist later:

- native module docs for modules like `fs`, `path`, `url`, `docs`
- jsdoc-derived module docs for JavaScript code loaded into the runtime
- possibly Glazed help-backed application/module docs where that mapping makes sense

### Recommended future abstraction

Do not hardcode “plugin” into evaluator resolution APIs.

Instead define a generic module-doc resolver contract like:

```go
type ModuleDocsResolver interface {
    ResolveModule(moduleName string) (*docaccess.Entry, bool)
    ResolveMember(moduleName string, path []string) (*docaccess.Entry, bool)
}
```

Then implement:

- plugin-backed resolver first
- native-module resolver later

The evaluator should ask a generic resolver, not a plugin-only one.

## Design Decisions

### Decision 1: Do not use `require("docs")` internally

Reason:

- too indirect
- recursive
- ties help behavior to JS execution
- harder to test and reason about

### Decision 2: Runtime-scoped state belongs on the owned runtime

Reason:

- the evaluator already owns or accesses `engine.Runtime`
- runtime registrars produce setup-time data
- later help/status features will likely need similar state

### Decision 3: Plugin docs are the first implementation target

Reason:

- highest-value doc source already available
- easiest path to visible UX improvement
- strong fit with current alias-aware completion model

### Decision 4: Keep static signature fallback for now

Reason:

- built-in module docs are not fully modeled yet
- this avoids regressing current help for standard globals

### Decision 5: Use summaries for compact surfaces and bodies for the drawer

Reason:

- preserves readable UI
- matches the existing three-level information density of the REPL

## Alternatives Considered

## Alternative A: Call `require("docs")` from the evaluator

Rejected because:

- it is indirect and fragile
- it introduces reentrancy/locking concerns
- it is the wrong ownership layer

## Alternative B: Build a second docs index just for js-repl

Rejected because:

- duplicates `docaccess`
- risks drift between REPL docs and JS-visible docs
- doubles maintenance

## Alternative C: Keep using static signature maps and extend them for plugins

Rejected because:

- plugin docs are dynamic and runtime-scoped
- method docs already exist in manifests
- static maps would immediately become stale

## Alternative D: Solve “modules in general” first before plugins

Rejected because:

- too broad
- more source modeling work
- slower path to a useful result

## Implementation Plan

## Phase 1: Runtime state plumbing

- Add stable value access on `engine.Runtime`.
- Copy runtime module context values into the owned runtime.
- Add a named key for the docs hub in `pkg/docaccess/runtime/registrar.go`.
- Store the hub under that key when registering the docs module.

### Expected files

- `engine/runtime.go`
- `engine/factory.go`
- `pkg/docaccess/runtime/registrar.go`

## Phase 2: Evaluator-side docs resolver

- Add a helper file in `pkg/repl/evaluators/javascript`.
- Read the runtime-scoped docs hub from the owned runtime.
- Implement module/export/method resolution for plugin docs.
- Keep the API generic enough for future module-wide extension.

### Expected files

- `pkg/repl/evaluators/javascript/evaluator.go`
- `pkg/repl/evaluators/javascript/dochelp.go`

## Phase 3: Hook docs into autocomplete and help

- Enrich completion display text with doc summaries when available.
- Prefer docs-derived summaries in the help bar.
- Render full doc bodies in the help drawer.
- Keep existing static/rich fallback behavior for unsupported contexts.

### Expected files

- `pkg/repl/evaluators/javascript/evaluator.go`

## Phase 4: Tests

- unit tests for resolver behavior:
  - module resolution
  - export resolution
  - method resolution
  - alias parsing edge cases
- integration tests for:
  - plugin-aware autocomplete text
  - plugin-aware help bar text
  - plugin-aware help drawer markdown

### Expected files

- `pkg/repl/evaluators/javascript/evaluator_test.go`
- possibly a new focused `dochelp_test.go`

## Phase 5: Module-general follow-up seam

- Add the generic resolver interface even if only plugin-backed initially.
- Document deferred native-module integration.
- Optionally teach the `docs` module itself to self-describe through the same resolver path.

## Suggested Test Matrix

### Autocomplete

- `const kv = require("plugin:examples:kv"); kv.st`
  - suggestion `store`
  - display includes export summary

- `const kv = require("plugin:examples:kv"); kv.store.g`
  - suggestion `get`
  - display includes method summary

### Help bar

- cursor on `kv`
  - module summary shown

- cursor on `kv.store`
  - export summary shown

- cursor on `kv.store.get`
  - method summary shown

### Help drawer

- module body rendered
- export body rendered
- method body rendered
- related refs shown

### Fallbacks

- `console.log`
  - old static signature still works

- unknown user-defined object
  - no crash
  - fallback behavior preserved

## Example API Sketch

```go
const RuntimeDocHubContextKey = "docaccess.hub"

func (e *Evaluator) docsHub() *docaccess.Hub {
    if e.ownedRuntime == nil {
        return nil
    }
    value, ok := e.ownedRuntime.Value(RuntimeDocHubContextKey)
    if !ok {
        return nil
    }
    hub, _ := value.(*docaccess.Hub)
    return hub
}

func (e *Evaluator) resolvePluginMethodDoc(moduleName, exportName, methodName string) *docaccess.Entry {
    hub := e.docsHub()
    if hub == nil {
        return nil
    }
    entry, err := hub.FindByID(
        "plugin-manifests",
        "plugin-method",
        moduleName+"/"+exportName+"."+methodName,
    )
    if err != nil {
        return nil
    }
    return entry
}
```

## Review Guidance For An Intern

If you are implementing this for the first time, read in this order:

1. `pkg/docaccess/plugin/provider.go`
2. `pkg/docaccess/runtime/registrar.go`
3. `engine/runtime_modules.go`
4. `engine/runtime.go`
5. `pkg/jsparse/repl_completion.go`
6. `pkg/repl/evaluators/javascript/evaluator.go`
7. `cmd/js-repl/main.go`

Then ask these questions:

- Where does runtime-scoped state live today?
- Where does contextual symbol resolution happen today?
- Which help surface wants summary text versus full body text?
- Which path already knows `require()` aliases?
- Which path already knows plugin docs?

If you can answer those five questions, you understand the ticket.

## Open Questions

- Should the runtime store the raw hub, or should it store a narrower resolver object?
- Should the source ID `plugin-manifests` remain hardcoded in evaluator resolution, or should the registrar expose it centrally?
- Should autocomplete display strings include the source kind label by default, or only summary text?
- Should the help drawer gain explicit keyboard actions for “open related entry” later, or stay read-only?

## Recommendation Summary

Implement GOJA-12 as a plugin-first, runtime-scoped, Go-side docs integration:

- no second docs index
- no JS-level recursive docs lookup
- no attempt to solve every module source at once

The shortest path to value is:

1. persist runtime-scoped docs hub access on `engine.Runtime`
2. add evaluator-side plugin doc resolution
3. enrich completion/help bar/help drawer
4. keep a generic resolver seam for future non-plugin module docs

That gives `js-repl` immediate documentation-aware behavior while preserving the architecture already established in GOJA-11.
