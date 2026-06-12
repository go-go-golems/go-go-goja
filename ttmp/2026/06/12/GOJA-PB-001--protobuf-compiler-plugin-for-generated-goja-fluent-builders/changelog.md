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


## 2026-06-12

Implemented builder-ref message-field conversion so generated builder objects can be accepted wherever ProtoMessage refs are accepted.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder.go — Defines hidden BuilderRef attachment/extraction and message-field conversion from builder refs
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder_test.go — Tests generated-style builder refs as message-field inputs


## 2026-06-12

Added protoc-gen-goja-builder skeleton with option parsing and a golden test that generates the first companion Go file for a synthetic protobuf descriptor (commit b3deaf4).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/internal/generator/generator.go — Emits one *_goja.pb.go companion file per generated proto file
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/internal/generator/options.go — Parses Phase 4 plugin options
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/main.go — Protoc plugin entry point using protogen
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/main_test.go — Golden test harness that creates a CodeGeneratorRequest and verifies generated Go output
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/testdata/fixture_goja.pb.go.golden — First generated Go companion file golden output


## 2026-06-12

Added the Phase 4 compile test that writes generated companion output into a temporary Go module and runs go test ./... (commit 43a13e4).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/main_test.go — Compile-style generated companion file validation


## 2026-06-12

Generated initial Phase 5 Goja protobuf APIs: per-message namespace functions, builder constructors, fluent field setters, clear/build/clone helpers, and nested message support (commit 2100678).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/internal/generator/generator.go — Generates initial Goja namespace and builder APIs
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/main_test.go — Golden and compile tests for generated builder API output
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/testdata/fixture_goja.pb.go.golden — Expanded golden output with generated namespace and builder functions


## 2026-06-12

Added runtime Goja validation for generated protobuf builders, proving generated fluent calls build a concrete proto.Message recoverable through protogoja.MessageFromValue (commit db84885).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/main_test.go — Runtime generated builder test written into the temporary compile fixture

