# Changelog

## 2026-03-18

- Initial workspace created
- Created GOJA-08, imported the plugin source memo, and mapped the current go-go-goja runtime/module architecture.
- Wrote an intern-facing design and implementation guide that keeps goja ownership in the host, proposes runtime-scoped plugin registration, and identifies the required engine lifecycle refactors.

## 2026-03-18

Created the GOJA-08 design ticket, imported the source memo, wrote the plugin architecture guide, validated it with docmgr doctor, and uploaded the bundle to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/design-doc/01-hashicorp-plugin-support-for-go-go-goja-architecture-and-implementation-guide.md — Primary intern-facing design deliverable
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/reference/01-diary.md — Chronological design diary

## 2026-03-18

Implemented the engine lifecycle refactor for plugin support and verified the engine package in isolation.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go — Factory now rebuilds a fresh require registry per runtime and executes runtime-scoped registrars
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/runtime.go — Runtime now supports ordered cleanup hooks
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/runtime_modules.go — New runtime-scoped module registrar interface and context
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/runtime_modules_test.go — Added tests covering per-runtime registration and close-hook behavior
