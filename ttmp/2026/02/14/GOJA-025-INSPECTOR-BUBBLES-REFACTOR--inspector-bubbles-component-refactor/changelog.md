# Changelog

## 2026-02-14

- Initial workspace created


## 2026-02-14

Step 1: Completed mode-aware keymap migration and replaced static help/footer wiring with bubbles help and spinner plumbing (commit 3339aa86b12bbe450ebb01e184241fe2ff47a541).

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/inspector/app/keymap.go — New keymap definitions with mode tags and help metadata
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/inspector/app/model.go — Help/spinner models and mode-sync plumbing


## 2026-02-14

Step 2: Refactored source/tree panes to use bubbles viewport and bubbles list with selection synchronization preserved (commit 13d7bbfcecd801d6bd75743826ac5e360f13062b).

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/inspector/app/model.go — Viewport/list integration and sync updates
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/inspector/app/tree_list.go — New tree list adapter and list model setup


## 2026-02-14

Step 3: Added textinput command mode and table-driven metadata panel for selected AST node context (commit 8e1e1ce4b5c2f4caf3e202fb521bf3b4d2919f99).

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/inspector/app/keymap.go — Added command binding for ':' mode
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/inspector/app/model.go — Command mode and metadata table implementation


## 2026-02-14

Step 4: Finalized design guide, diary, index links, and task bookkeeping for commit-level traceability.

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-025-INSPECTOR-BUBBLES-REFACTOR--inspector-bubbles-component-refactor/index.md — Ticket entrypoint update
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-025-INSPECTOR-BUBBLES-REFACTOR--inspector-bubbles-component-refactor/reference/01-inspector-refactor-design-guide.md — Detailed implementation guide
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-025-INSPECTOR-BUBBLES-REFACTOR--inspector-bubbles-component-refactor/reference/02-diary.md — Per-task diary


## 2026-02-14

Step 5: Committed finalized ticket documentation and completed task checklist (commit 9725df9e5523ecb82f9d36b9ca081f11ade91573).

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-025-INSPECTOR-BUBBLES-REFACTOR--inspector-bubbles-component-refactor/reference/02-diary.md — Diary updated with final docs commit
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/ttmp/2026/02/14/GOJA-025-INSPECTOR-BUBBLES-REFACTOR--inspector-bubbles-component-refactor/tasks.md — All tasks complete

