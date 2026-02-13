# Changelog

## 2026-02-13

- Initial workspace created


## 2026-02-13

Implemented syntax highlighting toggle and AST-select mode with parser-index navigation in ast-parse-editor, plus regression tests and tmux validation (commit 1d1a88e).

### Related Files

- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/cmd/ast-parse-editor/app/model.go — Added mode switching
- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/cmd/ast-parse-editor/app/model_test.go — Added mode toggle
- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/ttmp/2026/02/13/GOJA-003-SYNTAX-MODES-AST-SELECTION--syntax-highlighting-modes-and-ast-selection-mode-in-ast-parse-editor/reference/01-diary.md — Step 2 execution diary and verification output


## 2026-02-13

Follow-up: switched global toggles to ctrl bindings and added inspector-style expandable AST tree widget in AST-select mode, including pane navigation controls and regression tests (commit e162ccf).

### Related Files

- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/cmd/ast-parse-editor/app/model.go — Ctrl keybindings and AST tree-pane rendering/navigation
- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/cmd/ast-parse-editor/app/model_test.go — Tests for m/s literal insertion and AST tree expand toggle
- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/ttmp/2026/02/13/GOJA-003-SYNTAX-MODES-AST-SELECTION--syntax-highlighting-modes-and-ast-selection-mode-in-ast-parse-editor/reference/01-diary.md — Step 3 follow-up diary


## 2026-02-13

Follow-up: added go-to-definition (ctrl+d), find-usages toggle (ctrl+g), usage highlighting, and dual tree-sitter/AST cursor highlights in insert mode; added regression tests and tmux validation (commit 9c6489b).

### Related Files

- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/cmd/ast-parse-editor/app/model.go — Symbol navigation
- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/cmd/ast-parse-editor/app/model_test.go — Regression tests for go-to-definition and find-usages
- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/ttmp/2026/02/13/GOJA-003-SYNTAX-MODES-AST-SELECTION--syntax-highlighting-modes-and-ast-selection-mode-in-ast-parse-editor/reference/01-diary.md — Step 4 follow-up implementation diary


## 2026-02-13

Step 5: cursor-synced TS/AST SEXP pane highlighting with line auto-scroll and regression tests (commit 90ca11a).

### Related Files

- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/cmd/ast-parse-editor/app/model.go — SEXP selected-line tracking and pane rendering
- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/cmd/ast-parse-editor/app/model_test.go — SEXP tracking tests
- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/ttmp/2026/02/13/GOJA-003-SYNTAX-MODES-AST-SELECTION--syntax-highlighting-modes-and-ast-selection-mode-in-ast-parse-editor/reference/01-diary.md — Step 5 diary details

## 2026-02-13

Step 6: fixed byte-vs-rune column mapping for source highlights and AST cursor jumps on multibyte lines (commit 99ec6d2).

### Related Files

- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/cmd/ast-parse-editor/app/model.go — byte/rune conversion helpers and rendering fixes
- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/cmd/ast-parse-editor/app/model_test.go — multibyte regression tests
- /home/manuel/workspaces/2026-02-13/ast-parse-editor/go-go-goja/ttmp/2026/02/13/GOJA-003-SYNTAX-MODES-AST-SELECTION--syntax-highlighting-modes-and-ast-selection-mode-in-ast-parse-editor/reference/01-diary.md — Step 6 diary details

