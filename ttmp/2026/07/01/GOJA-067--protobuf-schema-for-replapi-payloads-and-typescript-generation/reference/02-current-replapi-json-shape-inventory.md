---
Title: Current replapi JSON shape inventory
Ticket: GOJA-067
Status: active
Topics:
    - replapi
    - protobuf
    - typescript
    - api-design
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/repldb/types.go
      Note: Persistence/export DTO inventory
    - Path: pkg/replhttp/handler.go
      Note: Route inventory and current response envelopes
    - Path: pkg/replsession/policy.go
      Note: Session policy and eval mode inventory
    - Path: pkg/replsession/types.go
      Note: Live REPL JSON DTO inventory
ExternalSources: []
Summary: Field-by-field inventory of the current replapi/replhttp JSON payloads and proposed protobuf representation choices.
LastUpdated: 2026-07-01T16:25:00-07:00
WhatFor: Use as the source checklist when authoring `proto/goja/replapi/v1/replapi.proto`.
WhenToUse: Use before adding or changing protobuf messages for replapi payloads.
---


# Current replapi JSON shape inventory

## Goal

This reference records the current JSON payload surface of `go-go-goja`'s REPL API and maps it to proposed protobuf messages. It exists so the `.proto` implementation can proceed from a concrete inventory instead of re-discovering fields from the Go code during schema authoring.

## Context

The current public HTTP API is implemented in `pkg/replhttp/handler.go`. It uses `encoding/json` and returns a mix of live REPL DTOs from `pkg/replsession` and persistence DTOs from `pkg/repldb`. The DTO source files are:

- `pkg/replsession/types.go` — live session, evaluation, static analysis, rewrite, execution, runtime, binding, AST/CST, and utility views.
- `pkg/replsession/policy.go` — session policy structs and evaluation mode values.
- `pkg/repldb/types.go` — durable session records, evaluation records, binding versions, and binding docs.
- `pkg/replhttp/handler.go` — route inventory and inline response envelopes.

The proposed protobuf package is `goja.replapi.v1`, with generated Go under `pkg/replapi/pb` and generated TypeScript under a frontend-facing package path chosen in Phase B.


## Final implementation links

The Phase A inventory was implemented as a schema-first transport layer rather than a replacement for the internal REPL service model.

- Schema: `proto/goja/replapi/v1/replapi.proto`
- Generated Go: `pkg/replapi/pb/proto/goja/replapi/v1/replapi.pb.go`
- Generated TypeScript: `web/packages/replapi-types/src/generated/proto/goja/replapi/v1/replapi_pb.ts`
- Go conversion adapters: `pkg/replapi/pbconv/`
- Opt-in protobuf JSON handler: `pkg/replhttp/proto_handler.go`
- TypeScript package and smoke tests: `web/packages/replapi-types/`
- npm package: `replapi-types`

The legacy `encoding/json` handler has been removed. `replhttp.NewHandler` exposes protobuf JSON routes under `/api/...`.

Notable final choices relative to the initial inventory:

- `ExportSessionResponse` wraps `SessionExport` for HTTP response consistency.
- `ExecutionReport.result_json` remains a string in v1, preserving existing `replsession.ExecutionReport.ResultJSON` behavior.
- `json.RawMessage` persistence fields are represented with `google.protobuf.Value`, which supports object, array, string, number, boolean, and null JSON shapes.
- Broad runtime/parser status fields remain strings; only `EvalMode` is modeled as a protobuf enum.
- CI publishing for the TypeScript bindings uses a compiled `dist/` artifact and npm Trusted Publishing, not source-only package publication.

## Route inventory

| Method | Path | Request body | Current response body | Source | Proposed protobuf response |
|---|---|---|---|---|---|
| `GET` | `/api/sessions` | none | `{ "sessions": []SessionRecord }` | `pkg/replhttp/handler.go:22-30` | `ListSessionsResponse` |
| `POST` | `/api/sessions` | none | `{ "session": SessionSummary }` | `pkg/replhttp/handler.go:32-39` | `CreateSessionResponse` |
| `GET` | `/api/sessions/{id}` | none | `{ "session": SessionSummary }` | `pkg/replhttp/handler.go:41-49` | `GetSessionResponse` |
| `DELETE` | `/api/sessions/{id}` | none | `{ "deleted": true }` | `pkg/replhttp/handler.go:51-57` | `DeleteSessionResponse` |
| `POST` | `/api/sessions/{id}/evaluate` | `EvaluateRequest` | `EvaluateResponse` | `pkg/replhttp/handler.go:59-72` | `EvaluateResponse` |
| `POST` | `/api/sessions/{id}/restore` | none | `{ "session": SessionSummary }` | `pkg/replhttp/handler.go:74-82` | `RestoreSessionResponse` |
| `GET` | `/api/sessions/{id}/history` | none | `{ "history": []EvaluationRecord }` | `pkg/replhttp/handler.go:84-92` | `HistoryResponse` |
| `GET` | `/api/sessions/{id}/bindings` | none | `{ "bindings": []BindingView }` | `pkg/replhttp/handler.go:94-102` | `BindingsResponse` |
| `GET` | `/api/sessions/{id}/docs` | none | `{ "docs": []BindingDocRecord }` | `pkg/replhttp/handler.go:104-112` | `DocsResponse` |
| `GET` | `/api/sessions/{id}/export` | none | `SessionExport` | `pkg/replhttp/handler.go:114-122` | `ExportSessionResponse` or `SessionExport` |

Inline response envelopes should become named protobuf messages. The current API has several anonymous map payloads. The protobuf API should avoid anonymous JSON envelopes because generated TypeScript needs stable named schemas.

## Representation decisions

| Current Go type/pattern | Current JSON behavior | Proposed protobuf representation | Notes |
|---|---|---|---|
| `time.Time` | RFC3339 JSON string through `encoding/json` | `google.protobuf.Timestamp` | Protojson also emits RFC3339-like strings. Use `timestamppb.New`. |
| `*time.Time` | omitted when nil through `omitempty` | `google.protobuf.Timestamp deleted_at` | Absence represents nil. |
| `int` IDs/counts/line numbers | JSON number | `uint32` when non-negative, `int32` only if negative possible | Cell IDs, counts, lines, node IDs, depths are non-negative in practice. |
| `int64` duration/evaluation IDs | JSON number today | `int64` | Protojson emits int64 as JSON strings; TypeScript receives bigint through `fromJson`. Document this. |
| `uint64` if added later | not currently in repl DTOs | `uint64` | Protojson emits strings. |
| `json.RawMessage` | arbitrary raw JSON | `google.protobuf.Value` | Needed for `repldb` persisted fields. |
| `ExecutionReport.ResultJSON string` | string containing JSON envelope | `string result_json` in v1 | Preserve current semantics first; normalize later only with migration. |
| string status/kind/change/origin | JSON string | `string` in v1 | Avoid premature enums except `EvalMode`; current values are broad and partly parser/runtime-derived. |
| `EvalMode` string | `"raw"` / `"instrumented"` | enum `EvalMode` | Small stable domain; adapter maps enum to/from strings. |
| optional pointer fields | omitted when nil | message field presence | Use pointer generated fields in Go and optional access in TS. |
| repeated slices | arrays; nil may encode `null` in some structs if not initialized | repeated fields | Adapter should output empty repeated fields where useful; protojson omits empty unless EmitUnpopulated changes. |

## Live replsession DTO map

### EvaluateRequest

Source: `pkg/replsession/types.go:25-28`.

| JSON field | Go field | Go type | Proto field | Proto type | Notes |
|---|---|---|---|---|---|
| `source` | `Source` | `string` | `source` | `string` | Add `schema_version` to protobuf request envelope. |

### EvaluateResponse

Source: `pkg/replsession/types.go:30-34`.

| JSON field | Go field | Go type | Proto field | Proto type | Notes |
|---|---|---|---|---|---|
| `session` | `Session` | `*SessionSummary` | `session` | `SessionSummary` | Response should include `schema_version`. |
| `cell` | `Cell` | `*CellReport` | `cell` | `CellReport` | Optional for error paths if needed, but normal evaluate returns both. |

### SessionSummary

Source: `pkg/replsession/types.go:11-23`.

| JSON field | Go field | Go type | Proto type | Notes |
|---|---|---|---|---|
| `id` | `ID` | `string` | `string` | Session ID. |
| `profile` | `Profile` | `string` | `string` | Typically `raw`, `interactive`, or `persistent`. |
| `policy` | `Policy` | `SessionPolicy` | `SessionPolicy` | From `pkg/replsession/policy.go`. |
| `createdAt` | `CreatedAt` | `time.Time` | `google.protobuf.Timestamp` | Required in summaries. |
| `cellCount` | `CellCount` | `int` | `uint32` | Non-negative. |
| `bindingCount` | `BindingCount` | `int` | `uint32` | Non-negative. |
| `bindings` | `Bindings` | `[]BindingView` | `repeated BindingView` | Current bindings. |
| `history` | `History` | `[]HistoryEntry` | `repeated HistoryEntry` | Compact history. |
| `currentGlobals` | `CurrentGlobals` | `[]GlobalStateView` | `repeated GlobalStateView` | Runtime snapshot. |
| `provenance` | `Provenance` | `[]ProvenanceRecord` | `repeated ProvenanceRecord` | How fields were obtained. |

### SessionPolicy and subpolicies

Source: `pkg/replsession/policy.go:8-37`.

| JSON field | Go type | Proto type | Notes |
|---|---|---|---|
| `eval.mode` | `EvalMode string` | `EvalMode enum` | Values: `raw`, `instrumented`. |
| `eval.captureLastExpression` | `bool` | `bool` | Instrumented behavior. |
| `eval.supportTopLevelAwait` | `bool` | `bool` | Top-level await support. |
| `eval.timeoutMs` | `int64` | `int64` | Protojson string in JSON; TS bigint. |
| `observe.staticAnalysis` | `bool` | `bool` | Analysis toggle. |
| `observe.runtimeSnapshot` | `bool` | `bool` | Runtime snapshot toggle. |
| `observe.bindingTracking` | `bool` | `bool` | Binding tracking toggle. |
| `observe.consoleCapture` | `bool` | `bool` | Console capture toggle. |
| `observe.jsdocExtraction` | `bool` | `bool` | JSDoc extraction toggle. |
| `persist.enabled` | `bool` | `bool` | Persistence toggle. |
| `persist.evaluations` | `bool` | `bool` | Persist evaluations. |
| `persist.bindingVersions` | `bool` | `bool` | Persist binding versions. |
| `persist.bindingDocs` | `bool` | `bool` | Persist binding docs. |

### CellReport

Source: `pkg/replsession/types.go:38-48`.

| JSON field | Go field | Go type | Proto type | Notes |
|---|---|---|---|---|
| `id` | `ID` | `int` | `uint32` | Cell ID. |
| `createdAt` | `CreatedAt` | `time.Time` | `Timestamp` | Creation time. |
| `source` | `Source` | `string` | `string` | Original source. |
| `static` | `Static` | `StaticReport` | `StaticReport` | Parser/static analysis. |
| `rewrite` | `Rewrite` | `RewriteReport` | `RewriteReport` | Transformation report. |
| `execution` | `Execution` | `ExecutionReport` | `ExecutionReport` | Runtime outcome. |
| `runtime` | `Runtime` | `RuntimeReport` | `RuntimeReport` | Global/binding diffs. |
| `provenance` | `Provenance` | `[]ProvenanceRecord` | `repeated ProvenanceRecord` | Data sources. |

### ExecutionReport

Source: `pkg/replsession/types.go:50-61`.

| JSON field | Go field | Go type | Proto type | Notes |
|---|---|---|---|---|
| `status` | `Status` | `string` | `string` | Examples: `ok`, `empty-source`, `parse-error`. Keep string in v1. |
| `result` | `Result` | `string` | `string` | Human preview. |
| `resultJson` | `ResultJSON` | `string` | `string` | JSON envelope string, not `Value` in v1. |
| `error` | `Error` | `string` | `string` | Empty string when absent. |
| `durationMs` | `DurationMS` | `int64` | `int64` | Protojson string. |
| `awaited` | `Awaited` | `bool` | `bool` | Promise/await behavior. |
| `console` | `Console` | `[]ConsoleEvent` | `repeated ConsoleEvent` | Captured console output. |
| `hadSideEffects` | `HadSideFX` | `bool` | `bool` | Note Go field abbreviation differs from JSON. |
| `helperError` | `HelperError` | `bool` | `bool` | Instrumentation helper status. |

### StaticReport

Source: `pkg/replsession/types.go:69-84`.

| JSON field | Proto type | Notes |
|---|---|---|
| `diagnostics` | `repeated DiagnosticView` | Parser/static diagnostics. |
| `topLevelBindings` | `repeated TopLevelBindingView` | Declarations in submitted cell. |
| `unresolved` | `repeated IdentifierUseView` | Unresolved identifier uses. |
| `references` | `repeated BindingReferenceGroup` | Identifier usages grouped by binding. |
| `scope` | `ScopeView` | Optional recursive tree. |
| `ast` | `repeated ASTRowView` | Flattened AST rows. |
| `astNodeCount` | `uint32` | Total AST nodes before truncation. |
| `astTruncated` | `bool` | Truncation flag. |
| `cst` | `repeated CSTNodeView` | Flattened tree-sitter CST rows. |
| `cstNodeCount` | `uint32` | Total CST nodes before truncation. |
| `cstTruncated` | `bool` | Truncation flag. |
| `finalExpression` | `RangeView` | Optional source range. |
| `summary` | `repeated StaticSummaryFact` | UI facts. |

### RewriteReport

Source: `pkg/replsession/types.go:92-110`.

| JSON field | Proto type | Notes |
|---|---|---|
| `mode` | `string` | Examples: `raw`, `async-iife-with-binding-capture`. Keep string. |
| `declaredNames` | `repeated string` | Names declared by cell. |
| `helperNames` | `repeated string` | Instrumentation helper names. |
| `lastHelperName` | `string` | Helper that stores final value. |
| `bindingHelperName` | `string` | Helper for binding capture. |
| `capturedLastExpr` | `bool` | Whether last expression was captured. |
| `transformedSource` | `string` | Rewritten source. |
| `operations` | `repeated RewriteStep` | Rewrite operations. |
| `warnings` | `repeated string` | Optional warnings. |
| `finalExpressionSource` | `string` | Optional expression text. |

### RuntimeReport

Source: `pkg/replsession/types.go:112-123`.

| JSON field | Proto type | Notes |
|---|---|---|
| `beforeGlobals` | `repeated GlobalStateView` | Snapshot before evaluation. |
| `afterGlobals` | `repeated GlobalStateView` | Snapshot after evaluation. |
| `diffs` | `repeated GlobalDiffView` | Global changes. |
| `newBindings` | `repeated string` | New bindings. |
| `updatedBindings` | `repeated string` | Updated bindings. |
| `removedBindings` | `repeated string` | Removed bindings. |
| `leakedGlobals` | `repeated string` | Globals not captured as bindings. |
| `persistedByWrap` | `repeated string` | Names persisted by wrapper. |
| `currentCellValue` | `string` | Final cell value preview. |

## Binding/runtime detail DTOs

Source: `pkg/replsession/types.go:143-229`.

| Go type | Proposed proto message | Key fields and representations |
|---|---|---|
| `BindingView` | `BindingView` | `name`, `kind`, `origin`, `declared_in_cell uint32`, `last_updated_cell uint32`, `declared_line uint32`, `declared_snippet`, optional `static`, `runtime`, provenance. |
| `BindingStaticView` | `BindingStaticView` | repeated `IdentifierUseView`, repeated params, `extends`, repeated `MemberView`. |
| `BindingRuntimeView` | `BindingRuntimeView` | `value_kind`, `preview`, repeated own properties, repeated prototype chain, optional function mapping. |
| `PrototypeLevelView` | `PrototypeLevelView` | `name`, repeated `PropertyView`. |
| `PropertyView` | `PropertyView` | `name`, `kind`, `preview`, `is_symbol`, optional descriptor. |
| `DescriptorView` | `DescriptorView` | `writable`, `enumerable`, `configurable`, `has_getter`, `has_setter`. |
| `FunctionMappingView` | `FunctionMappingView` | `name`, optional `class_name`, start/end line/col, `node_id uint32`. |
| `GlobalStateView` | `GlobalStateView` | `name`, `kind`, `preview`, `identity`, `property_count uint32`. |
| `GlobalDiffView` | `GlobalDiffView` | `name`, `change`, before/after previews and kinds, `session_bound`. |

Keep `kind`, `origin`, and `change` as strings in v1. These values come from parser/runtime layers and may evolve.

## Static-analysis detail DTOs

Source: `pkg/replsession/types.go:231-321`.

| Go type | Proposed proto message | Key fields and representations |
|---|---|---|
| `DiagnosticView` | `DiagnosticView` | `severity`, `message`. Keep severity as string in v1. |
| `TopLevelBindingView` | `TopLevelBindingView` | `name`, `kind`, `line uint32`, `snippet`, `extends`, `reference_count uint32`. |
| `BindingReferenceGroup` | `BindingReferenceGroup` | `name`, `kind`, repeated `IdentifierUseView`. |
| `IdentifierUseView` | `IdentifierUseView` | `line uint32`, `col uint32`, `context`, `node_id uint32`, `snippet`. |
| `ScopeView` | `ScopeView` | `id uint32`, `kind`, `start uint32`, `end uint32`, repeated `ScopeBinding`, repeated child `ScopeView`. |
| `ScopeBinding` | `ScopeBinding` | `name`, `kind`. |
| `ASTRowView` | `ASTRowView` | `node_id uint32`, `title`, `description`. |
| `CSTNodeView` | `CSTNodeView` | `depth uint32`, `kind`, `text`, start/end row/col, `is_error`, `is_missing`. |
| `RangeView` | `RangeView` | start/end line/col as `uint32`. |
| `MemberView` | `MemberView` | `name`, `kind`, `preview`, `inherited`, `source`. |
| `StaticSummaryFact` | `StaticSummaryFact` | `label`, `value`. |
| `ConsoleEvent` | `ConsoleEvent` | `kind`, `message`. |
| `ProvenanceRecord` | `ProvenanceRecord` | `section`, `source`, repeated notes. |
| `HistoryEntry` | `HistoryEntry` | `cell_id uint32`, timestamp, `source_preview`, `result_preview`, `status`. |

## repldb persistence DTO map

Source: `pkg/repldb/types.go`.

### SessionRecord

| JSON field | Go type | Proposed proto type | Notes |
|---|---|---|---|
| `sessionId` | `string` | `string` | Durable session ID. |
| `createdAt` | `time.Time` | `Timestamp` | Required. |
| `updatedAt` | `time.Time` | `Timestamp` | Required. |
| `deletedAt` | `*time.Time` | `Timestamp` | Optional. |
| `engineKind` | `string` | `string` | Engine/runtime flavor. |
| `metadataJson` | `json.RawMessage` | `google.protobuf.Value` | Arbitrary metadata JSON. |

### SessionExport

| JSON field | Go type | Proposed proto type | Notes |
|---|---|---|---|
| `session` | `SessionRecord` | `SessionRecord` | Durable session. |
| `evaluations` | `[]EvaluationRecord` | `repeated EvaluationRecord` | Full evaluation history. |

### EvaluationRecord

| JSON field | Go type | Proposed proto type | Notes |
|---|---|---|---|
| `evaluationId` | `int64` | `int64` | Protojson string. |
| `sessionId` | `string` | `string` | Durable session ID. |
| `cellId` | `int` | `uint32` | Cell ID. |
| `createdAt` | `time.Time` | `Timestamp` | Required. |
| `rawSource` | `string` | `string` | Original source. |
| `rewrittenSource` | `string` | `string` | Transformed source. |
| `ok` | `bool` | `bool` | Execution success. |
| `resultJson` | `json.RawMessage` | `google.protobuf.Value` | Full cell report JSON today; arbitrary JSON. |
| `errorText` | `string` | `string` | Error text. |
| `analysisJson` | `json.RawMessage` | `google.protobuf.Value` | Static report JSON. |
| `globalsBeforeJson` | `json.RawMessage` | `google.protobuf.Value` | Runtime snapshot JSON. |
| `globalsAfterJson` | `json.RawMessage` | `google.protobuf.Value` | Runtime snapshot JSON. |
| `consoleEvents` | `[]ConsoleEventRecord` | `repeated ConsoleEventRecord` | Durable console records. |
| `bindingVersions` | `[]BindingVersionRecord` | `repeated BindingVersionRecord` | Binding changes. |
| `bindingDocs` | `[]BindingDocRecord` | `repeated BindingDocRecord` | Extracted docs. |

### Nested persistence records

| Go type | Proposed proto message | Key fields and representations |
|---|---|---|
| `ConsoleEventRecord` | `ConsoleEventRecord` | `stream`, `seq uint32`, `text`. |
| `BindingVersionRecord` | `BindingVersionRecord` | `name`, timestamp, `cell_id uint32`, `action`, `runtime_type`, `display_value`, `summary_json Value`, `export_kind`, `export_json Value`, `doc_digest`. |
| `BindingDocRecord` | `BindingDocRecord` | `symbol_name`, `cell_id uint32`, `source_kind`, `raw_doc`, `normalized_json Value`. |

## Proposed response-envelope messages

Use named response envelopes for every HTTP route:

```proto
message ListSessionsResponse {
  uint32 schema_version = 1;
  repeated SessionRecord sessions = 2;
}

message CreateSessionResponse {
  uint32 schema_version = 1;
  SessionSummary session = 2;
}

message GetSessionResponse {
  uint32 schema_version = 1;
  SessionSummary session = 2;
}

message DeleteSessionResponse {
  uint32 schema_version = 1;
  bool deleted = 2;
}

message RestoreSessionResponse {
  uint32 schema_version = 1;
  SessionSummary session = 2;
}

message HistoryResponse {
  uint32 schema_version = 1;
  repeated EvaluationRecord history = 2;
}

message BindingsResponse {
  uint32 schema_version = 1;
  repeated BindingView bindings = 2;
}

message DocsResponse {
  uint32 schema_version = 1;
  repeated BindingDocRecord docs = 2;
}

message ExportSessionResponse {
  uint32 schema_version = 1;
  SessionExport export = 2;
}
```

`EvaluateResponse` already has a named response shape and should also include `schema_version`.

## Usage examples

### Authoring the proto

When writing `replapi.proto`, start with these message groups in order:

1. route envelopes and `EvaluateRequest` / `EvaluateResponse`,
2. live session summaries and policy messages,
3. cell reports and subreports,
4. binding/runtime/static-analysis detail messages,
5. persistence/export messages.

This ordering matches the dependency direction: route messages depend on live/persistence messages; live messages depend on detail messages.

### TypeScript decoding expectation

Frontend code should decode protojson with generated schemas:

```ts
import { fromJson } from "@bufbuild/protobuf";
import { EvaluateResponseSchema } from "./generated/proto/goja/replapi/v1/replapi_pb";

const raw = await response.json();
const decoded = fromJson(EvaluateResponseSchema, raw);
console.log(decoded.cell?.execution?.status);
```

Do not write a hand-maintained `interface CellReport` mirror in the frontend.

## Checklist before Phase B

- [x] Every `pkg/replhttp` route has a named proposed response message.
- [x] Every `pkg/replsession/types.go` public DTO has a proposed protobuf representation.
- [x] Every `pkg/replsession/policy.go` policy field has a proposed protobuf representation.
- [x] Every `pkg/repldb/types.go` HTTP-exposed persistence record has a proposed protobuf representation.
- [x] Dynamic JSON fields have a deliberate representation.
- [x] Timestamp and integer-width choices are recorded.
- [x] String-vs-enum choices are recorded.
