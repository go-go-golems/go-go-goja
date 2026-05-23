---
Title: Diary
Ticket: XGOJA-003
Status: active
Topics:
    - xgoja
    - goja
    - engine
    - runtime
    - modules
    - refactor
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/05/23/XGOJA-003--unify-engine-runtime-module-registration/design-doc/01-runtime-aware-module-registration-design-and-implementation-guide.md
      Note: Primary design and migration guide for the hard cutover
ExternalSources: []
Summary: Implementation diary for unifying engine module registration around one runtime-aware RegisterRuntimeModule contract and adapting xgoja to use engine runtimes safely.
LastUpdated: 2026-05-23T22:10:00-04:00
WhatFor: Use this diary to resume or review XGOJA-003 implementation work.
WhenToUse: Read before changing engine module registration APIs, xgoja runtime construction, or jsverbs invocation paths.
---

# Diary

## Goal

This diary records the hard cutover from split engine module registration APIs to one runtime-aware module API, plus the xgoja adaptation that should let generated apps use `engine.Runtime` safely from xgoja specs.

## Step 1: Create the runtime-aware module registration ticket and design

This step created the follow-up ticket for the engine/xgoja runtime abstraction work. The previous review discussion showed that xgoja's separate lightweight runtime existed partly because `engine.ModuleSpec` could not receive runtime context. The new design replaces the split between static modules and runtime registrars with one runtime-aware `RegisterRuntimeModule` contract.

The design is intentionally a hard cutover. It does not preserve compatibility wrappers or legacy builder methods. That keeps the API smaller and makes each module registration path explicit: if a value registers modules into a runtime, it receives `RuntimeModuleContext`.

### Prompt Context

**User prompt (verbatim):** "Ok, create a ticket for that, hard cutover, add small migration guide. Add design + implementation guide, then tasks to the ticket, and implement them one by one, keeping a diary as you work, committing at appropriat eintervals. No backwards compatibility or legacy wrappers or so."

**Assistant interpretation:** Create a new docmgr ticket for the hard cutover to a runtime-aware engine module registration API, write a design/migration guide, add tasks, then implement incrementally with commits and diary updates.

**Inferred user intent:** Reduce xgoja/engine runtime duplication by making engine module registration always runtime-aware, without carrying old APIs forward.

### What I did

- Created ticket `XGOJA-003 — Unify engine runtime module registration`.
- Added design doc `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/23/XGOJA-003--unify-engine-runtime-module-registration/design-doc/01-runtime-aware-module-registration-design-and-implementation-guide.md`.
- Added this diary.
- Added implementation tasks for API cutover, built-in module updates, call-site migration, xgoja runtime adaptation, and validation.

### Why

- The existing split between `ModuleSpec.Register(reg)` and `RuntimeModuleRegistrar.RegisterRuntimeModules(ctx, reg)` makes runtime-aware modules a special case.
- xgoja provider modules naturally need runtime context and config. A unified runtime-aware engine module API should let xgoja use `engine.Runtime` without losing spec-selected module semantics.

### What worked

- The ticket, design doc, diary, and tasks were created.
- The design identifies the hard-cutover migration path and the validation set before implementation starts.

### What didn't work

- The generated task list initially included the default placeholder task `Add tasks here`. This should be removed or marked obsolete during task cleanup.

### What I learned

- The cleanest API is not an additional wrapper structure. It is replacing both existing module registration concepts with one runtime-aware module concept.

### What was tricky to build

- The design has to preserve engine's current default builder behavior while allowing xgoja to avoid default module exposure. The distinction is in how the builder is used: a plain engine builder can still select default modules, while xgoja must always pass explicit spec-selected modules.

### What warrants a second pair of eyes

- Review whether `RuntimeModuleSpec` should replace every old registrar name in one commit or whether a mechanical rename commit should be separated from xgoja runtime adaptation.
- Review whether `RuntimeInitializer` should remain separate. The current design keeps it separate because it runs after `require` is enabled.

### What should be done in the future

- Implement tasks in order and commit after focused phases.

### Code review instructions

- Start with the design guide before reviewing code changes.
- During implementation, review engine API changes before xgoja adaptation.

### Technical details

Target interface:

```go
type RuntimeModuleSpec interface {
    ID() string
    RegisterRuntimeModule(ctx *RuntimeModuleContext, reg *require.Registry) error
}
```
