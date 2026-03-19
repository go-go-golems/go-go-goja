---
Title: Investigation diary
Ticket: GOJA-12-JS-REPL-DOC-AWARE-HELP
Status: active
Topics:
    - goja
    - repl
    - ui
    - architecture
    - tooling
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/js-repl/main.go
      Note: Current TUI entrypoint and source of the user-facing autocomplete/help configuration
    - Path: engine/runtime.go
      Note: Candidate location for stable runtime-scoped documentation state after setup
    - Path: pkg/repl/evaluators/javascript/docs_resolver.go
      Note: Evaluator-side resolver that translates aliases and property chains into docaccess entries and plugin-backed completion candidates
    - Path: pkg/docaccess/plugin/provider.go
      Note: Existing plugin doc provider exposing module/export/method entries
    - Path: pkg/docaccess/runtime/registrar.go
      Note: Existing runtime-scoped docs hub construction path from GOJA-11
    - Path: pkg/repl/evaluators/javascript/evaluator.go
      Note: Main current implementation of completions, help bar, and help drawer
    - Path: pkg/repl/evaluators/javascript/docs_resolver_test.go
      Note: Focused resolver and end-to-end evaluator coverage for plugin-backed docs-aware help
ExternalSources: []
Summary: Chronological record of how GOJA-12 was scoped and why the recommended architecture centers on direct evaluator access to the existing runtime-scoped docs hub rather than JS-side self-querying.
LastUpdated: 2026-03-18T19:30:03-04:00
WhatFor: Capture the reasoning, repository evidence, and closeout steps for the js-repl doc-aware help ticket.
WhenToUse: Use when continuing GOJA-12 implementation later or reviewing why the proposed design chose plugin-first resolver integration.
---


# Investigation diary

## Goal

Record how GOJA-12 was created, which code paths were inspected, what design options were considered, and why the final recommendation is to make the evaluator itself documentation-aware using runtime-scoped Go-side state.

## Context

GOJA-11 already established a unified docs hub and `require("docs")` module. The new problem is narrower and more interactive: the Bobatea `js-repl` should surface that same documentation during typing, not only after explicit docs queries from JavaScript.

The design challenge is to enrich three UI surfaces at once:

- autocomplete rows
- one-line help bar
- full help drawer

without duplicating documentation indexes or recursively calling back into JavaScript.

## Quick Reference

### User prompt (verbatim)

```text
Create a new ticket for all of that, with a detailed implementation / design / analysis doc.

Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable.
```

### Assistant interpretation

Create a new ticket for doc-aware `js-repl` autocomplete/help, inspect the current evaluator and docs-hub seams, write a detailed intern-facing design guide, relate it to the right code, validate it, and upload it to reMarkable.

### Main recommendation

Use the existing Go-side `docaccess.Hub` as the single source of truth, persist access to it on the owned runtime, and teach the evaluator to resolve module/export/method docs directly from that runtime-scoped state. Start with plugin docs first and preserve a generic seam for broader module docs later.

## Usage Examples

### If you are implementing GOJA-12

1. Read the design doc in `design-doc/01-doc-aware-js-repl-autocomplete-and-help-architecture-and-implementation-guide.md`.
2. Start from `engine/runtime.go` and `pkg/docaccess/runtime/registrar.go` to understand how runtime-scoped state can be made visible after runtime construction.
3. Then read `pkg/repl/evaluators/javascript/evaluator.go` and locate:
   - `CompleteInput`
   - `GetHelpBar`
   - `GetHelpDrawer`
   - `helpBarFromContext`
4. Only after that should you implement a resolver helper for plugin docs.

### If you are reviewing the design

Focus on:

- whether the evaluator should read the raw hub or a narrower resolver
- whether plugin-first scope is the right first slice
- whether `engine.Runtime` is the correct lifetime/ownership home for runtime-scoped docs state
- whether fallback behavior remains clean for built-ins and user-defined objects

## Related

- Design guide: `design-doc/01-doc-aware-js-repl-autocomplete-and-help-architecture-and-implementation-guide.md`
- Task list: `tasks.md`
- Changelog: `changelog.md`

## Step 1: Create GOJA-12 and identify the real integration seam

The first thing to avoid was a shallow answer like “just call `require("docs")` from the help drawer.” That would technically reuse the docs system, but it would do so at the wrong layer. I created a dedicated ticket first and then read the current `js-repl` evaluator, the docs runtime registrar, and the plugin docs provider together.

That immediately clarified the actual seam:

- the docs hub already exists at runtime
- the evaluator already knows the parser context and `require()` aliases
- what is missing is a stable owned-runtime path from evaluator code to docs-hub data

### What I did

- Created `GOJA-12-JS-REPL-DOC-AWARE-HELP`.
- Inspected:
  - `cmd/js-repl/main.go`
  - `pkg/repl/evaluators/javascript/evaluator.go`
  - `pkg/docaccess/runtime/registrar.go`
  - `pkg/docaccess/plugin/provider.go`
  - `engine/runtime.go`
  - `engine/runtime_modules.go`
  - `pkg/jsparse/repl_completion.go`

### Why

- The user asked for a design/analysis pass, not direct implementation.
- The right answer depends on understanding current ownership and lifetime boundaries.

### What worked

- The code already has most of the necessary building blocks.
- The design problem is much more about joining existing systems than inventing a new one.

### What didn't work

- There was no single pre-existing accessor from `Evaluator` to runtime-scoped registrar values after runtime construction. That is the main structural gap the ticket now centers on.

### What I learned

- GOJA-11 solved “how JavaScript code accesses docs.”
- GOJA-12 needs to solve “how the evaluator accesses the same docs without going through JavaScript.”

### Technical details

- Commands run:
  - `docmgr status --summary-only`
  - multiple `sed -n ...` inspections of the files listed above
  - `docmgr ticket create-ticket --ticket GOJA-12-JS-REPL-DOC-AWARE-HELP --title "Doc-aware autocomplete and help surfaces in js-repl" --topics goja,repl,ui,documentation,plugins`
  - `docmgr doc add --ticket GOJA-12-JS-REPL-DOC-AWARE-HELP --doc-type design-doc --title "Doc-aware js-repl autocomplete and help architecture and implementation guide"`
  - `docmgr doc add --ticket GOJA-12-JS-REPL-DOC-AWARE-HELP --doc-type reference --title "Investigation diary"`

## Step 2: Choose the architectural center and write the design

Once the seams were clear, the main architectural decision followed naturally: the docs hub must remain the source of truth, and the evaluator must consume it directly. I explicitly rejected two tempting but wrong shortcuts:

- calling `require("docs")` from evaluator code
- building a second documentation index just for the TUI

The design document therefore focuses on:

- runtime-scoped persistence of docs state
- an evaluator-side resolver over that state
- plugin-first contextual resolution for module, export, and method docs
- reuse of existing completion/help surfaces rather than a UI rewrite

### What I did

- Wrote an intern-facing design doc with:
  - subsystem inventory
  - problem framing
  - runtime ownership model
  - diagrams
  - pseudocode
  - phased implementation plan
  - test matrix
  - future path for “modules in general”

### Why

- The ticket should be implementable by someone who did not build GOJA-11.
- The evaluator/docs-hub integration is subtle enough that the “why” matters as much as the “what.”

### What worked

- Plugin docs are a strong first slice because the provider already exposes module/export/method entries.
- The existing alias-aware completion model makes plugin contextual lookup straightforward.

### What didn't work

- Nothing failed technically in the design step, but one constraint became clearer: built-in module docs should remain a follow-up, not part of the first implementation slice.

### What warrants a second pair of eyes

- Whether the owned runtime should store the raw hub or a narrower resolver object.
- Whether source IDs like `plugin-manifests` should be centralized before implementation starts.

## Step 3: Ticket hygiene, validation, and publication

The ticket should be usable as a real handoff artifact, not just a local markdown draft. That means relating the design to the relevant code, validating the workspace, and publishing the bundle to reMarkable.

### What I did

- Related the design and diary to the key evaluator/runtime/docs files.
- Ran `docmgr doctor --ticket GOJA-12-JS-REPL-DOC-AWARE-HELP --stale-after 30`.
- Dry-ran and then uploaded the bundle to reMarkable.
- Verified the remote listing.

### Why

- The user explicitly asked for the design to be stored in the ticket and uploaded.
- Validation and publication make the ticket durable and reviewable later.

### Technical details

- Commands run:
  - `docmgr doc relate ...`
  - `docmgr doctor --ticket GOJA-12-JS-REPL-DOC-AWARE-HELP --stale-after 30`
  - `remarquee upload bundle ... --dry-run ...`
  - `remarquee upload bundle ... --force ...`
  - `remarquee cloud ls /ai/2026/03/18/GOJA-12-JS-REPL-DOC-AWARE-HELP --long --non-interactive`

## Step 4: Implement evaluator-side plugin docs resolution and UI integration

The implementation ended up being slightly broader than the initial “read the hub and render docs” sketch. The first focused test showed that plugin-aware help needs two separate behaviors:

- resolve docs for an already-known symbol such as `kv.store.get`
- contribute completion candidates for symbols that do not exist in the runtime object graph yet, such as `kv.store.g`

That second requirement matters because the evaluator does not execute `require("plugin:...")` during typing, and the existing runtime completion logic only knows how to inspect simple runtime identifiers or a small static module catalog.

### What I changed

- Added `pkg/repl/evaluators/javascript/docs_resolver.go`.
  - Reads the runtime-scoped `docaccess.Hub` from `engine.Runtime`.
  - Indexes plugin module, export, and method entries.
  - Resolves:
    - alias -> module docs
    - alias.property -> export docs
    - alias.object.method -> method docs
  - Supplies manifest-backed completion candidates for exports and methods.
- Updated `pkg/repl/evaluators/javascript/evaluator.go`.
  - Always installs the docs registrar when plugin directories are configured, not only when Glazed/jsdoc sources exist.
  - Stores a resolver on the evaluator when it owns the runtime.
  - Merges plugin-doc completion candidates into `CompleteInput`, `GetHelpBar`, and `GetHelpDrawer`.
  - Prefers docs-derived help text before falling back to static signatures or runtime inspection.
  - Renders documentation bodies, metadata, and related refs in the help drawer.
- Added `pkg/repl/evaluators/javascript/docs_resolver_test.go`.
  - Synthetic manifest test for module/export/method resolution.
  - End-to-end test that builds `plugins/examples/kv`, loads it through plugin discovery, and asserts docs-aware completions/help.

### Why this shape worked

- It keeps the docs hub as the single source of truth.
- It avoids running JavaScript during autocomplete/help.
- It keeps the evaluator-specific parsing and alias logic local to the evaluator package.
- It still leaves a future seam for non-plugin module docs because the resolver is a separate helper, not embedded into the TUI rendering code.

### What was trickier than expected

The original plan assumed plugin docs would only decorate existing completion candidates. That was insufficient. For nested plugin expressions like `kv.store.g`, the evaluator had no candidate source at all, because:

- static parser candidates do not know plugin manifests
- runtime property inspection only handles simple identifiers
- `NodeModuleCandidates(...)` only covers the built-in `fs` / `path` / `url` catalog

The fix was to let the resolver produce candidates directly from the plugin manifest index.

### What remains intentionally out of scope

- No native-module (`fs`, `path`, `url`, `docs`) doc provider is wired into the resolver yet.
- Source ID centralization is still unresolved.
- The resolver is generic enough to admit more sources later, but the first implementation is plugin-first on purpose.

### Technical details

- Commands run:
  - `go test ./pkg/repl/evaluators/javascript -count=1`
  - `gofmt -w pkg/repl/evaluators/javascript/evaluator.go pkg/repl/evaluators/javascript/docs_resolver.go pkg/repl/evaluators/javascript/docs_resolver_test.go`
