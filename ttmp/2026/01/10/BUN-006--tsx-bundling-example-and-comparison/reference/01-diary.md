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
