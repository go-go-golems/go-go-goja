---
Title: Investigation Diary
Ticket: XGOJA-CLUBMED-MODULES
Status: active
Topics:
    - xgoja
    - goja
    - modules
    - architecture
    - clubmedmeetup
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ClubMedMeetup/ttmp/2026/06/08/xgoja-modules-improvement-second-edition--improve-xgoja-and-goja-modules-from-clubmedmeetup-usage-patterns-second-edition/design-doc/01-xgoja-and-goja-module-improvement-map-second-edition.md
      Note: |-
        User-specified source analysis read before creating this ticket.
        Initial prompt-specified source analysis
    - Path: go-go-goja/examples/xgoja/07-embedded-jsverbs/README.md
      Note: Embedded JSVerb source filter example
    - Path: go-go-goja/examples/xgoja/README.md
      Note: XGoja examples overview documents jsverb source filters
    - Path: go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md
      Note: Developer guide updated with scanner filter model and debugging guidance
    - Path: go-go-goja/pkg/doc/11-jsverbs-example-reference.md
      Note: Reference documentation for ScanOptions include/exclude/extensions
    - Path: go-go-goja/pkg/jsverbs/scan.go
      Note: Main implementation file for JSVerb filtering
    - Path: go-go-goja/ttmp/2026/06/08/XGOJA-CLUBMED-MODULES--address-xgoja-goja-module-improvements-from-clubmedmeetup-usage/design-doc/01-xgoja-clubmedmeetup-module-improvements-implementation-guide.md
      Note: |-
        Primary design and implementation guide created for this ticket.
        Primary design guide created in this ticket
    - Path: go-go-goja/ttmp/2026/06/08/XGOJA-CLUBMED-MODULES--address-xgoja-goja-module-improvements-from-clubmedmeetup-usage/design-doc/02-jsverb-source-filtering-implementation-guide.md
      Note: Focused JSVerb-only guide added during refocus
ExternalSources: []
Summary: Chronological diary for the XGOJA-CLUBMED-MODULES documentation and delivery work.
LastUpdated: 2026-06-08T18:06:00-04:00
WhatFor: Use this diary to understand how the ticket was created, what evidence was read, and how the design guide was prepared and delivered.
WhenToUse: Read when resuming this ticket or reviewing why these go-go-goja improvements were selected.
---




# Diary

## Goal

This diary records the creation of the `XGOJA-CLUBMED-MODULES` ticket in `go-go-goja/ttmp`, the evidence read from the ClubMedMeetup second-edition analysis, and the design guide written for a new intern.

## Step 1: Create the ticket and write the intern-facing design guide

I created a new docmgr ticket under `go-go-goja/ttmp` for the go-go-goja-side improvements identified by the ClubMedMeetup minitrace-viz usage analysis. The ticket is intentionally scoped to reusable xgoja/goja runtime improvements rather than site-specific ClubMedMeetup application changes.

I then read the source analysis and key go-go-goja implementation files, mapped the relevant build/runtime/provider flows, and wrote a detailed design document with prose, diagrams, API sketches, pseudocode, implementation phases, tests, and file references.

### Prompt Context

**User prompt (verbatim):** "Read @ClubMedMeetup/ttmp/2026/06/08/xgoja-modules-improvement-second-edition--improve-xgoja-and-goja-modules-from-clubmedmeetup-usage-patterns-second-edition/design-doc/01-xgoja-and-goja-module-improvement-map-second-edition.md and create a new ticket in go-go-goja/ttmp/ using `docmgr --root go-go-goja/ttmp ...`  to address the issue, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new docmgr ticket in the `go-go-goja` ticket workspace, use the existing ClubMedMeetup analysis as input evidence, write a comprehensive intern-facing design/implementation guide, store it in the ticket, and upload the resulting bundle to reMarkable.

**Inferred user intent:** Preserve the ClubMedMeetup-derived xgoja/goja module improvement plan as actionable go-go-goja ticket documentation that a new intern can use to implement the work.

**Commit (code):** N/A — documentation-only work in ticket workspace.

### What I did

- Ran `docmgr --root go-go-goja/ttmp status --summary-only` to confirm the target documentation root.
- Created ticket `XGOJA-CLUBMED-MODULES` with title `Address xgoja/goja module improvements from ClubMedMeetup usage`.
- Added a design document: `go-go-goja/ttmp/2026/06/08/XGOJA-CLUBMED-MODULES--address-xgoja-goja-module-improvements-from-clubmedmeetup-usage/design-doc/01-xgoja-clubmedmeetup-module-improvements-implementation-guide.md`.
- Added this investigation diary: `go-go-goja/ttmp/2026/06/08/XGOJA-CLUBMED-MODULES--address-xgoja-goja-module-improvements-from-clubmedmeetup-usage/reference/01-investigation-diary.md`.
- Read the user-specified source analysis under `ClubMedMeetup/ttmp/.../01-xgoja-and-goja-module-improvement-map-second-edition.md`.
- Read and referenced key go-go-goja files:
  - `cmd/xgoja/internal/buildspec/build_spec.go`
  - `cmd/xgoja/internal/buildspec/validate.go`
  - `cmd/xgoja/internal/generate/gomod.go`
  - `cmd/xgoja/cmd_build.go`
  - `cmd/xgoja/cmd_list_modules.go`
  - `pkg/xgoja/app/runtime_spec.go`
  - `pkg/xgoja/app/factory.go`
  - `pkg/xgoja/app/module_sections.go`
  - `pkg/xgoja/app/root.go`
  - `pkg/jsverbs/scan.go`
  - `pkg/xgoja/providerapi/module.go`
  - `pkg/xgoja/providerapi/capabilities.go`
  - `pkg/xgoja/providers/http/http.go`
  - `modules/express/express.go`
  - `pkg/xgoja/providers/host/host.go`
  - `modules/fs/fs.go`
  - `modules/fs/backend_embed.go`
- Wrote an intern-facing guide covering JSVerb filters, selected-module inventory, provider pin warnings, Express lifecycle, and fs capability metadata.

### Why

- The ClubMedMeetup analysis identified real integration pressure against xgoja modules, not isolated style issues.
- The requested deliverable needed to be clear enough for a new intern, so the guide starts with system architecture and moves gradually toward implementation tasks.
- The ticket needed to live in `go-go-goja/ttmp` because most selected improvements belong in the xgoja runtime and module framework.

### What worked

- `docmgr --root go-go-goja/ttmp ticket create-ticket ...` created the new ticket workspace successfully.
- `docmgr --root go-go-goja/ttmp doc add ...` created both the design doc and diary in the correct ticket.
- The source analysis already contained a strong problem inventory, allowing this ticket to focus on turning those findings into go-go-goja implementation guidance.
- The go-go-goja source files contained clear seams for each proposed change: buildspec DTOs, runtime spec DTOs, scanner options, generated runtime commands, provider capabilities, HTTP provider startup, and fs backend behavior.

### What didn't work

- Reading `go-go-goja/modules/fs/module.go` failed because the actual filesystem module implementation file is `go-go-goja/modules/fs/fs.go`.
  - Exact error: `ENOENT: no such file or directory, access '/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/modules/fs/module.go'`
- Reading `go-go-goja/pkg/xgoja/providerapi/provider.go` failed because provider API files are split across `provider_registry.go`, `module.go`, `capabilities.go`, `help.go`, `commands.go`, and related files.
  - Exact error: `ENOENT: no such file or directory, access '/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providerapi/provider.go'`

### What I learned

- `ModuleInstanceSpec.As` is the actual CommonJS alias installed into the runtime, and this is implemented consistently in both build-time and runtime spec types.
- Generated runtime `modules` currently reports provider catalog entries, while `xgoja list-modules -f xgoja.yaml` reports selected buildspec aliases. This explains the confusion observed in the ClubMedMeetup source analysis.
- The HTTP provider starts the listener from `NewExpressLoader`, which means `require("express")` can bind the configured port.
- The fs module already has a clean read-only backend; the missing piece is discoverable metadata, not mutation enforcement.

### What was tricky to build

- The main tricky part was separating repository responsibilities. The source analysis includes improvements for ClubMedMeetup, go-minitrace, goja-text, and rag-widget-site, but the new ticket is in `go-go-goja/ttmp`. The design therefore had to extract only the go-go-goja-owned work while still explaining the ClubMedMeetup evidence that motivated it.
- Another tricky part was documenting Express lifecycle changes without breaking existing xgoja apps. The guide proposes moving listener binding out of module load but preserving normal autostart behavior when routes are registered in `run server.js --keep-alive` flows.
- JSVerb filtering required careful wording because the immediate app-level workaround is to narrow `jsverbs.path`, but the reusable go-go-goja improvement should live in scanner options and xgoja schema.

### What warrants a second pair of eyes

- The proposed Express autostart boundary should be reviewed carefully because it changes when port binding errors occur.
- The provider pin warning should be reviewed against existing `Report`/doctor severity semantics before implementation.
- The `modules` command column rename should be checked for CLI compatibility if any generated app scripts parse it.
- JSVerb glob semantics should be agreed before implementation so docs, tests, and behavior match.

### What should be done in the future

- Validate the ticket with `docmgr doctor --ticket XGOJA-CLUBMED-MODULES --stale-after 30` after docmgr relations/changelog/tasks are updated.
- Upload the design/diary bundle to reMarkable as requested.
- Implement the phases in order: selected-module command, JSVerb filters, provider pin warnings, Express lifecycle, fs capabilities.

### Code review instructions

- Start with the design doc section `Current-state architecture` to understand the evidence and current implementation seams.
- Review proposed implementation phases against these files:
  - `pkg/xgoja/app/root.go`
  - `pkg/jsverbs/scan.go`
  - `cmd/xgoja/internal/buildspec/build_spec.go`
  - `pkg/xgoja/providers/http/http.go`
  - `modules/express/express.go`
  - `modules/fs/fs.go`
- Validate future implementation with focused tests:
  - `go test ./pkg/jsverbs ./pkg/xgoja/app ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate ./modules/fs ./modules/express ./pkg/xgoja/providers/http -count=1`

### Technical details

Commands used:

```bash
docmgr --root go-go-goja/ttmp status --summary-only
docmgr --root go-go-goja/ttmp ticket create-ticket \
  --ticket XGOJA-CLUBMED-MODULES \
  --title "Address xgoja/goja module improvements from ClubMedMeetup usage" \
  --topics xgoja,goja,modules,architecture,clubmedmeetup
docmgr --root go-go-goja/ttmp doc add \
  --ticket XGOJA-CLUBMED-MODULES \
  --doc-type design-doc \
  --title "XGoja ClubMedMeetup Module Improvements Implementation Guide"
docmgr --root go-go-goja/ttmp doc add \
  --ticket XGOJA-CLUBMED-MODULES \
  --doc-type reference \
  --title "Investigation Diary"
```

## Step 2: Refocus on JSVerb source filtering and implement it

The ticket was narrowed from the broad xgoja/goja module improvement map to the JSVerb source scanning issue only. I created a second guide focused exclusively on JSVerb filters, then implemented the focused code path one task at a time: schema fields, scanner options, runtime scan dispatch, validation, tests, formatting, and focused test execution.

The implementation keeps existing defaults unchanged while adding opt-in filtering. Existing JSVerb sources still scan `.js` and `.cjs` files unless configured otherwise, but xgoja buildspec authors can now declare include/exclude glob patterns and extension overrides to keep generated assets out of the scanner.

### Prompt Context

**User prompt (verbatim):** "nvm, I wantt o address the jsverb part only. create a second guide that focuses exclusively on that, then add tasks to the tticket, and work on them one by one, keeping a detailed diary as you work."

**Assistant interpretation:** Narrow the ticket to the JSVerb scanning problem, add a focused guide, update ticket tasks, implement the JSVerb work incrementally, and keep the diary detailed as implementation proceeds.

**Inferred user intent:** Avoid spending time on unrelated xgoja module improvements and turn the concrete ClubMedMeetup JSVerb scanner failure into a completed, test-backed go-go-goja change.

**Commit (code):** Not committed yet — working tree contains code and ticket documentation changes.

### What I did

- Added a focused design document:
  - `go-go-goja/ttmp/2026/06/08/XGOJA-CLUBMED-MODULES--address-xgoja-goja-module-improvements-from-clubmedmeetup-usage/design-doc/02-jsverb-source-filtering-implementation-guide.md`
- Rewrote `tasks.md` so the active checklist is JSVerb-only and broader ticket items are deferred.
- Extended build-time `JSVerbSourceSpec` in `cmd/xgoja/internal/buildspec/build_spec.go` with:
  - `Include []string`
  - `Exclude []string`
  - `Extensions []string`
- Extended runtime `JSVerbSourceSpec` in `pkg/xgoja/app/runtime_spec.go` with matching JSON fields.
- Extended `providerapi.JSVerbSourceDescriptor` in `pkg/xgoja/providerapi/commands.go` so command providers can inspect filters.
- Updated `pkg/xgoja/app/jsverb_sources.go` to include filter fields when listing JSVerb source descriptors.
- Extended `pkg/jsverbs.ScanOptions` in `pkg/jsverbs/model.go` with include/exclude slices.
- Implemented slash-normalized include/exclude matching in `pkg/jsverbs/scan.go` using `github.com/bmatcuk/doublestar/v4`.
- Normalized extension matching so configured extensions can be written with or without a leading dot.
- Updated `pkg/xgoja/app/root.go` so `scanVerbSource` builds scan options from the runtime source spec and passes them into provider, embedded, and filesystem source scans.
- Added validation in `cmd/xgoja/internal/buildspec/validate.go` for empty include/exclude/extension entries.
- Added tests:
  - `pkg/jsverbs/jsverbs_test.go`: `TestScanDirIncludeExcludeFilters`
  - `pkg/jsverbs/jsverbs_test.go`: `TestScanFSIncludeFilters`
  - `cmd/xgoja/internal/buildspec/validate_test.go`: filter validation tests
  - `cmd/xgoja/internal/generate/generate_test.go`: embedded runtime spec preserves `include`, `exclude`, and `extensions`
- Ran formatting and focused tests.

### Why

- The ClubMedMeetup failure was caused by scanning too much JavaScript, not by JSVerb extraction itself.
- The safest reusable fix is to let application authors constrain scan candidates declaratively in `xgoja.yaml`.
- Putting the filter logic in `pkg/jsverbs` instead of only in xgoja makes the behavior available to direct scanner callers and keeps traversal decisions close to scanner traversal.

### What worked

- Adding fields to both build-time and runtime specs was straightforward because `generate.RenderEmbeddedSpec` marshals `buildspec.JSVerbSourceSpec` into embedded runtime JSON.
- Existing scanner structure already collected relative paths before reading files, so filtering could be inserted before file reads/parsing.
- `doublestar.PathMatch` was already present in `go.mod` as an indirect dependency, so globstar-style patterns such as `assets/**` could be supported without writing a custom matcher.
- Focused validation passed:
  - `go test ./pkg/jsverbs ./pkg/xgoja/app ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1`

### What didn't work

- My first attempt to update `cmd/xgoja/internal/generate/generate_test.go` used an `oldText` block that appeared twice, so the edit tool rejected it.
  - Exact tool error: `Found 2 occurrences of edits[0] in go-go-goja/cmd/xgoja/internal/generate/generate_test.go. Each oldText must be unique. Please provide more context to make it unique.`
- I corrected this by including the surrounding `func TestRenderMainIncludesEmbeddedVerbFS` context in the replacement.

### What I learned

- `ScanDir` previously checked extensions before computing the slash-normalized relative path. The new filtering flow benefits from computing `relPathSlash` first, then applying both extension and pattern filters to the same normalized path.
- `ScanFS` needed the same filtering behavior as `ScanDir`; otherwise embedded and provider-shipped JSVerb sources would behave differently from filesystem sources.
- Runtime command providers see configured JSVerb sources through `providerapi.JSVerbSourceDescriptor`, so adding filters there keeps introspection consistent.
- Generated runtime spec preservation was important to test because build-time and runtime source spec types live in different packages.

### What was tricky to build

- The main implementation sharp edge was preserving default behavior. `ScanOptions` already had defaults for public function inclusion, extensions, and diagnostic behavior. The new `jsVerbScanOptions` helper in `pkg/xgoja/app/root.go` starts from `jsverbs.DefaultScanOptions()` and only overrides extensions when the source explicitly provides them, so old buildspecs keep scanning `.js` and `.cjs`.
- Another tricky part was path matching order. Matching OS-native paths would make filters platform-dependent, so the implementation normalizes relative paths and patterns with `filepath.ToSlash` before matching.
- Extension matching needed to be forgiving. The scanner now treats `js` and `.js` equivalently by adding a leading dot when necessary.

### What warrants a second pair of eyes

- Confirm that `doublestar.PathMatch` semantics match expected xgoja documentation, especially whether root-level files require both `*.js` and `**/*.js` include patterns.
- Review whether invalid glob patterns should be validation errors earlier. The current matcher ignores invalid patterns at scan time by treating them as non-matches; stricter validation could be added later.
- Review whether provider-shipped JSVerb sources should honor buildspec include/exclude filters. This implementation applies filters uniformly to provider, embedded, and filesystem sources.

### What should be done in the future

- Add user-facing documentation/examples for `include`, `exclude`, and `extensions` in xgoja docs or help pages.
- Consider adding a doctor warning for `jsverbs.path: .` with no filters once the report model supports warnings.
- Consider stricter glob validation if users need early feedback on malformed patterns.

### Code review instructions

- Start with the focused guide `design-doc/02-jsverb-source-filtering-implementation-guide.md`.
- Review schema changes first:
  - `cmd/xgoja/internal/buildspec/build_spec.go`
  - `pkg/xgoja/app/runtime_spec.go`
  - `pkg/xgoja/providerapi/commands.go`
- Review scanner behavior next:
  - `pkg/jsverbs/model.go`
  - `pkg/jsverbs/scan.go`
- Review xgoja runtime wiring:
  - `pkg/xgoja/app/root.go`
  - `pkg/xgoja/app/jsverb_sources.go`
- Review tests:
  - `pkg/jsverbs/jsverbs_test.go`
  - `cmd/xgoja/internal/buildspec/validate_test.go`
  - `cmd/xgoja/internal/generate/generate_test.go`
- Validate with:
  - `cd go-go-goja && go test ./pkg/jsverbs ./pkg/xgoja/app ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1`

### Technical details

Validation command run:

```bash
cd go-go-goja && gofmt -w \
  cmd/xgoja/internal/buildspec/build_spec.go \
  cmd/xgoja/internal/buildspec/validate.go \
  cmd/xgoja/internal/buildspec/validate_test.go \
  cmd/xgoja/internal/generate/generate_test.go \
  pkg/xgoja/app/runtime_spec.go \
  pkg/xgoja/app/root.go \
  pkg/xgoja/app/jsverb_sources.go \
  pkg/xgoja/providerapi/commands.go \
  pkg/jsverbs/model.go \
  pkg/jsverbs/scan.go \
  pkg/jsverbs/jsverbs_test.go && go test ./pkg/jsverbs ./pkg/xgoja/app ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1
```

Result:

```text
ok  	github.com/go-go-golems/go-go-goja/pkg/jsverbs	0.139s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/app	0.142s
ok  	github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec	0.006s
ok  	github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate	23.501s
```

## Step 3: Update user-facing JSVerb and xgoja documentation

After the code and focused guide were in place, I updated the product-facing documentation so users can discover the new JSVerb filtering options without reading ticket notes or source code. The documentation now covers both library-level `ScanOptions` and xgoja buildspec-level `jsverbs.include` / `jsverbs.exclude` / `jsverbs.extensions` fields.

The docs emphasize the operational reason for the feature: application roots often contain authored verb files next to bundled browser assets, generated files, or distribution output. Narrow scan roots remain the best default, but filters provide a safe declarative escape hatch when the layout cannot be narrowed cleanly.

### Prompt Context

**User prompt (verbatim):** "update docs."

**Assistant interpretation:** Update repository documentation for the JSVerb filtering feature implemented in Step 2.

**Inferred user intent:** Make the new feature discoverable and explain how to use it safely in both direct `pkg/jsverbs` callers and generated xgoja applications.

**Commit (code):** Not committed yet — working tree contains code and documentation changes.

### What I did

- Updated `pkg/doc/11-jsverbs-example-reference.md` with a new `Scan options` section covering:
  - default scanner options,
  - filtering order,
  - slash-relative path matching,
  - extension normalization,
  - a `ScanDir` example with include/exclude filters.
- Updated `pkg/doc/10-jsverbs-example-developer-guide.md` to explain scanner filters in the onboarding guide and add filter checks to the missing-verb debugging path.
- Updated `pkg/doc/08-jsverbs-example-overview.md` to mention that library callers and generated xgoja binaries can filter directory/`fs.FS` scans.
- Updated xgoja example docs:
  - `examples/xgoja/README.md`
  - `examples/xgoja/06-runtime-filesystem/README.md`
  - `examples/xgoja/07-embedded-jsverbs/README.md`
  - `examples/xgoja/08-provider-shipped-jsverbs/README.md`
- Updated `tasks.md` to include the documentation update as a completed JSVerb task.
- Ran focused validation including `pkg/doc`.

### Why

- The code feature changes xgoja buildspec shape, so the docs need to show the YAML fields explicitly.
- Direct `pkg/jsverbs` callers also need the `ScanOptions` contract, not only xgoja examples.
- The original ClubMedMeetup failure involved bundled assets under an application root, so the docs should explain when filters are appropriate and when a narrow source root is still preferable.

### What worked

- The docs were already organized around jsverbs overview/developer/reference pages, so the new content fit naturally:
  - overview: quick concept,
  - developer guide: operational debugging and mental model,
  - reference: exact API and rules.
- The xgoja examples already separate runtime filesystem, embedded local, and provider-shipped JSVerb sources, making it straightforward to document that filters apply to all three source kinds.
- Focused validation passed:
  - `go test ./pkg/doc ./pkg/jsverbs ./pkg/xgoja/app ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1`

### What didn't work

- While searching docs with `bash`, I used a command string containing unescaped backticks. Bash attempted command substitution for names such as `jsverbs-example`, `debug`, and `list`.
  - Exact stderr included:
    - `/bin/bash: line 35: jsverbs-example: command not found`
    - `/bin/bash: line 35: debug: command not found`
    - `/bin/bash: line 35: list: command not found`
- The command still returned useful surrounding document lines, but this is a reminder to avoid unquoted backticks in shell snippets.

### What I learned

- The docs already distinguish between `jsverbs-example` as a simple harness and `pkg/jsverbs` as a broader library. That was the right place to explain `ScanOptions`.
- xgoja users need examples at the source-kind level: runtime filesystem, embedded local, and provider-shipped sources.
- Documentation should explicitly say filters are relative to the source root; otherwise users may try repo-root-relative patterns that do not match.

### What was tricky to build

- The main tricky part was avoiding over-documenting this as an invitation to scan large application roots by default. The docs now say to prefer a narrow `path` when possible and use filters when a narrow root is inconvenient.
- Another tricky point was documenting root-level include patterns. Because glob semantics can vary, the examples use explicit patterns such as `site.js` plus `jsverbs/**/*.js` instead of implying one pattern handles every layout.

### What warrants a second pair of eyes

- Review whether the examples should also modify actual example `xgoja.yaml` files, or whether documenting optional filters in README snippets is sufficient.
- Review if a dedicated xgoja buildspec reference page should be added later; currently the guidance is distributed across examples and jsverbs help pages.

### What should be done in the future

- Add a concise schema/reference page for all xgoja buildspec fields if the project does not already have one.
- Consider adding `xgoja doctor` warnings for `jsverbs.path: .` without include/exclude filters.

### Code review instructions

- Review documentation changes in this order:
  - `pkg/doc/11-jsverbs-example-reference.md`
  - `pkg/doc/10-jsverbs-example-developer-guide.md`
  - `pkg/doc/08-jsverbs-example-overview.md`
  - `examples/xgoja/README.md`
  - `examples/xgoja/06-runtime-filesystem/README.md`
  - `examples/xgoja/07-embedded-jsverbs/README.md`
  - `examples/xgoja/08-provider-shipped-jsverbs/README.md`
- Validate with:
  - `cd go-go-goja && go test ./pkg/doc ./pkg/jsverbs ./pkg/xgoja/app ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1`

### Technical details

Validation command run:

```bash
cd go-go-goja && gofmt -w pkg/doc/doc.go >/dev/null && go test ./pkg/doc ./pkg/jsverbs ./pkg/xgoja/app ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1
```

Result:

```text
?   	github.com/go-go-golems/go-go-goja/pkg/doc	[no test files]
ok  	github.com/go-go-golems/go-go-goja/pkg/jsverbs	0.148s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/app	0.135s
ok  	github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec	0.005s
ok  	github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate	29.130s
```
