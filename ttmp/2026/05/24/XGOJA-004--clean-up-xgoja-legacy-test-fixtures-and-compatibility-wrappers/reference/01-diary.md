---
Title: Diary
Ticket: XGOJA-004
Status: active
Topics:
    - xgoja
    - cleanup
    - goja
    - refactor
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/24/XGOJA-004--clean-up-xgoja-legacy-test-fixtures-and-compatibility-wrappers/design-doc/01-cleanup-implementation-guide.md
      Note: Cleanup implementation guide created in Step 1
ExternalSources: []
Summary: Chronological diary for the hard-cutover cleanup of xgoja legacy fixtures and compatibility wrappers.
LastUpdated: 2026-05-24T12:26:47.115261226-04:00
WhatFor: Record cleanup decisions, commands, failures, commits, and validation evidence.
WhenToUse: Read before continuing XGOJA-004 cleanup or reviewing the cleanup commits.
---

# Diary

## Goal

This diary records the XGOJA-004 hard-cutover cleanup work: removing dead xgoja fixture code, deleting tracked artifact files, removing obsolete jsverbs lightweight invocation, removing deprecated engine default-registry wrappers, updating docs/tests, and validating each step.

## Step 1: Create cleanup ticket and hard-cutover implementation guide

The first step converted the cleanup discussion into a tracked docmgr ticket. The user asked whether duplicate xgoja fixture providers, deprecated compatibility wrappers, and old tests could be removed. The answer was yes for the legacy fixture and tracked artifact files, probably yes for the jsverbs lightweight invocation path after the xgoja engine-runtime migration, and yes for deprecated engine DefaultRegistry wrappers under a hard-cutover policy.

I created XGOJA-004 and wrote an implementation guide that separates low-risk file deletion from public API cleanup. The guide deliberately avoids backwards-compatibility shims because the follow-up instruction was explicit: this cleanup should be a hard cutover.

### Prompt Context

**User prompt (verbatim):** "make a separate cleanup ticket, and add all these things as tasks + implementation guide. then work on them step by stepp, committing at appropriate intervals, keeping a detailed diary"

**Assistant interpretation:** Create a docmgr cleanup ticket with tasks and an implementation guide, then implement the cleanup incrementally with commits and diary updates.

**Inferred user intent:** Keep cleanup work disciplined and reviewable rather than mixing it into prior xgoja feature commits.

**Commit (code):** Pending for docs/ticket creation.

### What I did

- Created ticket `XGOJA-004 — Clean up xgoja legacy test fixtures and compatibility wrappers`.
- Added documents:
  - `design-doc/01-cleanup-implementation-guide.md`
  - `reference/01-diary.md`
- Added tasks for:
  - deleting `cmd/xgoja/internal/testprovider` and tracked `.orig` files,
  - removing `InvokeInGojaRuntime`,
  - removing deprecated engine `DefaultRegistry*` wrappers,
  - removing `modules.EnableAll`,
  - updating docs/tests/validation.
- Wrote the implementation guide with concrete file paths, removal sequence, and validation commands.

### Why

- The codebase had accumulated transitional APIs and files during the xgoja implementation sequence.
- The cleanup touches exported APIs and docs, so it needs a separate review trail.
- The user explicitly requested step-by-step work and a detailed diary.

### What worked

- `docmgr` created the ticket workspace, guide, diary, tasks, README, index, and changelog.
- The cleanup scope is now explicit enough to implement in focused commits.

### What didn't work

- N/A

### What I learned

- `cmd/xgoja/internal/testprovider` is only referenced by historical ticket docs; current tests/examples use `pkg/xgoja/testprovider`.
- The tracked `.orig` files include stale backup content and one file with merge-conflict markers.
- `InvokeInGojaRuntime` is now a transition API because xgoja uses `engine.Runtime` and `InvokeInRuntime`.
- The deprecated engine DefaultRegistry wrappers require a broader docs/test update than the fixture cleanup.

### What was tricky to build

- The tricky part was distinguishing low-risk dead code from exported API cleanup. Removing tracked `.orig` files is straightforward. Removing `DefaultRegistry*` helpers affects public documentation and tests, so it needs more careful validation.

### What warrants a second pair of eyes

- The engine default-registry API removal should be reviewed carefully because it intentionally breaks exported deprecated helpers.
- The docs update should be checked to ensure every old helper reference is replaced by the correct middleware pattern.

### What should be done in the future

- Implement Step 2: delete the legacy fixture and `.orig` files.
- Then remove `InvokeInGojaRuntime`.
- Then remove deprecated engine wrappers and update docs.

### Code review instructions

- Start with the implementation guide:
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/24/XGOJA-004--clean-up-xgoja-legacy-test-fixtures-and-compatibility-wrappers/design-doc/01-cleanup-implementation-guide.md`
- Review the task list:
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/24/XGOJA-004--clean-up-xgoja-legacy-test-fixtures-and-compatibility-wrappers/tasks.md`

### Technical details

Initial cleanup inventory commands included:

```bash
find cmd/xgoja/internal/testprovider pkg/xgoja/testprovider -type f -maxdepth 5 -print -exec wc -l {} \;
rg -n "internal/testprovider|pkg/xgoja/testprovider|testprovider" cmd/xgoja pkg/xgoja examples ttmp -S
rg -n "Deprecated|deprecated|legacy|compat|backwards|backward|wrapper|WithRuntimeModuleRegistrars|RegisterRuntimeModules|RuntimeModuleRegistrar|engine.ModuleSpec|ModuleSpec|InvokeInGojaRuntime|DataOnlyDefaultRegistryModules|ImplicitDefault" engine pkg cmd modules -S
```

## Step 2: Delete legacy fixture and tracked artifact files

This step removed the safest cleanup targets first. The internal xgoja test provider was an early fixture package that became obsolete once generated-binary tests needed a public importable provider under `pkg/xgoja/testprovider`. The tracked `.orig` files were stale editor/merge artifacts and one of them still contained merge-conflict markers.

The important result is that the active xgoja fixture surface is now unambiguous: tests and examples use `pkg/xgoja/testprovider`. The repository also no longer carries tracked backup files that are not part of the build.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Continue the hard-cutover cleanup by implementing the low-risk deletion task first.

**Inferred user intent:** Remove obviously dead code and artifact files before tackling exported API cleanup.

**Commit (code):** Pending for this step.

### What I did

- Deleted:
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/xgoja/internal/testprovider/provider.go`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/config.go.orig`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/runtime.go.orig`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/modules/exports.go.orig`
- Marked task 1 complete.
- Updated the XGOJA-004 changelog.
- Validated with:

```bash
GOWORK=off go test ./cmd/xgoja ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./pkg/xgoja/testprovider -count=1
```

### Why

- `cmd/xgoja/internal/testprovider` was no longer imported by active code.
- `pkg/xgoja/testprovider` is the current fixture provider and supports provider-shipped verbs and owner-binding tests.
- Tracked `.orig` files are repository noise and should not be maintained.

### What worked

- Focused xgoja tests passed after deletion.
- No active code referenced the deleted internal provider.

### What didn't work

- N/A

### What I learned

- The duplicate fixture provider was already documented as older in the intern guide, but it had not yet been removed from the repository.
- The `.orig` files were tracked, not just untracked local editor backups.

### What was tricky to build

- Nothing was technically tricky in this step. The main caution was checking active references before deleting the internal provider.

### What warrants a second pair of eyes

- N/A

### What should be done in the future

- Remove the obsolete `InvokeInGojaRuntime` path next.

### Code review instructions

- Verify that no active imports reference `cmd/xgoja/internal/testprovider`.
- Verify focused xgoja tests pass.

### Technical details

Validation output:

```text
ok  github.com/go-go-golems/go-go-goja/cmd/xgoja
ok  github.com/go-go-golems/go-goja/cmd/xgoja/internal/generate
ok  github.com/go-go-golems/go-goja/pkg/xgoja/app
?   github.com/go-go-golems/go-goja/pkg/xgoja/testprovider [no test files]
```

## Step 3: Remove obsolete jsverbs bare-Goja invocation API

This step removed the transitional `Registry.InvokeInGojaRuntime` API. That method existed so generated xgoja binaries could invoke jsverbs before xgoja reused `engine.Runtime`. After XGOJA-003, the generated app runtime is engine-backed and the canonical path is `Registry.InvokeInRuntime(ctx, runtime, verb, values)`.

The cleanup leaves one jsverbs host invocation API for caller-owned managed runtimes. It also removes the direct-runtime test that existed only to exercise the deleted bare-Goja path.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Continue the cleanup by removing transition-only jsverbs API rather than keeping a compatibility wrapper.

**Inferred user intent:** Make `engine.Runtime` the unambiguous runtime abstraction for jsverbs hosts.

**Commit (code):** Pending for this step.

### What I did

- Removed from `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbs/runtime.go`:
  - exported `Registry.InvokeInGojaRuntime(...)`,
  - private `invokeInGojaRuntime(...)`,
  - private `waitForPromiseWithOwner(...)`,
  - private `waitForPromiseDirect(...)`,
  - now-unused `runtimebridge` import.
- Deleted `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbs/runtime_direct_test.go`.
- Ran `gofmt`.
- Verified no active docs/code still reference the deleted names with `rg`.
- Marked task 2 complete.
- Updated the changelog.
- Validated with:

```bash
GOWORK=off go test ./pkg/jsverbs ./pkg/jsverbscli ./pkg/xgoja/app ./cmd/xgoja/internal/generate -count=1
```

### Why

- `InvokeInGojaRuntime` encoded the old lightweight runtime boundary.
- Current xgoja runtime construction uses `engine.Runtime`, which includes owner scheduling, runtimebridge bindings, lifecycle context, and close semantics.
- Keeping both exported invocation APIs would make new hosts wonder whether they should construct raw Goja runtimes or managed engine runtimes.

### What worked

- Focused jsverbs/xgoja tests passed after deleting the API and test.
- Active code already used `InvokeInRuntime`, so no call sites needed migration.

### What didn't work

- N/A

### What I learned

- The bare-Goja invocation code was fully isolated in `pkg/jsverbs/runtime.go` and its direct test.
- Public docs did not contain active `InvokeInGojaRuntime` examples, so this tranche did not require public doc rewrites.

### What was tricky to build

- The only subtlety was preserving promise handling for `InvokeInRuntime`. The deleted helper functions had similar polling logic, but `waitForPromise(ctx, runtime, promise)` remains and uses `runtime.Owner.Call`, which is the canonical managed-runtime path.

### What warrants a second pair of eyes

- Check that no downstream package in this repository still needs raw-Goja jsverbs invocation. This was intentionally a hard cutover.

### What should be done in the future

- Remove deprecated engine default-registry wrappers and rewrite docs/tests to middleware-based module selection.

### Code review instructions

- Start in `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbs/runtime.go`.
- Confirm `InvokeInRuntime` still uses `runtime.Owner.Call` and `waitForPromise`.
- Confirm `pkg/jsverbs/runtime_direct_test.go` is deleted because it tested only the removed API.

### Technical details

Validation output:

```text
ok  github.com/go-go-golems/go-go-goja/pkg/jsverbs
ok  github.com/go-go-golems/go-go-goja/pkg/jsverbscli
ok  github.com/go-go-golems/go-go-goja/pkg/xgoja/app
ok  github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate
```

## Step 4: Remove deprecated engine default-registry wrappers

This step completed the hard cutover for module-selection APIs. The exported `engine.DefaultRegistryModules`, `engine.DefaultRegistryModule`, `engine.DefaultRegistryModulesNamed`, and `engine.DataOnlyDefaultRegistryModules` helpers were removed. The engine still has private helper specs internally, but public code now selects default-registry modules through builder middleware.

I also removed `modules.EnableAll`, because its only active role was to support the deprecated all-default wrapper. Public docs and tests now point at `UseModuleMiddleware(engine.MiddlewareSafe())`, `UseModuleMiddleware(engine.MiddlewareOnly(...))`, or a plain `engine.NewBuilder().Build()` for the all-default trusted-runtime case.

### Prompt Context

**User prompt (verbatim):** "hard cutover btw, continue"

**Assistant interpretation:** Do not leave compatibility wrappers for the cleanup targets; remove the deprecated public APIs and update callers/docs immediately.

**Inferred user intent:** Ensure the cleanup actually simplifies the API surface instead of preserving old names as aliases.

**Commit (code):** Pending for this step.

### What I did

- Removed exported deprecated wrappers from `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/module_specs.go`:
  - `DefaultRegistryModules()`
  - `DefaultRegistryModule(name string)`
  - `DefaultRegistryModulesNamed(names ...string)`
  - `DataOnlyDefaultRegistryModules()`
- Added/kept private internal helpers:
  - `defaultRegistryModule(name string)`
  - `defaultRegistryModulesNamed(names ...string)`
  - `dataOnlyDefaultRegistryModules()`
- Updated `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/factory.go` to use the private helpers.
- Removed `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/modules/common.go` package-level `EnableAll` helper.
- Updated `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/granular_modules_test.go` so direct wrapper tests became middleware-selection tests.
- Updated public docs in:
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/README.md`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/doc/bun-goja-bundling-playbook.md`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/doc/11-jsverbs-example-reference.md`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/doc/16-nodejs-primitives.md`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/doc/16-yaml-module.md`
  - `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/doc/17-connected-eventemitters-developer-guide.md`
- Marked tasks 3, 4, and 5 complete.
- Ran focused engine/module validation, full focused xgoja/runtime validation, and all xgoja example smokes.

### Why

- Middleware is now the canonical public API for selecting default-registry modules.
- Deprecated wrappers encouraged old composition examples and duplicated the new middleware vocabulary.
- `modules.EnableAll` was a compatibility helper with no active caller after the wrapper removal.

### What worked

- `GOWORK=off go test ./engine ./modules/... -count=1` passed.
- The full focused xgoja/runtime suite passed.
- All three xgoja example smokes passed.
- A search for removed public helper calls no longer reports active docs/code references; remaining matches are private helper names or unrelated option names.

### What didn't work

- A mechanical documentation replacement initially produced `Build().Build()` in one bundling playbook bullet. I corrected that bullet to `engine.NewBuilder().WithRequireOptions(require.WithLoader(loader)).Build()`.
- The same replacement made a README bullet say "The old `Build()` is deprecated". I corrected it to explain that omitting middleware loads all default-registry modules.
- The first commit attempt failed in the pre-commit hook because historical `ttmp/.../scripts` Go packages still called `engine.DefaultRegistryModules()`:

```text
ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/04-engine-runtimeowner-interrupt-sync-loop/main.go:16:57: undefined: engine.DefaultRegistryModules
ttmp/2026/04/20/GOJA-044-PR28-REPL-SERVICE-REVIEW--pr-28-review-repl-service-architecture-bug-report-and-regression-analysis/scripts/exp04_lowlevel.go:21:57: undefined: engine.DefaultRegistryModules
ttmp/2026/04/20/GOJA-050--fuzzing-go-go-goja-replapi-analysis-design-and-implementation-guide/scripts/01-basic-replapi-fuzz/main.go:32:57: undefined: engine.DefaultRegistryModules
```

  I updated those historical script packages to use the plain `engine.NewBuilder().Build()` all-default path.

### What I learned

- Most active code had already moved to middleware after XGOJA-003; the remaining references were mostly docs and tests written to exercise the deprecated helpers directly.
- The internal engine still benefits from small private RuntimeModuleSpec helpers because middleware resolves selected module names before runtime creation.

### What was tricky to build

- The hard cutover had to remove exported wrappers while preserving engine behavior. A plain `engine.NewBuilder().Build()` still needs to expose all default-registry modules, and every runtime still needs the data-only default modules unless explicitly disabled. The solution was to keep private helper specs and update only internal factory construction to call those helpers.

### What warrants a second pair of eyes

- Review the public docs for wording around trusted all-default runtimes versus restricted middleware-based runtimes.
- Review the public API break in `engine/module_specs.go`; this is intentional but should be called out if downstream code consumes the deprecated helpers.

### What should be done in the future

- Run `docmgr doctor` and close XGOJA-004 if no follow-up cleanup remains.

### Code review instructions

- Start in `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/module_specs.go` to confirm only private default-registry helper functions remain.
- Then review `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/factory.go` for unchanged runtime construction behavior.
- Review `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/engine/granular_modules_test.go` for middleware-based coverage.
- Validate with:

```bash
GOWORK=off go test ./engine ./modules/... -count=1
GOWORK=off go test ./engine ./pkg/runtimebridge ./pkg/jsverbs ./pkg/xgoja/app ./cmd/xgoja/internal/generate ./cmd/xgoja ./cmd/xgoja/internal/buildspec ./pkg/xgoja/providerapi ./pkg/xgoja/testprovider ./pkg/xgoja/testcobra ./pkg/xgoja/testadapter ./modules/express ./modules/uidsl ./pkg/hashiplugin/host ./pkg/repl/evaluators/javascript ./pkg/docaccess/runtime ./pkg/jsverbscli ./pkg/gojahttp ./pkg/doc -count=1
for dir in runtime-filesystem embedded-jsverbs provider-shipped-jsverbs; do make -C examples/xgoja/$dir smoke; done
```

### Technical details

The remaining search hits for `DefaultRegistryModule` are private implementation names or option names:

```text
engine/module_specs.go: dataOnlyDefaultRegistryModuleNames, defaultRegistryModule, defaultRegistryModulesNamed
engine/options.go: WithImplicitDefaultRegistryModules, WithDataOnlyDefaultRegistryModules
engine/factory.go: dataOnlyDefaultRegistryModules()
```
