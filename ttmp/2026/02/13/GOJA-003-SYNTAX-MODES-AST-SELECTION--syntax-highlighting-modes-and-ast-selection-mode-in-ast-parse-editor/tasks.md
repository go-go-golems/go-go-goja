# Tasks

## TODO

- [x] Create analysis + diary docs for GOJA-003
- [x] Write detailed implementation plan for syntax highlighting + AST-selection mode
- [x] Add editor mode state and keybindings (`m` mode toggle)
- [x] Add AST index storage + selection state synced from cursor offset
- [x] Implement AST-select navigation (`h/j/k/l`) and source cursor sync
- [x] Add syntax highlighting toggle (`s`) and CST token coloring
- [x] Update header/status/help for mode + selection visibility
- [x] Add/adjust model tests for mode/selection/syntax toggles
- [x] Run `GOWORK=off go test ./cmd/ast-parse-editor/... ./pkg/jsparse -count=1` in tmux
- [x] Commit code changes for GOJA-003
- [x] Update GOJA-003 diary/changelog and check off completed tasks

## Follow-up: Key Conflicts and Tree Widget

- [x] Move global mode/syntax toggles to `ctrl+` bindings to avoid insert-mode key capture
- [x] Show inspector-style expandable AST tree widget in right pane while in AST-select mode
- [x] Add tree pane controls (`space` toggle, `h/l` collapse/expand, `j/k` move) when AST pane focused
- [x] Add regression tests for literal `m/s` insertion and tree expand toggle
- [x] Run `GOWORK=off go test ./cmd/ast-parse-editor/... ./pkg/jsparse -count=1` in tmux
- [x] Commit follow-up code and update diary/changelog

## Follow-up: Definition/Usages and Dual Cursor Highlights

- [x] Add `ctrl+d` go-to-definition using AST resolution bindings
- [x] Add `ctrl+g` find-usages toggle and `esc` clear behavior
- [x] Highlight usage nodes in source and AST tree panes
- [x] Keep both tree-sitter cursor node and AST cursor node highlights active in insert mode
- [x] Add regression tests for go-to-definition and usages toggling
- [x] Run `GOWORK=off go test ./cmd/ast-parse-editor/... ./pkg/jsparse -count=1` in tmux
- [x] Commit follow-up code and update diary/changelog

## Follow-up: Cursor-Synced SEXP Pane Highlighting

- [x] Track selected TS SEXP line from cursor node range metadata
- [x] Track selected AST SEXP line from AST cursor node span metadata
- [x] Highlight selected SEXP line in TS/AST SEXP panes and auto-scroll it into view
- [x] Add regression tests for SEXP line tracking and invalid-parse clearing
- [x] Run `GOWORK=off go test ./cmd/ast-parse-editor/... ./pkg/jsparse -count=1` in tmux
- [x] Commit follow-up code and update diary/changelog

## Follow-up: Byte/Rune Column Correctness

- [x] Fix AST-select cursor jump to convert AST byte column to editor rune column
- [x] Fix source-pane highlight column checks to compare byte columns against byte-based spans
- [x] Add regression tests for multibyte go-to-definition cursor placement and byte/rune conversion helpers
- [x] Run `GOWORK=off go test ./cmd/ast-parse-editor/... ./pkg/jsparse -count=1` in tmux
- [x] Commit follow-up code and update diary/changelog
