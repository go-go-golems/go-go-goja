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
      Note: |-
        Bobatea REPL entrypoint inspected for runtime-side plugin and help integration constraints
        Bobatea js-repl wiring for the runtime-scoped docs module
    - Path: cmd/repl/main.go
      Note: |-
        Main REPL entrypoint inspected for current help and plugin surface wiring
        Line REPL wiring for the runtime-scoped docs module
    - Path: pkg/doc/15-docs-module-guide.md
      Note: User-facing help page documenting the docs module API
    - Path: pkg/repl/evaluators/javascript/evaluator.go
      Note: Evaluator integration point for docs-module and plugin/runtime registrars
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

## Step 5: Make plugin method docs first-class before building the hub

The first real implementation slice changed the plugin contract before anything else. That order matters. The documentation hub only works cleanly if plugin modules, exports, and methods are all honest first-class entries. The SDK already had method-level doc strings locally, but the manifest schema dropped them. I fixed that first instead of carrying the old limitation deeper into GOJA-11.

### What I did

- Extended `pkg/hashiplugin/contract/jsmodule.proto` with `MethodSpec`.
- Regenerated the protobuf Go code.
- Updated the SDK manifest builder so object methods emit rich doc metadata.
- Updated host validation, reification, and reporting to use `MethodSpec` instead of raw string method names.
- Added a retained runtime-scoped plugin manifest snapshot helper and a value slot on `engine.RuntimeModuleContext` so later registrars can consume loaded plugin metadata without global state.

### Why

- The docs hub should not invent synthetic method entries when the manifest can simply represent them.
- Runtime-scoped registrar value exchange is the cleanest way to let the later docs registrar see plugin metadata without reloading plugins or introducing a global registry.

### What worked

- The plugin SDK examples already used `sdk.ExportDoc(...)` for methods, so the new schema immediately made those docs visible.
- `go generate ./pkg/hashiplugin/contract` was enough to refresh the protobuf bindings cleanly.

### What didn't work

- The first compile pass failed in `pkg/hashiplugin/host/reify.go` because one loader path still tried to use the full `*MethodSpec` where a string method name was expected.

### Commit

- `af6c7e5` - `hashiplugin: add method-level plugin docs`

### Technical details

- Commands run:
  - `go generate ./pkg/hashiplugin/contract`
  - `go test ./pkg/hashiplugin/... ./engine/... -count=1`

## Step 6: Build the shared Go-side documentation hub and providers

With plugin method docs fixed, the next step was the actual shared core. I kept this slice intentionally small and explicit. `pkg/docaccess` defines a shared model and hub, and then three adapter packages turn the existing subsystems into providers instead of trying to rewrite them.

### What I did

- Added `pkg/docaccess` with:
  - source descriptors
  - entry references
  - shared entries
  - query shape
  - provider interface
  - hub implementation
- Added providers for:
  - Glazed help
  - jsdoc stores
  - retained plugin manifests
- Added focused tests for the hub and each provider.

### Why

- This is the real architectural center of GOJA-11.
- Once the providers exist, the JavaScript module becomes thin runtime glue instead of another place where the three subsystems get re-modeled ad hoc.

### What worked

- The adapter shape fit the current repository better than expected.
- The plugin provider could now expose method-level entries honestly because the earlier schema slice had already landed.

### What didn't work

- The first pass had a few small issues:
  - an over-strict test shape assumption in provider tests
  - one helper name that referenced a non-existent function
  - a couple of dead utility helpers that lint correctly rejected

### Commit

- `74b95a4` - `docaccess: add unified documentation providers`

### Technical details

- Commands run:
  - `go test ./pkg/docaccess/... -count=1`

## Step 7: Register `require("docs")` at runtime and wire it into both REPLs

Once the hub and providers existed, the runtime slice was mostly about ownership and surfacing. I added a runtime-scoped registrar that builds a hub for each runtime from configured help/jsdoc sources plus the plugin manifest snapshot left by the plugin registrar, and then registers one JS-facing `docs` module.

### What I did

- Added `pkg/docaccess/runtime/registrar.go`.
- Added JS exports:
  - `sources()`
  - `search(query)`
  - `get(ref)`
  - `byID(sourceId, kind, id)`
  - `bySlug(sourceId, slug)`
  - `bySymbol(sourceId, symbol)`
- Extended the JavaScript evaluator config so `js-repl` can add runtime registrars cleanly.
- Wired the registrar into:
  - `cmd/repl/main.go`
  - `cmd/js-repl/main.go`
- Added runtime integration tests that cover:
  - help + jsdoc sources
  - plugin method docs through `require("docs")`

### Why

- The goal of GOJA-11 was explicit: JavaScript users should be able to inspect the rich documentation systems from inside the runtime, not only from Go.

### What worked

- The runtime-scoped registrar model fit naturally into the existing `engine.Factory` design.
- The plugin registrar and docs registrar compose cleanly when the plugin registrar runs first.

### What didn't work

- A runtime integration test initially redeclared `const docs` across multiple `RunString(...)` calls and hit a JS `SyntaxError`. Switching to inline `require("docs").bySymbol(...)` style fixed that.

### Commit

- `1b8d2ef` - `repl: expose unified docs module`

### Technical details

- Commands run:
  - `go test ./pkg/docaccess/runtime ./pkg/repl/evaluators/javascript ./cmd/repl ./cmd/js-repl -count=1`

## Step 8: Add user-facing docs and decide the remaining deferred edges

The implementation was usable at this point, but still a bit hidden. I added a dedicated help page for the `docs` module, linked it from REPL usage docs, and updated the line REPL `:help` output so the feature is discoverable. I also revisited the remaining two policy questions from the task list.

Decisions:

- `modules/glazehelp` remains in place for now as a legacy surface, but GOJA-11 does not rework it into a wrapper in this slice.
- optional `docmgr` provider support is explicitly deferred; the core three-source system is now stable enough that this deferment is a conscious product decision rather than an omission.

### Commit

- `78bdf97` - `docs: teach unified docs module`

## Step 9: Close out the ticket and publish the refreshed bundle

Once the implementation and user-facing docs were in place, the remaining work was ticket hygiene. I updated the GOJA-11 task list and changelog to reflect the implementation slices, related the new code paths back into the design and diary docs, and reran the ticket validation and publication flow so the ticket state matches the repository state.

This closeout matters because GOJA-11 is now both design and implementation history. Without the final relate/doctor/upload pass, the ticket would still describe the right architecture but would not point reviewers at the actual implementation files or the refreshed reMarkable bundle.

### What I did

- Related the implemented `pkg/docaccess`, runtime wiring, and plugin-contract files back into the GOJA-11 design doc and diary.
- Re-ran `docmgr doctor --ticket GOJA-11-DOC-ACCESS-SURFACES --stale-after 30`.
- Dry-ran and then uploaded the refreshed bundle to reMarkable.
- Verified the remote listing under `/ai/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES`.

### Why

- The ticket needs to remain a trustworthy handoff artifact for future work.
- The user explicitly asked for the ticket deliverables to be stored there and uploaded to reMarkable.

### What worked

- `docmgr doctor` passed cleanly after the new related-file links were added.
- The refreshed upload succeeded without requiring any additional formatting fixes.

### What didn't work

- One `docmgr doc relate` attempt failed because a file note included the literal text `require("docs")`, which broke CSV-style parsing inside the CLI flag handler. Re-running the command with plain-text wording fixed it.

### What I learned

- Ticket closeout commands are sensitive to quoting inside `--file-note` values, so notes should avoid embedded quotes unless they are absolutely necessary.

### Technical details

- Commands run:
  - `docmgr doc relate --doc /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES--unified-documentation-access-surfaces-for-go-and-javascript-runtimes/design-doc/01-unified-documentation-access-architecture-and-implementation-guide.md --file-note "..."`
  - `docmgr doc relate --doc /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES--unified-documentation-access-surfaces-for-go-and-javascript-runtimes/reference/01-investigation-diary.md --file-note "..."`
  - `docmgr doctor --ticket GOJA-11-DOC-ACCESS-SURFACES --stale-after 30`
  - `remarquee upload bundle ttmp/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES--unified-documentation-access-surfaces-for-go-and-javascript-runtimes --dry-run --name "GOJA-11 Unified documentation access surfaces" --remote-dir "/ai/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES" --toc-depth 2`
  - `remarquee upload bundle ttmp/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES--unified-documentation-access-surfaces-for-go-and-javascript-runtimes --force --name "GOJA-11 Unified documentation access surfaces" --remote-dir "/ai/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES" --toc-depth 2`
  - `remarquee cloud ls /ai/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES --long --non-interactive`
