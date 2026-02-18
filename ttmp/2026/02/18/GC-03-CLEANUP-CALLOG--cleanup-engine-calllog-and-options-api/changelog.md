# Changelog

## 2026-02-18

- Initial workspace created.
- Added implementation plan, execution tasks, and diary baseline.

## 2026-02-18

- Commit `f8e2142e44f9dbe335f06ed0ff251c192c0d8c99`: added option-based `engine.Open(...)` API and runtime-scoped calllog binding, with compatibility wrappers preserved for existing constructors.
- Added targeted tests for engine option behavior and runtime/default calllog routing.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/engine/options.go — New option definitions for `Open`
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/engine/runtime.go — Unified constructor flow + runtime calllog setup
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/pkg/calllog/calllog.go — Runtime-scoped logger bindings and routing
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/engine/runtime_test.go — Engine option/calllog behavior tests
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/pkg/calllog/calllog_test.go — Calllog runtime binding tests

## 2026-02-18

- Commit `12b2fcf12f0d4d656e157fd5156417d0202f0a34`: added graceful `serve` shutdown path so Ctrl-C triggers clean server shutdown instead of hanging.
- Added focused tests for cancellation and listen error handling in server runner helper.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/cmd/goja-perf/serve_command.go — Signal-aware shutdown wiring
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/cmd/goja-perf/serve_shutdown.go — Shared serve lifecycle helper
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/cmd/goja-perf/serve_shutdown_test.go — Graceful shutdown tests
