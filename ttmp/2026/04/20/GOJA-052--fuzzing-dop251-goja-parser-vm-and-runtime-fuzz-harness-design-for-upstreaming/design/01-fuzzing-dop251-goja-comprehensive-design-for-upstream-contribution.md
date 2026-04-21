---
Title: 'Fuzzing dop251/goja: Comprehensive Design for Upstream Contribution'
Ticket: GOJA-052
Status: active
Topics:
    - fuzzing
    - goja
    - parser
    - upstream
    - security
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: fuzz/fuzz_rewrite_isolated_test.go
      Note: Harness that found BUG-9
    - Path: pkg/jsparse/analyze.go
      Note: Where we added the defensive recover() for upstream parser panics
ExternalSources: []
Summary: Detailed design and implementation guide for fuzzing the dop251/goja library directly, targeting the parser, compiler, and VM runtime, with the goal of upstreaming the harnesses.
LastUpdated: 2026-04-20T00:00:00Z
WhatFor: Guide an intern to build goja-level fuzz harnesses that can be PR'd upstream to dop251/goja
WhenToUse: When implementing or extending goja fuzzing, or preparing an upstream PR
---


# Fuzzing dop251/goja: Design for Upstream Contribution

## Executive Summary

`dop251/goja` is a pure-Go ECMAScript 5.1+ interpreter used by our go-go-goja project and many others. It has **zero fuzz tests**. Our go-go-goja fuzz infrastructure (GOJA-050) already found two crashes in goja's parser through indirect testing. This document designs a fuzz suite that targets goja directly — parser, compiler, and VM — structured so it can be PR'd upstream to `github.com/dop251/goja`.

**Proof of value**: In 30 seconds of fuzzing our indirect harness (`FuzzRewriteIsolated`), we found a panic in goja's `parser.checkComma` function caused by malformed destructuring syntax (`const[...0( 0`). A dedicated parser harness would find more, faster.

---

## Why Fuzz goja Directly?

### The goja attack surface

goja processes **arbitrary, untrusted JavaScript strings**. The data flows through three distinct stages, each with independent crash risks:

```
┌─────────────────────────────────────────────────────────┐
│  Stage 1: PARSER (parser/)                              │
│  ~4,800 lines                                            │
│                                                          │
│  Input: raw string (user JS source)                     │
│  Output: *ast.Program tree                              │
│                                                          │
│  Key files:                                              │
│    parser.go (268 lines) — ParseFile, ParseFunction     │
│    expression.go (1666 lines) — expressions, bindings   │
│    statement.go (1078 lines) — statements               │
│    lexer.go (1210 lines) — tokenization                 │
│    regexp.go (472 lines) — regex parsing                │
│                                                          │
│  Known crash: checkComma slice bounds (BUG-9)           │
│  Estimated speed: 50,000-100,000 exec/sec                │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│  Stage 2: COMPILER (compiler*.go)                       │
│  ~6,300 lines                                            │
│                                                          │
│  Input: *ast.Program from parser                        │
│  Output: *Program (bytecode)                            │
│                                                          │
│  Key files:                                              │
│    compiler.go (1487 lines) — main compiler             │
│    compiler_expr.go (3654 lines) — expression compiling │
│    compiler_stmt.go (1127 lines) — statement compiling  │
│                                                          │
│  Risk: assertion failures, nil dereferences on          │
│    malformed AST nodes the parser didn't reject         │
│  Estimated speed: 20,000-50,000 exec/sec                │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│  Stage 3: VM RUNTIME (runtime.go + builtins)            │
│  ~3,200 lines in runtime.go + ~15,000 in builtins       │
│                                                          │
│  Input: *Program (compiled bytecode)                    │
│  Output: goja.Value (execution result)                  │
│                                                          │
│  Key entry points:                                       │
│    runtime.go: RunString(str) → parse + compile + exec  │
│    runtime.go: RunProgram(p) → execute compiled program │
│    runtime.go: New(construct, args) → construct objects │
│                                                          │
│  Risk: Go-level panics during execution, infinite       │
│    loops, memory exhaustion, stack overflows,            │
│    race conditions in concurrent access                  │
│  Estimated speed: 1,000-10,000 exec/sec                 │
└─────────────────────────────────────────────────────────┘
```

### Why our GOJA-050 harnesses aren't enough

Our GOJA-050 harnesses test goja **indirectly** — they go through go-go-goja's `replapi` → `replsession` → `engine` → goja. This means:

1. **Slow**: Each invocation creates an `engine.Factory`, `runtimeowner.Runner`, event loop, etc. (~1ms overhead per invocation).
2. **Impure coverage**: The fuzzer can't distinguish goja bugs from go-go-goja bugs in coverage reports.
3. **Can't upstream**: The harnesses depend on go-go-goja, not goja.
4. **Missing compiler coverage**: Our `jsparse.Analyze` calls goja's parser but not the compiler. The compiler is a separate crash surface.

A goja-level harness would be pure `parser.ParseFile(nil, "", source, 0)` — no dependencies, no overhead, PR-ready.

---

## Harness Design

### Harness 1: FuzzParseFile

**Target**: `parser.ParseFile(nil, "", source, 0)`
**File**: `fuzz_test.go` in `github.com/dop251/goja/parser/`
**Input**: Single string
**Speed**: ~50,000-100,000 exec/sec

**Invariant**: `ParseFile` must either return a valid `*ast.Program` or a non-nil `error`. It must **never panic**.

```go
// In dop251/goja/parser/fuzz_test.go
package parser

import "testing"

func FuzzParseFile(f *testing.F) {
    // Seed corpus (see Seed Corpus section below)
    for _, seed := range seeds {
        f.Add(seed)
    }

    f.Fuzz(func(t *testing.T, source string) {
        program, err := ParseFile(nil, "", source, 0)
        // Invariant: never panic (the fuzzer framework catches this)
        // Invariant: either program or err is non-nil, not both
        if program == nil && err == nil {
            t.Fatalf("both program and err are nil for source=%q", truncate(source, 100))
        }
    })
}
```

**Why this is the highest-value target**:
- Pure function — no state, no goroutines, no I/O.
- The parser has the most string manipulation and index arithmetic of any goja component.
- We already have a proven crash (`checkComma` slice bounds).
- The parser has ~4,800 lines of code with 60+ parse functions.

### Harness 2: FuzzParseFileIgnoreRegexErrors

**Target**: `parser.ParseFile(nil, "", source, IgnoreRegExpErrors)`
**Input**: Single string
**Speed**: ~50,000-100,000 exec/sec

Same as Harness 1 but with `IgnoreRegExpErrors` mode. This exercises the regex parser's error-tolerant path, which has different slice arithmetic.

### Harness 3: FuzzCompile

**Target**: `goja.Compile("", source, false)`
**File**: `fuzz_test.go` in `github.com/dop251/goja/` (root package)
**Input**: Single string
**Speed**: ~20,000-50,000 exec/sec

**What it exercises**: Parse → Compile (two stages). The compiler receives the AST from the parser and produces bytecode. If the parser accepts a malformed input that the compiler doesn't handle, this catches it.

```go
// In dop251/goja/fuzz_test.go
package goja

import "testing"

func FuzzCompile(f *testing.F) {
    for _, seed := range compileSeeds {
        f.Add(seed)
    }

    f.Fuzz(func(t *testing.T, source string) {
        _, err := Compile("", source, false)
        // Invariant: never panic
        _ = err
    })
}
```

### Harness 4: FuzzRunString

**Target**: `runtime.RunString(source)`
**File**: `fuzz_test.go` in `github.com/dop251/goja/`
**Input**: Single string
**Speed**: ~1,000-10,000 exec/sec
**Requires timeout**: Must use context or interrupt to prevent infinite loops.

**What it exercises**: Full pipeline — parse → compile → execute. This catches VM-level panics (e.g., nil dereferences during property access, stack overflow from deep recursion).

```go
func FuzzRunString(f *testing.F) {
    for _, seed := range runSeeds {
        f.Add(seed)
    }

    f.Fuzz(func(t *testing.T, source string) {
        vm := New()
        _, err := vm.RunString(source)
        _ = err
    })
}
```

**Key design decision**: Each invocation creates a fresh `Runtime`. This is expensive (~0.5ms) but provides perfect isolation. For overnight runs, this is fine. For fast CI, we may need a dedicated faster approach.

**Infinite loop handling**: goja supports `vm.Interrupt(reason)` from another goroutine. The harness should set a 5-second timeout and interrupt:

```go
func FuzzRunString(f *testing.F) {
    // ...
    f.Fuzz(func(t *testing.T, source string) {
        vm := New()
        done := make(chan struct{})
        go func() {
            select {
            case <-done:
            case <-time.After(5 * time.Second):
                vm.Interrupt("timeout")
            }
        }()
        _, _ = vm.RunString(source)
        close(done)
    })
}
```

### Harness 5: FuzzParseFunction

**Target**: `parser.ParseFunction(params, body)`
**Input**: Two strings (parameter list, function body)
**Speed**: ~50,000 exec/sec

**What it exercises**: The `ParseFunction` entry point wraps input in `(function(params) { body })`. This exercises the parser's function-specific paths (parameter destructuring, default values, rest parameters).

### Harness 6: FuzzRunStringStrict

**Target**: `goja.Compile("", source, true)` → `vm.RunProgram(prog)`
**Input**: Single string
**Speed**: ~1,000-10,000 exec/sec

Same as Harness 4 but in strict mode. Exercises different parser/compiler paths (e.g., `with` statement is forbidden, duplicate parameter names are errors).

---

## Seed Corpus Design

### Source: Existing goja test data

goja already has extensive test data in its repository. The most valuable seeds come from:

1. **`parser/parser_test.go`** (1334 lines) — contains hundreds of parse test cases with valid and invalid JS snippets. These should be extracted into the seed corpus.
2. **`compiler_test.go`** (6000+ lines) — contains JS code snippets that exercise specific compiler paths.
3. **Test262 fixtures** — goja may have ECMAScript test262 fixtures that exercise spec edge cases.

### Manual seed categories

In addition to extracting existing tests, these categories target known-fragile areas:

#### Parser: Destructuring (known crash area)

```go
seeds := []string{
    "const [] = []",
    "const {} = {}",
    "const [a] = [1]",
    "const {a} = {a:1}",
    "const [a, ...b] = [1]",
    "const [...a] = [1]",
    // Malformed — these should error, not panic
    "const[",
    "const[...",
    "const[...0",
    "const[...0(",
    "const[...0( 0",   // The input that triggered BUG-9
    "let[...a",
    "var[...a(",
    "const{...a",
    "const{a:",
    "const{a:...b",
}
```

#### Parser: Regular expressions

```go
seeds := []string{
    "/regex/",
    "/regex/gi",
    "/[a-z]+/",
    "/\\d{1,3}/",
    "/(?=lookahead)/",
    "/a|b/",
    "/(group)/",
    "/[\\s\\S]/",
    "/\\u{10FFFF}/",  // Large unicode escape
    "/[\\x00-\\xff]/",
    // Edge cases
    "/a/giu",
    "/./s",
}
```

#### Parser: Template literals

```go
seeds := []string{
    "`hello`",
    "`hello ${name}`",
    "tag`hello`",
    "`${1 + 2}`",
    "`a${`b${c}`}`",  // Nested
    "String.raw`\\d+`",
}
```

#### Parser: Class syntax

```go
seeds := []string{
    "class A {}",
    "class A extends B {}",
    "class A { constructor() {} }",
    "class A { get x() { return 1 } }",
    "class A { static m() {} }",
    "class A { #private = 1 }",
    "class A { #method() {} }",
    // Malformed
    "class {",
    "class extends",
    "class A extends {",
}
```

#### Parser: Async/generators

```go
seeds := []string{
    "async function f() {}",
    "function* g() { yield 1 }",
    "async function* ag() { yield 1 }",
    "const a = async () => {}",
    "const g = function*() { yield 1 }",
    "await 1",
    "yield 1",
}
```

#### Parser: Expressions and operators

```go
seeds := []string{
    "1 + 2 * 3",
    "a ?? b",
    "a?.b?.c",
    "a?.[0]",
    "a?.()",
    "**",            // Exponentiation
    "a **= 2",       // Exponentiation assignment
    "...a",           // Spread standalone
    "({a: 1, ...b})", // Spread in object
    "[...a, ...b]",   // Spread in array
}
```

#### Compiler/VM edge cases

```go
seeds := []string{
    // Proxy trap handling
    "new Proxy({}, { get: () => { throw 1 } }).x",
    // with statement
    "with({a:1}) { a }",
    // eval
    "eval('1+1')",
    // Arguments object edge cases
    "(function() { return arguments })(1,2,3)",
    // Prototype chain manipulation
    "Object.create(null)",
    // Symbol edge cases
    "Symbol.iterator",
    "Symbol.toPrimitive",
    // WeakRef/FinalizationRegistry
    "new WeakRef({})",
}
```

---

## Invariant Checks

Beyond "doesn't panic", each harness should verify structural invariants:

### Parser invariants

```go
// After parsing:
program, err := ParseFile(nil, "", source, 0)

// Invariant 1: Exactly one of (program, err) is non-nil
if program == nil && err == nil { ... }
if program != nil && err != nil { ... }

// Invariant 2: If program is non-nil, it has valid structure
if program != nil {
    // All node Idx values should be within source bounds
    checkNodeBounds(t, program, source)
}

// Invariant 3: Idempotency — parsing twice produces the same result
program2, err2 := ParseFile(nil, "", source, 0)
// program and program2 should be structurally identical
```

### Compiler invariants

```go
// After compiling:
prog, err := Compile("", source, false)

// Invariant: compiled program can be executed without panic
if prog != nil {
    vm := New()
    defer vm.Close()
    _, execErr := vm.RunProgram(prog)
    _ = execErr  // Errors are OK, panics are not
}
```

### VM invariants

```go
// After execution:
vm := New()
value, err := vm.RunString(source)

// Invariant: returned value is always a valid goja.Value (or nil for undefined)
// Invariant: global object is still in a consistent state
_ = vm.GlobalObject().Keys()  // Should not panic

// Invariant: the runtime can accept further evaluations
_, err2 := vm.RunString("1 + 1")
if err2 != nil { ... }
```

---

## File Structure for Upstream PR

The harnesses should live directly in goja's package directories, following Go convention:

```
dop251/goja/
├── parser/
│   ├── fuzz_test.go              # FuzzParseFile, FuzzParseFileIgnoreRegexErrors
│   └── testdata/
│       └── fuzz/
│           └── FuzzParseFile/    # Crash regressions (auto-populated)
├── fuzz_test.go                  # FuzzCompile, FuzzRunString, FuzzRunStringStrict
└── testdata/
    └── fuzz/
        └── FuzzCompile/          # Crash regressions
```

### Why in the upstream repo?

1. **CI integration**: goja uses GitHub Actions. Fuzz tests run as `go test` with a time budget.
2. **Regression corpus**: Crash inputs live in `testdata/fuzz/` and are replayed on every `go test ./...` run.
3. **No dependencies**: The harnesses only import `testing` and goja's own packages.
4. **Community benefit**: Every goja user gets the regression tests for free.

---

## Proposed Makefile Addition (for goja repo)

```makefile
FUZZTIME ?= 30s

fuzz-parser: ## Fuzz the parser
	go test ./parser/ -fuzz=FuzzParseFile -fuzztime=$(FUZZTIME) -v -count=1

fuzz-compile: ## Fuzz parse + compile
	go test ./ -fuzz=FuzzCompile -fuzztime=$(FUZZTIME) -v -count=1

fuzz-vm: ## Fuzz parse + compile + execute
	go test ./ -fuzz=FuzzRunString -fuzztime=$(FUZZTIME) -v -count=1

fuzz-vm-strict: ## Fuzz in strict mode
	go test ./ -fuzz=FuzzRunStringStrict -fuzztime=$(FUZZTIME) -v -count=1

fuzz: fuzz-parser fuzz-compile fuzz-vm fuzz-vm-strict ## Run all fuzz targets
```

---

## Evidence: Bugs We Expect to Find

Based on our experience with GOJA-050 and analysis of goja's source code, these are the highest-probability bug areas:

### High probability

1. **`checkComma` slice bounds** (CONFIRMED — BUG-9): `parser/expression.go:1418` does `self.str[int(from)-self.base:int(to)-self.base]` without checking `from <= to`. Triggered by malformed destructuring.

2. **`reinterpretAs*` functions** (3 functions): `expression.go` has `reinterpretAsArrayAssignmentPattern`, `reinterpretAsArrayBindingPattern`, and `reinterpretAsObjectAssignmentPattern`. These reinterpret AST nodes that were originally parsed as expressions into binding patterns. The reinterpretation assumes specific AST shapes — malformed input could violate those assumptions.

3. **Template literal parsing with nested expressions**: `parseTemplateLiteral` handles backtick strings with `${expr}` interpolations. Deeply nested or malformed templates could stress the lexer's state machine.

### Medium probability

4. **Regex character class parsing**: `regexp.go` (472 lines) handles character classes like `[\s\S]`, `\p{L}`, and unicode property escapes. Edge cases in character class ranges could cause panics.

5. **`parseObjectPropertyKey` with computed keys**: Computed property keys like `{[expr]: value}` involve re-parsing expressions within property positions.

6. **Compiler assertion failures**: The compiler assumes the parser produces well-formed AST. If the parser accepts a syntactically valid but semantically unusual input, the compiler's assertions may fail.

### Lower probability but high impact

7. **VM stack overflow from deep recursion**: `RunString("function f(){f()} f()")` — the VM should handle stack overflow gracefully, not crash.

8. **`Proxy` trap handling**: Complex proxy trap combinations (get → has → getPrototypeOf) could cause nil dereferences.

9. **Concurrent `RunString` on the same runtime**: The VM is not thread-safe. Calling `RunString` from two goroutines should error, not corrupt memory.

---

## Implementation Plan

### Phase 1: Parser harness (1 day)

1. Fork `dop251/goja`.
2. Extract seed corpus from `parser/parser_test.go` test cases.
3. Create `parser/fuzz_test.go` with `FuzzParseFile` and `FuzzParseFileIgnoreRegexErrors`.
4. Run for 1 hour, collect crashes.
5. Fix crashes in goja's parser.
6. Commit crash reproducers to `testdata/fuzz/`.

### Phase 2: Compiler harness (1 day)

1. Create `fuzz_test.go` in root package with `FuzzCompile`.
2. Add compiler-specific seeds (classes, async/generators, destructuring).
3. Run for 1 hour.
4. Fix compiler panics.

### Phase 3: VM harness (2 days)

1. Create `FuzzRunString` and `FuzzRunStringStrict`.
2. Add timeout/interrupt handling for infinite loops.
3. Run for 2-4 hours overnight.
4. Categorize findings: parser bugs vs compiler bugs vs VM bugs.

### Phase 4: Upstream PR (1 day)

1. Clean up harness code.
2. Write PR description with evidence (crash count, categories).
3. Submit to `dop251/goja`.

---

## Relationship to GOJA-050

| Aspect | GOJA-050 (our project) | GOJA-052 (upstream goja) |
|--------|----------------------|-------------------------|
| **Scope** | go-go-goja replapi | dop251/goja core |
| **Dependencies** | Full go-go-goja stack | Only goja |
| **Speed** | 200-5,000 exec/sec | 1,000-100,000 exec/sec |
| **Upstreamable** | No | Yes |
| **Finds bugs in** | go-go-goja + goja | goja only |
| **Test location** | `go-go-goja/fuzz/` | `dop251/goja/parser/`, `dop251/goja/` |

GOJA-050 should **keep** its `FuzzRewriteIsolated` harness as a regression test for BUG-9, even after GOJA-052 lands upstream. The defensive `recover()` in `jsparse.Analyze` should also stay — it protects against unfixed upstream bugs.
