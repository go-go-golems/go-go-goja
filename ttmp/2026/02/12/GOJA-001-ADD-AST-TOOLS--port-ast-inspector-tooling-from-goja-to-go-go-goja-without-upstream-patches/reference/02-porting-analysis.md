---
Title: Porting Analysis
Ticket: GOJA-001-ADD-AST-TOOLS
Status: active
Topics:
    - goja
    - analysis
    - tooling
    - migration
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/repl/main.go
      Note: Target repo command architecture reference
    - Path: go-go-goja/go.mod
      Note: Version compatibility and dependency pin strategy
    - Path: goja/cmd/goja-inspector/main.go
      Note: CLI entrypoint mapping into go-go-goja
    - Path: goja/internal/inspector/completion.go
      Note: Drawer completion logic and dependency surface
    - Path: goja/internal/inspector/drawer.go
      Note: Bottom drawer editor behavior to preserve
    - Path: goja/internal/inspector/index.go
      Note: Primary AST tool to be ported
    - Path: goja/internal/inspector/model.go
      Note: Overall TUI orchestration layer
    - Path: goja/internal/inspector/resolve.go
      Note: Binding/scope analysis engine to be ported
    - Path: goja/internal/inspector/treesitter.go
      Note: Tree-sitter integration and required external deps
ExternalSources: []
Summary: Detailed analysis for porting goja inspector code into go-go-goja to avoid upstream source modifications.
LastUpdated: 2026-02-12T16:14:47.909842894-05:00
WhatFor: Provide implementation-ready migration plan, dependency guidance, and validation strategy.
WhenToUse: Use before and during migration implementation of inspector/AST tooling into go-go-goja.
---


# Porting Analysis

## Goal

Define a concrete, low-risk path to move all inspector functionality currently added under `goja/` into `go-go-goja/`, so upstream `github.com/dop251/goja` can remain unmodified.

## Context

### Current Source of Truth (in `goja`)

Inspector-related code currently lives in:
- `goja/cmd/goja-inspector/main.go`
- `goja/internal/inspector/*.go`

Current inspector package contents (12 files, ~5,076 LOC including tests):
- `index.go`, `noderecord.go`, `resolve.go` (AST index + lexical resolution core)
- `model.go` (Bubble Tea TUI model)
- `treesitter.go`, `completion.go`, `drawer.go` (drawer editor + CST completions)
- tests for each area (`*_test.go`)

Evolution commits (chronological):
1. `7136b8e` initial inspector + TUI + index
2. `0e70664` tmux rendering/crash fixes
3. `1766bef` integration tests
4. `06712eb` character-level highlighting improvements
5. `591402a` AST walker fix for pointer-receiver edge case
6. `ebe6d55` scope resolver + go-to-def + highlight usages
7. `71ab3ff` tree-sitter wrapper + tests
8. `c86b9c3` bottom drawer + completion engine
9. `c679c6f` drawer polish + scope-wide go-to-def refinements

### Findings About Coupling

Observed coupling is lower than expected:
- Inspector code imports public goja packages only:
  - `github.com/dop251/goja/ast`
  - `github.com/dop251/goja/file`
  - `github.com/dop251/goja/token`
  - `github.com/dop251/goja/unistring`
  - `github.com/dop251/goja/parser` (command/tests)
- It does **not** require non-public goja internals.
- The only upstream-specific element is file location and import path (`internal/inspector`) and command residence (`cmd/goja-inspector`).

Conclusion: Port is feasible without upstream code changes.

## Quick Reference

### Validation Evidence Collected

#### 1. Baseline in goja
- `go test ./internal/inspector/... -count=1` => pass
- `go build ./cmd/goja-inspector` => pass

#### 2. Baseline in go-go-goja
- `go test ./... -count=1` => existing unrelated failure:
  - `cmd/bun-demo/main.go:18:25: pattern assets-split/*: no matching files found`
- This is pre-existing and not caused by inspector migration.

#### 3. External Portability Smoke Test (critical)
I copied inspector sources into a clean temporary module and ran with `GOWORK=off`.

Initial failures:
- missing sums (`missing go.sum entry ...`)
- Charm stack version skew (`x/cellbuf` API mismatch against newer `x/ansi`)

After pinning compatible versions, tests/build passed:
- `GOWORK=off go test ./inspector -count=1` => pass
- `GOWORK=off go build ./cmd/inspector-smoke` => pass

Interpretation:
- Source code is portable.
- Dependency pinning is the key operational risk.

### Recommended Target Layout in go-go-goja

Adopt a mandatory two-layer architecture:

1. Reusable framework layer (packageable, tool-agnostic):
- `go-go-goja/pkg/jsparse/`
  - parser/session abstractions over goja parser + file offsets
  - AST index graph (`NodeRecord`, parent/child, offset lookups)
  - lexical scope/binding resolver
  - completion engine interfaces + generic candidate model
  - optional tree-sitter adapters for CST-driven completion context

2. Inspector tool layer (UI/application-specific):
- `go-go-goja/cmd/inspector/main.go`
  - thin CLI entrypoint (example consumer of public APIs)
- `go-go-goja/cmd/inspector/app/`
  - Bubble Tea model
  - drawer editor state
  - keybindings and rendering
  - adapters that consume `pkg/jsparse` APIs

Why this split:
- enables a general purpose JS parsing/completion framework for dev tooling, diagnostics, and error reporting
- decouples reusable analysis from terminal UI concerns
- keeps inspector implementation command-local so it is clearly an example application, not reusable library surface
- preserves ability to evolve inspector without destabilizing shared analysis APIs

### File Mapping Table (Split by Reuse Level)

| From (goja) | To (go-go-goja) | Layer | Notes |
| --- | --- | --- | --- |
| `internal/inspector/index.go` | `pkg/jsparse/index.go` | reusable | core AST indexing |
| `internal/inspector/noderecord.go` | `pkg/jsparse/noderecord.go` | reusable | node model for any tooling |
| `internal/inspector/resolve.go` | `pkg/jsparse/resolve.go` | reusable | lexical binding analysis |
| `internal/inspector/treesitter.go` | `pkg/jsparse/treesitter.go` | reusable | CST adapter and parse snapshots |
| `internal/inspector/completion.go` | `pkg/jsparse/completion.go` | reusable | completion context + candidate resolution |
| `internal/inspector/model.go` | `cmd/inspector/app/model.go` | inspector-specific | UI orchestration only |
| `internal/inspector/drawer.go` | `cmd/inspector/app/drawer.go` | inspector-specific | editor interactions + rendering |
| `internal/inspector/index_test.go` | `pkg/jsparse/index_test.go` | reusable | move analysis tests with logic |
| `internal/inspector/resolve_test.go` | `pkg/jsparse/resolve_test.go` | reusable | move analysis tests with logic |
| `internal/inspector/treesitter_test.go` | `pkg/jsparse/treesitter_test.go` | reusable | move parser adapter tests |
| `internal/inspector/completion_test.go` | `pkg/jsparse/completion_test.go` | reusable | move completion tests |
| `internal/inspector/drawer_test.go` | `cmd/inspector/app/drawer_test.go` | inspector-specific | keep UI/editor behavior tests |
| `cmd/goja-inspector/main.go` | `cmd/inspector/main.go` | inspector-specific | import rewrites to new packages |

### Dependency Plan (go-go-goja)

Add/verify these requirements (direct or stable indirect pins):
- `github.com/tree-sitter/go-tree-sitter v0.25.0`
- `github.com/tree-sitter/tree-sitter-javascript v0.25.0`
- Charm stack should remain coherent with existing versions in `go-go-goja`:
  - `bubbletea v1.3.10`
  - `lipgloss v1.1.1-0.20250404203927-76690c660834`
  - `x/ansi v0.11.3`
  - `x/cellbuf v0.0.14`
  - `x/term v0.2.2`

Important:
- Avoid floating one Charm package independently; run module updates as a set.

### Phased Implementation Plan

#### Phase 1: Extract Reusable JS Analysis Framework (`pkg/jsparse`)
1. Port and normalize reusable files into `pkg/jsparse`:
   - `index`, `noderecord`, `resolve`, `treesitter`, `completion`
2. Remove UI terms from exported APIs and use generic domain language:
   - parse session, cursor position, completion context, binding graph
3. Keep goja parser integration but design APIs so other parsers/AST normalizers can be added later.
4. Move corresponding tests to `pkg/jsparse`.

Acceptance criteria:
- `go test ./pkg/jsparse -count=1` passes
- no dependency on inspector model/drawer types in `pkg/jsparse`

#### Phase 2: Rebuild Inspector Tool on Top of Framework
1. Port inspector-specific UI files to `cmd/inspector/app` (`model`, `drawer`, UI-only tests).
2. Replace direct internal analysis calls with `pkg/jsparse` API usage.
3. Port `cmd/inspector/main.go` to wire:
   - parse/index/resolve/completion from `pkg/jsparse`
   - rendering and interactions from `cmd/inspector/app`

Acceptance criteria:
- `go test ./cmd/inspector/... -count=1` passes
- `go build ./cmd/inspector` succeeds
- runtime UX parity with existing inspector

#### Phase 3: Packageable Reuse for Dev Tools and Errors
1. Add a small service facade in `pkg/jsparse`:
   - `Analyze(source, opts) -> {ASTIndex, Resolution, Diagnostics, CompletionProvider}`
2. Add usage examples for non-TUI use cases:
   - better parser error messages with context windows
   - LSP/dev-tool completion endpoints
   - static diagnostics pipelines in CI
3. Add a stable API contract section in docs (inputs/outputs/versioning expectations).

Acceptance criteria:
- reusable API exercised by at least one non-inspector example/test
- no Charm/Bubble Tea imports in `pkg/jsparse`

#### Phase 4: Dependency Stabilization and CI Guardrails
1. Add tree-sitter dependencies and run `go mod tidy`.
2. Add focused CI targets:
   - `go test ./pkg/jsparse -count=1`
   - `go test ./cmd/inspector/... -count=1`
3. Keep full-suite test strategy explicit around bun-demo asset generation constraints.

Acceptance criteria:
- no version-skew build failures
- reusable framework and inspector tool can be validated independently

#### Phase 5: Upstream Cleanup Strategy
1. Stop carrying inspector patches in goja fork branch for this project.
2. Keep go-go-goja as the home for framework + inspector evolution.

Acceptance criteria:
- no local project dependency on modified upstream goja tree

### Test Matrix for Port Execution

Run these after each phase:
1. `go test ./pkg/jsparse -run TestBuildIndex -count=1`
2. `go test ./pkg/jsparse -run TestResolve -count=1`
3. `go test ./pkg/jsparse -run TestExtractCompletionContext -count=1`
4. `go test ./cmd/inspector/... -run TestDrawer -count=1`
5. `go build ./cmd/inspector`

Optional full-suite gate:
- `go generate ./... && go test ./... -count=1`

### Risk Register and Mitigations

1. Risk: dependency drift in Charm ecosystem
- Symptom: compile errors in `x/cellbuf`/`x/ansi`
- Mitigation: pin coherent versions and upgrade as a group

2. Risk: false migration failures from unrelated bun-demo setup
- Symptom: `assets-split/*` missing during full tests
- Mitigation: use scoped inspector suite or run required generation path first

3. Risk: feature regressions in TUI interactions (non-unit behavior)
- Symptom: keybindings, sync, drawer completion differ
- Mitigation: manual smoke checklist plus targeted tests in `cmd/inspector/app`

4. Risk: framework APIs accidentally shaped around inspector-only needs
- Symptom: hard to use parsing/completion logic in non-TUI tools
- Mitigation:
  - enforce `pkg/jsparse` package boundary
  - keep UI-free interfaces and add non-inspector examples/tests early

### Open Decisions for Implementation Start

1. Final reusable package name: `pkg/jsparse` (proposed) vs `pkg/jstools`.
2. Completion API shape: pull-based `Complete(ctx)` vs precomputed provider object.
3. Command strategy: keep `cmd/inspector` as standalone example first (recommended), `repl` subcommand later.

### Decision Log

- 2026-02-12: Keep `cmd/inspector` as standalone command for now.
  - Rationale: preserves clear package boundary (`pkg/jsparse` reusable, inspector UI example-only), reduces coupling into `cmd/repl`, and keeps migration risk focused.
  - Follow-up: optional `repl` integration can be added later as a separate task if there is demand for inline inspector workflows.

## Usage Examples

### Example: Split-first port sequence

```bash
# 1) Create split packages
mkdir -p go-go-goja/pkg/jsparse
mkdir -p go-go-goja/cmd/inspector/app

# 2) Copy reusable core files to pkg/jsparse and UI files to cmd/inspector/app
# 3) Port cmd/inspector/main.go and rewrite imports to new packages

# 4) Add deps + validate
cd go-go-goja
go mod tidy
go test ./pkg/jsparse -count=1
go test ./cmd/inspector/... -count=1
go build ./cmd/inspector
```

### Example: smoke run

```bash
cd go-go-goja
go run ./cmd/inspector ../goja/testdata/sample.js
```

## Related

- Ticket index: `../index.md`
- Diary: `./01-diary.md`
- Tasks: `../tasks.md`
- Changelog: `../changelog.md`
