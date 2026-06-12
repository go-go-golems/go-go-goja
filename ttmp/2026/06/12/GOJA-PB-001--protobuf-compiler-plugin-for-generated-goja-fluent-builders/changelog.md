# Changelog

## 2026-06-12

- Initial workspace created


## 2026-06-12

Created reusable protobuf-to-Goja fluent builder generator ticket and deep design covering protogoja runtime helpers, protoc-gen-goja-builder, generated APIs, TypeScript declarations, field mapping, and acceptance tests.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/12/GOJA-PB-001--protobuf-compiler-plugin-for-generated-goja-fluent-builders/design-doc/01-generated-goja-protobuf-fluent-builders-design.md — Primary generated-builder design document
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/12/GOJA-PB-001--protobuf-compiler-plugin-for-generated-goja-fluent-builders/reference/01-investigation-diary.md — Investigation diary for the ticket


## 2026-06-12

Validated GOJA-PB-001 docs with docmgr doctor and uploaded the design bundle to reMarkable at /ai/2026/06/12/GOJA-PB-001.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/12/GOJA-PB-001--protobuf-compiler-plugin-for-generated-goja-fluent-builders/design-doc/01-generated-goja-protobuf-fluent-builders-design.md — Uploaded as part of reMarkable bundle
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/12/GOJA-PB-001--protobuf-compiler-plugin-for-generated-goja-fluent-builders/reference/01-investigation-diary.md — Uploaded as part of reMarkable bundle


## 2026-06-12

Expanded GOJA-PB-001 into detailed implementation phases and task checklists, starting with a narrow pkg/protogoja runtime foundation phase.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/12/GOJA-PB-001--protobuf-compiler-plugin-for-generated-goja-fluent-builders/reference/01-investigation-diary.md — Diary updated with phase-planning step
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/12/GOJA-PB-001--protobuf-compiler-plugin-for-generated-goja-fluent-builders/tasks.md — Phase-by-phase implementation checklist


## 2026-06-12

Implemented Phase 1 protogoja runtime foundation: hidden ProtoMessage refs, direct MessageFromValue extraction, JS typeName/toJSON/clone/equals methods, and focused tests passing with go test ./pkg/protogoja -count=1.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/ref.go — MessageRef runtime wrapper and extraction helpers
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/ref_test.go — Phase 1 tests for runtime wrapper behavior


## 2026-06-12

Phase 1 committed: protogoja MessageRef runtime foundation with hidden ProtoMessage refs and tests (commit eb0269d1874a68867c7d70080fc66c797246dbf5).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/logcopter.go — Generated package logger included by pre-commit workflow
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/ref.go — Committed Phase 1 runtime wrapper
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/ref_test.go — Committed Phase 1 tests


## 2026-06-12

Implemented first Phase 2 BuilderRef slice with Set/Add/Put/Clear/Build/Clone, scalar and enum conversions, repeated field support, ProtoMessage field conversion, and passing protogoja tests.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder.go — BuilderRef runtime helper implementation
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder_test.go — BuilderRef runtime helper tests


## 2026-06-12

Phase 2 slice committed: BuilderRef lifecycle and initial scalar/enum/repeated/message conversions (commit 05f5bd484bf028c27429c6f108ec944d45413d95).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder.go — Committed BuilderRef runtime helper slice
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder_test.go — Committed BuilderRef tests

