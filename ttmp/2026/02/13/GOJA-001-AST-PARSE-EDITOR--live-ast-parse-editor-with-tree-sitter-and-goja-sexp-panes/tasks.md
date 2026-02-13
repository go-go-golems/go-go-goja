# Tasks

## TODO

- [x] Create ticket `GOJA-001-AST-PARSE-EDITOR`
- [x] Produce detailed analysis of current tree-sitter + goja AST setup in `go-go-goja/`
- [x] Write implementation blueprint for 3-pane live editor with CST/AST SEXP output
- [x] Maintain detailed implementation diary during work
- [x] Upload ticket docs bundle to reMarkable

## Task 1: Reusable SEXP Renderers (`pkg/jsparse`)

- [x] Add `pkg/jsparse/sexp.go`
- [x] Implement `SExprOptions`
- [x] Implement `CSTToSExpr(root *TSNode, opts *SExprOptions) string`
- [x] Implement `ASTIndexToSExpr(idx *Index, opts *SExprOptions) string`
- [x] Implement `ASTToSExpr(program *ast.Program, src string, opts *SExprOptions) string`
- [x] Add `pkg/jsparse/sexp_test.go`
- [x] Run `GOWORK=off go test ./pkg/jsparse -count=1`
- [x] Commit Task 1 and update diary/changelog

## Task 2: New Live 3-Pane Editor Command

- [x] Add `cmd/ast-parse-editor/main.go`
- [x] Add `cmd/ast-parse-editor/app/model.go`
- [x] Implement 3-pane layout (editor, TS SEXP, AST SEXP)
- [x] Implement live tree-sitter parse on each edit
- [x] Implement debounced goja AST parse and valid-only AST pane rendering
- [x] Add keybindings/help/status for the new command
- [x] Add integration tests for command model behavior
- [x] Run command-level tests in tmux
- [x] Commit Task 2 and update diary/changelog

## Task 3: Test Coverage and Hardening

- [x] Add deterministic SEXP golden/assertion tests
- [x] Add truncation/depth guard tests
- [x] Add parse-error state transition tests (valid <-> invalid while typing)
- [x] Run focused regression tests in tmux
- [x] Run `GOWORK=off go test ./pkg/jsparse ./cmd/ast-parse-editor/... -count=1`
- [x] Commit Task 3 and update diary/changelog

## Task 4: Empty Source Editing Validity

- [x] Reproduce empty-file AST pane behavior (`(waiting-for-valid-parse)` regression)
- [x] Make empty source parse render as valid AST S-expression (`(Program)`)
- [x] Add regression tests in `pkg/jsparse/sexp_test.go`
- [x] Add regression test in `cmd/ast-parse-editor/app/model_test.go`
- [x] Run `GOWORK=off go test ./pkg/jsparse ./cmd/ast-parse-editor/... -count=1` in tmux
- [x] Commit Task 4 and update diary/changelog

## Task 5: CLI Empty-Buffer Entry Flow

- [x] Reproduce CLI failure when no argument is provided
- [x] Reproduce CLI failure when target file path does not exist
- [x] Allow `ast-parse-editor` to start with no args (`untitled.js`, empty source)
- [x] Allow `ast-parse-editor <missing.js>` to start with empty source
- [x] Add unit tests for `loadInput` behavior
- [x] Run `GOWORK=off go test ./cmd/ast-parse-editor/... ./pkg/jsparse -count=1` in tmux
- [x] Commit Task 5 and update diary/changelog
