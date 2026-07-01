# Tasks

## Completed ticket setup and design delivery

- [x] Create GOJA-067 ticket under `go-go-goja/ttmp`.
- [x] Inspect current `replapi`, `replhttp`, `replsession`, and `repldb` payload boundaries.
- [x] Inspect the local `sessionstream` protobuf/protojson pattern.
- [x] Write intern-facing protobuf replapi schema and TypeScript generation implementation guide.
- [x] Include current-state analysis, proposed architecture, API sketches, decision records, phased implementation plan, and testing strategy.
- [x] Write investigation diary.
- [x] Relate key source files through docmgr.
- [x] Update changelog.
- [x] Run `docmgr --root ttmp doctor --ticket GOJA-067 --stale-after 30`.
- [x] Upload guide bundle to reMarkable.

## Implementation Phase A — Detailed field inventory and route map

### A.1 Route inventory

- [ ] Create `reference/02-current-replapi-json-shape.md`.
- [ ] List every `pkg/replhttp` route with method, path, handler function responsibility, request body, response body, and current Go source type.
- [ ] Mark which responses are inline envelopes such as `{session: ...}` or `{history: ...}`.
- [ ] Mark which responses expose `repldb` persistence types rather than live `replsession` types.

### A.2 DTO inventory

- [ ] Map `replsession.SessionSummary` to a proposed protobuf message.
- [ ] Map `replsession.SessionPolicy`, `EvalPolicy`, `ObservePolicy`, and `PersistPolicy` to proposed protobuf fields/enums.
- [ ] Map `replsession.EvaluateRequest` and `EvaluateResponse`.
- [ ] Map `replsession.CellReport` and subreports: `StaticReport`, `RewriteReport`, `ExecutionReport`, and `RuntimeReport`.
- [ ] Map binding/runtime detail views: `BindingView`, `BindingStaticView`, `BindingRuntimeView`, `PropertyView`, descriptor/prototype/function mapping views.
- [ ] Map static-analysis detail views: diagnostics, top-level bindings, references, identifier usages, scopes, AST rows, CST rows, ranges, and members.
- [ ] Map `repldb.SessionRecord`, `SessionExport`, `EvaluationRecord`, `ConsoleEventRecord`, `BindingVersionRecord`, and `BindingDocRecord`.

### A.3 Representation decisions

- [ ] Record timestamp mapping (`time.Time` -> `google.protobuf.Timestamp`).
- [ ] Record integer-width mapping (`int` -> `uint32` or `int32`; `int64` -> `int64`).
- [ ] Record dynamic JSON mapping (`json.RawMessage` -> `google.protobuf.Value`).
- [ ] Record compatibility decision for `ExecutionReport.resultJson` as a string in v1.
- [ ] Record enum-vs-string decisions for status, kind, severity, origin, change, source kind, and export kind.

### A.4 Phase A validation and bookkeeping

- [ ] Run `docmgr --root ttmp doctor --ticket GOJA-067 --stale-after 30`.
- [ ] Update diary with Phase A implementation details.
- [ ] Update changelog and doc relations.
- [ ] Commit Phase A.

## Implementation Phase B — Protobuf schema and Buf generation setup

### B.1 Schema/config skeleton

- [ ] Add `proto/goja/replapi/v1/replapi.proto`.
- [ ] Add or update root `buf.yaml` using Buf v2 format.
- [ ] Add or update root `buf.gen.yaml` with Go and TypeScript generation targets.
- [ ] Choose generated Go output path and package name.
- [ ] Choose generated TypeScript output path and package layout.

### B.2 Live REPL messages

- [ ] Define `EvaluateRequest` and `EvaluateResponse`.
- [ ] Define response envelopes: `ListSessionsResponse`, `CreateSessionResponse`, `GetSessionResponse`, `DeleteSessionResponse`, `RestoreSessionResponse`, `HistoryResponse`, `BindingsResponse`, `DocsResponse`, and `ExportSessionResponse`.
- [ ] Define `SessionSummary`.
- [ ] Define session policy messages and `EvalMode` enum.
- [ ] Define `CellReport` and subreports.
- [ ] Define binding, runtime, static-analysis, AST/CST, range, and provenance messages.

### B.3 Persistence/export messages

- [ ] Define `SessionRecord` and `SessionExport`.
- [ ] Define `EvaluationRecord`.
- [ ] Define `ConsoleEventRecord`, `BindingVersionRecord`, and `BindingDocRecord`.
- [ ] Use `google.protobuf.Value` for raw JSON persistence fields.

### B.4 Generation and validation

- [ ] Run `buf lint`.
- [ ] Run `buf generate`.
- [ ] Verify generated Go files compile with `GOWORK=off go test ./pkg/replapi/... -count=1` or equivalent.
- [ ] Verify generated TypeScript files import `@bufbuild/protobuf` and expose schemas.
- [ ] Update diary/changelog/doc relations.
- [ ] Commit Phase B.

## Implementation Phase C — Go conversion adapters and protojson helpers

### C.1 Adapter package skeleton

- [ ] Add `pkg/replapi/pbconv` package.
- [ ] Add `doc.go` explaining internal DTO to public protobuf conversion boundary.
- [ ] Add shared `protojson` marshal/unmarshal options with camelCase output and strict unknown-field input.
- [ ] Add helper for `time.Time` <-> `timestamppb.Timestamp`.
- [ ] Add helper for `json.RawMessage` <-> `structpb.Value`.

### C.2 Live DTO conversion

- [ ] Convert `replsession.EvaluateRequest` from proto to internal request.
- [ ] Convert `replsession.EvaluateResponse` to proto.
- [ ] Convert `SessionSummary` and policy types.
- [ ] Convert `CellReport`, `StaticReport`, `RewriteReport`, `ExecutionReport`, and `RuntimeReport`.
- [ ] Convert binding/runtime detail views.
- [ ] Convert static-analysis/AST/CST detail views.

### C.3 Persistence conversion

- [ ] Convert `repldb.SessionRecord`.
- [ ] Convert `repldb.SessionExport`.
- [ ] Convert `repldb.EvaluationRecord` and nested records.
- [ ] Preserve arbitrary raw JSON through `google.protobuf.Value`.

### C.4 Adapter tests

- [ ] Add unit tests for simple `EvaluateResponse` conversion.
- [ ] Add unit tests for empty-source evaluation response conversion.
- [ ] Add unit tests for policy enum/string conversion.
- [ ] Add unit tests for raw JSON object/array/string/number/boolean/null conversion.
- [ ] Add golden protojson output tests for a representative evaluate response.
- [ ] Run focused package tests.
- [ ] Update diary/changelog/doc relations.
- [ ] Commit Phase C.

## Implementation Phase D — Protobuf JSON HTTP routes

### D.1 Handler design

- [ ] Decide whether to add `/api/v1` routes or `NewProtoJSONHandler` only.
- [ ] Add `pkg/replhttp/proto_handler.go` or equivalent.
- [ ] Implement proto request decode for evaluate.
- [ ] Implement proto response encode helper.
- [ ] Preserve existing legacy `NewHandler` behavior unless explicitly migrated.

### D.2 Route implementation

- [ ] Implement `GET /api/v1/sessions`.
- [ ] Implement `POST /api/v1/sessions`.
- [ ] Implement `GET /api/v1/sessions/{id}`.
- [ ] Implement `DELETE /api/v1/sessions/{id}`.
- [ ] Implement `POST /api/v1/sessions/{id}/evaluate`.
- [ ] Implement `POST /api/v1/sessions/{id}/restore`.
- [ ] Implement `GET /api/v1/sessions/{id}/history`.
- [ ] Implement `GET /api/v1/sessions/{id}/bindings`.
- [ ] Implement `GET /api/v1/sessions/{id}/docs`.
- [ ] Implement `GET /api/v1/sessions/{id}/export`.

### D.3 Handler tests

- [ ] Add handler test for creating a session through v1 protobuf JSON route.
- [ ] Add handler test for evaluating a cell through v1 protobuf JSON route.
- [ ] Add handler test that unknown fields in evaluate request fail.
- [ ] Add handler test for session-not-found errors.
- [ ] Compare legacy and v1 core semantics where appropriate.
- [ ] Update diary/changelog/doc relations.
- [ ] Commit Phase D.

## Implementation Phase E — TypeScript package and decode tests

### E.1 TypeScript package skeleton

- [ ] Add `web/packages/replapi-types/package.json` or selected package path.
- [ ] Add `tsconfig.json` for generated-code smoke tests.
- [ ] Add dependency on `@bufbuild/protobuf`.
- [ ] Add generated TypeScript output to package exports or documented import paths.

### E.2 Decode tests

- [ ] Add sample `EvaluateResponse` JSON fixture emitted by Go protojson.
- [ ] Add TypeScript test that decodes `EvaluateResponse` with `fromJson`.
- [ ] Assert execution status, result, console events, and bindings decode correctly.
- [ ] Add sample `SessionExport` JSON fixture with raw JSON fields.
- [ ] Assert `google.protobuf.Value` fields preserve JSON shape.
- [ ] Document `int64`/BigInt behavior and JSON.stringify caveat.

### E.3 Tooling validation

- [ ] Add npm/pnpm script for TypeScript smoke test if a package exists.
- [ ] Run the TS smoke test locally.
- [ ] Decide whether CI should run the TS smoke test immediately or in a later frontend ticket.
- [ ] Update diary/changelog/doc relations.
- [ ] Commit Phase E.

## Implementation Phase F — Documentation, help, and final validation

### F.1 Documentation

- [ ] Update `pkg/doc/04-repl-usage.md` with protobuf JSON endpoint notes.
- [ ] Add a dedicated help/reference doc for `replapi` protobuf payloads if appropriate.
- [ ] Update design guide with implementation status and any deviations.
- [ ] Update field inventory with final schema links.

### F.2 Validation

- [ ] Run `buf lint`.
- [ ] Run `buf generate` and verify clean generated output.
- [ ] Run `GOWORK=off go test ./pkg/replapi/... ./pkg/replhttp/... ./pkg/replsession/... -count=1`.
- [ ] Run `GOWORK=off go test ./... -count=1` if feasible.
- [ ] Run `make test` if feasible.
- [ ] Run `make lint` if feasible.
- [ ] Run TypeScript smoke tests if added.
- [ ] Run `docmgr --root ttmp doctor --ticket GOJA-067 --stale-after 30`.

### F.3 Delivery

- [ ] Upload updated GOJA-067 bundle to reMarkable.
- [ ] Commit final docs.
- [ ] Push branch.
