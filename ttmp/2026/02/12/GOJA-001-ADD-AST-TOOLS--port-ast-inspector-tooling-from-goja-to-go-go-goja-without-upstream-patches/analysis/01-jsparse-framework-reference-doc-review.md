---
Title: JSParse Framework Reference Doc Review
Ticket: GOJA-001-ADD-AST-TOOLS
Status: active
Topics:
    - goja
    - analysis
    - tooling
    - migration
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/pkg/doc/05-jsparse-framework-reference.md
      Note: Documentation being reviewed
    - Path: go-go-goja/pkg/jsparse/analyze.go
      Note: Core Analyze API and AnalysisResult
    - Path: go-go-goja/pkg/jsparse/completion.go
      Note: Completion context extraction and candidate resolution
    - Path: go-go-goja/pkg/jsparse/index.go
      Note: Index
    - Path: go-go-goja/pkg/jsparse/noderecord.go
      Note: NodeRecord struct definition
    - Path: go-go-goja/pkg/jsparse/resolve.go
      Note: Scope resolution
    - Path: go-go-goja/pkg/jsparse/treesitter.go
      Note: TSParser
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-12T18:20:49.429486255-05:00
WhatFor: ""
WhenToUse: ""
---


# Documentation Review: 05-jsparse-framework-reference.md

**Ticket:** GOJA-001-ADD-AST-TOOLS  
**Date:** 2026-02-12  
**Scope:** Verification of `pkg/doc/05-jsparse-framework-reference.md` against `pkg/jsparse/` source code  
**Status:** All 30 tests pass; API surface is functional and well-tested

---

## 1. Executive Summary

The documentation is **well-structured and broadly accurate**. It correctly describes the high-level architecture, the reusable/tool-specific boundary, and the three core integration patterns. However, there are **several inaccuracies, gaps, and missed opportunities** that would trip up a newcomer trying to use the framework solely from the docs.

**Verdict:** Good foundation, needs a focused revision pass (estimated ~2h).

---

## 2. Accuracy Audit: Claim-by-Claim Verification

### 2.1 Correct Claims ✅

| Doc Claim | Code Evidence |
|---|---|
| `Analyze` returns `AnalysisResult` with Program, Index, Resolution, ParseErr | `analyze.go:36-57` — exact match |
| `Diagnostics()` normalizes parse errors | `analyze.go:60-67` — returns `[]Diagnostic` with severity "error" |
| `NodeAtOffset` returns most specific node | `index.go:123-134` — linear scan, picks smallest span |
| Index has `NodeAtOffset`, `VisibleNodes`, `ExpandTo`, `AncestorPath` | All present in `index.go` |
| Resolution has `Scopes`, `RootScopeID`, `NodeBinding`, `Unresolved` | All present in `resolve.go:87-93` |
| CompletionCandidate has Label, Kind, Detail | `completion.go:36-40` — exact match |
| API is pure-Go and command-agnostic | No UI imports in `pkg/jsparse/` |
| Parser errors don't prevent index/resolution when partial AST available | `analyze.go:46-52` — guarded by `if program != nil` |
| Completion tolerates partial/error CST states | `completion.go:57-98` — cursor-1 fallback + ERROR node handling |

### 2.2 Inaccuracies / Misleading Claims ⚠️

#### 2.2.1 `CompletionCandidate.Kind` type mismatch

**Doc says:**
> `Kind` (`property`, `method`, `variable`, ...)

**Code has:**
```go
type CandidateKind int
const (
    CandidateProperty CandidateKind = iota
    CandidateMethod
    CandidateVariable
    CandidateFunction
    CandidateKeyword
)
```

The doc implies string values. The actual type is an `int` enum (`CandidateKind`). Additionally, `CandidateFunction` and `CandidateKeyword` are undocumented.

**Fix:** Replace with: `Kind` (`CandidateProperty`, `CandidateMethod`, `CandidateVariable`, `CandidateFunction`, `CandidateKeyword`)

#### 2.2.2 `ExtractCompletionContext` and `ResolveCandidates` are not the primary API

**Doc says (§ Core API Surface, item 3):**
```go
ctx := result.CompletionContextAt(root, row, col)
candidates := result.CompleteAt(root, row, col)
```

This is correct, but it's the convenience wrapper. The doc also mentions `ExtractCompletionContext` and `ResolveCandidates` in the "What Is Reusable" list without clarifying these are the lower-level functions that `CompletionContextAt` and `CompleteAt` delegate to. A reader may wonder about the relationship.

**Fix:** Add a brief note: "The convenience methods delegate to the lower-level `ExtractCompletionContext` and `ResolveCandidates`, which can be used directly for custom pipelines."

#### 2.2.3 `NodeRecord` fields incomplete

**Doc says:**
> `NodeRecord` — `Kind`, `Label`, `Start`, `End`

**Code has** (in `noderecord.go`):
```go
type NodeRecord struct {
    ID, Kind, Start, End, StartLine, StartCol, EndLine, EndCol,
    Label, Snippet, ParentID, ChildIDs, Depth, Expanded
}
```

The doc omits `ID`, `Snippet`, `StartLine/StartCol/EndLine/EndCol`, `ParentID`, `ChildIDs`, `Depth`, and `Expanded` — several of which are essential for any tool integrator.

**Fix:** Either list all fields or add a "see godoc" note. At minimum document `ID`, `ParentID`, `ChildIDs`, `Depth`, and line/col fields.

#### 2.2.4 `Analyze` determinism claim is aspirational

**Doc says:**
> `Analyze` is deterministic for a given source/options.

**Code reality:** `Analyze` uses `parser.ParseFile` from goja, which is deterministic for parsing. However, `BuildIndex` uses `reflect`-based child discovery (`childNodes`) and map iteration (`map[NodeID]*NodeRecord`). The `NodeID` assignment is sequential and deterministic (walk order), and `OrderedByStart` is sorted, so the claim holds. But `NodeAtOffset` iterates `OrderedByStart` linearly and picks smallest span with a depth tiebreaker — if two nodes have identical `[Start, End)` and `Depth`, the first encountered wins, which is deterministic only because `OrderedByStart` sort is stable on those criteria.

**Verdict:** Technically true but fragile. Worth noting in the doc as a design invariant that must be preserved.

### 2.3 Missing from Documentation 🕳️

#### 2.3.1 Scope resolution model

The doc describes `Resolution` at a very high level (scope graph, declaration-to-reference links, unresolved list) but omits:

- **ScopeKind enum** (`ScopeGlobal`, `ScopeFunction`, `ScopeBlock`, `ScopeCatch`, `ScopeFor`) — critical for understanding scope chain behavior
- **BindingKind enum** (`BindingVar`, `BindingLet`, `BindingConst`, `BindingFunction`, `BindingClass`, `BindingParameter`, `BindingCatchParam`) — needed to interpret bindings
- **Hoisting semantics** — `var` and `function` hoist to nearest function/global scope; `let`/`const` don't. This is implemented but undocumented.
- **Helper methods** on `Resolution`: `BindingForNode()`, `IsDeclaration()`, `IsReference()`, `IsUnresolved()`
- **`AllUsages()`** on `BindingRecord` — returns decl + all references

These are the most powerful parts of the API for building dev tools, and they're invisible in the reference.

#### 2.3.2 TSParser lifecycle

The doc shows:
```go
tsParser, _ := jsparse.NewTSParser()
defer tsParser.Close()
root := tsParser.Parse([]byte(source))
```

But does not mention:
- `TSParser` wraps `tree-sitter/go-tree-sitter` (Go binding for tree-sitter)
- `Parse()` returns a **snapshot** (`*TSNode`) that survives parser reparse/close — this is a key design choice
- `TSNode.HasError()`, `TSNode.ChildCount()`, `TSNode.NodeAtPosition()` are available
- `TSNode` is a lightweight owning tree distinct from `tree_sitter.Node`
- No incremental parse (explicitly by design — source is small)

#### 2.3.3 Index tree navigation

The doc mentions `VisibleNodes`, `ExpandTo`, `AncestorPath` but omits:
- `ToggleExpand(id)` — toggle expand/collapse state
- `LineColToOffset(line, col)` and `OffsetToLineCol(offset)` — bidirectional position conversion
- `Source()` — access to the original source string
- The auto-expand behavior: `depth < 2` is auto-expanded on index build

#### 2.3.4 Drawer-local completion

`ExtractDrawerBindings(root *TSNode)` and the drawer-local integration in `ResolveCandidates` are not documented at all. This is used to provide completion for variables in an interactive drawer/REPL context. The variadic `drawerRoot ...*TSNode` parameter on `ResolveCandidates` is a key integration point.

#### 2.3.5 Built-in prototype knowledge

`completion.go` contains a hardcoded `builtinPrototypes` map with `Object`, `Array`, `String`, `console`, `Math`, `JSON` methods. This is a significant feature (and limitation) not mentioned in the reference.

---

## 3. Structural / Presentation Issues

### 3.1 Code examples are minimal

Only 3 code snippets. For a reference doc, this is thin. Suggestions:
- Add a scope resolution example (find binding for identifier, list usages)
- Add a tree navigation example (walk ancestors, expand to target)
- Add a position conversion example (offset → line:col → offset round-trip)

### 3.2 Troubleshooting table is generic

The 4 rows in the troubleshooting table are too vague to be actionable. For example, "Check syntax first" is not useful advice. Better: show how to extract the exact parse error position from `ParseErr` (it's an `ErrorList` with positions).

### 3.3 "See Also" references are unanchored

The doc ends with:
> - `inspector-example-user-guide`  
> - `repl-usage`  
> - `async-patterns`

These are slug references but don't resolve to file paths. A reader can't navigate to them. Should be relative file links.

### 3.4 Missing package import path

The doc never states the full import path: `github.com/go-go-golems/go-go-goja/pkg/jsparse`. A reference doc should include this.

---

## 4. Suggested Improvements (Prioritized)

### P0 — Must Fix (accuracy)

1. **Fix `CompletionCandidate.Kind` type description** — list the actual `CandidateKind` enum values
2. **Document `NodeRecord` fields completely** — at minimum ID, ParentID, ChildIDs, Depth, line/col
3. **Add scope resolution API section** — `ScopeKind`, `BindingKind`, `BindingForNode()`, `IsDeclaration()`, `IsReference()`, `IsUnresolved()`, `AllUsages()`
4. **State the import path** — `github.com/go-go-golems/go-go-goja/pkg/jsparse`

### P1 — Should Fix (completeness)

5. **Document TSNode snapshot design** — explain it's a tree-sitter CST snapshot that outlives the parser
6. **Document built-in prototype knowledge** — list what's in `builtinPrototypes` and how to extend it
7. **Document position conversion helpers** — `LineColToOffset`, `OffsetToLineCol`
8. **Document drawer-local completion** — `ExtractDrawerBindings`, variadic `drawerRoot` on `ResolveCandidates`
9. **Fix "See Also" to use relative paths** — e.g. `./06-inspector-example-user-guide.md`

### P2 — Nice to Have (quality)

10. **Add scope resolution code example** — 5-10 lines showing binding lookup + usages
11. **Add tree navigation code example** — ancestor path, expand-to-target
12. **Improve troubleshooting table** — add concrete error message patterns and recovery steps
13. **Add a "Limitations" section** — property completion is heuristic-only (no type inference), scope resolution covers ES2020+ but not modules/imports, built-in prototypes are hardcoded
14. **Note the determinism invariant** explicitly as a maintained contract

---

## 5. Test Coverage Assessment

| Component | Tests | Verdict |
|---|---|---|
| `Analyze` (happy + error) | 3 tests | ✅ Good |
| `BuildIndex` (simple + complex + error) | 5 tests | ✅ Good |
| `NodeAtOffset` | offset table + sync roundtrip | ✅ Good |
| `AncestorPath` | 1 test | ⚠️ Minimal |
| `VisibleNodes` / `ExpandCollapse` | 2 tests | ✅ OK |
| `LineColToOffset` round-trip | 1 test | ✅ OK |
| `Resolve` (var, let, const, function, param, catch, arrow, dot-exclusion, hoisting) | 10 tests | ✅ Excellent |
| `TSParser` (parse, incremental, error recovery, snapshot, position) | 5 tests | ✅ Good |
| Completion (property dot, partial, identifier, chain, with-index, drawer-local, none) | 7 tests | ✅ Good |
| **Total** | **30 passing** | ✅ Solid |

Notable gaps:
- No test for `ExpandTo` (only `ToggleExpand` is tested)
- No test for `Source()` or `OffsetToLineCol` public method (only the private `offsetToLineCol` is tested via `LineColToOffset` round-trip)
- No test for `CompletionArgument` kind (always returns nil — dead code?)

---

## 6. Architecture Observations

### 6.1 Dual-parser design is sound but could be explained better

The framework uses **two parsers**:
1. **goja parser** (`parser.ParseFile`) → produces `*ast.Program` for semantic analysis (index, scope resolution)
2. **tree-sitter** (`TSParser.Parse`) → produces `*TSNode` CST for error-tolerant completion

This is a deliberate design: goja's parser gives a rich typed AST for binding analysis, while tree-sitter gives error-tolerant concrete syntax trees for completion at arbitrary cursor positions. The doc should explicitly state this dual-parser rationale.

### 6.2 Reflection-based child discovery is a maintenance risk

`childNodes()` in `index.go` uses `reflect` to discover AST children. This works because goja's AST types embed `ast.Node` values as struct fields. But:
- It's O(fields × children) per node
- Any change in goja's AST struct layout could silently change child ordering
- The `sort.SliceStable` by `Idx0()/Idx1()` provides stability, but edge cases with embedded value-type nodes (handled by the `CanAddr()` path) are subtle

This is worth noting as a known design trade-off.

### 6.3 `CompletionArgument` is dead code

`CompletionArgument` is defined in the `CompletionKind` enum but never produced by `ExtractCompletionContext` and returns `nil` from `ResolveCandidates`. Either implement it or remove it.

---

## 7. Conclusion

The `05-jsparse-framework-reference.md` document provides a solid introduction to the framework but falls short as a **reference** document. The most critical gap is the complete absence of scope resolution API documentation (§2.3.1), which is arguably the most sophisticated part of the framework. The P0 fixes should be applied before the doc is used as an integration guide.

The code itself is well-organized, well-tested (30/30 passing), and the separation between reusable framework (`pkg/jsparse`) and tool-specific code is clean. The dual-parser architecture is a pragmatic choice that works well for the stated use cases.

## 8. Resolution Update (2026-02-12)

Review items were addressed in `pkg/doc/05-jsparse-framework-reference.md`:

- P0 addressed:
  - `CompletionCandidate.Kind` now documents actual `CandidateKind` enum values.
  - `NodeRecord` field documentation expanded (identity, span, line/col, hierarchy, expanded state).
  - Scope/binding model documented (`ScopeKind`, `BindingKind`, hoisting, helper methods, `AllUsages`).
  - Full import path added (`github.com/go-go-golems/go-go-goja/pkg/jsparse`).
- P1 addressed:
  - TS parser/snapshot lifecycle documented.
  - built-in prototype knowledge documented.
  - position conversion helpers documented.
  - drawer-local completion and `ResolveCandidates(..., drawerRoot)` documented.
  - see-also entries now include concrete relative file references.
- P2 addressed:
  - added scope/binding example and tree navigation/position example.
  - troubleshooting guidance made concrete.
  - limitations section added (heuristics, hardcoded prototypes, missing module graph, reflection trade-off).
  - determinism invariant explicitly documented.
