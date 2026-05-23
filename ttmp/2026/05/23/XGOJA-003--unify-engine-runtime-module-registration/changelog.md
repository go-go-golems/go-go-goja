# Changelog

## 2026-05-23

- Initial workspace created


## 2026-05-23

Created runtime-aware module registration ticket, design/migration guide, diary, and implementation tasks.

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/23/XGOJA-003--unify-engine-runtime-module-registration/design-doc/01-runtime-aware-module-registration-design-and-implementation-guide.md — Design and migration guide
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/23/XGOJA-003--unify-engine-runtime-module-registration/reference/01-diary.md — Implementation diary


## 2026-05-23

Implemented hard cutover to RuntimeModuleSpec, migrated runtime module call sites, adapted xgoja to engine.Runtime, updated public docs, and validated focused tests/examples (commits 29e07e7, 7ff4dac).

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/factory.go — Registers runtime-aware modules and removes WithRuntimeModuleRegistrars
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/runtime_modules.go — Defines unified RuntimeModuleSpec contract
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/doc/13-plugin-developer-guide.md — Updates public runtime module API references
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/factory.go — Adapts xgoja provider modules into engine.Runtime


## 2026-05-23

Hardened xgoja engine reuse so generated runtimes disable implicit engine default modules and added regression tests for explicit module exposure (commit 5911a8d).

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/options.go — Adds explicit controls for implicit default module exposure
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/factory.go — Disables implicit engine defaults for xgoja-generated runtimes
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/root_test.go — Verifies xgoja exposes only spec-selected modules

