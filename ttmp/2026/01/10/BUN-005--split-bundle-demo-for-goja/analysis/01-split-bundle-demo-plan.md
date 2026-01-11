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
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-10T21:53:54.945880241-05:00
WhatFor: "Plan the Model B split-bundle demo and the Go changes needed to load a runtime module graph."
WhenToUse: "Use before implementing split bundles or updating the Go loader."
---

# Split bundle demo plan

## Overview
This analysis describes how to demo a split-bundle (Model B) workflow for Goja where multiple CommonJS outputs are embedded and loaded at runtime via `require`. The demo will build multiple entrypoints, keep shared modules external, and validate that the loader can resolve the runtime module graph.

## Goals
- Build a split bundle demo with multiple CJS files and a runtime `require` graph.
- Embed the split outputs under `cmd/bun-demo/assets-split`.
- Add Makefile targets to build and run the split demo.
- Update documentation to explain the split workflow.

## Approach
- Add `src/split/app.ts` as the entrypoint and `src/split/modules/metrics.ts` as an externalized module.
- Use esbuild with `--bundle` for each entrypoint and `--external:./modules/*` so `app.js` keeps a runtime `require("./modules/metrics")`.
- Output to `js/dist-split` with `--outdir` + `--outbase` to preserve directory structure.
- Copy `dist-split` into `cmd/bun-demo/assets-split` for embedding.
- Update Go `//go:embed` patterns to include `assets-split` files.

## Go-side changes
The existing loader already reads from `embed.FS`, but we need to embed a directory instead of a single file. We will update the `//go:embed` directive to include `assets-split` and call the demo with `--entry ./assets-split/app.js`.

## Validation
- Run `make -C cmd/bun-demo go-run-bun-split`.
- Confirm output includes the split bundle prefix and SVG metrics.
