---
Title: Investigation diary
Ticket: GOJA-XGOJA-SOURCEGRAPH-PARSER-001
Status: active
Topics:
    - xgoja
    - sourcegraph
    - typescript
    - code-generation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Diary for parser-backed xgoja sourcegraph import resolution investigation."
LastUpdated: 2026-06-14T09:42:16.138610864-04:00
WhatFor: "Chronological notes for sourcegraph parser/runtime alias investigation."
WhenToUse: "Read before implementing GOJA-XGOJA-SOURCEGRAPH-PARSER-001."
---

# Diary

## Goal

This diary captures the investigation and design work for replacing xgoja sourcegraph's regex-based import scanning with parser-backed import collection and correct RuntimePlan alias propagation.

## Step 1: Create parser-backed sourcegraph ticket and compare parser options

I created a dedicated docmgr ticket under `go-go-goja/ttmp` for the sourcegraph import parser/runtime alias work. The ticket now contains a design document, this diary, task list, and three runnable experiment scripts that compare Goja parser, esbuild Build/metafile collection, and tree-sitter JavaScript parsing.

The third experiment confirms that tree-sitter would help substantially for JavaScript/ESM/CommonJS import extraction because it already parses import/export forms that Goja rejects and avoids regex false positives in comments and strings. The caveat is TypeScript: the repo currently has `tree-sitter-javascript`, not `tree-sitter-typescript`, so TS-specific syntax recovers with parse errors unless we add the TypeScript grammar or use esbuild for TS/TSX.

### Prompt Context

**User prompt (verbatim):** "Ok, create a new ticket in `docmgr --root go-go-goja/ttmp ...` to handle these issues, copy all thes cripts you've been using for testing in there too. Would using treesitter work and make your job easier? run a third experiment"

**User prompt (verbatim):** "(we already use treesitter in parts of go-go-goja)"

**Assistant interpretation:** Create a new docmgr ticket in the go-go-goja ticket workspace for sourcegraph parser/runtime alias issues, preserve the parser experiments as ticket scripts, run an additional tree-sitter experiment, and evaluate whether tree-sitter is a good implementation path.

**Inferred user intent:** Turn the ad-hoc parser investigation into a structured implementation ticket with reproducible evidence and a concrete recommendation for the next sourcegraph fix.

### What I did

- Created ticket `GOJA-XGOJA-SOURCEGRAPH-PARSER-001` with `docmgr --root go-go-goja/ttmp`.
- Added design document `design-doc/01-parser-backed-xgoja-sourcegraph-import-resolution-design.md`.
- Added this investigation diary.
- Copied/wrote experiment scripts:
  - `scripts/01-goja-parser-experiment.go`
  - `scripts/02-esbuild-import-experiment.go`
  - `scripts/03-treesitter-import-experiment.go`
- Ran all three experiments from the go-go-goja repo root.
- Updated `tasks.md` with completed setup/experiment tasks and implementation follow-ups.

### Why

- The current sourcegraph regex parser is too fragile for xgoja v2's RuntimePlan/source graph contract.
- Literal aliases such as `fs:assets` should be statically visible, validated, and allowed when selected by `runtime.modules[].as`.
- Parser experiments make the implementation choice evidence-based instead of speculative.

### What worked

The Goja parser experiment confirmed Goja can parse CommonJS `require("fs:assets")` but not ESM import/export forms.

The esbuild experiment confirmed that esbuild can collect JS/TS/ESM imports via Build/metafile when runtime aliases are marked external.

The tree-sitter experiment confirmed the existing tree-sitter JavaScript grammar can extract:

```text
require("fs:assets")
import assets from "fs:assets"
import "./setup.js"
export { x } from "./x.js"
await import("./x.js")
```

It also emitted no false import edges for:

```js
const s = "require('not-real')"; // import "also-not-real"
```

### What didn't work

The existing tree-sitter JavaScript grammar reports parse errors for TypeScript-only syntax:

```text
hasError=true
```

for a source containing `import type` and `export interface`. It still recovered enough to extract import strings, but that is not a clean production TS parser story.

Non-literal dynamic require remains intentionally not statically resolvable:

```js
require(["fs", "host"].join(":"))
```

The tree-sitter experiment recorded this as a dynamic edge rather than a concrete `fs:host` import.

### What I learned

- Tree-sitter would make the JS/ESM/CommonJS side easier than using Goja AST directly because Goja rejects ESM import/export syntax.
- Esbuild is still the best already-present tool for TypeScript/TSX import collection because xgoja already uses it for TypeScript compilation and it handles alias externals cleanly.
- Parser choice does not replace the alias propagation fix: sourcegraph still needs configured RuntimePlan aliases such as `fs:assets` as inputs.

### What was tricky to build

- The tree-sitter experiment had to inspect concrete syntax tree node kinds rather than rely on a high-level import API. The useful nodes were `import_statement`, `export_statement`, and `call_expression` with function `require` or `import`.
- The current repo dependency is `tree-sitter-javascript`, so TypeScript support is only error-tolerant recovery unless another grammar is added.
- Distinguishing literal imports from non-literal dynamic imports is important. A parser can tell us that `require(["fs", "host"].join(":"))` is a dynamic call, but it should not pretend that this is statically known.

### What warrants a second pair of eyes

- Decide whether TS/TSX sourcegraph parsing should use esbuild only or add `tree-sitter-typescript`.
- Decide whether strict generated builds should fail or warn on non-literal dynamic imports.
- Review the planned alias propagation path so provider-wide aliases do not accidentally replace command-selected RuntimePlan aliases.

### What should be done in the future

- Implement failing regressions for literal `require("fs:assets")` and ESM/TS alias imports.
- Fix configured RuntimePlan alias propagation before swapping parsers.
- Introduce an `ImportCollector` abstraction and implement parser-backed collection.

### Code review instructions

- Start with the design document for the parser comparison and staged implementation plan.
- Run the experiments from the repo root:

```bash
go run ttmp/2026/06/14/GOJA-XGOJA-SOURCEGRAPH-PARSER-001--replace-xgoja-sourcegraph-regex-import-scanning-with-parser-backed-runtime-alias-resolution/scripts/01-goja-parser-experiment.go
go run ttmp/2026/06/14/GOJA-XGOJA-SOURCEGRAPH-PARSER-001--replace-xgoja-sourcegraph-regex-import-scanning-with-parser-backed-runtime-alias-resolution/scripts/02-esbuild-import-experiment.go
go run ttmp/2026/06/14/GOJA-XGOJA-SOURCEGRAPH-PARSER-001--replace-xgoja-sourcegraph-regex-import-scanning-with-parser-backed-runtime-alias-resolution/scripts/03-treesitter-import-experiment.go
```

### Technical details

The preferred staged direction is:

```text
RuntimePlan selected modules
  -> configured runtime aliases including runtime.modules[].as
  -> SourceRegistry command source scope
  -> parser-backed sourcegraph ImportCollector
  -> local/runtime/unknown import resolution
```

Tree-sitter is attractive for JS because it is already present and catches ESM syntax without regex false positives. Esbuild remains attractive for TS because it already understands TypeScript and can mark xgoja runtime aliases external.

## Step 2: Try tree-sitter TypeScript and TSX support

I ran a fourth experiment using `github.com/tree-sitter/tree-sitter-typescript` in a temporary Go module so the repository `go.mod` stayed unchanged. This tested whether a tree-sitter-only sourcegraph parser can cover TypeScript and TSX cleanly instead of delegating TS/TSX import collection to esbuild.

The experiment was positive: the TypeScript grammar parsed `import type`, `export interface`, dynamic `import("...")`, literal `require("...")`, and non-literal dynamic `require(expr)` without errors. The TSX grammar also parsed JSX/TSX syntax and extracted `react`, `fs:assets`, and local component imports. This makes tree-sitter with the TypeScript grammar the preferred implementation path for sourcegraph import collection.

### Prompt Context

**User prompt (verbatim):** "can we try with typescript treesitter support?"

**Assistant interpretation:** Run an additional experiment with the actual tree-sitter TypeScript grammar, not just the existing JavaScript grammar's error recovery, and update the ticket evidence/recommendation.

**Inferred user intent:** Determine whether adding tree-sitter TypeScript support would be simpler and more robust than using esbuild as the TS/TSX import collector.

### What I did

- Added `scripts/04-treesitter-typescript-experiment.sh`.
- The script creates a temporary Go module and installs:
  - `github.com/tree-sitter/go-tree-sitter@v0.25.0`
  - `github.com/tree-sitter/tree-sitter-typescript@v0.23.2`
- Tested `LanguageTypescript()` on TypeScript source containing:
  - `import type`
  - normal ESM imports
  - `fs:assets` alias import
  - `export ... from`
  - `export interface`
  - dynamic `import("./dynamic")`
  - literal `require("fs:host")`
  - non-literal `require(["fs", "assets"].join(":"))`
- Tested `LanguageTSX()` on TSX source containing React-style JSX and imports.
- Updated the design doc and task list to recommend tree-sitter TypeScript support as the first implementation path.

### Why

- The existing JavaScript tree-sitter grammar recovered from TypeScript syntax but reported parse errors.
- A production sourcegraph parser should not depend on error recovery for standard `.ts` and `.tsx` files.
- The implementation should use the simplest parser model that covers xgoja's JS/TS source surface.

### What worked

The TypeScript grammar parsed without errors:

```text
== typescript-imports ==
hasError=false
edge kind=import specifier="./types" dynamic=false
edge kind=import specifier="./helper" dynamic=false
edge kind=import specifier="fs:assets" dynamic=false
edge kind=export specifier="./more" dynamic=false
edge kind=import specifier="./dynamic" dynamic=false
edge kind=require specifier="fs:host" dynamic=false
edge kind=require specifier="" dynamic=true
```

The TSX grammar also parsed without errors:

```text
== tsx-imports ==
hasError=false
edge kind=import specifier="react" dynamic=false
edge kind=import specifier="fs:assets" dynamic=false
edge kind=import specifier="./Widget" dynamic=false
```

The false-positive case still emitted no import edges:

```text
== comment-string-false-positive ==
hasError=false
```

### What didn't work

N/A. The experiment succeeded. The only caveat is that adopting this approach requires adding `github.com/tree-sitter/tree-sitter-typescript` as a repository dependency.

### What I learned

- `tree-sitter-typescript` exposes both `LanguageTypescript()` and `LanguageTSX()` from `github.com/tree-sitter/tree-sitter-typescript/bindings/go`.
- Tree-sitter gives sourcegraph the exact data it needs: literal import specifiers plus the ability to distinguish dynamic/non-literal imports from static imports.
- Esbuild remains a useful comparison point, but tree-sitter is a cleaner fit for sourcegraph import extraction because it does not require running a no-write bundle build.

### What was tricky to build

- The experiment is intentionally a shell script rather than a plain Go file because the repo does not yet depend on `tree-sitter-typescript`. The script creates a temporary module so the evidence is reproducible without modifying `go.mod` during investigation.
- Dynamic `require(["fs", "assets"].join(":"))` contains string literals inside the argument expression. The collector must not treat those nested strings as import specifiers; it should mark the call dynamic instead.

### What warrants a second pair of eyes

- Confirm that adding `tree-sitter-typescript` is acceptable dependency-wise.
- Review whether the production collector should use tree-sitter queries or explicit CST walking by node kind/field name.
- Decide whether dynamic non-literal imports should be hard errors in generated builds.

### What should be done in the future

- Add `tree-sitter-typescript` to `go.mod` when implementing the parser-backed collector.
- Implement a collector that selects JavaScript, TypeScript, or TSX grammar by extension.
- Add table-driven tests for the exact cases from this experiment.

### Code review instructions

- Run:

```bash
ttmp/2026/06/14/GOJA-XGOJA-SOURCEGRAPH-PARSER-001--replace-xgoja-sourcegraph-regex-import-scanning-with-parser-backed-runtime-alias-resolution/scripts/04-treesitter-typescript-experiment.sh
```

- Review the design doc's updated Tree-sitter TypeScript experiment and Phase 4 recommendation.

### Technical details

The production parser should map extensions approximately as follows:

```text
.js/.jsx/.mjs/.cjs -> JavaScript or TSX grammar
.ts/.mts/.cts      -> LanguageTypescript()
.tsx               -> LanguageTSX()
```

For import extraction, the relevant nodes are still `import_statement`, `export_statement`, and `call_expression` where the function node is `require` or `import`.

## Step 3: Implement tree-sitter sourcegraph parsing and runtime alias propagation

I turned the parser experiments into active sourcegraph behavior. The regex import extractor is gone from `pkg/xgoja/sourcegraph`; sourcegraph now parses JavaScript, ESM, TypeScript, and TSX with tree-sitter, extracts static import/export/require/dynamic-import specifiers, and rejects non-literal dynamic imports instead of silently missing them.

I also fixed the runtime scanning side of the original `fs:assets` issue. Provider command-set source scans now receive aliases from the command-selected RuntimePlan modules, so a command using `runtime.modules[].as: fs:assets` can literally import `require("fs:assets")` without the dynamic workaround.

### Prompt Context

**User prompt (verbatim):** "Create a detailed list of tasks and add to the ticket, then work on them one by one, committing at appropriate intervals, and keeping a detailed diary."

**Assistant interpretation:** Expand the ticket task plan, begin implementing the sourcegraph parser/runtime-alias fixes, update the diary as work proceeds, and commit at sensible milestones.

**Inferred user intent:** Move the parser investigation from design into working code while preserving task bookkeeping and review context.

**Commit (code):** bf00208 — "Add parser-backed xgoja sourcegraph imports"

### What I did

- Rewrote the ticket task list into a detailed implementation plan with completed investigation tasks and implementation phases.
- Added `github.com/tree-sitter/tree-sitter-typescript v0.23.2` to `go.mod` / `go.sum`.
- Added `pkg/xgoja/sourcegraph/imports.go` with tree-sitter-backed import extraction.
- Removed the regex-based `parseImports` implementation from `pkg/xgoja/sourcegraph/graph.go`.
- Updated sourcegraph import resolution to reject dynamic non-literal import calls.
- Added sourcegraph tests for:
  - colon runtime aliases;
  - TypeScript/TSX import extraction;
  - side-effect imports;
  - export-from imports;
  - dynamic `import("...")`;
  - false-positive strings/comments;
  - non-literal dynamic `require(...)` diagnostics.
- Kept planner coverage for configured `runtime.modules[].as` aliases in `cmd/xgoja/internal/plan/plan_test.go`.
- Threaded RuntimePlan module aliases through runtime source scanning:
  - `SourceRegistry` now stores runtime aliases;
  - `SourceRegistry.JSVerbs()` passes those aliases into scans;
  - provider command sets override scoped source registries with aliases from their selected modules;
  - host-level source registry receives aliases from `RuntimePlan.Runtime.Modules`.
- Ran the sourcegraph/runtime alias example smoke target successfully.

### Why

- Regex import scanning was the wrong abstraction for v2 sourcegraph validation.
- `fs:assets` is a configured runtime alias, so sourcegraph must validate it against RuntimePlan module selections rather than provider-wide default module names.
- Dynamic `require(["fs", "assets"].join(":"))` hides dependencies from the static graph; sourcegraph should encourage literal imports and flag non-literal dynamic imports.

### What worked

Focused tests passed:

```bash
go test ./cmd/xgoja/internal/plan ./pkg/xgoja/sourcegraph ./pkg/xgoja/app ./pkg/xgoja/providers/http -count=1
```

The broad sourcegraph/runtime alias example passed after the alias propagation fix:

```bash
make -C examples/xgoja/17-sourcegraph-runtime-aliases clean smoke
make -C examples/xgoja/17-sourcegraph-runtime-aliases serve-smoke prove-self-contained
```

The example builds a generated binary and validates literal `require("fs:assets")` in the HTTP serve provider command path.

### What didn't work

The earlier example smoke had failed before this fix with:

```text
create command set go-go-goja-http.serve: resolve imports for jsverb source js-site: site.js imports unknown bare specifier "fs:assets"
```

The root cause was not the regex parser's ability to match `fs:assets`; it was runtime scan alias propagation. Provider command scans used provider-wide aliases instead of the configured RuntimePlan alias set selected for the command.

### What I learned

- The parser and alias issues were separate. Tree-sitter fixes syntax coverage and false positives; alias propagation fixes the literal runtime alias rejection.
- Tree-sitter TypeScript support makes sourcegraph parsing straightforward for TSX without invoking esbuild's build pipeline.
- Command-selected module aliases are the right input for provider command-set scans because command providers can select a subset of runtime modules.

### What was tricky to build

- Dynamic calls need careful CST handling. The collector must not treat nested string literals inside `require(["fs", "assets"].join(":"))` as static import specifiers.
- The existing `SourceRegistry` API is provider-facing and does not expose aliases, so aliases were added as internal runtime state while keeping the provider-facing interface unchanged.
- Host-level scans and provider command scans have different alias scopes: host setup can use all RuntimePlan aliases, while command providers should use their selected module descriptors.

### What warrants a second pair of eyes

- Review whether dynamic non-literal imports should be hard errors in all sourcegraph scans, or whether some runtime script modes need a warning-only policy.
- Review whether `.jsx` should use the JavaScript grammar or the TSX grammar in production; the current implementation uses JavaScript for JS/JSX and TypeScript TSX only for `.tsx`.
- Review alias scoping in `SourceRegistry.ScopedWithRuntimeAliases` and `Host.newCommandSet` to confirm commands with explicit `modules:` are isolated as intended.

### What should be done in the future

- Finalize and commit the example directory.
- Update user-facing docs if dynamic import errors need documentation.
- Run broader xgoja validation before final handoff.

### Code review instructions

- Start with `pkg/xgoja/sourcegraph/imports.go` and `pkg/xgoja/sourcegraph/graph.go`.
- Then review alias propagation in:
  - `pkg/xgoja/app/source_registry.go`
  - `pkg/xgoja/app/jsverb_sources.go`
  - `pkg/xgoja/app/command_providers.go`
  - `pkg/xgoja/app/host.go`
- Review tests in:
  - `pkg/xgoja/sourcegraph/graph_test.go`
  - `cmd/xgoja/internal/plan/plan_test.go`
- Validate with:

```bash
go test ./cmd/xgoja/internal/plan ./pkg/xgoja/sourcegraph ./pkg/xgoja/app ./pkg/xgoja/providers/http -count=1
make -C examples/xgoja/17-sourcegraph-runtime-aliases smoke
```

### Technical details

The new static import collector recognizes these CST nodes:

```text
import_statement source: string
export_statement source: string
call_expression function: require/import, first argument: string
```

Non-literal dynamic calls produce a dynamic import spec and sourcegraph currently rejects them with:

```text
<file> contains dynamic non-literal <kind> import
```

### Pre-commit failure note

The first commit attempt failed because the three copied Go experiment scripts lived in the same `scripts` directory as `package main` files, so `go test ./...` treated them as one package and saw multiple `main` declarations:

```text
ttmp/2026/06/14/GOJA-XGOJA-SOURCEGRAPH-PARSER-001--replace-xgoja-sourcegraph-regex-import-scanning-with-parser-backed-runtime-alias-resolution/scripts/02-esbuild-import-experiment.go:75:6: main redeclared in this block
	ttmp/2026/06/14/GOJA-XGOJA-SOURCEGRAPH-PARSER-001--replace-xgoja-sourcegraph-regex-import-scanning-with-parser-backed-runtime-alias-resolution/scripts/01-goja-parser-experiment.go:9:6: other declaration of main
ttmp/2026/06/14/GOJA-XGOJA-SOURCEGRAPH-PARSER-001--replace-xgoja-sourcegraph-regex-import-scanning-with-parser-backed-runtime-alias-resolution/scripts/03-treesitter-import-experiment.go:17:6: main redeclared in this block
	ttmp/2026/06/14/GOJA-XGOJA-SOURCEGRAPH-PARSER-001--replace-xgoja-sourcegraph-regex-import-scanning-with-parser-backed-runtime-alias-resolution/scripts/01-goja-parser-experiment.go:9:6: other declaration of main
```

I fixed this by adding `//go:build ignore` to each standalone Go experiment script. They remain runnable with `go run path/to/script.go`, but they are excluded from package builds and broad `go test ./...`.
