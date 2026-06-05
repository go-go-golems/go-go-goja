# Changelog

## 2026-06-04

- Initial workspace created


## 2026-06-04

Created GOJA-066 design package for simplifying xgoja to a single runtime module set; documented current architecture, proposed schema/API changes, implementation phases, and validation plan.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-066--simplify-xgoja-to-a-single-runtime-profile/design-doc/01-single-runtime-profile-simplification-design-and-implementation-guide.md — Primary design guide
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-066--simplify-xgoja-to-a-single-runtime-profile/reference/01-diary.md — Chronological investigation diary


## 2026-06-04

Validated GOJA-066 and uploaded design/diary bundle to reMarkable at /ai/2026/06/04/GOJA-066.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-066--simplify-xgoja-to-a-single-runtime-profile/design-doc/01-single-runtime-profile-simplification-design-and-implementation-guide.md — Uploaded primary design guide
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-066--simplify-xgoja-to-a-single-runtime-profile/reference/01-diary.md — Uploaded diary and recorded upload result


## 2026-06-04

Implemented core single-runtime xgoja schema/runtime refactor and updated examples/tests; focused xgoja tests pass.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/buildspec/build_spec.go — BuildSpec now uses top-level modules
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/examples/xgoja/01-core-provider/xgoja.yaml — Example migrated to top-level modules
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/factory.go — RuntimeFactory now creates from one module set
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/root.go — Built-in commands no longer select runtime profiles


## 2026-06-04

Updated xgoja docs and renamed example 03 to single-runtime-modules; focused xgoja tests and renamed example smoke pass.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/doc/02-user-guide.md — User guide migrated to top-level modules
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/examples/xgoja/03-single-runtime-modules/README.md — Renamed example documenting the one module set model
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-066--simplify-xgoja-to-a-single-runtime-profile/reference/01-diary.md — Recorded documentation migration step


## 2026-06-04

Refreshed GOJA-066 reMarkable bundle after implementation and documentation commits.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-066--simplify-xgoja-to-a-single-runtime-profile/reference/01-diary.md — Recorded final upload and commit hashes

