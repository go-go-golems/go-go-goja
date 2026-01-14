---
Title: Split bundle demo plan
Ticket: BUN-005
Status: active
Topics:
    - bun
    - bundling
    - goja
    - commonjs
    - docs
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/bun-demo/Makefile
      Note: Makefile targets now invoke the Dagger pipeline
    - Path: cmd/bun-demo/dagger/main.go
      Note: Dagger pipeline CLI and export helpers
    - Path: cmd/bun-demo/js/package.json
      Note: Render script updated for Node to match Dagger pipeline
    - Path: go.mod
      Note: Dagger Go SDK dependency for pipeline
    - Path: go.sum
      Note: Checksum updates for Dagger SDK
ExternalSources: []
Summary: Plan to replace bun-driven Makefile steps with a Dagger Go pipeline for bundling and split outputs.
LastUpdated: 2026-01-14T16:18:03-05:00
WhatFor: Plan the Model B split-bundle demo and the Go changes needed to load a runtime module graph.
WhenToUse: Use before implementing split bundles or updating the Go loader.
---






# Split bundle demo plan

## Overview
This analysis describes how to demo a split-bundle (Model B) workflow for Goja where multiple CommonJS outputs are embedded and loaded at runtime via `require`. It now also outlines how to replace the bun-driven Makefile workflow with a Dagger Go pipeline that performs the TypeScript bundling and exports the assets needed by `cmd/bun-demo`.

## Goals
- Keep the split bundle demo (multiple CJS files + runtime `require` graph) intact.
- Replace bun usage in `cmd/bun-demo/Makefile` with a Dagger Go pipeline.
- Ensure the Dagger pipeline exports the same embed-ready assets:
  - `cmd/bun-demo/assets/bundle.cjs`
  - `cmd/bun-demo/assets/tsx-bundle.cjs`
  - `cmd/bun-demo/assets/tsx.html`
  - `cmd/bun-demo/assets-split/**`
- Keep the Go runtime entrypoint unchanged (`cmd/bun-demo/main.go` still loads assets from `embed.FS`).

## Non-goals
- Changing the Goja runtime loader behavior.
- Reworking the JS sources or bundle contents beyond the build pipeline.

## Why the plan is structured this way
1. The Dagger pipeline must define the authoritative build outputs before the Makefile can switch away from bun. This avoids a mismatch between produced assets and `go:embed` expectations in `cmd/bun-demo/main.go`.
2. The Makefile should remain the developer entrypoint, so it becomes a thin wrapper that calls the pipeline and keeps the existing target names (`js-bundle`, `js-bundle-split`, etc.).
3. The JS `render:tsx-html` script needs to avoid bun once the Makefile stops invoking bun; updating it to use Node keeps the workflow consistent across host and container runs.

## Planned changes

### 1) Add a Dagger Go pipeline for bun-demo
Create a new Go command at `cmd/bun-demo/dagger/main.go` that uses the Dagger Go SDK.

Key symbols and responsibilities:
- `func main()` + Cobra root command `bun-demo-dagger` for CLI entry.
- `type pipeline` holding the Dagger client and `projectDir`.
- `func (p *pipeline) jsContainer()`:
  - Base container: `node:20.18.1`.
  - Mount host `projectDir` into `/src`.
  - Run `npm install --no-audit --no-fund`.
  - Mount cache volume `bun-demo-npm-cache` at `/root/.npm`.
- Export helpers:
  - `exportFile(...)` for `assets/bundle.cjs`, `assets/tsx-bundle.cjs`, `assets/tsx.html`.
  - `exportDir(...)` for `assets-split/**`.
- Subcommands (Cobra) that map to Makefile targets:
  - `bundle` -> `npm run build` -> export `assets/bundle.cjs`.
  - `bundle-split` -> `npm run build:split` -> export `assets-split/**`.
  - `bundle-tsx` -> `npm run build:tsx` -> export `assets/tsx-bundle.cjs`.
  - `render-tsx` -> run `build:tsx` then `node -e` to write `dist/tsx.html` -> export `assets/tsx.html`.
  - `typecheck` -> `npm run typecheck`.
  - `transpile` -> `npm run build` + `esbuild dist/bundle.cjs --target=es5` -> export `js/dist/bundle.es5.cjs`.

### 2) Update the Makefile to call Dagger
Edit `cmd/bun-demo/Makefile` to replace bun invocations with `dagger run -- go run ./dagger ...`.

Targets to update:
- `js-install` -> `dagger` subcommand `deps`.
- `js-typecheck` -> `dagger` subcommand `typecheck`.
- `js-bundle` -> `dagger` subcommand `bundle`.
- `js-bundle-tsx` -> `dagger` subcommand `bundle-tsx`.
- `js-render-tsx` -> `dagger` subcommand `render-tsx`.
- `js-bundle-split` -> `dagger` subcommand `bundle-split`.
- `js-transpile` -> `dagger` subcommand `transpile`.

Keep `js-clean`, `go-build`, and `go-run-*` targets unchanged so the developer workflow remains stable.

### 3) Align the JS render script with Node
Update `cmd/bun-demo/js/package.json` so `render:tsx-html` uses `node -e` instead of `bun -e`. This keeps the render step runnable inside the Dagger Node container.

## Go-side behavior
`cmd/bun-demo/main.go` continues to embed `assets/*` and `assets-split/*` (including nested modules). The Dagger pipeline only replaces how those assets are built and copied.

## Validation plan
- `make -C cmd/bun-demo js-bundle` and confirm `cmd/bun-demo/assets/bundle.cjs` updates.
- `make -C cmd/bun-demo js-bundle-split` and confirm `cmd/bun-demo/assets-split/app.js` + `modules/metrics.js` exist.
- `make -C cmd/bun-demo js-bundle-tsx` and `make -C cmd/bun-demo js-render-tsx` to refresh `assets/tsx-bundle.cjs` and `assets/tsx.html`.
- `make -C cmd/bun-demo go-run-bun` and `make -C cmd/bun-demo go-run-bun-split` to validate runtime output.

## Risks / considerations
- Dagger requires the engine/CLI (`dagger`) to be installed for Makefile targets to work.
- Export paths must stay aligned with `go:embed` globs in `cmd/bun-demo/main.go`.
- Container Node versions should remain stable to avoid bundle drift; pin the image tag in the pipeline.
