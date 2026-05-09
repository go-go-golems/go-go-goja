---
Title: Merge db-browser and goja-hosting-site web shells
Ticket: GO-GO-GOJA-EXPRESS-UIDSL-MODULES
Status: active
Topics:
  - goja
  - ui-dsl
  - web-ui
  - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
  - path: /home/manuel/workspaces/2026-05-08/extract-express-goja/2026-05-03--goja-hosting-site/pkg/app/server.go
    note: Current goja-site single-site runtime/server shell to become the canonical web shell.
  - path: /home/manuel/workspaces/2026-05-08/extract-express-goja/2026-05-03--goja-hosting-site/pkg/app/multi.go
    note: Current goja-site multi-site server support to keep in the unified shell.
  - path: /home/manuel/workspaces/2026-05-08/extract-express-goja/2026-05-07--db-browser/internal/app/server.go
    note: Current db-browser runtime/server shell to retire after its unique behavior is migrated.
  - path: /home/manuel/workspaces/2026-05-08/extract-express-goja/2026-05-07--db-browser/internal/verbcli
    note: db-browser jsverbs CLI layer to move into the canonical shell or upstream reusable package.
  - path: /home/manuel/workspaces/2026-05-08/extract-express-goja/2026-05-07--db-browser/internal/verbrepos
    note: db-browser verb repository discovery and built-in verbs to migrate.
  - path: /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/pkg/jsverbs
    note: Existing reusable jsverbs scanner/registry/invocation core.
ExternalSources: []
Summary: "Design for making goja-hosting-site the canonical web/jsverbs shell, migrating db-browser's unique jsverbs and SQLite-browser assets into it, and retiring db-browser."
LastUpdated: 2026-05-08T22:40:00-04:00
WhatFor: "Use this before merging db-browser and goja-hosting-site after express/ui.dsl/gojahttp were upstreamed."
WhenToUse: "Read when implementing goja-site verbs, multi-script support, unified database policy, db-browser example migration, and db-browser retirement."
---

# Merge db-browser and goja-hosting-site web shells

## Executive Summary

After upstreaming the shared HTTP host, Express-style module, and `ui.dsl` module into `go-go-goja`, `db-browser` and `goja-hosting-site` are no longer meaningfully separate runtime stacks. Both now create a Goja runtime, configure SQLite access, register `express` and `ui.dsl`, load trusted JavaScript, and serve HTTP routes registered by those scripts.

The remaining difference is product shape:

- `goja-hosting-site` is the richer web shell: single-site serving, multi-site Host dispatch, `kanban.dsl`, `dbguard`, deployment config, and site examples.
- `db-browser` is the SQLite/jsverbs shell: database-browser examples, `verbs` CLI, repository discovery, built-in jsverbs, and a simple read/write-gated DB wrapper.

The chosen direction is **Option A — delete/retire `db-browser` as an independent shell** after moving its unique behavior into `goja-hosting-site` or reusable `go-go-goja` packages. For database policy, choose **Option C — support both `dbguard` and a normal/simple read-write guard**, so goja-site can serve the current site examples and the db-browser-style generic SQLite tools without forcing one policy onto all apps.

The final target is:

```text
go-go-goja/
  pkg/gojahttp
  modules/express
  modules/uidsl
  modules/database
  pkg/jsverbs
  pkg/jsverbscli        # optional reusable extraction target
  pkg/jsverbrepos       # optional reusable extraction target

goja-hosting-site/
  cmd/goja-site
    serve
    serve-multi
    verbs              # migrated db-browser CLI
    inspect            # optional db-browser inspect equivalent
  pkg/app              # canonical web shell
  pkg/dbguard
  pkg/kanbanddsl
  examples/db-browser  # migrated db-browser examples
  examples/verbs       # migrated built-in verb examples
  sites/*

db-browser/
  retired/deleted, or temporarily left as a tiny compatibility wrapper only if needed
```

## Problem Statement

Before the upstream extraction, `db-browser` and `goja-hosting-site` each carried local copies of the HTTP host and UI DSL. That duplication justified separate shells because each repo owned critical runtime pieces. That is no longer true.

Current duplication now lives at the shell level:

- both shells open SQLite databases;
- both construct Goja runtimes;
- both register upstream `express` and `ui.dsl`;
- both load JavaScript route files from a scripts directory;
- both expose database modules to JavaScript;
- both have examples that are just trusted JavaScript plus a database.

Keeping both as full applications would create renewed drift in CLI flags, database policy, runtime options, docs, and examples. The better result is one canonical web/jsverbs shell and one reusable upstream runtime library.

## Chosen Options

### Option A — delete/retire `db-browser`

`db-browser` should not remain a separate full shell once its unique jsverbs and database-browser examples are migrated. The name can remain temporarily only as a compatibility wrapper if users/scripts need it, but the implementation should live in `goja-hosting-site` and/or `go-go-goja`.

Rationale:

- avoids two commands that both mean "serve trusted Goja web scripts with SQLite";
- avoids two database policy implementations drifting apart;
- makes goja-site examples the canonical place for small web apps;
- lets `goja-site verbs` cover the current db-browser CLI workflow;
- reduces maintenance surface after the shared runtime pieces moved upstream.

### Option C — support both `dbguard` and normal/simple DB policy

The unified shell should support two database modes:

1. **normal/simple DB policy** for db-browser-style tools:
   - preconfigured `database` and `db` modules;
   - read-only by default;
   - writes allowed only when `--readonly=false --allow-writes=true`;
   - easy to understand and suitable for generic SQLite inspection.

2. **`dbguard` policy** for goja-hosting-site apps:
   - metered/guarded DB wrapper;
   - `db.guard` module available to JavaScript;
   - existing site/kanban behavior preserved.

The shell should make this explicit, probably through a flag such as:

```bash
goja-site serve --db-policy simple
goja-site serve --db-policy guarded
```

Default can remain `guarded` for existing goja-site compatibility, while db-browser examples can document `--db-policy simple --readonly`.

## Proposed User-Facing CLI

### Serve one site

```bash
goja-site serve \
  --addr :8080 \
  --db app.db \
  --scripts sites/pizza/scripts \
  --scripts shared/scripts \
  --db-policy guarded \
  --dev
```

`--scripts` should become repeatable. Loading order is deterministic:

1. script directories in CLI/config order;
2. files sorted lexicographically within each directory.

Keep `--scripts-dir` as an alias only if needed for a short migration window. If not needed, do not add it.

### Serve multiple sites

```bash
goja-site serve-multi --config deploy/sites.yaml
```

Multi-site should keep working, and the site config should eventually gain the same fields:

```yaml
sites:
  - host: pizza.localhost
    db: sites/pizza/app.db
    scripts:
      - sites/pizza/scripts
      - shared/scripts
    dbPolicy: guarded
```

### Run jsverbs

```bash
goja-site verbs list

goja-site verbs \
  --repository examples/verbs \
  --db app.db \
  examples tables
```

The `verbs` command should expose:

- `fs`, `path`, `time`, `timer`, `yaml`;
- `ui.dsl` / `ui`;
- `database` / `db` when `--db` is set;
- simple read/write policy by default;
- optionally `db.guard` and `kanban.dsl` if the shell config requests them.

## Proposed Package Layout

### Preferred clean layout

Move reusable jsverbs CLI/repository plumbing to `go-go-goja`:

```text
go-go-goja/pkg/jsverbscli
  command.go
  list.go
  runtime.go

go-go-goja/pkg/jsverbrepos
  bootstrap.go
  builtin/*.js
```

Then wire those packages into `goja-hosting-site/cmd/goja-site`.

### Fast local layout

If the migration should avoid upstreaming more packages initially, move db-browser internals into goja-site:

```text
goja-hosting-site/pkg/verbcli
  command.go
  list.go
  runtime.go

goja-hosting-site/pkg/verbrepos
  bootstrap.go
  builtin/*.js
```

This is quicker but less reusable. Since `go-go-goja/pkg/jsverbs` already exists, the preferred layout is to upstream the CLI/repository plumbing as a reusable sibling.

## Runtime Architecture

The unified `pkg/app` should factor runtime construction into smaller pieces:

```text
pkg/app/config.go       public Config, DBPolicy, ScriptSource
pkg/app/database.go     open DB, choose guarded/simple policy, produce module specs/registrars
pkg/app/runtime.go      build engine.Factory with common modules and runtime registrars
pkg/app/scripts.go      deterministic multi-directory script discovery/loading
pkg/app/server.go       single-site HTTP server
pkg/app/multi.go        host-dispatched multi-site server
```

The key invariant is that web serving and jsverbs invocation share one module vocabulary but not necessarily one runtime lifetime:

- web server: one runtime per site, route scripts loaded for side effects;
- jsverbs: one runtime per invocation, verb repository loader installed, selected verb invoked.

## Script Loading Model

Current goja-site loads all `.js` files from one directory. The unified shell should support multiple roots:

```go
type Config struct {
    Addr       string
    DBPath     string
    ScriptDirs []string
    Dev        bool
    DBPolicy   DBPolicy
    ReadOnly   bool
    AllowWrites bool
}
```

For compatibility inside Go code, `ScriptsDir` can remain temporarily only if existing callers still need it. Prefer moving call sites to `ScriptDirs` and deleting `ScriptsDir` if this is a breaking internal cleanup.

Rules:

- missing script directory is an error;
- non-directory path is an error;
- only `.js` files are loaded;
- files are sorted within each directory;
- duplicate absolute file paths are loaded once;
- errors include the file path that failed.

## Database Policy Model

Introduce:

```go
type DBPolicy string

const (
    DBPolicySimple  DBPolicy = "simple"
    DBPolicyGuarded DBPolicy = "guarded"
)
```

Simple policy:

- use a lightweight wrapper similar to db-browser's `guardedDB`;
- expose `database` and `db`;
- reject writes unless both `AllowWrites` and `!ReadOnly` are true.

Guarded policy:

- keep current `dbguard.New`, `dbguard.NewMeteredDB`, and `dbguard.NewRegistrar`;
- expose `database`, `db`, and `db.guard`;
- preserve current goja-site behavior.

Open question for implementation: whether `guarded` should also honor `ReadOnly`/`AllowWrites` directly or whether `dbguard` remains the policy authority. The design preference is for all policies to honor CLI flags, but do not hack this in if `dbguard` requires a proper API change.

## jsverbs Migration Details

The db-browser jsverbs layer contains three separable pieces:

1. repository discovery and config:
   - CLI `--repository` / `--verb-repository`;
   - env var currently `DB_BROWSER_VERB_REPOSITORIES`;
   - config files currently `.db-browser.yml` and `.db-browser.override.yml`;
   - built-in repository embedding.

2. command generation:
   - scan repos with `jsverbs.ScanDir` / `jsverbs.ScanFS`;
   - collect verbs;
   - generate Glazed/Cobra commands;
   - add list command.

3. runtime invocation:
   - create one Goja runtime per verb invocation;
   - install repo require loader;
   - install default modules and `ui.dsl`;
   - optionally install `database` and `db`.

Neutral names should replace db-browser-specific names:

```text
GOJA_VERB_REPOSITORIES
.goja-verbs.yml
.goja-verbs.override.yml
```

Because the chosen retirement path does not require backwards compatibility unless explicitly requested, do not add compatibility aliases by default. If existing scripts need them, add aliases deliberately and document them.

## Example Migration

Move:

```text
2026-05-07--db-browser/examples/generic-browser
2026-05-07--db-browser/examples/yaml-dashboard
2026-05-07--db-browser/examples/playwright-smoke
2026-05-07--db-browser/examples/builtin-verbs
```

To:

```text
2026-05-03--goja-hosting-site/examples/db-browser/generic-browser
2026-05-03--goja-hosting-site/examples/db-browser/yaml-dashboard
2026-05-03--goja-hosting-site/examples/db-browser/playwright-smoke
2026-05-03--goja-hosting-site/examples/verbs/builtin
```

Update docs and smoke scripts to run with `goja-site`.

## Retirement Plan for db-browser

The selected option is deletion/retirement. Practical steps:

1. finish migrating code/examples/docs;
2. validate goja-site covers serve and verbs workflows;
3. leave a short retirement note in db-browser if the repo remains during transition;
4. remove db-browser-specific runtime packages;
5. archive/delete db-browser repo when no scripts depend on it.

If a compatibility wrapper is requested later, keep it tiny and make it delegate to the canonical implementation. Do not keep a second runtime stack.

## Implementation Plan

### Phase 0 — Baseline and docs

- Commit the completed express/ui.dsl/gojahttp upstream extraction and downstream migration baseline.
- Add this design document.
- Add detailed ticket tasks.
- Keep a diary after every implementation step.

### Phase 1 — Reusable jsverbs CLI/repository packages

- Move db-browser `internal/verbrepos` to `go-go-goja/pkg/jsverbrepos` or goja-site `pkg/verbrepos`.
- Move db-browser `internal/verbcli` to `go-go-goja/pkg/jsverbscli` or goja-site `pkg/verbcli`.
- Rename package imports and db-browser-specific identifiers.
- Change env/config names to neutral goja names.
- Preserve built-in smoke verbs under the new package.
- Add focused tests for repository discovery and command construction.

### Phase 2 — Add `goja-site verbs`

- Wire the migrated jsverbs command into `cmd/goja-site/main.go`.
- Ensure verb runtime has `ui.dsl`, `yaml`, common modules, and optional `database`/`db`.
- Validate built-in verbs:
  - hello;
  - yaml keys;
  - table list with a temp SQLite DB;
  - render sample table.

### Phase 3 — Unify web server config

- Add `ScriptDirs []string` to app config.
- Update `serve` CLI to accept repeatable `--scripts`.
- Refactor script discovery/loading into `pkg/app/scripts.go`.
- Keep deterministic load order and good error messages.
- Update single-site tests.

### Phase 4 — Add database policy selection

- Add `DBPolicy` to app config.
- Implement simple read/write-gated policy.
- Keep guarded policy with `dbguard` and `db.guard` registrar.
- Add CLI flags: `--db-policy`, `--readonly`, `--allow-writes`.
- Validate both policies.

### Phase 5 — Migrate db-browser examples and scripts

- Move db-browser examples into goja-site examples.
- Update paths and docs.
- Update smoke scripts to call `goja-site`.
- Validate generic browser and yaml dashboard examples.

### Phase 6 — Retire db-browser

- Remove or archive db-browser runtime code.
- If the repo remains, replace implementation docs with a retirement note pointing to goja-site.
- Remove redundant validation scripts after they are represented in goja-site.

## Detailed Task List

The ticket task list should include the implementation tasks below as T10 onward. The tasks are intentionally small enough to commit after each phase or subphase.

## Validation Plan

Run after each major phase:

```bash
cd go-go-goja && go test ./pkg/jsverbs ./pkg/jsverbscli ./pkg/jsverbrepos -count=1
cd go-go-goja && go test ./... -count=1
cd 2026-05-03--goja-hosting-site && go test ./... -count=1
```

Run after adding `goja-site verbs`:

```bash
go run ./cmd/goja-site verbs list
go run ./cmd/goja-site verbs examples hello --name Manuel
```

Run after example migration:

```bash
go run ./cmd/goja-site serve --db /tmp/test.sqlite --scripts examples/db-browser/generic-browser/scripts --db-policy simple --readonly --addr :18080
```

Use tmux for long-running server smoke tests.

## Risks and Review Notes

### Risk: mixing route scripts and verb scripts

Do not auto-load verb repositories as web route scripts. Keep `--scripts` and `--repository` separate unless an explicit require-loader integration is designed.

### Risk: unclear write policy

The unified shell must make database write behavior obvious. `--readonly` should be the safe default for generic SQLite browser use.

### Risk: config-name churn

Renaming `.db-browser.yml` to `.goja-verbs.yml` is a breaking cleanup. This is acceptable under Option A unless compatibility is explicitly required.

### Risk: package placement churn

Moving `verbcli`/`verbrepos` directly into goja-site is fastest, but moving to go-go-goja is cleaner. Prefer upstream if the implementation is not blocked by dependency cycles.

## Alternatives Considered

### Keep both shells

Rejected. The shared runtime has moved upstream, so separate shells would mostly duplicate CLI and app bootstrapping.

### Make db-browser the canonical shell

Rejected. goja-hosting-site already has multi-site support, deployment config, site examples, and `kanban.dsl`; it is the broader shell.

### Keep only `dbguard`

Rejected. db-browser-style generic SQLite inspection benefits from a simple, transparent read/write guard and should not require dbguard semantics.

### Keep only simple DB policy

Rejected. goja-hosting-site already uses `dbguard`, and existing site behavior should remain available.

## Open Questions

1. Should reusable `jsverbscli` and `jsverbrepos` live in go-go-goja immediately, or start in goja-site and upstream later?
2. Should `kanban.dsl` be available in CLI jsverbs runtimes by default?
3. Should multi-site config support verb repositories per site, or should `verbs` remain a process-level CLI concern?
4. Should `db-browser` be deleted in this workspace or left with a retirement README until external scripts are updated?
