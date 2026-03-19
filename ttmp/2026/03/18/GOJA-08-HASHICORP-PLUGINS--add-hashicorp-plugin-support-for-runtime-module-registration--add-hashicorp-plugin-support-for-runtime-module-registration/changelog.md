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

## 2026-03-18

Added the shared HashiCorp plugin transport scaffold: dependency, protobuf contract, generated bindings, and the shared gRPC adapter used by both host and plugin subprocesses.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/go.mod — Added direct dependency on github.com/hashicorp/go-plugin
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/contract/jsmodule.proto — Defined the JS module manifest and invoke RPC schema
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/contract/jsmodule.pb.go — Generated protobuf bindings for the contract
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/contract/jsmodule_grpc.pb.go — Generated gRPC bindings for the contract
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/shared/plugin.go — Added shared handshake, plugin set helpers, and GRPCPlugin adapter
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/shared/plugin_test.go — Added a round-trip gRPC dispense test for the adapter

## 2026-03-18

Implemented the host-side plugin loading path, added test plugins plus integration tests, and wired plugin directories into runtime creation for both the basic REPL and the reusable evaluator configuration.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/config.go — Plugin discovery/runtime config defaults
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/discover.go — Discovery and executable filtering
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/validate.go — Manifest validation rules
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/client.go — Plugin client startup, manifest fetch, and lifecycle management
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/reify.go — CommonJS module reification into goja
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/registrar.go — Runtime registrar that plugs host loading into the engine seam
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/registrar_test.go — End-to-end plugin loading and cleanup tests
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/testplugin/echo/main.go — Example valid plugin used by integration tests
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/testplugin/invalid/main.go — Invalid plugin fixture used by validation tests
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go — Evaluator config now accepts plugin directories
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/cmd/repl/main.go — REPL now accepts `--plugin-dir`

## 2026-03-18

Ran the full repository test suite, reran `docmgr doctor`, updated the GOJA-08 task/diary state with concrete implementation commits, and prepared the refreshed ticket bundle for reMarkable publication.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/tasks.md — Final task state for the implementation sequence
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/reference/01-diary.md — Diary now records the implementation commit hashes and closeout validation step
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/changelog.md — Closeout entry for validation and publication

## 2026-03-18

Started the GOJA-08 productization pass by separating user-facing sample plugins from integration-test fixtures. Added a new example plugin plus README under `plugins/examples`, and updated the help docs to teach from the example path instead of the test fixture path.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/plugins/examples/README.md — New operator-facing entrypoint for plugin examples
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/plugins/examples/greeter/main.go — New sample plugin intended to be copied and built manually
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/doc/12-plugin-user-guide.md — User guide now points at the example plugin
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/doc/13-plugin-developer-guide.md — Developer guide now distinguishes examples from test fixtures
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/doc/14-plugin-tutorial-build-install.md — Tutorial now uses the example plugin path

## 2026-03-18

Added plugin discovery visibility to the REPL surfaces. The host registrar can now record discovery/load reports, `repl` exposes `:plugins` and `--plugin-status`, `js-repl` exposes `--plugin-status`, and the help docs now describe the diagnostics surface.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/report.go — New runtime load report and CLI-friendly formatting helpers
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/registrar.go — Registrar now records discovered and loaded modules
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/cmd/repl/main.go — Added startup summaries, `:plugins`, and `--plugin-status`
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/cmd/js-repl/main.go — Added `--plugin-status` and TUI plugin summary wiring
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go — Evaluator config now accepts a plugin report collector

## 2026-03-18

Added allowlist-style trust-policy knobs to the supported entrypoints. `repl`, `js-repl`, and the shared evaluator config now accept explicit plugin-module allowlists, and the host integration tests cover rejection when a loaded module is outside the requested allowlist.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/cmd/repl/main.go — Added `--allow-plugin-module`
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/cmd/js-repl/main.go — Added `--allow-plugin-module`
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go — Evaluator config now forwards plugin allowlists
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/config.go — Allowlist values are normalized centrally
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/registrar_test.go — Added integration coverage for allowlist rejection

## 2026-03-18

Wired plugin configuration into `cmd/bun-demo` so the plugin system is no longer just a REPL feature. The Bun/CommonJS demo now accepts the same `--plugin-dir` and `--allow-plugin-module` controls, and the bundling playbook now documents how bundled code can require plugin-backed modules.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/cmd/bun-demo/main.go — Bun demo now opts into plugin discovery and allowlisting
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/doc/bun-goja-bundling-playbook.md — Bundling guide now documents plugin-backed runtime extensions
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/doc/13-plugin-developer-guide.md — Architecture guide now lists `cmd/bun-demo` as a wired entrypoint

## 2026-03-18

Refreshed the GOJA-08 ticket after the productization pass: cleaned the task list, updated the diary with the later commit hashes, reran the full repository and ticket validation commands, and uploaded the updated bundle back to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/tasks.md — Final executed task sequence across the design, implementation, and productization phases
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/reference/01-diary.md — Diary now records the later productization commits and final closeout step
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/changelog.md — Changelog reflects the refreshed ticket validation and publication pass
