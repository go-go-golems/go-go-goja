---
Title: Diary
Ticket: XGOJA-001
Status: active
Topics:
    - goja
    - xgoja
    - architecture
    - cli
    - modules
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/go-go-parc/Projects/2026/05/08/ARTICLE - Go Plugin Strategies - xgoja Compile-Time Module Composition.md
      Note: Read during Step 1 and used as primary architecture input
    - Path: go-go-goja/cmd/xgoja
      Note: Implementation target selected by user prompt
    - Path: go-go-goja/pkg/hashiplugin/host/registrar.go
      Note: Out-of-process plugin path inspected as an alternative boundary
    - Path: go-go-goja/ttmp/2026/05/22/XGOJA-001--create-xgoja-compile-time-goja-module-composition-builder/design-doc/01-xgoja-analysis-design-and-implementation-guide.md
      Note: Primary deliverable produced in Step 1
ExternalSources:
    - /home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/05/08/ARTICLE - Go Plugin Strategies - xgoja Compile-Time Module Composition.md
Summary: Chronological diary for creating the xgoja ticket, reading the source article, investigating existing go-go-goja runtime/jsverbs/plugin architecture, and producing the implementation guide.
LastUpdated: 2026-05-22T19:06:53-04:00
WhatFor: Use this diary to resume the xgoja design/implementation work without redoing the initial investigation.
WhenToUse: Read before implementing xgoja, changing the design guide, or continuing ticket XGOJA-001.
---



# Diary

## Goal

This diary records the work to create ticket `XGOJA-001`, read the motivating xgoja compile-time module composition article, inspect the current go-go-goja architecture, and produce an intern-facing analysis/design/implementation guide that can drive the first implementation.

## Step 1: Create the ticket and write the implementation guide

This step turned the request into a structured docmgr ticket with a primary design document and a chronological diary. I started from the article's central architecture claim—compile-time composition is the reliable in-process boundary for Go-backed goja modules—and then checked that claim against the existing `go-go-goja` runtime, modules, jsverbs, REPL, and out-of-process plugin code.

The main output is the design guide. It explains what `xgoja` is, why it should generate a new Go program instead of loading native Go plugins into an old binary, how runtime profiles should map to require-able modules, and how an intern should implement the CLI, build spec parser, provider API, generator, pure target, diagnostics, and later adapter/Cobra modes.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket to create xgoja, a tool to configure a precompiled binary with multiple goja modules registered.

Read /home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/05/08/ARTICLE - Go Plugin Strategies - xgoja Compile-Time Module Composition.md

Keep a diary as you work.

We want to build xgoja.

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a docmgr ticket for building `xgoja`, use the supplied article as the primary source, inspect the current codebase enough to ground the guide in real files, maintain a diary, and upload the resulting documentation bundle to reMarkable.

**Inferred user intent:** Provide a clear, technical onboarding and implementation package so a new intern can start building `xgoja` without re-deriving the architecture, plugin tradeoffs, runtime model, YAML shape, and phased implementation plan.

### What I did

- Created docmgr ticket `XGOJA-001` with title `Create xgoja compile-time goja module composition builder`.
- Added the primary design document:
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/22/XGOJA-001--create-xgoja-compile-time-goja-module-composition-builder/design-doc/01-xgoja-analysis-design-and-implementation-guide.md`
- Added this diary document:
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/22/XGOJA-001--create-xgoja-compile-time-goja-module-composition-builder/reference/01-diary.md`
- Read the motivating article:
  - `/home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/05/08/ARTICLE - Go Plugin Strategies - xgoja Compile-Time Module Composition.md`
- Inspected current go-go-goja implementation files that materially shaped the guide:
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/modules/common.go`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/factory.go`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/module_specs.go`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/runtime.go`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbs/scan.go`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbs/runtime.go`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbscli/command.go`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbscli/runtime.go`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbrepos/bootstrap.go`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/hashiplugin/host/registrar.go`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/hashiplugin/sdk/module.go`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/goja-repl/root.go`
- Wrote the guide with:
  - Executive summary.
  - Problem statement and scope.
  - Current-state architecture with file/line references.
  - Build-time versus runtime mental model.
  - Proposed architecture and repository layout.
  - Provider API, runtime plan, YAML schema, generated `main.go`, generated `go.mod`.
  - Command design for `build`, `doctor`, `inspect`, and `list-modules`.
  - Target modes for pure xgoja, STDBIN adapter, and Cobra attach.
  - JS verbs integration plan.
  - Failure modes, testing strategy, security/capability notes, alternatives, and open questions.

### Why

- The article provides the high-level design direction, but implementation work needs a repository-grounded guide that shows where existing pieces live and what needs to change.
- `go-go-goja` already has a runtime factory, native module registry, jsverbs scanner/invoker, and out-of-process plugin system. The guide needed to explain those pieces before proposing new abstractions.
- A new intern needs clear sequencing. The implementation plan starts with parsing and validation, then provider API and generation, then build execution, then diagnostics and advanced target modes.

### What worked

- `docmgr ticket create-ticket` created the ticket workspace successfully.
- `docmgr doc add` created both the design doc and diary doc successfully.
- The supplied article was available and directly usable as the primary source.
- The existing codebase already contains strong integration points: `engine.FactoryBuilder`, `engine.ModuleSpec`, `engine.RuntimeModuleRegistrar`, `jsverbs.InvokeInRuntime`, and `jsverbscli` command mounting.
- Line-referenced evidence could be gathered from the current files, so the guide is not merely speculative.

### What didn't work

- Attempted to read `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbscli/commands.go`, but the file does not exist. The correct file is `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbscli/command.go`.
- Attempted to read `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/hashiplugin/host/discovery.go`, but the correct file is `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/hashiplugin/host/discover.go`.
- Attempted to read `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/hashiplugin/shared/module.go`, but the correct file for the shared plugin wrapper is `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/hashiplugin/shared/plugin.go`.

### What I learned

- `go-go-goja` already supports explicit runtime composition through `engine.FactoryBuilder`; `xgoja` does not need to invent runtime ownership from scratch.
- The current native module system is still centered on a global default registry populated by imports and `init()`, while `xgoja` needs provider packages that explicitly advertise modules and verb sources.
- `jsverbscli` is close to reusable for `xgoja`, but its current runtime factory is hard-coded. A caller-provided runtime factory/profile seam is the key integration point.
- The HashiCorp plugin path is useful context because it already handles dynamic out-of-process modules, but it is intentionally a different extension boundary than compile-time composition.

### What was tricky to build

- The main conceptual trap is that JavaScript `require(...)` makes source-loaded JS modules and Go-backed native modules look similar at runtime, even though they have different distribution rules. I handled this by structuring the guide around build time versus run time and by making runtime profiles select module instances from provider packages.
- Another tricky point is preserving the article's recommendation without ignoring the existing code. The current code has global module registration and an out-of-process plugin system; the guide explains how those fit instead of pretending they do not exist.
- The file evidence needed careful path checking because some guessed filenames were wrong. I corrected those by listing the relevant directories and then reading the actual files.

### What warrants a second pair of eyes

- The proposed provider API shape should be reviewed before implementation. In particular, reviewers should decide whether module factories should return `require.ModuleLoader` directly or a higher-level object that `xgoja` converts into a loader.
- The plan assumes `xgoja` should likely be a separate module/repository, but the open questions preserve the alternative of starting inside `go-go-goja`.
- The jsverbs integration proposal should be checked against existing Glazed command mounting expectations so we do not duplicate command construction logic.

### What should be done in the future

- Decide the repository location for `xgoja`.
- Implement Phase 1 and Phase 2 from the guide: CLI skeleton, buildspec parsing, and validation.
- Add a tiny fixture provider before integrating larger providers such as `web-stuff`.
- Add a caller-owned runtime factory seam to `jsverbscli` when implementation reaches JS verb mounting.

### Code review instructions

- Start with the design doc at `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/22/XGOJA-001--create-xgoja-compile-time-goja-module-composition-builder/design-doc/01-xgoja-analysis-design-and-implementation-guide.md`.
- Check the file references in the "Reader orientation" section against the source files listed above.
- Validate that the proposed phased plan is consistent with the current `go-go-goja` runtime APIs.
- For documentation validation, run:
  - `docmgr doctor --ticket XGOJA-001 --stale-after 30`

### Technical details

Primary commands used:

```bash
docmgr ticket create-ticket --ticket XGOJA-001 --title "Create xgoja compile-time goja module composition builder" --topics goja,xgoja,architecture,cli,modules,documentation

docmgr doc add --ticket XGOJA-001 --doc-type design-doc --title "xgoja Analysis Design and Implementation Guide"
docmgr doc add --ticket XGOJA-001 --doc-type reference --title "Diary"

docmgr doc list --ticket XGOJA-001
docmgr task list --ticket XGOJA-001
```

Key design conclusion:

```text
Go-backed goja modules are source-level dependencies of the generated binary.
JS verbs may remain runtime-loaded source files.
The YAML is the user interface; generated Go code is deterministic, readable, and disposable.
```

## Step 2: Validate and deliver the ticket bundle

This step made the ticket ready for handoff. After writing the guide and diary, I related the important source files to the focused documents, fixed the only docmgr vocabulary issue, ran the ticket doctor successfully, and uploaded the bundle to reMarkable.

The reMarkable upload is intentionally a bundle rather than separate PDFs. That gives the reader one document with the guide, diary, task list, and changelog in a single destination folder for this ticket.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Complete the requested documentation workflow by validating docmgr state and publishing the ticket documents to reMarkable.

**Inferred user intent:** Ensure the guide is not only written locally, but also organized, validated, and available on the reading device.

### What I did

- Related the design document to the source article and key implementation files with `docmgr doc relate`.
- Related the diary to the source article, primary guide, and plugin file that shaped the alternatives analysis.
- Updated the docmgr changelog with the guide and diary files.
- Ran `docmgr doctor --ticket XGOJA-001 --stale-after 30`.
- Added the missing `xgoja` topic to the docmgr vocabulary after doctor reported it as unknown.
- Re-ran `docmgr doctor --ticket XGOJA-001 --stale-after 30`, which passed.
- Uploaded the ticket bundle to reMarkable at `/ai/2026/05/22/XGOJA-001`.

### Why

- File relationships make the design document searchable by the code and article paths that shaped it.
- Running doctor catches stale metadata and vocabulary drift before handoff.
- Uploading a bundle makes the guide easier to read on reMarkable and keeps the implementation context with it.

### What worked

- `docmgr doc relate` updated related files for both the design doc and diary.
- `docmgr vocab add --category topics --slug xgoja --description "Compile-time goja module composition builder and generated binary tooling"` resolved the only doctor warning.
- Final doctor output was: `✅ All checks passed`.
- `remarquee upload bundle ... --name "XGOJA 001 Design Guide" --remote-dir "/ai/2026/05/22/XGOJA-001" --toc-depth 2 --non-interactive` succeeded with: `OK: uploaded XGOJA 001 Design Guide.pdf -> /ai/2026/05/22/XGOJA-001`.

### What didn't work

- The first doctor run reported an unknown vocabulary value:

```text
1) [warning] Unknown vocabulary value for Topics
File: /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/22/XGOJA-001--create-xgoja-compile-time-goja-module-composition-builder/index.md
Field: Topics
Value: "xgoja"
```

- This was fixed by adding `xgoja` to the `topics` vocabulary and rerunning doctor.

### What I learned

- New ticket topics need vocabulary entries before doctor passes cleanly.
- The reMarkable upload command reports success directly, so no separate cloud listing is needed for routine verification.

### What was tricky to build

- The only process wrinkle was ordering: the first upload happened before the upload evidence was recorded in the diary. To keep the delivered bundle complete, I updated this diary step and re-uploaded the bundle with `--force` so the reMarkable copy includes the validation and delivery record.

### What warrants a second pair of eyes

- Confirm that overwriting the just-uploaded reMarkable document is acceptable. It was done only to ensure the final delivered PDF includes the final diary step.

### What should be done in the future

- Add the delivery/validation diary step before the first upload to avoid a second upload.

### Code review instructions

- Check the ticket doctor result with:
  - `docmgr doctor --ticket XGOJA-001 --stale-after 30`
- Confirm the bundle source files are:
  - design doc
  - diary
  - tasks
  - changelog

### Technical details

Validation command:

```bash
docmgr doctor --ticket XGOJA-001 --stale-after 30
```

Vocabulary fix:

```bash
docmgr vocab add --category topics --slug xgoja --description "Compile-time goja module composition builder and generated binary tooling"
```

Upload command:

```bash
remarquee upload bundle \
  /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/22/XGOJA-001--create-xgoja-compile-time-goja-module-composition-builder/design-doc/01-xgoja-analysis-design-and-implementation-guide.md \
  /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/22/XGOJA-001--create-xgoja-compile-time-goja-module-composition-builder/reference/01-diary.md \
  /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/22/XGOJA-001--create-xgoja-compile-time-goja-module-composition-builder/tasks.md \
  /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/22/XGOJA-001--create-xgoja-compile-time-goja-module-composition-builder/changelog.md \
  --name "XGOJA 001 Design Guide" \
  --remote-dir "/ai/2026/05/22/XGOJA-001" \
  --toc-depth 2 \
  --non-interactive
```

## Step 3: Scope xgoja into go-go-goja/cmd/xgoja

This step records the implementation direction change before code work starts. The tool should not begin as a separate repository; it should be built inside the existing `go-go-goja` module under `cmd/xgoja`, and it should use Glazed for command definitions and Cobra integration.

That decision narrows Phase 1. Instead of creating a new module, the first implementation should reuse the existing top-level `go.mod`, existing Glazed dependency, and the root command conventions already used by `cmd/goja-repl` and `cmd/jsverbs-example`.

### Prompt Context

**User prompt (verbatim):** "Build as part of go-go-goja/ in go-go-goja/cmd/xgoja . Use the glazed framework to build the tool. Create tasks if you need to, then work on them one by one, committing at appropriate intervals, keeping a detailed diary"

**Assistant interpretation:** Implement xgoja directly in the go-go-goja repository at `cmd/xgoja`, using Glazed command abstractions. Keep docmgr tasks and diary current, and make focused commits as phases complete.

**Inferred user intent:** Move from design to implementation while keeping the work structured, reviewable, and documented.

### What I did

- Marked task 8 complete because the user chose the repository and command location.
- Updated the changelog with the scope decision.
- Planned the first code task as Phase 1: CLI skeleton and command wiring under `cmd/xgoja`.

### Why

- The implementation location affects every code path: imports should be intra-module, no new `go.mod` should be created, and tests should run from the existing `go-go-goja` module.
- Recording the decision in the diary prevents future confusion with the design guide's open question about whether xgoja should live in a separate repository.

### What worked

- `docmgr task check --ticket XGOJA-001 --id 8` marked the repository-location decision complete.
- `docmgr changelog update --ticket XGOJA-001 ...` recorded the decision.

### What didn't work

- N/A

### What I learned

- For this ticket, `xgoja` is explicitly a command inside `go-go-goja`, not a separate module.

### What was tricky to build

- The design guide still describes both repository options. The implementation must follow the user's latest instruction and treat the separate-repository option as rejected for now.

### What warrants a second pair of eyes

- Confirm whether future package APIs should live under `cmd/xgoja/internal/...` first or under reusable `pkg/xgoja/...` packages immediately. I will start with command-local/internal code where possible and only promote stable APIs later.

### What should be done in the future

- Update the design guide if the implementation settles enough details that the open repository-location question should be removed.

### Code review instructions

- Check `cmd/xgoja` first once code is added.
- Confirm no nested `go.mod` is introduced.

### Technical details

Planned first implementation command:

```bash
cd /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja
go run ./cmd/xgoja --help
```
