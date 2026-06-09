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

## Step 3: Add filesystem read-only capability metadata

I implemented the `fs:assets` discoverability improvement by adding backend capability metadata to the filesystem module. Embedded read-only backends now report that they are read-only, while the host OS backend reports read/write capability. JavaScript code can inspect `fs.isReadOnly` and `fs.capabilities()` without attempting a write just to discover behavior.

The existing mutation behavior is unchanged: embedded filesystems still reject writes, mkdirs, removes, appends, renames, and copies with read-only errors where appropriate. The new API is purely descriptive.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue the task list after selected-module inventory by making `fs:assets` advertise read-only behavior.

**Inferred user intent:** Improve runtime self-discovery so generated xgoja app authors can distinguish writable host fs aliases from read-only embedded asset fs aliases.

**Commit (code):** Pending — to be committed after this diary update.

### What I did

- Added `Capabilities`, `MountInfo`, `CapabilityReporter`, and `CapabilitiesForBackend` in `modules/fs/backend.go`.
- Added `FSCapabilities()` to `OSBackend`, returning `{backend: "host", read: true, write: true}`.
- Added `FSCapabilities()` to `ReadOnlyFSBackend`, returning `{backend: "embedded", read: true, write: false, embedded: true, mounts: [...]}`.
- Exported `isReadOnly` and `capabilities()` from the JavaScript fs module in `modules/fs/fs.go`.
- Updated TypeScript declaration metadata with `FSMountInfo`, `FSCapabilities`, `isReadOnly`, and `capabilities()`.
- Added host and embedded fs tests:
  - `TestHostFsCapabilities`
  - extended `TestReadOnlyEmbeddedFsSync`
- Updated `tasks.md` to check off fs capability metadata tasks.

### Why

- `fs:assets` already behaves correctly at mutation time, but users need an introspection API that says writes are impossible.
- Keeping one fs module API is backward compatible; metadata avoids splitting read-only and writable modules into separate JavaScript APIs.

### What worked

- The backend abstraction was a good place to attach capability reporting because the module can derive `isReadOnly` from actual backend behavior.
- Focused fs tests passed after normalizing the JS export shape:
  - `go test ./modules/fs -count=1`

### What didn't work

- My first `capabilities()` export returned the Go `Capabilities` struct directly. In JavaScript, `JSON.stringify(fs.capabilities())` did not expose lower-camel/json-tagged fields as expected; the host test only saw `{"isReadOnly":false}`.
- The embedded test also failed because `caps.mounts` was undefined:
  - `TypeError: Cannot read property '0' of undefined at <eval>:12:311(112)`
- I fixed this by converting `Capabilities` to an explicit `map[string]any` with lower-camel keys before returning it from the JS function.

### What I learned

- Go struct JSON tags are not the right mechanism for shaping Goja-exported JavaScript objects.
- For JS-facing native module APIs, explicit maps or objects are safer when field casing is part of the public contract.

### What was tricky to build

- The main tricky point was making the capability API backend-owned without forcing every backend implementation to change. `CapabilitiesForBackend` returns custom read/write defaults when a backend does not implement `CapabilityReporter`.
- Mount metadata needed to avoid exposing `fs.FS` internals. The public mount info only includes virtual mount and cleaned root.

### What warrants a second pair of eyes

- Whether the default for unknown custom backends should be `write: true` or a more conservative `write: false`.
- Whether detailed mount metadata should include asset IDs when the backend came from xgoja assets; currently the backend only knows mount/root.

### What should be done in the future

- Update `pkg/doc/24-fs-module.md` with the new capability API.
- Consider exposing backend capabilities through generated module inventory later if useful.

### Code review instructions

- Start with `modules/fs/backend.go` to review the public capability types.
- Review backend implementations in `modules/fs/fs_sync.go` and `modules/fs/backend_embed.go`.
- Review JS exports and TypeScript declaration changes in `modules/fs/fs.go`.
- Validate with:
  - `go test ./modules/fs -count=1`

### Technical details

Focused validation command:

```bash
cd go-go-goja && gofmt -w modules/fs/backend.go modules/fs/backend_embed.go modules/fs/fs.go modules/fs/fs_sync.go modules/fs/fs_test.go modules/fs/fs_embed_test.go && go test ./modules/fs -count=1
```

Result:

```text
ok  	github.com/go-go-golems/go-go-goja/modules/fs	0.052s
```

## Step 4: Make Express require side-effect-light

I changed the xgoja HTTP provider and Express module so that `require("express")` no longer binds the configured HTTP listener. Listener startup is now deferred until the app is actually used for route/static registration or `app.listen()` is called.

This keeps introspection safe while preserving the generated app pattern where `server.js` requires Express, creates an app, registers routes/static mounts, and then relies on `run --keep-alive` to keep the runtime alive.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue the runtime polish task list by addressing Express eager binding.

**Inferred user intent:** Make `require("express")` safe for eval/repl/introspection and avoid port conflicts unless the script actually starts using HTTP server behavior.

**Commit (code):** Pending — to be committed after this diary update.

### What I did

- Added `StartFunc` and `WithOnUse` to `modules/express`.
- Added `Registrar.start(vm)` as a small helper that invokes the optional start hook.
- Changed route registration (`get`, `post`, `put`, `patch`, `delete`, `all`) to call the start hook before registering the handler.
- Changed static/SPAs mount helpers to call the start hook before registering static handlers.
- Added `app.listen()` as an explicit start path.
- Changed `pkg/xgoja/providers/http.NewExpressLoader` so module loading only constructs exports and passes the start hook into Express; it no longer calls `c.start(...)` during `require("express")`.
- Added `TestExpressRequireDoesNotBindHTTPPort` in `pkg/xgoja/providers/http/http_test.go`.
- Updated `tasks.md` to check off Express lifecycle and test tasks.

### Why

- CommonJS `require()` should be safe for introspection and should not perform network IO merely by loading a module.
- Route/static registration is a better compatibility boundary because existing server scripts register routes immediately after creating the app.
- An explicit `app.listen()` hook gives scripts a direct start path without importing provider internals.

### What worked

- The existing Express registrar option pattern made it easy to add `WithOnUse` without creating an import cycle between `modules/express` and `pkg/xgoja/providers/http`.
- Focused tests passed:
  - `go test ./pkg/xgoja/providers/http ./modules/express -count=1`

### What didn't work

- N/A for the final implementation. The design fit the existing registrar structure cleanly.

### What I learned

- The HTTP provider already centralizes listener state in `runtimeEntry`; the Express module only needed a start callback to defer listener binding.
- This is a cleaner separation than having Express know about xgoja HTTP settings.

### What was tricky to build

- The main tricky point was preserving compatibility. Starting only on explicit `listen()` would be cleaner, but it could break existing scripts that expect route registration to be enough. Starting on first route/static registration preserves that behavior while fixing require-only introspection.
- The test needed to prove both sides: require-only succeeds while route registration still reports an occupied port when configured to bind one.

### What warrants a second pair of eyes

- Whether `app.listen()` should accept a listen address override. The first implementation simply triggers the configured xgoja listener.
- Whether static handler construction should happen before or after start. The current implementation starts first, matching route behavior.

### What should be done in the future

- Document the lifecycle: `require()` is pure, route/static registration autostarts when HTTP is enabled, `app.listen()` explicitly starts.
- Consider exposing server state for diagnostics if users ask for it.

### Code review instructions

- Review `pkg/xgoja/providers/http/http.go` to confirm listener startup moved out of module load.
- Review `modules/express/express.go` for the `WithOnUse` callback and the registration-time start calls.
- Review `pkg/xgoja/providers/http/http_test.go` for the require-only port-conflict regression test.
- Validate with:
  - `go test ./pkg/xgoja/providers/http ./modules/express -count=1`

### Technical details

Focused validation command:

```bash
cd go-go-goja && gofmt -w modules/express/express.go pkg/xgoja/providers/http/http.go pkg/xgoja/providers/http/http_test.go && go test ./pkg/xgoja/providers/http ./modules/express -count=1
```

Result:

```text
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http	0.027s
ok  	github.com/go-go-golems/go-go-goja/modules/express	0.035s
```
