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

- [ ] Add deterministic SEXP golden/assertion tests
- [ ] Add truncation/depth guard tests
- [ ] Add parse-error state transition tests (valid <-> invalid while typing)
- [ ] Run focused regression tests in tmux
- [ ] Run `GOWORK=off go test ./pkg/jsparse ./cmd/ast-parse-editor/... -count=1`
- [ ] Commit Task 3 and update diary/changelog
