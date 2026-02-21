# Changelog

## 2026-02-20

- Initial workspace created


## 2026-02-20

Added a detailed architecture investigation document for EngineFactory module composition (WithModules), including lifecycle model, ordering/dependency solver requirements, typing tradeoffs, conflict policy, and phased migration from global registries.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/design/01-enginefactory-withmodules-architecture-deep-dive.md — Primary analysis and design brainstorming artifact

## 2026-02-20

Added follow-up backlog tasks and enriched ticket index metadata/overview so the EngineFactory module-composition design can be resumed later with minimal context loss.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/index.md — Added summary
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/tasks.md — Seeded concrete implementation backlog for future execution

## 2026-02-21

Replaced the research-oriented task list with an implementation sequence for a no-backward-compatibility rewrite, explicitly excluding dependency solver work from this pass. Added a new diary document and started step-by-step execution logging.

### Related Files

- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/tasks.md — Detailed execution tasks for clean API cutover
- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/reference/01-diary.md — Implementation diary initialized

## 2026-02-21

Step 2: landed no-compat builder/factory/runtime rewrite and migrated callsites (commit 4059db5).

### Related Files

- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/engine/factory.go — Canonical builder/factory runtime construction and initialization sequencing
- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/engine/module_specs.go — ModuleSpec and RuntimeInitializer contracts with explicit default registry module
- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go — REPL evaluator migrated to owned runtime lifecycle and reset handling


## 2026-02-21

Step 3: closed remaining docs checklist item after per-step diary/changelog alignment.

### Related Files

- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/reference/01-diary.md — Added closure step for execution checklist
- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/tasks.md — Marked final docs execution task complete


## 2026-02-21

Ticket closed


## 2026-02-21

Step 4: refreshed ticket index metadata and overview to reflect completed implementation state and v2 design links.

### Related Files

- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/index.md — Updated summary/status/key links and related-files metadata for completed ticket state
- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/reference/01-diary.md — Added Step 4 documentation consistency audit and remediation record


## 2026-02-21

Step 5: refreshed root README for canonical builder/factory/runtime API and removed legacy wrapper guidance (commit 998a03b).

### Related Files

- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/README.md — Updated runtime API docs to NewBuilder/Build/NewRuntime/Close model
- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/reference/01-diary.md — Added Step 5 entry for README refresh


## 2026-02-21

Step 6: fixed evaluator owned-runtime cleanup on init errors and made reset non-destructive by swapping before close (commit 993fbfe).

### Related Files

- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go — Lifecycle hardening for constructor failure cleanup and reset ordering
- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/reference/01-diary.md — Added Step 6 review-response entry


## 2026-02-21

Step 7: added explicit evaluator Close lifecycle and wired teardown at REPL call sites (commit 7a842da).

### Related Files

- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/cmd/js-repl/main.go — Deferred evaluator close on REPL shutdown
- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/cmd/smalltalk-inspector/app/model.go — Model.Close now disposes assist evaluator
- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/cmd/smalltalk-inspector/app/repl_widgets.go — Close previous assist evaluator before replacement
- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/cmd/smalltalk-inspector/main.go — Close final model resources after tea program run
- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/pkg/repl/adapters/bobatea/javascript.go — Adapter Close() forwards teardown to core evaluator
- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go — Added evaluator Close() to release owned runtime resources
- /home/manuel/workspaces/2026-02-21/entity-extraction-js/go-go-goja/ttmp/2026/02/20/GC-05-ENGINE-MODULE-COMPOSITION--enginefactory-module-composition-and-registration-lifecycle/reference/01-diary.md — Added Step 7 lifecycle teardown documentation

