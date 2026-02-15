---
Title: Inspector API Hybrid Service Guide
Slug: inspectorapi-hybrid-service-guide
Short: Build inspector workflows with the new pkg/inspectorapi facade and cleanly separate UI adapters from analysis/runtime logic.
Topics:
- inspector
- api
- architecture
- goja
- tooling
Commands:
- smalltalk-inspector
- inspector
- repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

`pkg/inspectorapi` is the user-facing facade for inspector workflows. It wraps reusable analysis, runtime, navigation, and tree helpers into task-oriented methods so adapters can stay thin and deterministic.

## Why This Layer Exists

The hybrid layer exists to stop UI models from owning business orchestration. Before `pkg/inspectorapi`, command models were directly composing `pkg/jsparse`, `pkg/inspector/analysis`, and `pkg/inspector/runtime` calls. That worked, but the same workflow logic was duplicated in adapter code and was difficult to reuse for REST or LSP endpoints.

The facade solves this by centralizing use cases:

- document session lifecycle (open, update, close)
- global/member listing and declaration jumps
- runtime global merge and REPL declaration extraction
- source/tree sync and tree row generation

This keeps adapter responsibilities focused on interaction and rendering.

## Package Overview

The package is intentionally small and explicit. `contracts.go` defines user-facing request/response DTOs and service errors. `service.go` implements use cases over the existing extracted packages.

Import path:

```go
import "github.com/go-go-golems/go-go-goja/pkg/inspectorapi"
```

Core constructor:

```go
svc := inspectorapi.NewService()
```

The service stores document sessions in-memory and exposes methods keyed by `DocumentID`.

## Core Workflow

Most integrations follow a simple lifecycle: open a document, run static/rich queries, optionally merge runtime data, then close the document session. This keeps stateful context explicit and makes adapter behavior testable.

### 1) Open a document

```go
resp, err := svc.OpenDocument(inspectorapi.OpenDocumentRequest{
    Filename: "sample.js",
    Source:   source,
})
if err != nil {
    return err
}

docID := resp.DocumentID
if len(resp.Diagnostics) > 0 {
    // parse diagnostics are still available even if analysis exists
}
```

### 2) List globals and members

```go
globalsResp, err := svc.ListGlobals(inspectorapi.ListGlobalsRequest{DocumentID: docID})
if err != nil {
    return err
}

for _, g := range globalsResp.Globals {
    fmt.Printf("%s (%s)\n", g.Name, g.Kind.String())
}

membersResp, err := svc.ListMembers(inspectorapi.ListMembersRequest{
    DocumentID: docID,
    GlobalName: "Foo",
}, nil)
if err != nil {
    return err
}
```

### 3) Jump to declarations

```go
bindingDecl, err := svc.BindingDeclarationLine(inspectorapi.BindingDeclarationRequest{
    DocumentID: docID,
    Name:       "Foo",
})
if err != nil {
    return err
}
if bindingDecl.Found {
    fmt.Printf("Foo declared at line %d\n", bindingDecl.Line)
}
```

### 4) Merge runtime globals

```go
rt := inspectorruntime.NewSession()
_ = rt.Load(source)

declared := inspectorapi.DeclaredBindingsFromSource(`const fromRepl = 1`)
merged, err := svc.MergeRuntimeGlobals(inspectorapi.MergeRuntimeGlobalsRequest{
    DocumentID: docID,
    Declared:   declared,
}, rt)
if err != nil {
    return err
}

_ = merged
```

### 5) Close document

```go
_ = svc.CloseDocument(inspectorapi.CloseDocumentRequest{DocumentID: docID})
```

## API Surface Reference

The service API is designed around use cases rather than parser internals. Each method takes a typed request struct and returns a typed response struct.

Document lifecycle:

- `OpenDocument`
- `OpenDocumentFromAnalysis`
- `UpdateDocument`
- `CloseDocument`
- `Analysis`

Static analysis use cases:

- `ListGlobals`
- `ListMembers`
- `BindingDeclarationLine`
- `MemberDeclarationLine`

Runtime-aware helpers:

- `DeclaredBindingsFromSource` (package-level helper)
- `MergeRuntimeGlobals`

Navigation/tree wrappers:

- `BuildTreeRows`
- `SyncSourceToTree`
- `SyncTreeToSource`

## Coordinate and State Conventions

A reliable integration depends on understanding coordinate and state expectations. The facade follows existing inspector conventions and keeps them explicit in request/response contracts.

- `SyncSourceToTreeRequest.CursorLine` and `CursorCol` are **0-based**.
- `BindingDeclarationLine` and `MemberDeclarationLine` return **1-based** source lines.
- Tree sync methods operate over current `Index.VisibleNodes()` ordering.
- `SyncSourceToTree` may expand index ancestors (`ExpandTo`) to maintain visibility semantics.

When building adapters, avoid ad hoc conversion logic in key handlers. Keep conversions in one boundary function.

## Adapter Integration Pattern

The preferred adapter design is: state model calls service methods, then maps DTOs to view state. The model should not re-implement analysis orchestration.

Recommended split:

- **Adapter model**: focus mode, key routing, view composition, and component state.
- **Service calls**: globals/members/jumps/merge/sync operations.
- **View rendering**: consume model state, not low-level analysis packages directly.

This is the same pattern used in the current `cmd/smalltalk-inspector` cutover.

## Testing Strategy

`pkg/inspectorapi` has focused service tests in `pkg/inspectorapi/service_test.go`. These validate end-to-end behavior for the new facade, including runtime merge and sync wrappers.

For changes to this package, run:

```bash
go test ./pkg/inspectorapi/... -count=1
go test ./cmd/smalltalk-inspector/... -count=1
go test ./... -count=1
```

If you change contracts, update both service tests and adapter tests in the same commit.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `ErrDocumentNotFound` from service calls | Document was never opened or already closed | Store returned `DocumentID` from `OpenDocument` and close only when done |
| `ListMembers` returns empty on value globals | Runtime session not provided for runtime-derived members | Pass `*runtime.Session` to `ListMembers` |
| Jump methods return `Found=false` | Name is missing, unresolved, or runtime-only | Verify global/member name comes from `ListGlobals`/`ListMembers` output |
| Source/tree sync not found | Cursor line/column out of range or no node at cursor | Clamp cursor in adapter and retry with current source state |
| Runtime merge misses REPL symbols | Declarations not extracted from snippet or runtime value undefined | Use `inspectorapi.DeclaredBindingsFromSource` and ensure symbol exists in runtime |

## Migration Notes (Clean Cutover)

If you are migrating an older adapter, remove direct usage of `inspector/analysis.Session` orchestration from the UI model and replace it with service calls. Do not add temporary compatibility wrappers unless explicitly required by a ticket.

Clean-cutover checklist:

1. Add `inspectorapi.Service` to model state.
2. Register/open document on file-load.
3. Replace global/member/jump orchestration with service methods.
4. Replace REPL declaration extraction with `inspectorapi.DeclaredBindingsFromSource`.
5. Replace runtime merge call with `MergeRuntimeGlobals`.
6. Update tests to construct/open document sessions via service.

## See Also

- `glaze help jsparse-framework-reference`
- `glaze help inspector-example-user-guide`
- `glaze help repl-usage`
- `glaze help creating-modules`
