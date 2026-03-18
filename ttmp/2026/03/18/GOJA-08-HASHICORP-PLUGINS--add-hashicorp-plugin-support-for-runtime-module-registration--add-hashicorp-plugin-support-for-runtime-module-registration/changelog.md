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
