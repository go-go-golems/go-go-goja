---
Title: Investigation diary
Ticket: xgoja-jsverbs-loader-only-sources
Status: active
Topics:
    - xgoja
    - jsverbs
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological implementation diary for separating jsverb command discovery from helper-module require visibility in xgoja."
LastUpdated: 2026-06-09T00:49:33.733960289-04:00
WhatFor: "Record implementation decisions, commands, failures, validation, and review notes for this ticket."
WhenToUse: "Read before resuming work on xgoja jsverb source scanning or HTTP serve helper-module loading."
---

# Diary

## Goal

Capture the implementation of a small xgoja improvement that lets HTTP serve jsverb applications include CommonJS helper files for `require()` without accidentally exposing every helper function as a CLI command.

## Step 1: Create ticket and pin down the implementation shape

This step turned the ClubMed devctl failure into a focused xgoja task. The central observation is that xgoja already has a scanner-level `IncludePublicFunctions` switch, but the xgoja buildspec/runtime source schema does not expose it.

The chosen first pass is intentionally narrow: add per-source `includePublicFunctions` with default-preserving semantics, then validate it with tests before considering larger `loaderInclude`/`verbInclude` separation.

### Prompt Context

**User prompt (verbatim):** "Ok, create a ticket in `docmgr --root go-go-goja/ttmp/ ...` and a clear concise implementation guide and tasks, and then work on it task by task, keeping a diary as you work, and committing at appropriat eintervals"

**Assistant interpretation:** Create a docmgr ticket under the go-go-goja ticket root, document the design and task list, then implement the xgoja/jsverbs fix incrementally with diary updates and commits.

**Inferred user intent:** Convert the ad-hoc devctl debugging result into a tracked upstream xgoja improvement and complete it carefully enough to unblock the dependent ClubMed app.

**Commit (code):** N/A — planning/docs only so far.

### What I did
- Created ticket `xgoja-jsverbs-loader-only-sources` under `go-go-goja/ttmp`.
- Added `design-doc/01-implementation-guide.md`.
- Added this diary at `reference/01-investigation-diary.md`.
- Replaced the default `tasks.md` placeholder with a concrete seven-step checklist.
- Re-read the relevant implementation files:
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/internal/buildspec/build_spec.go`
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/app/runtime_spec.go`
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/app/root.go`
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/jsverbs/model.go`
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/jsverbs/scan.go`
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/http/serve.go`

### Why
- The ClubMed app needs `site.js` to expose one `start` command while `server.js` and `lib/**/*.js` remain regular modules.
- xgoja currently has no buildspec knob for this shape even though the underlying jsverbs scanner supports explicit-only discovery.

### What worked
- `docmgr --root go-go-goja/ttmp ticket create-ticket ...` created the desired workspace.
- The existing scanner option provides a small implementation path.

### What didn't work
- Earlier debugging showed that including helper files in `jsverbs.include` caused the scanner to discover many helper functions as public commands.
- Earlier debugging also showed that including only `site.js` made `require("./server.js")` fail at runtime because `Registry.RequireLoader()` only serves scanned files.

### What I learned
- The bug is not only an app launch issue; it is an xgoja ergonomics/API issue.
- The smallest safe fix is option propagation rather than a new loader architecture.

### What was tricky to build
- The tricky invariant is preserving current jsverbs behavior. `jsverbs.DefaultScanOptions()` enables public function discovery by default, so xgoja must not turn omission into false accidentally.
- The implementation should likely use a pointer/nullable bool in the xgoja source spec so explicit `false` can be distinguished from an omitted field.

### What warrants a second pair of eyes
- The exact YAML/JSON field name: `includePublicFunctions` is clear and maps directly to the scanner option, but reviewers should confirm it is the right long-term API name.
- The generated runtime JSON semantics should be checked so old buildspecs remain unchanged.

### What should be done in the future
- Consider a larger follow-up with separate `loaderInclude` and command-discovery include globs if explicit-only discovery is insufficient for larger applications.

### Code review instructions
- Start with the implementation guide, then review `JSVerbSourceSpec` in buildspec/runtime structs and `jsVerbScanOptions`.
- Validate with targeted tests before exercising generated binaries.

### Technical details
- Current scanner switch: `jsverbs.ScanOptions.IncludePublicFunctions`.
- Current default: `jsverbs.DefaultScanOptions().IncludePublicFunctions == true`.
- Desired xgoja config:

```yaml
jsverbs:
  - id: minitrace-viz-site
    path: .
    includePublicFunctions: false
    include:
      - site.js
      - server.js
      - lib/**/*.js
```

## Step 2: Remove implicit public-function command discovery

This step changed the design from a configurable opt-out to a clean break. The scanner now treats `__verb__()` metadata as the only command declaration mechanism, while scanned helper modules remain available to the jsverb `require()` loader.

The immediate result is simpler than the first-pass option design: xgoja source specs do not need a compatibility field, and application authors do not need to mark helper modules specially. If a function should be a command, it must have matching `__verb__()` metadata.

### Prompt Context

**User prompt (verbatim):** "let's remove the option in its entirety, and only allow __verb__ . remove the options and backwards wrappers and compabitility."

**Assistant interpretation:** Do not add or keep `includePublicFunctions`; remove implicit public-function discovery from jsverbs/xgoja entirely and make explicit `__verb__()` metadata mandatory.

**Inferred user intent:** Prefer a clean, non-legacy API over a backwards-compatible switch, even if existing public-function-only fixtures need to be updated.

**Commit (code):** 833edd587db205318cc408464053566d74c11b5f — "Require explicit jsverb command metadata"

### What I did
- Removed `ScanOptions.IncludePublicFunctions` from `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/jsverbs/model.go`.
- Removed the implicit public-function verb creation loop from `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/jsverbs/scan.go`.
- Removed transient `includePublicFunctions` schema/descriptor wiring from:
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/internal/buildspec/build_spec.go`
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/app/runtime_spec.go`
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providerapi/commands.go`
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/app/jsverb_sources.go`
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/app/root.go`
- Updated fixtures/tests so intended commands have explicit `__verb__()` metadata.
- Added/kept HTTP serve regression coverage in `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/http/serve_test.go` showing helper functions do not become commands while helper modules remain require-loadable.
- Updated docs in:
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/doc/06-buildspec-reference.md`
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/doc/12-tutorial-http-serve-jsverbs.md`
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md`
  - `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/doc/11-jsverbs-example-reference.md`

### Why
- Public-function command discovery is convenient for sketches but too surprising for apps with helper modules.
- The user requested no compatibility wrappers/options, so the final API should have exactly one command declaration mechanism.

### What worked
- Targeted validation passes:

```bash
go test ./pkg/jsverbs ./pkg/xgoja/app ./cmd/xgoja/internal/... ./pkg/xgoja/providers/http
```

- The code commit pre-commit hook also passed full repository lint/generate/test:

```text
✔️ lint (10.52 seconds)
✔️ test (45.67 seconds)
```

- `rg -n "IncludePublicFunctions|includePublicFunctions" -S --glob '!ttmp/**'` returns no active code/doc references.

### What didn't work
- The first HTTP serve regression test attempt timed out:

```text
--- FAIL: TestServeVerbLoadsIncludedHelperModulesWithoutHelperCommands (5.05s)
    serve_test.go:148: timed out waiting for http://127.0.0.1:33719/healthz
```

- Root cause: the test used a fresh `newHTTPCapability()` in `SelectedModules`, but the Express module loader was registered with the capability instance stored inside the provider registry. Section initialization configured the wrong capability instance.
- Fix: obtain the provider capability via `providers.ResolvePackageCapabilities(PackageID)` and pass that exact capability slice in `SelectedModules`.

### What I learned
- The HTTP provider capability stores runtime listener settings per Goja runtime, so tests that manually construct `CommandSetContext.SelectedModules` must use the same capability instance as the provider registry.
- Removing implicit discovery did not require loader changes because scanned files are still stored in `registry.filesByModule`; only command finalization changed.

### What was tricky to build
- The main tricky part was separating two facts that used to be coupled: a scanned file is loader-visible, but a scanned function is not automatically a command. The implementation had to remove only the implicit command creation loop while leaving file scanning and `RequireLoader()` untouched.
- The HTTP regression test also needed realistic command-provider section initialization. Without the correct capability instance, route registration appeared to succeed from JavaScript but no listener was configured on the expected address.

### What warrants a second pair of eyes
- Confirm the clean break is acceptable for any external users who relied on public-function-only command discovery.
- Review docs/examples to ensure no remaining prose suggests plain public functions become commands.
- Review the HTTP test for its manual `CommandSetContext` construction; it mirrors generated-provider behavior but is slightly lower-level than a full generated binary test.

### What should be done in the future
- Validate the dependent ClubMed `minitrace-viz` devctl path by rebuilding with the local xgoja checkout and scanning `site.js`, `server.js`, and `lib/**/*.js`.
- Consider a future `loaderInclude`/`verbInclude` split only if explicit `__verb__()` discovery is not enough.

### Code review instructions
- Start in `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/jsverbs/scan.go`, specifically `finalizeVerbs`.
- Then review `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/jsverbs/model.go` to confirm the scanner option is gone.
- Check `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/http/serve_test.go` for the helper-module regression.
- Validate with:

```bash
go test ./pkg/jsverbs ./pkg/xgoja/app ./cmd/xgoja/internal/... ./pkg/xgoja/providers/http
```

### Technical details
- New discovery rule: scanned top-level functions are only command candidates when a matching `__verb__()` declaration exists.
- Helper modules are still available through `Registry.RequireLoader()` because scanning still records every included source file in `registry.filesByModule`.

### Validation addendum: ClubMed devctl path

After the targeted go-go-goja tests passed, I rebuilt the dependent ClubMed binary with the local xgoja checkout and the local go-go-goja module replacement:

```bash
cd ClubMedMeetup/minitrace-viz
go run ../../go-go-goja/cmd/xgoja build -f xgoja.yaml --xgoja-version v0.8.6 --xgoja-replace "$(cd ../../go-go-goja && pwd)"
```

The generated command tree contained the intended command only:

```text
minitrace-viz serve site start
```

The direct smoke test passed:

```bash
cd ClubMedMeetup/minitrace-viz
(lsof -ti tcp:18787 | xargs -r kill || true) && bash test-fixtures/smoke-test.sh
```

Result:

```text
=== Results: 6 passed, 0 failed ===
```

The original devctl path also passed:

```bash
cd ClubMedMeetup
(lsof -ti tcp:8787 | xargs -r kill || true) && devctl build && devctl up
devctl status
curl -fsS http://127.0.0.1:8787/api/widget/health
```

Result:

```text
up complete services=1
{"service":"minitrace-viz-widget-site","status":"ok"}
```
