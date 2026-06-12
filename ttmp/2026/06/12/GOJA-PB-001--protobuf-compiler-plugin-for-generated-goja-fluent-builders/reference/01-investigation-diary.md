---
Title: Investigation diary
Ticket: GOJA-PB-001
Status: active
Topics:
    - goja
    - protobuf
    - bindings
    - typescript
    - codegen
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/modules/common.go
      Note: |-
        Native module shape studied for generated module design
        Native module API studied for generated module shape
    - Path: go-go-goja/pkg/protogoja/ref.go
      Note: Phase 1 runtime foundation implementation for hidden ProtoMessage refs
    - Path: go-go-goja/pkg/protogoja/ref_test.go
      Note: Phase 1 tests for clone semantics
    - Path: go-go-goja/pkg/tsgen/spec/types.go
      Note: |-
        TypeScript declaration descriptor studied for generated DTS design
        TypeScript descriptor API studied for generated DTS integration
    - Path: go-go-goja/ttmp/2026/06/12/GOJA-PB-001--protobuf-compiler-plugin-for-generated-goja-fluent-builders/design-doc/01-generated-goja-protobuf-fluent-builders-design.md
      Note: |-
        Primary generated Goja protobuf fluent builders design document
        Primary design document produced by this investigation
ExternalSources: []
Summary: Chronological notes for designing a reusable go-go-goja protobuf fluent builder generator.
LastUpdated: 2026-06-12T16:15:00-04:00
WhatFor: Continuation context for GOJA-PB-001 implementation and review.
WhenToUse: Read before implementing protoc-gen-goja-builder or revising the builder generator design.
---



# Diary

## Goal

This diary records the investigation and documentation work for `GOJA-PB-001`, a reusable `go-go-goja` feature that generates fluent Goja builders for protobuf messages.

## Step 1: Move protobuf builder design into go-go-goja

The earlier sessionstream binding design raised a broader idea: instead of requiring sessionstream's JavaScript bindings to accept plain objects and convert them through JSON/protojson, generate Goja-native fluent builders from protobuf schemas. The user clarified that this should live in `go-go-goja/ttmp`, not the sessionstream ticket, because it is a generic reusable capability for all Goja-consuming code.

I created a new `GOJA-PB-001` ticket under `go-go-goja/ttmp` using `docmgr --root go-go-goja/ttmp ...`, then wrote a dedicated design document for a full protoc plugin and runtime support package. The design intentionally targets generated fluent builders as the first real version, not just a reflection-only prototype.

### Prompt Context

**User prompt (verbatim):** "Create a second analysis document that goes deep into building the proper full thing. we want at least phase 2 for a first version."

**Assistant interpretation:** Add a deeper analysis document for building a real protobuf-to-Goja fluent builder generator, with generated builders as the first serious milestone.

**Inferred user intent:** The user wants to avoid an underpowered reflection-only prototype and instead plan a proper reusable builder generator.

**Commit (code):** N/A — documentation-only work.

**User prompt (verbatim):** "Actually, make a ticket in go-go-goja/ttmp with `docmgr --root go-go-goja/ttmp ...` and store the analysis document in there, since that is a generically resusable feature that is useful for all goja consuming code."

**Assistant interpretation:** Create a separate go-go-goja ticket and move the generic protobuf builder analysis there.

**Inferred user intent:** Keep reusable Goja infrastructure planning in the reusable Goja repository rather than a sessionstream-specific ticket.

**User prompt (verbatim):** "upload to remarkable as well."

**Assistant interpretation:** Upload the resulting go-go-goja ticket document bundle to reMarkable after validation.

**Inferred user intent:** Make the design available as a PDF for reading/review outside the terminal.

### What I did

- Loaded the protobuf Go/TS schema exchange skill for protobuf/TypeScript mapping context.
- Checked `docmgr --root go-go-goja/ttmp status --summary-only`.
- Created ticket `GOJA-PB-001` under `go-go-goja/ttmp`.
- Added a design doc titled `Generated Goja protobuf fluent builders design`.
- Added this investigation diary.
- Investigated local `go-go-goja` files relevant to native modules, TypeScript descriptors, xgoja DTS generation, and existing protoc generation.
- Related the key files to the design document.
- Wrote the full design document with architecture, API sketches, field mapping rules, plugin options, implementation milestones, tests, decision records, and acceptance criteria.

### Why

- Generated protobuf builders are useful across all Goja-consuming packages, not just `sessionstream`.
- Putting the ticket under `go-go-goja/ttmp` keeps the reusable runtime/compiler-plugin work in the correct repository context.
- The design needs to be detailed before implementation because protobuf coverage has many edge cases: oneofs, maps, optional fields, int64/uint64 precision, enums, bytes, and well-known types.

### What worked

- `go-go-goja` already has the key integration surfaces:
  - `modules.NativeModule` for native CommonJS modules.
  - `engine.NativeModuleRegistrar` for runtime registration.
  - `spec.Module.RawDTS` for rich TypeScript declarations.
  - `xgoja/dtsgen` for provider module DTS bundling.
- `go-go-goja` already depends on `google.golang.org/protobuf`, so adding a `protogen`-based plugin is natural.
- Existing protobuf fixtures in the workspace can support tests.

### What didn't work

- No code implementation was attempted in this step.
- No Go tests were run because this was a design/documentation task.

### What I learned

- The best design is a hybrid: generated fluent methods for user ergonomics, backed by a common runtime conversion package for correctness.
- TypeScript declarations should use `RawDTS`, because fluent builders, enums, generic message refs, and namespace-like exports are richer than the current structured `spec.Function` model.
- Generated builder modules should not register globally by default; hosts should opt in through `RegisterGojaModule`, `NewGojaLoader`, `GojaModule`, or provider descriptors.

### What was tricky to build

- The main tricky design issue is field coverage. Protobuf field kinds have very different JavaScript ergonomics, especially 64-bit integers, bytes, oneofs, maps, optional presence, and well-known types.
- Another tricky issue is deciding where direct generated code ends and generic runtime helpers begin. The design recommends generated methods that call shared protoreflect helpers in the first version. This gives a full fluent API without duplicating fragile conversion code in every generated file.

### What warrants a second pair of eyes

- Whether generated companion files should always live in the same Go package as `protoc-gen-go` output.
- Whether `emit_provider=true` should be part of version 1 or deferred to avoid pulling xgoja provider dependencies into every generated package.
- Whether plain JS object input should be rejected for all non-WKT message fields, as proposed.

### What should be done in the future

- Implement `pkg/protogoja` first and test field conversion comprehensively.
- Add `cmd/protoc-gen-goja-builder` with golden and compile tests.
- Add fixture schemas that cover all version-1 field kinds.
- Add a small consuming-module test that proves `protogoja.MessageFromValue` avoids JSON/protojson.

### Code review instructions

- Start with `design-doc/01-generated-goja-protobuf-fluent-builders-design.md`.
- Review the decision records before the implementation plan.
- Cross-check the RelatedFiles frontmatter to see which current go-go-goja APIs the design depends on.
- Validate docs with `docmgr --root go-go-goja/ttmp doctor --ticket GOJA-PB-001 --stale-after 30`.

### Technical details

Representative commands:

```bash
docmgr --root go-go-goja/ttmp status --summary-only
docmgr --root go-go-goja/ttmp ticket create-ticket --ticket GOJA-PB-001 --title "Protobuf compiler plugin for generated Goja fluent builders" --topics goja,protobuf,bindings,typescript,codegen
docmgr --root go-go-goja/ttmp doc add --ticket GOJA-PB-001 --doc-type design-doc --title "Generated Goja protobuf fluent builders design"
docmgr --root go-go-goja/ttmp doc add --ticket GOJA-PB-001 --doc-type reference --title "Investigation diary"
```

## Step 2: Expand the ticket into implementation phases

I converted the high-level design into a detailed phased task list so implementation can proceed in reviewable chunks. The phases start with the reusable `pkg/protogoja` runtime foundation, then add builder conversion helpers, well-known types, the protoc plugin, generated fluent builders, generated TypeScript declarations, host integration examples, and final validation/delivery.

This step intentionally makes the first implementation task small: message references and JS-facing built-message objects. That foundation gives consuming modules a stable `protogoja.MessageFromValue` path before the more complicated builder and compiler-plugin work begins.

### Prompt Context

**User prompt (verbatim):** "Ok, let's add a detailed set of phases and for each phase a list of tasks to the go-go-goja protobuf ticket, then work on it task by task, committing at appropriate intervals, keeping a detailed diary."

**Assistant interpretation:** Add a detailed implementation task breakdown to GOJA-PB-001, then start implementing tasks sequentially with focused commits and diary updates.

**Inferred user intent:** The user wants the design converted into an execution plan and wants implementation progress to be traceable through commits and diary entries.

**Commit (code):** pending — this was a documentation/task update before the first implementation commit.

### What I did

- Rewrote `tasks.md` into eight phases:
  - Phase 0: ticket setup and design baseline.
  - Phase 1: `pkg/protogoja` runtime foundation.
  - Phase 2: builder runtime conversion helpers.
  - Phase 3: well-known type support.
  - Phase 4: `cmd/protoc-gen-goja-builder` skeleton.
  - Phase 5: generated fluent builders, enums, and message exports.
  - Phase 6: generated TypeScript declarations and xgoja DTS integration.
  - Phase 7: host integration helpers and examples.
  - Phase 8: final validation and delivery.
- Kept the first implementation phase narrow enough for a focused commit.

### Why

- The generator is large enough that a single vague task would be hard to review or resume.
- The phased plan aligns with the design doc's implementation milestones but makes them actionable in docmgr's task checklist.
- Starting with runtime message references reduces risk because every later generated builder will depend on the hidden proto-message ref contract.

### What worked

- The existing design doc already had enough detail to translate into phase-level tasks.
- Phase 1 can be implemented and tested independently of the compiler plugin.

### What didn't work

- N/A — no code was changed in this step.

### What I learned

- The right first implementation seam is not the protoc plugin. It is the runtime bridge object that all generated code and consuming modules will share.

### What was tricky to build

- The task list needed enough detail to guide implementation without pretending every design detail is already settled. The split keeps high-risk areas, such as well-known types and TypeScript generation, in separate phases.

### What warrants a second pair of eyes

- Whether Phase 2 and Phase 3 should remain separate once implementation starts; well-known types may affect generic field conversion earlier than expected.

### What should be done in the future

- Check off tasks as code lands.
- Record commit hashes in this diary after each focused commit.

### Code review instructions

- Review `tasks.md` first to understand the intended commit sequence.
- Then review the first implementation commit against Phase 1 only.

### Technical details

Updated file:

```text
go-go-goja/ttmp/2026/06/12/GOJA-PB-001--protobuf-compiler-plugin-for-generated-goja-fluent-builders/tasks.md
```

## Step 3: Implement Phase 1 `pkg/protogoja` message references

I implemented the first runtime foundation slice: Go-backed JavaScript `ProtoMessage` objects that carry a hidden protobuf reference. This gives future generated builders and consuming modules a stable direct path from Goja values to concrete `proto.Message` values without JSON/protojson conversion.

The implementation deliberately clones messages when wrapping and extracting them. That keeps the JavaScript-visible built message stable and prevents callers from accidentally mutating the same Go message instance across API boundaries.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Start working through the new phase/task list, beginning with the first implementation phase.

**Inferred user intent:** The user wants incremental implementation with validation and traceable commits rather than only planning.

**Commit (code):** pending — this step will be committed after the diary/changelog are updated.

### What I did

- Added `pkg/protogoja/ref.go` with:
  - `MessageRef`.
  - `NewMessageRef`.
  - `ToValue`.
  - `MessageFromValue`.
  - `MessageRefFromValue`.
  - `MustMessageFromValue`.
  - `TypeNameFromValue`.
  - `IsMessageValueOf`.
  - JS methods `typeName`, `toJSON()`, `clone()`, and `equals(other)`.
- Added `pkg/protogoja/ref_test.go` with tests for:
  - clone-on-wrap and clone-on-extract behavior;
  - JavaScript method behavior;
  - hidden reference non-enumerability;
  - type-name helpers;
  - equality behavior;
  - invalid input errors;
  - `MustMessageFromValue` panic behavior.
- Ran formatting and tests.
- Checked off Phase 1 tasks 5 through 9.

### Why

- Every generated builder needs a way to return a concrete protobuf message into JavaScript while allowing Go consumers to recover that message safely.
- Starting with this small runtime bridge validates the core representation choice before adding builder setters or compiler generation.

### What worked

- The existing `hashiplugin.contract.v1.ModuleManifest` protobuf type was sufficient as a local fixture.
- `protojson.MarshalOptions{UseProtoNames:false}` produced JS-facing camelCase JSON from `toJSON()`.
- Hidden properties can be attached with `DefineDataProperty` as non-writable, non-enumerable, and non-configurable.
- `go test ./pkg/protogoja -count=1` passed.

### What didn't work

- N/A. The first implementation and tests passed on the first run.

### What I learned

- A small wrapper object is enough for the initial built-message API; it does not need to expose generated Go struct fields.
- Clone-on-boundary semantics are easy to test and should remain part of the contract for generated builders.

### What was tricky to build

- The important sharp edge is avoiding accidental mutability. If `ToValue` stored the original message pointer and `MessageFromValue` returned it directly, Go and JS-adjacent code could mutate supposedly built values. The implementation solves this by cloning on wrap and on extraction.
- Another subtle point is `toJSON()`: Goja should receive an ordinary JS object, not a JSON string. The implementation marshals with `protojson` and decodes into `any` before calling `vm.ToValue`.

### What warrants a second pair of eyes

- The hidden key name is currently a package-private string. Review whether future generated packages need an exported constant or should only use exported attach/extract helpers.
- Review whether `equals` should compare type names before `proto.Equal`; `proto.Equal` already returns false for different message descriptors, but an explicit descriptor check might produce clearer future diagnostics.

### What should be done in the future

- Phase 2 should add `BuilderRef` and field conversion helpers that return these `ProtoMessage` wrappers from `Build()`.
- Generated message namespace objects should carry prototype/schema tokens in addition to built-message refs.

### Code review instructions

- Start with `pkg/protogoja/ref.go` and verify clone semantics in `NewMessageRef`, `MessageRef.Message`, and `MessageFromValue`.
- Then review `pkg/protogoja/ref_test.go` for the JS-facing contract.
- Validate with:

```bash
gofmt -w pkg/protogoja/ref.go pkg/protogoja/ref_test.go
go test ./pkg/protogoja -count=1
```

### Technical details

Commands run:

```bash
cd /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja
gofmt -w pkg/protogoja/ref.go pkg/protogoja/ref_test.go
go test ./pkg/protogoja -count=1
```

Test result:

```text
ok  	github.com/go-go-golems/go-go-goja/pkg/protogoja	0.004s
```
