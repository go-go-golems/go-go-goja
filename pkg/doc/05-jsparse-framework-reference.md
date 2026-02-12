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

`pkg/jsparse` is the reusable analysis framework extracted from the inspector prototype. It provides parser-facing data structures and helper APIs that other tools can build on: static diagnostics, dev tooling, completion endpoints, and source-to-AST mapping workflows.

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

This boundary is the key contract: new tools should depend on `pkg/jsparse`, not inspector internals.

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

```go
tsParser, _ := jsparse.NewTSParser()
defer tsParser.Close()
root := tsParser.Parse([]byte(source))

ctx := result.CompletionContextAt(root, row, col)
candidates := result.CompleteAt(root, row, col)
```

This is suitable for editor-like completion features without importing UI code.

## Data Model Highlights

`Index`:
- node registry (`map[NodeID]*NodeRecord`)
- containment query (`NodeAtOffset`)
- tree operations (`VisibleNodes`, `ExpandTo`, `AncestorPath`)

`Resolution`:
- scope graph (`Scopes`, `RootScopeID`)
- declaration-to-reference links (`NodeBinding`)
- unresolved identifiers list (`Unresolved`)

`CompletionCandidate`:
- `Label`
- `Kind` (`property`, `method`, `variable`, ...)
- `Detail` (display hint)

## Design Constraints and Guarantees

- API is pure-Go and command-agnostic.
- `Analyze` is deterministic for a given source/options.
- Completion APIs tolerate partial/error CST states.
- Parser errors do not prevent index/resolution when a partial AST is available.

## Common Integration Patterns

### Pattern A: Better parse errors with source context

1. Call `Analyze`.
2. Render `Diagnostics()` first.
3. If `Index` is non-nil, map diagnostic offsets to nearby node metadata.

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
| `ParseErr` is non-nil and no index present | Parse failed before AST could be built | Check syntax first; rerun after fixing parse blockers |
| Completion candidates look generic only | CST context not property/identifier-specific at cursor | Verify row/col and use cursor and cursor-1 fallback strategy |
| Missing symbol links in `Resolution` | Identifier is property access, not lexical binding | Treat property completion separately from lexical resolution |
| Inconsistent behavior across modules | Workspace dependencies masking missing go.mod requirements | Validate with `GOWORK=off` in CI and local checks |

## See Also

- `inspector-example-user-guide`
- `repl-usage`
- `async-patterns`
