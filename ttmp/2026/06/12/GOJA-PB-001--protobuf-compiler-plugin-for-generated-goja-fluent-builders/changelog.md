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


## 2026-06-12

Generated Goja enum export objects and validated enum setter conversion through generated runtime tests (commit 5495137).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/internal/generator/generator.go — Generates New<Enum>GojaEnum objects with enum value properties
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/main_test.go — Runtime test passes generated enum values into generated enum field setters
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/testdata/fixture_goja.pb.go.golden — Golden enum export output


## 2026-06-12

Added generated message prototype tokens: protogoja namespace prototype refs, generated namespace attachment, and runtime extraction validation (commit bcaf348).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/internal/generator/generator.go — Generated namespace prototype attachment
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/main_test.go — Generated runtime test extracts prototype tokens
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/prototype.go — Runtime MessagePrototypeRef and hidden namespace token attachment
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/prototype_test.go — Prototype token extraction and clone tests


## 2026-06-12

Closed Phase 5 generated fluent builders: namespace exports, builder methods, enum exports, nested messages, prototype tokens, and runtime generated-builder validation are complete through commit bcaf348.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/internal/generator/generator.go — Phase 5 generated fluent builder implementation
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/prototype.go — Phase 5 prototype token runtime support


## 2026-06-12

Generated TypeScript RawDTS descriptors for protobuf builder modules, validated through tsgen/render and xgoja/dtsgen, and documented usage examples (commit 611dfc7).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/internal/generator/generator.go — Generates TypeScriptModule with RawDTS
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/main_test.go — Validates generated RawDTS through tsgen/render and xgoja/dtsgen
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/12/GOJA-PB-001--protobuf-compiler-plugin-for-generated-goja-fluent-builders/design-doc/01-generated-goja-protobuf-fluent-builders-design.md — Documents generated TypeScript declaration examples


## 2026-06-12

Generated host integration helpers for protobuf builder modules: require loaders, native module wrappers, TypeScript-capable modules, registry registration, and message-type registration callbacks (commit 0342d57).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/internal/generator/generator.go — Generates host integration helper functions
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/main_test.go — Validates generated loader/module/message registration helpers


## 2026-06-12

Implemented Phase 2 map builder helpers: object and JavaScript Map replacement input, Put, Delete, Clear semantics, atomic replacement validation, and dynamic-protobuf tests (commit 6411e52).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder.go — Added BuilderRef.Delete plus object/Map entry normalization for map fields (commit 6411e52)
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder_test.go — Added dynamic map descriptor tests for object input


## 2026-06-12

Implemented Phase 2 oneof runtime helpers: BuilderRef.WhichOneof, BuilderRef.ClearOneof, descriptor validation, and dynamic-protobuf tests for selection switching and clearing (commit bc3949c).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder.go — Added oneof validation plus WhichOneof and ClearOneof helpers (commit bc3949c)
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder_test.go — Added dynamic oneof descriptor tests for unset


## 2026-06-12

Implemented Phase 2 optional presence helpers: BuilderRef.Has, generated has<Field>() methods for fields with explicit presence, TypeScript declarations, and dynamic proto3 optional tests (commit 0205764).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/protoc-gen-goja-builder/internal/generator/generator.go — Generated has<Field>() methods and DTS entries for fields with explicit presence (commit 0205764)
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder.go — Added BuilderRef.Has for protobuf presence checks (commit 0205764)
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder_test.go — Added dynamic proto3 optional presence tests (commit 0205764)


## 2026-06-12

Completed Phase 2 builder runtime helpers: added field-path-rich repeated/map conversion errors, validated pkg/protogoja and generator runtime tests, and closed the Phase 2 runtime helper milestone (commit f98cdf0; prior Phase 2 commits 6411e52, bc3949c, 0205764).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder.go — Added contextual repeated/map error wrappers and completed Phase 2 runtime helpers (commit f98cdf0)
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder_test.go — Added field-path-rich error tests for repeated indexes and map keys (commit f98cdf0)


## 2026-06-12

Completed Phase 3 well-known type support: Timestamp RFC3339/Date, Duration strings, Any wrapping, Struct/Value/ListValue JSON-shaped input, wrapper scalars, FieldMask input, strict plain-object rules, focused tests, and Phase 3 commit (commit b65868b).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder.go — Added WKT conversion support and plain-object restrictions (commit b65868b)
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/protogoja/builder_test.go — Added dynamic WKT descriptor tests for Timestamp Date/RFC3339

