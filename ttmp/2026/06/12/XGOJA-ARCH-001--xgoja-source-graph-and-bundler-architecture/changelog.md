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

