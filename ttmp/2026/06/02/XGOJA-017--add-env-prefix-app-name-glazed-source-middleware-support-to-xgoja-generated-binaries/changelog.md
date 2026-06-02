# Changelog

## 2026-06-02

- Initial workspace created


## 2026-06-02

Research and design: mapped xgoja architecture, Glazed middleware chain, and pinocchio reference patterns. Created comprehensive design doc with schema proposal, decision records, and phased implementation plan.

### Related Files

- /home/manuel/workspaces/2026-06-02/add-xgoja-env-app-name/go-go-goja/cmd/xgoja/internal/buildspec/spec.go — Baseline for proposed schema extensions


## 2026-06-02

Created research logbook documenting all 34 resources consulted during design phase, with usefulness ratings, out-of-date assessments, and implementation priority rankings.

### Related Files

- /home/manuel/workspaces/2026-06-02/add-xgoja-env-app-name/go-go-goja/ttmp/2026/06/02/XGOJA-017--add-env-prefix-app-name-glazed-source-middleware-support-to-xgoja-generated-binaries/reference/02-research-logbook.md — Research logbook with 34 resource entries


## 2026-06-02

Created review document for the intern research/design package, highlighting strengths, corrections, missing checks, and a narrower recommended MVP before implementation.

### Related Files

- /home/manuel/workspaces/2026-06-02/add-xgoja-env-app-name/go-go-goja/ttmp/2026/06/02/XGOJA-017--add-env-prefix-app-name-glazed-source-middleware-support-to-xgoja-generated-binaries/analysis/01-review-of-intern-feature-plan-and-research-package.md — Intern-facing technical review and coaching notes


## 2026-06-02

Phase 1 MVP implemented: appName/envPrefix support for generated xgoja binaries, shell-safe prefix derivation, runtime middleware factory propagation, tests, and buildspec docs (commit f773542).

### Related Files

- /home/manuel/workspaces/2026-06-02/add-xgoja-env-app-name/go-go-goja/pkg/xgoja/app/middlewares.go — Runtime middleware policy


## 2026-06-02

Phase 2 implemented: config file support for generated xgoja binaries with layered discovery (system/xdg/home/git-root/cwd/explicit), config < env < CLI precedence, validation, and integration tests.

### Related Files

- /home/manuel/workspaces/2026-06-02/add-xgoja-env-app-name/go-go-goja/pkg/xgoja/app/middlewares.go — Config plan builder and middleware ordering


## 2026-06-02

Phase 4 complete: built all existing examples, added 11-config-env example, fixed RenderEmbeddedSpec generator bug, verified end-to-end config/env/CLI precedence in generated binaries.

### Related Files

- /home/manuel/workspaces/2026-06-02/add-xgoja-env-app-name/go-go-goja/cmd/xgoja/internal/generate/main.go — Fixed embedded spec JSON to include appName/envPrefix/config


## 2026-06-02

Implementation review follow-up: tightened explicit config-layer semantics, relaxed appName validation for local config layers, added generator regression coverage, expanded buildspec docs, and verified with focused/full tests (commit e3f6986).

### Related Files

- /home/manuel/workspaces/2026-06-02/add-xgoja-env-app-name/go-go-goja/cmd/xgoja/doc/06-buildspec-reference.md — Public config schema documented
- /home/manuel/workspaces/2026-06-02/add-xgoja-env-app-name/go-go-goja/cmd/xgoja/internal/generate/generate_test.go — Generator embedded-spec regression added
- /home/manuel/workspaces/2026-06-02/add-xgoja-env-app-name/go-go-goja/pkg/xgoja/app/middlewares.go — Runtime config-layer semantics fixed

