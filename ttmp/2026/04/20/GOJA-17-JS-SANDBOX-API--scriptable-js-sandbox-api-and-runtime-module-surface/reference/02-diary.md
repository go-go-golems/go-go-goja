---
Title: Diary
Ticket: GOJA-17-JS-SANDBOX-API
Status: active
Topics:
    - goja
    - js-bindings
    - architecture
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: Runtime factory behavior traced during the research phase
    - Path: pkg/jsverbs/runtime.go
      Note: Runtime invocation pattern that the sandbox API should keep separate from
    - Path: pkg/jsverbs/scan.go
      Note: Scan/extract behavior that informed the sandbox-vs-jsverbs distinction
ExternalSources: []
Summary: Chronological work log for the JS sandbox host API ticket.
LastUpdated: 2026-04-20T11:10:00-04:00
WhatFor: Track the research, design, and documentation work for the scriptable JS sandbox API.
WhenToUse: Use when reviewing how the sandbox API was shaped and which repo files informed the design.
---


# Diary

## Goal

Record the work performed on this ticket in a chronological, review-friendly way. The diary should show how the sandbox API was derived from the existing goja runtime, how the current code influenced the design, and what remains open for implementation.

## Step 1: Map the current goja runtime and decide what the sandbox actually is

I started by tracing the current runtime and module architecture in `go-go-goja` before writing any API prose. That turned out to matter a lot: the repository already has a strong JS execution model, so the new sandbox API should build on those seams instead of inventing another runtime story.

The most important discovery was that the existing `pkg/jsverbs` subsystem already solves a command-scanning problem, not a bot-scripting problem. That distinction shaped the rest of the work: the sandbox API should be a runtime-scoped host capability layer with in-memory state and explicit builder helpers, while `pkg/jsverbs` stays the precedent for source scanning and Glazed command compilation.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket to add JS from go-go-goja (~/code/wesen/corporate-headquarters/go-go-goja) and expose this API. Use in memory storage at first, no need for perrmission handling, goja provides a way to register modules and such.

reate a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
references.
It should be very clear and detailed. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new docmgr ticket in the go-go-goja repo, design a scriptable JS sandbox API that uses goja’s module/runtime seams, keep storage in memory for v1, skip permissions handling, and write a very detailed intern-friendly analysis/design/implementation guide plus diary documentation for the ticket.

**Inferred user intent:** Produce a high-quality design package that explains how a future JS sandbox should fit into the existing go-go-goja runtime, with enough detail for a new contributor to understand the system and implement it later.

### What I did
- Created a new docmgr ticket for the JS sandbox host API work.
- Read the current runtime, module, owner-thread, and JS verb subsystems to identify the right architectural seam.
- Confirmed that `pkg/jsverbs` is the closest existing precedent, but not the same problem as the sandbox API.

### Why
- The ticket needs to build on the repo’s actual runtime shape instead of assuming a brand-new host model.
- Understanding the current code first helps avoid designing an API that fights the existing goja lifecycle.
- The sandbox should be capability-based and runtime-scoped, so the current engine and runtime-owner abstractions matter more than a naive module-only design.

### What worked
- The repo already has a clean runtime factory and owner-thread model, which makes a sandbox API feasible.
- The `engine.RuntimeModuleRegistrar` seam is a good fit for per-runtime bot state.
- The existing `pkg/jsverbs` docs are a strong writing and architecture precedent.

### What didn't work
- The first-pass intuition that this would be a completely new JS subsystem was wrong.
- The repository already has a more mature JS command pipeline than expected, so the new sandbox has to complement that instead of duplicating it.

### What I learned
- `go-go-goja` already has most of the mechanics needed for a JS sandbox: runtime ownership, module registration, require loaders, and owner-thread scheduling.
- The missing piece is a small, explicit host API for script authors.
- In-memory storage is enough for the first iteration and keeps the contract easy to understand.

### What was tricky to build
- The hardest part of the research phase was separating “JS command scanning” from “JS host scripting.”
- They both use goja, but they answer different questions and belong at different layers.
- It was also important not to over-design permissions or persistence before the core API shape was clear.

### What warrants a second pair of eyes
- Whether the sandbox module name should remain `sandbox`, or whether the host should expose a more domain-specific module name.
- Whether the builder-style API should use `defineBot(...)` only, or also expose lower-level constructors for advanced users.
- Whether the first implementation should live beside `pkg/jsverbs` or in a new host-oriented package tree.

### What should be done in the future
- Implement the runtime-scoped sandbox registrar.
- Add the in-memory store and command/event dispatch table.
- Add runtime tests for a tiny bot script such as `ping` and `count`.

### Code review instructions
- Start with `engine/factory.go`, `engine/runtime.go`, `engine/runtime_modules.go`, `pkg/runtimeowner/runner.go`, and `pkg/runtimebridge/runtimebridge.go` to understand the host lifecycle.
- Then review `pkg/jsverbs/scan.go`, `pkg/jsverbs/binding.go`, `pkg/jsverbs/command.go`, and `pkg/jsverbs/runtime.go` to see the existing JS-to-Glazed precedent.
- Finally check `cmd/jsverbs-example/main.go` and the jsverbs developer guide/reference docs to see how the current JS path is presented to users.

### Technical details
- The sandbox should be CommonJS-based (`require("sandbox")`), not ESM-only.
- v1 should use in-memory storage and skip permissions handling entirely.
- The host should prefer runtime-scoped registration over global module state.
- The current repo already uses `runtimeowner.Runner` for owner-thread-safe JS execution and `runtimebridge` for VM-local bindings.

## Step 2: Draft the design and reference docs for the sandbox API

After mapping the current architecture, I wrote the actual design package: one long-form design guide and one compact API reference with example bot scripts. The design guide explains the runtime shape in detail for a new intern, while the reference doc keeps the proposed JS-facing API easy to skim and reuse.

The main writing challenge was keeping the sandbox API small while still making it expressive enough for real bots. I settled on a capability-oriented `defineBot(...)` style with `command`, `event`, `reply`, `log`, and `store`, because it reads naturally in JS and still fits the existing Goja runtime model.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Turn the architecture research into a concrete design guide and a copy/paste-friendly API reference with example bots.

**Inferred user intent:** Give future implementers a clear, detailed blueprint they can follow without having to rediscover the runtime model or the intended JS API shape.

### What I did
- Wrote a detailed design document for the sandbox host API and runtime architecture.
- Wrote a compact JS API reference with example bot scripts.
- Kept the documentation intentionally CommonJS-focused and runtime-scoped.

### Why
- The design doc needs to explain the system in depth for a new contributor.
- The reference doc needs to give maintainers and implementers a fast way to check the intended JS surface.
- The two-doc split keeps the main architecture readable while still preserving copy/paste-ready examples.

### What worked
- The design guide could reuse concrete repo files as anchors for the architecture discussion.
- The API reference makes the proposed JS surface obvious at a glance.
- Example bots make the shape of the API easier to understand than prose alone.

### What didn't work
- A more “magical” API would have been shorter to describe, but it would have been harder to explain and harder to implement safely.
- The current repo already has a rich JS command pipeline, so I had to be careful to keep the sandbox API distinct from `pkg/jsverbs`.

### What I learned
- The best sandbox API is probably the one that looks boring in Go and pleasant in JS.
- In-memory storage is a strength here because it lets the API stay simple and testable.
- A runtime registrar is the right place to attach host state because the state belongs to a VM, not to the process globally.

### What was tricky to build
- The trickiest part of the design was deciding how much of the host to expose directly to JS.
- Too little API and the sandbox feels cramped; too much and it becomes a raw runtime escape hatch.
- The final shape tries to stay on the right side of that line by exposing only capabilities the host intentionally registered.

### What warrants a second pair of eyes
- Whether `ctx.reply` should stay Discord-like, or whether the API should also expose a transport-neutral emit/send abstraction.
- Whether the in-memory store should offer explicit namespace helpers in v1.
- Whether the implementation plan should lean on `engine.RuntimeModuleRegistrar` only, or split into separate registrar and dispatcher packages.

### What should be done in the future
- Implement the sandbox host API in Go.
- Add a tiny demo host or CLI harness.
- Add runtime tests proving that `defineBot`, `command`, `event`, and `store` behave as intended.

### Code review instructions
- Review the design doc first for the architecture and API shape.
- Then review the reference doc for the exact JS-facing names and examples.
- Cross-check the proposal against `pkg/jsverbs` so the new sandbox API stays distinct from the command-scanning pipeline.

### Technical details
- The design uses CommonJS and `require("sandbox")` as the JS entrypoint.
- `defineBot(...)` is the recommended top-level builder entrypoint.
- The `ctx` object intentionally exposes only a small capability set: `reply`, `defer`, `log`, `store`, and runtime metadata.
- The host-side storage model is in-memory only for v1.

## Step 3: Relate the docs, validate the workspace, and deliver the bundle to reMarkable

Once the design and reference docs were in place, I connected them back to the ticket index and validated the workspace with `docmgr doctor`. After that I bundled the ticket docs into one PDF and uploaded the bundle to reMarkable so it can be reviewed offline as a single artifact.

The only real snag was a path typo in the first reMarkable dry-run. Fixing that reinforced why the ticket docs should use absolute paths when referencing workspace files: it is very easy to miss a single directory component in a long `ttmp/...` path.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish the ticket bookkeeping, verify the docs are healthy, and upload the deliverable bundle to reMarkable.

**Inferred user intent:** Make the design package easy to review and preserve the work in a portable PDF bundle.

**Commit (docs):** 42dec4a — "Document JS sandbox host API"

### What I did
- Related the design doc, API reference, and diary back to the ticket index.
- Ran `docmgr doctor --ticket GOJA-17-JS-SANDBOX-API --stale-after 30`.
- Ran a dry-run reMarkable upload, fixed a path typo, and then uploaded the final bundle.
- Verified the uploaded document appears in the remote `/ai/2026/04/20/GOJA-17-JS-SANDBOX-API` folder.

### Why
- The ticket needs to stay navigable from its index page.
- `docmgr doctor` is the fastest sanity check for related-file metadata and frontmatter health.
- The reMarkable bundle gives reviewers one PDF with a table of contents instead of several isolated docs.

### What worked
- `docmgr doctor` reported that all checks passed.
- The dry-run made it obvious how the bundle would be laid out before upload.
- The final upload succeeded and the remote listing confirmed the PDF is present.

### What didn't work
- The first reMarkable dry-run failed because I mistyped one of the long `ttmp` paths.
- The exact error was:

  `Error: path not found: /home/manuel/code/wesen/corporative-headquarters/go-go-goja/ttmp/...`

- After correcting the path, the dry-run and upload both succeeded.

### What I learned
- The ticket docs are easiest to review when the index, design doc, reference doc, and diary are all connected.
- For long workspace paths, a path typo is more likely than a formatting problem.
- The reMarkable bundle workflow is a useful last-mile delivery check because it exposes broken paths early.

### What was tricky to build
- The tricky part here was not the upload itself; it was making sure every long absolute path was spelled exactly right.
- The long `ttmp/...` paths are reliable once entered correctly, but they are long enough that a typo is easy to miss in review.

### What warrants a second pair of eyes
- Whether the sandbox API should expose a transport-neutral emit helper in addition to `ctx.reply`.
- Whether the first implementation should stay focused on command/event handling only, or whether storage namespacing should be added immediately.
- Whether the design should eventually reuse any of the `pkg/jsverbs` command-compiler pieces for script-driven host bots.

### What should be done in the future
- Implement the sandbox host API in Go.
- Add runtime tests and a small demo host.
- Revisit the API reference once the actual code exists, especially if the host chooses slightly different names.

### Code review instructions
- Start with the ticket index, then read the design doc and API reference together.
- Check the reMarkable bundle output if you want to review the docs as a single delivery package.
- Use `docmgr doctor` as the quick integrity check before any further edits.

### Technical details
- Validation command used: `docmgr doctor --ticket GOJA-17-JS-SANDBOX-API --stale-after 30`.
- Upload command used: `remarquee upload bundle --name "GOJA-17 JS Sandbox API" --remote-dir "/ai/2026/04/20/GOJA-17-JS-SANDBOX-API" ...`.
- Remote location verified with: `remarquee cloud ls /ai/2026/04/20/GOJA-17-JS-SANDBOX-API --long --non-interactive`.

## Related

- `design-doc/01-js-sandbox-host-api-and-runtime-architecture.md`
- `reference/01-js-sandbox-api-reference-and-example-bots.md`
