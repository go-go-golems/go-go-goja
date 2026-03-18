# Changelog

## 2026-03-18

- Initial workspace created

## 2026-03-18

Created GOJA-09 for the plugin authoring SDK workstream, inspected the current plugin contract/host/example code, wrote an intern-oriented SDK architecture guide, validated the ticket with `docmgr doctor`, and prepared the bundle for reMarkable delivery.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins--create-a-plugin-authoring-sdk-layer-for-hashicorp-go-go-goja-plugins/design-doc/01-plugin-authoring-sdk-layer-for-hashicorp-go-go-goja-plugins-architecture-and-implementation-guide.md — Primary intern-facing design deliverable for the SDK proposal
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins--create-a-plugin-authoring-sdk-layer-for-hashicorp-go-go-goja-plugins/reference/01-investigation-diary.md — Chronological investigation record and delivery evidence
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins--create-a-plugin-authoring-sdk-layer-for-hashicorp-go-go-goja-plugins/tasks.md — Completed checklist for the ticket deliverable

## 2026-03-18

Expanded GOJA-09 from a research-only ticket into a detailed implementation backlog for the richer, more explicit SDK surface. The task list now breaks the work into package skeleton, manifest generation, dispatch/conversion, serve wrapper, example migration, integration testing, documentation, and closeout phases.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins--create-a-plugin-authoring-sdk-layer-for-hashicorp-go-go-goja-plugins/tasks.md — Detailed richer-SDK execution plan for the next implementation phase

## 2026-03-18

Implemented the first GOJA-09 code slice: the new `pkg/hashiplugin/sdk` package with explicit module/export builders, manifest generation, invoke dispatch, argument/result conversion helpers, the `Serve(...)` wrapper, and SDK-focused tests that prove compatibility with both host validation and the shared gRPC transport.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/sdk/module.go — Core `Module` type, metadata helpers, validation, and manifest generation
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/sdk/export.go — Declarative `Function`, `Object`, and `Method` builders
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/sdk/call.go — Author-facing call helper API
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/sdk/convert.go — Plain Go value to/from `structpb.Value` helpers
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/sdk/dispatch.go — Invoke routing implementation
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/sdk/serve.go — Thin serve wrapper over the existing shared transport
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/sdk/sdk_test.go — Manifest, dispatch, conversion, and gRPC compatibility tests

## 2026-03-18

Migrated the user-facing example plugin and the positive test fixture to the new SDK, kept the invalid fixture handwritten for raw-contract coverage, added host integration coverage for an SDK-authored example, and rewrote the plugin help pages to teach the SDK-based authoring path.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/plugins/examples/greeter/main.go — Example plugin now uses `sdk.MustModule(...)` and `sdk.Serve(...)`
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/plugins/examples/README.md — Example README now describes the richer SDK surface
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/plugins/testplugin/echo/main.go — Positive fixture now exercises the SDK path too
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/registrar_test.go — Added SDK-authored example integration coverage
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/doc/12-plugin-user-guide.md — User guide now points authors at the SDK-backed example
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/doc/13-plugin-developer-guide.md — Developer guide now describes the `sdk` layer explicitly
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/doc/14-plugin-tutorial-build-install.md — Tutorial now teaches the SDK-based authoring path

## 2026-03-18

Closed out GOJA-09 by syncing the checklist and diary with the landed SDK work, rerunning the full validation pass, and refreshing the reMarkable bundle so the external deliverable matches the final branch state.

### Related Files

- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/sdk/module.go — Shipped author-facing module builder and manifest logic
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/sdk/export.go — Shipped declarative function/object/method builder layer
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/sdk/dispatch.go — Shipped invoke routing implementation
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/plugins/examples/greeter/main.go — Example plugin proving the SDK is the recommended authoring path
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/doc/12-plugin-user-guide.md — User-facing plugin guidance updated to point at the SDK
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/doc/13-plugin-developer-guide.md — Developer-facing layering guide updated for the SDK
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins--create-a-plugin-authoring-sdk-layer-for-hashicorp-go-go-goja-plugins/reference/01-investigation-diary.md — Final diary with validation and upload evidence
- /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins--create-a-plugin-authoring-sdk-layer-for-hashicorp-go-go-goja-plugins/tasks.md — Fully checked-off execution checklist
