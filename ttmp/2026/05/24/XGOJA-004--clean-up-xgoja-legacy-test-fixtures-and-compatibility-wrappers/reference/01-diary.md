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
