# Changelog

## 2026-06-12

- Initial workspace created


## 2026-06-12

Created architecture design for reframing xgoja as a Go-backed JavaScript runtime compiler with source graph, import resolver, provider graph, build plan, and runtime plan.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/design/01-xgoja-source-graph-and-bundler-architecture.md — Primary architecture design
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/reference/01-investigation-diary.md — Investigation diary


## 2026-06-12

Updated architecture design v2 with Go workspace resolution for generated builds and DTS sidecars, including go.work discovery, GoModulePlan, precedence rules, doctor output, and implementation phases.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/generate/gomod.go — Current generated module rendering target
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/design/01-xgoja-source-graph-and-bundler-architecture.md — Architecture v2 workspace section


## 2026-06-12

Added second architecture document for xgoja v2 spec and migration tooling, treating v1 as migratable legacy input and v2 as the native planner schema.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/design/02-xgoja-v2-spec-and-migration-architecture.md — V2 spec and migration architecture


## 2026-06-12

Simplified the v2 spec design around goja-executed source only: removed broad bundler/package-manager knobs, kept intent-level compile fields, and documented external frontend bundles as assets.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/design/02-xgoja-v2-spec-and-migration-architecture.md — Simplified v2 spec design


## 2026-06-12

Expanded tasks into a phase-by-phase hard-cutover plan for v2 xgoja, including migration tooling, workspace resolution, provider graph, source graph, command cutover, docs/examples, and v1 removal.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/reference/01-investigation-diary.md — Diary Step 5 task planning entry
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/tasks.md — Detailed v2 hard-cutover task list


## 2026-06-12

Implemented initial specv2 package with DTOs, defaults, strict loading, validation, rendering, tests, v1 migration diagnostic, and v1 spec inventory.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/specv2/specv2_test.go — Initial v2 schema test coverage
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/specv2/types.go — Initial v2 config DTOs


## 2026-06-12

Added v1-to-v2 migration conversion for providers, runtime modules, command surfaces, jsverbs, TypeScript compile intent, help/assets, target artifacts, replacements, and migration warnings.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/specv2/migrate_v1.go — V1-to-v2 conversion implementation
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/specv2/migrate_v1_test.go — Migration coverage for TypeScript jsverbs


## 2026-06-12

Added xgoja migrate-spec command with v1-to-v2 output, in-place migration, backups, check mode, warning output, and root command coverage.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/cmd_migrate_spec.go — migrate-spec CLI implementation
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/root.go — Wires migrate-spec into xgoja root
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/root_test.go — Root command migration output and in-place backup tests

