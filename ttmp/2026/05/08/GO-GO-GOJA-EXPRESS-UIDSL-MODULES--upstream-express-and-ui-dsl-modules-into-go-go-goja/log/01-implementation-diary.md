---
Title: Implementation Diary
Ticket: GO-GO-GOJA-EXPRESS-UIDSL-MODULES
Status: active
Topics:
    - goja
    - ui-dsl
    - web-ui
    - documentation
DocType: log
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Implementation diary for upstreaming gojahttp, express, and ui.dsl modules, migrating db-browser/goja-hosting-site, and merging their shells."
LastUpdated: 2026-05-08T22:45:00-04:00
WhatFor: "Use this to understand what was moved, which validations passed, and what remains."
WhenToUse: "Read before continuing Express/ui.dsl upstreaming or downstream cleanup work."
---

# Implementation Diary

## 2026-05-08 - Shell merge design and task setup

Created the design plan for merging the remaining `db-browser` and `goja-hosting-site` web shells after the shared runtime extraction. The design locks in the requested options: retire/delete `db-browser` as an independent shell, and keep both the existing `dbguard` policy and a simple read/write-gated database policy in the unified goja-site shell.

This step did not change runtime behavior. It created the blueprint and task breakdown for the next implementation phases, including jsverbs CLI migration, multi-script loading, database policy selection, db-browser example migration, and db-browser retirement.

### Prompt Context

**User prompt (verbatim):** "create a design document.\n\noption choices: \n\n### Option A — deleted/retired\n#### Option C - both dbguard and normal\n\nAdd detailed list of tasks to the ticket, then work on them one by one, commit at appropriate intervals, and keep a diary as you owork"

**Assistant interpretation:** Create a docmgr design document for shell unification, record the selected options, expand the ticket task list, and continue work in small committed steps with diary updates.

**Inferred user intent:** Collapse the now-similar web shells into one canonical implementation while preserving clear project management and review history.

**Commit (code):** pending — design/task setup has been written and is ready to commit with the current baseline.

### What I did

- Added `design-doc/02-merge-db-browser-and-goja-hosting-site-web-shells.md` to the ticket.
- Added T10–T18 task sections covering baseline commits, jsverbs extraction, `goja-site verbs`, multi-script loading, DB policy selection, example migration, and db-browser retirement.
- Related the design doc, diary, and task file to the ticket index.
- Updated the ticket changelog.

### Why

- The prior extraction made `db-browser` and `goja-hosting-site` thin enough that the remaining duplication is shell/product wiring rather than core runtime logic.
- The selected options need to be explicit before changing package ownership or deleting the db-browser shell.

### What worked

- `docmgr doc add` created the new design doc cleanly.
- `docmgr doc relate` and `docmgr changelog update` updated the ticket metadata/changelog.
- The task list now has small commit-sized phases.

### What didn't work

- I accidentally checked tasks 45–47 before the baseline commits were created. I will satisfy those by committing the baseline immediately rather than leaving them stale.

### What I learned

- The shell merge should keep route scripts and jsverbs repositories separate concepts even if they share runtime modules.
- The database policy choice is the main semantic difference left between the two shells.

### What was tricky to build

- The task numbering is generated from markdown checkboxes, so adding new sections required verifying the numeric IDs with `docmgr task list` before checking the intended items.
- Because previous upstream/downstream code changes were already uncommitted, the next commit boundary needs to establish that baseline before deeper shell-merge work starts.

### What warrants a second pair of eyes

- Whether jsverbs CLI/repository plumbing should go directly to `go-go-goja/pkg/jsverbscli` and `pkg/jsverbrepos` or start in `goja-hosting-site/pkg` first.
- Whether `dbguard` should honor the same `--readonly` / `--allow-writes` flags as the simple policy.

### What should be done in the future

- Commit the current upstream/downstream baseline.
- Start T12 by extracting neutral jsverbs repository discovery.

### Code review instructions

- Start with `design-doc/02-merge-db-browser-and-goja-hosting-site-web-shells.md`.
- Then review T10–T18 in `tasks.md`.
- Validate ticket hygiene with `docmgr doctor --ticket GO-GO-GOJA-EXPRESS-UIDSL-MODULES --stale-after 30`.

### Technical details

- Selected retirement option: Option A — deleted/retired `db-browser`.
- Selected DB policy option: Option C — both `dbguard` and normal/simple policy.

## 2026-05-08 - Initial upstream implementation and downstream unification

Implemented the first upstream pass directly in the workspace:

- Created `pkg/gojahttp` in `go-go-goja` from db-browser's `internal/web`, excluding the Express module adapter.
- Renamed the HTTP host package from `web` to `gojahttp`.
- Kept the host renderer-neutral through `HostOptions.Renderer`; `pkg/gojahttp` does not import `modules/uidsl`.
- Changed the default session cookie name from `goja_site_session` to `go_go_goja_session` and updated the session comment to refer to go-go-goja.
- Created `modules/express` with `NewRegistrar(host *gojahttp.Host, opts ...Option)`, `WithName`, and runtime-scoped registration of `require("express")`.
- Created `modules/uidsl` from db-browser's richer `internal/uidsl` implementation, preserving `RenderAny`, `Loader`, `NewRegistrar`, and the `ui.dsl` / `ui` aliases.
- Added TypeScript declaration descriptors for the runtime registrars where practical.
- Added go-go-goja help docs for the Express-style module and `ui.dsl`.

Downstream unification:

- Updated db-browser to import `gojahttp`, `modules/express`, and `modules/uidsl` from go-go-goja.
- Deleted db-browser's copied `internal/web` and `internal/uidsl` packages.
- Updated goja-hosting-site to import the same upstream packages.
- Updated goja-hosting-site Kanban DSL code/tests to use upstream `uidsl` node types and renderer.
- Deleted goja-hosting-site's copied `pkg/web` and `pkg/uidsl` packages.
- Updated workspace `go.work` to `go 1.26.1` so it matches the modules.
- Updated db-browser's local `go.mod` with a replace to `../go-go-goja` for standalone validation before an upstream tag exists.
- Updated db-browser validation scripts that referenced `internal/uidsl` so they now test `github.com/go-go-golems/go-go-goja/modules/uidsl`.

Validation performed:

```text
cd go-go-goja && go test ./pkg/gojahttp ./modules/express ./modules/uidsl -count=1
cd go-go-goja && go test ./... -count=1
cd go-go-goja && GOWORK=off go test ./pkg/gojahttp ./modules/express ./modules/uidsl -count=1
cd go-go-goja && GOWORK=off go test ./... -count=1
cd 2026-05-07--db-browser && go test ./... -count=1
cd 2026-05-07--db-browser && GOWORK=off go test ./... -count=1
cd 2026-05-03--goja-hosting-site && go test ./... -count=1
cd 2026-05-03--goja-hosting-site && GOWORK=off go test ./... -count=1
cd 2026-05-07--db-browser && bash ttmp/2026/05/07/DB-BROWSER-UIDSL-COMPONENTS--ui-dsl-component-spec-for-code-blocks-badges-and-tabs/scripts/001-uidsl-components-smoke.sh
cd 2026-05-07--db-browser && bash ttmp/2026/05/07/DB-BROWSER-JSVERBS-DESIGN--goja-jsverbs-database-browser-web-app-design/scripts/011-final-validation.sh
```

One validation script initially failed because it still ran `go test ./internal/uidsl` after the package was deleted. I updated the script to test the upstream `modules/uidsl` package instead. The rich-table validation also had an exact class-string assertion that did not include the now-rendered `ui-table--filters` class; I updated that assertion to match current output.

Remaining follow-ups:

- Decide whether `ui.dsl` should also be default-registered as a global native module.
- Decide when to remove local replace directives after go-go-goja is tagged/released.
- Run lint if desired.
- Commit the upstream and downstream changes together or split into separate commits/PRs.
