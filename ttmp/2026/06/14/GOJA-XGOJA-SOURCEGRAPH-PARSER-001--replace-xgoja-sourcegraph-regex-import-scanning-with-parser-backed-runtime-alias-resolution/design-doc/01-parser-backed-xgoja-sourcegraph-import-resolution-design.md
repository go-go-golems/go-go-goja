---
Title: Parser-backed xgoja sourcegraph import resolution design
Ticket: GOJA-XGOJA-SOURCEGRAPH-PARSER-001
Status: active
Topics:
    - xgoja
    - sourcegraph
    - typescript
    - code-generation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/examples/xgoja/17-sourcegraph-runtime-aliases
      Note: Broad xgoja sourcegraph/runtime alias example
    - Path: go-go-goja/pkg/jsparse/treesitter.go
      Note: Existing tree-sitter integration pattern used by experiments
    - Path: go-go-goja/pkg/xgoja/app/command_providers.go
      Note: Command-selected aliases for provider command source registries
    - Path: go-go-goja/pkg/xgoja/app/root.go
      Note: Runtime jsverb source scanning and runtime alias input path
    - Path: go-go-goja/pkg/xgoja/app/source_registry.go
      Note: Runtime alias propagation for source scans
    - Path: go-go-goja/pkg/xgoja/sourcegraph/graph.go
      Note: |-
        Current regex import parsing and sourcegraph resolution target
        Sourcegraph resolution now uses parser-backed import specs and dynamic import diagnostics
    - Path: go-go-goja/pkg/xgoja/sourcegraph/graph_test.go
      Note: Parser-backed sourcegraph regression coverage
    - Path: go-go-goja/pkg/xgoja/sourcegraph/imports.go
      Note: Tree-sitter-backed import collector implementation
    - Path: go-go-goja/ttmp/2026/06/14/GOJA-XGOJA-SOURCEGRAPH-PARSER-001--replace-xgoja-sourcegraph-regex-import-scanning-with-parser-backed-runtime-alias-resolution/scripts/04-treesitter-typescript-experiment.sh
      Note: Self-contained TypeScript/TSX tree-sitter experiment
ExternalSources: []
Summary: Design and experiment notes for replacing regex-based xgoja sourcegraph import scanning with parser-backed resolution and correct RuntimePlan alias propagation.
LastUpdated: 2026-06-14T09:42:16.138610864-04:00
WhatFor: Planning the xgoja sourcegraph import parser/runtime-alias fix.
WhenToUse: Use when implementing parser-backed import graph resolution for xgoja RuntimePlan sources.
---



# Parser-backed xgoja sourcegraph import resolution design

## Executive Summary

xgoja v2 made `RuntimePlan` the authoritative model for runtime modules, source sets, and command source scopes. The current source graph still extracts imports with a regular expression in `pkg/xgoja/sourcegraph/graph.go`, and some scan paths use provider capability aliases rather than the configured `runtime.modules[].as` aliases. That combination is fragile: literal runtime aliases such as `require("fs:assets")` should be accepted when the RuntimePlan selects `as: fs:assets`, while unknown bare imports should still fail early.

The recommended solution is a staged replacement of regex import scanning with a parser-backed collector, plus a first fix that propagates configured command/runtime aliases into every sourcegraph scan. The experiments in this ticket show that Goja's parser is useful for CommonJS `require(...)` only, esbuild's public Build/metafile API robustly handles JS/TS/ESM imports with runtime aliases marked external, and the already-present tree-sitter JavaScript grammar can extract imports without false positives but needs either a TypeScript grammar dependency or error-tolerant handling for TS-specific syntax.

## Problem Statement

Current sourcegraph import extraction is based on:

```go
var importRE = regexp.MustCompile(`(?m)(?:import\s+(?:[^"']+\s+from\s+)?|require\()\s*["']([^"']+)["']`)
```

That regex is not a real JavaScript or TypeScript parser. It can miss valid import forms, can be confused by comments/strings, and cannot classify dynamic imports precisely. The immediate user-visible issue is runtime aliases containing `:` (for example `fs:assets`) in jsverb sources. Those aliases should be validated against selected RuntimePlan modules, not rejected as unknown bare imports and not hidden behind dynamic workarounds such as `require(["fs", "assets"].join(":"))`.

The fix must preserve the xgoja v2 guarantees documented in `GOJA-XGOJA-V2-RUNTIME-001`:

- `RuntimePlan` is authoritative.
- `sources[]` are first-class and command-scoped.
- Provider command sets receive only their command-selected sources.
- Empty/omitted command source scopes stay empty.
- Runtime module aliases come from configured runtime modules, including aliases with punctuation such as `fs:assets`.

## Evidence and Experiments

The ticket scripts live in `scripts/`:

- `01-goja-parser-experiment.go`
- `02-esbuild-import-experiment.go`
- `03-treesitter-import-experiment.go`

### Goja parser experiment

Goja's AST parser accepts CommonJS-style source:

```text
const x = require("fs:assets"); -> <nil>
```

But it rejects ESM and dynamic import syntax:

```text
import assets from "fs:assets";   -> Unexpected reserved word
import "./setup.js";              -> Unexpected reserved word
export { x } from "./x.js";       -> Unexpected reserved word
const x = await import("./x.js"); -> Unexpected reserved word
```

Conclusion: Goja can be a CommonJS collector for literal `require("...")`, but it is not sufficient as the sole sourcegraph parser.

### Esbuild experiment

Esbuild does not expose a public AST parser API; its public Go API exposes `Build`, `Transform`, and `Context`. However, `api.Build` with `Bundle: true`, `Write: false`, and `Metafile: true` can serve as an import collector.

Without configured externals, esbuild rejects runtime aliases and unknown bare imports:

```text
ERR: Could not resolve "fs:assets"
ERR: Could not resolve "express"
```

With aliases marked external:

```go
External: []string{"fs:assets", "fs:host", "express"}
```

it records imports in the metafile:

```text
path="fs:assets" kind=require-call external=true
path="express" kind=require-call external=true
path="./dynamic.js" kind=dynamic-import external=false
```

It also handles TypeScript/ESM import syntax:

```text
path="fs:assets" kind=import-statement external=true
path="./more.js" kind=import-statement external=false
```

Conclusion: esbuild is the strongest single existing dependency for robust JS/TS import collection, even though it is not an AST parser API.

### Tree-sitter JavaScript experiment

The repo already depends on:

```text
github.com/tree-sitter/go-tree-sitter
github.com/tree-sitter/tree-sitter-javascript
```

The experiment used that existing JavaScript grammar directly. It successfully parsed and extracted:

```text
require("fs:assets")           -> fs:assets
import assets from "fs:assets" -> fs:assets
import "./setup.js"            -> ./setup.js
export { x } from "./x.js"     -> ./x.js
await import("./x.js")         -> ./x.js
```

It also avoided regex false positives in strings/comments:

```js
const s = "require('not-real')"; // import "also-not-real"
```

No import edges were emitted for that case.

For TypeScript-ish source, the JavaScript grammar reported `hasError=true` but still recovered enough to extract import/export sources:

```text
./types
./helper
fs:assets
./more
```

### Tree-sitter TypeScript experiment

A fourth experiment used `github.com/tree-sitter/tree-sitter-typescript@v0.23.2` in a temporary module so the repo `go.mod` stayed untouched. The package exposes both `LanguageTypescript()` and `LanguageTSX()`.

The TypeScript grammar parsed TypeScript-specific syntax cleanly:

```text
hasError=false
import type { Thing } from "./types" -> ./types
import { helper } from "./helper"    -> ./helper
import assets from "fs:assets"       -> fs:assets
export { more } from "./more"        -> ./more
await import("./dynamic")            -> ./dynamic
require("fs:host")                   -> fs:host
require(["fs", "assets"].join(":")) -> dynamic=true
```

The TSX grammar also parsed JSX/TSX without errors and extracted:

```text
react
fs:assets
./Widget
```

It also avoided false positives inside TypeScript string literals and comments.

Conclusion: tree-sitter with the TypeScript grammar is now the most attractive parser-backed implementation path. It avoids esbuild's no-write build/metafile indirection, supports JS/ESM/TypeScript/TSX as concrete syntax trees, and already matches the sourcegraph need: extract literal import specifiers and flag dynamic imports. The cost is adding one direct dependency on `tree-sitter-typescript`.

## Proposed Solution

### Phase 1: Fix configured runtime alias propagation

Before changing parsers, make sourcegraph scans use the aliases selected by the active RuntimePlan command/module set.

Do not rely on provider-wide aliases alone:

```go
allProviderRuntimeAliases(providers)
```

Instead, derive aliases from the configured runtime module descriptors selected for the command:

```go
moduleAliases(selectedModules)
```

and ensure aliases such as `fs:assets` flow into:

- planner sourcegraph validation;
- builtin jsverb scans;
- provider command-set scans via `SourceRegistry`;
- hot-reload rescans.

### Phase 2: Introduce an import collector interface

Add a small interface inside `pkg/xgoja/sourcegraph`:

```go
type ImportCollector interface {
    CollectImports(filename string, source []byte) ([]ImportSpec, error)
}

type ImportSpec struct {
    Specifier string
    Kind      ImportSyntaxKind // import-statement, export-from, require-call, dynamic-import
    Dynamic   bool
}
```

`Graph.ResolveImports` should depend on the interface instead of directly calling `parseImports`.

### Phase 3: Implement tree-sitter collector for JS/ESM/CommonJS

Use tree-sitter for `.js`, `.jsx`, `.mjs`, and `.cjs` first. It covers the forms that Goja's parser rejects and avoids regex false positives.

Production behavior should include:

- `import ... from "x"`
- `import "x"`
- `export ... from "x"`
- `require("x")`
- `import("x")`
- diagnostic for `require(expr)` and `import(expr)` when `expr` is not a string literal.

### Phase 4: Add tree-sitter TypeScript/TSX support

Add `github.com/tree-sitter/tree-sitter-typescript` and choose the grammar by file extension:

- `.js`, `.jsx`, `.mjs`, `.cjs` -> `tree-sitter-javascript` or the TSX grammar if it proves sufficient for JSX.
- `.ts`, `.mts`, `.cts` -> `tree-sitter-typescript` `LanguageTypescript()`.
- `.tsx` -> `tree-sitter-typescript` `LanguageTSX()`.

Recommendation: use tree-sitter for TypeScript/TSX sourcegraph parsing. Esbuild remains useful for compilation and as an alternative import-collector design, but the TypeScript tree-sitter experiment showed clean parsing and simpler static import extraction for sourcegraph's needs.

### Phase 5: Remove regex collector after parity tests

Keep the regex parser only as a temporary fallback during implementation. Remove it after tests cover:

- CommonJS require;
- ESM imports;
- side-effect imports;
- export-from imports;
- dynamic imports;
- non-literal dynamic import diagnostics;
- colon runtime aliases;
- unknown bare imports;
- local extension resolution;
- path escape rejection;
- TypeScript imports.

## Design Decisions

### Decision: Sourcegraph remains the authority

Keep import resolution in `pkg/xgoja/sourcegraph`. Parser-backed extraction is an implementation detail. The package should continue to own classification into local imports, runtime aliases, and unknown imports.

### Decision: RuntimePlan aliases are explicit inputs

Parser choice does not fix alias semantics by itself. `fs:assets` only works if the sourcegraph receives `fs:assets` as a valid runtime alias for the command/source scan.

### Decision: Dynamic non-literal imports need diagnostics

A workaround such as:

```js
require(["fs", "assets"].join(":"))
```

hides dependencies from a static source graph. The implementation should warn or fail for non-literal dynamic imports in strict generated-app validation.

## Alternatives Considered

### Keep regex and patch edge cases

Rejected. It may fix `fs:assets`, but it will keep failing on syntax coverage and false positives.

### Use Goja AST parser only

Rejected as a sole solution. It is useful for CommonJS but rejects ESM import/export forms and TypeScript syntax.

### Use esbuild only

Viable. It handles TypeScript and dependency resolution well. The drawback is that it is a build/resolution API rather than a direct parser API, so sourcegraph would need a no-write build/metafile integration and careful custom external/local semantics.

### Use tree-sitter only

Preferred after the TypeScript experiment. With `tree-sitter-typescript`, tree-sitter covers JS, ESM, CommonJS, TypeScript, and TSX import syntax directly. Sourcegraph can extract literal specifiers from CST nodes and classify dynamic imports without invoking a bundler-style resolver.

## Implementation Plan

1. Add failing regression tests for literal `require("fs:assets")` and `import assets from "fs:assets"` using configured RuntimePlan aliases.
2. Fix alias propagation so command/runtime scans use selected configured module aliases.
3. Add `ImportCollector` to `pkg/xgoja/sourcegraph`.
4. Add `tree-sitter-typescript` and implement tree-sitter collectors for JS/ESM/CommonJS/TS/TSX.
5. Add table-driven sourcegraph tests for all import forms and diagnostics.
6. Keep esbuild as a reference experiment and fallback option, not the first implementation path.
7. Update `examples/xgoja/17-sourcegraph-runtime-aliases` to use literal `require("fs:assets")` and ESM/TS import coverage.
8. Remove regex collector after parity tests pass.
9. Update xgoja v2 docs to explain static import requirements and dynamic import diagnostics.

## Open Questions

- Should generated xgoja builds fail on non-literal dynamic imports, or should they warn and ignore them?
- Should TypeScript parsing use esbuild only, or should the repo add `tree-sitter-typescript` for CST parity?
- Should sourcegraph allow node_modules/package imports in a future mode, or continue to reject all unknown bare imports?

## References

- `pkg/xgoja/sourcegraph/graph.go`
- `cmd/xgoja/internal/plan/plan.go`
- `pkg/xgoja/app/root.go`
- `pkg/xgoja/app/jsverb_sources.go`
- `pkg/jsparse/treesitter.go`
- `pkg/jsparse/analyze.go`
- Ticket: `GOJA-XGOJA-V2-RUNTIME-001`
