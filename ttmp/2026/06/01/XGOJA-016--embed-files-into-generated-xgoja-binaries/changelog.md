# Changelog

## 2026-06-01

- Initial workspace created


## 2026-06-01

Created XGOJA-016 research/design package for arbitrary embedded assets and fs module exposure; mapped current xgoja generation/runtime/fs architecture and added an investigation script.

### Related Files

- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/ttmp/2026/06/01/XGOJA-016--embed-files-into-generated-xgoja-binaries/design-doc/01-embedding-files-into-xgoja-binaries.md — Primary implementation guide and design rationale.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/ttmp/2026/06/01/XGOJA-016--embed-files-into-generated-xgoja-binaries/reference/01-investigation-diary.md — Chronological investigation diary.


## 2026-06-01

Validated XGOJA-016 docs with docmgr doctor and uploaded the bundled design package to reMarkable at /ai/2026/06/01/XGOJA-016.

### Related Files

- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/ttmp/2026/06/01/XGOJA-016--embed-files-into-generated-xgoja-binaries/design-doc/01-embedding-files-into-xgoja-binaries.md — Uploaded as part of the reMarkable bundle.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/ttmp/2026/06/01/XGOJA-016--embed-files-into-generated-xgoja-binaries/reference/01-investigation-diary.md — Uploaded as part of the reMarkable bundle.


## 2026-06-01

Updated the investigation diary with validation and reMarkable delivery details.

### Related Files

- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/ttmp/2026/06/01/XGOJA-016--embed-files-into-generated-xgoja-binaries/reference/01-investigation-diary.md — Step 2 records doctor and reMarkable upload results.


## 2026-06-01

Updated the embedded assets design to make multiple fs module instances under distinct aliases (fs:assets and fs:host) the primary API.

### Related Files

- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/ttmp/2026/06/01/XGOJA-016--embed-files-into-generated-xgoja-binaries/design-doc/01-embedding-files-into-xgoja-binaries.md — Runtime fs configuration now documents per-alias module instances.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/ttmp/2026/06/01/XGOJA-016--embed-files-into-generated-xgoja-binaries/reference/01-investigation-diary.md — Step 3 records the API alignment.


## 2026-06-01

Phase 1 task 7: added asset spec structs and buildspec validation; focused buildspec tests pass.

### Related Files

- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/cmd/xgoja/internal/buildspec/spec.go — Build-time assets schema.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/cmd/xgoja/internal/buildspec/validate.go — Asset validation.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/pkg/xgoja/app/spec.go — Runtime assets schema mirror.


## 2026-06-01

Phase 1 task 8: added generator support for copying embedded assets, rewriting paths, rendering embeddedAssets, and focused generator tests.

### Related Files

- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/cmd/xgoja/internal/generate/generate.go — Asset copy pipeline.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/cmd/xgoja/internal/generate/main.go — Asset root rewriting.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/cmd/xgoja/internal/generate/templates/main.go.tmpl — Generated asset embed declaration.


## 2026-06-01

Phase 2 task 9: added app AssetStore/HostServices and passed ModuleContext.Host into provider module factories; app and focused generator tests pass.

### Related Files

- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/pkg/xgoja/app/assets.go — Asset store and host services.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/pkg/xgoja/app/factory.go — ModuleContext.Host plumbing.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/pkg/xgoja/providerapi/module.go — Host services contract.


## 2026-06-01

Phase 3 task 10: refactored modules/fs behind a Backend abstraction while preserving OS fs behavior; modules/fs tests pass after fixing rm force semantics.

### Related Files

- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/modules/fs/backend.go — Backend abstraction.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/modules/fs/fs.go — Backend-backed JS exports.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/modules/fs/fs_async.go — Backend-backed async helpers.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/modules/fs/fs_sync.go — OSBackend implementation.


## 2026-06-01

Phase 3 task 11: added read-only embedded fs backend with virtual mounts, EROFS errors, and sync/async module tests.

### Related Files

- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/modules/fs/backend_embed.go — Embedded backend implementation.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/modules/fs/fs_embed_test.go — Embedded fs behavior tests.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/modules/fs/fs_errors.go — EROFS error code mapping.


## 2026-06-01

Phase 4 task 12: extended host provider fs config to support separate fs:assets and fs:host aliases; host/app/fs tests pass.

### Related Files

- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/pkg/xgoja/providers/host/host.go — Provider config implementation.
- /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja/pkg/xgoja/providers/host/host_test.go — Alias behavior tests.

