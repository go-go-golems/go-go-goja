# Tasks

## Phase 0 — Ticket setup and design baseline

- [x] Create reusable go-go-goja ticket for protobuf Goja fluent builders
- [x] Investigate native module, TypeScript descriptor, xgoja DTS, protobuf generation, and fixture surfaces
- [x] Write deep generated-builder design document with phase-2 first-version scope
- [x] Validate ticket docs and upload initial design bundle to reMarkable

## Phase 1 — `pkg/protogoja` runtime foundation

- [x] Add `pkg/protogoja` message reference type with hidden non-enumerable Goja object attachment
- [x] Add `MessageFromValue`, `MustMessageFromValue`, and type-name checking helpers for consuming modules
- [x] Add JS-facing built-message methods: `typeName`, `toJSON()`, `clone()`, and `equals(other)`
- [x] Add focused tests for hidden refs, cloning semantics, JSON output, equality, and non-enumerability
- [x] Run `go test ./pkg/protogoja -count=1`
- [x] Commit Phase 1 runtime foundation

## Phase 2 — Builder runtime conversion helpers

- [x] Add `BuilderRef` with `Set`, `Add`, `Put`, `Clear`, `Build`, and `Clone`
- [x] Implement scalar conversions for string, bool, float/double, int32/uint32, int64/uint64, bytes, and enums
- [x] Implement message field conversion from generated `ProtoMessage` refs and builder refs
- [x] Implement repeated field helpers with replace and append semantics
- [ ] Implement map field helpers with object/Map input, put, delete, and clear semantics
- [ ] Implement oneof helpers with clear/which semantics
- [ ] Implement optional presence helpers with `has<Field>` and `clear<Field>` semantics
- [ ] Add field-path-rich error messages and table-driven tests for all supported field kinds
- [ ] Run `go test ./pkg/protogoja -count=1`
- [ ] Commit Phase 2 builder runtime helpers

## Phase 3 — Well-known type support

- [ ] Add helpers for `Timestamp`, `Duration`, `Any`, `Struct`, `Value`, `ListValue`, wrappers, and `FieldMask`
- [ ] Define strict plain-object acceptance rules for `Struct`/`Value` only
- [ ] Add tests for Date/RFC3339 timestamp input, duration strings, Any wrapping, Struct/Value objects, wrappers, and FieldMask
- [ ] Run `go test ./pkg/protogoja -count=1`
- [ ] Commit Phase 3 well-known type support

## Phase 4 — `cmd/protoc-gen-goja-builder` skeleton

- [x] Add protoc plugin command using `google.golang.org/protobuf/compiler/protogen`
- [x] Parse plugin options: `module_name`, `paths`, `emit_dts`, `emit_provider`, `register_global`, `builder_suffix`, and `message_ref_name`
- [x] Generate one companion Go file per proto file with stable headers and imports
- [x] Add golden test harness for plugin output
- [x] Add compile test for a tiny generated fixture package
- [x] Run `go test ./cmd/protoc-gen-goja-builder ./pkg/protogoja -count=1`
- [x] Commit Phase 4 plugin skeleton

## Phase 5 — Generated fluent builders, enums, and message exports

- [x] Generate per-message namespace exports with `typeName`, `builder()`, `from()`, `is()`, and `clone()`
- [x] Generate builder prototypes with fluent field methods and clear/build/clone helpers
- [x] Generate enum exports and enum setter conversion support
- [x] Generate nested message support and stable names for nested builders
- [x] Generate schema/prototype tokens consumable by other Goja modules
- [x] Add runtime Goja tests requiring a generated fixture module and building concrete proto messages
- [x] Run generated fixture tests and `go test ./cmd/protoc-gen-goja-builder ./pkg/protogoja -count=1`
- [x] Commit Phase 5 generated fluent builders

## Phase 6 — Generated TypeScript declarations and xgoja DTS integration

- [x] Generate `TypeScriptModule(moduleName string) *spec.Module` with RawDTS
- [ ] Emit interfaces for `ProtoMessage`, message types, builder types, enums, oneofs, repeated helpers, and map helpers
- [x] Add tests through `pkg/tsgen/render` and `pkg/xgoja/dtsgen`
- [x] Document TypeScript examples for generated modules
- [x] Run `go test ./pkg/tsgen/... ./pkg/xgoja/dtsgen ./cmd/protoc-gen-goja-builder ./pkg/protogoja -count=1`
- [ ] Commit Phase 6 DTS integration

## Phase 7 — Host integration helpers and examples

- [ ] Generate or provide `NewGojaLoader`, `RegisterGojaModule`, `GojaModule`, and `RegisterMessageTypes` helpers
- [ ] Add examples for raw `require.Registry`, `engine.NativeModuleRegistrar`, and optional xgoja provider integration
- [ ] Add a consuming-module demonstration showing `protogoja.MessageFromValue` avoids JSON/protojson conversion
- [ ] Add docs for protoc, Buf, and `go:generate` workflows
- [ ] Run relevant package tests and `go test ./... -count=1` if feasible
- [ ] Commit Phase 7 integration helpers and examples

## Phase 8 — Final validation and delivery

- [ ] Run full `go test ./... -count=1` from `go-go-goja`
- [ ] Run `docmgr --root go-go-goja/ttmp doctor --ticket GOJA-PB-001 --stale-after 30`
- [ ] Update diary and changelog with final commands, failures, fixes, and commit hashes
- [ ] Upload updated GOJA-PB-001 implementation bundle to reMarkable
- [ ] Commit final docs update
