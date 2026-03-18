# Changelog

## 2026-03-18

- Initial workspace created

## 2026-03-18

Created GOJA-13 to review the full `origin/main..HEAD` plugin and docs branch. The main deliverable is a cleanup-oriented architecture review report that identifies the highest-leverage maintainability issues in runtime state ownership, entrypoint duplication, legacy docs surfaces, validation drift, and plugin diagnostics.

## 2026-03-18

Validated GOJA-13 with `docmgr doctor`, uploaded the review bundle to reMarkable at `/ai/2026/03/18/GOJA-13-ORIGIN-MAIN-ARCHITECTURE-REVIEW`, and verified the remote listing `GOJA-13 Origin main architecture review`.

## 2026-03-18

Added a focused implementation plan for registrar state ownership and plugin diagnostics/cancellation hardening, then expanded the GOJA-13 task list into an execution backlog. Also marked the first two cleanup slices for implementation: remove the legacy `glazehelp` module and centralize duplicated HashiPlugin manifest validation.

## 2026-03-18

Implemented the first GOJA-13 cleanup slices. The legacy `modules/glazehelp` path was removed from default runtime composition and from `repl`, and shared plugin manifest validation was extracted into `pkg/hashiplugin/contract/validate.go` so the host and SDK no longer duplicate namespace/export/method shape checks.

## 2026-03-18

Implemented runtime-scoped value persistence. Registrar-produced values now survive onto `engine.Runtime`, runtime initializers can read and write the same value map, and the docs registrar now stores its `docaccess.Hub` as runtime-owned state alongside the plugin registrar's loaded-module snapshot.

## 2026-03-18

Completed the rest of GOJA-13 Phase 7. Owned runtimes now carry a lifecycle context used by plugin invocations, plugin startup/load errors preserve bounded stderr diagnostics and error-first report summaries, and the plugin setup path is shared across `repl`, `js-repl`, and `bun-demo` through `pkg/hashiplugin/host/runtime_setup.go`.
