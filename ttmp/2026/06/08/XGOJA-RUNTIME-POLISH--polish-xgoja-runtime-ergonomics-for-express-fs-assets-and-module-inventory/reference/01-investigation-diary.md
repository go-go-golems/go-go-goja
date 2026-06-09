---
Title: Investigation Diary
Ticket: XGOJA-RUNTIME-POLISH
Status: active
Topics:
    - xgoja
    - goja
    - modules
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/ttmp/2026/06/08/XGOJA-RUNTIME-POLISH--polish-xgoja-runtime-ergonomics-for-express-fs-assets-and-module-inventory/design-doc/01-runtime-ergonomics-polish-implementation-guide.md
      Note: |-
        Initial implementation guide for the runtime polish ticket.
        Initial runtime polish implementation guide
ExternalSources: []
Summary: Chronological diary for the runtime ergonomics polish ticket.
LastUpdated: 2026-06-08T23:50:00-04:00
WhatFor: Use this diary when implementing the smaller go-go-goja runtime polish items.
WhenToUse: Read before resuming work on Express lazy startup, fs:assets capability metadata, or selected module inventory.
---


# Diary

## Goal

Capture the setup and implementation history for small `go-go-goja` runtime ergonomics improvements: Express listener lifecycle, read-only embedded filesystem discovery, and selected runtime module inventory.

## Step 1: Create the runtime polish ticket

I created a new `docmgr` ticket for the smaller follow-up issues that remained after the JSVerb source-filtering work was completed. These items are intentionally grouped because they are all framework/runtime polish items in `go-go-goja`, not site-specific ClubMedMeetup changes.

The initial guide records the ownership answer explicitly: yes, these should live in `go-go-goja`. ClubMedMeetup exposed the pain points, but the code paths are the xgoja HTTP provider, the reusable Express module, the reusable fs module, and generated runtime commands.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket to tackle smaller issues:

- express should not eagerly bind on require
- fs:assets should avertise read-only behavior
- runtime module inventory i sonfusing (if we aren't using glazed command for the strucutred that, which then provides the requested --output json flag, then we should do that)"

**Assistant interpretation:** Create a new go-go-goja docmgr ticket for three smaller runtime ergonomics improvements and prepare implementation guidance/tasks.

**Inferred user intent:** Continue addressing the ClubMedMeetup-derived runtime friction, but split the remaining smaller go-go-goja issues out of the JSVerb-focused ticket.

**Commit (code):** N/A — ticket setup only.

### What I did

- Created ticket `XGOJA-RUNTIME-POLISH` under `go-go-goja/ttmp`.
- Added design doc `design-doc/01-runtime-ergonomics-polish-implementation-guide.md`.
- Added this diary `reference/01-investigation-diary.md`.
- Wrote tasks for:
  - Express lazy/no-eager bind behavior,
  - `fs:assets` read-only capability metadata,
  - selected runtime module inventory as a structured Glazed command.

### Why

- The three issues all live in reusable `go-go-goja` runtime/provider/module code.
- Keeping them in one ticket is reasonable because each is small, but the guide recommends implementing them independently and in risk order.
- The selected-module inventory issue specifically should use Glazed command machinery to get structured renderers such as `--output json` rather than hand-printing ad hoc text.

### What worked

- `docmgr --root go-go-goja/ttmp ticket create-ticket` created the ticket workspace successfully.
- The design guide could reuse evidence and file ownership already discovered during the earlier ClubMedMeetup analysis.

### What didn't work

- N/A.

### What I learned

- These issues are best framed as generated runtime ergonomics rather than ClubMedMeetup app bugs.
- Express lifecycle is the riskiest because changing listener start timing can affect existing apps, so it should come after the additive selected-module and fs metadata work.

### What was tricky to build

- The main tricky point was keeping scope tight. The earlier broad guide also discussed provider pinning, go-minitrace consolidation, and goja-text helpers, but this new ticket should only track the three small runtime polish items named in the prompt.

### What warrants a second pair of eyes

- Express lazy-start compatibility plan.
- Whether selected-module inventory should be a new command or an option on the existing `modules` provider catalog command.
- Exact shape of `fs.capabilities()` and whether mount metadata should be included in the first patch.

### What should be done in the future

- Implement selected module inventory first.
- Implement fs read-only metadata second.
- Implement Express lazy binding third.
- Keep this diary updated with tests, failures, and commit hashes.

### Code review instructions

- Start with `design-doc/01-runtime-ergonomics-polish-implementation-guide.md`.
- Review implementation by subsystem:
  - generated commands: `pkg/xgoja/app/root.go`, `pkg/xgoja/app/runtime_spec.go`,
  - fs metadata: `modules/fs/fs.go`, `modules/fs/backend_embed.go`,
  - Express lifecycle: `pkg/xgoja/providers/http/http.go`, `modules/express/express.go`.

### Technical details

Initial ticket commands:

```bash
docmgr --root go-go-goja/ttmp ticket create-ticket \
  --ticket XGOJA-RUNTIME-POLISH \
  --title "Polish xgoja runtime ergonomics for Express FS assets and module inventory" \
  --topics xgoja,goja,modules,architecture
```

## Step 2: Add selected runtime module inventory

I implemented the lowest-risk runtime polish item first: generated binaries now get a `selected-modules` command that lists the actual CommonJS aliases selected into the runtime. The existing `modules` command remains a provider catalog, but its help text and output column now make that clearer by using `provider_ref` instead of implying a valid `require()` name.

This matters because xgoja lets one provider module be selected multiple times under different aliases, such as `fs:host` and `fs:assets`. A provider catalog can tell you that `go-go-goja-host.fs` exists, but only the runtime spec can tell you which aliases the generated binary actually installed.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Work through the ticket tasks incrementally, starting with a small commit-sized runtime inventory improvement.

**Inferred user intent:** Make generated xgoja runtimes easier to inspect and ensure module inventory output supports structured output such as `--output json`.

**Commit (code):** Pending — to be committed after this diary update.

### What I did

- Added `selectedModulesCommand` in `pkg/xgoja/app/root.go`.
- Attached it from `Host.AttachDefaultCommands` via new `AttachSelectedModules` in `pkg/xgoja/app/host.go`.
- Kept `modules` as a compiled provider catalog, but clarified its long help and changed the structured column from `require` to `provider_ref`.
- Added a test in `pkg/xgoja/app/root_test.go` that runs `selected-modules --output json` and checks aliases, provider refs, and config output.
- Updated `tasks.md` to check off the module inventory tasks.

### Why

- This is additive, low risk, and directly addresses the confusing distinction between compiled provider modules and selected runtime aliases.
- Implementing it as a Glazed command means existing renderer/middleware support provides `--output json` without custom JSON printing.

### What worked

- The existing `modulesCommand` pattern was already a Glazed command, so `selectedModulesCommand` could follow the same structure.
- Runtime spec already carries `Modules`, including `package`, `name`, `as`, and static config, so no schema change was needed.
- Focused test passed:
  - `go test ./pkg/xgoja/app -run 'TestGeneratedRoot(ModulesCommand|SelectedModulesCommand)' -count=1`

### What didn't work

- My first selected-modules test tried to capture output through `Options.Out`, but Glazed JSON output was written to stdout in this test path.
- The failed test printed the JSON to the console and then saw an empty buffer.
- I fixed the test by using the existing `captureStdout(...)` helper.

### What I learned

- `Options.Out` is sufficient for some generated command paths but not all Glazed renderer output in tests.
- Testing the actual `--output json` invocation is valuable because it proves the command is structured rather than hand-printed.

### What was tricky to build

- The main tricky point was compatibility: changing `modules` too aggressively could break users who expect a provider catalog. I kept the command and clarified it rather than replacing it.
- Config output is represented as a compact JSON string per row. This keeps table output simple while preserving structured data for JSON renderers.

### What warrants a second pair of eyes

- Whether `modules` should keep a legacy `require` column for one release in addition to `provider_ref`.
- Whether `selected-modules.config` should eventually be a nested object rather than a JSON string.

### What should be done in the future

- Add user-facing docs for `selected-modules` after the full runtime polish ticket lands.
- Consider adding `selected-modules` examples to xgoja generated binary docs.

### Code review instructions

- Start in `pkg/xgoja/app/host.go` to see command attachment.
- Then review `pkg/xgoja/app/root.go`, especially `selectedModulesCommand` and the clarified provider catalog command.
- Validate with:
  - `go test ./pkg/xgoja/app -run 'TestGeneratedRoot(ModulesCommand|SelectedModulesCommand)' -count=1`

### Technical details

Focused validation command:

```bash
cd go-go-goja && gofmt -w pkg/xgoja/app/root.go pkg/xgoja/app/host.go pkg/xgoja/app/root_test.go && go test ./pkg/xgoja/app -run 'TestGeneratedRoot(ModulesCommand|SelectedModulesCommand)' -count=1
```

Result:

```text
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/app	0.049s
```
