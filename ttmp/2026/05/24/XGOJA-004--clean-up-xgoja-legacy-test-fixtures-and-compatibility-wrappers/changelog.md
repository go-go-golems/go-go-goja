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

