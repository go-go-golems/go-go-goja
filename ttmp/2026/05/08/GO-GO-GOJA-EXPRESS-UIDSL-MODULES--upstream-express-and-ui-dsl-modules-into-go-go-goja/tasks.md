# Tasks

## GO-GO-GOJA-EXPRESS-UIDSL-MODULES

### T01 — Ticket and initial design

- [x] Create docmgr ticket workspace.
- [x] Store initial upstreaming analysis and design guide.
- [x] Add implementation task plan.
- [ ] Commit ticket planning docs.

### T02 — Upstream `pkg/gojahttp`

- [x] In `/home/manuel/code/wesen/corporate-headquarters/go-go-goja`, create `pkg/gojahttp`.
- [x] Copy host infrastructure from db-browser `internal/web`, excluding `express_module.go`.
- [x] Rename package from `web` to `gojahttp`.
- [x] Rename/update session cookie comments and default naming.
- [x] Keep renderer injection and avoid importing `modules/uidsl`.
- [x] Move/adapt route, body, session, request/response, and host tests.
- [x] Run `go test ./pkg/gojahttp -count=1`.

### T03 — Add `modules/express`

- [x] Create `modules/express` in go-go-goja.
- [x] Move/adapt the Express registrar from db-browser `internal/web/express_module.go`.
- [x] Implement `NewRegistrar(host *gojahttp.Host, opts ...Option)`.
- [x] Keep `express` runtime-scoped via `engine.RuntimeModuleRegistrar`.
- [x] Do not default-register `express` with `modules.Register` in the first version.
- [x] Add runtime integration tests using `require("express")` and `httptest`.

### T04 — Add `modules/uidsl`

- [x] Copy db-browser `internal/uidsl` into go-go-goja `modules/uidsl`.
- [x] Keep `RenderAny`, `Loader`, and `NewRegistrar` public.
- [x] Register aliases `ui.dsl` and `ui` through the registrar.
- [x] Move/adapt render, table, filters, links, and component tests.
- [x] Run `go test ./modules/uidsl -count=1`.

### T05 — Documentation and TypeScript declarations

- [x] Add go-go-goja docs for the Express-style module.
- [x] Add go-go-goja docs for the `ui.dsl` module.
- [x] Add TypeScript declarations or declaration descriptors for `express`.
- [x] Add TypeScript declarations or declaration descriptors for `ui.dsl`.
- [x] Document that `express` is Express-style, not full Express-compatible.

### T06 — Upstream validation

- [x] Run `go test ./pkg/gojahttp ./modules/express ./modules/uidsl -count=1`.
- [x] Run `go test ./... -count=1` in go-go-goja.
- [x] Run `GOWORK=off go test ./... -count=1` in go-go-goja.
- [x] Run lint if practical.

### T07 — Migrate goja-hosting-site

- [x] Replace local `pkg/web` and `pkg/uidsl` imports with upstream go-go-goja packages.
- [x] Validate goja-hosting-site tests.
- [x] Delete or deprecate local packages once green.

### T08 — Migrate db-browser

- [x] Replace `internal/web` with `pkg/gojahttp` + `modules/express` imports.
- [x] Replace `internal/uidsl` with `modules/uidsl` imports.
- [x] Validate `go test ./...`.
- [x] Run db-browser final validation and component smoke scripts.
- [x] Delete local copied packages if no local behavior remains.

### T09 — Follow-ups after migration

- [ ] Decide whether `ui.dsl` should become a default registry module.
- [ ] Decide whether scoped table query state should be implemented upstream or first in db-browser.
- [ ] Consider static theme asset support after the upstream packages are stable.
- [ ] Consider server-interactive UI events as a separate project after the host boundary is stable.

### T10 — Commit upstream extraction baseline

- [x] Review and commit go-go-goja upstream packages/docs/docmgr updates.
- [x] Review and commit goja-hosting-site migration to upstream `gojahttp`, `express`, and `uidsl`.
- [x] Review and commit db-browser migration to upstream `gojahttp`, `express`, and `uidsl`.
- [x] Record baseline commit hashes in the implementation diary.

### T11 — Shell merge design and task setup

- [x] Add design document for merging db-browser and goja-hosting-site shells.
- [x] Record selected options: Option A retire/delete db-browser; Option C support both dbguard and normal/simple DB policy.
- [x] Add detailed implementation task list to the ticket.
- [x] Relate design and diary files to the ticket.
- [x] Commit design/task/diary setup.

### T12 — Extract reusable jsverbs repository discovery

- [x] Move db-browser `internal/verbrepos` into reusable package location (`go-go-goja/pkg/jsverbrepos` preferred, or `goja-hosting-site/pkg/verbrepos` if blocked).
- [x] Rename db-browser-specific constants to neutral names (`GOJA_VERB_REPOSITORIES`, `.goja-verbs.yml`, `.goja-verbs.override.yml`).
- [x] Move embedded built-in verb scripts with the package.
- [x] Update repository discovery tests for the new package and neutral names.
- [x] Validate focused tests and commit.
- [x] Update diary and changelog with commit hash.

### T13 — Extract reusable jsverbs CLI/runtime invocation

- [x] Move db-browser `internal/verbcli` into reusable package location (`go-go-goja/pkg/jsverbscli` preferred, or `goja-hosting-site/pkg/verbcli` if blocked).
- [x] Update imports from db-browser `verbrepos` to the new neutral repository package.
- [x] Keep runtime invocation support for common modules, `ui.dsl`, and optional `database`/`db`.
- [x] Add or adapt command/list/runtime tests.
- [x] Validate focused tests and commit.
- [x] Update diary and changelog with commit hash.

### T14 — Add `goja-site verbs`

- [x] Wire the migrated jsverbs CLI command into `cmd/goja-site/main.go`.
- [x] Ensure command help and logging integrate with existing goja-site root command.
- [x] Validate built-in verbs without a DB (`list`, `hello`, `yamlKeys`, `renderSampleTable`).
- [x] Validate DB-backed `tables` verb with a temp SQLite database.
- [x] Commit goja-site verbs integration.
- [x] Update diary and changelog with commit hash.

### T15 — Generalize goja-site script loading

- [x] Add `ScriptDirs []string` to `pkg/app.Config`.
- [x] Refactor script discovery/loading into deterministic multi-directory helper.
- [x] Update `goja-site serve` to accept repeated `--scripts` values, replacing the single-directory-only model.
- [x] Update `serve-multi` config parsing to support script lists per site.
- [x] Add tests for multi-directory load ordering, missing dirs, non-dir paths, and duplicate files.
- [x] Validate and commit.
- [x] Update diary and changelog with commit hash.

### T16 — Add unified database policy selection

- [x] Add `DBPolicy` with `simple` and `guarded` modes to goja-site app config.
- [x] Implement simple read/write-gated DB wrapper for generic SQLite browser use.
- [x] Preserve guarded `dbguard` behavior and `db.guard` module registration.
- [x] Add CLI flags `--db-policy`, `--readonly`, and `--allow-writes` to `goja-site serve`.
- [x] Extend multi-site config with database policy fields.
- [x] Add tests for simple read-only rejection and guarded policy registration.
- [x] Validate and commit.
- [x] Update diary and changelog with commit hash.

### T17 — Migrate db-browser examples into goja-hosting-site

- [x] Move db-browser generic browser example to `goja-hosting-site/examples/db-browser/generic-browser`.
- [x] Move db-browser YAML dashboard example to `goja-hosting-site/examples/db-browser/yaml-dashboard`.
- [x] Move db-browser Playwright smoke example to `goja-hosting-site/examples/db-browser/playwright-smoke`.
- [x] Move/adapt built-in verb examples to `goja-hosting-site/examples/verbs` if they should be user-visible outside the embedded package.
- [x] Update paths in smoke scripts and documentation.
- [x] Validate migrated examples with `goja-site serve` and `goja-site verbs`.
- [x] Commit example migration.
- [x] Update diary and changelog with commit hash.

### T18 — Retire db-browser as an independent shell

- [x] Remove or archive db-browser runtime packages after goja-site covers serve and verbs workflows.
- [x] Add a retirement README or repo-level note pointing users to goja-site, unless the repo is deleted outright.
- [x] Remove redundant db-browser validation scripts after equivalent goja-site scripts exist.
- [x] Run final validation across go-go-goja and goja-hosting-site.
- [x] Commit db-browser retirement.
- [x] Update diary and changelog with commit hash.
