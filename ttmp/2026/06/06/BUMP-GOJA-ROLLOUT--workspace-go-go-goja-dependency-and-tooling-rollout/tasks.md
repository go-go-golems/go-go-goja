# Tasks

## Completed setup tasks

- [x] Create docmgr ticket for the workspace rollout.
- [x] Read infra-tooling Glazed linting rollout playbook.
- [x] Read infra-tooling logcopter rollout instructions.
- [x] Create ticket-local inventory script in `scripts/`.
- [x] Inventory workspace repositories and exclude `glazed`/`go-go-goja`.
- [x] Capture inventory output in `sources/01-workspace-inventory.md`.
- [x] Write implementation guide with rollout phases and migration guidance.
- [x] Start diary and record initial planning step.
- [x] Relate guide, diary, inventory, and source playbooks with `docmgr doc relate`.
- [x] Run `docmgr doctor --ticket BUMP-GOJA-ROLLOUT --stale-after 30` and resolve initial findings.

## Phase 0: Baseline, hygiene, and sequencing

- [ ] Refresh inventory with `scripts/01-inventory-workspace.py > sources/01-workspace-inventory.md` before code changes begin.
- [ ] Verify every target repository has a clean or intentionally recorded `git status --short`.
- [ ] Triage `cozodb-goja` and decide whether it is in scope despite the `github.com/go-go-golems/XXX` signal.
- [ ] Build a dependency ordering table from each repo's direct `github.com/go-go-golems/...` dependencies.
- [ ] Identify upstream repositories that must be merged/released before downstream repos can validate with `GOWORK=off`.
- [ ] Decide branch naming convention for per-repository PRs, for example `task/bump-goja-tooling`.
- [ ] Add a diary entry that records final repository ordering and any explicitly excluded repos.

## Phase 1: Add or verify release-train dependency bump targets

- [ ] For `go-go-app-inventory`, add/verify `bump-go-go-golems` Makefile target.
- [ ] For `go-go-gepa`, add/verify `bump-go-go-golems` Makefile target.
- [ ] For `go-go-host`, add/verify `bump-go-go-golems` Makefile target.
- [ ] For `go-go-os-backend`, add/verify `bump-go-go-golems` Makefile target.
- [ ] For `goja-github-actions`, add/verify `bump-go-go-golems` Makefile target.
- [ ] For `jesus`, add/verify `bump-go-go-golems` Makefile target.
- [ ] For `js-analyzer`, add/verify `bump-go-go-golems` Makefile target.
- [ ] For `plz-confirm`, add/verify `bump-go-go-golems` Makefile target.
- [ ] For `smailnail`, add/verify `bump-go-go-golems` Makefile target.
- [ ] For `vm-system`, add/verify `bump-go-go-golems` Makefile target.
- [ ] For repositories that already have the target, run `make -n bump-go-go-golems` and record whether it uses `GOWORK=off`.
- [ ] Prefer the infra-tooling `Makefile.bump-go-go-golems-gowork-off.snippet.mk` variant unless a repo has a documented reason not to.
- [ ] Commit or stage dependency-bump target changes separately from generated/logcopter/API changes where practical.

## Phase 2: Complete logcopter setup and generated logger checks

- [ ] For every target repo, run or verify `go get github.com/go-go-golems/logcopter@latest`.
- [ ] For every target repo, run or verify `go get -tool github.com/go-go-golems/logcopter/cmd/logcopter-gen@latest`.
- [ ] For `go-go-os-backend`, add or repair root `logcopter_generate.go`.
- [ ] For `js-analyzer`, add or repair root `logcopter_generate.go`.
- [ ] Verify each repo's `logcopter_generate.go` uses the real root package name.
- [ ] Verify package patterns in `logcopter_generate.go` and `logcopter-check` match exactly.
- [ ] Include `./cmd/...` in logcopter patterns for repos with command packages that need package loggers.
- [ ] Include `./internal/...` only when internal packages should receive generated package loggers.
- [ ] Use `-include-main` where command package logger generation is intended.
- [ ] Use `-var zlog` only where generated `var log` collides with existing imports or variables.
- [ ] Run `go generate ./...` in each repo that receives or refreshes logcopter generation.
- [ ] Run `make logcopter-check` in every repo with logcopter setup.
- [ ] Resolve generated logger collisions by removing obsolete global logger imports or aliasing `log` imports as `stdlog`/`zlog`.
- [ ] Record logcopter generation failures and exact fixes in the diary.

## Phase 3: Add or verify Glazed CLI policy linting

- [ ] For `go-go-gepa`, add/verify `glazed-lint-build` and `glazed-lint` Makefile targets.
- [ ] For `go-go-host`, add/verify `glazed-lint-build` and `glazed-lint` Makefile targets.
- [ ] For `goja-github-actions`, add/verify `glazed-lint-build` and `glazed-lint` Makefile targets.
- [ ] For `goja-text`, add/verify `glazed-lint-build` and `glazed-lint` Makefile targets.
- [ ] For `plz-confirm`, add/verify `glazed-lint-build` and `glazed-lint` Makefile targets.
- [ ] For `scraper`, add/verify `glazed-lint-build` and `glazed-lint` Makefile targets.
- [ ] For `vm-system`, add/verify `glazed-lint-build` and `glazed-lint` Makefile targets.
- [ ] Verify repositories that already have `glazed-lint` still build the vettool from the repository's resolved Glazed version or an intentional fallback.
- [ ] Wire `glazed-lint` into `lint`/`lintmax` where those targets exist.
- [ ] Add CI workflow step `make glazed-lint` only where local Makefile wiring is insufficient or CI does not call lint targets.
- [ ] Run `make glazed-lint` in every Glazed-dependent repo.
- [ ] Fix diagnostics by migrating raw env/flag usage to Glazed fields/config middleware where feasible.
- [ ] Add only narrow `GLAZED_LINT_FLAGS` allow paths for intentional legacy bridge/helper code.
- [ ] Add reasoned `//glazedclilint:ignore ...` suppressions only at the smallest affected statement when needed.
- [ ] Record every suppression and its rationale in the diary or commit message.

## Phase 4: Bump go-go-golems dependencies, especially go-go-goja

- [ ] Run `GOWORK=off make bump-go-go-golems` in dependency order for every in-scope repo.
- [ ] Confirm resulting `go.mod` uses the intended latest published `github.com/go-go-golems/go-go-goja` version.
- [ ] Run `GOWORK=off go mod tidy` after any manual `go get` or API fix.
- [ ] Check for accidental local `replace github.com/go-go-golems/... => ...` directives and remove before PR.
- [ ] If a downstream repo cannot resolve a required upstream symbol, record the missing module/tag and pause downstream work until the upstream release exists.
- [ ] Keep dependency-only changes separate from large API rewrites when feasible.
- [ ] Record old/new `go-go-goja` versions per repo in the diary.

## Phase 5: Adapt downstream code to current go-go-goja APIs

- [ ] Replace older runtime construction patterns with `engine.NewRuntimeFactoryBuilder(...).Build()` and `factory.NewRuntime(...)`.
- [ ] Use `engine.WithStartupContext(ctx)` and `engine.WithLifetimeContext(ctx)` when runtime lifecycle context matters.
- [ ] Replace ad hoc module-selection logic with `UseModuleMiddleware(engine.MiddlewareSafe())`, `MiddlewareOnly(...)`, `MiddlewareExclude(...)`, or `MiddlewareAdd(...)`.
- [ ] Ensure native modules implement `modules.NativeModule` with `Name()`, `Doc()`, and `Loader(*goja.Runtime, *goja.Object)`.
- [ ] Register default modules with `modules.Register(...)` only when global default exposure is intended.
- [ ] Pass explicit modules through `WithModules(...)` when the runtime should be self-contained.
- [ ] For async modules, use `runtimebridge.Lookup(vm)` to obtain runtime services.
- [ ] Settle promises or re-enter JS through `RuntimeServices.PostWithCurrentContext`, `PostWithLifetimeContext`, or `PostWithCustomContext` rather than arbitrary goroutines.
- [ ] Move JS runtime interactions in tests under `rt.Owner.Call(ctx, op, func(ctx context.Context, vm *goja.Runtime) (any, error) { ... })`.
- [ ] Use `rt.AddCloser(...)` and `rt.Close(ctx)` for runtime-owned cleanup.
- [ ] Avoid adding compatibility shims unless a concrete downstream package cannot be migrated safely.
- [ ] Record each API migration pattern and exact failing compiler/test output in the diary.

## Phase 6: Per-repository validation

- [ ] Run `make logcopter-check` in every repo with logcopter setup.
- [ ] Run `make glazed-lint` in every repo with Glazed dependency and lint target.
- [ ] Run `GOWORK=off go test ./...` in every changed repo.
- [ ] Run `make lintmax` where available.
- [ ] Run `make lint` where `lintmax` is unavailable.
- [ ] Run `make test` where it includes repo-specific non-Go tests or integration checks.
- [ ] Run `make build` for CLI/application repos.
- [ ] Run repo-specific smoke tests documented in Makefiles or READMEs.
- [ ] Confirm `git status --short` only shows intentional files.
- [ ] Inspect diffs for `go.mod`, `go.sum`, `Makefile`, `.github/workflows`, `lefthook.yml`, `logcopter_generate.go`, and `**/logcopter.go`.
- [ ] Capture validation commands and outcomes in the diary for each repo.

## Phase 7: PR, merge, and downstream release train

- [ ] Create one focused branch per repository or per tightly coupled repository group.
- [ ] Commit focused changes with messages that distinguish tooling, logcopter, dependency bumps, and API migrations.
- [ ] Push branches and open PRs rather than committing rollout changes directly to `main`.
- [ ] Allow CI to run and wait for Codex/automated review readiness where configured.
- [ ] Use `ggg pr codex-trigger ... --wait-for-auto 30s` only if the automatic review does not appear.
- [ ] Use `ggg pr watch ...` or `ggg batch ready ... --watch` to track readiness.
- [ ] Merge upstream dependencies before downstream repositories.
- [ ] Use merge commits and delete remote branches; do not squash release-train/logcopter rollout PRs.
- [ ] After each upstream merge/release, rerun downstream `GOWORK=off make bump-go-go-golems` where applicable.
- [ ] Update the ticket changelog after each merged repo or meaningful blocked attempt.
- [ ] Keep the diary chronological and include exact failure outputs, fixes, validation commands, PR URLs, and merge commits.

## Phase 8: Final ticket closeout

- [ ] Refresh `sources/01-workspace-inventory.md` after all repositories are updated.
- [ ] Update `design-doc/01-implementation-guide.md` with any migration patterns discovered during implementation.
- [ ] Ensure all per-repository tasks above are checked or explicitly documented as skipped/out of scope.
- [ ] Run `docmgr doctor --ticket BUMP-GOJA-ROLLOUT --stale-after 30`.
- [ ] Add final changelog entry summarizing completed repositories, skipped repositories, and remaining risks.
- [ ] Add final diary entry with validation summary and continuation instructions.
