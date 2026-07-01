---
Title: Investigation diary
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
    - Path: pkg/replhttp/handler.go
      Note: Evidence for route inventory
    - Path: pkg/replsession/types.go
      Note: Evidence for current implicit JSON transport DTOs
    - Path: ttmp/2026/07/01/GOJA-067--protobuf-schema-for-replapi-payloads-and-typescript-generation/design-doc/01-protobuf-replapi-schema-and-typescript-generation-implementation-guide.md
      Note: Primary intern-facing implementation guide produced in this step
ExternalSources: []
Summary: Chronological diary for the replapi protobuf schema and TypeScript generation design ticket.
LastUpdated: 2026-07-01T08:57:25.224828565-07:00
WhatFor: Read before resuming GOJA-067 implementation or changing replapi transport payloads.
WhenToUse: Use when implementing protobuf schemas, generated TypeScript bindings, protojson handlers, or replapi conversion adapters.
---


# Diary

## Goal

Capture the investigation and design work for converting `go-go-goja`'s `replapi` HTTP payloads from an implicit Go-struct JSON contract into a protobuf-defined contract with generated Go and TypeScript bindings.

## Step 1: Create Ticket and Write Protobuf replapi Design Guide

I created a new docmgr ticket in the `go-go-goja` repository and inspected the existing REPL API layers before writing the guide. The main finding is that `replapi` already has a clean DTO boundary in `pkg/replsession/types.go`, but the contract is implicit: Go structs and `encoding/json` define the wire shape, while frontend code has no generated TypeScript source of truth.

The design proposes a conservative migration: introduce a protobuf transport schema and generated TypeScript, keep the internal `replsession` service model unchanged at first, and add adapters at the HTTP boundary. This gives future workbench UIs a generated schema without destabilizing the JavaScript evaluation engine.

### Prompt Context

**User prompt (verbatim):** "ok, create a new ticket in go-go-goja to do that with docmgr --root go-go-goja/ttmp when working with docmgr. 

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new `go-go-goja` docmgr ticket using `docmgr --root ttmp`, analyze whether `replapi` payloads should be protobuf-defined for TypeScript generation, write a detailed intern-facing design/implementation guide, and upload it to reMarkable.

**Inferred user intent:** The user wants a durable implementation plan for schema-first REPL API payloads so future web/workbench UIs can consume generated TypeScript types instead of hand-maintained mirrors.

**Commit (code):** N/A — documentation/design ticket only.

### What I did

- Created ticket `GOJA-067` under `/home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/go-go-goja/ttmp`.
- Created the primary design doc:
  - `design-doc/01-protobuf-replapi-schema-and-typescript-generation-implementation-guide.md`
- Created this diary:
  - `reference/01-investigation-diary.md`
- Inspected current `go-go-goja` files:
  - `pkg/replsession/types.go`
  - `pkg/replsession/policy.go`
  - `pkg/replapi/app.go`
  - `pkg/replhttp/handler.go`
  - `pkg/repldb/types.go`
  - `pkg/replsession/evaluate.go`
  - `cmd/goja-repl/cmd_serve.go`
- Inspected the local `sessionstream` protobuf/protojson pattern:
  - `proto/sessionstream/v1/transport.proto`
  - `pkg/sessionstream/schema.go`
  - `pkg/sessionstream/transport/ws/server.go`
  - `pkg/js/modules/sessionstream/codec.go`
  - `buf.yaml`
  - `buf.gen.yaml`
- Wrote a design guide covering current state, gaps, proposed protobuf schema, Buf config, TypeScript usage, adapters, HTTP migration paths, decision records, implementation phases, tests, risks, and file references.

### Why

- `replapi` is likely to become a backend for richer workbench UIs.
- Hand-written frontend mirrors of `CellReport`, `SessionSummary`, `RuntimeReport`, and persistence/export records will drift.
- Protobuf can provide one schema for Go and TypeScript while preserving JSON transport through `protojson`.

### What worked

- The current DTO layer is already isolated enough to map into protobuf messages.
- `replhttp` has a small route surface that can be wrapped with protobuf response envelopes.
- `sessionstream` provides a nearby example of protobuf schemas and strict protojson options.

### What didn't work

- I first attempted to create ticket `GOJA-063`, but that ID already existed for an older ticket. `docmgr doc list --ticket GOJA-063` reported ambiguity because the accidental new directory conflicted with the existing `GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic` ticket. I removed the accidentally created new `GOJA-063` directory and created `GOJA-067` instead.

Exact command context:

```bash
docmgr --root ttmp ticket create-ticket --ticket GOJA-063 --title "Protobuf schema for replapi payloads and TypeScript generation" --topics replapi,protobuf,typescript,api-design
# later:
rm -rf ttmp/2026/07/01/GOJA-063--protobuf-schema-for-replapi-payloads-and-typescript-generation
```

### What I learned

- `pkg/replsession/types.go` explicitly declares itself as the JSON transport DTO layer, which makes it a strong source for schema extraction.
- The current HTTP routes expose both live session DTOs and durable `repldb` records. The durable records contain `json.RawMessage` fields, so the protobuf design must use `google.protobuf.Value` or a similar open JSON representation.
- `sessionstream` currently demonstrates protobuf JSON transport and Go generation. The TypeScript generation step should use Buf/protoc-gen-es and `@bufbuild/protobuf` for the future frontend package.

### What was tricky to build

- The design needed to avoid over-scoping. Replacing internal REPL structs with generated protobuf messages would be a large, risky rewrite. The guide instead recommends an adapter boundary so implementation can proceed incrementally.
- The API has two categories of dynamic data: live `ExecutionReport.resultJson`, which is currently a string, and persisted `json.RawMessage` fields, which are real JSON values. The guide preserves that distinction in the first schema version.

### What warrants a second pair of eyes

- Review whether the first implementation should add `/api/v1` protobuf JSON routes alongside legacy routes, or replace the existing route encoder in place.
- Review the proposed mapping of `json.RawMessage` to `google.protobuf.Value` and `ExecutionReport.result_json` to a string.
- Review whether generated TypeScript should live inside this repository or be emitted into a separate frontend/workbench package.

### What should be done in the future

- Implement Phase 1 field mapping before writing the `.proto` file.
- Add golden protojson tests as soon as the first adapter exists.
- Add a TypeScript `fromJson` smoke test in CI once generated TypeScript is checked in.

### Code review instructions

- Start with `design-doc/01-protobuf-replapi-schema-and-typescript-generation-implementation-guide.md`.
- Cross-check file-backed claims against:
  - `pkg/replsession/types.go`
  - `pkg/replhttp/handler.go`
  - `pkg/repldb/types.go`
  - `/home/manuel/code/wesen/go-go-golems/sessionstream/proto/sessionstream/v1/transport.proto`
- Validate docmgr hygiene with:
  - `docmgr --root ttmp doctor --ticket GOJA-067 --stale-after 30`

### Technical details

Primary ticket path:

```text
ttmp/2026/07/01/GOJA-067--protobuf-schema-for-replapi-payloads-and-typescript-generation
```

## Step 2: Expand the Implementation Task Plan

I expanded `tasks.md` from a short phase checklist into a detailed phase-by-phase implementation tracker. The new plan breaks the work into route inventory, DTO inventory, schema authoring, Buf generation, Go conversion adapters, protobuf JSON HTTP routes, TypeScript decode tests, documentation, validation, and delivery.

The purpose of this step was to make the implementation continuation-safe. Each phase now has small checkboxes that can be completed and committed independently, so a future session can resume without rereading the whole design guide to infer what remains.

### Prompt Context

**User prompt (verbatim):** "Add more detailed list of tasks per phase so that we can lose track of where we are. Then implement task by task, committing at appropriate intervals, and keeping a detailed diary"

**Assistant interpretation:** Expand the GOJA-067 task list enough to track progress precisely, then begin implementation one task/phase at a time with commits and diary updates.

**Inferred user intent:** The user wants a continuation-friendly implementation process that avoids losing state across long sessions.

**Commit (code):** N/A — task planning update before implementation.

### What I did

- Rewrote `tasks.md` with detailed sections for Phases A through F.
- Added per-phase checklist items for:
  - current route and DTO inventory,
  - protobuf schema and Buf setup,
  - adapter package implementation,
  - protobuf JSON HTTP routes,
  - TypeScript package and decode tests,
  - docs, final validation, upload, commit, and push.

### Why

- The design guide is intentionally broad. Implementation needs a narrower executable checklist.
- Detailed task boundaries make it clear when to commit and what validation belongs to each phase.

### What worked

- The task list now mirrors the design guide's implementation phases and can serve as the operational source of truth.

### What didn't work

- N/A

### What I learned

- The safest implementation order is to inventory the current JSON route/DTO surface before writing the `.proto` file. That prevents schema omissions.

### What was tricky to build

- The task list needed enough detail to prevent lost context without duplicating the full design document. I kept rationale in the guide and made tasks concrete and checkable.

### What warrants a second pair of eyes

- Review whether Phase D should add `/api/v1` alongside legacy routes or replace the current handler directly. The task list leaves the decision explicit.

### What should be done in the future

- Start Phase A by writing `reference/02-current-replapi-json-shape.md`.

### Code review instructions

- Review `tasks.md` in the GOJA-067 ticket.
- Compare Phase A-F tasks against `design-doc/01-protobuf-replapi-schema-and-typescript-generation-implementation-guide.md`.

### Technical details

Task file:

```text
ttmp/2026/07/01/GOJA-067--protobuf-schema-for-replapi-payloads-and-typescript-generation/tasks.md
```
