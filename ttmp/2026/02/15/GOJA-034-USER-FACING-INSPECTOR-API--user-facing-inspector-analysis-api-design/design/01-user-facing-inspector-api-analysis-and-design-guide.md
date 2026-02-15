---
Title: User-Facing Inspector API Analysis and Design Guide
Ticket: GOJA-034-USER-FACING-INSPECTOR-API
Status: active
Topics:
    - go
    - goja
    - inspector
    - api
    - architecture
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/inspector/app/model.go
      Note: Legacy inspector Bubble Tea adapter layer
    - Path: cmd/smalltalk-inspector/app/model.go
      Note: |-
        Smalltalk inspector Bubble Tea adapter layer
        Adapter layer examples used in migration section
    - Path: internal/inspectorui/listpane.go
      Note: Reused viewport/list utility behavior in UI layer
    - Path: pkg/doc/07-inspectorapi-hybrid-service-guide.md
      Note: Help-system-facing documentation derived from hybrid API design
    - Path: pkg/inspector/analysis/globals_merge.go
      Note: Static+runtime global merge strategy
    - Path: pkg/inspector/analysis/repl_declarations.go
      Note: Parser-backed REPL declaration extraction
    - Path: pkg/inspector/analysis/session.go
      Note: High-level analysis session facade used by inspector UIs
    - Path: pkg/inspector/analysis/smalltalk_session.go
      Note: |-
        Globals/members/jump helpers and inspector-centric adapters
        Current high-level static analysis behavior mapped in the design
    - Path: pkg/inspector/core/members.go
      Note: Inheritance-aware class/function member extraction
    - Path: pkg/inspector/navigation/sync.go
      Note: |-
        UI-agnostic source/tree synchronization helpers
        Extracted sync primitives referenced in API sketches
    - Path: pkg/inspector/runtime/function_map.go
      Note: Runtime function to AST/source mapping
    - Path: pkg/inspector/runtime/introspect.go
      Note: Object/property/prototype introspection
    - Path: pkg/inspector/runtime/session.go
      Note: |-
        Runtime eval/session abstractions over goja
        Runtime session API discussed for facade contracts
    - Path: pkg/inspector/tree/rows.go
      Note: |-
        UI-agnostic tree row shaping
        Extracted tree DTO shaping referenced in API sketches
    - Path: pkg/jsparse/analyze.go
      Note: |-
        Core parser+index+resolution entry point for static analysis
        Core entrypoint that the proposed service layer should wrap
    - Path: pkg/jsparse/completion.go
      Note: CST/AST-based completion context and candidate resolution
    - Path: pkg/jsparse/highlight.go
      Note: Syntax span generation and syntax class model
    - Path: pkg/jsparse/index.go
      Note: Node index and source offset/tree visibility primitives
    - Path: pkg/jsparse/resolve.go
      Note: Scope and binding resolution model
ExternalSources: []
Summary: Deep onboarding and design guide for building a user-facing API over extracted inspector analysis/runtime/navigation functionality.
LastUpdated: 2026-02-15T11:09:00-05:00
WhatFor: Help new and existing developers converge on a clean public API and packaging strategy for analysis/inspection features.
WhenToUse: Read before designing new CLI/REST/LSP adapters or restructuring inspector service boundaries.
---



# User-Facing Inspector API Analysis and Design Guide

## Executive Summary

The repository now has a meaningful split between reusable inspection logic in `pkg/` and Bubble Tea application behavior in `cmd/`. We are at a good inflection point: enough extraction has happened to design a stable user-facing API, but not so much has solidified that we are locked into accidental abstractions.

This guide has four goals:

1. onboard a new developer quickly by mapping current architecture and capabilities,
2. explain how the current APIs actually work and what contracts already exist,
3. propose multiple user-facing API shapes with tradeoffs,
4. recommend a practical path that supports CLI, TUI, REST, and eventual LSP use without duplicating business logic.

The core recommendation is a **hybrid layered API**:

1. keep low-level primitives in `pkg/jsparse` and `pkg/inspector/*`,
2. add a new orchestration layer (proposed `pkg/inspectorapi` or `pkg/inspector/service`) that exposes stable use-case level operations,
3. keep Bubble Tea in `cmd/*` as thin adapters,
4. later expose the same orchestration over REST/CLI/LSP adapters.

This gives us fast iteration now and stable long-term packaging.

## Who This Is For

This document is written for:

1. a new engineer joining the codebase who needs to become productive quickly,
2. maintainers deciding how to package extracted functionality,
3. implementers building next adapters (REST endpoints, non-interactive CLI commands, LSP features).

If you are brand new, start with:

1. `pkg/jsparse/analyze.go`,
2. `pkg/inspector/analysis/session.go`,
3. `pkg/inspector/runtime/session.go`,
4. `cmd/smalltalk-inspector/app/model.go`,
5. this guide’s “Recommended Architecture” and “Phased Plan” sections.

## Project Context and Current State

### What has already been extracted

Recent tickets extracted reusable logic from inspector command models into packages:

1. runtime/global merge and REPL declaration helpers moved to `pkg/inspector/analysis` and `pkg/inspector/runtime`,
2. source/tree synchronization moved to `pkg/inspector/navigation`,
3. tree row shaping moved to `pkg/inspector/tree`,
4. view-scroll behavior consolidated in `internal/inspectorui` for Bubble Tea adapters.

This means we already have reuse across:

1. old inspector (`cmd/inspector`),
2. smalltalk inspector (`cmd/smalltalk-inspector`),
3. shared parsing/resolution engine (`pkg/jsparse`).

### What is still adapter-local

The following remain in `cmd/*` and should not become the public API themselves:

1. key handling and mode transitions,
2. Bubble Tea component wiring,
3. command palette and status-line presentation,
4. layout rendering and styling,
5. some orchestration glue now duplicated between frontends.

This remaining glue is exactly what a user-facing API layer should absorb.

## Architecture Map

### Layer map as it exists today

```text
+---------------------------------------------------------+
| cmd/smalltalk-inspector, cmd/inspector                  |
| Bubble Tea models, keymaps, views, pane orchestration   |
+-------------------------|-------------------------------+
                          |
+-------------------------v-------------------------------+
| pkg/inspector/*                                          |
| analysis, runtime, navigation, tree, core               |
| use-case helpers + DTO-level logic                       |
+-------------------------|-------------------------------+
                          |
+-------------------------v-------------------------------+
| pkg/jsparse                                               |
| parse, AST index, scope resolution, completion, syntax   |
+-------------------------|-------------------------------+
                          |
+-------------------------v-------------------------------+
| dependencies                                              |
| goja parser/runtime, tree-sitter JS                       |
+----------------------------------------------------------+
```

### Layer intent

1. `pkg/jsparse` is the language foundation.
2. `pkg/inspector/*` is domain behavior and workflows around parsed code and runtime objects.
3. `cmd/*` is interaction and presentation.

That split is mostly good, but it is missing an explicit “public orchestration layer” that presents stable, task-oriented APIs.

## Codebase Tour for New Developers

### Static analysis foundation: `pkg/jsparse`

Primary entry points:

1. `Analyze(filename, source, opts)` returns `AnalysisResult` with parse output, AST index, and scope resolution.
2. `BuildIndex(program, src)` constructs a flattened, queryable AST node index.
3. `Resolve(program, idx)` builds lexical scopes and binding graph.
4. `BuildSyntaxSpans(root)` classifies tree-sitter leaf tokens for highlighting.
5. `ExtractCompletionContext` and `ResolveCandidates` power completion.

Important contracts:

1. offsets in index are 1-based byte offsets,
2. line/col from index are 1-based,
3. tree-sitter node positions are 0-based row/col then adapted where needed,
4. `Index.VisibleNodes()` depends on `NodeRecord.Expanded` state.

Why this matters for API design:

1. any user-facing API must define consistent coordinate systems,
2. cursor and range conversions need explicit helper contracts,
3. parse and completion may rely on different parse trees (goja AST vs tree-sitter CST).

### Domain helpers: `pkg/inspector/analysis`

Current scope:

1. session wrapper around `AnalysisResult` (`Session`),
2. sorted global bindings + class extends metadata,
3. class/function members for inspector panels,
4. declaration jump helpers (`BindingDeclLine`, `MemberDeclLine`),
5. cross references and method symbol extraction,
6. REPL declaration extraction and static/runtime global merge.

This package is closest to a current “user API”, but still narrowly tuned to current UI workflows.

### Runtime helpers: `pkg/inspector/runtime`

Current scope:

1. goja runtime session lifecycle (`NewSession`, `Load`, `Eval`, `EvalWithCapture`),
2. value preview formatting,
3. object inspection (`InspectObject`),
4. prototype traversal (`WalkPrototypeChain`, `PrototypeChainNames`),
5. descriptor extraction (`GetDescriptor`),
6. error parsing (`ParseException`),
7. function-to-source mapping (`MapFunctionToSource`),
8. runtime global kind inference (`RuntimeGlobalKinds`).

This already has enough capability for CLI or REST object inspection, as long as we wrap it in safe request/response structs.

### Navigation and tree helpers

`pkg/inspector/navigation`:

1. source cursor to node mapping,
2. node selection to source cursor mapping,
3. visible node index lookup,
4. cursor offset calculations.

`pkg/inspector/tree`:

1. tree row view model data from index and highlights.

These two packages are explicitly UI-framework agnostic and are good examples of extraction style to continue.

### UI support utilities

`internal/inspectorui` has reusable scroll and viewport helpers.

These are internal and Bubble Tea-aware in part, which is fine for UI adapter reuse but should not be exposed as public analysis API.

## Capability Inventory (What We Can Do Today)

This section reframes current code as user-visible capabilities.

### Parse and structural analysis

We can currently:

1. parse JavaScript source with goja parser,
2. index AST nodes with spans, tree relations, display labels, visibility state,
3. resolve lexical scopes and bindings,
4. find node at source offset,
5. compute diagnostics from parse errors.

Current rough API path:

```go
result := jsparse.Analyze(filename, source, nil)
if result.ParseErr != nil { ... }
node := result.NodeAtOffset(123)
```

### Symbol and member exploration

We can currently:

1. list top-level globals and classify by binding kind,
2. determine class inheritance parent name,
3. compute class own + inherited members with cycle/depth guards,
4. compute top-level function parameters,
5. derive declaration lines for globals and members.

Current rough API path:

```go
s := analysis.NewSessionFromResult(result)
globals := s.Globals()
members := s.ClassMembers("MyClass")
line, ok := s.BindingDeclLine("foo")
```

### Cross-reference and scope-centric helpers

We can currently:

1. list xrefs for a binding name or record,
2. derive method-local symbol summaries in a source span.

Rough path:

```go
xrefs := analysis.CrossReferences(result.Resolution, result.Index, "foo")
symbols := analysis.MethodSymbols(result.Resolution, result.Index, start, end)
```

### Runtime inspection and REPL integration

We can currently:

1. run source into goja VM,
2. evaluate expressions with structured error capture,
3. inspect object properties and prototype chain,
4. parse stack frames from goja exceptions,
5. map runtime function values back to AST source when possible,
6. infer runtime global kinds and merge with static bindings,
7. extract REPL declarations from snippet text safely via parser.

Rough path:

```go
rt := runtime.NewSession()
_ = rt.Load(fileSource)
res := rt.EvalWithCapture("someExpr")
if obj, ok := res.Value.(*goja.Object); ok {
    props := runtime.InspectObject(obj, rt.VM)
    _ = props
}
```

### Syntax highlighting and completion

We can currently:

1. build tree-sitter CST snapshots,
2. derive syntax spans by token class,
3. compute completion context and candidates from CST + AST resolution,
4. include drawer-local declarations in completion.

This is useful for editor-like frontends, but current API is still low-level and split across multiple functions.

## Current API Friction Points

### Friction 1: no single use-case facade

Consumers currently compose many calls manually across packages.

Impact:

1. repeated orchestration code in UI adapters,
2. inconsistent error handling and fallback behavior,
3. harder onboarding for new developers.

### Friction 2: coordinate conventions are implicit

Some helpers use 0-based row/col inputs, others produce 1-based line/col, index offsets are 1-based byte indices.

Impact:

1. subtle bugs when adding endpoints or adapters,
2. duplicated conversion logic.

### Friction 3: static and runtime models are separate but intertwined

Smalltalk inspector merges static globals + runtime values + REPL declarations in model code.

Impact:

1. business logic lives in adapter layer,
2. hard to reuse from CLI/REST without copying behavior.

### Friction 4: DTOs are package-local and partially UI-shaped

Some DTOs are great (`tree.Row`, `navigation.SourceSelection`), others are implicit through package types.

Impact:

1. no explicit public contract set for third-party or future adapters,
2. model evolution risk without versioned API boundaries.

### Friction 5: lifecycle ownership is unclear for long-lived sessions

`runtime.Session` and `jsparse.TSParser` have separate lifecycles. Current UIs handle this ad hoc.

Impact:

1. memory/resource leaks risk in future server mode,
2. inconsistent caching semantics.

## User-Facing API Design Goals

A new user-facing API should:

1. expose task-oriented operations, not parser internals,
2. stay transport-agnostic (usable by TUI, CLI, REST, LSP),
3. provide explicit DTOs and stable contracts,
4. centralize coordinate and range conventions,
5. compose static and runtime information predictably,
6. make session and cache ownership explicit,
7. preserve ability to call low-level packages directly for advanced needs.

Non-goals for initial version:

1. full language-server protocol in one step,
2. replacing all existing package-level APIs,
3. implementing distributed multi-user runtime state,
4. changing existing parser engines.

## Design Options

## Option A: Library-First Facade

### Idea

Add a Go package with high-level methods and typed request/response structs. No server assumptions.

Potential package:

1. `pkg/inspectorapi`.

Example API sketch:

```go
type Service struct {
    parserFactory ParserFactory
    runtimeFactory RuntimeFactory
}

func (s *Service) AnalyzeFile(ctx context.Context, req AnalyzeFileRequest) (AnalyzeFileResponse, error)
func (s *Service) ListGlobals(ctx context.Context, req ListGlobalsRequest) (ListGlobalsResponse, error)
func (s *Service) InspectValue(ctx context.Context, req InspectValueRequest) (InspectValueResponse, error)
func (s *Service) Complete(ctx context.Context, req CompleteRequest) (CompleteResponse, error)
```

Pros:

1. quickest to ship,
2. easy to integrate into existing cmd apps,
3. strong type safety in Go.

Cons:

1. REST/LSP adapters still need another layer later,
2. might accidentally leak Go-specific types into public contract.

Best when:

1. immediate priority is code reuse across in-repo Go apps.

## Option B: Service-First (REST-centric)

### Idea

Define HTTP API first and treat Go packages as implementation detail.

Pros:

1. strongest language/tooling neutrality,
2. easiest for external consumers.

Cons:

1. higher upfront scope,
2. harder local debugging initially,
3. risks premature stabilization of contracts before internal API matures.

Best when:

1. external clients are immediate priority.

## Option C: Hybrid Layered API (Recommended)

### Idea

Build a use-case layer with stable DTOs and interfaces in Go first, then expose adapter packages:

1. Go facade package for internal consumers,
2. optional REST adapter that reuses same service methods,
3. optional LSP/CLI adapters later.

Pros:

1. fast internal value with strong architecture,
2. avoids duplicate orchestration logic,
3. clean migration path to REST/LSP,
4. easier testing at domain layer.

Cons:

1. requires discipline to keep adapters thin,
2. needs careful DTO and interface design upfront.

Best when:

1. we want both near-term implementation speed and long-term reuse.

## Recommended Architecture (Hybrid)

### Proposed package layout

```text
pkg/inspectorapi/
  contracts.go          // shared request/response DTOs and enums
  coordinates.go        // canonical position/range helpers
  service.go            // high-level Service facade
  analysis_service.go   // static analysis use-cases
  runtime_service.go    // runtime eval/inspect use-cases
  completion_service.go // completion + syntax span use-cases
  session_store.go      // document/runtime session lifecycle

pkg/inspectorapi/adapters/
  rest/...
  cli/...
  tui/...
  lsp/... (future)
```

Keep existing packages as implementation dependencies:

1. `pkg/jsparse`,
2. `pkg/inspector/analysis`,
3. `pkg/inspector/runtime`,
4. `pkg/inspector/navigation`,
5. `pkg/inspector/tree`,
6. `pkg/inspector/core`.

### Proposed domain model

Introduce explicit concepts:

1. `DocumentSession`: static source + analysis snapshot + optional CST cache,
2. `RuntimeSession`: goja VM associated with a document or ephemeral snippet,
3. `InspectorService`: stateless facade over stores/managers,
4. `Snapshot`: immutable analysis/range/tree data returned to clients.

### Key principle

The service should return DTOs that are independent of Bubble Tea, goja internals, and AST node concrete types.

## API Contract Sketches

## Core contracts

```go
type Position struct {
    Line int // 1-based
    Col  int // 1-based
}

type ByteOffset struct {
    Value int // 1-based byte offset
}

type Range struct {
    Start Position
    End   Position // exclusive
}

type NodeRef struct {
    NodeID int
    Kind   string
    Label  string
    Range  Range
}
```

## Session API

```go
type OpenDocumentRequest struct {
    URI      string
    Source   string
    Language string // "javascript"
}

type OpenDocumentResponse struct {
    DocumentID string
    Diagnostics []Diagnostic
}

type UpdateDocumentRequest struct {
    DocumentID string
    Source     string
}

type UpdateDocumentResponse struct {
    Diagnostics []Diagnostic
    Version     int
}
```

## Analysis API

```go
type ListGlobalsRequest struct {
    DocumentID string
    IncludeRuntime bool
}

type GlobalEntry struct {
    Name    string
    Kind    string
    Extends string
    Source  string // "static" | "runtime" | "merged"
}

type ListGlobalsResponse struct {
    Globals []GlobalEntry
}
```

```go
type ListMembersRequest struct {
    DocumentID string
    SymbolName string
}

type MemberEntry struct {
    Name      string
    Kind      string
    Preview   string
    Inherited bool
    Source    string
}

type ListMembersResponse struct {
    Members []MemberEntry
}
```

```go
type GoToDeclarationRequest struct {
    DocumentID string
    SymbolName string
    Context    string // optional class/function scope
}

type GoToDeclarationResponse struct {
    Location *Location
}

type Location struct {
    URI   string
    Range Range
}
```

## Runtime inspection API

```go
type EvalRequest struct {
    RuntimeID   string
    Expression  string
    CaptureMeta bool
}

type EvalResponse struct {
    ValuePreview string
    ValueKind    string
    Error        *ErrorInfo
    ObjectHandle string
}
```

```go
type InspectObjectRequest struct {
    RuntimeID     string
    ObjectHandle  string
    IncludeProto  bool
    IncludeHidden bool
}

type PropertyEntry struct {
    Name       string
    Kind       string
    Preview    string
    IsSymbol   bool
    HasGetter  bool
    HasSetter  bool
    ChildHandle string
}

type InspectObjectResponse struct {
    Properties []PropertyEntry
    PrototypeChain []PrototypeEntry
}
```

## Navigation and tree API

```go
type SyncSourceToTreeRequest struct {
    DocumentID string
    Cursor     Position
}

type SyncSourceToTreeResponse struct {
    SelectedNode NodeRef
    VisibleIndex int
    Highlight    Range
}
```

```go
type BuildTreeRowsRequest struct {
    DocumentID      string
    UsageHighlights []int
}

type TreeRow struct {
    NodeID      int
    Title       string
    Description string
}

type BuildTreeRowsResponse struct {
    Rows []TreeRow
}
```

## Completion and syntax API

```go
type CompleteRequest struct {
    DocumentID string
    Cursor     Position
    Prefix     string
    IncludeRuntime bool
}

type CompletionItem struct {
    Label  string
    Kind   string
    Detail string
}

type CompleteResponse struct {
    Items []CompletionItem
}
```

```go
type SyntaxSpansRequest struct {
    DocumentID string
}

type SyntaxSpan struct {
    Start Position
    End   Position
    Class string
}

type SyntaxSpansResponse struct {
    Spans []SyntaxSpan
}
```

## Orchestration Pseudocode

### Open/update document lifecycle

```go
func (svc *Service) OpenDocument(ctx context.Context, req OpenDocumentRequest) (OpenDocumentResponse, error) {
    result := jsparse.Analyze(req.URI, req.Source, nil)

    doc := &DocumentSession{
        ID:       newID(),
        URI:      req.URI,
        Source:   req.Source,
        Analysis: result,
        Version:  1,
    }

    if svc.enableCST {
        doc.CST = svc.ts.Parse([]byte(req.Source))
    }

    svc.docs.Put(doc)

    return OpenDocumentResponse{
        DocumentID: doc.ID,
        Diagnostics: mapDiagnostics(result.Diagnostics()),
    }, nil
}
```

```go
func (svc *Service) UpdateDocument(ctx context.Context, req UpdateDocumentRequest) (UpdateDocumentResponse, error) {
    doc := svc.docs.MustGet(req.DocumentID)
    doc.Source = req.Source
    doc.Analysis = jsparse.Analyze(doc.URI, doc.Source, nil)
    doc.Version++

    if doc.CSTParser != nil {
        doc.CST = doc.CSTParser.Parse([]byte(doc.Source))
    }

    return UpdateDocumentResponse{
        Diagnostics: mapDiagnostics(doc.Analysis.Diagnostics()),
        Version:     doc.Version,
    }, nil
}
```

### Globals flow with static+runtime merge

```go
func (svc *Service) ListGlobals(ctx context.Context, req ListGlobalsRequest) (ListGlobalsResponse, error) {
    doc := svc.docs.MustGet(req.DocumentID)
    s := analysis.NewSessionFromResult(doc.Analysis)

    globals := s.Globals()
    if !req.IncludeRuntime {
        return ListGlobalsResponse{Globals: mapGlobals(globals, "static")}, nil
    }

    rt := svc.runtime.ForDocument(doc.ID)
    runtimeKinds := runtime.RuntimeGlobalKinds(rt.VM)
    declared := doc.REPLDeclared

    merged := analysis.MergeGlobals(
        globals,
        runtimeKinds,
        declared,
        func(name string) bool {
            v := rt.GlobalValue(name)
            return v != nil && !goja.IsUndefined(v)
        },
    )

    return ListGlobalsResponse{Globals: mapGlobals(merged, "merged")}, nil
}
```

### Cursor sync flow

```go
func (svc *Service) SyncSourceToTree(ctx context.Context, req SyncSourceToTreeRequest) (SyncSourceToTreeResponse, error) {
    doc := svc.docs.MustGet(req.DocumentID)
    idx := doc.Analysis.Index
    lines := strings.Split(doc.Source, "\n")

    sel, ok := navigation.SelectionAtSourceCursor(
        idx,
        lines,
        req.Cursor.Line-1,
        req.Cursor.Col-1,
    )
    if !ok {
        return SyncSourceToTreeResponse{}, ErrNoNodeAtCursor
    }

    idx.ExpandTo(sel.NodeID)
    visible := idx.VisibleNodes()
    visibleIdx := navigation.FindVisibleNodeIndex(visible, sel.NodeID)

    node := idx.Nodes[sel.NodeID]
    return SyncSourceToTreeResponse{
        SelectedNode: mapNode(node),
        VisibleIndex: visibleIdx,
        Highlight:    mapOffsetsToRange(idx, sel.HighlightStart, sel.HighlightEnd),
    }, nil
}
```

### Runtime evaluation and object inspection flow

```go
func (svc *Service) Eval(ctx context.Context, req EvalRequest) (EvalResponse, error) {
    rt := svc.runtimes.MustGet(req.RuntimeID)
    result := rt.EvalWithCapture(req.Expression)

    if result.Error != nil {
        var info *runtime.ErrorInfo
        if ex, ok := result.Error.(*goja.Exception); ok {
            parsed := runtime.ParseException(ex)
            info = &parsed
        }
        return EvalResponse{
            ValuePreview: "",
            ValueKind:    "error",
            Error:        info,
        }, nil
    }

    handle := ""
    kind := classifyValue(result.Value)
    if obj, ok := result.Value.(*goja.Object); ok {
        handle = svc.objects.Store(rt.ID, obj)
    }

    return EvalResponse{
        ValuePreview: runtime.ValuePreview(result.Value, rt.VM, 80),
        ValueKind:    kind,
        ObjectHandle: handle,
    }, nil
}
```

## API Shape Decisions That Matter

### Decision 1: explicit coordinate system

Recommendation:

1. all user-facing contracts use 1-based `Position` and `Range`,
2. keep internal adapters for 0-based UI or tree-sitter values,
3. centralize conversions in one utility module.

Why:

1. line-oriented tools and editors often present 1-based lines,
2. prevents repeated off-by-one bugs.

### Decision 2: session identity and mutability

Recommendation:

1. document sessions identified by `DocumentID`,
2. version increment per update,
3. runtime session either per document or explicit detached runtime for REPL experiments.

Why:

1. enables deterministic caches,
2. makes REST and LSP synchronization tractable.

### Decision 3: sync vs async boundaries

Recommendation:

1. keep parse/eval APIs synchronous initially,
2. permit context cancellation,
3. add async job model only if heavy workloads appear.

Why:

1. complexity stays low while workloads are small.

### Decision 4: error model

Recommendation:

1. structured domain errors (`ErrDocumentNotFound`, `ErrRuntimeNotFound`, `ErrInvalidPosition`),
2. include optional machine code + user message,
3. preserve raw runtime error payload where possible.

Why:

1. adapter layers need predictable HTTP status/CLI exits.

## Migration Strategy from Current Code

### Step 1: define contracts without moving code

1. create `pkg/inspectorapi/contracts.go` and map current package outputs into DTOs,
2. write pure mapping tests.

### Step 2: wrap existing logic with service methods

1. implement analysis flows by calling `jsparse` + `inspector/analysis`,
2. implement runtime flows by calling `inspector/runtime`,
3. implement navigation/tree flows via `inspector/navigation` and `inspector/tree`.

### Step 3: reduce command-model orchestration

1. move repeated model logic to service methods,
2. leave keymaps/view rendering untouched.

### Step 4: add one non-TUI adapter

1. add minimal CLI command or HTTP handler that consumes `inspectorapi.Service`,
2. validate no Bubble Tea imports in service package.

### Step 5: tighten compatibility tests

1. add golden or behavioral parity tests comparing old command behavior and new service-backed behavior,
2. freeze first contract set before externalizing.

## Suggested Public Interfaces (Detailed)

### Service interface

```go
type InspectorService interface {
    OpenDocument(ctx context.Context, req OpenDocumentRequest) (OpenDocumentResponse, error)
    UpdateDocument(ctx context.Context, req UpdateDocumentRequest) (UpdateDocumentResponse, error)
    CloseDocument(ctx context.Context, req CloseDocumentRequest) error

    ListGlobals(ctx context.Context, req ListGlobalsRequest) (ListGlobalsResponse, error)
    ListMembers(ctx context.Context, req ListMembersRequest) (ListMembersResponse, error)
    GoToDeclaration(ctx context.Context, req GoToDeclarationRequest) (GoToDeclarationResponse, error)
    CrossReferences(ctx context.Context, req CrossReferencesRequest) (CrossReferencesResponse, error)

    BuildTreeRows(ctx context.Context, req BuildTreeRowsRequest) (BuildTreeRowsResponse, error)
    SyncSourceToTree(ctx context.Context, req SyncSourceToTreeRequest) (SyncSourceToTreeResponse, error)
    SyncTreeToSource(ctx context.Context, req SyncTreeToSourceRequest) (SyncTreeToSourceResponse, error)

    Eval(ctx context.Context, req EvalRequest) (EvalResponse, error)
    InspectObject(ctx context.Context, req InspectObjectRequest) (InspectObjectResponse, error)
    PrototypeChain(ctx context.Context, req PrototypeChainRequest) (PrototypeChainResponse, error)

    Complete(ctx context.Context, req CompleteRequest) (CompleteResponse, error)
    SyntaxSpans(ctx context.Context, req SyntaxSpansRequest) (SyntaxSpansResponse, error)
}
```

### Storage interfaces

```go
type DocumentStore interface {
    Put(*DocumentSession)
    Get(id string) (*DocumentSession, bool)
    Delete(id string)
}

type RuntimeStore interface {
    Put(*RuntimeSession)
    Get(id string) (*RuntimeSession, bool)
    Delete(id string)
}
```

These interfaces allow:

1. in-memory implementation now,
2. pluggable persistence later if needed.

## Decision Matrix

| Criterion | Option A Library-First | Option B Service-First | Option C Hybrid (Recommended) |
|---|---:|---:|---:|
| Time to first value | High | Low | High |
| Internal code reuse | Medium | Medium | High |
| External API readiness | Low | High | Medium |
| Risk of premature contracts | Medium | High | Medium |
| Testability | High | Medium | High |
| Migration complexity | Low | High | Medium |
| Long-term flexibility | Medium | Medium | High |

Interpretation:

1. Option C gives strongest total score for this codebase phase.

## New Developer Onboarding Path

### Day 1 reading order

1. `pkg/jsparse/analyze.go`,
2. `pkg/jsparse/index.go`,
3. `pkg/inspector/analysis/smalltalk_session.go`,
4. `pkg/inspector/runtime/session.go`,
5. `pkg/inspector/navigation/sync.go`,
6. `cmd/smalltalk-inspector/app/model.go`.

### Day 1 runnable checklist

1. run `go test ./pkg/jsparse ./pkg/inspector/... -count=1`,
2. run `go test ./cmd/smalltalk-inspector/... ./cmd/inspector/... -count=1`,
3. open and follow one behavior path end-to-end:
   1. load file,
   2. list globals,
   3. jump to source,
   4. evaluate expression,
   5. inspect object.

### Day 2 contribution starter tasks

1. add one service-level DTO and mapper,
2. add one service method wrapping existing helper logic,
3. add tests for coordinate conversion edge-cases,
4. wire one cmd flow through new service method.

## Risks and Mitigations

### Risk: DTO churn during early design

Mitigation:

1. mark first contracts as experimental version,
2. isolate mappers so internal types can evolve.

### Risk: leaking goja/tree-sitter types into public API

Mitigation:

1. keep public DTO package free of goja/tree-sitter imports,
2. compile-time checks in adapter packages.

### Risk: behavior drift between command apps

Mitigation:

1. shared service parity tests,
2. maintain targeted command regression tests.

### Risk: runtime session safety in server mode

Mitigation:

1. avoid shared mutable runtime across concurrent requests unless locked,
2. expose explicit runtime/session ownership semantics.

## Testing Strategy for the New User-Facing API

### Unit tests

1. DTO mapping correctness,
2. coordinate conversion correctness,
3. service methods with mocked stores.

### Integration tests

1. parse to globals to members pipeline,
2. eval to inspect to function-map pipeline,
3. completion and syntax spans against known fixture files.

### Contract tests

1. JSON serialization stability for REST payloads,
2. backwards compatibility checks for key response fields.

### Parity tests

1. compare outputs of cmd model flow before and after service adoption on fixed fixtures,
2. ensure tree sync and highlight ranges remain equivalent.

## Implementation Brainstorming Notes

### Idea: “analysis snapshot” object

Introduce immutable snapshot object to avoid accidental mutation of shared index state:

```go
type AnalysisSnapshot struct {
    DocumentID string
    Version    int
    Result     *jsparse.AnalysisResult
}
```

Usage:

1. session updates create new snapshot,
2. read APIs consume snapshot by value or pointer without mutating index visibility directly,
3. UI-specific visibility state moves out of shared index if needed.

Tradeoff:

1. may need a separate `TreeState` structure to track expand/collapse per client.

### Idea: separate “tree state” from AST index

Current `Index` stores `Expanded` in nodes, which is convenient but UI-stateful.

Alternative:

1. keep index immutable structural data,
2. track expanded node IDs in per-client `TreeViewState`.

Sketch:

```go
type TreeViewState struct {
    Expanded map[int]bool
}

func VisibleNodes(idx *jsparse.Index, st TreeViewState) []int
```

Benefit:

1. avoids cross-client bleed in server contexts.

Cost:

1. more plumbing in existing cmd apps.

Recommendation:

1. defer until multi-client adapter appears,
2. design service methods so migration remains possible.

### Idea: runtime object handles

For REST/LSP adapters, object values cannot cross process boundaries directly.

Use handle registry:

1. store goja objects in runtime session object map,
2. return short opaque handles in API responses,
3. inspect by handle in follow-up calls.

Sketch:

```go
type ObjectRegistry interface {
    Put(runtimeID string, obj *goja.Object) string
    Get(runtimeID, handle string) (*goja.Object, bool)
    DeleteRuntime(runtimeID string)
}
```

### Idea: capability flags on responses

Add lightweight capability metadata to help adapters avoid assumptions.

Example:

```go
type Capability struct {
    Name string // e.g. "runtime-map-function-source"
    Enabled bool
}
```

Use case:

1. if runtime map-to-source is unavailable, UI can show fallback quickly.

## Recommended Next Steps

1. Create a follow-up implementation ticket to scaffold `pkg/inspectorapi` with contracts and minimal service skeleton.
2. Implement first four methods end-to-end:
   1. `OpenDocument`,
   2. `ListGlobals`,
   3. `ListMembers`,
   4. `GoToDeclaration`.
3. Move static+runtime globals merge behavior from `cmd/smalltalk-inspector/app/model.go` into service layer.
4. Add `SyncSourceToTree` and `BuildTreeRows` service methods using already extracted packages.
5. Wire one command path in `cmd/smalltalk-inspector` through service layer and add parity tests.
6. Add a thin CLI or REST proof-of-concept adapter to validate transport neutrality.

## Appendix A: Detailed API Mapping from Existing Code

### Existing function to future service mapping

| Existing API | Proposed service operation |
|---|---|
| `jsparse.Analyze` | `OpenDocument`, `UpdateDocument` |
| `analysis.Session.Globals` | `ListGlobals` |
| `analysis.Session.ClassMembers`, `FunctionMembers` | `ListMembers` |
| `analysis.Session.BindingDeclLine`, `MemberDeclLine` | `GoToDeclaration` |
| `analysis.CrossReferences` | `CrossReferences` |
| `runtime.Session.EvalWithCapture` | `Eval` |
| `runtime.InspectObject` | `InspectObject` |
| `runtime.WalkPrototypeChain` | `PrototypeChain` |
| `runtime.MapFunctionToSource` | `ResolveRuntimeFunctionLocation` |
| `navigation.SelectionAtSourceCursor` | `SyncSourceToTree` |
| `navigation.SelectionFromVisibleTree` | `SyncTreeToSource` |
| `tree.BuildRowsFromIndex` | `BuildTreeRows` |
| `jsparse.ResolveCandidates` | `Complete` |
| `jsparse.BuildSyntaxSpans` | `SyntaxSpans` |

## Appendix B: Practical “How to Decide” Checklist

When evaluating an API addition, use this checklist:

1. Is this behavior domain logic or UI interaction logic?
2. Can it be represented with plain DTOs and primitive types?
3. Does it need document session state, runtime state, or both?
4. What coordinate system does it accept and return?
5. Is behavior deterministic and testable without UI framework?
6. Can this be reused by at least two adapters (TUI + one of CLI/REST/LSP)?
7. What are failure modes, and are they represented structurally?
8. Does this leak internal dependency types?

If answers are strong on 1, 2, 5, 6, it belongs in user-facing service API.

## Appendix C: Minimal First Implementation Example

```go
// Package inspectorapi
func (s *Service) ListMembers(ctx context.Context, req ListMembersRequest) (ListMembersResponse, error) {
    doc, ok := s.docs.Get(req.DocumentID)
    if !ok {
        return ListMembersResponse{}, ErrDocumentNotFound
    }

    sess := analysis.NewSessionFromResult(doc.Analysis)
    globals := sess.Globals()

    var selected *analysis.GlobalBinding
    for i := range globals {
        if globals[i].Name == req.SymbolName {
            selected = &globals[i]
            break
        }
    }
    if selected == nil {
        return ListMembersResponse{}, nil
    }

    switch selected.Kind {
    case jsparse.BindingClass:
        members := sess.ClassMembers(selected.Name)
        return ListMembersResponse{Members: mapClassMembers(members)}, nil
    case jsparse.BindingFunction:
        members := sess.FunctionMembers(selected.Name)
        return ListMembersResponse{Members: mapFunctionMembers(members)}, nil
    default:
        return s.listRuntimeValueMembers(ctx, doc, selected.Name)
    }
}
```

This style keeps existing domain logic intact while giving adapters a stable surface.

## Closing Note

The hard part is no longer “how do we inspect JavaScript?”. We already do that across static and runtime paths. The hard part now is packaging and contract quality. A disciplined service layer with explicit DTOs will let us move quickly without re-embedding logic into each frontend.

This is the right moment to lock in those boundaries.
