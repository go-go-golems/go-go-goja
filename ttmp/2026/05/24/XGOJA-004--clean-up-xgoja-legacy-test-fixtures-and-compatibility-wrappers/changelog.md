# Changelog

## 2026-05-24

- Initial workspace created


## 2026-05-24

Created hard-cutover cleanup ticket, tasks, implementation guide, and diary.

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/24/XGOJA-004--clean-up-xgoja-legacy-test-fixtures-and-compatibility-wrappers/design-doc/01-cleanup-implementation-guide.md — Cleanup guide
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/24/XGOJA-004--clean-up-xgoja-legacy-test-fixtures-and-compatibility-wrappers/reference/01-diary.md — Diary


## 2026-05-24

Step 2: Deleted legacy internal xgoja testprovider and tracked .orig artifact files.

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/xgoja/internal/testprovider/provider.go — Deleted legacy fixture provider
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/config.go.orig — Deleted tracked merge artifact
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/runtime.go.orig — Deleted tracked stale compatibility artifact
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/modules/exports.go.orig — Deleted tracked stale artifact


## 2026-05-24

Step 3: Removed obsolete jsverbs InvokeInGojaRuntime lightweight invocation API and direct runtime test.

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbs/runtime.go — Removed obsolete bare-Goja invocation path
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbs/runtime_direct_test.go — Deleted direct runtime test for removed API


## 2026-05-24

Step 4: Hard-cutover removed deprecated engine DefaultRegistry* wrappers and modules.EnableAll; updated docs/tests to middleware-based module selection.

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/factory.go — Switched internal factory default module construction to private helpers
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/granular_modules_test.go — Updated tests to exercise MiddlewareOnly instead of deprecated wrappers
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/module_specs.go — Removed exported deprecated DefaultRegistry helpers and kept private module specs
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/modules/common.go — Removed modules.EnableAll compatibility helper
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/doc/16-nodejs-primitives.md — Updated public module-selection guidance


## 2026-05-24

Completed XGOJA-004 cleanup and added cleanup topic vocabulary (commits 798bd71, 70c96de, c93c30d, 5f90179).

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/24/XGOJA-004--clean-up-xgoja-legacy-test-fixtures-and-compatibility-wrappers/reference/01-diary.md — Finalized commit references
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/vocabulary.yaml — Added cleanup topic


## 2026-05-24

Ticket closed

