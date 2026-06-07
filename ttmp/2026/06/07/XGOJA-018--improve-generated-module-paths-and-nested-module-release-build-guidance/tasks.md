# Tasks

## TODO

- [x] **Task 1: Commit ticket planning baseline** — Commit XGOJA-018 ticket docs, local `.ttmp.yaml`, vocabulary updates, tasks, changelog, and diary before implementation code changes
- [x] **Task 2: Change default module path** — Change `applyDefaults()` in `buildspec/load.go` to use `xgoja.generated/`; update only default-dependent tests, and keep explicit-module fixtures unchanged unless intentionally modeling the default
- [x] **Task 3: Add explicit-module/defaulting tests** — Add or tighten tests proving missing `go.module` defaults to `xgoja.generated/<name>` and explicit `go.module` values are preserved
- [ ] **Task 4: Improve build workspace output** — Extend the existing `generated build workspace` output in `cmd/xgoja/cmd_build.go`; do not put user-facing output in `internal/buildexec`
- [ ] **Task 5: Update command-output tests** — Update `cmd/xgoja/root_test.go` or equivalent tests for the new guidance text
- [ ] **Task 6: Update xgoja docs** — Add release packaging/nested-module guidance and explicit `go.module` examples to `02-user-guide.md`, `06-buildspec-reference.md`, and any tutorial examples that teach the default
- [ ] **Task 7: Validate implementation** — Run unit tests for xgoja defaulting/rendering/output; smoke `xgoja build --keep-work`; run goja-bleve vector/GoReleaser checks only when FAISS/CGO/cross-compiler environment supports them
- [ ] **Task 8: Final diary/bookkeeping commit** — Update diary, tasks, changelog, related files, and docmgr doctor after implementation

