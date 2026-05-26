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

