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


## 2026-07-01

Phase E planning update: selected a package-local TypeScript workspace layout modeled after rag-evaluation-system, with web/packages/replapi-types exporting generated protobuf bindings and running focused decode smoke tests.

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/ttmp/2026/07/01/GOJA-067--protobuf-schema-for-replapi-payloads-and-typescript-generation/design-doc/01-protobuf-replapi-schema-and-typescript-generation-implementation-guide.md — Phase E package plan
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/ttmp/2026/07/01/GOJA-067--protobuf-schema-for-replapi-payloads-and-typescript-generation/tasks.md — Phase E checklist refinement


## 2026-07-01

Phase E: added replapi-types TypeScript package, pnpm workspace scripts, Go-emitted protojson fixtures, generated-schema decode smoke tests, and BigInt/Value usage documentation; validation passed with pnpm replapi-types:typecheck and pnpm replapi-types:test.

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/pnpm-lock.yaml — Locked TypeScript validation dependencies
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/web/packages/replapi-types/README.md — Consumer documentation
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/web/packages/replapi-types/src/replapi_decode.test.ts — TypeScript decode smoke tests


## 2026-07-01

Phase E.5: aligned replapi-types with the trusted npm publishing playbook by adding a compiled dist artifact, pack and clean-consumer smoke scripts, publish metadata, and a tokenless GitHub Actions trusted-publishing workflow.

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/.github/workflows/publish-npm.yml — Trusted publishing workflow
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/web/packages/replapi-types/package.json — Public npm metadata and artifact scripts
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/web/packages/replapi-types/scripts/consumer-smoke.mjs — Consumer validation


## 2026-07-01

Renamed the npm package from the scoped go-go-goja-specific name to replapi-types, and revalidated build, pack, clean-consumer smoke, TypeScript tests, and docmgr doctor.

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/.github/workflows/publish-npm.yml — Publish workflow now targets dist package name replapi-types
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/web/packages/replapi-types/package.json — Package name changed to replapi-types
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/web/packages/replapi-types/scripts/consumer-smoke.mjs — Consumer smoke imports updated to replapi-types


## 2026-07-01

Published replapi-types@0.1.0 to npm, configured npm Trusted Publishing for go-go-golems/go-go-goja publish-npm.yml in npm-production, created the GitHub environment, pushed the branch, and opened PR #91; workflow verification is pending merge because GitHub cannot dispatch a new workflow file before it exists on the default branch.

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/.github/workflows/publish-npm.yml — Trusted publishing workflow pending default-branch availability
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/web/packages/replapi-types/package.json — Published package metadata for replapi-types@0.1.0

