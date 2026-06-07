---
Title: Implementation guide
Ticket: BUMP-GOJA-ROLLOUT
Status: active
Topics:
    - go
    - tooling
    - maintenance
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/infra-tooling/docs/go-go-golems/glazed-linting-rollout-playbook.md
      Note: Source playbook for Glazed vettool rollout
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/infra-tooling/docs/go-go-golems/logcopter-rollout-colleague-instructions.md
      Note: Source playbook for logcopter rollout
    - Path: go-go-goja/pkg/engine/factory.go
      Note: Current go-go-goja runtime factory API used for migration guidance
    - Path: go-go-goja/pkg/runtimebridge/runtimebridge.go
      Note: Current go-go-goja async runtime services API used for migration guidance
    - Path: go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/scripts/01-inventory-workspace.py
      Note: Ticket-local script that inventories all target repositories and tooling gaps
    - Path: go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/sources/01-workspace-inventory.md
      Note: Captured inventory output used by the guide
ExternalSources:
    - /home/manuel/code/wesen/go-go-golems/infra-tooling/docs/go-go-golems/glazed-linting-rollout-playbook.md
    - /home/manuel/code/wesen/go-go-golems/infra-tooling/docs/go-go-golems/logcopter-rollout-colleague-instructions.md
    - /home/manuel/code/wesen/go-go-golems/infra-tooling/examples/go-go-golems/Makefile.bump-go-go-golems.snippet.mk
    - /home/manuel/code/wesen/go-go-golems/infra-tooling/examples/go-go-golems/Makefile.bump-go-go-golems-gowork-off.snippet.mk
Summary: Implementation guide for rolling Glazed linting, logcopter, and go-go-goja dependency/API updates through the bump-goja workspace.
LastUpdated: 2026-06-06T22:45:00-04:00
WhatFor: Use this to execute the multi-repository rollout consistently and in dependency order.
WhenToUse: Before changing any non-glazed, non-go-go-goja repository in /home/manuel/workspaces/2026-06-06/bump-goja.
---


# Implementation guide

## Executive Summary

This ticket covers a coordinated rollout across every Go repository in `/home/manuel/workspaces/2026-06-06/bump-goja` except `glazed` and `go-go-goja`. The work has three goals: apply Glazed CLI policy linting when missing, finish or verify logcopter setup, and bump go-go-golems dependencies so downstream repositories adapt to the current `go-go-goja` APIs.

The safe execution strategy is dependency-first. First inventory every repository, then update foundational/library repositories, then applications and leaf CLIs. Validate each repository with `GOWORK=off` so the local workspace cannot hide unpublished symbols or accidental reliance on sibling checkout state.

## Problem Statement and Scope

The workspace contains many repositories that depend directly or indirectly on `github.com/go-go-golems/go-go-goja`. Current observed versions range from very old `v0.0.4`/`v0.4.x` releases to `v0.8.0`, while the checked-out source repository reports `v0.8.3` with `git describe --tags --always`. Several repositories already have parts of the tooling rollout, but some are missing `bump-go-go-golems`, `glazed-lint`, or complete logcopter generation wiring.

In scope:

1. All Go repositories under `/home/manuel/workspaces/2026-06-06/bump-goja/*/go.mod` except `glazed` and `go-go-goja`.
2. Makefile and CI wiring for Glazed linting where the repo depends on `glazed` and does not already run `glazed-lint`.
3. Logcopter dependency, `logcopter_generate.go`, generated `logcopter.go` files, `logcopter-generate`, and `logcopter-check` where missing or incomplete.
4. Dependency bump target and actual dependency bumps, especially `github.com/go-go-golems/go-go-goja@latest`.
5. Compile/test fixes caused by current `go-go-goja` APIs.

Out of scope for this ticket:

1. Changing `glazed` or `go-go-goja` themselves, except reading them as references.
2. Large product behavior changes unrelated to dependency/API adaptation.
3. Pushing directly to `main`; rollout changes should be reviewed through branches/PRs.

## Current-State Analysis

### Workspace inventory

The script `scripts/01-inventory-workspace.py` inventories repository state and writes Markdown/JSON. Its captured output is stored in `sources/01-workspace-inventory.md`.

| repo | goja | glazed | logcopter dep | logcopter gen | bump | glazed-lint | logcopter-check | deps |
|---|---|---|---|---|---|---|---|---|
| cozodb-goja | no | no | no | no | no | no | no | XXX |
| css-visual-diff | yes | yes | yes | yes | yes | yes | yes | geppetto, glazed, go-go-goja, logcopter, pinocchio |
| discord-bot | yes | yes | yes | yes | yes | yes | yes | glazed, go-go-goja, logcopter |
| go-go-app-inventory | yes | yes | yes | yes | no | yes | yes | geppetto, glazed, go-go-goja, go-go-os-backend, go-go-os-chat, logcopter, pinocchio, plz-confirm |
| go-go-gepa | yes | yes | yes | yes | no | no | yes | clay, geppetto, glazed, go-go-goja, go-go-os-backend, logcopter |
| go-go-host | yes | yes | yes | yes | no | no | yes | glazed, go-go-goja, logcopter |
| go-go-os-backend | yes | no | yes | no | no | yes | yes | go-go-goja, logcopter |
| go-minitrace | yes | yes | yes | yes | yes | yes | yes | clay, glazed, go-go-goja, logcopter |
| goja-git | yes | yes | yes | yes | yes | yes | yes | glazed, go-go-goja, logcopter |
| goja-github-actions | yes | yes | yes | yes | no | no | yes | glazed, go-go-goja, logcopter |
| goja-text | yes | yes | yes | yes | yes | no | yes | glazed, go-go-goja, logcopter, sanitize |
| jesus | yes | yes | yes | yes | no | yes | yes | clay, geppetto, glazed, go-go-goja, go-go-mcp, logcopter, pinocchio |
| js-analyzer | yes | yes | yes | no | no | yes | yes | glazed, go-go-goja, logcopter |
| loupedeck | yes | yes | yes | yes | yes | yes | yes | glazed, go-go-goja, logcopter |
| pinocchio | yes | yes | yes | yes | yes | yes | yes | bobatea, clay, geppetto, glazed, go-go-goja, logcopter, sanitize, sessionstream, uhoh |
| plz-confirm | yes | yes | yes | yes | no | no | yes | glazed, go-go-goja, logcopter |
| scraper | yes | yes | yes | yes | yes | no | yes | glazed, go-go-goja, logcopter, sessionstream |
| smailnail | yes | yes | yes | yes | no | yes | yes | clay, geppetto, glazed, go-go-goja, go-go-mcp, logcopter |
| vm-system | yes | yes | yes | yes | no | no | yes | glazed, go-go-goja, logcopter |
| workspace-manager | yes | yes | yes | yes | yes | yes | yes | clay, glazed, go-go-goja, logcopter |

Immediate implications:

1. `cozodb-goja` looks anomalous: its module/dependency inventory reports `github.com/go-go-golems/XXX`, does not depend on `go-go-goja` or `glazed`, and should be triaged separately before applying this rollout pattern.
2. Repositories missing `bump-go-go-golems`: `go-go-app-inventory`, `go-go-gepa`, `go-go-host`, `go-go-os-backend`, `goja-github-actions`, `jesus`, `js-analyzer`, `plz-confirm`, `smailnail`, `vm-system`.
3. Repositories missing `glazed-lint` despite depending on Glazed: `go-go-gepa`, `go-go-host`, `goja-github-actions`, `goja-text`, `plz-confirm`, `scraper`, `vm-system`.
4. Repositories with incomplete logcopter generation signals: `go-go-os-backend` has no root `logcopter_generate.go`; `js-analyzer` has no root `logcopter_generate.go` despite a `logcopter-check` target.

### Evidence from infra-tooling playbooks

The Glazed lint rollout playbook defines the vettool as `github.com/go-go-golems/glazed/cmd/tools/glazed-lint` and wires it through `go vet -vettool=$(GLAZED_LINT_BIN)` with `GLAZED_LINT_FLAGS` allow paths. The same playbook says missing release-train repositories should add `make bump-go-go-golems` from the infra-tooling snippets before continuing.

The logcopter rollout instructions define a complete conversion as checked-in generated `logcopter.go` files, a root `logcopter_generate.go`, `make logcopter-check`, and the generic `bump-go-go-golems` target. They explicitly instruct downstream validation with `GOWORK=off go test ./...`.

### Evidence from current go-go-goja APIs

Current `go-go-goja` source exposes an explicit runtime factory flow:

1. `pkg/engine/factory.go` defines `NewRuntimeFactoryBuilder`, `WithModules`, `UseModuleMiddleware`, and `WithRuntimeInitializers`.
2. `UseModuleMiddleware` is documented as the preferred way to control default-registry module selection; plain builders preserve historical behavior by loading all default-registry modules.
3. `pkg/engine/runtime.go` shows default modules are registered by blank imports and can be restricted with `MiddlewareSafe` or `MiddlewareOnly`.
4. `modules/common.go` defines `NativeModule` as `Name()`, `Doc()`, and `Loader(*goja.Runtime, *goja.Object)`, with registration through `modules.Register`.
5. `pkg/runtimebridge/runtimebridge.go` defines `RuntimeServices` for modules that need runtime-owned scheduling, async settlement, and lifetime-aware owner calls/posts.

These are the expected APIs to adapt downstream compile errors toward, not away from.

## Gap Analysis

### Glazed linting gaps

Some repositories already have robust `glazed-lint` integration (`css-visual-diff`, `discord-bot`, `go-minitrace`, `goja-git`, `loupedeck`, `pinocchio`, `workspace-manager`). Use these as local examples before editing missing repos.

For missing repos, add:

1. `GLAZED_LINT_BIN`, `GLAZED_LINT_PKG`, `GLAZED_VERSION` or pinned fallback variables.
2. `glazed-lint-build` and `glazed-lint` targets.
3. `go vet -vettool=$(GLAZED_LINT_BIN) ...` inside `lint` and `lintmax`, or a CI step that runs `make glazed-lint`.
4. Narrow suppressions or `GLAZED_LINT_FLAGS` allow paths only where violations are intentional legacy bridge/helper code.

### Logcopter gaps

Most repos already depend on logcopter and have `logcopter-check`. Confirm they have generated files and that `make logcopter-check` passes. For incomplete repos, add or repair:

1. `go get github.com/go-go-golems/logcopter@latest`.
2. `go get -tool github.com/go-go-golems/logcopter/cmd/logcopter-gen@latest`.
3. Root `logcopter_generate.go` with the real root package name.
4. Generated `logcopter.go` files under the selected package patterns.
5. `logcopter-generate` and non-mutating `logcopter-check` targets using identical package patterns.

### Dependency and API adaptation gaps

The direct `go-go-goja` versions visible in `go.mod` files vary significantly. The bump should happen under `GOWORK=off` to force validation against published modules. Expect compile errors in repositories that still use older runtime construction patterns, direct `require.Registry` setup, or module registration patterns superseded by the current factory/module APIs.

## Proposed Solution

Run the rollout as a dependency-ordered sequence of small repository changes. Each repository gets one focused branch, one focused commit/PR when possible, and validation evidence captured in the diary.

### Repository ordering

Use this first-pass order, then refine with `go.mod` edges when a test failure reveals a missing upstream release:

1. Triage/anomaly: `cozodb-goja`.
2. Foundational libraries or low-level modules: `go-go-os-backend`, `goja-text`, `goja-git`, `goja-github-actions`, `js-analyzer`.
3. Shared application/framework repos: `plz-confirm`, `go-go-host`, `vm-system`, `scraper`, `workspace-manager`.
4. Larger apps depending on multiple go-go-golems repos: `go-go-gepa`, `go-go-app-inventory`, `jesus`, `smailnail`, `discord-bot`, `css-visual-diff`, `go-minitrace`, `loupedeck`, `pinocchio`.

If repository A depends on repository B and B requires code changes, merge/release B before bumping A.

### Standard per-repository loop

Run from each repository root:

```bash
# 0. Start clean and isolate work.
git status --short
git checkout -b task/bump-goja-tooling

# 1. Inspect direct go-go-golems dependencies.
awk '/^require[[:space:]]+github\.com\/go-go-golems\// { print $2 } /^[[:space:]]*github\.com\/go-go-golems\// { print $1 }' go.mod | sort -u

# 2. Add missing Makefile release-train target if needed.
make -n bump-go-go-golems || true

# 3. Bump dependencies using published modules only.
GOWORK=off make bump-go-go-golems

# 4. Add/fix logcopter if incomplete.
go get github.com/go-go-golems/logcopter@latest
go get -tool github.com/go-go-golems/logcopter/cmd/logcopter-gen@latest
go mod tidy
go generate ./...
make logcopter-check

# 5. Add/fix Glazed lint if repo depends on glazed.
make glazed-lint

# 6. Validate with workspace leakage disabled.
GOWORK=off go test ./...
make lintmax || make lint || true

git status --short
git diff -- go.mod go.sum Makefile .github/workflows lefthook.yml logcopter_generate.go '**/logcopter.go'
```

### Glazed lint implementation details

Add this style of Makefile wiring when missing, adapting package patterns to the repo:

```make
GLAZED_LINT_BIN ?= /tmp/glazed-lint
GLAZED_LINT_PKG ?= github.com/go-go-golems/glazed/cmd/tools/glazed-lint
GLAZED_VERSION ?= $(shell GOWORK=off go list -m -f '{{.Version}}' github.com/go-go-golems/glazed 2>/dev/null)
GLAZED_LINT_FLAGS ?= -glazedclilint.allow-paths=pkg/analysis/,pkg/cli/,pkg/cmds/fields/,pkg/cmds/logging/,pkg/cmds/sources/,pkg/help/
GLAZED_LINT_DIRS ?= ./cmd/... ./pkg/...

.PHONY: glazed-lint-build glazed-lint
glazed-lint-build:
	@echo "Building glazed-lint from Glazed module..."
	@if [ -n "$(GLAZED_VERSION)" ] && [ "$(GLAZED_VERSION)" != "(devel)" ]; then \
		echo "Installing $(GLAZED_LINT_PKG)@$(GLAZED_VERSION)"; \
		GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_VERSION); \
	else \
		echo "Installing $(GLAZED_LINT_PKG) from workspace/module"; \
		GOBIN=$(dir $(GLAZED_LINT_BIN)) go install $(GLAZED_LINT_PKG); \
	fi

glazed-lint: glazed-lint-build
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)
```

Rules:

1. Use `./pkg/...` for libraries without commands.
2. Use `./cmd/... ./pkg/...` for CLIs/apps.
3. Include `./internal/...` only when CLI policy code lives there.
4. Prefer fixing diagnostics to suppressing them.
5. If suppression is needed, use `//glazedclilint:ignore <reason>` on the smallest statement, or narrow `-glazedclilint.allow-paths=...` entries.

### Logcopter implementation details

Create `logcopter_generate.go` with the actual root package name:

```go
package <rootpkg>

//go:generate go tool logcopter-gen -area-prefix go-go-golems.<repo> -strip-prefix github.com/go-go-golems/<repo> ./pkg/...
```

For apps with command packages:

```go
package <rootpkg>

//go:generate go tool logcopter-gen -include-main -area-prefix go-go-golems.<repo> -strip-prefix github.com/go-go-golems/<repo> ./cmd/... ./pkg/...
```

Use `-var zlog` when generated `var log` would collide with package imports or intentional logger variables; `plz-confirm` and `scraper` already show this pattern. Add Makefile targets with the same package patterns:

```make
.PHONY: logcopter-generate
logcopter-generate:
	GOWORK=off go tool logcopter-gen -area-prefix go-go-golems.<repo> -strip-prefix github.com/go-go-golems/<repo> ./cmd/... ./pkg/...

.PHONY: logcopter-check
logcopter-check:
	GOWORK=off go tool logcopter-gen -area-prefix go-go-golems.<repo> -strip-prefix github.com/go-go-golems/<repo> -check ./cmd/... ./pkg/...
```

### go-go-goja API adaptation guide

When `GOWORK=off go test ./...` fails after bumping `go-go-goja`, categorize errors and apply these migrations:

1. Runtime creation: prefer `engine.NewRuntimeFactoryBuilder(...).UseModuleMiddleware(...).WithModules(...).WithRuntimeInitializers(...).Build()`, then `factory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))`.
2. Module selection: use `UseModuleMiddleware(engine.MiddlewareSafe())`, `MiddlewareOnly(...)`, `MiddlewareExclude(...)`, or `MiddlewareAdd(...)` instead of mutating global module state when a runtime should expose a specific module set.
3. Native modules: implement `modules.NativeModule` with `Name()`, `Doc()`, and `Loader(*goja.Runtime, *goja.Object)`. Register global default modules from `init()` with `modules.Register(...)`, or pass explicit modules via `engine.NativeModuleRegistrar`/registrars to `WithModules(...)`.
4. Async/background module code: use `runtimebridge.Lookup(vm)` to obtain `RuntimeServices`. Settle promises through `PostWithCurrentContext`, `PostWithLifetimeContext`, or `PostWithCustomContext` rather than calling into goja from arbitrary goroutines.
5. Lifecycle: use `rt.Close(ctx)` and `rt.AddCloser(...)` for runtime-owned cleanup. Avoid leaking event loops, goroutines, or direct `goja.Runtime` access after close.
6. Tests: rewrite tests to exercise code through `rt.Owner.Call(ctx, op, func(ctx context.Context, vm *goja.Runtime) (any, error) { ... })` when interacting with JS values. This aligns tests with owner-thread safety.

## Design Decisions

### Decision: Validate downstream with `GOWORK=off`

- **Context:** The workspace contains sibling checkouts of many go-go-golems modules. A local `go.work` can hide missing tags or unpublished API changes.
- **Options considered:** Use the workspace for fast local integration; force every downstream validation through published modules; use local replaces temporarily.
- **Decision:** Use `GOWORK=off` for dependency bumps, `go test`, Glazed lint tool installation, and logcopter checks unless intentionally debugging an unreleased upstream locally.
- **Rationale:** The infra-tooling logcopter instructions explicitly warn not to trust local `go.work`, and downstream repositories must work for users consuming published modules.
- **Consequences:** Some changes may need upstream releases before downstream PRs pass; this is slower but prevents false-green validation.
- **Status:** accepted

### Decision: Add tooling only where repository scope justifies it

- **Context:** `cozodb-goja` does not currently match the same dependency shape as the other repos, while most other repos directly depend on `go-go-goja` and/or Glazed.
- **Options considered:** Apply one blanket Makefile patch to every repo; gate tooling by observed dependencies; skip anomalous repos entirely.
- **Decision:** Gate Glazed linting on an actual Glazed dependency; gate logcopter adoption on Go packages that should use package loggers; triage anomalous repos before changing them.
- **Rationale:** Tooling that does not match repository purpose creates maintenance noise and failing hooks.
- **Consequences:** The inventory must be refreshed after dependency changes; `cozodb-goja` needs a deliberate call before edits.
- **Status:** accepted

### Decision: Prefer current go-go-goja factory/runtime APIs over compatibility shims

- **Context:** Some downstream repos may still use older runtime construction or direct module registry patterns.
- **Options considered:** Add compatibility shims locally; pin old `go-go-goja`; migrate call sites to the current builder/runtimebridge APIs.
- **Decision:** Migrate downstream code to current `go-go-goja` APIs.
- **Rationale:** The user asked to bump dependencies and adapt to new APIs, especially `go-go-goja`; compatibility shims would postpone the migration and diverge from upstream design.
- **Consequences:** More compile fixes are expected now, but future dependency bumps should be easier.
- **Status:** accepted

## Implementation Plan

### Phase 0: Baseline and branch discipline

1. Run `scripts/01-inventory-workspace.py > sources/01-workspace-inventory.md` and commit ticket docs/scripts separately from repo rollout changes.
2. For each target repo, ensure `git status --short` is clean or record unrelated changes before editing.
3. Create a branch per repo, for example `task/bump-goja-tooling`.

### Phase 1: Add missing release-train bump targets

1. Add the `bump-go-go-golems` Makefile target to every repo missing it and depending on go-go-golems modules.
2. Prefer the `GOWORK=off` snippet from infra-tooling unless the repo has a clear reason to use workspace mode.
3. Validate with `make -n bump-go-go-golems`.

### Phase 2: Complete logcopter setup

1. Run `go get github.com/go-go-golems/logcopter@latest` and `go get -tool github.com/go-go-golems/logcopter/cmd/logcopter-gen@latest`.
2. Add or fix `logcopter_generate.go`.
3. Run `go generate ./...`.
4. Resolve name collisions by removing obsolete zerolog global imports, aliasing standard `log` as `stdlog`, or using `-var zlog` when appropriate.
5. Run `make logcopter-check` and `GOWORK=off go test ./...`.

### Phase 3: Add missing Glazed lint integration

1. Add Makefile variables and targets.
2. Wire `lint`/`lintmax` or CI to run `make glazed-lint`.
3. Run `make glazed-lint`.
4. Fix diagnostics or add reasoned narrow suppressions.

### Phase 4: Bump go-go-goja and adapt APIs

1. Run `GOWORK=off make bump-go-go-golems`.
2. If compilation fails, sort errors by package and migrate toward current engine/runtimebridge/module APIs.
3. Re-run `GOWORK=off go test ./...` until clean.
4. Run repository-specific smoke/lint/build targets.

### Phase 5: PR and downstream sequencing

1. Commit a focused diff per repo.
2. Open PRs rather than pushing to `main`.
3. Let CI and Codex review complete.
4. Merge dependencies before dependents, using merge commits rather than squash commits for rollout auditability.
5. After upstream merges/releases, rerun downstream `make bump-go-go-golems`.

## Testing and Validation Strategy

Minimum validation for each changed repo:

```bash
make -n bump-go-go-golems
make logcopter-check
make glazed-lint        # only when the repo depends on Glazed
GOWORK=off go test ./...
git status --short
```

Preferred validation when available:

```bash
make lintmax
make test
make build
make smoke
```

Validation evidence to record in the diary:

1. Command run.
2. Exit status.
3. Exact failure output for compile/lint failures.
4. Files changed to fix each failure.
5. Whether the repo needed an upstream release before downstream validation could pass.

## Risks, Alternatives, and Open Questions

### Risks

1. **Workspace leakage:** solved by `GOWORK=off` validation.
2. **Unreleased upstream APIs:** downstream repos may require a published `go-go-goja` tag or releases of intermediate dependencies.
3. **Generated-file churn:** logcopter can touch many packages; review generated diffs carefully.
4. **Over-broad Glazed suppressions:** broad allow paths can disable policy where it matters.
5. **Anomalous repository state:** `cozodb-goja` reports `github.com/go-go-golems/XXX` and should not be blindly patched.

### Alternatives considered

1. **One workspace-wide script that edits every repo automatically:** rejected because API adaptation and linter suppressions require judgment.
2. **Pin old `go-go-goja` versions:** rejected because the requested outcome is to bump and adapt to current APIs.
3. **Use local `replace` directives for all downstream tests:** acceptable only for short debugging sessions, rejected for final validation.

### Open questions

1. Should `cozodb-goja` be repaired/renamed as part of this rollout, or excluded as unrelated?
2. Which repositories require actual GitHub PRs in this workspace versus local-only patches?
3. Is `go-go-goja v0.8.3` already published for every downstream consumer, or will a release train need to publish it first?

## References

- Inventory script: `scripts/01-inventory-workspace.py`
- Captured inventory: `sources/01-workspace-inventory.md`
- Glazed lint playbook: `/home/manuel/code/wesen/go-go-golems/infra-tooling/docs/go-go-golems/glazed-linting-rollout-playbook.md`
- Logcopter rollout instructions: `/home/manuel/code/wesen/go-go-golems/infra-tooling/docs/go-go-golems/logcopter-rollout-colleague-instructions.md`
- Bump target snippet: `/home/manuel/code/wesen/go-go-golems/infra-tooling/examples/go-go-golems/Makefile.bump-go-go-golems.snippet.mk`
- GOWORK-off bump target snippet: `/home/manuel/code/wesen/go-go-golems/infra-tooling/examples/go-go-golems/Makefile.bump-go-go-golems-gowork-off.snippet.mk`
- Current go-go-goja runtime factory: `/home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/pkg/engine/factory.go`
- Current go-go-goja runtime lifecycle: `/home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/pkg/engine/runtime.go`
- Current go-go-goja native module interface: `/home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/modules/common.go`
- Current go-go-goja runtime bridge: `/home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/pkg/runtimebridge/runtimebridge.go`
