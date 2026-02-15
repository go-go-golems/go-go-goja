---
Title: JSParse Framework Reference
Slug: jsparse-framework-reference
Short: Public reusable parsing, indexing, resolution, and completion APIs in pkg/jsparse
Topics:
- goja
- javascript
- parsing
- ast
- completion
Commands:
- repl
- inspector
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

`pkg/jsparse` is the reusable low-level analysis framework extracted from the inspector prototype. It provides parser-facing data structures and helper APIs that other tools can build on: static diagnostics, dev tooling, completion endpoints, and source-to-AST mapping workflows.

## Import Path

```go
import "github.com/go-go-golems/go-go-goja/pkg/jsparse"
```

## What Is Reusable vs Tool-Specific

The framework intentionally excludes Bubble Tea UI and command-specific behavior.

Reusable (`pkg/jsparse`):
- AST indexing (`BuildIndex`, `NodeRecord`, offset lookups)
- lexical scope/binding resolution (`Resolve`, `Resolution`)
- tree-sitter wrappers (`TSParser`, `TSNode`)
- completion analysis (`ExtractCompletionContext`, `ResolveCandidates`)
- high-level facade (`Analyze`, `AnalysisResult`)

Tool-specific (`cmd/inspector/app`):
- rendering
- keybindings
- pane synchronization UX
- interactive drawer controls

This boundary is the key contract: parser-centric features should depend on `pkg/jsparse`, not inspector command internals.

For user-facing workflow orchestration (document lifecycle, runtime merge, tree sync wrappers), prefer `pkg/inspectorapi` on top of `pkg/jsparse`.

## When to Use `pkg/jsparse` vs `pkg/inspectorapi`

Use `pkg/jsparse` when you need:
- low-level parser/index/resolution primitives
- custom completion heuristics and CST traversal logic
- direct control over parse and index lifecycle

Use `pkg/inspectorapi` when you need:
- adapter-facing workflows for CLI/REST/LSP surfaces
- session/document lifecycle and typed request/response contracts
- a stable high-level facade that already composes `pkg/jsparse` + inspector packages

## Why Two Parsers

`pkg/jsparse` intentionally combines:
- goja parser (`parser.ParseFile`) for typed AST + lexical scope resolution
- tree-sitter parser (`TSParser.Parse`) for error-tolerant cursor-local completion context

This gives semantic correctness for bindings while keeping completion usable on partially typed code.

## Core API Surface

### 1) Parse + Build Analysis Bundle

```go
result := jsparse.Analyze("example.js", source, nil)
if result.ParseErr != nil {
    for _, d := range result.Diagnostics() {
        log.Printf("%s: %s", d.Severity, d.Message)
    }
}
```

`AnalysisResult` contains:
- `Program` (`*ast.Program`) when parse produced an AST (even partial)
- `Index` for node navigation and source mapping
- `Resolution` for lexical bindings and references
- `ParseErr` plus normalized `Diagnostics()` output

### 2) Source Offset to AST Node

```go
node := result.NodeAtOffset(offset)
if node != nil {
    fmt.Printf("%s %s [%d,%d)\n", node.Kind, node.Label, node.Start, node.End)
}
```

Use this for cursor-hover inspectors, error annotation, and source navigation tools.

### 3) Completion Context + Candidates

High-level convenience methods:

```go
tsParser, _ := jsparse.NewTSParser()
defer tsParser.Close()
root := tsParser.Parse([]byte(source))

ctx := result.CompletionContextAt(root, row, col)
candidates := result.CompleteAt(root, row, col)
```

Low-level primitives (same behavior, more control):

```go
ctx := jsparse.ExtractCompletionContext(root, []byte(source), row, col)
candidates := jsparse.ResolveCandidates(ctx, result.Index, root)
```

`AnalysisResult.CompletionContextAt` and `AnalysisResult.CompleteAt` delegate to those low-level functions.

## Data Model Highlights

`Index`:
- node registry (`map[NodeID]*NodeRecord`)
- containment query (`NodeAtOffset`)
- tree operations (`VisibleNodes`, `ExpandTo`, `AncestorPath`)
- expand/collapse helper (`ToggleExpand`)
- source and position helpers (`Source`, `LineColToOffset`, `OffsetToLineCol`)

`Resolution`:
- scope graph (`Scopes`, `RootScopeID`)
- declaration-to-reference links (`NodeBinding`)
- unresolved identifiers list (`Unresolved`)
- lookup helpers (`BindingForNode`, `IsDeclaration`, `IsReference`, `IsUnresolved`)

`CompletionCandidate`:
- `Label`
- `Kind` (`CandidateProperty`, `CandidateMethod`, `CandidateVariable`, `CandidateFunction`, `CandidateKeyword`)
- `Detail` (display hint)

## NodeRecord Fields

`NodeRecord` contains source, tree, and display data:
- identity and shape: `ID`, `Kind`, `Label`, `Snippet`
- source span: `Start`, `End` (1-based offsets, end-exclusive)
- source position: `StartLine`, `StartCol`, `EndLine`, `EndCol` (1-based)
- tree links: `ParentID`, `ChildIDs`, `Depth`
- visibility state: `Expanded` (auto-expanded for `depth < 2` when index is built)

## Scope and Binding Model

Scope kinds:
- `ScopeGlobal`
- `ScopeFunction`
- `ScopeBlock`
- `ScopeCatch`
- `ScopeFor`

Binding kinds:
- `BindingVar`
- `BindingLet`
- `BindingConst`
- `BindingFunction`
- `BindingClass`
- `BindingParameter`
- `BindingCatchParam`

Hoisting behavior:
- `var` and `function` declarations resolve to nearest function/global scope
- `let` and `const` remain block scoped

Binding helpers:
- `BindingForNode(nodeID)` -> binding for an identifier node
- `IsDeclaration(nodeID)` / `IsReference(nodeID)` / `IsUnresolved(nodeID)`
- `AllUsages()` on `BindingRecord` returns declaration + references

### Example: Binding Lookup and Usages

```go
res := jsparse.Analyze("file.js", source, nil)
if res.Index == nil || res.Resolution == nil {
	return
}

node := res.NodeAtOffset(offset)
if node == nil {
	return
}
binding := res.Resolution.BindingForNode(node.ID)
if binding != nil {
	usages := binding.AllUsages()
	_ = usages
}
```

## TSParser and TSNode Lifecycle

- `NewTSParser` creates a JavaScript tree-sitter parser.
- `Parse` returns an owning snapshot tree (`*TSNode`).
- `TSNode` snapshots survive parser close/reparse (they are not raw `tree_sitter.Node` handles).
- helper methods:
  - `NodeAtPosition(row, col)`
  - `HasError()`
  - `ChildCount()`
- incremental parsing is intentionally not used in `Parse` (simple full reparse for current use case).

## Completion Model Notes

- `ResolveCandidates` supports optional drawer-local declarations through variadic `drawerRoot ...*TSNode`.
- `ExtractDrawerBindings(root)` extracts local variable/function declarations from drawer CST.
- built-in prototype/global knowledge currently includes:
  - `Object`, `Array`, `String`, `console`, `Math`, `JSON`

### Example: Tree Navigation and Position Helpers

```go
res := jsparse.Analyze("file.js", source, nil)
node := res.NodeAtOffset(offset)
if node == nil {
	return
}

path := res.Index.AncestorPath(node.ID)
res.Index.ExpandTo(node.ID)
line, col := res.Index.OffsetToLineCol(node.Start)
roundTrip := res.Index.LineColToOffset(line, col)
_ = path
_ = roundTrip
```

## Design Constraints and Guarantees

- API is pure-Go and command-agnostic.
- `Analyze` is deterministic for a given source/options under current parser/index ordering invariants.
- Completion APIs tolerate partial/error CST states.
- Parser errors do not prevent index/resolution when a partial AST is available.

Determinism invariant:
- maintain stable AST walk ordering and `OrderedByStart` sort behavior
- do not introduce unordered map iteration into externally visible result order

## Limitations

- Completion is heuristic and syntactic; there is no full type inference.
- `CompletionArgument` is currently reserved and yields no candidates.
- built-in prototype suggestions are hardcoded in `completion.go`.
- no module/import graph resolution is provided; analysis is per-source input.
- index child discovery uses reflection over goja AST structs; keep this in mind when upgrading goja AST internals.

## Common Integration Patterns

### Pattern A: Better parse errors with source context

1. Call `Analyze`.
2. Render `Diagnostics()` first.
3. If `Index` is non-nil, map diagnostic offsets to nearby node metadata.

If you need structured parse positions, inspect concrete parser error types:

```go
if errList, ok := res.ParseErr.(parser.ErrorList); ok && len(errList) > 0 {
	first := errList[0]
	_ = first.Position()
}
```

### Pattern B: Dev-tool endpoint for completion

1. Keep source text in memory.
2. Parse to CST via `TSParser`.
3. Use `CompletionContextAt` and `CompleteAt` per request.

### Pattern C: Static checks in CI

1. Analyze each JS file.
2. Fail on `Diagnostics()` severity `error`.
3. Optionally emit unresolved symbol summaries from `Resolution.Unresolved`.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `ParseErr` is non-nil and no index present | Parse failed before any partial AST was returned | Render `Diagnostics()`, inspect `res.ParseErr`, fix syntax at reported position |
| Completion candidates are empty after `.` | cursor/row/col do not point to `member_expression` or nearby `ERROR` fallback | retry at cursor and cursor-1; verify `root.NodeAtPosition(row,col)` and `root.HasError()` |
| Missing symbol links in `Resolution` | properties (`obj.foo`) are not lexical identifiers | use lexical resolution for identifiers, property completion for member access |
| Candidates miss local drawer vars | `ResolveCandidates` called without drawer CST | pass drawer root: `ResolveCandidates(ctx, idx, drawerRoot)` |
| Inconsistent behavior across modules | workspace masking or mismatched module graph | run with `GOWORK=off` and rerun `go mod tidy` |

## See Also

- `glaze help inspectorapi-hybrid-service-guide`
- `glaze help inspector-example-user-guide`
- `glaze help repl-usage`
- `glaze help async-patterns`
