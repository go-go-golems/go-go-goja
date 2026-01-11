---
Title: Diary
Ticket: BUN-005
Status: active
Topics:
    - bun
    - bundling
    - goja
    - commonjs
    - docs
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/bun-demo/Makefile
      Note: Split bundle targets
    - Path: cmd/bun-demo/js/src/split/app.ts
      Note: Split demo entrypoint
    - Path: cmd/bun-demo/js/src/split/modules/metrics.ts
      Note: Split demo module
    - Path: cmd/bun-demo/main.go
      Note: Embed assets-split
    - Path: pkg/doc/bun-goja-bundling-playbook.md
      Note: Model B playbook update
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-10T21:53:58.634516305-05:00
WhatFor: ""
WhenToUse: ""
---



# Diary

## Goal

Track the work to add a Model B split-bundle demo for Goja, including Go embed updates and documentation.

## Step 1: Ticket scaffolding and split-bundle plan

I created the ticket workspace for the split bundle demo and captured the planned architecture. This establishes the intended directory layout, build steps, and the minimal Go changes needed to load a runtime module graph.

The analysis doc now documents the split-bundle approach (multiple entrypoints, externalized modules, and a directory embed) so implementation can follow a clear path.

### What I did
- Created the BUN-005 ticket workspace with analysis and diary docs.
- Added tasks for the split demo, Go embed changes, and documentation updates.
- Wrote the initial split-bundle plan in the analysis doc.

### Why
- The work spans JS, Go, and documentation updates, so a ticket ensures consistent tracking.
- Writing the plan first reduces risk of missing path updates in the bundling pipeline.

### What worked
- The ticket workspace and initial analysis were created successfully.

### What didn't work
- N/A

### What I learned
- The split-bundle demo can reuse the existing loader with a broader `//go:embed` scope.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-005--split-bundle-demo-for-goja/analysis/01-split-bundle-demo-plan.md`.
- Confirm the tasks list aligns with the planned implementation steps.

### Technical details
- Ticket created with `docmgr ticket create-ticket --ticket BUN-005 --title "Split bundle demo for Goja" --topics bun,bundling,goja,commonjs,docs`.

## Step 2: Implement the split bundle demo

I added a new split-bundle entrypoint and module pair under `cmd/bun-demo/js/src/split`, wired a new `build:split` script, and introduced Makefile targets for running the split demo. The Go embed directive now includes the `assets-split` outputs so the runtime loader can resolve `require()` calls across multiple files.

The demo builds `app.js` and `modules/metrics.js` separately, then uses Goja to load `app.js` and resolve its runtime dependency. The output includes a `mode=split` prefix and SVG metrics to prove the module graph is working.

**Commit (code):** 4cc5e30 — "Add split bundle demo"

### What I did
- Added `cmd/bun-demo/js/src/split/app.ts` and `cmd/bun-demo/js/src/split/modules/metrics.ts` with runtime module dependencies.
- Added `cmd/bun-demo/js/src/split/assets/logo.svg` for asset import validation.
- Added `build:split` to `cmd/bun-demo/js/package.json` and new Makefile targets for split bundling.
- Updated `cmd/bun-demo/main.go` to embed `assets-split` outputs.
- Ran `make -C cmd/bun-demo go-run-bun-split` to verify output.

### Why
- The split-bundle demo illustrates the Model B workflow where multiple bundles load each other at runtime.
- Embedding the full `assets-split` directory is required for Goja to resolve the runtime `require()` graph.

### What worked
- The split build produced `assets-split/app.js` and `assets-split/modules/metrics.js`.
- The Goja demo printed `mode=split` output with SVG metrics.

### What didn't work
- N/A

### What I learned
- Externalizing `./modules/*` keeps the runtime `require()` intact while still bundling each module separately.

### What was tricky to build
- Ensuring `//go:embed` includes nested module files so the loader can resolve subpaths.

### What warrants a second pair of eyes
- Validate that the external pattern remains correct if module paths or directory structure change.

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/js/src/split/app.ts` and `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/js/src/split/modules/metrics.ts`.
- Review `cmd/bun-demo/Makefile` and `cmd/bun-demo/js/package.json` for the split build targets.
- Run `make -C cmd/bun-demo go-run-bun-split` to validate output.

### Technical details
- Build command: `make -C cmd/bun-demo go-run-bun-split`.

## Step 3: Document the split-bundle workflow

I updated the bundling playbook to include the Model B split-bundle workflow, including how to run the new demo target and what output to expect. This keeps the documentation aligned with the new runtime module graph example.

### What I did
- Added a new Model B section and updated the Makefile target list in the playbook.
- Recorded the split demo run command and expected output.

### Why
- Users need a concrete reference for how split bundles are built and loaded in Goja.

### What worked
- The playbook now documents the split workflow alongside the single-bundle path.

### What didn't work
- N/A

### What I learned
- Documenting the split demo next to the single-bundle flow helps clarify when to choose each model.

### What was tricky to build
- Ensuring the documentation remains concise while still describing the runtime module graph.

### What warrants a second pair of eyes
- Validate the playbook examples match the current Makefile targets and file paths.

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/pkg/doc/bun-goja-bundling-playbook.md`.
- Confirm the Model B section references `go-run-bun-split` and the expected output format.

### Technical details
- Document path: `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/pkg/doc/bun-goja-bundling-playbook.md`.
