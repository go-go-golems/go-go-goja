---
Title: Investigation diary
Ticket: XGOJA-TS-001
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
    - Path: go-go-goja/ttmp/2026/06/10/XGOJA-TS-001--typescript-support-for-go-go-goja-xgoja-and-hot-reload/design/01-typescript-support-analysis-and-implementation-guide.md
      Note: Primary deliverable produced during this diary
    - Path: go-go-goja/ttmp/2026/06/10/XGOJA-TS-001--typescript-support-for-go-go-goja-xgoja-and-hot-reload/sources/local/01-goja-typescript-esbuild-note.md
      Note: Imported source note studied before writing the design
ExternalSources:
    - local:01-goja-typescript-esbuild-note.md
Summary: Chronological diary for the TypeScript support research/design ticket.
LastUpdated: 2026-06-10T21:35:00-04:00
WhatFor: Use to understand how the TypeScript support design was researched, written, validated, and delivered.
WhenToUse: Read before continuing implementation or changing the design doc for XGOJA-TS-001.
---


# Diary

## Goal

This diary records the creation of the docmgr ticket, the imported TypeScript/esbuild source note, the repository investigation, and the design decisions for adding TypeScript support to go-go-goja, xgoja, and hot reload.

## Step 1: Create the ticket and import the source note

I created a dedicated docmgr ticket so the TypeScript support investigation has a stable workspace, task list, changelog, source archive, design document, and diary. I also imported `/tmp/goja-ts.md` into the ticket's `sources/local` directory so the design can cite it without depending on a temporary path.

The imported note establishes the central implementation idea: esbuild's Go API can transpile or bundle TypeScript in process, without Node/npm or an external `esbuild` executable at runtime. That shaped the rest of the investigation.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket ot add TS support to go-go-goja / xgoja / hot-reload : 

cp / `docmgr import file --file` the file /tmp/goja-ts.md and study it and then Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


Keep a diary as you work."

**Assistant interpretation:** Create a new docmgr ticket for TypeScript support across go-go-goja, xgoja, and hot reload; import and study `/tmp/goja-ts.md`; write a detailed intern-oriented design/implementation guide; maintain a diary; upload the resulting docs to reMarkable.

**Inferred user intent:** Produce a durable research/design package that a new intern can use as an implementation guide, including system orientation, concrete APIs, pseudocode, file references, and delivery to reMarkable.

**Commit (code):** N/A — documentation-only work; no code commit was created in this step.

### What I did

- Ran `docmgr status --summary-only` to confirm the active doc root.
- Created ticket `XGOJA-TS-001` with:
  - `docmgr ticket create-ticket --ticket XGOJA-TS-001 --title "TypeScript support for go-go-goja xgoja and hot reload" --topics goja,xgoja,typescript,tooling,developer-experience`
- Created the design document:
  - `docmgr doc add --ticket XGOJA-TS-001 --doc-type design --title "TypeScript support analysis and implementation guide"`
- Created this diary:
  - `docmgr doc add --ticket XGOJA-TS-001 --doc-type reference --title "Investigation diary"`
- Imported the source note:
  - `docmgr import file --ticket XGOJA-TS-001 --file /tmp/goja-ts.md --name "goja TypeScript esbuild note"`

### Why

- A ticket workspace keeps the analysis, source note, diary, task list, and changelog together.
- Importing `/tmp/goja-ts.md` preserves the source note as ticket evidence instead of relying on a temporary file path.

### What worked

- Ticket creation succeeded under `go-go-goja/ttmp/2026/06/10/XGOJA-TS-001--typescript-support-for-go-go-goja-xgoja-and-hot-reload`.
- The import command copied the note into `sources/local/goja TypeScript esbuild note.md`; I later renamed it to `sources/local/01-goja-typescript-esbuild-note.md` and added docmgr frontmatter so ticket validation passes.

### What didn't work

- No command failed in this step.

### What I learned

- The docmgr root for this workspace resolves to `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp`.
- The repository vocabulary already includes the relevant topics: `goja`, `xgoja`, `typescript`, `tooling`, and `developer-experience`.

### What was tricky to build

- The user asked for `cp / docmgr import file --file`; I used the docmgr import path because it records the source in ticket metadata and updates the index. That is better than a raw copy for long-term traceability.

### What warrants a second pair of eyes

- N/A for the ticket creation itself.

### What should be done in the future

- Keep the imported source note linked from the design document and ticket index.

### Code review instructions

- Start with the ticket workspace at `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/10/XGOJA-TS-001--typescript-support-for-go-go-goja-xgoja-and-hot-reload`.
- Validate with `docmgr doc list --ticket XGOJA-TS-001`.

### Technical details

- Imported source path: `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/10/XGOJA-TS-001--typescript-support-for-go-go-goja-xgoja-and-hot-reload/sources/local/01-goja-typescript-esbuild-note.md`.

## Step 2: Map the current go-go-goja/xgoja/hot reload architecture

I inspected the repository to identify the actual seams where TypeScript support would attach. The important finding is that TypeScript support should not change goja runtime ownership or xgoja provider registration; it should compile `.ts`/`.tsx` sources before they reach the existing JavaScript parser and goja execution paths.

The architecture map showed three high-value integration points: `xgoja run`, jsverbs scanning/loading, and the HTTP provider's blue/green hot reload manager. The existing `.d.ts` generation path is complementary and should remain the editor-facing type contract.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Investigate the codebase thoroughly enough to write an evidence-backed intern guide.

**Inferred user intent:** Avoid a speculative design; ground recommendations in actual files, APIs, and runtime flows.

**Commit (code):** N/A — documentation-only work; no code commit was created in this step.

### What I did

- Listed repository directories and xgoja files with `find`.
- Searched for hot reload, TypeScript, esbuild, jsverbs, runtime, and provider concepts with `rg`.
- Read the important files:
  - `cmd/xgoja/internal/buildspec/build_spec.go`
  - `pkg/xgoja/app/runtime_spec.go`
  - `pkg/xgoja/app/run.go`
  - `pkg/xgoja/app/root.go`
  - `pkg/jsverbs/scan.go`
  - `pkg/jsverbs/runtime.go`
  - `pkg/xgoja/dtsgen/dtsgen.go`
  - `pkg/xgoja/providerapi/module.go`
  - `pkg/xgoja/hotreload/manager.go`
  - `pkg/xgoja/hotreload/watch.go`
  - `pkg/xgoja/providers/http/serve.go`
  - `pkg/engine/factory.go`
  - `pkg/engine/module_roots.go`
  - `cmd/xgoja/doc/14-tutorial-typescript-declarations.md`
- Generated a line-number evidence bundle in `/tmp/xgoja-ts-evidence.txt` using `nl -ba` for local reference while writing.

### Why

- The design guide needed concrete file references and line-anchored evidence.
- TypeScript support crosses several subsystems, so the implementation plan needed to avoid changing the wrong layer.

### What worked

- The existing system has clear seams:
  - `JSVerbSourceSpec.Extensions` already exists in build and runtime specs.
  - jsverbs already has a `RequireLoader()` seam.
  - hot reload already rescans sources and swaps successful candidate runtimes.
  - `.d.ts` generation already models xgoja module aliases.

### What didn't work

- No command failed in this step.
- The investigation confirmed that simply adding `.ts` to `extensions` is insufficient because jsverbs parses with the JavaScript tree-sitter grammar and returns original source to goja.

### What I learned

- `xgoja run` currently executes script files through `rt.Require.Require(scriptPath)`, not raw `vm.RunScript`.
- jsverbs runtime invocation requires the module path and then looks up captured functions in `globalThis.__glazedVerbRegistry`.
- HTTP hot reload is already blue/green: it loads and smoke-tests a candidate before swapping it into the active snapshot.
- Provider modules already carry TypeScript descriptor metadata through `providerapi.Module.TypeScript`.

### What was tricky to build

- The tricky design issue is overlay ordering for bundled TypeScript jsverbs. If esbuild bundles a file, top-level functions may be wrapped in closures. Therefore the jsverbs overlay that captures functions should be appended to the original TypeScript source before bundling, not after bundling the generated JavaScript.

### What warrants a second pair of eyes

- The final implementation should carefully test CommonJS/IIFE output format choices for `xgoja run file.ts` and jsverb loader output.
- The `External` list passed to esbuild must include xgoja native module aliases, or esbuild may incorrectly try to resolve provider modules from npm.

### What should be done in the future

- When implementation begins, create a small failing test for each seam before adding the compiler facade.

### Code review instructions

- Review the design doc's "Current-state architecture" section against the files listed above.
- Validate repository claims with the line references in the design document.

### Technical details

- Current jsverbs default extensions are `.js` and `.cjs`.
- Current HTTP hot reload watch extensions default to `.js`, `.json`, `.md`, `.yaml`, and `.yml`.
- Current `xgoja gen-dts` documentation already describes how users generate editor declarations for xgoja module aliases.

## Step 3: Write the intern-oriented design and implementation guide

I wrote the primary design document as a detailed implementation guide rather than a terse summary. It explains the existing system, the TypeScript gaps, the proposed compiler facade, the xgoja schema changes, jsverbs scan/load changes, `xgoja run` behavior, hot reload integration, tests, risks, and phased work items.

The guide intentionally separates runtime compilation from type checking. esbuild is the recommended runtime compiler; `tsc --noEmit` remains a user/project CI concern.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Produce the requested long-form technical design and store it in the ticket.

**Inferred user intent:** Hand a new intern a clear, evidence-backed implementation plan with system context and concrete next steps.

**Commit (code):** N/A — documentation-only work; no code commit was created in this step.

### What I did

- Rewrote `design/01-typescript-support-analysis-and-implementation-guide.md` with:
  - executive summary;
  - problem statement and scope;
  - current-state architecture;
  - gap analysis;
  - proposed architecture;
  - pseudocode;
  - decision records;
  - phased implementation plan;
  - testing strategy;
  - risks and mitigations;
  - file reference map.
- Included ASCII diagrams and code sketches for the compiler facade, schema extensions, jsverb loader, and `xgoja run`.

### Why

- The user explicitly asked for a very clear and technical intern guide with prose paragraphs, bullet points, pseudocode, diagrams, API references, and file references.

### What worked

- The final design ties every major recommendation to a concrete file or existing API.
- The design preserves the existing runtime architecture and focuses TypeScript support around source compilation and loader integration.

### What didn't work

- No write command failed.

### What I learned

- The safest initial design is incremental: add a compiler facade first, then thread it through jsverbs and xgoja, then let HTTP hot reload inherit the behavior.
- Embedded and filesystem-backed sources need different compilation timing.

### What was tricky to build

- The design needed to balance two viable modes: production precompile for embedded sources and runtime compilation for editable hot reload sources. The resulting proposal keeps both modes but makes their ownership explicit.

### What warrants a second pair of eyes

- The API name `pkg/tsscript` is proposed, not final.
- The `TypeScriptSpec` schema should be reviewed before implementation to ensure it does not overfit to the first use case.

### What should be done in the future

- Convert the implementation phases into concrete code tasks when implementation starts.
- Add a tutorial example after the first working implementation lands.

### Code review instructions

- Start with `design/01-typescript-support-analysis-and-implementation-guide.md`.
- Pay special attention to:
  - `Decision: Bundle TypeScript jsverb runtime entries, not just transform them`;
  - `Phase 3: Add TypeScript-aware jsverb scan/load`;
  - `Phase 5: Update HTTP hot reload defaults`.

### Technical details

- The design recommends esbuild `api.ES2015` as the conservative default target for goja.
- It recommends marking xgoja native module aliases as esbuild externals.
- It recommends appending jsverb overlay code before bundling TypeScript runtime entries.

## Step 4: Validate ticket hygiene and upload to reMarkable

I ran docmgr validation before upload and fixed the one issue it found. The imported source note initially had no docmgr frontmatter and had a non-numeric filename because `docmgr import file` preserved the friendly import name. I renamed it to a numeric-prefixed Markdown source note and added minimal frontmatter so the ticket validates cleanly.

After validation passed, I uploaded a bundled PDF containing the index, design guide, diary, tasks, and imported source note to reMarkable under the ticket-specific remote directory.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Validate the ticket deliverables and publish them to reMarkable.

**Inferred user intent:** Make the research package readable off-machine and ensure the ticket is clean enough to continue later.

**Commit (code):** N/A — documentation-only work; no code commit was created in this step.

### What I did

- Ran `docmgr doctor --ticket XGOJA-TS-001 --stale-after 30`.
- Fixed the imported source note by renaming it from `sources/local/goja TypeScript esbuild note.md` to `sources/local/01-goja-typescript-esbuild-note.md` and adding frontmatter.
- Updated links and references in the index, design doc, and diary.
- Re-ran `docmgr doctor --ticket XGOJA-TS-001 --stale-after 30` successfully.
- Ran a reMarkable bundle dry run:
  - `remarquee upload bundle ... --name "XGOJA TS Support Design" --remote-dir "/ai/2026/06/10/XGOJA-TS-001" --toc-depth 2 --dry-run --non-interactive`
- Uploaded the bundle:
  - `remarquee upload bundle ... --name "XGOJA TS Support Design" --remote-dir "/ai/2026/06/10/XGOJA-TS-001" --toc-depth 2 --non-interactive`

### Why

- `docmgr doctor` catches ticket hygiene issues before handing off.
- The dry run verifies the exact PDF bundle contents and destination before upload.
- The final upload satisfies the requested reMarkable delivery.

### What worked

- `docmgr doctor` passed after the source note cleanup.
- The upload command returned: `OK: uploaded XGOJA TS Support Design.pdf -> /ai/2026/06/10/XGOJA-TS-001`.

### What didn't work

- First validation failed with:
  - `YAML/frontmatter syntax error`
  - `frontmatter delimiters '---' not found`
  - file: `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/10/XGOJA-TS-001--typescript-support-for-go-go-goja-xgoja-and-hot-reload/sources/local/goja TypeScript esbuild note.md`
- It also warned about `missing_numeric_prefix` for that imported source file.

### What I learned

- Imported markdown sources may need docmgr frontmatter and numeric prefixes when they live inside the ticket tree and are included in doctor validation.
- reMarkable bundle upload can include ticket index, design, diary, task list, and source note in one PDF.

### What was tricky to build

- The source file rename required updating references in multiple documents so the ticket remained navigable and doctor-clean.

### What warrants a second pair of eyes

- The design itself should be reviewed before implementation starts, especially the jsverbs overlay-before-bundling decision and the proposed TypeScript schema.

### What should be done in the future

- Start implementation with the `pkg/tsscript` facade and failing tests from the design guide.

### Code review instructions

- Validate ticket hygiene with `docmgr doctor --ticket XGOJA-TS-001 --stale-after 30`.
- Confirm uploaded bundle destination from the successful upload output: `/ai/2026/06/10/XGOJA-TS-001`.

### Technical details

- Bundle name: `XGOJA TS Support Design`.
- Bundle contents:
  - `index.md`
  - `design/01-typescript-support-analysis-and-implementation-guide.md`
  - `reference/01-investigation-diary.md`
  - `tasks.md`
  - `sources/local/01-goja-typescript-esbuild-note.md`
