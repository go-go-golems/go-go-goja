# Changelog

## 2026-04-28

- Initial workspace created


## 2026-04-28

Step 1: Investigation complete. Confirmed all dangerous modules (fs, exec, database, os, yaml) are already loaded by run/tui but there is no way to control this. Engine has all APIs needed. Design doc written.

### Related Files

- /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/cmd/goja-repl/cmd_run.go — Run command
- /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/cmd/goja-repl/root.go — Shared app construction
- /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/engine/module_specs.go — Module spec APIs


## 2026-04-28

Step 1b: Uploaded ticket docs (design, diary, tasks, changelog) to reMarkable at /ai/2026/04/28/GOJA-059


## 2026-04-28

Step 2: Design refined to ModuleMiddleware pipeline pattern (f(next Handler) Handler). Design doc and diary updated. Old strategy-enum approach replaced with composable override+transform middlewares.

### Related Files

- /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/ttmp/2026/04/28/GOJA-059--add-granular-module-enablement-flags-to-goja-repl-commands/design/01-module-enablement-design.md — Updated design doc with middleware pipeline pattern
- /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/ttmp/2026/04/28/GOJA-059--add-granular-module-enablement-flags-to-goja-repl-commands/reference/01-diary.md — Updated diary with Step 2 design evolution


## 2026-04-28

Step 3: Implemented engine/module_middleware.go with ModuleSelector, ModuleMiddleware, built-in middlewares (Safe, Only, Exclude, Add, Custom), Pipeline helper, and comprehensive unit tests. Added UseModuleMiddleware to FactoryBuilder. Deprecated old API family. Migrated all callers. (commit 9148dd6)

### Related Files

- /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/engine/factory.go — UseModuleMiddleware integration in FactoryBuilder
- /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/engine/module_middleware.go — Core middleware types and built-ins
- /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/engine/module_specs.go — Deprecated old API family

