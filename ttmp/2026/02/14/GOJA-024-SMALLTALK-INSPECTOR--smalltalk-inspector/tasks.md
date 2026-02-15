# Tasks

## TODO

- [x] Replace placeholder task list with phase-structured implementation plan

- [x] Phase 1 bootstrap: create `cmd/smalltalk-inspector/main.go` and `cmd/smalltalk-inspector/app/{model,update,view,messages,styles}.go` with alt-screen launch and root layout skeleton
- [x] Phase 1 key system: wire `mode-keymap` and `bubbles/help` using patterns from `cmd/inspector/app/keymap.go`; include explicit modes for globals/members/source/repl/stack
- [x] Phase 1 pane scaffolding: wire `bubbles/list` (globals/members), `bubbles/viewport` (source), `bubbles/table` (metadata), `bubbles/textinput` (command line), `bubbles/spinner` (status)
- [x] Phase 1 load flow: implement `:load` pipeline in root update loop (read file -> `jsparse.Analyze` -> goja runtime bootstrap -> UI state refresh)
- [x] Phase 1 static browsing: implement globals and members panes from `pkg/jsparse` bindings/resolution and render source jumps for selected class/method

- [x] Phase 2 domain extraction: add `pkg/inspector/analysis/{session,method_symbols,xref}.go` and `pkg/inspector/runtime/{session,introspect,function_map,errors}.go` to keep cmd layer thin
- [x] Phase 2 REPL/object inspect: implement eval command path and object browser model with inspect-target payloads (string keys, inherited keys, symbol keys)
- [x] Phase 2 breadcrumb navigation: implement navigation stack model (`push`, `back`, frame restore) for recursive object/method inspection
- [x] Phase 2 descriptor/symbol panes: add descriptor table and symbol-key section using runtime helpers for `Object.getOwnPropertyDescriptor` and `Object.Symbols()`

- [x] Phase 3 error inspection: implement exception capture and stack frame list (source jump + locals/object drill-in entry points)
- [x] Phase 3 tests: add focused tests for pane sync, inspect navigation, REPL eval behavior, descriptor/symbol rendering, and stack/source jump mapping
- [x] Phase 3 polish/docs: update `reference/02-smalltalk-goja-inspector-interface-and-component-design.md`, `reference/01-diary.md`, and `changelog.md` with final decisions and reviewer guidance

## Developer Handoff Checklist

- [x] Confirm `go test ./cmd/inspector/... -count=1` still passes (no regression in existing inspector)
- [x] Confirm new code path compiles/runs via `go run ./cmd/smalltalk-inspector <file.js>`
- [x] Ensure reusable components are imported/adapted rather than duplicated from `cmd/inspector/app`
- [x] Record each meaningful milestone in diary + changelog with exact commands and outcomes
