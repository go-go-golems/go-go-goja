---
Title: XGOJA-014 changelog
Ticket: XGOJA-014
Status: complete
Topics:
  - xgoja
  - command-providers
DocType: changelog
Intent: log
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Changelog for XGOJA-014."
LastUpdated: 2026-05-25T22:10:00-04:00
WhatFor: "Track notable implementation and documentation changes."
WhenToUse: "Use for review summaries and ticket closeout."
---

# XGOJA-014 changelog

## 2026-05-25

- Created XGOJA-014 ticket workspace for command providers in `go-minitrace`, `loupedeck`, and `css-visual-diff`.
- Added package-specific implementation guides for all three repositories.
- Added an initial task plan and implementation diary.
- Implemented `go-minitrace` command provider `go-minitrace.queries` with catalog command construction, tests, and dependency update to `go-go-goja v0.5.0`.
- Implemented `loupedeck` command provider `loupedeck.scenes` with optional `run`, annotated scene verb commands, construction tests, and dependency update to `go-go-goja v0.5.0`.
- Implemented public `css-visual-diff` xgoja provider with `css-visual-diff`, `diff`, and `report` modules plus `css-visual-diff.verbs` command provider using xgoja `RuntimeFactory`.
- Fixed css-visual-diff local verb repository config discovery to remain robust when git hooks set `GIT_DIR`/`GIT_WORK_TREE`.

- Added generated xgoja command-provider smoke design for all three providers and committed it in `go-go-goja` as `ce1bfb9 docs: design generated command provider smokes`.
- Added `go-minitrace/examples/xgoja/minitrace-command-provider`, whose generated binary mounts `go-minitrace.queries` and runs a JS report verb using `require("minitrace")` plus `require("fs")`; smoke passed and was committed as `4b4dca3 test: add xgoja query command smoke`.
- Upgraded `loupedeck.scenes` annotated verb execution to use xgoja `RuntimeFactory` when available, then added `loupedeck/examples/xgoja/loupedeck-command-provider`, whose generated binary opens an Express route and switches a non-hardware scene after `/deal`; smoke passed and was committed as `33ac9df test: add xgoja loupedeck command smoke`.
- Added `css-visual-diff/examples/xgoja/css-visual-diff-command-provider`, whose generated binary mounts `css-visual-diff.verbs` and writes Markdown/JSON review artifacts with the `report` and `fs` modules; smoke passed and was committed as `daada9e test: add xgoja css diff command smoke`.
- Re-ran focused package validations after the generated smokes; go-minitrace provider/query tests, loupedeck provider/verbs tests, and css-visual-diff provider/jsapi/verbcli/dsl tests passed.

## 2026-05-25

Generated command-provider smokes complete for go-minitrace, loupedeck, and css-visual-diff; all focused validations and doc doctor passed.


## 2026-05-26

Added deep go-go-goja context ownership/runtime package composition guide for future API hardening and Loupedeck deadlock follow-up.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/runtimebridge/runtimebridge.go — Primary API analyzed for context naming and helper proposal
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/runtimeowner/runner.go — Primary implementation analyzed for reentrancy-hardening proposal
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/design-doc/01-goja-context-ownership-and-runtime-package-composition-guide.md — New intern-oriented design and implementation guide


## 2026-05-26

Expanded context ownership guide with linked lifecycle/call contexts, Runner definition, WaitIdle/Interrupt shutdown design, and removed analogies for precise textbook style.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/engine/runtime.go — Proposed Runtime.Close ordering and cleanup semantics
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/runtimeowner/runner.go — Runner semantics
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/design-doc/01-goja-context-ownership-and-runtime-package-composition-guide.md — Updated guide with runtime shutdown and cancellation design


## 2026-05-26

Added full semantic YAML rendition of the context ownership guide and a compact schema spec for designer-facing layout workflows.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/design-doc/01-goja-context-ownership-and-runtime-package-composition-guide.semantic.yaml — Full semantic content export
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/reference/03-semantic-document-yaml-schema-spec.md — Schema specification and usage guide


## 2026-05-26

Updated context ownership guide with explicit WithStartupContext/WithLifetimeContext runtime creation, RuntimeServices naming, no Context compatibility field, explicit Call/Post helper names, and deferred EventSourceContext.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/engine/factory.go — Planned WithStartupContext and WithLifetimeContext runtime options
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/runtimebridge/runtimebridge.go — Planned RuntimeServices and explicit context helper changes
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/design-doc/01-goja-context-ownership-and-runtime-package-composition-guide.md — Updated API decisions and implementation plan


## 2026-05-26

Implemented runtime context API cleanup: RuntimeOwner naming, RuntimeServices naming, explicit startup/lifetime NewRuntime options, migration help entry, and runtimeServices local variable cleanup.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/cmd/xgoja/doc/07-migrating-runtime-context-api.md — Glazed help migration entry
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/engine/factory.go — NewRuntime startup/lifetime option handling
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/runtimebridge/runtimebridge.go — RuntimeServices and explicit context helper API
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/runtimeowner/types.go — RuntimeOwner interface rename
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/reference/01-diary.md — Diary step for runtime context cleanup


## 2026-05-26

Linked runtime service operation contexts to runtime lifetime cancellation, cleaned remaining internal runtime-owner runner naming, and kept runtime services available to closers during shutdown.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/engine/runtime.go — Runs closers before deleting runtime services
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/runtimebridge/runtimebridge.go — Links custom/current operation contexts to runtime lifetime
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/runtimeowner/runner.go — Internal runtime owner naming cleanup
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/reference/01-diary.md — Diary continuation for runtime lifetime linking


## 2026-05-26

Added RuntimeOwner.WaitIdle active-call tracking and runtime Close idle-wait/interrupt behavior for bounded shutdown after lifetime cancellation.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/engine/runtime.go — Close waits for idle and interrupts active JavaScript if needed
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/runtimeowner/runner.go — Tracks active owner calls for WaitIdle
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/runtimeowner/types.go — RuntimeOwner now exposes WaitIdle
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/reference/01-diary.md — Diary updated with WaitIdle shutdown step

