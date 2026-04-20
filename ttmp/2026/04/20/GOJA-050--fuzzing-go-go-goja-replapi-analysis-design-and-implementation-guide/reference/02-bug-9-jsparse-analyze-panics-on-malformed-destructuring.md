---
Title: 'BUG-9: jsparse.Analyze panics on malformed destructuring'
Ticket: GOJA-050
Status: active
Topics:
    - fuzzing
    - replapi
    - testing
    - security
    - goja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "jsparse.Analyze panics on malformed array destructuring patterns instead of returning a parse error"
LastUpdated: 2026-04-20
WhatFor: "Bug report for upstream goja parser panic triggered by fuzz testing"
WhenToUse: "Reference when fixing or tracking the jsparse panic bug"
---

# BUG-9: jsparse.Analyze panics on malformed destructuring

## Severity
**High** — any code path that calls `jsparse.Analyze` on user-supplied input (REPL evaluate, inspector, completion) will crash the process.

## Summary

`jsparse.Analyze` panics with `slice bounds out of range [14:13]` when parsing malformed array destructuring syntax. The panic originates in goja's upstream parser (`dop251/goja`), not in our resolve code.

## Reproduction

```go
analysis := jsparse.Analyze("<test>", "const[...0( 0", nil)
// PANIC: runtime error: slice bounds out of range [14:13]
```

Crash reproducer: `fuzz/testdata/fuzz/FuzzRewriteIsolated/4695462aa1e8927e`

Additional inputs that likely trigger the same path:
- `const[...0( 0` (minimal reproducer)
- Any `const[` or `let[` or `var[` followed by spread/rest with malformed RHS

## Stack Trace

```
runtime error: slice bounds out of range [14:13]
github.com/dop251/goja/parser.(*_parser).checkComma
    parser/expression.go:1418
github.com/dop251/goja/parser.(*_parser).reinterpretAsArrayBindingPattern
    parser/expression.go:1466
github.com/dop251/goja/parser.(*_parser).parseArrayBindingPattern
    parser/expression.go:1482
github.com/dop251/goja/parser.(*_parser).parseBindingTarget
    parser/expression.go:294
github.com/dop251/goja/parser.(*_parser).parseVariableDeclaration
    parser/expression.go:308
github.com/dop251/goja/parser.(*_parser).parseVariableDeclarationList
    parser/expression.go:325
github.com/dop251/goja/parser.(*_parser).parseLexicalDeclaration
    parser/statement.go:802
```

## Root Cause

The upstream goja parser's `checkComma` function in `expression.go:1418` performs a slice operation with bounds derived from the AST node positions. When the input is malformed (e.g., `const[...0( 0`), the AST positions are inconsistent, producing `start > end` which causes the slice to panic.

This is an upstream bug in `dop251/goja`. The correct fix would be in goja's parser to guard the slice bounds.

## Impact

Every code path that parses user-supplied JavaScript is affected:
- `replsession.Evaluate` (REPL evaluation)
- `jsparse.Analyze` (static analysis)
- Inspector completion and navigation
- JSDoc extraction

## Fix

**Our fix**: Add a defensive `recover()` in `jsparse.Analyze` to catch upstream parser panics and convert them to `ParseErr`.

**Upstream fix** (future): File an issue on `dop251/goja` and submit a PR to guard the slice bounds in `checkComma`.

## Discovered By

`FuzzRewriteIsolated` harness in the `fuzz/` package, during baseline seed coverage gathering. The mutator generated `"const[...0( 0"` from the seed `"const [...rest] = [1, 2, 3]"`.
