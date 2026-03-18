---
Title: Investigation diary
Ticket: GOJA-11-DOC-ACCESS-SURFACES
Status: active
Topics:
    - goja
    - architecture
    - tooling
    - js-bindings
    - glazed
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-jsdoc/doc/01-jsdoc-system.md
      Note: Existing intern-facing jsdoc guide used as evidence for the standalone documentation system already present
    - Path: cmd/js-repl/main.go
      Note: Bobatea REPL entrypoint inspected for runtime-side plugin and help integration constraints
    - Path: cmd/repl/main.go
      Note: Main REPL entrypoint inspected for current help and plugin surface wiring
    - Path: ttmp/2025-07-30/01-design-for-a-go-go-goja-module-for-the-glazed-helpsystem.md
      Note: Historical design note reviewed as background for the older glazehelp approach
ExternalSources: []
Summary: Chronological record of how the GOJA-11 documentation-access design ticket was scoped, researched in the repository, and prepared for future implementation.
LastUpdated: 2026-03-18T15:42:01.316237139-04:00
WhatFor: Capture why the unified documentation access ticket exists, what repo evidence informed the recommendation, and how the deliverable was validated and published.
WhenToUse: Use when reviewing the GOJA-11 design rationale or continuing implementation later.
---


# Investigation diary

## Goal

Document how GOJA-11 was created, what parts of the repository were inspected, what architectural options were considered, and why the recommended design is a runtime-scoped documentation hub with provider adapters.

## Context

This ticket is design-first. No implementation was requested yet. The goal is to create a future-facing but practical design that explains how to unify documentation access across:

- Glazed help pages
- jsdocex-extracted JavaScript docs
- plugin metadata
- and later, optionally, docmgr metadata

The key challenge is that these are not identical systems. They overlap conceptually, but they differ in ownership model, richness, and current surface API.

## Quick Reference

### Prompt Context

**User prompt (verbatim):**

```text
ok, let's do something interesting. I want to offer a way for users of the goja go side of things, but also users of the JS side of things to be able to aspect the rich documentation systems we have:

- glazed help (I have an old attempt in the modules that I don't know is great or not)
- jsdocex (to extract documentation from JS code)
- plugin meta information

Potentially I would also allow offering the docmgr metadata based doc navigation, but that's optional and is more about informing the design for the future.

Create a new ticket.

Do a deep design and thinking pass and do a design brainstorm. Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable.
```

**Assistant interpretation:** Create a new design ticket, inspect the current documentation-related subsystems in the repo, write an intern-facing architecture and implementation guide with design alternatives and a recommended direction, validate the ticket, and upload it to reMarkable.

**Inferred user intent:** Unify documentation access pragmatically, not just by adding another narrow wrapper, and make sure the next implementation step has a coherent architecture to follow.

### Key Repository Evidence Reviewed

- `modules/glazehelp/glazehelp.go`
- `modules/glazehelp/registry.go`
- `modules/glazehelp/glazehelp_test.go`
- `pkg/doc/doc.go`
- `cmd/repl/main.go`
- `cmd/js-repl/main.go`
- `pkg/repl/evaluators/javascript/evaluator.go`
- `pkg/jsdoc/model/model.go`
- `pkg/jsdoc/model/store.go`
- `cmd/goja-jsdoc/main.go`
- `cmd/goja-jsdoc/doc/01-jsdoc-system.md`
- `pkg/hashiplugin/contract/jsmodule.proto`
- `pkg/hashiplugin/host/reify.go`
- `pkg/hashiplugin/host/report.go`
- `pkg/hashiplugin/sdk/module.go`
- `engine/runtime_modules.go`
- `engine/factory.go`
- `ttmp/2025-07-30/01-design-for-a-go-go-goja-module-for-the-glazed-helpsystem.md`

### Main Conclusion

The old `glazehelp` module is a good proof of concept but a bad architectural center. The right center is a runtime-scoped Go-side hub with providers for Glazed help, jsdoc stores, and plugin metadata, then one JS-facing `docs` module layered on top.

## Usage Examples

### If you are implementing GOJA-11 later

1. Read the design guide in `design-doc/01-unified-documentation-access-architecture-and-implementation-guide.md`.
2. Start from the shared core package (`pkg/docaccess`) before touching any JS module wiring.
3. Implement the Glazed provider first, then jsdoc, then plugin metadata.
4. Only after the Go-side hub works should you add a new runtime-scoped native module.

### If you are reviewing the design

Focus review on:

- whether the shared model is too broad or too narrow,
- whether runtime-scoped ownership is the right choice,
- whether plugin metadata needs a richer retained source than `LoadReport`,
- whether `docmgr` should remain explicitly out of the MVP.

## Related

- Design guide: `design-doc/01-unified-documentation-access-architecture-and-implementation-guide.md`
- Ticket task list: `tasks.md`
- Changelog: `changelog.md`

## Step 1: Create GOJA-11 and inspect the current documentation surfaces

The first step was to resist the temptation to jump directly to a new JS module API. There are already multiple documentation-bearing systems in the repository, and the real design problem is deciding what should be shared between them versus what should remain source-specific. I created a focused ticket first, then read the current seams in the codebase.

The inspection quickly showed that there are three serious documentation systems and one prototype wrapper:

- Glazed help, loaded into `help.HelpSystem` and wired into Cobra commands
- jsdocex, migrated as `pkg/jsdoc` plus the `goja-jsdoc` CLI/server
- plugin metadata, present in manifests and SDK declarations but not really navigable yet
- `modules/glazehelp`, which is a useful old attempt but too narrow to be the long-term abstraction

### What I did

- Created `GOJA-11-DOC-ACCESS-SURFACES`.
- Read the existing Glazed help wrapper module and registry.
- Read the REPL and js-repl entrypoints to see how help and plugin status are exposed today.
- Read the jsdoc model/store and CLI help docs.
- Read the plugin manifest, reification, and report code to see exactly what metadata exists versus what is currently exposed.
- Read the old 2025 Glazed help design note as historical context.

### Why

- The user explicitly wanted a deep design and brainstorm, not a shallow API sketch.
- The best design here has to respect the existing runtime/module architecture rather than inventing a parallel documentation world.

### What worked

- The repository already contains enough architecture evidence to make a grounded recommendation.
- The old `glazehelp` attempt is especially useful because it shows both a viable pattern and its limitations.

### What didn't work

- Nothing failed technically in this step, but the design tension became clear fast: there is no honest way to pretend that Glazed help sections, jsdoc symbols, and plugin manifests are all the same kind of object.

### What I learned

- The right move is not “pick one existing system and stretch it”. It is to build a small hub abstraction over adapters.

### Technical details

- Commands run:
  - `docmgr status --summary-only`
  - `docmgr ticket create-ticket --ticket GOJA-11-DOC-ACCESS-SURFACES --title "Unified documentation access surfaces for Go and JavaScript runtimes" --topics goja,architecture,tooling,js-bindings,glazed`
  - `docmgr doc add --ticket GOJA-11-DOC-ACCESS-SURFACES --doc-type design-doc --title "Unified documentation access architecture and implementation guide"`
  - `docmgr doc add --ticket GOJA-11-DOC-ACCESS-SURFACES --doc-type reference --title "Investigation diary"`
  - multiple `sed -n ...` and `rg ...` inspections over the files listed above

## Step 2: Choose the recommended architecture and write the intern-facing guide

The central design decision was to recommend a shared documentation hub with provider adapters instead of either keeping separate modules or forcing a giant universal schema. That recommendation came out of comparing four real options:

- leave everything separate
- flatten everything into one universal type
- hub plus providers
- build directly around docmgr

The hub-plus-providers model is the only one that solves the user problem without discarding the strengths of the existing systems. It also fits the runtime architecture already used by plugins, because the engine supports runtime-scoped registration through `RuntimeModuleRegistrar`.

### What I did

- Wrote the design guide with:
  - subsystem inventory
  - problem framing
  - design brainstorm
  - recommended architecture
  - package/API sketches
  - pseudocode
  - diagrams
  - risks and phased implementation plan
- Made the guide explicitly intern-friendly and repository-grounded.

### Why

- The future implementation needs more than a list of tasks; it needs a coherent mental model.

### What worked

- The existing code split naturally into provider candidates: Glazed, jsdoc, and plugin metadata.
- The engine runtime seam made the runtime-scoped portion straightforward to explain.

### What didn't work

- The only real gap is plugin method-level docs. The current protobuf schema does not store them, so the design has to be honest about that limitation.

### What warrants a second pair of eyes

- Whether the shared `Entry` model is the right size.
- Whether the JS module should be named `docs` or something more explicit.

## Step 3: Prepare ticket bookkeeping, validation, and publication

This step exists so the design is usable as a real ticket artifact rather than a loose markdown note. The plan is to keep the ticket active, because the design is complete but the implementation has not been done yet.

### What I did

- Related the main design doc and diary to the most relevant code files.
- Ran `docmgr doctor --ticket GOJA-11-DOC-ACCESS-SURFACES --stale-after 30`.
- Dry-ran the reMarkable bundle upload.
- Uploaded the final bundle to `/ai/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES`.
- Verified the remote listing.

### Why

- The design ticket is supposed to be durable and reviewable.
- Validation and publication are part of the deliverable, not optional cleanup.

### What worked

- `docmgr doctor` passed cleanly after the design docs and related-file metadata were in place.
- The bundled upload worked without further formatting fixes.

### What didn't work

- Nothing failed in this closeout step.

### Technical details

- Commands run:
  - `docmgr doc relate --doc ...design-doc/01-unified-documentation-access-architecture-and-implementation-guide.md --file-note "..."`
  - `docmgr doc relate --doc ...reference/01-investigation-diary.md --file-note "..."`
  - `docmgr doctor --ticket GOJA-11-DOC-ACCESS-SURFACES --stale-after 30`
  - `remarquee status`
  - `remarquee cloud account --non-interactive`
  - `remarquee upload bundle ttmp/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES--unified-documentation-access-surfaces-for-go-and-javascript-runtimes --dry-run --name "GOJA-11 Unified documentation access surfaces" --remote-dir "/ai/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES" --toc-depth 2`
  - `remarquee upload bundle ttmp/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES--unified-documentation-access-surfaces-for-go-and-javascript-runtimes --force --name "GOJA-11 Unified documentation access surfaces" --remote-dir "/ai/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES" --toc-depth 2`
  - `remarquee cloud ls /ai/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES --long --non-interactive`
- Validation result:
  - `docmgr doctor`: all checks passed
  - reMarkable listing: `GOJA-11 Unified documentation access surfaces`

## Step 4: Re-scope GOJA-11 around first-class plugin method docs

Once implementation became part of the request, one part of the earlier design needed to change immediately. The original guide treated method-level plugin docs as a limitation of the existing protobuf contract and designed around that absence. That is the wrong tradeoff now. Since the plugin surface has not been published yet, the right move is to change the contract directly and make method docs first-class before the rest of the documentation hub is built.

This improves the rest of GOJA-11:

- the plugin provider can expose real method entries instead of synthetic placeholders
- JavaScript callers can inspect plugin methods with honest rich bodies
- the shared docs model stays cleaner because plugin methods are real entries, not inferred names

### What I changed in the ticket plan

- Declared `docs` as the intended JS module name.
- Declared that plugin method docs should be added to the protobuf contract without backward compatibility baggage.
- Expanded the task list into concrete implementation phases covering:
  - schema regeneration
  - SDK and host updates
  - provider implementation
  - runtime/module wiring
  - tests

### Why

- Temporary compatibility complexity would only slow down the implementation and muddy the data model.
- First-class method docs make the unified documentation story materially better.

### What warrants a second pair of eyes

- Whether `modules/glazehelp` should survive as a thin alias once `docs` exists.
