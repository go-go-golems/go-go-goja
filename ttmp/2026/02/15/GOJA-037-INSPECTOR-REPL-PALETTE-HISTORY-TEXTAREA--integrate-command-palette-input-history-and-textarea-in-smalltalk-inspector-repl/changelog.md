# Changelog

## 2026-02-15

- Initial workspace created


## 2026-02-15

Step 2: Integrated bobatea command palette into smalltalk-inspector command flow (commit 259f7d2).

### Related Files

- /home/manuel/workspaces/2026-02-15/use-bobatea-goja/go-go-goja/cmd/smalltalk-inspector/app/command_palette.go — Added command palette command registration and execution wiring


## 2026-02-15

Step 3: Switched REPL history navigation/storage to bobatea inputhistory (commit 9f26b2a).

### Related Files

- /home/manuel/workspaces/2026-02-15/use-bobatea-goja/go-go-goja/cmd/smalltalk-inspector/app/update.go — Replaced local history logic with inputhistory navigation and draft restore


## 2026-02-15

Step 4: Added multiline REPL mode using bobatea textarea with mode toggles and tests (commit be8c604).

### Related Files

- /home/manuel/workspaces/2026-02-15/use-bobatea-goja/go-go-goja/cmd/smalltalk-inspector/app/repl_multiline_test.go — Added multiline mode coverage and submit validation


## 2026-02-15

Step 5: Completed regression/smoke validation for inspector and repl commands; noted non-TTY limitation for smalltalk-inspector help.

### Related Files

- /home/manuel/workspaces/2026-02-15/use-bobatea-goja/go-go-goja/cmd/smalltalk-inspector/app/view.go — Validated multiline layout path under test/smoke cycle

