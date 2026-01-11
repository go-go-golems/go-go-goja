---
Title: Diary
Ticket: BUN-003
Status: active
Topics:
    - bun
    - bundling
    - typescript
    - assets
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Implementation diary for TypeScript + asset import support."
LastUpdated: 2026-01-10T20:41:59-05:00
WhatFor: "Track analysis, tasks, and implementation steps for BUN-003."
WhenToUse: "When reviewing or continuing the ticket."
---

# Diary

## Goal
Track the work to add TypeScript and asset import support to the Goja bundling pipeline.

## Step 1: Open ticket and draft analysis + testing plan

I created a new ticket for TypeScript and asset import support, then drafted an analysis document and a real-world testing playbook. This captures the rationale for switching bundling to esbuild (to gain loader support) and the validation steps needed to confirm SVG imports and TS typechecking.

I also added tasks to track the bundling changes, TypeScript scaffolding, and validation steps.

### What I did
- Created ticket BUN-003 and added analysis, playbook, and diary docs.
- Added tasks for TS scaffolding, bundler changes, and validation.
- Linked the analysis and playbook in the ticket index.

### Why
- The existing bundling flow cannot import `.svg` without a loader.
- We need a documented plan before changing the JS workspace and build pipeline.

### What worked
- docmgr created the ticket layout and docs without issues.

### What didn't work
- N/A.

### What I learned
- Bun's bundler lacks an explicit loader flag, so esbuild is the pragmatic choice for assets.

### What was tricky to build
- N/A.

### What warrants a second pair of eyes
- Confirm that switching to esbuild is acceptable for the broader demo goals.

### What should be done in the future
- Ensure we keep the CommonJS externals aligned with native module names.

### Code review instructions
- Start in `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-003--typescript-and-asset-imports-in-bun-bundling/analysis/01-typescript-and-asset-import-bundling-analysis.md`.
- Review `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-003--typescript-and-asset-imports-in-bun-bundling/playbook/01-typescript-asset-import-testing-plan.md` for the validation steps.

### Technical details
- Commands run:
  - `docmgr ticket create-ticket --ticket BUN-003 --title "TypeScript and asset imports in bun bundling" --topics bun,bundling,typescript,assets`
  - `docmgr doc add --ticket BUN-003 --doc-type analysis --title "TypeScript and asset import bundling analysis"`
  - `docmgr doc add --ticket BUN-003 --doc-type playbook --title "TypeScript + asset import testing plan"`
  - `docmgr doc add --ticket BUN-003 --doc-type reference --title "Diary"`
  - `docmgr task add --ticket BUN-003 --text "Define TS + asset import bundling approach (esbuild loaders, target settings, tsconfig)"`
  - `docmgr task add --ticket BUN-003 --text "Update js workspace to TypeScript (main.ts, tsconfig, svg typings, asset file)"`
  - `docmgr task add --ticket BUN-003 --text "Switch bundling script to esbuild with svg loader and update Makefile targets"`
  - `docmgr task add --ticket BUN-003 --text "Run typecheck and bundle/run validations; update analysis + playbook + diary"`
