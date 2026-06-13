---
Title: Investigation diary
Ticket: XGOJA-TS-002
Status: active
Topics:
    - goja
    - xgoja
    - typescript
    - tooling
    - developer-experience
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/pkg/jsverbs/scan.go
      Note: Evidence for ScanDir versus ScanFS metadata differences.
    - Path: go-go-goja/pkg/tsscript/compiler.go
      Note: Existing BundleVirtualEntry API assumes ResolveDir for relative imports.
    - Path: go-go-goja/pkg/xgoja/app/root.go
      Note: Embedded and provider jsverb sources enter through ScanFS.
    - Path: go-go-goja/pkg/xgoja/app/typescript.go
      Note: Runtime TypeScript jsverb bundling call site that triggered the review.
    - Path: pkg/jsverbs/scan.go
      Note: Diary Step 1 evidence for ScanFS metadata gap
    - Path: pkg/tsscript/compiler.go
      Note: Diary Step 1 current bundler behavior
    - Path: pkg/xgoja/app/typescript.go
      Note: Diary Step 1 review trigger and runtime bundling call site
ExternalSources:
    - local:01-code-review-embedded-ts-bundling.md
Summary: Chronological diary for the embedded TypeScript runtime bundling design ticket.
LastUpdated: 2026-06-12T10:33:07.503680401-04:00
WhatFor: Use to understand why XGOJA-TS-002 exists and how the implementation guide was produced.
WhenToUse: Read before implementing or reviewing runtime bundling for embedded/provider TypeScript jsverbs.
---


# Diary

## Goal

This diary records the creation of ticket `XGOJA-TS-002`, the code review issue that triggered it, the investigation into `ScanFS` and runtime TypeScript bundling, and the design guide for supporting bundled local imports from embedded/provider `fs.FS` jsverb sources.

## Step 1: Create the ticket and analyze the embedded TypeScript bundling review

I created a new docmgr ticket for the code review issue. The review points at a real gap in the first TypeScript implementation: filesystem-backed TypeScript jsverbs can bundle local imports because they have a real `ResolveDir`, but embedded and provider-shipped sources are loaded through `ScanFS` and currently do not carry enough `fs.FS` metadata for esbuild to read imported helper files.

I wrote the design around a runtime `fs.FS` bundler rather than treating this as a documentation-only limitation. The resulting guide explains the current scan/runtime split, why goja is not the component that resolves `import` statements, what metadata must be preserved, and how an esbuild plugin should resolve and load embedded TypeScript imports.

### Prompt Context

**User prompt (verbatim):** "Create a new ticekt to handle the code review issue and the support for runtime bundling of embedded ts resources. Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new docmgr ticket for the code review issue about embedded/provider TypeScript jsverb bundling, write a detailed intern-oriented design and implementation guide, keep a diary, and upload the deliverable to reMarkable.

**Inferred user intent:** Preserve the code review finding as a durable ticket and produce a clear implementation plan so a new engineer can fix runtime bundling for TypeScript jsverbs loaded from `fs.FS` sources.

**Commit (code):** N/A — documentation and design ticket only; no production code changed in this step.

### What I did

- Created ticket `XGOJA-TS-002` with:
  - `docmgr ticket create-ticket --ticket XGOJA-TS-002 --title "Runtime bundling for embedded TypeScript jsverbs" --topics goja,xgoja,typescript,tooling,developer-experience`
- Created the primary design document:
  - `ttmp/2026/06/12/XGOJA-TS-002--runtime-bundling-for-embedded-typescript-jsverbs/design/01-runtime-bundling-for-embedded-typescript-jsverbs.md`
- Created this diary:
  - `ttmp/2026/06/12/XGOJA-TS-002--runtime-bundling-for-embedded-typescript-jsverbs/reference/01-investigation-diary.md`
- Captured the code review comment as a local source note:
  - `ttmp/2026/06/12/XGOJA-TS-002--runtime-bundling-for-embedded-typescript-jsverbs/sources/local/01-code-review-embedded-ts-bundling.md`
- Read the relevant implementation files:
  - `pkg/xgoja/app/typescript.go`
  - `pkg/jsverbs/model.go`
  - `pkg/jsverbs/scan.go`
  - `pkg/tsscript/compiler.go`
  - `pkg/xgoja/app/root.go`
  - `pkg/xgoja/providerapi/verbs.go`
  - `cmd/xgoja/internal/generate/main.go`
  - `cmd/xgoja/internal/generate/generate.go`
  - esbuild Go API plugin definitions in `github.com/evanw/esbuild/pkg/api/api.go`
- Wrote a detailed design guide covering:
  - the failing scenario;
  - current filesystem-backed behavior;
  - current embedded/provider behavior;
  - the chosen runtime `fs.FS` resolver design;
  - API changes in `jsverbs` and `tsscript`;
  - esbuild `OnResolve` / `OnLoad` pseudocode;
  - extension probing, external handling, and safe path normalization;
  - implementation phases and tests.

### Why

- The review issue is a real correctness gap, not only a documentation issue.
- Embedded and provider-shipped jsverb sources both use `ScanFS`, so fixing only generated embedded sources by prebundling would leave provider sources unresolved.
- A design ticket gives the implementation a clear path and records the distinction between TypeScript authoring syntax, esbuild bundling, and goja runtime execution.

### What worked

- The architecture had a clean evidence trail:
  - `ScanDir` sets `ResolveDir` for disk files.
  - `ScanFS` currently omits equivalent filesystem metadata.
  - `scanVerbSource` routes both provider and embedded sources through `ScanFS`.
  - `BundleVirtualEntry` passes only `StdinOptions.ResolveDir` to esbuild.
- esbuild's Go API has the plugin hooks needed for the fix: `OnResolve` and `OnLoad`.

### What didn't work

- No command failed while creating the ticket or writing the design.
- The design records a known runtime failure mode from the review: bundled embedded/provider TypeScript jsverbs with local value imports can be discovered during scan but fail during invocation because esbuild cannot resolve imports from an empty `ResolveDir`.

### What I learned

- The issue is specifically about esbuild dependency resolution. goja still receives compiled JavaScript and does not execute ECMAScript `import` syntax in this runtime path.
- `ScanFS` has the information required to preserve the source filesystem, root, and path; the current structs simply do not carry it into `RuntimeTransformInput`.
- A general `fs.FS` bundler is the right first fix because it handles both embedded generated sources and provider-shipped sources.

### What was tricky to build

- The design had to separate three similar but different path concepts:
  - `RelPath`, which defines the jsverbs module path and command metadata identity;
  - `ResolveDir`, which is an OS directory for disk-backed bundling;
  - `FSPath`/`FSRoot`, which are slash-separated paths inside an `fs.FS` and must never be treated as OS paths.
- The design also had to define strict external handling. Bare imports such as `express` may be xgoja runtime modules, but silently externalizing every bare import would hide configuration errors. The guide recommends preserving configured externals and failing unknown bare imports with a clear diagnostic.

### What warrants a second pair of eyes

- Review whether the new `fs.FS` metadata should live directly in `jsverbs.SourceFile`/`FileSpec`/`RuntimeTransformInput` or behind a smaller source-context struct.
- Review whether unknown bare imports should fail by default or be externalized by default. The design chooses failure for better generated-binary correctness.
- Review whether prebundling embedded TypeScript should be implemented in the same ticket or kept as a later optimization after runtime `fs.FS` bundling works.

### What should be done in the future

- Implement failing tests first for embedded and provider TypeScript jsverbs with local imports.
- Implement `tsscript.BundleVirtualEntryFS` with an esbuild plugin.
- Thread `fs.FS` metadata through `jsverbs.ScanFS` into runtime transforms.
- Wire `pkg/xgoja/app/typescript.go` to call the fs-backed bundler when `input.SourceFS != nil`.
- Consider automatic xgoja module alias external population for jsverb TypeScript sources in a later ticket.

### Code review instructions

- Start with the design document:
  - `ttmp/2026/06/12/XGOJA-TS-002--runtime-bundling-for-embedded-typescript-jsverbs/design/01-runtime-bundling-for-embedded-typescript-jsverbs.md`
- Then inspect the current call chain:
  - `pkg/xgoja/app/root.go:314-350`
  - `pkg/jsverbs/scan.go:18-142`
  - `pkg/xgoja/app/typescript.go:15-69`
  - `pkg/tsscript/compiler.go:69-94`
- Validate the future implementation with:
  - `go test ./pkg/tsscript ./pkg/jsverbs ./pkg/xgoja/app -count=1`
  - `go test ./cmd/xgoja/internal/generate ./pkg/xgoja/providers/http ./pkg/xgoja/app ./pkg/tsscript -count=1`

### Technical details

- Current failing call site: `pkg/xgoja/app/typescript.go:57-58`.
- Current disk-backed metadata: `pkg/jsverbs/scan.go:70-75`.
- Current fs-backed metadata gap: `pkg/jsverbs/scan.go:126-134`.
- Embedded/provider scan entry points: `pkg/xgoja/app/root.go:316-344`.
- Current virtual bundler: `pkg/tsscript/compiler.go:69-88`.
- esbuild plugin hooks: `github.com/evanw/esbuild/pkg/api/api.go:563-689`.
