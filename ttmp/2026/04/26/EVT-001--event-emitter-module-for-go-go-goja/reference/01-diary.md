---
Title: Diary
Ticket: EVT-001
Status: active
Topics:
    - goja
    - javascript
    - event-emitter
    - module
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go.work
      Note: Workspace file showing goja_nodejs was added locally.
    - Path: ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/scripts/01-gather-event-emitter-evidence.sh
      Note: Ticket-local evidence script created during investigation.
    - Path: ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/sources/local/01-event-emitter.md
      Note: Imported user-provided source brief.
    - Path: ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/sources/local/evidence.txt
      Note: Captured line-referenced investigation evidence.
ExternalSources:
    - local:01-event-emitter.md
    - local:evidence.txt
Summary: Chronological diary for the event-emitter module implementation guide work.
LastUpdated: 2026-04-26T09:29:00-04:00
WhatFor: Record the investigation and documentation work for EVT-001.
WhenToUse: Read before resuming implementation or reviewing the event-emitter design.
---


# Diary

## Goal

This diary records the setup, investigation, design, and delivery work for EVT-001: adding an event-emitter module and Go-owned event bus design to `go-go-goja`.

## Step 1: Created ticket workspace and imported the source brief

I created a new docmgr ticket for the event-emitter work, imported the provided `/tmp/event-emitter.md` brief into the ticket sources, and created a design document plus this diary. This established a durable workspace under `ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja` before doing analysis.

The imported brief became the starting point for the design. I then read it from the ticket copy rather than relying on the original `/tmp` file, so future readers can reproduce the context from the ticket workspace alone.

### Prompt Context

**User prompt (verbatim):** "Creat a new docmgr ticket to implement an event-emitter module in go-go-goja/ . Keep a diary as you work.

Import the file /tmp/event-emitter.md with docmgr import file --file XXX --ticket YYY (replace XXX and YYY), and then read it, and reate a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet  
point sand pseudocode and diagrams and api references and file                                                                                                                                              
  references.                                                                                                                                                                                               
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable.                                                                                                                  
                                                                                                                                                                                                            
You can write scripts and all kinds of other things in the scripts/ folder of the ticket."

**Assistant interpretation:** Create a structured docmgr ticket, import and read the event-emitter brief, investigate the repository, write an intern-oriented implementation guide, maintain a diary, and upload the deliverables to reMarkable.

**Inferred user intent:** The user wants a durable, reviewable ticket package that an intern can use to implement the feature safely without violating goja concurrency rules.

**Commit (code):** N/A — documentation and ticket workspace only.

### What I did

- Ran `docmgr status --summary-only` from `go-go-goja` to confirm the docs root and existing ticket state.
- Created ticket `EVT-001`:
  - `docmgr ticket create-ticket --ticket EVT-001 --title "Event emitter module for go-go-goja" --topics goja,javascript,event-emitter,module`
- Imported the provided brief:
  - `docmgr import file --file /tmp/event-emitter.md --ticket EVT-001`
- Created documents:
  - `docmgr doc add --ticket EVT-001 --doc-type design-doc --title "Event emitter module implementation guide" --summary "Detailed analysis, design, and implementation guide for adding an event-emitter module to go-go-goja."`
  - `docmgr doc add --ticket EVT-001 --doc-type reference --title "Diary" --summary "Chronological diary for the event-emitter module implementation guide work."`
- Read the imported copy at `ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/sources/local/01-event-emitter.md`.

### Why

- Creating the ticket first ensures all analysis, scripts, sources, and final documents live in one searchable workspace.
- Importing the file with docmgr preserves the user's source material as a ticket source and updates ticket metadata.

### What worked

- Ticket creation succeeded and generated `index.md`, `tasks.md`, `changelog.md`, and the standard subdirectories.
- The file import succeeded and created `sources/local/01-event-emitter.md` plus `.meta/sources.yaml` metadata.
- The imported brief was readable and contained concrete starter code and references.

### What didn't work

- N/A. Ticket setup and import completed without command errors.

### What I learned

- The imported brief is a good high-level direction but uses a standalone `Engine` wrapper. The actual repository already has an `engine.Runtime`, `runtimeowner.Runner`, and `runtimebridge` that should be used instead.

### What was tricky to build

- The first design choice was not to copy the imported standalone `Engine` shape blindly. The repository's runtime lifecycle is already centralized, so the guide needed to adapt the idea into `engine.NewBuilder()`, runtime initializers, and runtime values.

### What warrants a second pair of eyes

- The eventual implementation should be reviewed for import cycles. In particular, a module under `modules/events` must not import `engine` if `engine/runtime.go` blank-imports that module.

### What should be done in the future

- Implement the feature in focused phases after reviewing the design document.

### Code review instructions

- Start with the ticket files under `ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/`.
- Confirm `sources/local/01-event-emitter.md` is the imported source brief.
- Confirm the design doc adapts the brief to existing runtime ownership APIs.

### Technical details

- Ticket path: `go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja`
- Imported source: `go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/sources/local/01-event-emitter.md`
- Design doc: `go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/design-doc/01-event-emitter-module-implementation-guide.md`

## Step 2: Gathered repository and dependency evidence

I inspected the runtime, module registry, async scheduling, and dependency contracts that shape the event-emitter implementation. I also added a ticket-local evidence script so the investigation can be rerun and so line references in the design are reproducible.

This step confirmed that the correct implementation seam is the existing runtime owner model, not a new independent event loop. It also confirmed the Watermill and fsnotify lifecycle constraints that the adapters must preserve.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Build the design from concrete repository evidence rather than only from the imported sketch.

**Inferred user intent:** The intern guide should explain how the feature fits the current codebase and cite the important files.

**Commit (code):** N/A — documentation workspace only.

### What I did

- Listed repository structure and key Go files with `rg --files` and `find`.
- Read these core files:
  - `engine/runtime.go`
  - `engine/factory.go`
  - `engine/module_specs.go`
  - `engine/runtime_modules.go`
  - `modules/common.go`
  - `modules/timer/timer.go`
  - `modules/fs/fs_async.go`
  - `pkg/runtimeowner/types.go`
  - `pkg/runtimeowner/runner.go`
  - `pkg/runtimebridge/runtimebridge.go`
  - `README.md`
  - `pkg/doc/03-async-patterns.md`
  - `pkg/doc/16-nodejs-primitives.md`
- Checked dependency source snippets for:
  - `goja_nodejs/eventloop.EventLoop.RunOnLoop()`
  - Watermill `message.Subscriber`, `Message.Ack()`, and `Message.Nack()`
  - `fsnotify.Watcher`, events/errors channels, and non-recursive watch behavior
- Created and ran:
  - `ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/scripts/01-gather-event-emitter-evidence.sh`
- Captured output in:
  - `ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/sources/local/evidence.txt`

### Why

- The design needed file-backed evidence for runtime ownership and module registration.
- The requested guide is for a new intern, so it needs reproducible references instead of broad claims.

### What worked

- The repository already has a clear runtime owner abstraction.
- Existing modules (`timer`, `fs`) demonstrate the correct pattern for goroutines posting back to the JS owner.
- Watermill and fsnotify dependency contracts were available locally in the module cache.

### What didn't work

- An initial read attempt for `pkg/runtimebridge/bridge.go` failed because the actual file is `pkg/runtimebridge/runtimebridge.go`:
  - `ENOENT: no such file or directory, access '/home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/runtimebridge/bridge.go'`
- I corrected the path and read `pkg/runtimebridge/runtimebridge.go`.

### What I learned

- `engine.Factory.NewRuntime` starts the event loop and creates a `runtimeowner.Runner` immediately after `goja.New()`.
- Runtime initializers run after `require` is enabled, which makes them the right place to install a global bus that uses `require("events")`.
- The pure `events` module should live under `modules/events`, while the Go-owned bus should live in a package such as `pkg/jsevents` to avoid `engine` import cycles.

### What was tricky to build

- The evidence script originally pointed at the module-cache copy of `goja_nodejs`. After the user added a local `goja_nodejs` checkout to the workspace, the evidence source needed to prefer `../goja_nodejs/eventloop/eventloop.go` when present and fall back to the module cache otherwise.

### What warrants a second pair of eyes

- The proposed package split should be reviewed for Go import-cycle safety before implementation.
- The decision to include both `events` and `node:events` in the data-only default module list should be reviewed against the repository's sandbox policy.

### What should be done in the future

- When implementation starts, run the evidence script again if runtime/module architecture changes.

### Code review instructions

- Review `scripts/01-gather-event-emitter-evidence.sh` to see exactly which file ranges shaped the guide.
- Review `sources/local/evidence.txt` when checking line references in the design document.

### Technical details

Key commands:

```bash
cd go-go-goja
rg -n "RegisterNativeModule|Registry|eventloop|RunOnLoop|Require|modules\.Register|Enable|NewRuntime|Init|require\(" engine modules pkg cmd -S
chmod +x ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/scripts/01-gather-event-emitter-evidence.sh
ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/scripts/01-gather-event-emitter-evidence.sh > ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/sources/local/evidence.txt
```

## Step 3: Incorporated the local goja_nodejs workspace checkout

The user added `goja_nodejs` to the workspace while the analysis was in progress. I verified that `go.work` now includes `./goja_nodejs` and updated the evidence script so it cites the local event-loop source when available.

This matters because the event-emitter design relies on the exact `RunOnLoop()` contract. Using the workspace checkout makes the references easier to inspect during implementation.

### Prompt Context

**User prompt (verbatim):** "I added goja_nodejs to the workspace"

**Assistant interpretation:** Prefer the local `goja_nodejs` checkout for event-loop references and analysis.

**Inferred user intent:** The user wants the design and later implementation to use the local workspace source instead of only the module cache.

**Commit (code):** N/A — documentation workspace only.

### What I did

- Ran `ls -la` and confirmed `/home/manuel/workspaces/2026-04-26/add-event-emitter-module/goja_nodejs` exists.
- Read `go.work` and confirmed it now contains:
  - `./glazed`
  - `./go-go-goja`
  - `./goja`
  - `./goja_nodejs`
- Read the local event-loop snippet:
  - `nl -ba goja_nodejs/eventloop/eventloop.go | sed -n '314,321p'`
- Edited `scripts/01-gather-event-emitter-evidence.sh` so it uses `../goja_nodejs/eventloop/eventloop.go` when present.
- Reran the script and regenerated `sources/local/evidence.txt`.

### Why

- The design cites `RunOnLoop()` as a core safety contract.
- A workspace checkout is more convenient for an implementer than a versioned module-cache path.

### What worked

- The local checkout contains the same key `RunOnLoop()` documentation lines.
- The evidence script now supports both workspace and fallback module-cache usage.

### What didn't work

- N/A. The workspace update was straightforward.

### What I learned

- The workspace now has four modules in `go.work`, and `goja_nodejs` can be inspected directly during implementation.

### What was tricky to build

- The script runs from the `go-go-goja` repository root, so the local `goja_nodejs` path is `../goja_nodejs/eventloop/eventloop.go`, not `./goja_nodejs/eventloop/eventloop.go`.

### What warrants a second pair of eyes

- If implementation changes `goja_nodejs` itself, those changes should be reviewed separately from `go-go-goja` feature code.

### What should be done in the future

- Use the local `goja_nodejs` checkout for debugging scheduler behavior if any event dispatch test becomes flaky.

### Code review instructions

- Check `/home/manuel/workspaces/2026-04-26/add-event-emitter-module/go.work` if the local checkout is not being used.
- Check `../goja_nodejs/eventloop/eventloop.go:314-321` for the scheduler contract.

### Technical details

Commands:

```bash
pwd && ls -la && find . -maxdepth 2 -type d \( -name 'goja_nodejs' -o -name 'goja-nodejs' -o -name 'goja_nodejs*' \) -print && cat go.work
nl -ba goja_nodejs/eventloop/eventloop.go | sed -n '314,321p'
cd go-go-goja && ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/scripts/01-gather-event-emitter-evidence.sh > ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/sources/local/evidence.txt
```

## Step 4: Wrote the intern-oriented implementation guide

I rewrote the generated design-doc stub into a detailed implementation guide. The guide explains the system from first principles, maps the relevant repository architecture, proposes a concrete package layout, gives API sketches, includes pseudocode for EventEmitter, Bus, Watermill, and fsnotify pieces, and lists phased implementation and validation steps.

The document emphasizes the safety invariant that all JavaScript calls must go through the runtime owner/event loop. It also adapts the imported brief into the repository's existing `engine.Runtime` model rather than introducing a parallel engine wrapper.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Produce a very clear, detailed design and implementation guide suitable for a new intern.

**Inferred user intent:** The intern should be able to implement the feature from the document without needing extensive prior knowledge of goja, goja_nodejs, Watermill, fsnotify, or the go-go-goja module system.

**Commit (code):** N/A — documentation workspace only.

### What I did

- Wrote `design-doc/01-event-emitter-module-implementation-guide.md`.
- Included sections for:
  - executive summary,
  - first-principles background,
  - evidence-backed current architecture,
  - problem statement and scope,
  - proposed package layout,
  - diagrams,
  - JavaScript API reference,
  - Go API reference,
  - Watermill adapter design,
  - fsnotify adapter design,
  - phased implementation plan,
  - validation strategy,
  - risks,
  - alternatives,
  - file references,
  - implementation checklist.
- Added task entries for setup, evidence gathering, guide writing, diary/bookkeeping, validation, and upload.

### Why

- The guide should reduce implementation risk by explaining both the JavaScript façade and the Go scheduling boundary.
- The phased plan makes it easier to review smaller PRs.

### What worked

- The current repository architecture maps cleanly to the requested feature.
- The imported brief's core ideas were preserved while adapting the implementation shape to `engine.Runtime` and `runtimeowner.Runner`.

### What didn't work

- N/A for writing. Validation and upload are recorded in later steps.

### What I learned

- The best first implementation is two-layered: pure `modules/events` for compatibility and `pkg/jsevents` for Go-owned runtime event dispatch.

### What was tricky to build

- The guide had to be detailed without implying that the code had already been implemented. I framed concrete code as pseudocode/API sketches and kept the actual deliverable as a design and implementation plan.

### What warrants a second pair of eyes

- Review the recommendation to add `events` and `node:events` to the default data-only module list.
- Review the Watermill no-auto-ack decision against product expectations.

### What should be done in the future

- Start implementation with `modules/events` tests before adding the Go-owned bus.

### Code review instructions

- Begin at `design-doc/01-event-emitter-module-implementation-guide.md`.
- Check the `File reference map` and `Implementation phases` sections first.
- Validate that every proposed goroutine-to-JS path schedules through `runtimeowner.Runner`.

### Technical details

Design doc path:

```text
go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/design-doc/01-event-emitter-module-implementation-guide.md
```

## Step 5: Validated ticket hygiene with docmgr doctor

I ran `docmgr doctor` after writing the guide and diary. The first validation found two hygiene issues: the new topic values were not yet in the vocabulary, and the imported Markdown source did not have docmgr frontmatter or a numeric prefix.

I fixed the vocabulary entries, renamed the imported source to `01-event-emitter.md`, added frontmatter to that source, updated references, and reran doctor. The second doctor run passed cleanly.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Validate the ticket before reMarkable upload and fix docmgr hygiene issues rather than leaving warnings behind.

**Inferred user intent:** The final ticket should be clean, searchable, and suitable for future continuation.

**Commit (code):** N/A — documentation workspace only.

### What I did

- Ran `docmgr doctor --ticket EVT-001 --stale-after 30`.
- Added topic vocabulary entries:
  - `docmgr vocab add --category topics --slug event-emitter --description "Event emitter APIs, listener dispatch, and Go-to-JavaScript event bus design."`
  - `docmgr vocab add --category topics --slug javascript --description "JavaScript runtime, modules, and API behavior."`
  - `docmgr vocab add --category topics --slug module --description "Single module implementation and module-level API work."`
- Renamed the imported source from `sources/local/event-emitter.md` to `sources/local/01-event-emitter.md`.
- Added docmgr frontmatter to `sources/local/01-event-emitter.md` while preserving the imported source body.
- Updated ticket references from `local:event-emitter.md` to `local:01-event-emitter.md`.
- Reran `docmgr doctor --ticket EVT-001 --stale-after 30`.
- Marked validation task 5 complete.

### Why

- The ticket-research workflow requires a clean doctor result before upload.
- Numeric prefixes and frontmatter make the imported Markdown source compatible with docmgr validation.

### What worked

- Adding vocabulary entries resolved the unknown topic warning.
- Renaming and adding frontmatter resolved the imported source validation error and numeric-prefix warning.
- The final doctor output was:
  - `## Doctor Report (1 findings)`
  - `- ✅ All checks passed`

### What didn't work

The first doctor run reported:

```text
[WARNING] unknown_topics — unknown topics: [event-emitter javascript module]
[ERROR] invalid_frontmatter — Failed to parse frontmatter: ... sources/local/event-emitter.md frontmatter delimiters '---' not found
[WARNING] missing_numeric_prefix — file without numeric prefix ... sources/local/event-emitter.md
```

### What I learned

- Imported Markdown files under ticket sources are still checked by docmgr doctor, so they need normal docmgr-compatible frontmatter and naming.

### What was tricky to build

- A broad string replacement temporarily changed mentions of `/tmp/event-emitter.md` to `/tmp/01-event-emitter.md` in the diary. I corrected those back to the original source path so the prompt context and command history remain accurate.

### What warrants a second pair of eyes

- Check that renaming the imported source to `01-event-emitter.md` is acceptable for future readers. The source frontmatter still records `/tmp/event-emitter.md` as the original external source.

### What should be done in the future

- Prefer importing Markdown sources with a numeric `--name` if docmgr supports it, or immediately normalize imported source filenames before writing references.

### Code review instructions

- Run `docmgr doctor --ticket EVT-001 --stale-after 30` from `go-go-goja`.
- Check `sources/local/01-event-emitter.md` for source frontmatter and preserved body.
- Check `ttmp/vocabulary.yaml` for the new topic entries.

### Technical details

Validation command:

```bash
cd go-go-goja
docmgr doctor --ticket EVT-001 --stale-after 30
```

## Step 6: Uploaded the document bundle to reMarkable

I uploaded the ticket bundle to reMarkable after a successful dry-run. The uploaded bundle included the ticket index, primary implementation guide, diary, and normalized imported source brief.

After the first upload, I recorded the upload in the task list, changelog, and diary. Because those bookkeeping updates happened after the first upload, I prepared a final bundle name for a second upload so the reMarkable copy can include the completed diary and task state without overwriting the first uploaded document.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Deliver the finished documentation bundle to reMarkable and verify it is present remotely.

**Inferred user intent:** The user wants the design package available for reading on the reMarkable device/cloud, not only in the repository.

**Commit (code):** N/A — documentation workspace only.

### What I did

- Checked `remarquee` availability and account:
  - `remarquee status`
  - `remarquee cloud account --non-interactive`
- Ran a dry-run bundle upload:
  - `remarquee upload bundle --dry-run ... --name "EVT-001 Event Emitter Module Guide" --remote-dir "/ai/2026/04/26/EVT-001" --toc-depth 2`
- Ran the real upload:
  - `remarquee upload bundle ... --name "EVT-001 Event Emitter Module Guide" --remote-dir "/ai/2026/04/26/EVT-001" --toc-depth 2`
- Verified the remote listing:
  - `remarquee cloud ls /ai/2026/04/26/EVT-001 --long --non-interactive`
- Marked task 6 complete and updated the changelog.

### Why

- The ticket workflow requires a dry-run before upload and remote listing verification after upload.
- The bundle format keeps the guide, diary, and source material together with a table of contents.

### What worked

- `remarquee status` returned `remarquee: ok`.
- Account verification returned `user=wesen@ruinwesen.com sync_version=1.5`.
- Dry-run showed the expected four inputs and destination.
- Upload succeeded with:
  - `OK: uploaded EVT-001 Event Emitter Module Guide.pdf -> /ai/2026/04/26/EVT-001`
- Remote listing showed:
  - `[f]	EVT-001 Event Emitter Module Guide`

### What didn't work

- The real upload printed a warning before succeeding:

```text
WARNING: 2026/04/26 09:18:13 apictx.go:259: remote tree has changed, refresh the file tree
OK: uploaded EVT-001 Event Emitter Module Guide.pdf -> /ai/2026/04/26/EVT-001
```

The warning did not block the upload.

### What I learned

- Uploading before final bookkeeping means the remote copy may not include the final upload diary step. Use a final second bundle name when exact final-state delivery matters and overwriting was not explicitly requested.

### What was tricky to build

- The no-overwrite upload workflow is safer for reMarkable annotations, but it means final bookkeeping updates after the first upload should be delivered as a second uniquely named bundle rather than using `--force`.

### What warrants a second pair of eyes

- If the first non-final reMarkable document is unwanted clutter, a human should decide whether to delete it from the device/cloud. I did not delete or overwrite it.

### What should be done in the future

- For future ticket uploads, record all final bookkeeping before the first real upload when possible.

### Code review instructions

- Verify remote contents with:
  - `remarquee cloud ls /ai/2026/04/26/EVT-001 --long --non-interactive`
- Review the final bundle inputs before uploading:
  - `index.md`
  - `design-doc/01-event-emitter-module-implementation-guide.md`
  - `reference/01-diary.md`
  - `sources/local/01-event-emitter.md`

### Technical details

Remote directory:

```text
/ai/2026/04/26/EVT-001
```

## Step 7: Revised design around JS-called connected emitter factories

The user clarified that the Watermill case is an instance of a more general pattern: JavaScript should call a Go function, and Go should return a configured EventEmitter connected to whatever Go-side resource is needed. That means Watermill support should be helper code for explicit factories, not default emitters configured at runtime startup.

I updated the primary design guide to reflect this. The revised design now centers on `pkg/jsevents.Manager` and `EmitterRef` connected-emitter primitives, plus opt-in helper factories such as `watermill.subscribe("orders")` and `fswatch.watch(path)`.

### Prompt Context

**User prompt (verbatim):** "i think the watermill connected emitters is actually (or rather, this is a general pattern, and we are just providing watermill helpers):

- call a go functionf rom js
- the go returns a configured event emitter which is connected to whateveer is needed on the go side

we shouldn't configure default event emitters with watermill

does that change your design?"

**Assistant interpretation:** Revise the design away from global/default Watermill event buses and toward JS-called Go factory functions that return per-resource connected emitters.

**Inferred user intent:** The user wants an API shape where JavaScript owns the resource handle returned by a helper call, while Go owns the safe connection and scheduling machinery.

**Commit (code):** N/A — design/documentation revision only.

### What I did

- Added a design revision section near the top of the design document.
- Replaced the global `goEvents` / `Bus` API with a connected-emitter `Manager` + `EmitterRef` API sketch.
- Reframed Watermill as an opt-in helper that installs a JS factory such as `watermill.subscribe(topic)`.
- Reframed fsnotify as an opt-in helper that installs a JS factory such as `fswatch.watch(path)`.
- Updated implementation phases, design decisions, alternatives, validation strategy, risks, and the intern checklist.

### Why

- A returned emitter is a better resource handle than a global event namespace.
- The returned emitter can own `close()` semantics.
- Watermill should not subscribe to topics by default; subscriptions should start only when JavaScript explicitly calls a helper factory that the Go application chose to expose.

### What worked

- The core concurrency invariant stayed the same: Go goroutines still schedule owner-thread callbacks before touching JavaScript.
- The design became more general and cleaner: Watermill is now just one helper built on reusable connected-emitter primitives.

### What didn't work

- The earlier design's global `goEvents` bus no longer matches the intended API. It remains mentioned only as an alternative that was rejected/superseded.

### What I learned

- The most important abstraction is not a global bus; it is an owner-thread emitter registry plus Go-held emitter references that allow background goroutines to address one returned JS emitter safely.

### What was tricky to build

- Returning a JS emitter while later emitting to that exact emitter from Go needs a safe identity strategy. The revised design recommends an owner-thread registry keyed by emitter ID. Go goroutines hold only an `EmitterRef`/ID, and scheduled owner callbacks look up the actual JS object by ID.

### What warrants a second pair of eyes

- Review the `EmitterRef` registry design carefully. It should avoid direct JS object usage from background goroutines while still cleaning up registry entries on close/runtime shutdown.

### What should be done in the future

- Implement a fake-source connected emitter test before implementing Watermill. That will validate the general pattern independently from Watermill semantics.

### Code review instructions

- Start in the design guide at `Design revision: connected emitters instead of default source buses`.
- Check that Watermill setup begins only inside a JS-called factory such as `watermill.subscribe(...)`.
- Check that old global `goEvents` material appears only as rejected alternative/superseded context.

### Technical details

Updated design doc:

```text
go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/design-doc/01-event-emitter-module-implementation-guide.md
```

## Step 8: Revised design so EventEmitter itself is Go-native

The user clarified that the EventEmitter implementation should be written in Go, with no JavaScript implementation source involved. That changes the `modules/events` design: `require("events")` should export a Go native constructor and Go-backed prototype methods, not a constructor produced by `vm.RunString(...)` from an embedded JavaScript string.

I updated the design guide to make this explicit and to connect it to the previous requirement that JavaScript-created emitters can be passed back into Go. The new design says Go functions should be able to unwrap/validate Go-native emitters passed from JavaScript and then register Go callbacks or safely emit to them through owner-thread scheduling.

### Prompt Context

**User prompt (verbatim):** "the event emitter should be implemented in go, there should be no JS involved"

**Assistant interpretation:** Replace the embedded-JavaScript EventEmitter implementation approach with a native Go implementation exposed to JavaScript through goja.

**Inferred user intent:** The user wants one Go-owned event abstraction that can be used from JavaScript but whose behavior, state, and Go integration hooks are implemented in Go.

**Commit (code):** N/A — design/documentation revision only.

### What I did

- Updated the executive summary to say the `events` module is Go-native.
- Added a `Design revision: EventEmitter itself is Go-native` section.
- Replaced the Phase 1 implementation guidance with native constructor/prototype guidance.
- Added `events.FromValue(...)` / adoption guidance for JS-created emitters passed into Go functions.
- Replaced the old loader pseudocode that used `vm.RunString(eventEmitterSource)` with Go-native constructor pseudocode.

### Why

- A Go-native emitter makes adoption from JS back into Go much cleaner.
- Go helper functions can validate that a passed object is the native EventEmitter type and can access its Go backing state.
- This avoids duplicating EventEmitter behavior in embedded JS and Go helper code.

### What worked

- goja supports native constructors using `func(goja.ConstructorCall) *goja.Object`.
- The existing `goja_nodejs/url` implementation provides an example pattern: create a Go-backed object with `vm.ToValue(...)`, set the prototype, and put Go functions on the prototype.

### What didn't work

- The earlier design still contained JS-source implementation sketches. I marked the behavioral JS sketch as superseded and replaced the actual loader pseudocode with Go-native pseudocode.

### What I learned

- The best final architecture is: Go-native `EventEmitter` + `events.FromValue(...)` + connected-emitter `EmitterRef` scheduling. JavaScript uses the EventEmitter API, but Go owns the implementation and integration identity.

### What was tricky to build

- The design has to distinguish between “no JS implementation” and “JS listeners are still called.” The EventEmitter internals are Go, but listener callbacks may be JavaScript functions and must still be invoked only on the goja owner thread.

### What warrants a second pair of eyes

- Review the native constructor/prototype pattern, especially how `instanceof EventEmitter` and `events.FromValue(...)` should work.
- Review runtime registry cleanup for any package-level maps keyed by `*goja.Runtime`.

### What should be done in the future

- Build a small `modules/events` prototype first and test `new EventEmitter()`, `instanceof`, `on`, `emit`, and passing an emitter into a Go function that adopts it.

### Code review instructions

- Start in the design guide at `Design revision: EventEmitter itself is Go-native`.
- Confirm implementation does not use `vm.RunString(eventEmitterSource)` or embedded JS source for EventEmitter behavior.
- Confirm background goroutines never invoke listener callbacks directly.

### Technical details

Updated design doc:

```text
go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/design-doc/01-event-emitter-module-implementation-guide.md
```
