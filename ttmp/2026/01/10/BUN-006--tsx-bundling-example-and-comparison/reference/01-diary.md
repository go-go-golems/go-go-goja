---
Title: Diary
Ticket: BUN-006
Status: active
Topics:
    - bun
    - bundling
    - tsx
    - goja
    - analysis
    - docs
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/vibes/2025/06/22/tui-ink/go-app/main.go
      Note: tui-ink runtime contract
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/vibes/2025/06/22/tui-ink/js-modules/webpack.config.js
      Note: tui-ink bundler reference
    - Path: cmd/bun-demo/Makefile
      Note: TSX build targets
    - Path: cmd/bun-demo/js/package.json
      Note: TSX build scripts
    - Path: cmd/bun-demo/js/src/tsx/App.tsx
      Note: TSX demo component
    - Path: cmd/bun-demo/js/src/tsx/render.tsx
      Note: TSX HTML renderer
    - Path: cmd/bun-demo/js/src/tsx/runtime.ts
      Note: Custom JSX runtime
    - Path: ttmp/2026/01/10/BUN-006--tsx-bundling-example-and-comparison/analysis/01-tsx-bundling-example-tui-ink-comparison.md
      Note: Analysis draft
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-10T22:22:34.044733556-05:00
WhatFor: ""
WhenToUse: ""
---



# Diary

## Goal

Capture the TSX bundling analysis work and the comparison with tui-ink.

## Step 1: Ticket scaffolding and TSX analysis draft

I created the BUN-006 ticket workspace and drafted the TSX bundling analysis. The analysis includes an end-to-end example pipeline for TSX-to-HTML output and a direct comparison with the tui-ink webpack + Babel approach.

This establishes the reference material needed to decide between CommonJS and global export contracts for Goja and prepares a concrete example for review.

### What I did
- Created the BUN-006 ticket workspace with analysis and diary docs.
- Drafted the TSX bundling example and comparison in the analysis doc.
- Added tasks for analysis, example pipeline, and reMarkable upload.

### Why
- We needed a structured place to capture the TSX pipeline and compare it to existing bundler usage in tui-ink.

### What worked
- The analysis doc now documents a TSX-to-HTML pipeline and identifies the differences with tui-ink.

### What didn't work
- N/A

### What I learned
- Using `--jsx-import-source=preact` is important for a TSX runtime that avoids React-specific tooling.

### What was tricky to build
- Balancing a conceptual example with the specific Goja compatibility constraints (ES5 + CommonJS).

### What warrants a second pair of eyes
- Confirm the example bundle flags and JSX runtime settings match the intended TSX stack.

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-006--tsx-bundling-example-and-comparison/analysis/01-tsx-bundling-example-tui-ink-comparison.md`.
- Review the comparison references against `/home/manuel/code/wesen/corporate-headquarters/vibes/2025/06/22/tui-ink/js-modules/webpack.config.js` and `/home/manuel/code/wesen/corporate-headquarters/vibes/2025/06/22/tui-ink/go-app/main.go`.

### Technical details
- Ticket created with `docmgr ticket create-ticket --ticket BUN-006 --title "TSX bundling example and comparison" --topics bun,bundling,tsx,goja,analysis,docs`.

## Step 2: Upload analysis to reMarkable

I exported the TSX bundling analysis to PDF and uploaded it to the reMarkable device. This makes the document reviewable outside the repo while keeping the ticket as the source of truth.

### What I did
- Ran a dry-run upload for the analysis doc.
- Uploaded the generated PDF to the reMarkable `ai/2026/01/10/` folder.

### Why
- The request required the analysis to be available for review on reMarkable.

### What worked
- The upload succeeded and the PDF is now on the device.

### What didn't work
- N/A (the upload logged a remote tree refresh warning but completed successfully).

### What I learned
- The remarkable uploader can emit a refresh warning even when the upload succeeds.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-006--tsx-bundling-example-and-comparison/analysis/01-tsx-bundling-example-tui-ink-comparison.md`.
- Confirm the uploaded PDF `01-tsx-bundling-example-tui-ink-comparison.pdf` exists under `ai/2026/01/10/`.

### Technical details
- Dry-run: `python3 /home/manuel/.local/bin/remarkable_upload.py --dry-run /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-006--tsx-bundling-example-and-comparison/analysis/01-tsx-bundling-example-tui-ink-comparison.md`.
- Upload: `python3 /home/manuel/.local/bin/remarkable_upload.py /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-006--tsx-bundling-example-and-comparison/analysis/01-tsx-bundling-example-tui-ink-comparison.md`.

## Step 3: Implement the TSX demo and validate output

I implemented a TSX demo under `cmd/bun-demo/js/src/tsx` and wired build/run targets so the demo can render HTML through Goja. The TSX flow now compiles to a CommonJS bundle, embeds it under `cmd/bun-demo/assets`, and runs the bundle via `go-run-bun-tsx`.

The first attempt used `preact-render-to-string`, but esbuild could not downlevel `const` to ES5 (its current limitation), so I replaced it with a small custom JSX runtime that renders HTML strings without extra transpilation. This keeps the pipeline ES5-compatible and avoids additional build tools.

**Commit (code):** 44ef943 — ":sparkles: Add TSX rendering to bun-demo"

### What I did
- Added a TSX app (`App.tsx`, `render.tsx`, `entry.tsx`) plus a custom runtime (`runtime.ts`).
- Updated `cmd/bun-demo/js/package.json` scripts for `build:tsx` and `render:tsx-html`.
- Updated `cmd/bun-demo/Makefile` with `js-bundle-tsx`, `js-render-tsx`, and `go-run-bun-tsx`.
- Adjusted `cmd/bun-demo/main.go` embed pattern to include `assets/*.cjs`.
- Ran `make -C cmd/bun-demo go-run-bun-tsx` and `make -C cmd/bun-demo js-render-tsx`.

### Why
- We needed a concrete TSX demo that renders HTML in Goja without requiring a browser runtime.
- The custom runtime avoids an extra Babel/SWC step while keeping ES5 output.

### What worked
- `go-run-bun-tsx` prints a full HTML string with the header and list content.
- `js-render-tsx` writes `assets/tsx.html` from the bundled output.

### What didn't work
- `make -C cmd/bun-demo go-run-bun-tsx` failed initially with esbuild 0.25.12:
  - `Transforming const to the configured target environment ("es5") is not supported yet`
  - `src/tsx/render.tsx:5:2` and `node_modules/preact-render-to-string/dist/index.mjs:1:2158`

### What I learned
- esbuild 0.25.x does not downlevel `const` to ES5, so dependencies must already be ES5 or require a separate transpile step.

### What was tricky to build
- Ensuring the custom JSX runtime preserves nested HTML without double-escaping required a `HtmlChunk` wrapper and a `renderToString` helper.

### What warrants a second pair of eyes
- Verify the custom runtime’s escaping and HTML concatenation logic is correct for nested elements and attributes.

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/js/src/tsx/runtime.ts` and `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/js/src/tsx/render.tsx`.
- Review `cmd/bun-demo/Makefile`, `cmd/bun-demo/js/package.json`, and `cmd/bun-demo/main.go` for build/entry changes.
- Run `make -C cmd/bun-demo go-run-bun-tsx` to verify output.

### Technical details
- Build command: `make -C cmd/bun-demo go-run-bun-tsx`.

## Step 4: Close the ticket

I closed the BUN-006 ticket now that the TSX demo is implemented, validated, and documented. This updates the ticket index and changelog to reflect completion.

### What I did
- Ran `docmgr ticket close --ticket BUN-006`.
- Verified the ticket status changed to complete.

### Why
- All tasks were complete and the demo is validated, so the ticket should be marked done.

### What worked
- The ticket status and changelog updated successfully.

### What didn't work
- N/A

### What I learned
- N/A

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-006--tsx-bundling-example-and-comparison/index.md`.
- Confirm the ticket status is `complete` and the changelog shows the closure entry.

### Technical details
- Command: `docmgr ticket close --ticket BUN-006`.
