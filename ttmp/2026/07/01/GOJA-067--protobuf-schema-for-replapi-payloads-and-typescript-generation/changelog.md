# Changelog

## 2026-07-01

- Initial workspace created


## 2026-07-01

Created protobuf replapi schema and TypeScript generation design guide; analyzed current replsession/replhttp DTOs, repldb raw JSON records, and sessionstream protojson reference pattern.

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/pkg/replhttp/handler.go — Current route source of truth
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/pkg/replsession/types.go — Current DTO source of truth
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/ttmp/2026/07/01/GOJA-067--protobuf-schema-for-replapi-payloads-and-typescript-generation/design-doc/01-protobuf-replapi-schema-and-typescript-generation-implementation-guide.md — Primary design deliverable


## 2026-07-01

Validated GOJA-067 with docmgr doctor and uploaded guide bundle to reMarkable at /ai/2026/07/01/GOJA-067 as GOJA-067 Protobuf replapi TypeScript Guide.pdf.

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/ttmp/2026/07/01/GOJA-067--protobuf-schema-for-replapi-payloads-and-typescript-generation/design-doc/01-protobuf-replapi-schema-and-typescript-generation-implementation-guide.md — Uploaded primary design guide


## 2026-07-01

Phase A: added current replapi JSON shape inventory mapping routes, live DTOs, persistence DTOs, timestamps, integer widths, dynamic JSON fields, and enum/string decisions.

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/ttmp/2026/07/01/GOJA-067--protobuf-schema-for-replapi-payloads-and-typescript-generation/reference/02-current-replapi-json-shape-inventory.md — Phase A field inventory


## 2026-07-01

Phase B: added replapi protobuf schema, Buf v2 configuration, generated Go bindings, and generated TypeScript bindings; validated with buf lint, targeted buf generate, and GOWORK=off go test ./pkg/replapi/... -count=1.

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/pkg/replapi/pb/proto/goja/replapi/v1/replapi.pb.go — Generated Go bindings
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/proto/goja/replapi/v1/replapi.proto — Schema source
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/web/packages/replapi-types/src/generated/proto/goja/replapi/v1/replapi_pb.ts — Generated TypeScript bindings


## 2026-07-01

Phase C: added replapi pbconv adapters for live session DTOs and repldb records, protojson helpers, raw JSON Value conversion, and focused tests; validated with GOWORK=off go test ./pkg/replapi/pbconv ./pkg/replapi/... -count=1.

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/pkg/replapi/pbconv/repldb.go — Persistence DTO conversion
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/pkg/replapi/pbconv/session.go — Live DTO conversion
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/pkg/replapi/pbconv/session_test.go — Focused tests


## 2026-07-01

Phase D: added NewProtoJSONHandler with /api/v1 protobuf JSON routes for session lifecycle, evaluation, history, bindings, docs, and export while preserving legacy NewHandler; focused replhttp tests pass.

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/pkg/replhttp/proto_handler.go — New v1 handler
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/pkg/replhttp/proto_handler_test.go — Route tests

