---
Title: Tree-sitter + AST live SEXP editor analysis
Ticket: GOJA-001-AST-PARSE-EDITOR
Status: active
Topics:
    - goja
    - analysis
    - tooling
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/inspector/app/drawer.go
      Note: Current live editor buffer and CST rendering path
    - Path: go-go-goja/cmd/inspector/app/model.go
      Note: Current pane layout and drawer parse event loop
    - Path: go-go-goja/cmd/inspector/main.go
      Note: Current static parse/bootstrap flow
    - Path: go-go-goja/pkg/doc/05-jsparse-framework-reference.md
      Note: Declared reusable boundary between jsparse and inspector UI
    - Path: go-go-goja/pkg/jsparse/analyze.go
      Note: Reusable analyze bundle and parse diagnostics entrypoint
    - Path: go-go-goja/pkg/jsparse/completion.go
      Note: CST completion context extraction and ERROR-node handling
    - Path: go-go-goja/pkg/jsparse/index.go
      Note: AST indexing and deterministic child ordering for AST SEXP
    - Path: go-go-goja/pkg/jsparse/index_test.go
      Note: Evidence for partial AST behavior on parse errors
    - Path: go-go-goja/pkg/jsparse/resolve.go
      Note: Scope and binding resolution for live semantic context
    - Path: go-go-goja/pkg/jsparse/sexp.go
      Note: Implemented SEXP renderer APIs proposed in this analysis
    - Path: go-go-goja/pkg/jsparse/sexp_test.go
      Note: Validation coverage for newly implemented SEXP renderer APIs
    - Path: go-go-goja/pkg/jsparse/treesitter.go
      Note: TSParser and TSNode snapshot data model
    - Path: go-go-goja/pkg/jsparse/treesitter_test.go
      Note: Evidence for tree-sitter error recovery behavior
ExternalSources: []
Summary: Detailed analysis of the current go-go-goja tree-sitter + goja parser setup, and implementation blueprint for a 3-pane live editor showing CST SEXP and AST SEXP (when parse-valid).
LastUpdated: 2026-02-13T15:56:00-05:00
WhatFor: Plan and de-risk implementation of a live AST parse editor with LISP S-expression output.
WhenToUse: Use when implementing or reviewing GOJA-001-AST-PARSE-EDITOR.
---



# Analysis: Live SEXP Editor (tree-sitter + goja AST)

## Goal

Design a new live editor experience in `go-go-goja` with:

1. left pane: editable JavaScript source
2. middle pane: live tree-sitter CST as LISP S-expression
3. right pane: live goja AST as LISP S-expression, only when parse-valid

while reusing as much of the existing `pkg/jsparse` + `cmd/inspector` stack as possible.

## Current Architecture in `go-go-goja`

### Startup parse path (static file)

- `cmd/inspector/main.go:35` parses file text once with `parser.ParseFile`.
- `cmd/inspector/main.go:40` builds index (`jsparse.BuildIndex`) and scope resolution (`jsparse.Resolve`) once.
- `cmd/inspector/main.go:45` passes static data into the Bubble Tea model.

Consequence: current AST pane is file-static; it does not follow live typing.

### `pkg/jsparse` reusable parse stack

- `pkg/jsparse/analyze.go:31` provides `Analyze(filename, source, opts)` and returns `Program`, `ParseErr`, `Index`, `Resolution`.
- `pkg/jsparse/treesitter.go:10` defines `TSNode` snapshot nodes; `Parse` returns an owning tree (`pkg/jsparse/treesitter.go:46`).
- `pkg/jsparse/index.go:31` builds AST index via reflection walk (`walkNode`), including line/column and source spans.
- `pkg/jsparse/resolve.go:110` resolves lexical scopes/bindings in two passes.
- `pkg/jsparse/completion.go:61` derives completion context from CST and `ERROR` nodes.

### Current live typing path (drawer only)

- Drawer text edits happen in `cmd/inspector/app/drawer.go` (`InsertChar`, `DeleteBack`, `InsertNewline`).
- Every edit calls `Drawer.Reparse()` (`cmd/inspector/app/drawer.go:63`), which reparses tree-sitter snapshot.
- `model.handleDrawerKey` triggers reparse and completion updates (`cmd/inspector/app/model.go:277` onward).
- Drawer right-half currently renders flattened CST labels, not SEXP (`cmd/inspector/app/drawer.go:325`, `flattenTSNode` at `361`).

Important note: `TSParser.Parse` is full reparse, not incremental edit-tree reuse (`pkg/jsparse/treesitter.go:44-53`), despite some historical comments/tests naming this as incremental.

### Validation evidence (current behavior)

- `GOWORK=off go test ./pkg/jsparse -count=1` passed.
- `GOWORK=off go test ./cmd/inspector/... -count=1` passed.
- `pkg/jsparse/treesitter_test.go:71` confirms error-recovery shape for `a.` (`ERROR` node includes `identifier` + `.`).
- `pkg/jsparse/index_test.go:295` confirms goja parser can return partial AST on parse errors.
- `pkg/jsparse/analyze_test.go:26` confirms diagnostics are produced from parse errors.
- Focused experiment (`GOWORK=off go test ./pkg/jsparse -run 'TestTSParserErrorRecovery|TestBuildIndexWithParseError' -v -count=1`) showed:
  - `Partial AST: 18 nodes` and `BadExpression` present for invalid goja parse path.
  - tree-sitter `ERROR` node kept both identifier and dot (`foundIdent=true`, `foundDot=true`).

## Gap Analysis vs Requested UX

Requested: editor-left + CST-SEXP + AST-SEXP(valid only).

Current:

- Left pane is read-only source, not editor (`cmd/inspector/app/model.go:903`).
- Live editor exists only inside optional bottom drawer (`cmd/inspector/app/model.go:839`, `865`).
- CST view is tree-style labels, not SEXP (`cmd/inspector/app/drawer.go:346`).
- AST view is static file AST tree labels (`cmd/inspector/app/model.go:1039`), not live SEXP.
- AST parse status is tied to initial file parse (`m.parseErr` in `renderTreePane`, `cmd/inspector/app/model.go:1060`), not the live edited text.

## SEXP Design

## 1) CST SEXP (tree-sitter)

Input: `*jsparse.TSNode`.

Recommended deterministic format:

- node: `(kind <children...>)`
- leaf with text: `(identifier "foo")`
- optional metadata (when enabled): `:range`, `:error`, `:missing`

Example for broken input `obj.`:

```lisp
(program
  (expression_statement
    (ERROR :error true
      (identifier "obj")
      (. "."))))
```

Key implementation detail: keep child order as `TSNode.Children` for stable output.

## 2) AST SEXP (goja)

Input: `*ast.Program` plus source string.

Validity rule (to match user request):

- AST SEXP pane renders only when `parseErr == nil` and `program != nil`.
- if invalid, pane shows parse error summary (first error line) and no AST SEXP.

Recommended AST SEXP shape (index-backed):

- derive structure from `Index`/`NodeRecord` tree (`pkg/jsparse/index.go`).
- node form: `(Kind :span (start end) :label "...")`

Example:

```lisp
(Program
  (LexicalDeclaration :label "const"
    (VariableExpression
      (Identifier :label "\"x\"")
      (NumberLiteral :label "1"))))
```

Rationale: this avoids duplicating AST traversal logic because `BuildIndex` already provides ordered children and spans.

## 3) Shared rendering options

Introduce one options struct for both renderers:

```go
type SExprOptions struct {
    IncludeSpan bool
    IncludeText bool
    IncludeFlags bool // error/missing
    MaxDepth int      // guard huge trees
    MaxNodes int      // guard huge outputs
    Compact bool
}
```

## Implementation Blueprint

## Phase 1: Add reusable SEXP renderers in `pkg/jsparse`

Create:

- `pkg/jsparse/sexp.go`
- `pkg/jsparse/sexp_test.go`

Add APIs:

- `func CSTToSExpr(root *TSNode, opts *SExprOptions) string`
- `func ASTIndexToSExpr(idx *Index, opts *SExprOptions) string`
- optional convenience: `func ASTToSExpr(program *ast.Program, src string, opts *SExprOptions) string`

Why this location: keeps rendering reusable and independent of Bubble Tea UI (consistent with `pkg/doc/05-jsparse-framework-reference.md:20-45`).

## Phase 2: Build a dedicated 3-pane editor command

Recommended to avoid destabilizing current inspector UX:

- new command: `cmd/ast-parse-editor/main.go`
- new app package: `cmd/ast-parse-editor/app/*`

Reuse from existing inspector:

- editor mechanics from drawer (`cmd/inspector/app/drawer.go`)
- parser + analysis from `pkg/jsparse`

Do not mutate existing `cmd/inspector` behavior in first pass.

## Phase 3: Live parse loop (while typing)

Per keypress:

1. update editor buffer
2. parse tree-sitter (`TSParser.Parse`) and regenerate CST SEXP synchronously
3. trigger AST parse command (debounced 50-100ms) for responsiveness
4. when AST parse result returns:
   - if valid: rebuild index/resolution + regenerate AST SEXP
   - if invalid: clear AST SEXP and show parse error banner

Bubble Tea message sketch:

```go
type astParsedMsg struct {
    Seq uint64
    Program *ast.Program
    ParseErr error
    Index *jsparse.Index
    ASTSExpr string
}
```

Use sequence numbers to drop stale async results.

## Phase 4: 3-pane rendering

Replace current 2-pane content section with 3 equal columns:

- `EDITOR`
- `TS SEXP`
- `AST SEXP`

Each pane needs independent vertical scrolling state because SEXP output can exceed screen height.

## File-by-File Change Map

Existing files used as integration anchors:

- `go-go-goja/pkg/jsparse/treesitter.go`: CST input model.
- `go-go-goja/pkg/jsparse/index.go`: AST tree model for deterministic SEXP output.
- `go-go-goja/pkg/jsparse/analyze.go`: parse/index/resolution bundle.
- `go-go-goja/cmd/inspector/app/drawer.go`: editable text buffer operations.
- `go-go-goja/cmd/inspector/app/model.go`: Bubble Tea layout/key handling reference.
- `go-go-goja/cmd/inspector/main.go`: current parse/bootstrap flow.

New files proposed:

- `go-go-goja/pkg/jsparse/sexp.go`
- `go-go-goja/pkg/jsparse/sexp_test.go`
- `go-go-goja/cmd/ast-parse-editor/main.go`
- `go-go-goja/cmd/ast-parse-editor/app/model.go`
- `go-go-goja/cmd/ast-parse-editor/app/model_test.go`

## Risks and Mitigations

1. Output volume: SEXP can explode for large files.
- Mitigation: `MaxDepth`, `MaxNodes`, pane scrolling, and truncation indicator.

2. UI stalls from per-keystroke goja parse.
- Mitigation: debounce + async parse commands.

3. Non-ASCII cursor mismatch.
- Existing editor tracks rune columns; tree-sitter columns are byte-based positions.
- Mitigation: add rune<->byte conversion helpers before `NodeAtPosition` and source span mapping.

4. Ambiguous AST validity.
- goja can return partial AST with parse error (`index_test.go:295`).
- Mitigation: treat AST pane as valid only when `parseErr == nil`.

## Test Plan

Unit tests:

- `pkg/jsparse/sexp_test.go`
  - CST leaf escaping
  - ERROR/MISSING marker emission
  - deterministic ordering
  - depth/node truncation
- AST SEXP tests from valid and invalid snippets

Integration tests:

- `cmd/ast-parse-editor/app/model_test.go`
  - typing updates CST SEXP every edit
  - AST SEXP appears on valid text
  - AST pane shows error state on invalid text
  - stale async parse results ignored by sequence check

Regression checks (existing):

- `GOWORK=off go test ./pkg/jsparse -count=1`
- `GOWORK=off go test ./cmd/inspector/... -count=1`

## Recommendation

Implement as a new command (`cmd/ast-parse-editor`) with reusable SEXP code in `pkg/jsparse`, then optionally converge UX into `cmd/inspector` after behavior is stable.

This gives fast delivery for requested UX and avoids regressions in the current inspector workflow.
