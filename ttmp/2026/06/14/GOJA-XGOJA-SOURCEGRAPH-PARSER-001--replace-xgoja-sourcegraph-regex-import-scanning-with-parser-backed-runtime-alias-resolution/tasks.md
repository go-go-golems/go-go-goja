# Tasks

## Completed Investigation

- [x] Create ticket workspace and add design/diary documents.
- [x] Copy Goja parser and esbuild import-collector experiments into ticket scripts.
- [x] Run a tree-sitter JavaScript import extraction experiment using the repo's existing dependency.
- [x] Run a tree-sitter TypeScript/TSX experiment in a temporary module without modifying repo go.mod.
- [x] Record initial parser comparison and implementation recommendation.

## Implementation Plan

### 1. Guard the runtime-alias regression

- [x] Add/keep focused sourcegraph unit coverage for colon-containing runtime aliases such as `fs:assets` and `db:readonly`.
- [x] Add/keep planner coverage proving configured `runtime.modules[].as` aliases flow into `Plan.RuntimeAliases`.
- [x] Add coverage for ESM, side-effect import, export-from, dynamic import, and false-positive string/comment cases.
- [x] Add coverage for non-literal dynamic `require(...)` / `import(...)` diagnostics.

### 2. Fix configured alias propagation in runtime scans

- [x] Audit all sourcegraph scan entry points (`cmd/xgoja/internal/plan`, builtin jsverbs, provider command contexts, hot reload).
- [x] Replace provider-wide alias fallbacks with selected RuntimePlan module aliases where a command/runtime context is available.
- [x] Ensure `SourceRegistry.JSVerbs()` scans receive aliases selected for the command that owns the source scope.
- [x] Validate literal `require("fs:assets")` works in the generated example without the dynamic `require(["fs", "assets"].join(":"))` workaround.

### 3. Add parser-backed import collection

- [x] Add `github.com/tree-sitter/tree-sitter-typescript` to `go.mod` / `go.sum`.
- [x] Introduce parser-backed import collection in `pkg/xgoja/sourcegraph`.
- [x] Implement tree-sitter import collection for JS/ESM/CommonJS/TS/TSX.
- [x] Select JavaScript, TypeScript, or TSX grammar by source file extension.
- [x] Remove the regex parser once table-driven tests pass.
- [x] Preserve sourcegraph's current local/runtime/unknown import classification and path-escape checks.

### 4. Strengthen example coverage

- [x] Finalize `examples/xgoja/17-sourcegraph-runtime-aliases` as a broad sourcegraph/runtime example.
- [x] Ensure the example uses literal runtime aliases (`require("fs:assets")`, `import ... from "fs:assets"`) rather than dynamic workarounds.
- [x] Cover JavaScript and TypeScript jsverb source sets, local imports, embedded assets, host FS, HTTP provider command sets, builtin jsverbs, and DTS generation.
- [x] Add a smoke target that builds the generated binary, runs it, and validates HTTP/API output.
- [x] Remove generated `dist/` binaries from the working tree and keep them ignored or generated-only.

### 5. Documentation, validation, and handoff

- [x] Update xgoja docs if parser behavior or dynamic import diagnostics become user-visible.
- [x] Update the ticket design doc with final implementation decisions.
- [x] Keep the investigation diary updated after each implementation step.
- [x] Relate modified code/example files to the ticket design doc.
- [x] Run focused tests: `go test ./pkg/xgoja/sourcegraph ./cmd/xgoja/internal/plan ./pkg/xgoja/app -count=1`.
- [x] Run example smoke: `make -C examples/xgoja/17-sourcegraph-runtime-aliases smoke`.
- [x] Run broader xgoja validation before final handoff.
- [x] Run `docmgr --root ttmp doctor --ticket GOJA-XGOJA-SOURCEGRAPH-PARSER-001 --stale-after 30`.
