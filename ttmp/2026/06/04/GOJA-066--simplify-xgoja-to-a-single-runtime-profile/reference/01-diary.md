---
Title: Diary
Ticket: GOJA-066
Status: active
Topics:
    - xgoja
    - refactor
    - code-generation
    - config
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md
      Note: Documented --debug-panic-stack before creating GOJA-066
    - Path: cmd/xgoja/internal/buildspec/build_spec.go
      Note: Current build-time multi-profile schema that shaped the design
    - Path: pkg/xgoja/app/factory.go
      Note: Current profile-based runtime factory that shaped the implementation plan
    - Path: ttmp/2026/06/04/GOJA-066--simplify-xgoja-to-a-single-runtime-profile/design-doc/01-single-runtime-profile-simplification-design-and-implementation-guide.md
      Note: Primary GOJA-066 design and implementation guide
ExternalSources: []
Summary: Chronological diary for GOJA-066, the single-runtime-profile xgoja simplification design work.
LastUpdated: 2026-06-04T20:15:00-04:00
WhatFor: Use to continue GOJA-066 implementation with the investigation context, decisions, and validation plan.
WhenToUse: Before changing xgoja buildspec/runtime schema, generated command runtime selection, provider command contexts, docs, or examples.
---


# Diary

## Goal

This diary records the setup and design work for GOJA-066: simplifying xgoja so generated binaries have one runtime module set instead of multiple named runtime profiles.

## Step 1: Create the single-runtime-profile simplification ticket and design guide

This step created GOJA-066 and wrote the initial implementation guide. The work focused on mapping the current multi-profile architecture before recommending changes, because the word "runtime" appears in several layers: build-time YAML DTOs, embedded runtime JSON, generated command settings, provider command context, and the lower-level `pkg/engine` runtime.

The resulting design recommends a clean xgoja schema simplification: replace `runtimes.<name>.modules` with top-level `modules`, remove command-level `runtime` selectors, and keep only a temporary internal context label such as `main` for provider APIs that still expect a `RuntimeProfile` string.

### Prompt Context

**User prompt (verbatim):** "document it. 

Then, create a new ticket to remove the multiple runtime profiels from xgoja. We have only one profile and that's it. We might come back to it later, but for now we want to have a cleaner and simple xgoja.

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


Keep a diary."

**Assistant interpretation:** First document the newly added xgoja `--debug-panic-stack` flag, then create a new docmgr ticket for removing multi-runtime-profile support from xgoja and write a detailed intern-ready design/implementation guide, diary, and reMarkable upload.

**Inferred user intent:** Make xgoja simpler and easier to understand by removing an underused abstraction, while preserving a clear design trail and producing a polished document bundle for review.

**Commit (code):** N/A — documentation/design step only.

### What I did

- Documented the `--debug-panic-stack` flag in:
  - `cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md`
- Committed that documentation as:
  - `ebcd9a9` — `Document xgoja debug panic stack flag`
- Created ticket `GOJA-066`:
  - `ttmp/2026/06/04/GOJA-066--simplify-xgoja-to-a-single-runtime-profile`
- Added a design document:
  - `design-doc/01-single-runtime-profile-simplification-design-and-implementation-guide.md`
- Added this diary:
  - `reference/01-diary.md`
- Inspected the current multi-profile implementation in buildspec, generation, runtime app DTOs, built-in commands, runtime factory, command providers, examples, and docs.
- Wrote the initial design guide with:
  - current-state explanation;
  - schema/API references;
  - ASCII diagrams;
  - pseudocode;
  - phase-by-phase implementation plan;
  - decision records;
  - validation checklist;
  - file reference index.
- Updated GOJA-066 tasks to track implementation phases.
- Related key source files to the design document with `docmgr doc relate`.

### Why

The current multi-profile design gives each generated command a runtime selector even though current usage overwhelmingly uses a single profile named `main` or `repl`. The abstraction makes generated xgoja harder to explain and touches many files. Writing the design first gives a future implementer a safe map of what to remove and what to preserve.

### What worked

- Repository search quickly identified the main profile-related symbols:
  - `Runtimes`
  - `RuntimeProfile`
  - `Runtime string`
  - `firstRuntime`
  - `commandRuntime`
  - `selectedModuleDescriptors`
  - `sectionsForRuntimeProfile`
  - `NewRuntimeFromSections`
- Line-anchored inspection showed the profile map is mostly a selector around `Modules []ModuleInstanceSpec`, which supports the recommendation to lift modules to the top level.
- `docmgr doc relate` successfully attached the source files that should guide the implementation.

### What didn't work

- I attempted to read `cmd/xgoja/internal/buildspec/defaults.go`, but defaults are implemented inside `cmd/xgoja/internal/buildspec/load.go`, so the read failed with:
  - `ENOENT: no such file or directory, access '/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/buildspec/defaults.go'`
- The repository already had an untracked GOJA-065 directory and a vocabulary change from earlier work. I did not modify or stage the untracked GOJA-065 ticket.

### What I learned

- The build-time and runtime-side profile DTOs are parallel: both currently store a `Runtimes` map whose values are just module lists.
- Built-in commands add user-facing `--runtime` flags even though the simplification goal is to remove user-facing runtime profile selection.
- Provider command contexts currently receive `RuntimeProfile`; removing that immediately would be a broader provider API change, so the design recommends keeping a constant internal label during the first simplification pass.
- `pkg/engine` should remain out of scope. The lower-level engine runtime is a real execution object and should not be confused with xgoja's named runtime profile abstraction.

### What was tricky to build

The tricky part was separating similar words that refer to different layers. `RuntimeSpec` in buildspec is currently a named profile, `app.RuntimeSpec` is the embedded app DTO, and `engine.Runtime` is the concrete Goja runtime. The guide explicitly explains these names so a new intern does not accidentally remove the wrong runtime concept.

Another tricky part is deciding how hard to break compatibility. The design recommends a clean break in YAML and generated command UX, but a narrower internal compatibility label for provider APIs. That splits the change into a user-facing simplification now and a later provider API cleanup if needed.

### What warrants a second pair of eyes

- Confirm whether `commandProviders[].modules` should remain as a filter over the one global module set.
- Confirm whether to reject old `runtimes`/`commands.*.runtime` fields with explicit validation errors or silently migrate a single runtime named `main`.
- Review whether keeping `RuntimeProfile: "main"` in provider contexts is acceptable as a transitional internal detail.

### What should be done in the future

- Implement the seven phases listed in `tasks.md` and the design document.
- Update all xgoja examples and docs in the same implementation branch.
- After the simplification lands, consider a second cleanup ticket to rename/remove providerapi `RuntimeProfile` fields.

### Code review instructions

- Start with the design guide:
  - `ttmp/2026/06/04/GOJA-066--simplify-xgoja-to-a-single-runtime-profile/design-doc/01-single-runtime-profile-simplification-design-and-implementation-guide.md`
- Then inspect the current profile-heavy files:
  - `cmd/xgoja/internal/buildspec/build_spec.go`
  - `cmd/xgoja/internal/buildspec/validate.go`
  - `cmd/xgoja/internal/generate/main.go`
  - `pkg/xgoja/app/runtime_spec.go`
  - `pkg/xgoja/app/factory.go`
  - `pkg/xgoja/app/root.go`
  - `pkg/xgoja/app/run.go`
  - `pkg/xgoja/app/tui.go`
  - `pkg/xgoja/app/command_providers.go`
  - `pkg/xgoja/providerapi/commands.go`
- Validate future implementation with:
  - `go test ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate ./pkg/xgoja/app -count=1`
  - `go test ./cmd/xgoja/... ./pkg/xgoja/... -count=1`
  - representative example smokes.

### Technical details

Current user-facing profile shape:

```yaml
runtimes:
  main:
    modules:
      - package: geppetto
        name: geppetto
        as: geppetto
commands:
  jsverbs:
    enabled: true
    runtime: main
```

Proposed shape:

```yaml
modules:
  - package: geppetto
    name: geppetto
    as: geppetto
commands:
  jsverbs:
    enabled: true
```

Current factory entry point:

```go
runtime, err := factory.NewRuntimeFromSections(ctx, profile, vals, requireOptions...)
```

Proposed factory entry point:

```go
runtime, err := factory.NewRuntimeFromSections(ctx, vals, requireOptions...)
```

## Step 2: Validate and Upload the GOJA-066 Design Bundle

This step validated the ticket workspace and uploaded the design package to reMarkable. The upload bundle includes the implementation guide and this diary, so reviewers can read both the proposed design and the investigation trail as one PDF.

The ticket passed `docmgr doctor`, and the upload succeeded to the requested date/ticket folder under `/ai/2026/06/04/GOJA-066`.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue the GOJA-066 documentation workflow by validating the ticket, uploading the design bundle to reMarkable, and recording the result.

**Inferred user intent:** Finish the requested documentation deliverable and make it available on reMarkable for review.

**Commit (code):** N/A — documentation/upload step only.

### What I did

- Ran:
  - `cd go-go-goja && docmgr doctor --ticket GOJA-066 --stale-after 30`
- Uploaded a reMarkable bundle containing:
  - `design-doc/01-single-runtime-profile-simplification-design-and-implementation-guide.md`
  - `reference/01-diary.md`
- Used remote destination:
  - `/ai/2026/06/04/GOJA-066`
- Used document name:
  - `GOJA-066 Single Runtime Profile Design.pdf`
- Updated the task list and changelog after upload.

### Why

The user explicitly asked for the ticket guide to be uploaded to reMarkable. Validating first catches frontmatter/relationship issues before producing the PDF bundle.

### What worked

- `docmgr doctor` passed:
  - `GOJA-066 — All checks passed`
- `remarquee upload bundle` succeeded with:
  - `OK: uploaded GOJA-066 Single Runtime Profile Design.pdf -> /ai/2026/06/04/GOJA-066`

### What didn't work

- N/A

### What I learned

- The GOJA-066 document relationships and vocabulary are valid enough for `docmgr doctor`.
- The reMarkable upload path accepted the bundled design and diary without needing a separate auth or verification command.

### What was tricky to build

The only tricky part was avoiding unrelated working-tree changes. The repository already had modified GOJA-064 files, a vocabulary change, and an untracked GOJA-065 ticket. I left those untouched and only prepared GOJA-066 files for this deliverable.

### What warrants a second pair of eyes

- Review the uploaded design for whether the schema break should be strict or include a one-release migration path.
- Confirm the proposed top-level `modules` name before implementation begins.

### What should be done in the future

- Commit the GOJA-066 ticket docs separately from unrelated GOJA-064/GOJA-065 working-tree changes.
- Begin implementation with Phase 1 from the design document.

### Code review instructions

- Review the uploaded bundle contents in the source files:
  - `ttmp/2026/06/04/GOJA-066--simplify-xgoja-to-a-single-runtime-profile/design-doc/01-single-runtime-profile-simplification-design-and-implementation-guide.md`
  - `ttmp/2026/06/04/GOJA-066--simplify-xgoja-to-a-single-runtime-profile/reference/01-diary.md`
- Validate with:
  - `cd go-go-goja && docmgr doctor --ticket GOJA-066 --stale-after 30`

### Technical details

Upload command:

```bash
cd go-go-goja
remarquee upload bundle \
  ttmp/2026/06/04/GOJA-066--simplify-xgoja-to-a-single-runtime-profile/design-doc/01-single-runtime-profile-simplification-design-and-implementation-guide.md \
  ttmp/2026/06/04/GOJA-066--simplify-xgoja-to-a-single-runtime-profile/reference/01-diary.md \
  --name "GOJA-066 Single Runtime Profile Design" \
  --remote-dir "/ai/2026/06/04/GOJA-066" \
  --toc-depth 2 \
  --non-interactive
```
