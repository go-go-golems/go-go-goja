---
Title: Unified documentation access architecture and implementation guide
Ticket: GOJA-11-DOC-ACCESS-SURFACES
Status: active
Topics:
    - goja
    - architecture
    - tooling
    - js-bindings
    - glazed
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: Runtime creation flow that should host the future runtime-scoped documentation hub
    - Path: modules/glazehelp/glazehelp.go
      Note: Old JS-facing help module whose strengths and limitations shape the new design
    - Path: pkg/docaccess/hub.go
      Note: Shared hub abstraction implemented from the ticket design
    - Path: pkg/docaccess/plugin/provider.go
      Note: Plugin provider that exposes module/export/method docs
    - Path: pkg/docaccess/runtime/registrar.go
      Note: Runtime-scoped registrar that builds the per-runtime docs hub and JS module
    - Path: pkg/hashiplugin/contract/jsmodule.proto
      Note: |-
        Defines what plugin metadata exists today and therefore what the docs hub can expose
        Plugin manifest contract updated to carry first-class method docs
    - Path: pkg/hashiplugin/host/catalog.go
      Note: Retained plugin manifest snapshot used as docs-provider input
    - Path: pkg/hashiplugin/host/reify.go
      Note: Shows where plugin docs are currently lost during JS reification
    - Path: pkg/jsdoc/model/store.go
      Note: Canonical jsdoc aggregation model that the new provider must adapt
ExternalSources: []
Summary: Detailed intern-facing design for a unified documentation access layer that can expose Glazed help, jsdoc stores, and plugin metadata consistently to both Go callers and JavaScript runtimes.
LastUpdated: 2026-03-18T15:42:01.102157415-04:00
WhatFor: Explain how to design a shared documentation hub and adapter system for Glazed help, jsdocex, and plugin metadata without over-coupling existing subsystems.
WhenToUse: Use when implementing, reviewing, or extending documentation lookup APIs for Go callers, JavaScript callers, or future docmgr-backed sources.
---



# Unified documentation access architecture and implementation guide

## Executive Summary

`go-go-goja` already contains three meaningful documentation systems, but each one is currently trapped inside its own local API:

- Glazed help content is exposed through `help.HelpSystem` and command-specific embedded docs.
- JavaScript source documentation is exposed through the `goja-jsdoc` extractor, `DocStore`, and HTTP/CLI surfaces.
- Plugin metadata is carried in plugin manifests and SDK declarations, but is only partially surfaced to users today.

There is also an older native module attempt, `modules/glazehelp`, which proves that exposing documentation into JavaScript is possible, but also shows why a narrow one-off wrapper is not enough. It is global instead of runtime-scoped, Glazed-specific instead of source-agnostic, and returns untyped maps rather than participating in a broader documentation model.

The recommendation in this ticket is to introduce a single shared Go-side documentation hub with provider adapters for each source type:

- `glazed` provider for `*help.HelpSystem`
- `jsdoc` provider for `*model.DocStore`
- `plugin` provider for loaded plugin manifests and export metadata
- later, optionally, a `docmgr` provider

That hub should be reusable directly from Go and also exposed to JavaScript through one runtime-scoped native module, tentatively `docs`. The hub should not erase source-specific richness. Instead, it should offer a stable shared query/get/render surface plus source-specific metadata bags and escape hatches where needed.

This document is intentionally detailed for a new intern. It explains the current code, the problem, the architectural options, the recommended design, concrete APIs, runtime integration points, risks, and a phased implementation plan.

## Problem Statement

### The practical problem

Today, a developer who wants documentation inside `go-go-goja` has to know which subsystem they are talking to:

- If they want command/help-page docs, they need a `help.HelpSystem`.
- If they want JavaScript symbol docs, they need `pkg/jsdoc`.
- If they want plugin docs, they have to know about plugin manifests and host load reports.
- If they want any of this from JavaScript, they need ad hoc glue.

This is manageable for core maintainers, but not a good long-term user experience.

### Why this matters

There are at least two real user groups:

1. Go-side integrators
They want programmatic access to documentation for building CLIs, UIs, inspectors, REPL help, or exported docs.

2. JavaScript-side users
They want to inspect the runtime from inside scripts or REPL sessions:
- "What modules exist?"
- "What does this plugin export?"
- "Show me help for this topic"
- "What symbols did jsdocex extract from my package?"

These users should not have to learn three different data models and three different integration stories for what is conceptually one thing: documentation lookup.

### Evidence in the current repository

The current codebase shows the fragmentation clearly.

#### Surface 1: Glazed help is embedded, loaded, and CLI-bound

- `pkg/doc/doc.go` loads embedded help sections into a `help.HelpSystem`.
- `cmd/repl/main.go` creates a help system, loads docs, and wires Cobra help commands via `help_cmd.SetupCobraRootCommand(...)`.
- `cmd/goja-jsdoc/main.go` does the same thing for the `goja-jsdoc` CLI.

This surface is rich, but mostly lives in Go CLI wiring.

#### Surface 2: jsdocex is a real documentation engine, but separate

- `pkg/jsdoc/model/model.go` defines `Package`, `SymbolDoc`, `Example`, and `FileDoc`.
- `pkg/jsdoc/model/store.go` aggregates those into a `DocStore`.
- `pkg/jsdoc/server/server.go` serves a UI and API.
- `cmd/goja-jsdoc/*` exposes the extractor and server.

This surface is already reusable, but not integrated into runtime help or the existing native module system.

#### Surface 3: plugin docs exist, but are only half-exposed

- `pkg/hashiplugin/contract/jsmodule.proto` includes `ModuleManifest.doc` and `ExportSpec.doc`.
- `pkg/hashiplugin/sdk/module.go` lets authors set module docs and export docs.
- `pkg/hashiplugin/host/reify.go` reifies exports into JS functions and objects, but drops doc metadata during that reification.
- `pkg/hashiplugin/host/report.go` exposes loaded names/versions/exports, but not rich docs.

This is the most obviously unfinished part of the story.

#### Surface 4: the old `glazehelp` module is useful, but too narrow

- `modules/glazehelp/glazehelp.go`
- `modules/glazehelp/registry.go`
- `modules/glazehelp/glazehelp_test.go`

This module proves that:
- we can expose documentation into JavaScript,
- a small wrapper is easy to test,
- users want query/get/list access.

But it also reveals the limitations:
- global registry instead of runtime-scoped ownership,
- only for Glazed help systems,
- returns plain maps with no shared doc identity model,
- no relation to jsdoc or plugin metadata.

### What “good” looks like

A better system should let code on either side of the Go/JavaScript boundary ask consistent questions:

- What documentation sources are available?
- What kinds of entries do they contain?
- Search or filter entries
- Fetch one entry by stable reference
- Render or summarize it
- Discover source-specific metadata when needed

That is the core problem this ticket addresses.

## Proposed Solution

### Recommendation in one sentence

Add a shared Go-side documentation hub with pluggable providers, then expose that hub to JavaScript through a runtime-scoped native module.

### Design stance

This design is intentionally not “perfect unification”.

The hub should:

- provide a stable shared abstraction for query/get/render,
- preserve source-specific richness in metadata,
- avoid forcing Glazed help, jsdoc, and plugin manifests into an artificially identical schema,
- remain small enough to ship incrementally.

### Recommended package layout

The cleanest package split is something like:

```text
pkg/docaccess/
  model.go          // shared entry/source/ref/query types
  registry.go       // hub / registry / service
  provider.go       // provider interfaces
  query.go          // query filtering helpers
  render.go         // render helpers / shapes

pkg/docaccess/glazed/
  provider.go       // adapts *help.HelpSystem

pkg/docaccess/jsdoc/
  provider.go       // adapts *model.DocStore

pkg/docaccess/plugin/
  provider.go       // adapts loaded plugin manifests

modules/docs/
  docs.go           // runtime-scoped JS module exposing the hub
```

I would keep `modules/glazehelp` around only as a temporary compatibility layer, then eventually reimplement it as a thin wrapper over the new hub or deprecate it outright.

### Shared data model

The shared model should be intentionally small.

#### `SourceDescriptor`

Represents one provider instance.

Suggested shape:

```go
type SourceKind string

const (
    SourceKindGlazedHelp SourceKind = "glazed-help"
    SourceKindJSDoc      SourceKind = "jsdoc"
    SourceKindPlugin     SourceKind = "plugin"
    SourceKindDocmgr     SourceKind = "docmgr"
)

type SourceDescriptor struct {
    ID          string
    Kind        SourceKind
    Title       string
    Summary     string
    RuntimeScoped bool
    Metadata    map[string]any
}
```

Examples:

- `default-help` for the main `repl` help system
- `jsdoc:workspace` for a loaded jsdoc store
- `plugin-manifests` for loaded runtime plugins

#### `EntryRef`

Stable identifier for one documentation item.

```go
type EntryRef struct {
    SourceID string
    Kind     string
    ID       string
}
```

Examples:

- `{SourceID:"default-help", Kind:"help-section", ID:"repl-usage"}`
- `{SourceID:"plugin-manifests", Kind:"plugin-export", ID:"plugin:examples:kv/store.get"}`
- `{SourceID:"jsdoc:workspace", Kind:"symbol", ID:"smoothstep"}`

#### `Entry`

Shared top-level representation returned to callers.

```go
type Entry struct {
    Ref        EntryRef
    Title      string
    Summary    string
    Body       string
    Topics     []string
    Tags       []string
    Path       string
    KindLabel  string
    Related    []EntryRef
    Metadata   map[string]any
}
```

Important design point:
- `Body` should be optional and usually Markdown-ish text.
- `Metadata` carries source-specific fields that should not be forced into the common shape.

#### `Query`

The initial query API should be deliberately simple.

```go
type Query struct {
    Text      string
    SourceIDs []string
    Kinds     []string
    Topics    []string
    Tags      []string
    Limit     int
}
```

This is not meant to replace the Glazed help DSL. It is meant to provide a shared lowest-friction query layer. Source-specific richer query APIs can still exist behind metadata or explicit provider hooks.

### Provider interface

The provider API should align with how people naturally inspect docs:

```go
type Provider interface {
    Descriptor() SourceDescriptor
    List(ctx context.Context) ([]EntryRef, error)
    Get(ctx context.Context, ref EntryRef) (*Entry, error)
    Search(ctx context.Context, q Query) ([]Entry, error)
}
```

Optional future extension:

```go
type Renderer interface {
    Render(ctx context.Context, ref EntryRef, opts RenderOptions) (*RenderedEntry, error)
}
```

Do not require that extension in v1. `Get(...)` returning a body string is enough initially.

### Hub / registry API

The hub should be the Go-side orchestration layer.

```go
type Hub struct {
    mu        sync.RWMutex
    providers map[string]Provider
}

func NewHub() *Hub
func (h *Hub) Register(p Provider) error
func (h *Hub) Sources() []SourceDescriptor
func (h *Hub) Get(ctx context.Context, ref EntryRef) (*Entry, error)
func (h *Hub) Search(ctx context.Context, q Query) ([]Entry, error)
func (h *Hub) FindByID(sourceID, kind, id string) (*Entry, error)
```

The hub should merge results across providers, sort them deterministically, and preserve source identity.

### JavaScript surface

The JavaScript API should be one module, not three unrelated ones.

Recommended module name:

- `docs`

Potential JS API:

```javascript
const docs = require("docs")

docs.sources()
docs.search({ text: "plugin", kinds: ["plugin-module"] })
docs.get({ sourceId: "plugin-manifests", kind: "plugin-module", id: "plugin:examples:kv" })
docs.bySlug("default-help", "repl-usage")
docs.bySymbol("jsdoc:workspace", "smoothstep")
```

Suggested exported functions:

- `sources(): SourceDescriptor[]`
- `search(query): Entry[]`
- `get(ref): Entry | null`
- `byID(sourceId, kind, id): Entry | null`
- `summary(ref): string`

Keep this API data-oriented. Do not try to mirror every source-specific method in v1.

### Runtime integration model

The module must be runtime-scoped, not global.

That is one of the key lessons from `modules/glazehelp`.

The implementation should fit the existing runtime architecture:

- `engine.Factory` already supports runtime-scoped module registration through `RuntimeModuleRegistrar`.
- plugin registration already uses that seam in `pkg/hashiplugin/host`.
- the documentation hub should use the same ownership model.

Recommended runtime flow:

```text
engine.Factory.NewRuntime()
  -> create VM + require.Registry
  -> register static modules
  -> build runtime-scoped doc hub
  -> attach glazed/jsdoc/plugin providers
  -> register "docs" native module bound to that hub
  -> enable require()
  -> run runtime initializers
```

### Architecture diagram

```mermaid
flowchart LR
    A[Go caller] --> H[docaccess.Hub]
    J[JavaScript require('docs')] --> H

    H --> G[glazed provider]
    H --> S[jsdoc provider]
    H --> P[plugin provider]
    H -. optional .-> D[docmgr provider]

    G --> GH[help.HelpSystem]
    S --> JS[pkg/jsdoc/model.DocStore]
    P --> PM[plugin manifests / loaded modules]
    D --> DM[docmgr ticket/docs metadata]
```

### Why this is the right shape

Because it balances three competing needs:

- one conceptual API for users,
- source-specific richness for maintainers,
- runtime-scoped ownership for correctness.

## Design Decisions

### Decision 1: use a hub + providers, not one monolithic schema loader

Rationale:

- Glazed help, jsdoc, and plugin metadata already exist as mature but different systems.
- Adapters let us reuse them instead of rewriting them.
- A provider system keeps the docmgr future path open without forcing that dependency now.

### Decision 2: runtime-scoped JavaScript access is required

Rationale:

- Plugin metadata is inherently runtime-scoped.
- `modules/glazehelp` currently uses a global registry, which is workable for tests but weak for real multi-runtime ownership.
- The engine already has a runtime-scoped registration seam; use it.

### Decision 3: the shared model should be intentionally shallow

Rationale:

- Trying to map every Glazed field and every jsdoc field into one giant universal schema would make the API hard to understand and hard to keep stable.
- A small shared `Entry` plus `Metadata` is easier to ship and extend.

### Decision 4: preserve source identity explicitly

Rationale:

- Users should always know where a result came from.
- A plugin export doc and a jsdoc symbol doc may both be called `greet`; source identity prevents ambiguity.

### Decision 5: keep source-specific advanced APIs out of v1

Rationale:

- The goal is to make shared doc inspection easy.
- Rich Glazed DSL queries, jsdoc graph traversal, and docmgr full-text/taxonomy navigation can remain source-native until there is a proven need to unify them further.

### Decision 6: `docmgr` is a future provider, not a required dependency

Rationale:

- The user explicitly framed `docmgr` as optional.
- `docmgr` metadata is likely useful for future navigation and authoring workflows, but it should not block the simpler Glazed/jsdoc/plugin story.

## Design Brainstorm

This section is intentionally broader than the final recommendation. It shows the main approaches that are plausible and why I would or would not choose them.

### Option A: keep separate modules

Examples:

- `glazehelp`
- `jsdoc`
- `plugins`

Pros:

- lowest implementation risk
- each subsystem stays native to itself

Cons:

- users still need to know multiple APIs
- no shared query or navigation experience
- plugin docs still need a new surface anyway
- duplicates lookup/render concepts across modules

Verdict:
- not good enough

### Option B: flatten everything into one universal document type

Pros:

- superficially simple API
- easy to serialize

Cons:

- discards important source-specific semantics
- risks turning into a giant unmaintainable schema
- encourages lowest-common-denominator design

Verdict:
- too flattening

### Option C: shared hub plus provider adapters

Pros:

- consistent user experience
- preserves source richness
- easy incremental rollout
- aligns with existing runtime/plugin architecture

Cons:

- some adapter code is unavoidable
- shared query semantics must stay intentionally modest

Verdict:
- recommended

### Option D: build everything around `docmgr`

Pros:

- potentially very rich metadata and navigation
- good long-term story for generated and human-authored docs

Cons:

- much heavier dependency and mental model
- not all runtime docs naturally belong in ticket/docmgr space
- overkill for the immediate JS/Go runtime need

Verdict:
- good future influence, bad current core

## Detailed Source Adapters

### Glazed provider

#### What it wraps

- `*help.HelpSystem`

#### Mapping

- `help.Section` -> `Entry{Kind:"help-section"}`
- top-level page groups -> either:
  - synthetic entries, or
  - omitted from v1 in favor of section-level access

#### Useful metadata to preserve

- `slug`
- `commands`
- `flags`
- `sectionType`
- `isTopLevel`
- `showPerDefault`

#### Notes

The Glazed provider should not try to expose the full internal DSL as the shared query language. If needed later, it can add a source-specific metadata filter or auxiliary API.

### jsdoc provider

#### What it wraps

- `*model.DocStore`

#### Mapping

- `Package` -> `Entry{Kind:"package"}`
- `SymbolDoc` -> `Entry{Kind:"symbol"}`
- `Example` -> `Entry{Kind:"example"}`

#### Useful metadata to preserve

- source file
- line number
- params
- returns
- concepts
- related
- docpage

#### Notes

This provider is the strongest argument for a common docs hub: `pkg/jsdoc` is already a documentation system, but it is currently isolated from runtime help.

### Plugin provider

#### What it wraps

- loaded plugin manifests
- later, possibly plugin discovery reports too

#### Mapping

- module manifest -> `Entry{Kind:"plugin-module"}`
- export spec -> `Entry{Kind:"plugin-export"}`
- object methods -> synthetic child entries such as `Entry{Kind:"plugin-method"}`

#### Useful metadata to preserve

- version
- capabilities
- module path
- export kind
- method names

#### Plugin manifest extension required

The original design analysis treated method-level plugin docs as a limitation of the existing protobuf contract. That should not survive implementation. Since the plugin system is not published yet, GOJA-11 should extend the contract now and make method docs first-class rather than designing around their absence.

Recommended contract shape:

```proto
message MethodSpec {
  string name = 1;
  string summary = 2;
  string doc = 3;
  repeated string tags = 4;
}

message ExportSpec {
  string name = 1;
  ExportKind kind = 2;
  string doc = 3;
  repeated MethodSpec method_specs = 4;
}
```

Important decision:

- do not keep the old `repeated string methods` field for compatibility
- regenerate the protobuf code and update the SDK and host together

Rationale:

- the plugin system has not been published yet
- carrying temporary compatibility fields would complicate validation, summarization, and provider code for no durable benefit

With this change:

- module docs stay on `ModuleManifest.doc`
- export docs stay on `ExportSpec.doc`
- object methods gain first-class rich docs through `MethodSpec.doc`
- the plugin provider can expose honest method entries rather than synthetic placeholders

### Future docmgr provider

#### What it might wrap

- ticket indexes
- design docs
- reference docs
- related file mappings

#### Why it is attractive later

- good for project- and codebase-level navigation
- could let JS tooling or inspectors surface design notes next to runtime objects

#### Why it is not part of the MVP

- much bigger scope
- likely needs search/indexing decisions beyond this ticket

## Go API Sketch

### Core API

```go
hub := docaccess.NewHub()
_ = hub.Register(glazedprovider.New("default-help", helpSystem))
_ = hub.Register(jsdocprovider.New("workspace-jsdoc", store))
_ = hub.Register(pluginprovider.New("plugin-manifests", manifests))

sources := hub.Sources()
results, _ := hub.Search(ctx, docaccess.Query{Text: "plugin"})
entry, _ := hub.FindByID("plugin-manifests", "plugin-module", "plugin:examples:kv")
```

### Runtime-scoped installer sketch

```go
builder := engine.NewBuilder().
    WithModules(engine.DefaultRegistryModules()).
    WithRuntimeModuleRegistrars(
        docaccessruntime.NewRegistrar(docaccessruntime.Config{
            HelpSystem:    helpSystem,
            JSDocStore:    store,
            PluginSource:  pluginSource,
        }),
    )
```

## JavaScript API Sketch

### Happy-path examples

```javascript
const docs = require("docs")

docs.sources()
```

```javascript
docs.search({ text: "repl", sourceIds: ["default-help"] })
```

```javascript
docs.get({
  sourceId: "plugin-manifests",
  kind: "plugin-module",
  id: "plugin:examples:kv",
})
```

```javascript
docs.search({ kinds: ["symbol"], text: "smoothstep" })
```

### Suggested result shape

```javascript
{
  ref: { sourceId: "default-help", kind: "help-section", id: "repl-usage" },
  title: "REPL Usage",
  summary: "How to use repl and js-repl",
  body: "...markdown...",
  topics: ["goja", "repl"],
  tags: [],
  path: "pkg/doc/04-repl-usage.md",
  kindLabel: "Help Section",
  related: [],
  metadata: {
    slug: "repl-usage",
    commands: ["repl", "js-repl"],
    flags: ["--plugin-dir"],
    sectionType: "GeneralTopic"
  }
}
```

## Implementation Plan

The plan below is intentionally phased so the system can ship without a huge flag day.

### Phase 1: shared core package

Build:

- extend the plugin manifest schema to support method docs without compatibility fields
- `pkg/docaccess/model.go`
- `pkg/docaccess/provider.go`
- `pkg/docaccess/registry.go`

Deliver:

- `SourceDescriptor`
- `EntryRef`
- `Entry`
- `Query`
- `Provider`
- `Hub`

Validation:

- protobuf regeneration compiles cleanly
- unit tests for provider registration, duplicate IDs, merged search ordering, and not-found behavior

### Phase 2: Glazed help provider

Build:

- `pkg/docaccess/glazed/provider.go`

Deliver:

- adapter from `*help.HelpSystem` to shared entry/search/get

Validation:

- adapt fixtures similar to `modules/glazehelp/glazehelp_test.go`

### Phase 3: jsdoc provider

Build:

- `pkg/docaccess/jsdoc/provider.go`

Deliver:

- package/symbol/example mapping from `*model.DocStore`

Validation:

- tests against `testdata/jsdoc/*`

### Phase 4: plugin provider

Build:

- `pkg/docaccess/plugin/provider.go`
- `pkg/hashiplugin/contract/jsmodule.proto`
- regenerated `pkg/hashiplugin/contract/jsmodule*.pb.go`

Deliver:

- runtime-scoped provider over loaded plugin manifests

Implementation notes:

- the host needs a retained manifest source, not just `LoadReport`, because `LoadReport` currently summarizes loaded state for humans rather than preserving a full queryable manifest model
- the provider should expose module, export, and method entries directly from those retained manifests
- reification should stay callable-function oriented; the richer metadata belongs in the docs provider and reporting layers

Validation:

- integration test using example plugins
- tests covering method-level docs

### Phase 5: runtime-scoped JS module

Build:

- `modules/docs/docs.go`
- `engine` registrar glue or runtime initializer glue

Deliver:

- `require("docs")`
- runtime-scoped access to the hub

Validation:

- line REPL smoke test
- `js-repl` evaluator integration test

### Phase 6: compatibility and migration

Decide:

- keep `glazehelp` as thin compatibility wrapper, or
- deprecate it and move callers to `docs`

Recommendation:

- keep it temporarily as a wrapper over the new hub

### Phase 7: optional docmgr provider

Only after phases 1 through 5 are stable:

- define what “docmgr entries” should look like
- decide whether search is in-process or delegated
- decide how much file/ticket metadata belongs in the runtime

## Pseudocode For Hub Search

```text
function Hub.Search(ctx, query):
  providers = providersMatching(query.SourceIDs)
  results = []

  for provider in providers:
    entries = provider.Search(ctx, query)
    for entry in entries:
      if matchesSharedFilters(entry, query):
        results.append(entry)

  sort results by:
    source id
    kind
    title

  if query.limit > 0:
    truncate results

  return results
```

## Pseudocode For Plugin Adapter

```text
function PluginProvider.Get(ref):
  if ref.kind == "plugin-module":
    manifest = manifests[ref.id]
    return entryFromManifest(manifest)

  if ref.kind == "plugin-export":
    moduleName, exportName = split(ref.id)
    manifest = manifests[moduleName]
    export = findExport(manifest, exportName)
    return entryFromExport(manifest, export)

  if ref.kind == "plugin-method":
    moduleName, exportName, methodName = splitMethod(ref.id)
    export = findExport(...)
    return syntheticMethodEntry(manifest, export, methodName)
```

## Risks And Failure Modes

### Risk 1: too much abstraction

If the shared model grows too large, it will be harder to understand than the separate systems it tries to unify.

Mitigation:

- keep the common model small
- rely on `Metadata` for richness

### Risk 2: runtime/global confusion

If Go-side registries remain global while JS-side access becomes runtime-scoped, the mental model will be inconsistent.

Mitigation:

- make the new `docs` module runtime-scoped from day one

### Risk 3: plugin metadata is incomplete

Plugin method docs are not first-class in the current protobuf contract.

Mitigation:

- surface what exists honestly
- treat method docs as a separate future ticket if needed

### Risk 4: search semantics drift

Glazed help search and jsdoc search are not naturally identical.

Mitigation:

- make shared search intentionally simple
- do not promise source-native query parity in v1

## Alternatives Considered

### Reusing `modules/glazehelp` as the central API

Rejected because:

- it is global, not runtime-scoped
- it only knows about Glazed help
- it bakes in a source-specific name and assumptions

### Extending plugin reports into a docs system

Rejected because:

- plugin metadata is only one of the sources
- `LoadReport` is for status reporting, not documentation navigation

### Making `goja-jsdoc` the documentation hub

Rejected because:

- jsdoc is an important provider, but not the right architectural owner for command help and plugin manifests

## File Map For A New Intern

Read these in this order:

1. `pkg/doc/doc.go`
What embedded CLI help looks like in this repo.

2. `cmd/repl/main.go`
How help and plugin reporting are wired today in the main line REPL.

3. `modules/glazehelp/glazehelp.go`
The old JS-facing help wrapper and its limitations.

4. `pkg/jsdoc/model/model.go`
The jsdoc data model.

5. `pkg/jsdoc/model/store.go`
How jsdoc docs are indexed.

6. `pkg/hashiplugin/contract/jsmodule.proto`
What plugin metadata actually contains.

7. `pkg/hashiplugin/host/reify.go`
Where plugin docs are currently dropped on the floor.

8. `engine/factory.go`
Where runtime-scoped registration belongs.

## Open Questions

### Question 1: should the shared module be named `docs`, `docaccess`, or `help`?

My recommendation:

- `docs`

Reason:

- short
- neutral across sources
- reads well in JS

### Question 2: should Glazed help DSL be surfaced directly?

My recommendation:

- not in v1

Reason:

- it complicates the shared model too early

### Question 3: should plugin metadata use `LoadReport` or a richer retained manifest store?

My recommendation:

- richer retained manifest store

Reason:

- `LoadReport` is a presentation summary, not a documentation source of truth

### Question 4: should docmgr be queried live or exported into a provider shape?

My recommendation:

- export or adapt selected docmgr metadata later, not live-couple the first implementation to docmgr internals

## References

- `modules/glazehelp/glazehelp.go`
- `modules/glazehelp/registry.go`
- `modules/glazehelp/glazehelp_test.go`
- `pkg/doc/doc.go`
- `cmd/repl/main.go`
- `cmd/js-repl/main.go`
- `pkg/repl/evaluators/javascript/evaluator.go`
- `pkg/jsdoc/model/model.go`
- `pkg/jsdoc/model/store.go`
- `cmd/goja-jsdoc/main.go`
- `cmd/goja-jsdoc/doc/01-jsdoc-system.md`
- `pkg/hashiplugin/contract/jsmodule.proto`
- `pkg/hashiplugin/host/reify.go`
- `pkg/hashiplugin/host/report.go`
- `pkg/hashiplugin/sdk/module.go`
- `engine/runtime_modules.go`
- `engine/factory.go`
- `ttmp/2025-07-30/01-design-for-a-go-go-goja-module-for-the-glazed-helpsystem.md`

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
