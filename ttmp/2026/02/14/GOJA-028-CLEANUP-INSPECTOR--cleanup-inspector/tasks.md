# Tasks

## TODO

- [x] Create GOJA-028 ticket workspace
- [x] Inventory GOJA-024 to GOJA-027 tickets, commits, and touched files
- [x] Run validation baseline (`go test ./...`, `go vet ./...`) and record outcomes
- [x] Reproduce and document critical inheritance recursion crash (`class A extends A {}`)
- [x] Author deep cleanup/review document with severity-ranked findings and architecture map

- [x] Implement cycle detection in inherited-member traversal and add regression test
- [x] Add inspect/stack scroll offsets and visibility guarantees
- [x] Fix source scroll bounds when showing REPL source
- [x] Rebuild REPL syntax spans on runtime fallback source append
- [ ] Replace command parsing with quoted-argument-safe parser for `:load`
- [x] Replace magic binding-kind numeric literals with typed constants
- [x] Introduce shared inspector UI components and migrate smalltalk panes incrementally
- [x] Integrate `pkg/inspector/analysis` into smalltalk-inspector command path
- [ ] Optimize syntax highlighting lookup/render path and add benchmark coverage
- [x] Close GOJA-027 doc/task hygiene gaps and validate with `docmgr doctor`

## Handoff Checklist

- [x] Confirm no regression in `cmd/inspector` while applying cleanup refactors
- [x] Keep each cleanup change small and test-backed
- [x] Update GOJA-028 changelog + diary (or equivalent notes) per milestone
